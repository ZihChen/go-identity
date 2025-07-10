package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

func getTaskID(task *asynq.Task) string {
	if task == nil {
		return "unknown"
	}
	if w := task.ResultWriter(); w != nil {
		return w.TaskID()
	}

	return fmt.Sprintf("no-writer-%s", uuid.NewString()) // 保證唯一性
}

// WorkerHandler Worker Handler
type WorkerHandler struct {
	merchantUseCase usecaseport.MerchantUseCase
	playerUseCase   usecaseport.PlayerUseCase
	managerUseCase  usecaseport.ManagerUseCase
	logger          *zap.Logger
}

// NewWorkerHandler 創建Worker Handler
func NewWorkerHandler(
	merchantUseCase usecaseport.MerchantUseCase,
	playerUseCase usecaseport.PlayerUseCase,
	managerUseCase usecaseport.ManagerUseCase,
	logger *zap.Logger,
) *WorkerHandler {
	return &WorkerHandler{
		merchantUseCase: merchantUseCase,
		playerUseCase:   playerUseCase,
		managerUseCase:  managerUseCase,
		logger:          logger,
	}
}

func (h *WorkerHandler) RegisterHandlers(mux *asynq.ServeMux) {
	// 使用追蹤包裝器
	mux.Handle(
		queue.TypeMerchantSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandleMerchantSync)),
	)
	mux.Handle(
		queue.TypePlayerSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandlePlayerSync)),
	)
	mux.Handle(
		queue.TypeManagerSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandleManagerSync)),
	)

	h.logger.Info("Registered worker handlers",
		zap.String("handler.merchant_sync", queue.TypeMerchantSync),
		zap.String("handler.player_sync", queue.TypePlayerSync),
		zap.String("handler.manager_sync", queue.TypeManagerSync))
}

// HandleMerchantSync 處理商戶同步任務
func (h *WorkerHandler) HandleMerchantSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	// 創建處理任務的追蹤
	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeMerchantSync, taskID)
	defer span.End()

	// 添加任務屬性
	span.SetAttributes(
		attribute.String("task.id", taskID),
		attribute.String("task.type", queue.TypeMerchantSync),
		attribute.Int("task.payload_size_bytes", len(task.Payload())),
	)

	h.logger.Info("Processing merchant sync task",
		zap.String("task_id", taskID),
		zap.Int("payload_size", len(task.Payload())))

	// 記錄開始處理
	tracing.TraceEvent(span, "Starting merchant sync processing")

	// 執行實際的同步邏輯
	if err := h.merchantUseCase.SyncMerchant(ctx, task.Payload()); err != nil {
		h.logger.Error("Failed to sync merchant",
			zap.String("task_id", taskID),
			zap.Error(err))

		// 記錄錯誤
		span.RecordError(err)

		return fmt.Errorf("failed to sync merchant: %w", err)
	}

	// 記錄成功完成任務
	tracing.TraceEvent(span, "Merchant sync completed successfully")

	h.logger.Info("Merchant sync task completed successfully",
		zap.String("task_id", taskID))

	return nil
}

// HandlePlayerSync 處理玩家同步任務
func (h *WorkerHandler) HandlePlayerSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	// 創建處理任務的追蹤
	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypePlayerSync, taskID)
	defer span.End()

	// 添加任務屬性
	span.SetAttributes(
		attribute.String("task.id", taskID),
		attribute.String("task.type", queue.TypePlayerSync),
		attribute.Int("task.payload_size_bytes", len(task.Payload())),
	)

	h.logger.Info("Processing player sync task",
		zap.String("task_id", taskID),
		zap.Int("payload_size", len(task.Payload())))

	// 記錄開始處理
	tracing.TraceEvent(span, "Starting player sync processing")

	// 執行實際的同步邏輯
	if err := h.playerUseCase.SyncPlayer(ctx, task.Payload()); err != nil {
		h.logger.Error("Failed to sync player",
			zap.String("task_id", taskID),
			zap.Error(err))

		// 記錄錯誤
		span.RecordError(err)

		return fmt.Errorf("failed to sync player: %w", err)
	}

	// 記錄成功完成任務
	tracing.TraceEvent(span, "Player sync completed successfully")

	h.logger.Info("Player sync task completed successfully",
		zap.String("task_id", taskID))

	return nil
}

// HandleManagerSync 處理管理員同步任務
func (h *WorkerHandler) HandleManagerSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	// 創建處理任務的追蹤
	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeManagerSync, taskID)
	defer span.End()

	// 添加任務屬性
	span.SetAttributes(
		attribute.String("task.id", taskID),
		attribute.String("task.type", queue.TypeManagerSync),
		attribute.Int("task.payload_size_bytes", len(task.Payload())),
	)

	h.logger.Info("Processing manager sync task",
		zap.String("task_id", taskID),
		zap.Int("payload_size", len(task.Payload())))

	// 記錄開始處理
	tracing.TraceEvent(span, "Starting manager sync processing")

	// 執行實際的同步邏輯
	if err := h.managerUseCase.SyncManager(ctx, task.Payload()); err != nil {
		h.logger.Error("Failed to sync manager",
			zap.String("task_id", taskID),
			zap.Error(err))

		// 記錄錯誤
		span.RecordError(err)

		return fmt.Errorf("failed to sync manager: %w", err)
	}

	// 記錄成功完成任務
	tracing.TraceEvent(span, "Manager sync completed successfully")

	h.logger.Info("Manager sync task completed successfully",
		zap.String("task_id", taskID))

	return nil
}
