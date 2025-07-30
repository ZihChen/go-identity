package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redsync/redsync/v4"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"go.opentelemetry.io/otel/attribute"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
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

func (u *TagUseCase) SyncTag(
	ctx context.Context,
	data []event.TagData,
	globalMerchantID string,
) error {
	ctx, span := tracing.StartSpan(ctx, "TagUseCase.SyncTag")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	var tagsToInsert []*entity.Tag
	for _, item := range data {
		tagsToInsert = append(tagsToInsert, &entity.Tag{
			GlobalTagID: item.Tag.GlobalTagID,
			Name:        item.Tag.Name,
			MerchantID:  merchant.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	tracing.TraceEvent(span, "Database operation completed")
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

	tracing.TraceEvent(span, "Batch upsert completed")

	tracing.TraceEvent(span, "Publishing player tags sync event to KDS")
	if err = u.publishPlayerTagsSyncEvent(ctx, tagsToInsert, globalMerchantID); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player tags sync event: %w", err)
	}
	tracing.TraceEvent(span, "Player tags sync completed successfully")
	return nil
}

func (u *TagUseCase) publishPlayerTagsSyncEvent(
	ctx context.Context,
	tags []*entity.Tag,
	globalMerchantID string,
) error {
	ctx, span := tracing.StartSpan(ctx, "TagUseCase.publishPlayerTagsSyncEvent")
	defer tracing.SpanEnd(span)

	syncEvents := make([]*event.IdentityPlayerTagSyncEvent, len(tags))
	for i, tag := range tags {
		syncEvents[i] = &event.IdentityPlayerTagSyncEvent{
			GlobalMerchantID: globalMerchantID,
			GlobalTagID:      tag.GlobalTagID,
			Name:             tag.Name,
			CreatedAt:        tag.CreatedAt.Format(time.RFC3339),
			UpdatedAt:        tag.UpdatedAt.Format(time.RFC3339),
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
		Data:            syncEvents,
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	if err := u.eventProducer.PublishPlayerTagsSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player level sync: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Player tags sync event published",
		u.logger.String("event_id", cloudEvent.ID))
	return nil
}

func (u *TagUseCase) SyncPlayerTag(
	ctx context.Context,
	globalPlayerID string,
	data []event.TagData,
) error {
	ctx, span := tracing.StartSpan(ctx, "TagUseCase.SyncPlayerTag")
	defer tracing.SpanEnd(span)

	if globalPlayerID == "" {
		return fmt.Errorf("invalid input: global_player_id=%s", globalPlayerID)
	}

	player, err := u.playerRepo.FindByGlobalID(ctx, globalPlayerID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find player by global_id: %w", err)
	}

	if len(data) == 0 {
		return u.executeLocked(ctx, player.ID, func() error {
			if err = u.playerTagRepo.BatchDeleteByPlayerID(ctx, player.ID); err != nil {
				tracing.RecordSpanError(span, err)
				return fmt.Errorf("batch delete player tags failed: %w", err)
			}
			u.logger.InfoWithContext(ctx, "Batch delete player tags completed",
				u.logger.UInt64("player_id", player.ID),
			)
			return nil
		})
	}

	var tagsGlobalIDs []string
	for _, item := range data {
		tagsGlobalIDs = append(tagsGlobalIDs, item.Tag.GlobalTagID)
	}

	tags, err := u.tagRepo.FindByGlobalIDs(ctx, tagsGlobalIDs)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find tags by global_ids: %w", err)
	}

	var tagIDs []uint64
	for _, tag := range tags {
		tagIDs = append(tagIDs, tag.ID)
	}

	return u.executeLocked(ctx, player.ID, func() error {
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
	})
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
