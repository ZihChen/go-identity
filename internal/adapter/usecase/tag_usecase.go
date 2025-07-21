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
	tagRepo      repositoryport.TagRepository
	merchantRepo repositoryport.MerchantRepository
	logger       infraport.Logger
}

func NewTagUseCase(
	tagRepo repositoryport.TagRepository,
	merchantRepo repositoryport.MerchantRepository,
	logger infraport.Logger,
) usecaseport.TagUseCase {
	return &TagUseCase{
		tagRepo:      tagRepo,
		merchantRepo: merchantRepo,
		logger:       logger,
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

	var tags []*entity.Tag

	for _, item := range data {
		tags = append(tags, &entity.Tag{
			GlobalTagID: item.Tag.GlobalTagID,
			Name:        item.Tag.Name,
			MerchantID:  merchant.ID,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		})
	}

	tracing.TraceEvent(span, "Database operation completed")
	if err = u.tagRepo.BatchUpsert(ctx, tags); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("batch upsert tags failed: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Batch upsert tags completed", u.logger.Int("count", len(tags)),
		u.logger.Any("tags", tags))

	tracing.TraceEvent(span, "Batch upsert completed")
	return nil
}
