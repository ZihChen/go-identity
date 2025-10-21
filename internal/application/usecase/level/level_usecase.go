package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"go.opentelemetry.io/otel/attribute"
)

type LevelUseCase struct {
	levelRepo     repository.LevelRepository
	merchantRepo  repository.MerchantRepository
	eventProducer service.EventProducer
	logger        infrastructure.Logger
	tracing       infrastructure.TracingService
}

func NewLevelUseCase(
	levelRepo repository.LevelRepository,
	merchantRepo repository.MerchantRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) inbound.PlayerLevelUseCase {
	return &LevelUseCase{
		levelRepo:     levelRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
		tracing:       tracing,
	}
}

func (u *LevelUseCase) SyncLevel(ctx context.Context, data *event.LevelSyncEvent) error {
	ctx, span := u.tracing.StartSpan(ctx, "LevelUseCase.SyncLevel")
	defer u.tracing.SpanEnd(span)

	u.tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}
	level := &entity.Level{
		GlobalPlayerLevelID: data.PlayerLevel.GlobalPlayerLevelID,
		GlobalMerchantID:    data.GlobalMerchantID,
		Name:                data.PlayerLevel.Name,
		MerchantID:          merchant.ID,
		CreatedAt:           data.PlayerLevel.UpdatedAt,
		UpdatedAt:           data.PlayerLevel.UpdatedAt,
	}
	err = u.levelRepo.Upsert(ctx, level)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert player level: %w", err)
	}
	u.logger.InfoWithContext(ctx, "Upsert level completed", u.logger.Any("level", data))
	u.tracing.TraceEvent(span, "Upsert completed")

	u.tracing.TraceEvent(span, "Publishing level sync event to KDS")
	if err = u.publishPlayerLevelSyncEvent(ctx, level); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish level sync event: %w", err)
	}

	u.tracing.TraceEvent(span, "Level sync completed successfully")
	return nil
}

func (u *LevelUseCase) publishPlayerLevelSyncEvent(
	ctx context.Context,
	level *entity.Level,
) error {
	ctx, span := u.tracing.StartSpan(ctx, "LevelUseCase.publishPlayerLevelSyncEvent")
	defer u.tracing.SpanEnd(span)

	u.tracing.TraceEvent(span, "Preparing player level sync event for KDS")

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
		TraceParent:     u.tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	u.tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	if err := u.eventProducer.PublishPlayerLevelSync(ctx, &cloudEvent); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player level sync: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Player level sync event published",
		u.logger.String("global_id", level.GlobalPlayerLevelID),
		u.logger.String("event_id", cloudEvent.ID))
	return nil
}
