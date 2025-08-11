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
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// ManagerUseCase 管理員用例
type ManagerUseCase struct {
	managerRepo   repositoryport.ManagerRepository
	merchantRepo  repositoryport.MerchantRepository
	eventProducer serviceport.EventProducer
	logger        infraport.Logger
}

// NewManagerUseCase 創建管理員用例
func NewManagerUseCase(
	managerRepo repositoryport.ManagerRepository,
	merchantRepo repositoryport.MerchantRepository,
	eventProducer serviceport.EventProducer,
	logger infraport.Logger,
) usecaseport.ManagerUseCase {
	return &ManagerUseCase{
		managerRepo:   managerRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

// SyncManager 同步管理員信息
func (u *ManagerUseCase) SyncManager(ctx context.Context, data *event.ManagerSyncEvent) error {
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.SyncManager")

	// 記錄事件開始處理
	tracing.TraceEvent(span, "Starting manager sync processing")
	tracing.RecordSpanAttributes(span,
		attribute.String("manager.global_id", data.Manager.GlobalManagerID),
		attribute.String("manager.account", data.Manager.Account),
		attribute.String("merchant.global_id", data.GlobalMerchantID))

	// 查找商戶是否存在
	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	manager := &entity.Manager{
		MerchantID:      merchant.ID,
		GlobalManagerID: data.Manager.GlobalManagerID,
		Account:         data.Manager.Account,
		Email:           &data.Manager.Email,
		CreatedAt:       data.Manager.UpdatedAt,
		UpdatedAt:       data.Manager.UpdatedAt,
		DeletedAt: func() *time.Time {
			if data.Manager.DeletedAt == "" {
				return nil
			}
			return &data.Manager.UpdatedAt
		}(),
	}

	tracing.TraceEvent(span, "Upsert manager")
	if err = u.managerRepo.Upsert(ctx, manager); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert manager: %w", err)
	}

	u.logger.InfoLog("Manager upserted successfully",
		u.logger.String("global_id", manager.GlobalManagerID),
		u.logger.String("account", manager.Account))
	tracing.TraceEvent(span, "Database operation completed")
	tracing.RecordSpanAttributes(span,
		attribute.String("manager.global_id", manager.GlobalManagerID),
		attribute.String("manager.account", manager.Account))

	// 發布管理員同步事件到KDS
	tracing.TraceEvent(span, "Publishing manager sync event to KDS")
	if err = u.publishManagerSyncEvent(ctx, manager, data.GlobalMerchantID); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish manager sync event: %w", err)
	}

	// 記錄處理完成
	tracing.TraceEvent(span, "Manager sync completed successfully")
	return nil
}

// publishManagerSyncEvent 發布管理員同步事件
func (u *ManagerUseCase) publishManagerSyncEvent(
	ctx context.Context,
	manager *entity.Manager,
	globalMerchantID string,
) error {
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.SyncManager")

	// 記錄發布事件開始
	tracing.TraceEvent(span, "Preparing manager sync event for KDS")

	// 構建事件數據
	syncEvent := event.IdentityManagerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		GlobalManagerID:  manager.GlobalManagerID,
		ID:               manager.ID,
		MerchantID:       manager.MerchantID,
		Account:          manager.Account,
		Email:            manager.Email,
		CreatedAt:        manager.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        manager.UpdatedAt.Format(time.RFC3339),
	}

	if manager.DeletedAt != nil {
		syncEvent.DeletedAt = manager.DeletedAt.Format(time.RFC3339)
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
		TraceParent:     tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type))

	// 發布事件
	if err := u.eventProducer.PublishManagerSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish manager sync: %w", err)
	}

	// 記錄事件發布成功
	tracing.TraceEvent(span, "Manager sync event published successfully")

	u.logger.InfoLog("Manager sync event published",
		u.logger.String("global_id", manager.GlobalManagerID),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetManagerByID 通過ID獲取管理員
func (u *ManagerUseCase) GetManagerByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.GetManagerByID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("manager.id", int64(id)))

	manager, err := u.managerRepo.FindByID(ctx, id)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("manager.global_id", manager.GlobalManagerID),
		attribute.String("manager.account", manager.Account))
	return manager, nil
}

// GetManagerByGlobalID 通過全局ID獲取管理員
func (u *ManagerUseCase) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.GetManagerByGlobalID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.String("manager.global_id", globalID))

	manager, err := u.managerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	tracing.RecordSpanAttributes(span, attribute.String("manager.account", manager.Account))
	return manager, nil
}
