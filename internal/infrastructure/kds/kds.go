package kds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/cenkalti/backoff/v4"
	"github.com/go-redsync/redsync/v4"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	cfg "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

const (
	// 處理過的事件在Redis中保留的時間
	eventProcessedTTL = 24 * time.Hour
	// 處理過的事件key前綴
	processedEventKeyPrefix = "kds:processed:"
	// 分佈式鎖Timeout時長
	consumerProcessedTTL = 1 * time.Minute
)

// KDSService KDS服務實現
type KDSService struct {
	client        *kinesis.Client
	dynamoClient  *dynamodb.Client
	redisManager  *redisCache.Manager
	streamName    string
	consumeStream string
	produceStream string
	tableName     string
	partitionKey  string
	sortKey       string
	config        *cfg.Config
	queueService  serviceport.QueueService
	logger        infraport.Logger
}

// NewKDSService 創建KDS服務
func NewKDSService(
	config *cfg.Config,
	queueService serviceport.QueueService,
	redisManager *redisCache.Manager,
	logger infraport.Logger,
) (*KDSService, error) {
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

	logger.InfoLog("Initialized KDS service",
		logger.String("stream_name", streamName),
		logger.String("stream_arn", streamARN),
		logger.String("dynamodb_table", config.AWS.DynamoDBTable))

	return &KDSService{
		client:        kinesisClient,
		dynamoClient:  dynamoClient,
		redisManager:  redisManager,
		streamName:    streamName,
		consumeStream: config.AWS.ConsumeStream,
		produceStream: config.AWS.ProduceStream,
		tableName:     config.AWS.DynamoDBTable,
		partitionKey:  config.AWS.PartitionKey,
		sortKey:       config.AWS.SortKey,
		config:        config,
		queueService:  queueService,
		logger:        logger,
	}, nil
}

// Send 發送事件到KDS
func (k *KDSService) Send(ctx context.Context, data []byte, eventType string) error {
	ctx, span := tracing.StartSpan(ctx, "KDS.Send")
	defer tracing.SpanEnd(span)

	// 添加屬性到 span
	tracing.RecordSpanAttributes(span,
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
				tracing.RecordSpanAttributes(
					span,
					attribute.Bool("messaging.trace_propagated", true),
				)
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
		StreamName:   aws.String(k.produceStream),
		PartitionKey: aws.String(partitionKey),
	})
	if err != nil {
		k.logger.ErrorLog("Failed to put record to kinesis",
			k.logger.String("event_type", eventType),
			k.logger.Error("err", err))
		tracing.RecordSpanError(span, err)
		tracing.RecordSpanStatus(span, codes.Error, err.Error())
		return fmt.Errorf("put record to kinesis: %w", err)
	}

	// 記錄成功事件
	tracing.TraceEvent(span, "Message sent to KDS successfully")

	k.logger.InfoLog("Published event to KDS",
		k.logger.String("event_type", eventType),
		k.logger.String("partition_key", partitionKey),
		k.logger.Any("response", res))

	return nil
}

// PublishMerchantSync 發布商戶同步事件
func (k *KDSService) PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	return k.publishEvent(ctx, event)
}

// PublishPlayerSync 發布玩家同步事件
func (k *KDSService) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	return k.publishEvent(ctx, event)
}

// PublishManagerSync 發布管理員同步事件
func (k *KDSService) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

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

// ConsumeAllEvents 消費所有事件類型
func (k *KDSService) ConsumeAllEvents(ctx context.Context) error {
	k.logger.InfoWithContext(
		ctx,
		"Starting to consume all events from KDS",
		k.logger.String("stream_name", k.consumeStream),
	)

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
		k.logger.InfoWithContext(
			ctx,
			"Starting shard consumer",
			k.logger.String("shard_id", shardId),
		)

		// 為每個 shard 建立分布式鎖
		mutexKey := fmt.Sprintf(consts.ShardMutexRedisKey, k.consumeStream, shardId)
		mutex, err := k.redisManager.GetMutex(mutexKey, consumerProcessedTTL)
		if err != nil {
			// Redis連線異常仍執行後面程序
			k.logger.WarnWithContext(ctx, "Failed to create mutex, skipping lock logic",
				k.logger.String("mutex_key", mutexKey),
				k.logger.Error("err", err))
			mutex = nil
		}

		// 嘗試取得鎖，若失敗則跳過
		if mutex != nil {
			if err := mutex.Lock(); err != nil {
				k.logger.WarnWithContext(ctx, "Shard already locked, skipping",
					k.logger.String("mutex_key", mutexKey),
					k.logger.Error("error", err))
				shardWaiters.Done()
				continue
			}
		}

		// 為每個分片創建一個協程
		go func(shardId, initialIterator string, shardMutex *redsync.Mutex) {
			shardCtx, shardCancel := context.WithCancel(ctx)
			defer shardCancel()
			defer shardWaiters.Done()
			defer func() {
				// 解鎖 shard mutex
				if ok, err := shardMutex.Unlock(); !ok || err != nil {
					k.logger.WarnWithContext(shardCtx, "Failed to release shard lock",
						k.logger.String("shard_id", shardId),
						k.logger.Error("err", err))
				}
			}()
			defer func() {
				if r := recover(); r != nil {
					k.logger.ErrorWithContext(
						shardCtx,
						"Recovered from panic in shard consumer",
						k.logger.String("shard_id", shardId),
						k.logger.Any("recover", r),
						k.logger.String("stacktrace", string(debug.Stack())),
					)
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
						Limit:         aws.Int32(1000),
					})
					if err != nil {
						k.logger.ErrorWithContext(
							shardCtx,
							"Failed to get records from shard",
							k.logger.String("shard_id", shardId),
							k.logger.Error("err", err),
						)

						// 遇到錯誤時增加退避時間
						backoffDuration = time.Duration(float64(backoffDuration) * 1.5)
						if backoffDuration > maxBackoff {
							backoffDuration = maxBackoff
						}
						time.Sleep(backoffDuration)

						// 重新獲取迭代器
						iterators, err := k.getShardIterators(shardCtx)
						if err != nil {
							k.logger.ErrorWithContext(
								shardCtx,
								"Failed to refresh shard iterator",
								k.logger.String("shard_id", shardId),
								k.logger.Error("error", err),
							)

							// 只有在上下文被取消時才報告錯誤
							if shardCtx.Err() == nil {
								shardErrs <- fmt.Errorf("failed to refresh shard iterator for shard %s: %w", shardId, err)
							}
							return
						}

						if newIterator, ok := iterators[shardId]; ok {
							currentIterator = newIterator
						} else {
							k.logger.ErrorWithContext(shardCtx, "Shard no longer available", k.logger.String("shard_id", shardId))
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
						eventCtx, eventSpan := tracing.StartSpan(
							ctxWithTrace,
							"KDS.EventRecord.StartProcessing",
						)

						sequenceNumber := *record.SequenceNumber
						// 只解析一次JSON數據
						var jsonData map[string]interface{}
						if err := json.Unmarshal(record.Data, &jsonData); err != nil {
							k.logger.WarnWithContext(
								eventCtx,
								"Failed to unmarshal record data, skipping",
								k.logger.String("sequence_number", sequenceNumber),
								k.logger.Error("error", err),
							)
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
							k.logger.WarnWithContext(
								eventCtx,
								"Skipping event with unknown type",
								k.logger.String("sequence_number", sequenceNumber),
								k.logger.String("event_id", eventID),
							)
							continue
						}

						// 記錄消息數據
						tracing.RecordSpanAttributes(eventSpan,
							attribute.String("messaging.shard_id", shardId),
							attribute.String("messaging.sequence_number", sequenceNumber),
							attribute.String("messaging.event_id", eventID),
							attribute.String("messaging.event_type", eventType),
						)

						// 檢查該事件是否已處理過（去重)
						processed, err := k.isEventProcessed(eventCtx, eventID)
						if err != nil {
							k.logger.WarnWithContext(
								eventCtx,
								"Failed to check if event is processed, will process anyway",
								k.logger.String("sequence_number", sequenceNumber),
								k.logger.String("event_id", eventID),
								k.logger.Error("error", err),
							)
						}

						// 事件已被處理則跳過
						if processed {
							k.logger.WarnWithContext(
								eventCtx,
								"Skipping already processed event",
								k.logger.String("sequence_number", sequenceNumber),
								k.logger.String("event_id", eventID),
								k.logger.String("event_type", eventType),
							)
							tracing.SpanEnd(eventSpan)
							continue
						}

						// 將事件ID添加到上下文中，避免隊列服務重複解析JSON
						msgCtxWithID := context.WithValue(eventCtx, consts.EventIDKey, eventID)

						// 根據事件類型選擇合適的處理函數
						var enqueueErr error
						switch eventType {
						case k.config.Events.MerchantSync:
							enqueueErr = k.queueService.EnqueueMerchantSync(
								msgCtxWithID,
								record.Data,
							)
						case k.config.Events.PlayerSync:
							enqueueErr = k.queueService.EnqueuePlayerSync(msgCtxWithID, record.Data)
						case k.config.Events.ManagerSync:
							enqueueErr = k.queueService.EnqueueManagerSync(
								msgCtxWithID,
								record.Data,
							)
						default:
							// TODO 新增checkpoint marked
							k.logger.WarnWithContext(
								eventCtx,
								"Unknown event type, skipping",
								k.logger.String("event_id", eventID),
								k.logger.String("event_type", eventType),
							)
							tracing.SpanEnd(eventSpan)
							continue
						}

						// 檢查入隊錯誤
						if enqueueErr != nil {
							k.logger.ErrorWithContext(
								eventCtx,
								"Failed to enqueue message",
								k.logger.String("event_type", eventType),
								k.logger.String("event_id", eventID),
								k.logger.String("sequence_number", sequenceNumber),
								k.logger.Error("error", enqueueErr),
							)
							tracing.RecordSpanError(eventSpan, enqueueErr)
							tracing.SpanEnd(eventSpan)
							continue
						}

						// 標記事件為已處理
						if err := k.markEventProcessed(eventCtx, eventID); err != nil {
							k.logger.WarnWithContext(
								eventCtx,
								"Failed to mark event as processed",
								k.logger.String("event_id", eventID),
								k.logger.String("sequence_number", sequenceNumber),
								k.logger.Error("error", err),
							)
						}

						// 記錄成功事件
						tracing.TraceEvent(eventSpan, "Message enqueued to Redis successfully")

						k.logger.InfoWithContext(
							eventCtx,
							"Consumed event from KDS and enqueued to Redis",
							k.logger.String("event_type", eventType),
							k.logger.String("event_id", eventID),
							k.logger.String("sequence_number", sequenceNumber),
						)

						// 更新檢查點
						if err := k.updateCheckpoint(eventCtx, shardId, sequenceNumber); err != nil {
							k.logger.WarnWithContext(
								eventCtx,
								"Failed to update checkpoint",
								k.logger.String("shard_id", shardId),
								k.logger.String("sequence_number", sequenceNumber),
								k.logger.Error("error", err),
							)
						} else {
							k.logger.InfoWithContext(eventCtx, "Updated checkpoint",
								k.logger.String("shard_id", shardId),
								k.logger.String("sequence_number", sequenceNumber),
								k.logger.String("global_merchant_id", globalMerchantID))
						}
						tracing.SpanEnd(eventSpan)
					}

					// 獲取下一個迭代器
					if recordsOutput.NextShardIterator != nil {
						currentIterator = *recordsOutput.NextShardIterator
					} else {
						// 分片已關閉
						k.logger.InfoLog("Shard has been closed", k.logger.String("shard_id", shardId))
						return
					}
					// 使用自適應退避策略：避免過度頻繁請求造成資源消耗
					time.Sleep(backoffDuration)
				}
			}
		}(shardId, iterator, mutex)
	}

	// 等待所有分片處理完成或出錯
	go func() {
		shardWaiters.Wait()
		close(shardErrs)
		k.logger.InfoLog("KDS consumer has been stopped")
	}()

	// 檢查是否有錯誤
	for err := range shardErrs {
		if err != nil {
			k.logger.ErrorLog("Error in shard processing", k.logger.Error("err", err))
			return err
		}
	}
	return nil
}

func (k *KDSService) getRecordsWithRetry(
	ctx context.Context,
	shardIterator string,
) (*kinesis.GetRecordsOutput, error) {
	var records *kinesis.GetRecordsOutput

	// 配置指數退避策略
	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.InitialInterval = 1 * time.Second // 初始重試間隔
	exponentialBackoff.MaxInterval = 30 * time.Second    // 最大重試間隔
	exponentialBackoff.MaxElapsedTime = 2 * time.Minute  // 最大總重試時間
	exponentialBackoff.Multiplier = 2.0                  // 每次重試間隔的倍數

	operation := func() error {
		var err error
		records, err = k.client.GetRecords(ctx, &kinesis.GetRecordsInput{
			ShardIterator: aws.String(shardIterator),
			Limit:         aws.Int32(10000),
		})

		if err != nil {
			// 檢查上下文超時
			if errors.Is(err, context.DeadlineExceeded) {
				k.logger.ErrorWithContext(
					ctx,
					"GetRecords timeout",
					k.logger.Error("err", err),
				)
				return err
			}

			// 檢查網絡超時
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				k.logger.ErrorWithContext(
					ctx,
					"Network timeout",
					k.logger.Error("err", err),
				)
				return err
			}

			// 檢查是否為可重試的 AWS 錯誤
			var throughputErr *types.ProvisionedThroughputExceededException
			if errors.As(err, &throughputErr) {
				k.logger.WarnWithContext(
					ctx,
					"Throughput exceeded, will retry",
					k.logger.Error("err", err),
				)
				return err
			}
			// 其他錯誤視為永久性錯誤，不再重試
			return backoff.Permanent(err)
		}
		return nil
	}

	err := backoff.Retry(operation, exponentialBackoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get records after retries: %w", err)
	}
	return records, nil
}

// getShardIterators 獲取所有分片的迭代器
func (k *KDSService) getShardIterators(ctx context.Context) (map[string]string, error) {
	// 獲取所有分片
	shardsOutput, err := k.client.ListShards(ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(k.consumeStream),
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
			k.logger.InfoLog("Current checkpoint",
				k.logger.String("shard_id", *shard.ShardId),
				k.logger.String("checkpoint", checkpoint))

			iteratorType := types.ShardIteratorTypeLatest
			var sequenceNumber *string = nil

			if err == nil && checkpoint != "" {
				// 如果找到checkpoint，從該位置開始讀取
				iteratorType = types.ShardIteratorTypeAfterSequenceNumber
				sequenceNumber = &checkpoint
				k.logger.InfoLog("Found checkpoint, starting from after sequence number",
					k.logger.String("shard_id", *shard.ShardId),
					k.logger.String("sequence_number", checkpoint))
			} else {
				k.logger.ErrorLog("No checkpoint found, starting from latest position",
					k.logger.String("shard_id", *shard.ShardId),
					k.logger.Error("err", err))
			}

			iterOutput, err := k.client.GetShardIterator(ctx, &kinesis.GetShardIteratorInput{
				StreamName:             aws.String(k.consumeStream),
				ShardId:                shard.ShardId,
				ShardIteratorType:      iteratorType,
				StartingSequenceNumber: sequenceNumber,
			})
			if err != nil {
				k.logger.ErrorLog("Failed to get shard iterator",
					k.logger.String("shard_id", *shard.ShardId),
					k.logger.Error("err", err))
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
		k.logger.ErrorLog("Failed to get checkpoint from DynamoDB",
			k.logger.String("tableName", k.tableName),
			k.logger.String("partitionKey", k.partitionKey),
			k.logger.String("checkPointKey", checkPointKey),
			k.logger.String("shardId", shardId),
			k.logger.Error("err", err))
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
func (k *KDSService) updateCheckpoint(
	ctx context.Context,
	shardId string,
	sequenceNumber string,
) error {
	checkPointKey := k.composeDynamoDBKey(shardId)
	_, err := k.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(k.tableName),
		Item: map[string]dynamodbtypes.AttributeValue{
			k.partitionKey:    &dynamodbtypes.AttributeValueMemberS{Value: checkPointKey},
			"sequence_number": &dynamodbtypes.AttributeValueMemberS{Value: sequenceNumber},
			"updated_at": &dynamodbtypes.AttributeValueMemberS{
				Value: time.Now().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update checkpoint in DynamoDB: %w", err)
	}
	k.logger.InfoLog(
		"Update checkpoint successfully",
		k.logger.String("tableName", k.tableName),
		k.logger.String("checkPointKey", checkPointKey),
	)
	return nil
}

// isEventProcessed 檢查事件是否已被處理（用於去重）
func (k *KDSService) isEventProcessed(ctx context.Context, eventId string) (bool, error) {
	if eventId == "" {
		return false, nil // 無法檢查沒有ID的事件
	}
	key := processedEventKeyPrefix + eventId
	success, err := k.redisManager.SetNX(ctx, key, "1", eventProcessedTTL)
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
	_, err := k.redisManager.Set(ctx, key, "1", eventProcessedTTL)
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
	return fmt.Sprintf("%s_%s_%s", k.consumeStream, shardId, k.config.App.Name)
}
