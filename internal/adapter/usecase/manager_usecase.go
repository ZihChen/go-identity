package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ManagerUseCase 管理員用例
type ManagerUseCase struct {
	managerRepo   repositoryport.ManagerRepository
	merchantRepo  repositoryport.MerchantRepository
	eventProducer serviceport.EventProducer
	logger        *zap.Logger
}

// NewManagerUseCase 創建管理員用例
func NewManagerUseCase(
	managerRepo repositoryport.ManagerRepository,
	merchantRepo repositoryport.MerchantRepository,
	eventProducer serviceport.EventProducer,
	logger *zap.Logger,
) usecaseport.ManagerUseCase {
	return &ManagerUseCase{
		managerRepo:   managerRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

// SyncManager 同步管理員信息
func (u *ManagerUseCase) SyncManager(ctx context.Context, eventData []byte) error {
	// 獲取當前 span
	span := trace.SpanFromContext(ctx)

	// 將事件解析為 CloudEvent
	var cloudEvent event.CloudEvent
	if err := json.Unmarshal(eventData, &cloudEvent); err != nil {
		span.RecordError(err)
		return fmt.Errorf("unmarshal cloud event: %w", err)
	}

	// 添加事件信息到 span
	span.SetAttributes(
		attribute.String("event.id", cloudEvent.ID),
		attribute.String("event.type", cloudEvent.Type),
		attribute.String("event.source", cloudEvent.Source),
	)

	// 記錄事件開始處理
	tracing.TraceEvent(span, "Starting manager sync processing")

	// 將 data 部分解析為 ManagerSyncEvent
	dataBytes, err := json.Marshal(cloudEvent.Data)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("marshal event data: %w", err)
	}

	var managerEvent event.ManagerSyncEvent
	if err := json.Unmarshal(dataBytes, &managerEvent); err != nil {
		span.RecordError(err)
		return fmt.Errorf("unmarshal manager event: %w", err)
	}

	// 添加管理員信息到 span
	span.SetAttributes(
		attribute.String("manager.global_id", managerEvent.Manager.GlobalManagerID),
		attribute.String("manager.account", managerEvent.Manager.Account),
		attribute.String("merchant.global_id", managerEvent.GlobalMerchantID),
	)

	// 查找商戶是否存在
	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, managerEvent.GlobalMerchantID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("find merchant: %w", err)
	}

	// 查找管理員是否存在
	tracing.TraceEvent(span, "Checking if manager exists")
	existing, err := u.managerRepo.FindByGlobalID(ctx, managerEvent.Manager.GlobalManagerID)
	if err != nil && err.Error() != "record not found" {
		span.RecordError(err)
		return fmt.Errorf("find manager: %w", err)
	}

	// 創建或更新管理員
	var manager entity.Manager
	if existing == nil {
		// 創建新管理員
		tracing.TraceEvent(span, "Creating new manager")
		manager = entity.Manager{
			MerchantID:      merchant.ID,
			GlobalManagerID: managerEvent.Manager.GlobalManagerID,
			Account:         managerEvent.Manager.Account,
			Email:           &managerEvent.Manager.Email,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := u.managerRepo.Create(ctx, &manager); err != nil {
			span.RecordError(err)
			return fmt.Errorf("create manager: %w", err)
		}
		u.logger.Info("Manager created",
			zap.String("global_id", manager.GlobalManagerID),
			zap.String("account", manager.Account))
	} else {
		// 更新現有管理員
		tracing.TraceEvent(span, "Updating existing manager")
		manager = *existing
		manager.Account = managerEvent.Manager.Account
		manager.Email = &managerEvent.Manager.Email
		manager.UpdatedAt = time.Now()

		if err := u.managerRepo.Update(ctx, &manager); err != nil {
			span.RecordError(err)
			return fmt.Errorf("update manager: %w", err)
		}
		u.logger.Info("Manager updated",
			zap.String("global_id", manager.GlobalManagerID),
			zap.String("account", manager.Account))
	}

	// 記錄資料庫操作完成
	tracing.TraceEvent(span, "Database operation completed")

	// 發布管理員同步事件到KDS
	tracing.TraceEvent(span, "Publishing manager sync event to KDS")
	if err := u.publishManagerSyncEvent(ctx, &manager, managerEvent.GlobalMerchantID, cloudEvent.TraceParent); err != nil {
		span.RecordError(err)
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
	traceParent string,
) error {
	// 獲取當前 span
	span := trace.SpanFromContext(ctx)

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

	// 添加事件信息到 span
	span.SetAttributes(
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
	)

	// 發布事件
	if err := u.eventProducer.PublishManagerSync(ctx, &cloudEvent); err != nil {
		span.RecordError(err)
		return fmt.Errorf("publish manager sync: %w", err)
	}

	// 記錄事件發布成功
	tracing.TraceEvent(span, "Manager sync event published successfully")

	u.logger.Info("Manager sync event published",
		zap.String("global_id", manager.GlobalManagerID),
		zap.String("event_id", cloudEvent.ID))

	return nil
}

// GetManagerByID 通過ID獲取管理員
func (u *ManagerUseCase) GetManagerByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.GetManagerByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("manager.id", int64(id)))

	manager, err := u.managerRepo.FindByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	// 添加管理員信息到 span
	span.SetAttributes(
		attribute.String("manager.global_id", manager.GlobalManagerID),
		attribute.String("manager.account", manager.Account),
	)

	return manager, nil
}

// GetManagerByGlobalID 通過全局ID獲取管理員
func (u *ManagerUseCase) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "ManagerUseCase.GetManagerByGlobalID")
	defer span.End()

	span.SetAttributes(attribute.String("manager.global_id", globalID))

	manager, err := u.managerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("find manager: %w", err)
	}

	// 添加管理員信息到 span
	span.SetAttributes(
		attribute.String("manager.account", manager.Account),
	)

	return manager, nil
}
