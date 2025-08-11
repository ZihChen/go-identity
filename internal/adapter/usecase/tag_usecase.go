package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type TagUseCase struct {
	tagRepo       repositoryport.TagRepository
	merchantRepo  repositoryport.MerchantRepository
	playerRepo    repositoryport.PlayerRepository
	playerTagRepo repositoryport.PlayerTagRepository
	eventProducer serviceport.EventProducer
	logger        infraport.Logger
	redisManager  *redisCache.Manager
}

func NewTagUseCase(
	tagRepo repositoryport.TagRepository,
	merchantRepo repositoryport.MerchantRepository,
	playerRepo repositoryport.PlayerRepository,
	playerTagRepo repositoryport.PlayerTagRepository,
	eventProducer serviceport.EventProducer,
	logger infraport.Logger,
	redisManager *redisCache.Manager,
) usecaseport.TagUseCase {
	return &TagUseCase{
		tagRepo:       tagRepo,
		merchantRepo:  merchantRepo,
		playerRepo:    playerRepo,
		playerTagRepo: playerTagRepo,
		eventProducer: eventProducer,
		logger:        logger,
		redisManager:  redisManager,
	}
}

func (u *TagUseCase) SyncPlayerTag(
	ctx context.Context,
	data []event.TagData,
	globalMerchantID, globalPlayerID string,
) error {
	ctx, span := tracing.StartSpan(ctx, "TagUseCase.SyncPlayerTag")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	tracing.TraceEvent(span, "Checking if player exists")
	player, err := u.playerRepo.FindByGlobalID(ctx, globalPlayerID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find player by global_id: %w", err)
	}

	tagsToInsert, tagsGlobalIDs := make([]*entity.Tag, len(data)), make([]string, len(data))
	for k, item := range data {
		tagsToInsert[k] = &entity.Tag{
			GlobalTagID: item.Tag.GlobalTagID,
			Name:        item.Tag.Name,
			MerchantID:  merchant.ID,
			CreatedAt:   item.Tag.UpdatedAt,
			UpdatedAt:   item.Tag.UpdatedAt,
		}
		tagsGlobalIDs[k] = item.Tag.GlobalTagID
	}

	// 更新或創建Tags
	tracing.TraceEvent(span, "Start operation tags batch upsert")
	if err = u.tagRepo.BatchUpsert(ctx, tagsToInsert); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("batch upsert tags failed: %w", err)
	}
	u.logger.InfoWithContext(
		ctx,
		"Batch upsert tags completed",
		u.logger.Int("count", len(tagsToInsert)),
		u.logger.Any("tags", tagsToInsert),
	)

	tags, err := u.tagRepo.FindByGlobalIDs(ctx, tagsGlobalIDs)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find tags by global_ids: %w", err)
	}

	tagIDs := make([]uint64, len(tags))
	for k, tag := range tags {
		tagIDs[k] = tag.ID
	}

	// 建立Player Tags關聯
	tracing.TraceEvent(span, "Start sync player tags relation")
	if err = u.executeLocked(ctx, player.ID, func() error {
		err = u.playerTagRepo.BatchUpdate(ctx, player.ID, tagIDs)
		if err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("batch update player tags failed: %w", err)
		}
		u.logger.InfoWithContext(ctx, "Batch upsert player tags completed",
			u.logger.UInt64("player_id", player.ID),
			u.logger.Int("count", len(tagIDs)),
			u.logger.Any("tag_ids", tagIDs),
		)
		return nil
	}); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("batch upsert player tags failed: %w", err)
	}
	tracing.TraceEvent(span, "Sync player tags relation completed")

	tracing.TraceEvent(span, "Publishing player tags sync event to KDS")
	if err = u.publishPlayerTagsSyncEvent(ctx, tagsToInsert, globalMerchantID, globalPlayerID); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player tags sync event: %w", err)
	}

	tracing.TraceEvent(span, "Player tags sync completed successfully")
	return nil
}

func (u *TagUseCase) SyncTag(ctx context.Context, data *event.TagSyncEvent) error {
	ctx, span := tracing.StartSpan(ctx, "TagUseCase.SyncTag")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	tagToInsert := &entity.Tag{
		GlobalTagID: data.Tag.GlobalTagID,
		Name:        data.Tag.Name,
		MerchantID:  merchant.ID,
		CreatedAt:   data.Tag.UpdatedAt,
		UpdatedAt:   data.Tag.UpdatedAt,
		DeletedAt: func() *time.Time {
			if data.Tag.IsOpen {
				return nil
			}
			return &data.Tag.UpdatedAt
		}(),
	}

	if err = u.tagRepo.Upsert(ctx, tagToInsert); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert tag: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Upsert tag completed", u.logger.Any("tag", tagToInsert))

	if err = u.publishTagSyncEvent(ctx, tagToInsert, data.GlobalMerchantID); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish tag sync event: %w", err)
	}
	tracing.TraceEvent(span, "Tag sync completed successfully")
	return nil
}

func (u *TagUseCase) publishPlayerTagsSyncEvent(
	ctx context.Context,
	tags []*entity.Tag,
	globalMerchantID, globalPlayerID string,
) error {
	ctx, span := tracing.StartSpan(ctx, "TagUseCase.publishPlayerTagsSyncEvent")
	defer tracing.SpanEnd(span)

	syncEvents := make([]*event.IdentityTagDataSyncEvent, len(tags))
	for i, tag := range tags {
		syncEvents[i] = &event.IdentityTagDataSyncEvent{
			GlobalTagID: tag.GlobalTagID,
			Name:        tag.Name,
			CreatedAt:   tag.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   tag.UpdatedAt.Format(time.RFC3339),
		}
	}

	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.playertags.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "player_tags_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     tracing.GetTraceparent(ctx),
		Data: event.IdentityPlayerTagSyncEvent{
			GlobalMerchantID: globalMerchantID,
			GlobalPlayerID:   globalPlayerID,
			Tags:             syncEvents},
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	if err := u.eventProducer.PublishPlayerTagsSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish tag sync: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Player tags sync event published",
		u.logger.String("event_id", cloudEvent.ID))
	return nil
}

func (u *TagUseCase) publishTagSyncEvent(
	ctx context.Context,
	tag *entity.Tag,
	globalMerchantID string,
) error {
	ctx, span := tracing.StartSpan(ctx, "TagUseCase.publishTagSyncEvent")
	defer tracing.SpanEnd(span)

	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.tag.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "tag_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     tracing.GetTraceparent(ctx),
		Data: event.IdentityTagSyncEvent{
			GlobalMerchantID: globalMerchantID,
			Tag: &event.IdentityTagDataSyncEvent{
				GlobalTagID: tag.GlobalTagID,
				Name:        tag.Name,
				CreatedAt:   tag.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   tag.UpdatedAt.Format(time.RFC3339),
				DeletedAt: func() string {
					if tag.DeletedAt == nil {
						return ""
					}
					return tag.DeletedAt.Format(time.RFC3339)
				}(),
			},
		},
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	if err := u.eventProducer.PublishTagSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish tag sync: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Tag sync event published",
		u.logger.String("event_id", cloudEvent.ID))
	return nil
}

func (u *TagUseCase) executeLocked(ctx context.Context, playerID uint64, fn func() error) error {
	mutexKey := fmt.Sprintf(consts.SyncPlayerTagRedisKey, playerID)

	mutex, err := u.redisManager.GetMutexWithOption(mutexKey,
		redsync.WithExpiry(5*time.Second),            // 鎖的過期時間
		redsync.WithTries(3),                         // 獲取鎖的重試次數
		redsync.WithRetryDelay(200*time.Millisecond), // 重試間隔
	)
	if err != nil {
		return fmt.Errorf("failed to create mutex for player %d: %w", playerID, err)
	}

	if err = mutex.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock for player %d: %w", playerID, err)
	}

	defer func() {
		ok, unlockErr := mutex.Unlock()
		if !ok || unlockErr != nil {
			u.logger.ErrorWithContext(ctx, "Failed to unlock mutex",
				u.logger.String("mutex_key", mutexKey),
				u.logger.UInt64("player_id", playerID),
				u.logger.Error("err", unlockErr),
				u.logger.Bool("unlock_success", ok),
			)
		}
	}()
	return fn()
}
