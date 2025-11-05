package usecase

import (
	"context"
	"fmt"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"go.opentelemetry.io/otel/attribute"
)

// MerchantUseCase 商戶用例
type MerchantUseCase struct {
	merchantRepo  repository.MerchantRepository
	eventProducer service.EventProducer
	logger        infrastructure.Logger
	tracing       infrastructure.TracingService
}

// NewMerchantUseCase 創建商戶用例
func NewMerchantUseCase(
	merchantRepo repository.MerchantRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) inbound.MerchantUseCase {
	return &MerchantUseCase{
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
		tracing:       tracing,
	}
}

// SyncMerchant 同步商戶信息
func (u *MerchantUseCase) SyncMerchant(ctx context.Context, data *event.MerchantSyncEvent) error {
	ctx, span := u.tracing.StartSpan(ctx, "MerchantUseCase.SyncMerchant")
	defer u.tracing.SpanEnd(span)

	// 記錄事件開始處理
	u.tracing.TraceEvent(span, "Starting merchant sync processing")
	u.tracing.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", data.GlobalMerchantID),
		attribute.String("merchant.name", data.Merchant.Name))

	// 使用 NewMerchantWithDisplayName 建構子建立 Merchant 實體
	merchant := entity.NewMerchantWithTimes(
		data.GlobalMerchantID,
		data.Merchant.Name,
		data.Merchant.DisplayName,
		data.Merchant.UpdatedAt,
	)

	u.tracing.TraceEvent(span, "Upsert merchant")
	if err := u.merchantRepo.Upsert(ctx, merchant); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert merchant: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Merchant upserted successfully",
		u.logger.String("global_id", merchant.GetGlobalMerchantID()),
		u.logger.String("name", merchant.GetName()))
	u.tracing.TraceEvent(span, "Database operation completed")

	// 發布商戶同步事件到KDS
	u.tracing.TraceEvent(span, "Publishing merchant sync event to KDS")
	if err := u.eventProducer.PublishMerchantSync(ctx, merchant); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish merchant sync event: %w", err)
	}

	u.tracing.TraceEvent(span, "Merchant sync completed successfully")
	return nil
}

// GetMerchantByID 通過ID獲取商戶
func (u *MerchantUseCase) GetMerchantByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracing.StartSpan(ctx, "MerchantUseCase.GetMerchantByID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.Int64("merchant.id", int64(id)))

	merchant, err := u.merchantRepo.FindByID(ctx, id)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	u.tracing.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", merchant.GetGlobalMerchantID()),
		attribute.String("merchant.name", merchant.GetName()),
	)

	return merchant, nil
}

// GetMerchantByGlobalID 通過全局ID獲取商戶
func (u *MerchantUseCase) GetMerchantByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracing.StartSpan(ctx, "MerchantUseCase.GetMerchantByGlobalID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.String("merchant.global_id", globalID))

	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find merchant: %w", err)
	}

	// 添加商戶信息到 span
	u.tracing.RecordSpanAttributes(span,
		attribute.String("merchant.name", merchant.GetName()),
	)

	return merchant, nil
}
