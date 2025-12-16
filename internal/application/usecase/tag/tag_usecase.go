package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/utils"
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
	cache         infrastructure.CacheManager
}

func NewTagUseCase(
	tagRepo repository.TagRepository,
	merchantRepo repository.MerchantRepository,
	playerRepo repository.PlayerRepository,
	playerTagRepo repository.PlayerTagRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
	cache infrastructure.CacheManager,
) inbound.TagUseCase {
	return &TagUseCase{
		tagRepo:       tagRepo,
		merchantRepo:  merchantRepo,
		playerRepo:    playerRepo,
		playerTagRepo: playerTagRepo,
		eventProducer: eventProducer,
		logger:        logger,
		tracing:       tracing,
		cache:         cache,
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
	merchantCacheKey := fmt.Sprintf(consts.RedisMerchantGlobalIDKey, globalMerchantID)
	merchant, err := utils.QueryWithCache(
		ctx,
		u.cache,
		merchantCacheKey,
		5*time.Minute,
		"merchant",
		func(ctx context.Context) (*entity.Merchant, error) {
			return u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
		},
	)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	u.tracing.TraceEvent(span, "Checking if player exists")

	playerCacheKey := fmt.Sprintf(consts.RedisPlayerGlobalIDKey, globalPlayerID)
	player, err := utils.QueryWithCache(
		ctx,
		u.cache,
		playerCacheKey,
		5*time.Minute,
		"player",
		func(ctx context.Context) (*entity.Player, error) {
			return u.playerRepo.FindByGlobalID(ctx, globalPlayerID)
		},
	)
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
	merchantCacheKey := fmt.Sprintf(consts.RedisMerchantGlobalIDKey, data.GlobalMerchantID)
	merchant, err := utils.QueryWithCache(
		ctx,
		u.cache,
		merchantCacheKey,
		5*time.Minute,
		"merchant",
		func(ctx context.Context) (*entity.Merchant, error) {
			return u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
		},
	)
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

	// 智能重試機制：多層重試策略
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		u.logger.DebugWithContext(ctx, "Attempting to acquire lock",
			u.logger.UInt64("player_id", playerID),
			u.logger.Int("attempt", attempt),
			u.logger.Int("max_attempts", maxAttempts),
		)

		// 每次重試使用遞增的參數
		expiry := time.Duration(10+attempt*5) * time.Second       // 15s, 20s, 25s
		tries := 5 + attempt*2                                    // 7, 9, 11次
		baseDelay := time.Duration(50*attempt) * time.Millisecond // 50ms, 100ms, 150ms

		mutex, err := u.cache.GetMutexWithOption(mutexKey,
			redsync.WithExpiry(expiry),
			redsync.WithTries(tries),
			redsync.WithRetryDelay(baseDelay),
		)
		if err != nil {
			u.logger.WarnWithContext(ctx, "Failed to create mutex",
				u.logger.UInt64("player_id", playerID),
				u.logger.Int("attempt", attempt),
				u.logger.Error("error", err),
			)
			if attempt == maxAttempts {
				return fmt.Errorf(
					"failed to create mutex after %d attempts for player %d: %w",
					maxAttempts,
					playerID,
					err,
				)
			}
			continue
		}

		// 嘗試獲取鎖
		if err = mutex.Lock(); err != nil {
			u.logger.WarnWithContext(ctx, "Failed to acquire lock",
				u.logger.UInt64("player_id", playerID),
				u.logger.Int("attempt", attempt),
				u.logger.String("expiry", expiry.String()),
				u.logger.Int("tries", tries),
				u.logger.Error("error", err),
			)

			if attempt < maxAttempts {
				// 計算退避時間：遞增退避
				backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
				u.logger.InfoWithContext(ctx, "Retrying after backoff",
					u.logger.UInt64("player_id", playerID),
					u.logger.String("backoff_time", backoffTime.String()),
					u.logger.Int("next_attempt", attempt+1),
				)
				time.Sleep(backoffTime)
				continue
			} else {
				return fmt.Errorf("failed to acquire lock after %d attempts for player %d: %w", maxAttempts, playerID, err)
			}
		}

		// 成功獲取鎖
		u.logger.InfoWithContext(ctx, "Successfully acquired lock",
			u.logger.UInt64("player_id", playerID),
			u.logger.Int("attempt", attempt),
			u.logger.String("expiry", expiry.String()),
		)

		// 執行業務邏輯
		defer func() {
			ok, unlockErr := mutex.Unlock()
			if !ok || unlockErr != nil {
				u.logger.ErrorWithContext(ctx, "Failed to unlock mutex",
					u.logger.String("mutex_key", mutexKey),
					u.logger.UInt64("player_id", playerID),
					u.logger.Error("err", unlockErr),
					u.logger.Bool("unlock_success", ok),
				)
			} else {
				u.logger.DebugWithContext(ctx, "Successfully unlocked mutex",
					u.logger.UInt64("player_id", playerID),
				)
			}
		}()

		return fn()
	}

	// 不應該到達這裡
	return fmt.Errorf("unexpected error: failed to acquire lock for player %d", playerID)
}
