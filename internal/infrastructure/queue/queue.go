package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/logger"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// 任務類型常量
const (
	TypeMerchantSync = "merchant:sync"
	TypePlayerSync   = "player:sync"
	TypeManagerSync  = "manager:sync"
)

// QueueService 佇列服務實現
type QueueService struct {
	client *asynq.Client
	logger *zap.Logger
}

// NewQueueService 創建佇列服務
func NewQueueService(cfg *config.Config, logger *zap.Logger) (serviceport.QueueService, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)

	logger.Info("Connecting to Redis",
		zap.String("redis_addr", redisAddr),
		zap.Int("redis_db", cfg.Redis.DB))

	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	client := asynq.NewClient(redisOpt)

	// 測試Redis連接
	inspector := asynq.NewInspector(redisOpt)
	_, err := inspector.Queues()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Successfully connected to Redis queue")

	return &QueueService{
		client: client,
		logger: logger,
	}, nil
}

// EnqueueMerchantSync 將商戶同步任務加入佇列
func (q *QueueService) EnqueueMerchantSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeMerchantSync, data)
}

// EnqueuePlayerSync 將玩家同步任務加入佇列
func (q *QueueService) EnqueuePlayerSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypePlayerSync, data)
}

// EnqueueManagerSync 將管理員同步任務加入佇列
func (q *QueueService) EnqueueManagerSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeManagerSync, data)
}

// enqueueTask 通用方法，將任務加入佇列並添加追蹤
func (q *QueueService) enqueueTask(ctx context.Context, taskType string, data []byte) error {
	// 從當前上下文中獲取 span
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("messaging.destination", "redis_queue"),
		attribute.String("messaging.task_type", taskType),
	)

	// 提取事件ID並添加到span - 嘗試從上下文中獲取已解析的事件ID
	var eventID string
	if id, ok := ctx.Value("event_id").(string); ok && id != "" {
		// 如果上下文中已有事件ID，直接使用
		eventID = id
		span.SetAttributes(attribute.String("messaging.event_id", id))
	} else {
		// 否則從數據中解析
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			if id, ok := jsonData["id"].(string); ok {
				eventID = id
				span.SetAttributes(attribute.String("messaging.event_id", id))
			}
		}
	}

	// 將追蹤上下文注入數據中 - 使用更高效的方法
	tracedData, err := tracing.InjectTraceparentToJSON(ctx, data)
	if err == nil {
		data = tracedData
	}

	// 創建任務
	task := asynq.NewTask(taskType, data)

	// 記錄任務創建事件
	tracing.TraceEvent(span, "Task created for Redis queue")

	// 設置任務選項 - 改進的重試策略
	opts := []asynq.Option{
		// 指數退避重試策略，最多重試5次
		asynq.MaxRetry(5),
		// 使用自訂的重試延遲函數設定
		asynq.ProcessIn(0),              // 立即處理
		asynq.Timeout(30 * time.Second), // 任務超時設置
	}

	// 將任務加入佇列
	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, fmt.Sprintf("failed to enqueue task: %v", err))
		q.logger.Error("Failed to enqueue task",
			zap.String("task_type", taskType),
			zap.String("event_id", eventID),
			zap.Error(err))
		return fmt.Errorf("failed to enqueue %s task: %w", taskType, err)
	}

	// 記錄成功事件
	tracing.TraceEvent(span, "Task enqueued successfully",
		attribute.String("task.id", info.ID),
		attribute.String("task.queue", info.Queue))

	// 為任務添加更多屬性
	span.SetAttributes(
		attribute.String("task.id", info.ID),
		attribute.String("task.queue", info.Queue),
	)

	q.logger.Info("Enqueued task successfully",
		zap.String("task_type", taskType),
		zap.String("task_id", info.ID),
		zap.String("queue", info.Queue),
		zap.String("event_id", eventID))

	return nil
}

// WrapHandlerWithTracing 包裝處理器以添加追蹤功能
func WrapHandlerWithTracing(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		// 從任務中提取 traceparent
		data := task.Payload()
		ctxWithTrace := tracing.ExtractTraceContext(ctx, data)

		// 創建處理任務的 span
		taskID := "unknown"
		if w := task.ResultWriter(); w != nil {
			taskID = w.TaskID()
		} else {
			taskID = fmt.Sprintf("no-writer-%s", uuid.New().String()) // 保證唯一性
		}
		ctxWithTrace, span := tracing.TraceRedisToWorker(ctxWithTrace, task.Type(), taskID)
		defer span.End()

		// 記錄任務開始處理
		tracing.TraceEvent(span, "Starting worker task processing")
		span.SetAttributes(
			attribute.Int("task.payload_size_bytes", len(data)),
			// 修復：移除 Retried 方法的調用，因為它不存在
		)

		// 提取事件ID記錄在span中
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			if id, ok := jsonData["id"].(string); ok {
				span.SetAttributes(attribute.String("messaging.event_id", id))
			}
		}

		// 處理任務
		err := h.ProcessTask(ctxWithTrace, task)

		// 處理錯誤情況
		if err != nil {
			// 記錄錯誤
			span.RecordError(err)
			span.SetStatus(codes.Error, fmt.Sprintf("task processing failed: %v", err))

			// 檢查錯誤類型，決定是否需要重試
			if strings.Contains(err.Error(), "(will retry)") {
				// 可重試錯誤，例如暫時性的資源不可用
				tracing.TraceEvent(span, "Task processing failed, will retry",
					attribute.String("error", err.Error()))
				return fmt.Errorf("retriable error: %w", err)
			} else {
				// 無法重試的錯誤
				tracing.TraceEvent(span, "Task processing failed, will not retry",
					attribute.String("error", err.Error()))
				return err
			}
		}

		// 記錄成功處理
		tracing.TraceEvent(span, "Task processed successfully")
		return nil
	})
}

// Close 關閉佇列連接
func (q *QueueService) Close() error {
	return q.client.Close()
}

// NewWorkerServer 創建Worker服務器
func NewWorkerServer(cfg *config.Config, zapLogger *zap.Logger) (*asynq.Server, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)

	zapLogger.Info("Creating worker server",
		zap.String("redis_addr", redisAddr),
		zap.Int("redis_db", cfg.Redis.DB))

	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// 創建日誌適配器
	asynqLogger := logger.NewAsynqLoggerAdapter(zapLogger)

	// 設置服務器配置
	concurrency := 10
	queues := map[string]int{
		"default":  5,  // 默認優先級
		"critical": 10, // 高優先級
	}

	zapLogger.Info("Worker server configuration",
		zap.Int("concurrency", concurrency),
		zap.Any("queues", queues))

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: concurrency,
			Queues:      queues,
			Logger:      asynqLogger,
			RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
				// 增加指標記錄重試
				taskID := "unknown"
				if w := task.ResultWriter(); w != nil {
					taskID = w.TaskID()
				} else {
					taskID = fmt.Sprintf("no-writer-%s", uuid.New().String()) // 保證唯一性
				}
				zapLogger.Info("Task retry scheduled",
					zap.String("task_id", taskID),
					zap.String("task_type", task.Type()),
					zap.Int("retry_count", n),
					zap.Error(err))
				return time.Duration(n*n) * time.Second // 指數退避策略
			},
		},
	)

	// 測試Redis連接
	inspector := asynq.NewInspector(redisOpt)
	_, err := inspector.Queues()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	zapLogger.Info("Worker server created successfully")
	return server, nil
}
