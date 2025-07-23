package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
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
	tagUseCase      usecaseport.TagUseCase
	levelUseCase    usecaseport.PlayerLevelUseCase
	logger          infraport.Logger
}

// NewWorkerHandler 創建Worker Handler
func NewWorkerHandler(
	merchantUseCase usecaseport.MerchantUseCase,
	playerUseCase usecaseport.PlayerUseCase,
	managerUseCase usecaseport.ManagerUseCase,
	tagUseCase usecaseport.TagUseCase,
	levelUseCase usecaseport.PlayerLevelUseCase,
	logger infraport.Logger,
) *WorkerHandler {
	return &WorkerHandler{
		merchantUseCase: merchantUseCase,
		playerUseCase:   playerUseCase,
		managerUseCase:  managerUseCase,
		tagUseCase:      tagUseCase,
		levelUseCase:    levelUseCase,
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

	h.logger.InfoLog("Registered worker handlers",
		h.logger.String("handler.merchant_sync", queue.TypeMerchantSync),
		h.logger.String("handler.player_sync", queue.TypePlayerSync),
		h.logger.String("handler.manager_sync", queue.TypeManagerSync))
}

// HandleMerchantSync 處理商戶同步任務
func (h *WorkerHandler) HandleMerchantSync(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return errors.New("task is empty")
	}
	taskID := getTaskID(task)

	// 創建處理任務的追蹤
	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeMerchantSync, taskID)
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.String("task.id", taskID),
		attribute.String("task.type", queue.TypeMerchantSync),
		attribute.Int("task.payload_size_bytes", len(task.Payload())))

	h.logger.InfoLog("Processing merchant sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	// 記錄開始處理
	tracing.TraceEvent(span, "Starting merchant sync processing")

	// 執行實際的同步邏輯
	if err := h.merchantUseCase.SyncMerchant(ctx, task.Payload()); err != nil {
		h.logger.ErrorLog("Failed to sync merchant",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync merchant: %w", err)
	}

	// 記錄成功完成任務
	tracing.TraceEvent(span, "Merchant sync completed successfully")

	h.logger.InfoLog("Merchant sync task completed successfully",
		h.logger.String("task_id", taskID))

	return nil
}

// HandlePlayerSync 處理玩家同步任務
func (h *WorkerHandler) HandlePlayerSync(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return errors.New("task is empty")
	}
	taskID := getTaskID(task)

	// 創建處理任務的追蹤
	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypePlayerSync, taskID)
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.String("task.id", taskID),
		attribute.String("task.type", queue.TypePlayerSync),
		attribute.Int("task.payload_size_bytes", len(task.Payload())))

	h.logger.InfoLog("Processing player sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	// 將事件解析為 CloudEvent
	var cloudEvent event.CloudEvent
	if err := jsoniter.Unmarshal(task.Payload(), &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal cloud event: %w", err)
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("event.id", cloudEvent.ID),
		attribute.String("event.type", cloudEvent.Type),
		attribute.String("event.source", cloudEvent.Source))

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerEvent event.PlayerSyncEvent
	if err := jsoniter.Unmarshal(dataBytes, &playerEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal player event: %w", err)
	}

	tracing.TraceEvent(span, "Starting player sync processing")

	if playerEvent.PlayerLevel.GlobalPlayerLevelID != "" {
		var levelId uint64
		levelId, err = h.levelUseCase.SyncPlayerLevel(ctx, &playerEvent.PlayerLevel,
			playerEvent.GlobalMerchantID,
			cloudEvent.TraceParent)
		if err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("failed to sync player level: %w", err)
		}
		playerEvent.Player.LevelID = levelId
	}

	if err = h.playerUseCase.SyncPlayer(ctx,
		&playerEvent.Player,
		playerEvent.GlobalMerchantID,
		cloudEvent.TraceParent); err != nil {
		h.logger.ErrorLog("Failed to sync player",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))

		tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync player: %w", err)
	}

	if len(playerEvent.PlayerTags) != 0 {
		if err = h.tagUseCase.SyncTag(ctx, playerEvent.PlayerTags,
			playerEvent.GlobalMerchantID,
			cloudEvent.TraceParent); err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("failed to sync player tags: %w", err)
		}
	}

	if err = h.tagUseCase.SyncPlayerTag(ctx, playerEvent.Player.GlobalPlayerID, playerEvent.PlayerTags); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync player tags relation: %w", err)
	}

	tracing.TraceEvent(span, "Player sync completed successfully")

	h.logger.InfoLog("Player sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandleManagerSync 處理管理員同步任務
func (h *WorkerHandler) HandleManagerSync(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return errors.New("task is empty")
	}
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeManagerSync, taskID)
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span,
		attribute.String("task.id", taskID),
		attribute.String("task.type", queue.TypeManagerSync),
		attribute.Int("task.payload_size_bytes", len(task.Payload())))

	h.logger.InfoLog("Processing manager sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	// 記錄開始處理
	tracing.TraceEvent(span, "Starting manager sync processing")

	// 執行實際的同步邏輯
	if err := h.managerUseCase.SyncManager(ctx, task.Payload()); err != nil {
		h.logger.ErrorLog("Failed to sync manager",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))

		// 記錄錯誤
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync manager: %w", err)
	}

	// 記錄成功完成任務
	tracing.TraceEvent(span, "Manager sync completed successfully")

	h.logger.InfoLog("Manager sync task completed successfully",
		h.logger.String("task_id", taskID))

	return nil
}
