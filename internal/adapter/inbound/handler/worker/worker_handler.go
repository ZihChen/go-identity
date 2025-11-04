package worker

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
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
	merchantUseCase inbound.MerchantUseCase
	playerUseCase   inbound.PlayerUseCase
	managerUseCase  inbound.ManagerUseCase
	tagUseCase      inbound.TagUseCase
	levelUseCase    inbound.PlayerLevelUseCase
	agentUseCase    inbound.AgentUseCase
	logger          infrastructure.Logger
	tracing         infrastructure.TracingService
}

// NewWorkerHandler 創建Worker Handler
func NewWorkerHandler(
	merchantUseCase inbound.MerchantUseCase,
	playerUseCase inbound.PlayerUseCase,
	managerUseCase inbound.ManagerUseCase,
	tagUseCase inbound.TagUseCase,
	levelUseCase inbound.PlayerLevelUseCase,
	agentUseCase inbound.AgentUseCase,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) *WorkerHandler {
	return &WorkerHandler{
		merchantUseCase: merchantUseCase,
		playerUseCase:   playerUseCase,
		managerUseCase:  managerUseCase,
		tagUseCase:      tagUseCase,
		levelUseCase:    levelUseCase,
		agentUseCase:    agentUseCase,
		logger:          logger,
		tracing:         tracing,
	}
}

func (h *WorkerHandler) RegisterHandlers(mux *asynq.ServeMux) {
	// 直接使用產組器，不需要追蹤包裝器
	mux.Handle(
		queue.TypeMerchantSync,
		asynq.HandlerFunc(h.HandleMerchantSync),
	)
	mux.Handle(
		queue.TypePlayerSync,
		asynq.HandlerFunc(h.HandlePlayerSync),
	)
	mux.Handle(
		queue.TypeManagerSync,
		asynq.HandlerFunc(h.HandleManagerSync),
	)
	mux.Handle(
		queue.TypeTagSync,
		asynq.HandlerFunc(h.HandleTagSync),
	)
	mux.Handle(
		queue.TypeLevelSync,
		asynq.HandlerFunc(h.HandleLevelSync),
	)
	mux.Handle(
		queue.TypeAgentSync,
		asynq.HandlerFunc(h.HandleAgentSync),
	)

	h.logger.InfoLog("Registered worker handlers",
		h.logger.String("handler.merchant_sync", queue.TypeMerchantSync),
		h.logger.String("handler.player_sync", queue.TypePlayerSync),
		h.logger.String("handler.manager_sync", queue.TypeManagerSync),
		h.logger.String("handler.level_sync", queue.TypeLevelSync),
		h.logger.String("handler.tag_sync", queue.TypeTagSync),
		h.logger.String("handler.agent_sync", queue.TypeAgentSync))
}

// HandleMerchantSync 處理商戶同步任務
func (h *WorkerHandler) HandleMerchantSync(ctx context.Context, task *asynq.Task) error {
	taskID := getTaskID(task)

	// 創建處理任務的追蹤
	ctx, span := h.tracing.TraceWorkerProcessing(ctx, queue.TypeMerchantSync, taskID)
	defer h.tracing.SpanEnd(span)

	h.logger.InfoLog("Processing merchant sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracing)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var merchantEvent event.MerchantSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &merchantEvent); err != nil {
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal merchant event: %w", err)
	}

	h.tracing.TraceEvent(span, "Starting merchant sync processing")

	// 同步資料
	if err = h.merchantUseCase.SyncMerchant(ctx, &merchantEvent); err != nil {
		h.logger.ErrorLog("Failed to sync merchant",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync merchant: %w", err)
	}

	h.tracing.TraceEvent(span, "Merchant sync completed successfully")

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

	ctx, span := h.tracing.TraceWorkerProcessing(ctx, queue.TypePlayerSync, taskID)
	defer h.tracing.SpanEnd(span)

	h.logger.InfoLog("Processing player sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracing)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerEvent event.PlayerSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &playerEvent); err != nil {
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal player event: %w", err)
	}

	h.tracing.TraceEvent(span, "Starting player sync processing")

	if err = h.playerUseCase.SyncPlayer(ctx,
		&playerEvent); err != nil {
		h.logger.ErrorLog("Failed to sync player",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))

		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync player: %w", err)
	}

	if len(playerEvent.PlayerTags) != 0 {
		if err = h.tagUseCase.SyncPlayerTag(ctx,
			playerEvent.PlayerTags,
			playerEvent.GlobalMerchantID,
			playerEvent.Player.GlobalPlayerID); err != nil {
			h.tracing.RecordSpanError(span, err)
			return fmt.Errorf("failed to sync player tags relation: %w", err)
		}
	}

	h.tracing.TraceEvent(span, "Player sync completed successfully")

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

	ctx, span := h.tracing.TraceWorkerProcessing(ctx, queue.TypeManagerSync, taskID)
	defer h.tracing.SpanEnd(span)

	h.logger.InfoLog("Processing manager sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))
	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracing)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var managerEvent event.ManagerSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &managerEvent); err != nil {
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal manager event: %w", err)
	}

	h.tracing.TraceEvent(span, "Starting manager sync processing")

	// 同步資料
	if err = h.managerUseCase.SyncManager(ctx, &managerEvent); err != nil {
		h.logger.ErrorLog("Failed to sync manager",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync manager: %w", err)
	}

	h.tracing.TraceEvent(span, "Manager sync completed successfully")
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

	ctx, span := h.tracing.TraceWorkerProcessing(ctx, queue.TypeTagSync, taskID)
	defer h.tracing.SpanEnd(span)
	h.tracing.TraceEvent(span, "Starting level sync processing")

	h.logger.InfoLog("Processing tag sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))
	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracing)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var tagEvent event.TagSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &tagEvent); err != nil {
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal tag event: %w", err)
	}

	if err = h.tagUseCase.SyncTag(ctx, &tagEvent); err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorLog("Failed to sync tag",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		return fmt.Errorf("failed to sync tag: %w", err)
	}

	h.tracing.TraceEvent(span, "Tag sync completed successfully")
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

	ctx, span := h.tracing.TraceWorkerProcessing(ctx, queue.TypeLevelSync, taskID)
	defer h.tracing.SpanEnd(span)

	h.tracing.TraceEvent(span, "Starting level sync processing")
	h.logger.InfoLog("Processing level sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))
	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracing)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var levelEvent event.LevelSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &levelEvent); err != nil {
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal level event: %w", err)
	}

	if err = h.levelUseCase.SyncLevel(ctx, &levelEvent); err != nil {
		h.logger.ErrorLog("Failed to sync level",
			h.logger.String("task_id", taskID),
			h.logger.Error("err", err))
		return fmt.Errorf("failed to sync level: %w", err)
	}

	h.tracing.TraceEvent(span, "Level sync completed successfully")
	h.logger.InfoLog("Level sync task completed successfully",
		h.logger.String("task_id", taskID))
	return nil
}

// HandleAgentSync 處理代理同步任務
func (h *WorkerHandler) HandleAgentSync(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return fmt.Errorf("task is empty")
	}
	taskID := getTaskID(task)

	ctx, span := h.tracing.TraceWorkerProcessing(ctx, queue.TypeAgentSync, taskID)
	defer h.tracing.SpanEnd(span)

	h.logger.InfoLog("Processing agent sync task",
		h.logger.String("task_id", taskID),
		h.logger.Int("payload_size", len(task.Payload())))

	cloudEvent, err := parseCloudEvent(task.Payload(), span, h.tracing)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to parse cloud event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.String("data", string(task.Payload())))
		return err
	}

	dataBytes, err := jsoniter.Marshal(cloudEvent.Data)
	if err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Failed to marshal event data",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID),
			h.logger.Any("data", cloudEvent.Data))
		return fmt.Errorf("marshal event data: %w", err)
	}

	var agentEvent event.AgentSyncEvent
	if err = jsoniter.Unmarshal(dataBytes, &agentEvent); err != nil {
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("unmarshal agent event: %w", err)
	}

	h.tracing.TraceEvent(span, "Starting agent sync processing")

	// 驗證事件數據
	if err := h.validateAgentSyncEvent(&agentEvent); err != nil {
		h.tracing.RecordSpanError(span, err)
		h.logger.ErrorWithContext(ctx, "Invalid agent sync event",
			h.logger.Error("err", err),
			h.logger.String("task_id", taskID))
		return fmt.Errorf("invalid event data: %w", err)
	}

	// 同步資料
	if err = h.agentUseCase.SyncAgentData(ctx, &agentEvent); err != nil {
		h.logger.ErrorLog("Failed to sync agent",
			h.logger.String("task_id", taskID),
			h.logger.String("global_agent_id", agentEvent.Agent.GlobalAgentID),
			h.logger.Error("err", err))
		h.tracing.RecordSpanError(span, err)
		return fmt.Errorf("failed to sync agent: %w", err)
	}

	h.tracing.TraceEvent(span, "Agent sync completed successfully")
	h.logger.InfoLog("Agent sync task completed successfully",
		h.logger.String("task_id", taskID),
		h.logger.String("global_agent_id", agentEvent.Agent.GlobalAgentID))
	return nil
}

// validateAgentSyncEvent - 驗證代理同步事件數據完整性
func (h *WorkerHandler) validateAgentSyncEvent(event *event.AgentSyncEvent) error {
	if event.Agent.GlobalAgentID == "" {
		return fmt.Errorf("global_agent_id is required")
	}

	if event.Agent.Account == "" {
		return fmt.Errorf("account is required")
	}

	if event.Merchant.ID == 0 {
		return fmt.Errorf("merchant_id is required")
	}

	return nil
}

func parseCloudEvent(
	eventData []byte,
	span trace.Span,
	tracing infrastructure.TracingService,
) (*event.CloudEvent, error) {
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
