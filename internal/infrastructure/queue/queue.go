package queue

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
)

// 任務類型常量
const (
	TypeMerchantSync = "merchant:sync"
	TypePlayerSync   = "player:sync"
	TypeManagerSync  = "manager:sync"
	TypeLevelSync    = "level:sync"
	TypeTagSync      = "tag:sync"
	TypeAgentSync    = "agent:sync"
)

// QueueService 佇列服務實現
type QueueService struct {
	client  *asynq.Client
	logger  infrastructure.Logger
	tracing infrastructure.TracingService
	cfg     *config.Config
}

var _ service.QueueService = (*QueueService)(nil)

// NewQueueService 創建佇列服務
func NewQueueService(
	cfg *config.Config,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) (service.QueueService, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)

	// 根據環境獲取動態配置
	workerConfig := getWorkerConfigByEnv(cfg.App.Env)

	logger.InfoLog("Connecting to Redis",
		logger.String("redis_addr", redisAddr),
		logger.Int("redis_db", cfg.Redis.DB),
		logger.String("environment", cfg.App.Env),
		logger.Int("pool_size", workerConfig.RedisPoolSize))

	redisOpt := asynq.RedisClientOpt{
		Addr:         redisAddr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     workerConfig.RedisPoolSize,
		DialTimeout:  workerConfig.RedisDialTimeout,
		ReadTimeout:  workerConfig.RedisReadTimeout,
		WriteTimeout: workerConfig.RedisWriteTimeout,
	}

	client := asynq.NewClient(redisOpt)

	logger.InfoLog("Redis queue service created successfully")

	return &QueueService{
		client:  client,
		logger:  logger,
		tracing: tracing,
		cfg:     cfg,
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

// EnqueueAgentSync 將代理同步任務加入佇列
func (q *QueueService) EnqueueAgentSync(ctx context.Context, data []byte) error {
	return q.enqueueTask(ctx, TypeAgentSync, data)
}

// enqueueTask 通用方法，將任務加入佇列並添加追蹤
func (q *QueueService) enqueueTask(ctx context.Context, taskType string, data []byte) error {
	ctx, span := q.tracing.StartSpan(ctx, "QueueService.enqueueTask")
	q.tracing.RecordSpanAttributes(span,
		entity.StringAttr("messaging.destination", "redis_queue"),
		entity.StringAttr("messaging.task_type", taskType),
	)

	// 提取事件ID並添加到span - 嘗試從上下文中獲取已解析的事件ID
	var eventID string
	if id, ok := ctx.Value("event_id").(string); ok && id != "" {
		// 如果上下文中已有事件ID，直接使用
		eventID = id
		q.tracing.RecordSpanAttributes(span, entity.StringAttr("messaging.event_id", id))
	} else {
		// 否則從數據中解析
		var jsonData map[string]interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			if id, ok := jsonData["id"].(string); ok {
				eventID = id
				q.tracing.RecordSpanAttributes(span, entity.StringAttr("messaging.event_id", id))
			}
		}
	}

	// 將追蹤上下文注入數據中 - 使用更高效的方法
	tracedData, err := q.tracing.InjectTraceparentToJSON(ctx, data)
	if err == nil {
		data = tracedData
	}

	// 創建任務
	task := asynq.NewTask(taskType, data)

	// 記錄任務創建事件
	q.tracing.TraceEvent(span, "Task created for Redis queue")

	// 根據環境獲取任務配置
	workerConfig := getWorkerConfigByEnv(q.cfg.App.Env)

	// 設置環境感知的任務選項
	opts := []asynq.Option{
		asynq.MaxRetry(workerConfig.MaxRetries), // 環境感知的重試次數
		asynq.ProcessIn(0),                      // 立即處理
		asynq.Timeout(workerConfig.TaskTimeout), // 環境感知的任務超時
		asynq.Retention(0),                      // 任務完成後立即刪除
	}

	// 將任務加入佇列
	info, err := q.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		q.tracing.RecordSpanError(span, err)
		q.tracing.RecordSpanStatus(
			span,
			false,
			fmt.Sprintf("failed to enqueue task: %v", err),
		)
		q.logger.ErrorLog("Failed to enqueue task",
			q.logger.String("task_type", taskType),
			q.logger.String("event_id", eventID),
			q.logger.Error("err", err))
		return fmt.Errorf("failed to enqueue %s task: %w", taskType, err)
	}

	// 記錄成功事件
	q.tracing.TraceEvent(span, "Task enqueued successfully",
		entity.StringAttr("task.id", info.ID),
		entity.StringAttr("task.queue", info.Queue))

	q.tracing.RecordSpanAttributes(span,
		entity.StringAttr("task.id", info.ID),
		entity.StringAttr("task.queue", info.Queue),
	)

	q.logger.InfoLog("Enqueued task successfully",
		q.logger.String("task_type", taskType),
		q.logger.String("task_id", info.ID),
		q.logger.String("queue", info.Queue),
		q.logger.String("event_id", eventID),
		q.logger.String("environment", q.cfg.App.Env),
		q.logger.Int("max_retries", workerConfig.MaxRetries),
		q.logger.String("task_timeout", workerConfig.TaskTimeout.String()))

	return nil
}

// Close 關閉佇列連接
func (q *QueueService) Close() error {
	return q.client.Close()
}

// NewWorkerServer 創建Worker服務器
func NewWorkerServer(
	cfg *config.Config,
	logger infrastructure.Logger,
	failedTaskUseCase inbound.FailedTaskEventUseCase,
	redisManager infrastructure.CacheManager,
	tracing infrastructure.TracingService,
) (*asynq.Server, error) {
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)

	logger.InfoLog("Creating worker server",
		logger.String("redis_addr", redisAddr),
		logger.Int("redis_db", cfg.Redis.DB))

	// 根據環境獲取動態配置
	workerConfig := getWorkerConfigByEnv(cfg.App.Env)

	redisOpt := asynq.RedisClientOpt{
		Addr:         redisAddr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     workerConfig.RedisPoolSize,
		DialTimeout:  workerConfig.RedisDialTimeout,
		ReadTimeout:  workerConfig.RedisReadTimeout,
		WriteTimeout: workerConfig.RedisWriteTimeout,
	}

	logger.InfoLog("Environment-aware worker server configuration",
		logger.String("environment", cfg.App.Env),
		logger.Int("concurrency", workerConfig.Concurrency),
		logger.Any("queues", workerConfig.QueuePriorities),
		logger.String("task_timeout", workerConfig.TaskTimeout.String()),
		logger.Int("max_retries", workerConfig.MaxRetries),
		logger.Int("redis_pool_size", workerConfig.RedisPoolSize),
		logger.String("redis_dial_timeout", workerConfig.RedisDialTimeout.String()))

	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency:     workerConfig.Concurrency,
			Queues:          workerConfig.QueuePriorities,
			ShutdownTimeout: 30 * time.Second,
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
					logger.String("environment", cfg.App.Env),
				}

				if task != nil {
					logFields = append(logFields,
						logger.String("task_type", task.Type()),
						logger.String("payload", string(task.Payload())))
				}

				logger.InfoLog("Task retry scheduled with environment-aware strategy", logFields...)

				// 使用環境感知的重試延遲策略
				return workerConfig.RetryDelay(n)
			},
			ErrorHandler: asynq.ErrorHandlerFunc(
				func(ctx context.Context, task *asynq.Task, err error) {
					taskID := "unknown"

					// 方法1: 通過ResultWriter (首選方法)
					if w := task.ResultWriter(); w != nil {
						taskID = w.TaskID()
						logger.DebugLog(
							"TaskID obtained from ResultWriter",
							logger.String("task_id", taskID),
						)
					}

					// 方法2: 如果ResultWriter不可用，從payload中提取或生成唯一ID
					if taskID == "unknown" || taskID == "" {
						logger.WarnLog("ResultWriter unavailable, generating taskID from payload")
						taskID = generateTaskIDFromPayload(task.Type(), task.Payload())
						logger.DebugLog(
							"TaskID generated from payload",
							logger.String("task_id", taskID),
						)
					}

					logger.ErrorLog("Task processing failed - storing to DB and cleaning Redis",
						logger.String("task_id", taskID),
						logger.String("type", task.Type()),
						logger.Error("err", err))

					// 異步處理：存儲錯誤事件到DB + 清理Redis任務數據
					go func() {
						bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
						defer cancel()

						// 1. 記錄失敗任務事件到資料庫
						redisKey := fmt.Sprintf("asynq:default:t:%s", taskID)
						if createErr := failedTaskUseCase.CreateFailedTaskEventWithRedisInfo(
							bgCtx,
							taskID,
							task.Type(),
							"default", // 默認queue
							string(task.Payload()),
							err.Error(),
							redisKey,
							"failed", // 設置為失敗狀態
							0,        // ErrorHandler中的重試次數為0（不會重試）
						); createErr != nil {
							logger.ErrorLog("Failed to record failed task event",
								logger.Error("err", createErr),
								logger.String("task_id", taskID),
								logger.String("task_type", task.Type()))
						} else {
							logger.InfoLog("Failed task event recorded to DB",
								logger.String("task_id", taskID),
								logger.String("task_type", task.Type()))
						}

					}()
				},
			),
		},
	)

	logger.InfoLog("Worker server created successfully")
	return server, nil
}

// generateTaskIDFromPayload 從payload生成或提取taskID
func generateTaskIDFromPayload(taskType string, payload []byte) string {
	// 首先嘗試從JSON payload中提取事件ID
	var jsonData map[string]interface{}
	if err := json.Unmarshal(payload, &jsonData); err == nil {
		// 嘗試多個可能的ID字段名稱
		idFields := []string{"id", "event_id", "task_id", "messageId", "ID", "global_id"}
		for _, field := range idFields {
			if id, exists := jsonData[field]; exists {
				if idStr, ok := id.(string); ok && idStr != "" {
					return idStr
				}
			}
		}
	}

	// 如果無法提取ID，則生成一個基於payload內容的唯一標識符
	hash := md5.Sum(payload)
	hashStr := hex.EncodeToString(hash[:])

	// 結合任務類型和時間戳，確保唯一性
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s_%d_%s", taskType, timestamp, hashStr[:12])
}
