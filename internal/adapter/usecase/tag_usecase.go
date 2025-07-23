package usecase

import (
	"context"
	"fmt"
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
	logger        infraport.Logger
}

func NewTagUseCase(
	tagRepo repositoryport.TagRepository,
	merchantRepo repositoryport.MerchantRepository,
	playerRepo repositoryport.PlayerRepository,
	playerTagRepo repositoryport.PlayerTagRepository,
	logger infraport.Logger,
) usecaseport.TagUseCase {
	return &TagUseCase{
		tagRepo:       tagRepo,
		merchantRepo:  merchantRepo,
		playerRepo:    playerRepo,
		playerTagRepo: playerTagRepo,
		logger:        logger,
	}
}

func (u *TagUseCase) SyncTag(
	ctx context.Context,
	data []event.TagData,
	globalMerchantID, traceParent string,
) error {
	ctx, span := tracing.StartSpan(ctx, "TagUseCase.SyncTag")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil {
		if err.Error() == "record not found" {
			merchant.ID = 0
		} else {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("find merchant: %w", err)
		}
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
		if err = u.playerTagRepo.BatchDeleteByPlayerID(ctx, player.ID); err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("batch delete player tags failed: %w", err)
		}
		u.logger.InfoWithContext(ctx, "Batch delete player tags completed",
			u.logger.UInt64("player_id", player.ID),
		)
		return nil
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
}
