package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"go.opentelemetry.io/otel/attribute"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

type LevelUseCase struct {
	levelRepo     repositoryport.LevelRepository
	merchantRepo  repositoryport.MerchantRepository
	eventProducer serviceport.EventProducer
	logger        infraport.Logger
}

func NewLevelUseCase(
	levelRepo repositoryport.LevelRepository,
	merchantRepo repositoryport.MerchantRepository,
	eventProducer serviceport.EventProducer,
	logger infraport.Logger,
) usecaseport.PlayerLevelUseCase {
	return &LevelUseCase{
		levelRepo:     levelRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

func (u *LevelUseCase) SyncPlayerLevel(
	ctx context.Context,
	data *event.LevelData,
	globalMerchantID string,
) (uint64, error) {
	ctx, span := tracing.StartSpan(ctx, "LevelUseCase.SyncPlayerLevel")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return 0, fmt.Errorf("find merchant: %w", err)
	}
	levelToInsert := &entity.Level{
		GlobalPlayerLevelID: data.GlobalPlayerLevelID,
		GlobalMerchantID:    globalMerchantID,
		Name:                data.Name,
		MerchantID:          merchant.ID,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	err = u.levelRepo.Upsert(ctx, levelToInsert)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return 0, fmt.Errorf("upsert player level: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Upsert player level completed", u.logger.Any("level", data))
	tracing.TraceEvent(span, "Upsert completed")

	tracing.TraceEvent(span, "Publishing player level sync event to KDS")
	if err = u.publishPlayerLevelSyncEvent(ctx, levelToInsert); err != nil {
		tracing.RecordSpanError(span, err)
		return 0, fmt.Errorf("publish player level sync event: %w", err)
	}

	level, err := u.levelRepo.FindByGlobalID(ctx, levelToInsert.GlobalPlayerLevelID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return 0, fmt.Errorf("find player level: %w", err)
	}

	tracing.TraceEvent(span, "Player level sync completed successfully")
	return level.ID, nil
}

func (u *LevelUseCase) SyncLevel(ctx context.Context, data *event.LevelSyncEvent) error {
	ctx, span := tracing.StartSpan(ctx, "LevelUseCase.SyncLevel")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}
	level := &entity.Level{
		GlobalPlayerLevelID: data.PlayerLevel.GlobalPlayerLevelID,
		GlobalMerchantID:    data.GlobalMerchantID,
		Name:                data.PlayerLevel.Name,
		MerchantID:          merchant.ID,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	err = u.levelRepo.Upsert(ctx, level)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert player level: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Upsert level completed", u.logger.Any("level", data))
	tracing.TraceEvent(span, "Upsert completed")

	tracing.TraceEvent(span, "Publishing level sync event to KDS")
	if err = u.publishPlayerLevelSyncEvent(ctx, level); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish level sync event: %w", err)
	}

	tracing.TraceEvent(span, "Level sync completed successfully")
	return nil
}

func (u *LevelUseCase) publishPlayerLevelSyncEvent(
	ctx context.Context,
	level *entity.Level,
) error {
	ctx, span := tracing.StartSpan(ctx, "LevelUseCase.publishPlayerLevelSyncEvent")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Preparing player level sync event for KDS")

	syncEvent := event.IdentityPlayerLevelSyncEvent{
		GlobalMerchantID:    level.GlobalMerchantID,
		GlobalPlayerLevelID: level.GlobalPlayerLevelID,
		Name:                level.Name,
		CreatedAt:           level.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           level.UpdatedAt.Format(time.RFC3339),
	}

	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.playerlevel.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "player_level_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	if err := u.eventProducer.PublishPlayerLevelSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player level sync: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Player level sync event published",
		u.logger.String("global_id", level.GlobalPlayerLevelID),
		u.logger.String("event_id", cloudEvent.ID))
	return nil
}
