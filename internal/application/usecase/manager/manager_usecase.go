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

// ManagerUseCase 管理員用例
type ManagerUseCase struct {
	managerRepo   repository.ManagerRepository
	merchantRepo  repository.MerchantRepository
	eventProducer service.EventProducer
	logger        infrastructure.Logger
	tracing       infrastructure.TracingService
}

// NewManagerUseCase 創建管理員用例
func NewManagerUseCase(
	managerRepo repository.ManagerRepository,
	merchantRepo repository.MerchantRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) inbound.ManagerUseCase {
	return &ManagerUseCase{
		managerRepo:   managerRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
		tracing:       tracing,
	}
}

// SyncManager 同步管理員信息
func (u *ManagerUseCase) SyncManager(ctx context.Context, data *event.ManagerSyncEvent) error {
	ctx, span := u.tracing.StartSpan(ctx, "ManagerUseCase.SyncManager")

	// 記錄事件開始處理
	u.tracing.TraceEvent(span, "Starting manager sync processing")
	u.tracing.RecordSpanAttributes(span,
		attribute.String("manager.global_id", data.Manager.GlobalManagerID),
		attribute.String("manager.account", data.Manager.Account),
		attribute.String("merchant.global_id", data.GlobalMerchantID))

	// 查找商戶是否存在
	u.tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	if data.Manager.UpdatedAt.IsZero() {
		data.Manager.UpdatedAt = time.Now()
	}

	// 使用 NewManagerWithTimes 建構子建立 Manager 實體
	manager := entity.NewManagerWithTimes(
		merchant.GetID(),
		data.Manager.GlobalManagerID,
		data.Manager.Account,
		&data.Manager.Email,
		data.Manager.UpdatedAt,
		data.Manager.UpdatedAt,
	)

	if err = manager.IsValid(); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("validate manager: %w", err)
	}

	// 設置 DeletedAt
	if data.Manager.DeletedAt != "" {
		manager.SetDeletedAt(&data.Manager.UpdatedAt)
	}

	u.tracing.TraceEvent(span, "Upsert manager")
	if err = u.managerRepo.Upsert(ctx, manager); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert manager: %w", err)
	}

	u.logger.InfoLog("Manager upserted successfully",
		u.logger.String("global_id", manager.GetGlobalManagerID()),
		u.logger.String("account", manager.GetAccount()))
	u.tracing.TraceEvent(span, "Database operation completed")
	u.tracing.RecordSpanAttributes(span,
		attribute.String("manager.global_id", manager.GetGlobalManagerID()),
		attribute.String("manager.account", manager.GetAccount()))

	// 發布管理員同步事件到KDS
	u.tracing.TraceEvent(span, "Publishing manager sync event to KDS")
	if err = u.publishManagerSyncEvent(ctx, manager, data.GlobalMerchantID); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish manager sync event: %w", err)
	}

	// 記錄處理完成
	u.tracing.TraceEvent(span, "Manager sync completed successfully")
	return nil
}

// publishManagerSyncEvent 發布管理員同步事件
func (u *ManagerUseCase) publishManagerSyncEvent(
	ctx context.Context,
	manager *entity.Manager,
	globalMerchantID string,
) error {
	ctx, span := u.tracing.StartSpan(ctx, "ManagerUseCase.SyncManager")

	// 記錄發布事件開始
	u.tracing.TraceEvent(span, "Preparing manager sync event for KDS")

	// 構建事件數據
	syncEvent := event.IdentityManagerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		GlobalManagerID:  manager.GetGlobalManagerID(),
		ID:               manager.GetID(),
		MerchantID:       manager.GetMerchantID(),
		Account:          manager.GetAccount(),
		Email:            manager.GetEmail(),
		CreatedAt:        manager.GetCreatedAt().Format(time.RFC3339),
		UpdatedAt:        manager.GetUpdatedAt().Format(time.RFC3339),
	}

	if manager.GetDeletedAt() != nil {
		syncEvent.DeletedAt = manager.GetDeletedAt().Format(time.RFC3339)
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.manager.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "manager_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     u.tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	u.tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	// 發布事件
	if err := u.eventProducer.PublishManagerSync(ctx, &cloudEvent); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish manager sync: %w", err)
	}

	// 記錄事件發布成功
	u.tracing.TraceEvent(span, "Manager sync event published successfully")

	u.logger.InfoLog("Manager sync event published",
		u.logger.String("global_id", manager.GetGlobalManagerID()),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetManagerByID 通過ID獲取管理員
func (u *ManagerUseCase) GetManagerByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	ctx, span := u.tracing.StartSpan(ctx, "ManagerUseCase.GetManagerByID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.Int64("manager.id", int64(id)))

	manager, err := u.managerRepo.FindByID(ctx, id)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	u.tracing.RecordSpanAttributes(span,
		attribute.String("manager.global_id", manager.GetGlobalManagerID()),
		attribute.String("manager.account", manager.GetAccount()))
	return manager, nil
}

// GetManagerByGlobalID 通過全局ID獲取管理員
func (u *ManagerUseCase) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracing.StartSpan(ctx, "ManagerUseCase.GetManagerByGlobalID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.String("manager.global_id", globalID))

	manager, err := u.managerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	u.tracing.RecordSpanAttributes(span, attribute.String("manager.account", manager.GetAccount()))
	return manager, nil
}
