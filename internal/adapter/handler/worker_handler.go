package handler

import (
	"context"
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
	"go.opentelemetry.io/otel/trace"
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
	mux.Handle(
		queue.TypeTagSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandleTagSync)),
	)
	mux.Handle(
		queue.TypeLevelSync,
		queue.WrapHandlerWithTracing(asynq.HandlerFunc(h.HandleLevelSync)),
	)

	h.logger.InfoLog("Registered worker handlers",
		h.logger.String("handler.merchant_sync", queue.TypeMerchantSync),
		h.logger.String("handler.player_sync", queue.TypePlayerSync),
		h.logger.String("handler.manager_sync", queue.TypeManagerSync),
		h.logger.String("handler.level_sync", queue.TypeLevelSync),
		h.logger.String("handler.tag_sync", queue.TypeTagSync))
}

// HandleMerchantSync 處理商戶同步任務
func (h *WorkerHandler) HandleMerchantSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	// 創建處理任務的追蹤
	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeMerchantSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoLog("Processing merchant sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var merchantEvent event.MerchantSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &merchantEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal merchant event: %w", err)
	}

	tracing.TraceEvent(span, "Starting merchant sync processing")

	// 同步資料
	if err = h.merchantUseCase.SyncMerchant(ctx, &merchantEvent); err != nil {
		h.logger.ErrorLog("Failed to sync merchant",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync merchant: %w", err)
	}

	tracing.TraceEvent(span, "Merchant sync completed successfully")

	h.logger.InfoLog("Merchant sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandlePlayerSync 處理玩家同步任務
func (h *WorkerHandler) HandlePlayerSync(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return fmt.Errorf("task is empty")
	}
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypePlayerSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoLog("Processing player sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerEvent event.PlayerSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &playerEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal player event: %w", err)
	}

	tracing.TraceEvent(span, "Starting player sync processing")

	if err = h.playerUseCase.SyncPlayer(ctx,
		&playerEvent); err != nil {
		h.logger.ErrorLog("Failed to sync player",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))

		tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync player: %w", err)
	}

	if len(playerEvent.PlayerTags) != 0 {
		if err = h.tagUseCase.SyncPlayerTag(ctx,
			playerEvent.PlayerTags,
			playerEvent.GlobalMerchantID,
			playerEvent.Player.GlobalPlayerID); err != nil {
			tracing.RecordSpanError(span, err)
			return fmt.Errorf("failed to sync player tags relation: %w", err)
		}
	}

	tracing.TraceEvent(span, "Player sync completed successfully")

	h.logger.InfoLog("Player sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandleManagerSync 處理管理員同步任務
func (h *WorkerHandler) HandleManagerSync(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return fmt.Errorf("task is empty")
	}
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeManagerSync, taskID)
	defer tracing.SpanEnd(span)

	h.logger.InfoLog("Processing manager sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))
	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var managerEvent event.ManagerSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &managerEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal manager event: %w", err)
	}

	tracing.TraceEvent(span, "Starting manager sync processing")

	// 同步資料
	if err = h.managerUseCase.SyncManager(ctx, &managerEvent); err != nil {
		h.logger.ErrorLog("Failed to sync manager",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync manager: %w", err)
	}

	tracing.TraceEvent(span, "Manager sync completed successfully")
	h.logger.InfoLog("Manager sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandleTagSync 處理會員標籤同步任務
func (h *WorkerHandler) HandleTagSync(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return fmt.Errorf("task is empty")
	}
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeTagSync, taskID)
	defer tracing.SpanEnd(span)
	tracing.TraceEvent(span, "Starting level sync processing")

	h.logger.InfoLog("Processing tag sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))
	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var tagEvent event.TagSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &tagEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal tag event: %w", err)
	}

	if err = h.tagUseCase.SyncTag(ctx, &tagEvent); err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorLog("Failed to sync tag",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		return fmt.Errorf("failed to sync tag: %w", err)
	}

	tracing.TraceEvent(span, "Tag sync completed successfully")
	h.logger.InfoLog("Tag sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandleLevelSync 處理會員等級同步任務
func (h *WorkerHandler) HandleLevelSync(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return fmt.Errorf("task is empty")
	}
	taskID := getTaskID(task)

	ctx, span := tracing.TraceWorkerProcessing(ctx, queue.TypeLevelSync, taskID)
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Starting level sync processing")
	h.logger.InfoLog("Processing level sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))
	cloudEvent, err := parseCloudEvent(task.Payload(), span)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var levelEvent event.LevelSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &levelEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal level event: %w", err)
	}

	if err = h.levelUseCase.SyncLevel(ctx, &levelEvent); err != nil {
		h.logger.ErrorLog("Failed to sync level",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		return fmt.Errorf("failed to sync level: %w", err)
	}

	tracing.TraceEvent(span, "Level sync completed successfully")
	h.logger.InfoLog("Level sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

func parseCloudEvent(eventData []byte, span trace.Span) (*event.CloudEvent, error) {
	var cloudEvent event.CloudEvent
	if err := jsoniter.Unmarshal(eventData, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("unmarshal cloud event: %w", err)
	}
	tracing.RecordSpanAttributes(span,
		attribute.String("event.id", cloudEvent.ID),
		attribute.String("event.type", cloudEvent.Type),
		attribute.String("event.source", cloudEvent.Source))
	return &cloudEvent, nil
}
