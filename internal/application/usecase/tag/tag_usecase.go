package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/utils"
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
	lockService   infrastructure.DistributedLockService
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
	lockService infrastructure.DistributedLockService,
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
		lockService:   lockService,
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

	// 使用快取優化的標籤批次處理邏輯
	u.tracing.TraceEvent(span, "Start optimized tags batch processing with cache")
	tagsToUpsert, err := u.filterTagsNeedingUpdate(ctx, tagsToInsert)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("filter tags needing update failed: %w", err)
	}

	if len(tagsToUpsert) == 0 {
		u.logger.InfoWithContext(ctx, "No tags need updating, skipping database operation")
		// 使用原始列表進行後續查詢
	} else {
		// 只對需要更新的標籤執行資料庫操作
		u.tracing.TraceEvent(span, "Executing batch upsert for filtered tags")
		if err = u.tagRepo.BatchUpsert(ctx, tagsToUpsert); err != nil {
			u.tracing.RecordSpanError(span, err)
			return fmt.Errorf("batch upsert tags failed: %w", err)
		}

		// 更新快取 (非同步)
		go u.updateTagCache(context.Background(), tagsToUpsert)
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
	mutexKey := fmt.Sprintf(consts.SyncPlayerTagRedisKey, player.GetID())
	if err = utils.ExecuteWithLock(
		ctx,
		u.lockService,
		u.logger,
		mutexKey,
		player.GetID(),
		"player_tags",
		func() error {
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
		},
	); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("batch upsert player tags failed: %w", err)
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
	cacheKey := fmt.Sprintf(consts.RedisTagGlobalIDKey, tagToInsert.GetGlobalTagID())
	if err = u.cache.Del(ctx, cacheKey); err != nil {
		u.logger.WarnWithContext(ctx, "Cache invalidation failed",
			u.logger.String("cache_key", cacheKey),
			u.logger.Error("error", err))
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
		entity.StringAttr("outgoing.event.id", eventID),
		entity.StringAttr("outgoing.event.type", cloudEvent.Type))

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
		entity.StringAttr("outgoing.event.id", eventID),
		entity.StringAttr("outgoing.event.type", cloudEvent.Type))

	if err := u.eventProducer.PublishTagSync(ctx, &cloudEvent); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish tag sync: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Tag sync event published",
		u.logger.String("event_id", cloudEvent.ID))
	return nil
}

// filterTagsNeedingUpdate 使用快取篩選出需要更新的標籤
func (u *TagUseCase) filterTagsNeedingUpdate(
	ctx context.Context,
	tags []*entity.Tag,
) ([]*entity.Tag, error) {
	if len(tags) == 0 {
		return []*entity.Tag{}, nil
	}

	// 如果快取不可用，回退至所有標籤都需要處理
	if u.cache == nil {
		return tags, nil
	}

	tagsNeedingUpdate := make([]*entity.Tag, 0, len(tags))

	for _, tag := range tags {
		cacheKey := fmt.Sprintf(consts.RedisTagGlobalIDKey, tag.GetGlobalTagID())

		// 嘗試從快取獲取現有標籤
		cachedData, err := u.cache.Get(ctx, cacheKey)
		if err != nil {
			// 標籤不在快取中或快取錯誤，需要處理
			tagsNeedingUpdate = append(tagsNeedingUpdate, tag)
			continue
		}

		// 解析快取中的標籤
		var cachedTag entity.Tag
		if err = json.Unmarshal([]byte(cachedData), &cachedTag); err != nil {
			// 快取資料格式錯誤，需要處理
			tagsNeedingUpdate = append(tagsNeedingUpdate, tag)
			continue
		}

		// 比較標籤是否需要更新
		if u.tagNeedsUpdate(tag, &cachedTag) {
			tagsNeedingUpdate = append(tagsNeedingUpdate, tag)
		}
	}

	u.logger.InfoWithContext(
		ctx,
		"Filtered tags for update",
		u.logger.Int("original_count", len(tags)),
		u.logger.Int("filtered_count", len(tagsNeedingUpdate)),
		u.logger.Float64(
			"reduction_ratio",
			float64(len(tags)-len(tagsNeedingUpdate))/float64(len(tags))*100,
		),
	)

	return tagsNeedingUpdate, nil
}

// tagNeedsUpdate 比較兩個標籤以確定是否需要更新
func (u *TagUseCase) tagNeedsUpdate(newTag, cachedTag *entity.Tag) bool {
	// 比較可能變更的關鍵欄位
	if newTag.GetName() != cachedTag.GetName() {
		return true
	}
	return false
}

// updateTagCache 更新標籤快取（非同步）使用 BatchSet 批次執行
func (u *TagUseCase) updateTagCache(ctx context.Context, tags []*entity.Tag) {
	if u.cache == nil || len(tags) == 0 {
		return
	}

	entries := make([]entity.CacheSetEntry, 0, len(tags))
	for _, tag := range tags {
		cacheKey := fmt.Sprintf(consts.RedisTagGlobalIDKey, tag.GetGlobalTagID())
		tagData, err := json.Marshal(tag)
		if err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to marshal tag for cache",
				u.logger.String("global_tag_id", tag.GetGlobalTagID()),
				u.logger.Error("error", err))
			continue
		}
		entries = append(entries, entity.CacheSetEntry{Key: cacheKey, Value: string(tagData)})
	}
	if len(entries) > 0 {
		if err := u.cache.BatchSet(ctx, entries, 10*time.Minute); err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to batch update tag cache",
				u.logger.Int("commands_count", len(entries)),
				u.logger.Error("error", err))
			u.updateTagCacheFallback(ctx, tags)
		} else {
			u.logger.DebugWithContext(ctx, "Successfully updated tag cache via batch",
				u.logger.Int("tags_updated", len(entries)))
		}
	}
}

// updateTagCacheFallback 回退方案：逐個更新快取
func (u *TagUseCase) updateTagCacheFallback(ctx context.Context, tags []*entity.Tag) {
	for _, tag := range tags {
		cacheKey := fmt.Sprintf(consts.RedisTagGlobalIDKey, tag.GetGlobalTagID())

		tagData, err := json.Marshal(tag)
		if err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to marshal tag for cache (fallback)",
				u.logger.String("global_tag_id", tag.GetGlobalTagID()),
				u.logger.Error("error", err),
			)
			continue
		}

		// 快取 5 分鐘
		if _, err = u.cache.Set(ctx, cacheKey, string(tagData), 10*time.Minute); err != nil {
			u.logger.ErrorWithContext(ctx, "Failed to set tag cache (fallback)",
				u.logger.String("cache_key", cacheKey),
				u.logger.Error("error", err),
			)
		}
	}
}
