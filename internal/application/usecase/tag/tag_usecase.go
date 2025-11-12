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
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"go.opentelemetry.io/otel/attribute"
)

type TagUseCase struct {
	tagRepo       repository.TagRepository
	merchantRepo  repository.MerchantRepository
	playerRepo    repository.PlayerRepository
	playerTagRepo repository.PlayerTagRepository
	eventProducer service.EventProducer
	logger        infrastructure.Logger
	tracing       infrastructure.TracingService
	redisManager  *redisCache.Manager
}

func NewTagUseCase(
	tagRepo repository.TagRepository,
	merchantRepo repository.MerchantRepository,
	playerRepo repository.PlayerRepository,
	playerTagRepo repository.PlayerTagRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
	redisManager *redisCache.Manager,
) inbound.TagUseCase {
	return &TagUseCase{
		tagRepo:       tagRepo,
		merchantRepo:  merchantRepo,
		playerRepo:    playerRepo,
		playerTagRepo: playerTagRepo,
		eventProducer: eventProducer,
		logger:        logger,
		tracing:       tracing,
		redisManager:  redisManager,
	}
}

func (u *TagUseCase) SyncPlayerTag(
	ctx context.Context,
	data []event.TagData,
	globalMerchantID, globalPlayerID string,
) error {
	ctx, span := u.tracing.StartSpan(ctx, "TagUseCase.SyncPlayerTag")
	defer u.tracing.SpanEnd(span)

	u.tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	u.tracing.TraceEvent(span, "Checking if player exists")
	player, err := u.playerRepo.FindByGlobalID(ctx, globalPlayerID)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find player by global_id: %w", err)
	}

	tagsToInsert, tagsGlobalIDs := make([]*entity.Tag, len(data)), make([]string, len(data))
	for k, item := range data {
		// 使用 NewTagWithTimes 建構子建立 Tag 實體
		tagsToInsert[k] = entity.NewTagWithTimes(
			merchant.GetID(),
			item.Tag.Name,
			item.Tag.GlobalTagID,
			item.Tag.UpdatedAt,
		)
		tagsGlobalIDs[k] = item.Tag.GlobalTagID
	}

	// 更新或創建Tags
	u.tracing.TraceEvent(span, "Start operation tags batch upsert")
	if err = u.tagRepo.BatchUpsert(ctx, tagsToInsert); err != nil {
		u.tracing.RecordSpanError(span, err)
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
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find tags by global_ids: %w", err)
	}

	tagIDs := make([]uint64, len(tags))
	for k, tag := range tags {
		tagIDs[k] = tag.GetID()
	}

	// 建立Player Tags關聯
	u.tracing.TraceEvent(span, "Start sync player tags relation")
	if err = u.executeLocked(ctx, player.GetID(), func() error {
		err = u.playerTagRepo.BatchUpdate(ctx, player.GetID(), tagIDs)
		if err != nil {
			u.tracing.RecordSpanError(span, err)
			return fmt.Errorf("batch update player tags failed: %w", err)
		}
		u.logger.InfoWithContext(ctx, "Batch upsert player tags completed",
			u.logger.UInt64("player_id", player.GetID()),
			u.logger.Int("count", len(tagIDs)),
			u.logger.Any("tag_ids", tagIDs),
		)
		return nil
	}); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("batch upsert player tags failed: %w", err)
	}
	u.tracing.TraceEvent(span, "Sync player tags relation completed")

	u.tracing.TraceEvent(span, "Publishing player tags sync event to KDS")
	if err = u.publishPlayerTagsSyncEvent(ctx, tagsToInsert, globalMerchantID, globalPlayerID); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player tags sync event: %w", err)
	}

	u.tracing.TraceEvent(span, "Player tags sync completed successfully")
	return nil
}

func (u *TagUseCase) SyncTag(ctx context.Context, data *event.TagSyncEvent) error {
	ctx, span := u.tracing.StartSpan(ctx, "TagUseCase.SyncTag")
	defer u.tracing.SpanEnd(span)

	u.tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	if data.Tag.UpdatedAt.IsZero() {
		data.Tag.UpdatedAt = time.Now()
	}

	// 使用 NewTagWithTimes 建構子建立 Tag 實體
	tagToInsert := entity.NewTagWithTimes(
		merchant.GetID(),
		data.Tag.Name,
		data.Tag.GlobalTagID,
		data.Tag.UpdatedAt,
	)

	// 設置 DeletedAt
	if !data.Tag.IsOpen {
		tagToInsert.SetDeletedAt(&data.Tag.UpdatedAt)
	}

	if err = u.tagRepo.Upsert(ctx, tagToInsert); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert tag: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Upsert tag completed", u.logger.Any("tag", tagToInsert))

	if err = u.publishTagSyncEvent(ctx, tagToInsert, data.GlobalMerchantID); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish tag sync event: %w", err)
	}
	u.tracing.TraceEvent(span, "Tag sync completed successfully")
	return nil
}

func (u *TagUseCase) publishPlayerTagsSyncEvent(
	ctx context.Context,
	tags []*entity.Tag,
	globalMerchantID, globalPlayerID string,
) error {
	ctx, span := u.tracing.StartSpan(ctx, "TagUseCase.publishPlayerTagsSyncEvent")
	defer u.tracing.SpanEnd(span)

	syncEvents := make([]*event.IdentityTagDataSyncEvent, len(tags))
	for i, tag := range tags {
		syncEvents[i] = &event.IdentityTagDataSyncEvent{
			GlobalTagID: tag.GetGlobalTagID(),
			Name:        tag.GetName(),
			CreatedAt:   tag.GetCreatedAt().Format(time.RFC3339),
			UpdatedAt:   tag.GetUpdatedAt().Format(time.RFC3339),
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
		TraceParent:     u.tracing.GetTraceparent(ctx),
		Data: event.IdentityPlayerTagSyncEvent{
			GlobalMerchantID: globalMerchantID,
			GlobalPlayerID:   globalPlayerID,
			Tags:             syncEvents},
	}

	u.tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	if err := u.eventProducer.PublishPlayerTagsSync(ctx, &cloudEvent); err != nil {
		u.tracing.RecordSpanError(span, err)
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
	ctx, span := u.tracing.StartSpan(ctx, "TagUseCase.publishTagSyncEvent")
	defer u.tracing.SpanEnd(span)

	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.tag.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "tag_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     u.tracing.GetTraceparent(ctx),
		Data: event.IdentityTagSyncEvent{
			GlobalMerchantID: globalMerchantID,
			Tag: &event.IdentityTagDataSyncEvent{
				GlobalTagID: tag.GetGlobalTagID(),
				Name:        tag.GetName(),
				CreatedAt:   tag.GetCreatedAt().Format(time.RFC3339),
				UpdatedAt:   tag.GetUpdatedAt().Format(time.RFC3339),
				DeletedAt: func() string {
					if tag.GetDeletedAt() == nil {
						return ""
					}
					return tag.GetDeletedAt().Format(time.RFC3339)
				}(),
			},
		},
	}

	u.tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	if err := u.eventProducer.PublishTagSync(ctx, &cloudEvent); err != nil {
		u.tracing.RecordSpanError(span, err)
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
