package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// 任務類型常量
const (
	TypeMerchantSync = "merchant:sync"
	TypePlayerSync   = "player:sync"
	TypeManagerSync  = "manager:sync"
	TypeLevelSync    = "level:sync"
	TypeTagSync      = "tag:sync"
)

// QueueService 佇列服務實現
type QueueService struct {
	client *asynq.Client
	logger infrastructure.Logger
}

// NewQueueService 創建佇列服務
func NewQueueService(
	cfg *config.Config,
	logger infrastructure.Logger,
) (service.QueueService, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)

	logger.InfoLog("Connecting to Redis",
		logger.String("redis_addr", redisAddr),
		logger.Int("redis_db", cfg.Redis.DB))

	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	client := asynq.NewClient(redisOpt)

	logger.InfoLog("Redis queue service created successfully")

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

// EnqueueLevelSync 將會員等級同步任務加入佇列
func (q *QueueService) EnqueueLevelSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeLevelSync, data)
}

// EnqueueTagSync 將會員標籤同步任務加入佇列
func (q *QueueService) EnqueueTagSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeTagSync, data)
}

// enqueueTask 通用方法，將任務加入佇列並添加追蹤
func (q *QueueService) enqueueTask(ctx context.Context, taskType string, data []byte) error {
	ctx, span := tracing.StartSpan(ctx, "QueueService.enqueueTask")
	tracing.RecordSpanAttributes(span,
		attribute.String("messaging.destination", "redis_queue"),
		attribute.String("messaging.task_type", taskType),
	)

	// 提取事件ID並添加到span - 嘗試從上下文中獲取已解析的事件ID
	var eventID string
	if id, ok := ctx.Value("event_id").(string); ok && id != "" {
		// 如果上下文中已有事件ID，直接使用
		eventID = id
		tracing.RecordSpanAttributes(span, attribute.String("messaging.event_id", id))
	} else {
		// 否則從數據中解析
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			if id, ok := jsonData["id"].(string); ok {
				eventID = id
				tracing.RecordSpanAttributes(span, attribute.String("messaging.event_id", id))
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
		asynq.Retention(0),              // 任務完成後立即刪除
	}

	// 將任務加入佇列
	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		tracing.RecordSpanError(span, err)
		tracing.RecordSpanStatus(span, codes.Error, fmt.Sprintf("failed to enqueue task: %v", err))
		q.logger.ErrorLog("Failed to enqueue task",
			q.logger.String("task_type", taskType),
			q.logger.String("event_id", eventID),
			q.logger.Error("err", err))
		return fmt.Errorf("failed to enqueue %s task: %w", taskType, err)
	}

	// 記錄成功事件
	tracing.TraceEvent(span, "Task enqueued successfully",
		attribute.String("task.id", info.ID),
		attribute.String("task.queue", info.Queue))

	tracing.RecordSpanAttributes(span,
		attribute.String("task.id", info.ID),
		attribute.String("task.queue", info.Queue),
	)

	q.logger.InfoLog("Enqueued task successfully",
		q.logger.String("task_type", taskType),
		q.logger.String("task_id", info.ID),
		q.logger.String("queue", info.Queue),
		q.logger.String("event_id", eventID))

	return nil
}

// WrapHandlerWithTracing 包裝處理器以添加追蹤功能
func WrapHandlerWithTracing(h asynq.Handler) asynq.Handler {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		if task == nil || len(task.Payload()) == 0 || task.Type() == "" {
			return asynq.SkipRetry
		}
		// 從任務中提取 traceparent
		data := task.Payload()
		ctxWithTrace := tracing.ExtractTraceContext(ctx, data)

		// 創建處理任務的 span
		ctxWithTrace, span := tracing.TraceRedisToWorker(
			ctxWithTrace,
			task.Type(),
			task.ResultWriter().TaskID(),
		)

		defer tracing.SpanEnd(span)

		tracing.TraceEvent(span, "Starting worker task processing")
		tracing.RecordSpanAttributes(span, attribute.Int("task.payload_size_bytes", len(data)))

		// 提取事件ID記錄在span中
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			if id, ok := jsonData["id"].(string); ok {
				tracing.RecordSpanAttributes(span, attribute.String("messaging.event_id", id))
			}
		}

		// 處理任務
		err := h.ProcessTask(ctxWithTrace, task)

		// 處理錯誤情況
		if err != nil {
			tracing.RecordSpanError(span, err)
			tracing.RecordSpanStatus(
				span,
				codes.Error,
				fmt.Sprintf("task processing failed: %v", err),
			)

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
func NewWorkerServer(cfg *config.Config, logger infrastructure.Logger) (*asynq.Server, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)

	logger.InfoLog("Creating worker server",
		logger.String("redis_addr", redisAddr),
		logger.Int("redis_db", cfg.Redis.DB))

	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// 設置服務器配置
	concurrency := 10
	queues := map[string]int{
		"default":  5,  // 默認優先級
		"critical": 10, // 高優先級
	}

	logger.InfoLog("Worker server configuration",
		logger.Int("concurrency", concurrency),
		logger.Any("queues", queues))

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: concurrency,
			Queues:      queues,
			RetryDelayFunc: func(n int, err error, task *asynq.Task) time.Duration {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorLog("Panic in RetryDelayFunc",
							logger.Any("recover", r),
							logger.Int("retry_count", n))
					}
				}()

				logFields := []*entity.LoggerFiled{
					logger.Int("retry_count", n),
					logger.Error("err", err),
				}

				if task != nil {
					logFields = append(logFields,
						logger.String("task_type", task.Type()),
						logger.String("payload", string(task.Payload())))
				}

				logger.InfoLog("Task retry scheduled", logFields...)

				// 使用指數退避策略，但設置上限
				delay := time.Duration(n*n) * time.Second
				maxDelay := 5 * time.Minute
				if delay > maxDelay {
					delay = maxDelay
				}
				return delay
			},
			ErrorHandler: asynq.ErrorHandlerFunc(
				func(ctx context.Context, task *asynq.Task, err error) {
					logger.ErrorLog("Task processing error",
						logger.String("type", task.Type()),
						logger.Error("err", err))
				},
			),
		},
	)

	logger.InfoLog("Worker server created successfully")
	return server, nil
}
