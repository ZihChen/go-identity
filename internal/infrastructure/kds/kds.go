package kds

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"runtime/debug"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	cfg "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

const (
	// 處理過的事件在Redis中保留的時間
	eventProcessedTTL = 24 * time.Hour
	// 批量提交checkpoint的記錄數
	checkpointBatchSize = 10
	// 最大重試次數
	maxRetries = 5
	// checkpoint key 前綴
	checkpointKeyPrefix = "kds:checkpoint:"
	// 處理過的事件key前綴
	processedEventKeyPrefix = "kds:processed:"
)

// KDSService KDS服務實現
type KDSService struct {
	client       *kinesis.Client
	dynamoClient *dynamodb.Client
	redisClient  *redis.Client
	streamName   string
	tableName    string
	partitionKey string
	sortKey      string
	config       *cfg.Config
	queueService serviceport.QueueService
	logger       *zap.Logger
	sLogger      infraport.Logger
}

// NewKDSService 創建KDS服務
func NewKDSService(config *cfg.Config, queueService serviceport.QueueService, redisClient *redis.Client, logger *zap.Logger, serviceLog infraport.Logger) (*KDSService, error) {
	// 創建AWS配置
	awsConfig, err := config.LoadAWSConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 創建Kinesis客戶端
	kinesisClient := kinesis.NewFromConfig(awsConfig)

	// 創建DynamoDB客戶端
	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	// 從ARN中提取stream名稱
	streamARN := config.AWS.KinesisStream
	streamName := streamARN
	if len(streamARN) > 0 {
		// 處理可能的ARN格式
		for i := len(streamARN) - 1; i >= 0; i-- {
			if streamARN[i] == '/' || streamARN[i] == ':' {
				streamName = streamARN[i+1:]
				break
			}
		}
	}

	logger.Info("Initialized KDS service",
		zap.String("stream_name", streamName),
		zap.String("stream_arn", streamARN),
		zap.String("dynamodb_table", config.AWS.DynamoDBTable))

	return &KDSService{
		client:       kinesisClient,
		dynamoClient: dynamoClient,
		redisClient:  redisClient,
		streamName:   streamName,
		tableName:    config.AWS.DynamoDBTable,
		partitionKey: config.AWS.PartitionKey,
		sortKey:      config.AWS.SortKey,
		config:       config,
		queueService: queueService,
		logger:       logger,
		sLogger:      serviceLog,
	}, nil
}

// Send 發送事件到KDS
func (k *KDSService) Send(ctx context.Context, data []byte, eventType string) error {
	ctx, span := tracing.StartSpan(ctx, "KDS.Send")
	defer span.End()

	// 添加屬性到 span
	span.SetAttributes(
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "send"),
		attribute.String("messaging.event_type", eventType),
		attribute.Int("messaging.payload_size_bytes", len(data)),
	)

	// 嘗試在 JSON 載荷中添加 traceparent
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		traceparent := tracing.GetTraceparent(ctx)
		if traceparent != "" {
			jsonData["traceparent"] = traceparent
			if newData, err := json.Marshal(jsonData); err == nil {
				data = newData
				span.SetAttributes(attribute.Bool("messaging.trace_propagated", true))
			}
		}
	}

	// 生成隨機分區鍵
	partitionKey := uuid.New().String()

	// 記錄事件到 span
	tracing.TraceEvent(span, "Sending message to KDS",
		attribute.String("messaging.partition_key", partitionKey),
	)

	res, err := k.client.PutRecord(ctx, &kinesis.PutRecordInput{
		Data:         data,
		StreamName:   aws.String(k.streamName),
		PartitionKey: aws.String(partitionKey),
	})
	if err != nil {
		k.logger.Error("Failed to put record to kinesis",
			zap.String("event_type", eventType),
			zap.Error(err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("put record to kinesis: %w", err)
	}

	// 記錄成功事件
	tracing.TraceEvent(span, "Message sent to KDS successfully")

	k.logger.Info("Published event to KDS",
		zap.String("event_type", eventType),
		zap.String("partition_key", partitionKey),
		zap.Any("response", res))

	return nil
}

// PublishMerchantSync 發布商戶同步事件
func (k *KDSService) PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer span.End()

	return k.publishEvent(ctx, event)
}

// PublishPlayerSync 發布玩家同步事件
func (k *KDSService) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer span.End()

	return k.publishEvent(ctx, event)
}

// PublishManagerSync 發布管理員同步事件
func (k *KDSService) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer span.End()

	return k.publishEvent(ctx, event)
}

// 內部方法：發布事件到KDS
func (k *KDSService) publishEvent(ctx context.Context, event *event.CloudEvent) error {
	// 確保 traceparent 在事件中
	event.TraceParent = tracing.GetTraceparent(ctx)

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return k.Send(ctx, eventBytes, event.Type)
}

// ConsumeMerchantSync 消費商戶同步事件
func (k *KDSService) ConsumeMerchantSync(ctx context.Context) error {
	return k.consumeEvents(ctx, k.config.Events.MerchantSync, k.queueService.EnqueueMerchantSync)
}

// ConsumePlayerSync 消費玩家同步事件
func (k *KDSService) ConsumePlayerSync(ctx context.Context) error {
	return k.consumeEvents(ctx, k.config.Events.PlayerSync, k.queueService.EnqueuePlayerSync)
}

// ConsumeManagerSync 消費管理員同步事件
func (k *KDSService) ConsumeManagerSync(ctx context.Context) error {
	return k.consumeEvents(ctx, k.config.Events.ManagerSync, k.queueService.EnqueueManagerSync)
}

// ConsumeAllEvents 消費所有事件類型
func (k *KDSService) ConsumeAllEvents(ctx context.Context) error {
	k.sLogger.InfoWithContext(ctx, "[Info][KDS][ConsumeAllEvents] Starting to consume all events from KDS",
		k.sLogger.String("stream_name", k.streamName))

	// 獲取分片信息
	shards, err := k.getShardIterators(ctx)
	if err != nil {
		return fmt.Errorf("failed to get shard iterators: %w", err)
	}

	// 為每個分片創建一個goroutine處理
	var shardWaiters sync.WaitGroup
	shardErrs := make(chan error, len(shards))

	// 處理每個分片
	for shardId, iterator := range shards {
		shardWaiters.Add(1)
		k.sLogger.InfoWithContext(ctx, "[Info][KDS][ConsumeAllEvents] Starting shard consumer", k.sLogger.String("shard_id", shardId))

		// 為每個分片創建一個協程
		go func(shardId, initialIterator string) {
			shardCtx, shardCancel := context.WithCancel(ctx)
			defer shardCancel()

			k.sLogger.InfoWithContext(shardCtx, fmt.Sprintf("[Info][KDS][ConsumeAllEvents] ShardsLoop ShardId:%s", shardId))

			defer shardWaiters.Done()
			defer func() {
				if r := recover(); r != nil {
					k.sLogger.ErrorWithContext(shardCtx, "[Error][KDS][ConsumeAllEvents] Recovered from panic in shard consumer",
						k.sLogger.String("shard_id", shardId),
						k.sLogger.Any("recover", r),
						k.sLogger.String("stacktrace", string(debug.Stack())))
				}
			}()

			currentIterator := initialIterator

			// 自適應退避策略參數設置
			backoffDuration, minBackoff, maxBackoff := 500*time.Millisecond, 500*time.Millisecond, 5*time.Second

			for {
				select {
				case <-shardCtx.Done():
					return
				default:
					// 獲取記錄
					recordsOutput, err := k.client.GetRecords(shardCtx, &kinesis.GetRecordsInput{
						ShardIterator: aws.String(currentIterator),
						Limit:         aws.Int32(100), // 每次獲取的記錄數
					})
					if err != nil {
						k.sLogger.ErrorWithContext(shardCtx, "[Error][KDS][ConsumeAllEvents] Failed to get records from shard",
							k.sLogger.String("shard_id", shardId),
							k.sLogger.Error("error", err))

						// 遇到錯誤時增加退避時間
						backoffDuration = time.Duration(float64(backoffDuration) * 1.5)
						if backoffDuration > maxBackoff {
							backoffDuration = maxBackoff
						}
						time.Sleep(backoffDuration)

						// 重新獲取迭代器
						iterators, err := k.getShardIterators(shardCtx)
						if err != nil {
							k.sLogger.ErrorWithContext(shardCtx, "[Error][KDS][ConsumeAllEvents] Failed to refresh shard iterator",
								k.sLogger.String("shard_id", shardId),
								k.sLogger.Error("error", err))

							// 只有在上下文被取消時才報告錯誤
							if shardCtx.Err() == nil {
								shardErrs <- fmt.Errorf("failed to refresh shard iterator for shard %s: %w", shardId, err)
							}
							return
						}

						if newIterator, ok := iterators[shardId]; ok {
							currentIterator = newIterator
						} else {
							k.sLogger.ErrorWithContext(shardCtx, "[Error][KDS][ConsumeAllEvents] Shard no longer available", k.sLogger.String("shard_id", shardId))
							return
						}

						continue
					}

					// 處理記錄
					recordsCount := len(recordsOutput.Records)

					// 根據獲取的記錄數調整退避時間
					if recordsCount == 0 {
						// 沒有記錄，增加退避時間
						backoffDuration = time.Duration(float64(backoffDuration) * 1.2)
						if backoffDuration > maxBackoff {
							backoffDuration = maxBackoff
						}
					} else {
						// 有記錄，減少退避時間
						backoffDuration = time.Duration(float64(backoffDuration) * 0.8)
						if backoffDuration < minBackoff {
							backoffDuration = minBackoff
						}
					}

					// 批量處理記錄，減少重複解析JSON
					for _, record := range recordsOutput.Records {
						// 從資料中取得上層traceparent作為事件追蹤用
						ctxWithTrace := tracing.ExtractTraceContext(ctx, record.Data)
						// 根據事件類型創建追踪
						eventCtx, eventSpan := tracing.StartSpan(ctxWithTrace, "KDS.EventRecord.StartProcessing")

						sequenceNumber := *record.SequenceNumber
						// 只解析一次JSON數據
						var jsonData map[string]interface{}
						if err := json.Unmarshal(record.Data, &jsonData); err != nil {
							k.sLogger.WarnWithContext(eventCtx, "[Warn][KDS][ConsumeAllEvents] Failed to unmarshal record data, skipping",
								k.sLogger.String("sequence_number", sequenceNumber),
								k.sLogger.Error("error", err))
							continue
						}

						// 提取事件ID和事件類型
						eventID, _ := jsonData["id"].(string)
						eventType, _ := jsonData["type"].(string)

						// 提取全局商戶ID
						var globalMerchantID string
						if data, ok := jsonData["data"].(map[string]interface{}); ok {
							globalMerchantID, _ = data["global_merchant_id"].(string)
						}

						// 如果無法確定事件類型，則跳過
						if eventType == "" {
							k.sLogger.WarnWithContext(eventCtx, "[Warn][KDS][ConsumeAllEvents] Skipping event with unknown type",
								k.sLogger.String("sequence_number", sequenceNumber),
								k.sLogger.String("event_id", eventID))
							continue
						}

						// 記錄消息數據
						eventSpan.SetAttributes(
							attribute.String("messaging.shard_id", shardId),
							attribute.String("messaging.sequence_number", sequenceNumber),
							attribute.String("messaging.event_id", eventID),
							attribute.String("messaging.event_type", eventType),
						)

						// 檢查該事件是否已處理過（去重)
						processed, err := k.isEventProcessed(eventCtx, eventID)
						if err != nil {
							k.sLogger.WarnWithContext(eventCtx, "[Warn][KDS][ConsumeAllEvents] Failed to check if event is processed, will process anyway",
								k.sLogger.String("sequence_number", sequenceNumber),
								k.sLogger.String("event_id", eventID),
								k.sLogger.Error("error", err))
						}

						// 事件已被處理則跳過
						if processed {
							k.sLogger.WarnWithContext(eventCtx, "[Warn][KDS][ConsumeAllEvents] Skipping already processed event",
								k.sLogger.String("sequence_number", sequenceNumber),
								k.sLogger.String("event_id", eventID),
								k.sLogger.String("event_type", eventType))
							eventSpan.End()
							continue
						}

						// 將事件ID添加到上下文中，避免隊列服務重複解析JSON
						msgCtxWithID := context.WithValue(eventCtx, "event_id", eventID)

						// 根據事件類型選擇合適的處理函數
						var enqueueErr error
						switch {
						case eventType == k.config.Events.IdentityMerchantSync:
							enqueueErr = k.queueService.EnqueueMerchantSync(msgCtxWithID, record.Data)
						case eventType == k.config.Events.IdentityPlayerSync:
							enqueueErr = k.queueService.EnqueuePlayerSync(msgCtxWithID, record.Data)
						case eventType == k.config.Events.IdentityManagerSync:
							enqueueErr = k.queueService.EnqueueManagerSync(msgCtxWithID, record.Data)
						default:
							k.sLogger.WarnWithContext(eventCtx, "[Warn][KDS][ConsumeAllEvents] Unknown event type, skipping",
								k.sLogger.String("event_id", eventID),
								k.sLogger.String("event_type", eventType))
							eventSpan.End()
							continue
						}

						// 檢查入隊錯誤
						if enqueueErr != nil {
							k.sLogger.ErrorWithContext(eventCtx, "[Error][KDS][ConsumeAllEvents] Failed to enqueue message",
								k.sLogger.String("event_type", eventType),
								k.sLogger.String("event_id", eventID),
								k.sLogger.String("sequence_number", sequenceNumber),
								k.sLogger.Error("error", enqueueErr))
							eventSpan.RecordError(enqueueErr)
							eventSpan.End()
							continue
						}

						// 標記事件為已處理
						if err := k.markEventProcessed(eventCtx, eventID); err != nil {
							k.sLogger.WarnWithContext(eventCtx, "[Warn][KDS][ConsumeAllEvents] Failed to mark event as processed",
								k.sLogger.String("event_id", eventID),
								k.sLogger.String("sequence_number", sequenceNumber),
								k.sLogger.Error("error", err))
						}

						// 記錄成功事件
						tracing.TraceEvent(eventSpan, "Message enqueued to Redis successfully")

						k.sLogger.InfoWithContext(eventCtx, "[Info][KDS][ConsumeAllEvents] Consumed event from KDS and enqueued to Redis",
							k.sLogger.String("event_type", eventType),
							k.sLogger.String("event_id", eventID),
							k.sLogger.String("sequence_number", sequenceNumber))

						// 更新檢查點
						if err := k.updateCheckpoint(eventCtx, shardId, sequenceNumber); err != nil {
							k.sLogger.WarnWithContext(eventCtx, "[Warn][KDS][ConsumeAllEvents] Failed to update checkpoint",
								k.sLogger.String("shard_id", shardId),
								k.sLogger.String("sequence_number", sequenceNumber),
								k.sLogger.Error("error", err))
						} else {
							k.sLogger.InfoWithContext(eventCtx, "[Info][KDS][ConsumeAllEvents] Updated checkpoint",
								k.sLogger.String("shard_id", shardId),
								k.sLogger.String("sequence_number", sequenceNumber),
								k.sLogger.String("global_merchant_id", globalMerchantID))
						}
						eventSpan.End()
					}

					// 獲取下一個迭代器
					if recordsOutput.NextShardIterator != nil {
						currentIterator = *recordsOutput.NextShardIterator
					} else {
						// 分片已關閉
						k.logger.Info("Shard has been closed", zap.String("shard_id", shardId))
						return
					}
					// 使用自適應退避策略：避免過度頻繁請求造成資源消耗
					time.Sleep(backoffDuration)
				}
			}
		}(shardId, iterator)
	}

	// 等待所有分片處理完成或出錯
	go func() {
		shardWaiters.Wait()
		close(shardErrs)
		k.logger.Info("KDS consumer has been stopped")
	}()

	// 檢查是否有錯誤
	for err := range shardErrs {
		if err != nil {
			k.logger.Error("Error in shard processing", zap.Error(err))
			return err
		}
	}
	return nil
}

// 內部方法：从KDS消費事件並轉發到Redis隊列
func (k *KDSService) consumeEvents(ctx context.Context, eventType string, enqueueFunc func(ctx context.Context, data []byte) error) error {
	rootCtx, rootSpan := tracing.StartSpan(ctx, "KDS.ConsumeEvents."+eventType)
	defer rootSpan.End()

	rootSpan.SetAttributes(
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.consumer_group", "ms-identity-cat"),
		attribute.String("messaging.event_type", eventType),
	)

	k.logger.Info("Starting to consume events from KDS",
		zap.String("event_type", eventType),
		zap.String("stream_name", k.streamName))

	// 獲取分片信息
	shards, err := k.getShardIterators(rootCtx)
	if err != nil {
		rootSpan.RecordError(err)
		return fmt.Errorf("failed to get shard iterators: %w", err)
	}

	// 為每個分片創建一個goroutine處理
	var shardWaiters sync.WaitGroup
	shardErrs := make(chan error, len(shards))

	// 處理每個分片
	for shardId, iterator := range shards {
		shardWaiters.Add(1)
		k.logger.Info("Starting shard consumer", zap.String("shard_id", shardId))

		// 為每個分片創建一個協程
		go func(shardId, initialIterator string) {
			defer shardWaiters.Done()
			defer func() {
				if r := recover(); r != nil {
					k.sLogger.ErrorWithContext(rootCtx, "Recovered from panic in shard consumer",
						k.sLogger.String("shard_id", shardId),
						k.sLogger.Any("recover", r),
						k.sLogger.String("stacktrace", "stack trace omitted"))
				}
			}()
			shardCtx, shardCancel := context.WithCancel(rootCtx)
			defer shardCancel()

			currentIterator := initialIterator
			//recordCount := 0

			for {
				select {
				case <-shardCtx.Done():
					k.logger.Info("Stopping shard processing due to context cancellation",
						zap.String("shard_id", shardId))
					return
				default:
					// 獲取記錄
					recordsOutput, err := k.client.GetRecords(shardCtx, &kinesis.GetRecordsInput{
						ShardIterator: aws.String(currentIterator),
						Limit:         aws.Int32(100), // 每次獲取的記錄數
					})

					if err != nil {
						k.logger.Error("Failed to get records from shard",
							zap.String("shard_id", shardId),
							zap.Error(err))

						// 遇到錯誤時暫停一下再重試
						time.Sleep(1 * time.Second)

						// 重新獲取迭代器
						iterators, err := k.getShardIterators(shardCtx)
						if err != nil {
							k.logger.Error("Failed to refresh shard iterator",
								zap.String("shard_id", shardId),
								zap.Error(err))

							// 只有在上下文被取消時才報告錯誤
							if shardCtx.Err() == nil {
								shardErrs <- fmt.Errorf("failed to refresh shard iterator for shard %s: %w", shardId, err)
							}
							return
						}

						if newIterator, ok := iterators[shardId]; ok {
							currentIterator = newIterator
						} else {
							k.logger.Error("Shard no longer available",
								zap.String("shard_id", shardId))
							return
						}

						continue
					}

					// 處理獲取的記錄
					for _, record := range recordsOutput.Records {
						sequenceNumber := *record.SequenceNumber
						msgCtx, msgSpan := tracing.TraceKDSToRedis(shardCtx, eventType, "")

						// 記錄消息數據
						msgSpan.SetAttributes(
							attribute.String("messaging.shard_id", shardId),
							attribute.String("messaging.sequence_number", sequenceNumber),
							attribute.Int("messaging.payload_size_bytes", len(record.Data)),
						)

						// 提取事件ID和全局商戶ID
						var eventID string
						var globalMerchantID string
						var jsonData map[string]interface{}
						if err := json.Unmarshal(record.Data, &jsonData); err == nil {
							if id, ok := jsonData["id"].(string); ok {
								eventID = id
								msgSpan.SetAttributes(attribute.String("messaging.event_id", id))
							}
							// 提取 global_merchant_id
							if data, ok := jsonData["data"].(map[string]interface{}); ok {
								if gmid, ok := data["global_merchant_id"].(string); ok {
									globalMerchantID = gmid
								}
							}
						}

						// 檢查該事件是否已處理過（去重）
						processed, err := k.isEventProcessed(msgCtx, eventID)
						if err != nil {
							k.logger.Warn("Failed to check if event is processed, will process anyway",
								zap.String("event_id", eventID),
								zap.String("sequence_number", sequenceNumber),
								zap.Error(err))
						}

						if processed {
							k.logger.Info("Skipping already processed event",
								zap.String("event_id", eventID),
								zap.String("event_type", eventType),
								zap.String("sequence_number", sequenceNumber))
							msgSpan.End()
							continue
						}

						// 提取追蹤上下文
						msgCtx = tracing.ExtractTraceContext(msgCtx, record.Data)

						// 記錄重要事件
						tracing.TraceEvent(msgSpan, "Message received from KDS")

						// 將消息加入隊列
						if err := enqueueFunc(msgCtx, record.Data); err != nil {
							k.logger.Error("Failed to enqueue message",
								zap.String("event_type", eventType),
								zap.String("event_id", eventID),
								zap.String("sequence_number", sequenceNumber),
								zap.Error(err))
							msgSpan.RecordError(err)
							msgSpan.End()
							continue
						}

						// 標記事件為已處理
						if err := k.markEventProcessed(msgCtx, eventID); err != nil {
							k.logger.Warn("Failed to mark event as processed",
								zap.String("event_id", eventID),
								zap.String("sequence_number", sequenceNumber),
								zap.Error(err))
						}

						// 記錄成功事件
						tracing.TraceEvent(msgSpan, "Message enqueued to Redis successfully")

						k.logger.Info("Consumed event from KDS and enqueued to Redis",
							zap.String("event_type", eventType),
							zap.String("event_id", eventID),
							zap.String("sequence_number", sequenceNumber))

						// 更新檢查點（每處理checkpointBatchSize條記錄更新一次）
						//recordCount++
						//if recordCount >= checkpointBatchSize {
						if err := k.updateCheckpoint(msgCtx, shardId, sequenceNumber); err != nil {
							k.logger.Warn("Failed to update checkpoint",
								zap.String("shard_id", shardId),
								zap.String("sequence_number", sequenceNumber),
								zap.Error(err))
						} else {
							k.logger.Info("Updated checkpoint",
								zap.String("shard_id", shardId),
								zap.String("sequence_number", sequenceNumber),
								zap.String("global_merchant_id", globalMerchantID))
							//recordCount = 0
						}
						//}

						msgSpan.End()
					}

					// 獲取下一個迭代器
					if recordsOutput.NextShardIterator != nil {
						currentIterator = *recordsOutput.NextShardIterator

						// 如果沒有記錄，短暫暫停避免過快請求
						if len(recordsOutput.Records) == 0 {
							time.Sleep(500 * time.Millisecond)
						}
					} else {
						// 分片已關閉
						k.logger.Info("Shard has been closed", zap.String("shard_id", shardId))
						return
					}
				}
			}
		}(shardId, iterator)
	}

	// 等待上下文取消或任一分片錯誤
	select {
	case <-ctx.Done():
		// 等待所有分片處理協程結束
		shardWaiters.Wait()
		return ctx.Err()
	case err := <-shardErrs:
		rootSpan.RecordError(err)
		return err
	}
}

// getShardIterators 獲取所有分片的迭代器
func (k *KDSService) getShardIterators(ctx context.Context) (map[string]string, error) {
	// 獲取所有分片
	shardsOutput, err := k.client.ListShards(ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(k.streamName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list shards: %w", err)
	}

	shardIterators := make(map[string]string)

	// 獲取每個分片的迭代器
	for _, shard := range shardsOutput.Shards {
		if shard.ShardId != nil && *shard.ShardId != "" {
			// 檢查DynamoDB中是否有這個分片的checkpoint
			checkpoint, err := k.getCheckpoint(ctx, *shard.ShardId)
			k.logger.Info("Current checkpoint",
				zap.String("shard_id", *shard.ShardId),
				zap.String("checkpoint", checkpoint))

			iteratorType := types.ShardIteratorTypeLatest
			var sequenceNumber *string = nil

			if err == nil && checkpoint != "" {
				// 如果找到checkpoint，從該位置開始讀取
				iteratorType = types.ShardIteratorTypeAfterSequenceNumber
				sequenceNumber = &checkpoint
				k.logger.Info("Found checkpoint, starting from after sequence number",
					zap.String("shard_id", *shard.ShardId),
					zap.String("sequence_number", checkpoint))
			} else {
				k.logger.Error("No checkpoint found, starting from latest position",
					zap.String("shard_id", *shard.ShardId),
					zap.Error(err))
			}

			iterOutput, err := k.client.GetShardIterator(ctx, &kinesis.GetShardIteratorInput{
				StreamName:             aws.String(k.streamName),
				ShardId:                shard.ShardId,
				ShardIteratorType:      iteratorType,
				StartingSequenceNumber: sequenceNumber,
			})
			if err != nil {
				k.logger.Error("Failed to get shard iterator",
					zap.String("shard_id", *shard.ShardId),
					zap.Error(err))
				continue
			}

			if iterOutput.ShardIterator != nil {
				shardIterators[*shard.ShardId] = *iterOutput.ShardIterator
			}
		}
	}

	if len(shardIterators) == 0 {
		return nil, fmt.Errorf("no valid shard iterators retrieved")
	}

	return shardIterators, nil
}

// getCheckpoint 從DynamoDB獲取指定分片的checkpoint
func (k *KDSService) getCheckpoint(ctx context.Context, shardId string) (string, error) {
	checkPointKey := k.composeDynamoDBKey(shardId)
	result, err := k.dynamoClient.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(k.tableName),
		Key: map[string]dynamodbtypes.AttributeValue{
			k.partitionKey: &dynamodbtypes.AttributeValueMemberS{Value: checkPointKey},
		},
	})
	if err != nil {
		k.logger.Error("Failed to get checkpoint from DynamoDB",
			zap.String("tableName", k.tableName),
			zap.String("partitionKey", k.partitionKey),
			zap.String("checkPointKey", checkPointKey),
			zap.String("shardId", shardId),
			zap.Error(err))
		return "", fmt.Errorf("failed to get checkpoint from DynamoDB: %w", err)
	}
	// 從DynamoDB項目中提取序列號
	sequenceAttr, ok := result.Item["sequence_number"]
	if !ok {
		return "", fmt.Errorf("sequence_number not found in checkpoint record")
	}

	sequenceNumber, ok := sequenceAttr.(*dynamodbtypes.AttributeValueMemberS)
	if !ok {
		return "", fmt.Errorf("invalid sequence_number type in checkpoint record")
	}

	return sequenceNumber.Value, nil
}

// updateCheckpoint 更新指定分片的checkpoint到DynamoDB
func (k *KDSService) updateCheckpoint(ctx context.Context, shardId string, sequenceNumber string) error {
	checkPointKey := k.composeDynamoDBKey(shardId)
	_, err := k.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(k.tableName),
		Item: map[string]dynamodbtypes.AttributeValue{
			k.partitionKey:    &dynamodbtypes.AttributeValueMemberS{Value: checkPointKey},
			"sequence_number": &dynamodbtypes.AttributeValueMemberS{Value: sequenceNumber},
			"updated_at":      &dynamodbtypes.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		},
	})
	if err != nil {
		return fmt.Errorf("[KDS][DynamoDB]Failed to update checkpoint in DynamoDB: %w", err)
	}
	k.logger.Info("[KDS][DynamoDB]Update checkpoint successfully", zap.String("tableName", k.tableName), zap.String("checkPointKey", checkPointKey))
	return nil
}

// isEventProcessed 檢查事件是否已被處理（用於去重）
func (k *KDSService) isEventProcessed(ctx context.Context, eventId string) (bool, error) {
	if eventId == "" {
		return false, nil // 無法檢查沒有ID的事件
	}

	key := processedEventKeyPrefix + eventId
	// 嘗試設置，如果已存在則返回false，表示之前已處理過
	success, err := k.redisClient.SetNX(ctx, key, "1", eventProcessedTTL).Result()
	if err != nil {
		return false, err
	}

	// 如果設置成功，返回false（未處理過）；如果設置失敗，返回true（已處理過）
	return !success, nil
}

// markEventProcessed 標記事件為已處理
func (k *KDSService) markEventProcessed(ctx context.Context, eventId string) error {
	if eventId == "" {
		return nil // 無法標記沒有ID的事件
	}

	key := processedEventKeyPrefix + eventId
	// 設置key，帶過期時間
	_, err := k.redisClient.Set(ctx, key, "1", eventProcessedTTL).Result()
	return err
}

// extractMerchantID 從事件中提取商戶 ID
func extractMerchantID(event *event.CloudEvent) string {
	if event == nil || event.Data == nil {
		return ""
	}

	// 將 data 轉換為 map
	dataBytes, _ := json.Marshal(event.Data)
	var dataMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &dataMap); err != nil {
		return ""
	}

	// 嘗試提取 global_merchant_id
	if globalID, ok := dataMap["global_merchant_id"].(string); ok {
		return globalID
	}

	return ""
}

func (k *KDSService) composeDynamoDBKey(shardId string) string {
	return fmt.Sprintf("%s_%s_%s", k.streamName, shardId, k.config.App.Name)
}
