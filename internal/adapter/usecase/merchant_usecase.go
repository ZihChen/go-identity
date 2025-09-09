package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// MerchantUseCase 商戶用例
type MerchantUseCase struct {
	merchantRepo  repository.MerchantRepository
	eventProducer service.EventProducer
	logger        infraport.Logger
}

// NewMerchantUseCase 創建商戶用例
func NewMerchantUseCase(
	merchantRepo repository.MerchantRepository,
	eventProducer service.EventProducer,
	logger infraport.Logger,
) inbound.MerchantUseCase {
	return &MerchantUseCase{
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

// SyncMerchant 同步商戶信息
func (u *MerchantUseCase) SyncMerchant(ctx context.Context, data *event.MerchantSyncEvent) error {
	ctx, span := tracing.StartSpan(ctx, "MerchantUseCase.SyncMerchant")
	defer tracing.SpanEnd(span)

	// 記錄事件開始處理
	tracing.TraceEvent(span, "Starting merchant sync processing")
	tracing.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", data.GlobalMerchantID),
		attribute.String("merchant.name", data.Merchant.Name))

	merchant := &entity.Merchant{
		GlobalMerchantID: data.GlobalMerchantID,
		Name:             data.Merchant.Name,
		DisplayName:      data.Merchant.DisplayName,
		APIKey:           uuid.New().String(),
		CreatedAt:        data.Merchant.UpdatedAt,
		UpdatedAt:        data.Merchant.UpdatedAt,
	}

	tracing.TraceEvent(span, "Upsert merchant")
	if err := u.merchantRepo.Upsert(ctx, merchant); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert merchant: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Merchant upserted successfully",
		u.logger.String("global_id", merchant.GlobalMerchantID),
		u.logger.String("name", merchant.Name))
	tracing.TraceEvent(span, "Database operation completed")

	// 發布商戶同步事件到KDS
	tracing.TraceEvent(span, "Publishing merchant sync event to KDS")
	if err := u.publishMerchantSyncEvent(ctx, merchant); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish merchant sync event: %w", err)
	}

	tracing.TraceEvent(span, "Merchant sync completed successfully")
	return nil
}

// publishMerchantSyncEvent 發布商戶同步事件
func (u *MerchantUseCase) publishMerchantSyncEvent(
	ctx context.Context,
	merchant *entity.Merchant,
) error {
	ctx, span := tracing.StartSpan(ctx, "MerchantUseCase.publishMerchantSyncEvent")

	// 記錄發布事件開始
	tracing.TraceEvent(span, "Preparing merchant sync event for KDS")

	// 構建事件數據
	syncEvent := event.IdentityMerchantSyncEvent{
		GlobalMerchantID: merchant.GlobalMerchantID,
		ID:               merchant.ID,
		Name:             merchant.Name,
		DisplayName:      merchant.DisplayName,
		APIKey:           merchant.APIKey,
		CreatedAt:        merchant.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        merchant.UpdatedAt.Format(time.RFC3339),
	}

	if merchant.DeletedAt != nil {
		syncEvent.DeletedAt = merchant.DeletedAt.Format(time.RFC3339)
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.merchant.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "merchant_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
	)

	// 發布事件
	if err := u.eventProducer.PublishMerchantSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish merchant sync: %w", err)
	}

	// 記錄事件發布成功
	tracing.TraceEvent(span, "Merchant sync event published successfully")

	u.logger.InfoLog("Merchant sync event published",
		u.logger.String("global_id", merchant.GlobalMerchantID),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetMerchantByID 通過ID獲取商戶
func (u *MerchantUseCase) GetMerchantByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "MerchantUseCase.GetMerchantByID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("merchant.id", int64(id)))

	merchant, err := u.merchantRepo.FindByID(ctx, id)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", merchant.GlobalMerchantID),
		attribute.String("merchant.name", merchant.Name),
	)

	return merchant, nil
}

// GetMerchantByGlobalID 通過全局ID獲取商戶
func (u *MerchantUseCase) GetMerchantByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "MerchantUseCase.GetMerchantByGlobalID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.String("merchant.global_id", globalID))

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	// 添加商戶信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("merchant.name", merchant.Name),
	)

	return merchant, nil
}
