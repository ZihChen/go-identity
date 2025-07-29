package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// MerchantUseCase 商戶用例
type MerchantUseCase struct {
	merchantRepo  repositoryport.MerchantRepository
	eventProducer serviceport.EventProducer
	logger        infraport.Logger
}

// NewMerchantUseCase 創建商戶用例
func NewMerchantUseCase(
	merchantRepo repositoryport.MerchantRepository,
	eventProducer serviceport.EventProducer,
	logger infraport.Logger,
) usecaseport.MerchantUseCase {
	return &MerchantUseCase{
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

// SyncMerchant 同步商戶信息
func (u *MerchantUseCase) SyncMerchant(ctx context.Context, eventData []byte) error {
	ctx, span := tracing.StartSpan(ctx, "MerchantUseCase.SyncMerchant")
	nowTime := time.Now()

	// 將事件解析為 CloudEvent
	var cloudEvent event.CloudEvent
	if err := json.Unmarshal(eventData, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal cloud event: %w", err)
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("event.id", cloudEvent.ID),
		attribute.String("event.type", cloudEvent.Type),
		attribute.String("event.source", cloudEvent.Source))

	// 記錄事件開始處理
	tracing.TraceEvent(span, "Starting merchant sync processing")

	// 將 data 部分解析為 MerchantSyncEvent
	dataBytes, err := json.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("marshal event data: %w", err)
	}

	var merchantEvent event.MerchantSyncEvent
	if err = json.Unmarshal(dataBytes, &merchantEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal merchant event: %w", err)
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("merchant.global_id", merchantEvent.GlobalMerchantID),
		attribute.String("merchant.name", merchantEvent.Merchant.Name))

	// 查找商戶是否存在
	tracing.TraceEvent(span, "Checking if merchant exists")
	existing, err := u.merchantRepo.FindByGlobalID(ctx, merchantEvent.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	// 創建或更新商戶
	var merchant entity.Merchant
	if errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		// 創建新商戶
		tracing.TraceEvent(span, "Creating new merchant")
		merchant = entity.Merchant{
			GlobalMerchantID: merchantEvent.GlobalMerchantID,
			Name:             merchantEvent.Merchant.Name,
			DisplayName:      merchantEvent.Merchant.DisplayName,
			APIKey:           uuid.New().String(), // 生成新的API密鑰
			CreatedAt:        nowTime,
			UpdatedAt:        nowTime,
		}
		if err = u.merchantRepo.FirstOrCreate(ctx, &merchant); err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("create merchant: %w", err)
		}
		u.logger.InfoLog("Merchant created",
			u.logger.String("global_id", merchant.GlobalMerchantID),
			u.logger.String("name", merchant.Name))
	} else {
		// 更新現有商戶
		tracing.TraceEvent(span, "Updating existing merchant")

		var eventTime time.Time
		if cloudEvent.Time.After(time.Time{}) {
			eventTime = cloudEvent.Time
		} else {
			eventTime = nowTime
		}

		// 如果現有記錄的更新時間較新，則跳過更新（確保幂等性）
		if existing.UpdatedAt.After(eventTime) {
			u.logger.InfoWithContext(ctx, "Skipping merchant update as existing data is newer",
				u.logger.String("global_id", existing.GlobalMerchantID),
				u.logger.String("existing_updated_at", existing.UpdatedAt.String()),
				u.logger.String("event_time", eventTime.String()))
			return nil
		}

		merchant = *existing
		merchant.Name = merchantEvent.Merchant.Name
		merchant.DisplayName = merchantEvent.Merchant.DisplayName
		merchant.UpdatedAt = nowTime

		if err = u.merchantRepo.Update(ctx, &merchant); err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("update merchant: %w", err)
		}
		u.logger.InfoLog("Merchant updated",
			u.logger.String("global_id", merchant.GlobalMerchantID),
			u.logger.String("name", merchant.Name))
	}

	// 記錄資料庫操作完成
	tracing.TraceEvent(span, "Database operation completed")

	// 發布商戶同步事件到KDS
	tracing.TraceEvent(span, "Publishing merchant sync event to KDS")
	if err = u.publishMerchantSyncEvent(ctx, &merchant, cloudEvent.TraceParent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish merchant sync event: %w", err)
	}

	// 記錄處理完成
	tracing.TraceEvent(span, "Merchant sync completed successfully")
	return nil
}

// publishMerchantSyncEvent 發布商戶同步事件
func (u *MerchantUseCase) publishMerchantSyncEvent(
	ctx context.Context,
	merchant *entity.Merchant,
	traceParent string,
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
