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

type LevelUseCase struct {
	levelRepo    repositoryport.LevelRepository
	merchantRepo repositoryport.MerchantRepository
	logger       infraport.Logger
}

func NewLevelUseCase(
	levelRepo repositoryport.LevelRepository,
	merchantRepo repositoryport.MerchantRepository,
	logger infraport.Logger,
) usecaseport.PlayerLevelUseCase {
	return &LevelUseCase{
		levelRepo:    levelRepo,
		merchantRepo: merchantRepo,
		logger:       logger,
	}
}

func (u *LevelUseCase) SyncPlayerLevel(
	ctx context.Context,
	data *event.LevelData,
	globalMerchantID, traceParent string,
) (uint64, error) {
	ctx, span := tracing.StartSpan(ctx, "LevelUseCase.SyncPlayerLevel")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil {
		if err.Error() == "record not found" {
			merchant.ID = 0
		} else {
			tracing.RecordSpanError(span, err)
			return 0, fmt.Errorf("find merchant: %w", err)
		}
	}

	levelID, err := u.levelRepo.Upsert(ctx, &entity.Level{
		GlobalPlayerLevelID: data.GlobalPlayerLevelID,
		Name:                data.Name,
		MerchantID:          merchant.ID,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	})
	if err != nil {
		tracing.RecordSpanError(span, err)
		return 0, fmt.Errorf("upsert level: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Upsert level completed", u.logger.Any("level", data))
	tracing.TraceEvent(span, "Upsert completed")
	return levelID, nil
}
