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
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// 處理過的事件在Redis中保留的時間
	eventProcessedTTL = 24 * time.Hour
	// 處理過的事件key前綴
	processedEventKeyPrefix = "kds:processed:"
	// 分佈式鎖Timeout時長
	consumerProcessedTTL = 1 * time.Minute
	minBackoff           = 500 * time.Millisecond
	maxBackoff           = 5 * time.Second
	// 批次處理相關配置
	batchSize            = 100  // 每批次處理的記錄數
	workerPoolSize       = 10   // 並行處理的 worker 數量
	maxBatchWaitTime     = 500 * time.Millisecond // 批次等待時間
)

type EventPayload struct {
	ID               string
	Type             string
	GlobalMerchantID string
	Data             map[string]interface{}
}

// RecordBatch 批次處理的記錄
type RecordBatch struct {
	Records        []types.Record
	ShardID        string
	ProcessedCount int
	Errors         []error
}

// ProcessResult 處理結果
type ProcessResult struct {
	EventID        string
	EventType      string
	SequenceNumber string
	Success        bool
	Error          error
}

// processBatch 批次處理記錄
func (k *KDSService) processBatch(ctx context.Context, batch RecordBatch) []ProcessResult {
	results := make([]ProcessResult, 0, len(batch.Records))
	resultChan := make(chan ProcessResult, len(batch.Records))
	
	// 使用 worker pool 並行處理
	var wg sync.WaitGroup
	workerChan := make(chan types.Record, len(batch.Records))
	
	// 啟動 workers
	numWorkers := workerPoolSize
	if len(batch.Records) < numWorkers {
		numWorkers = len(batch.Records)
	}
	
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for record := range workerChan {
				result := k.processRecord(ctx, record, batch.ShardID)
				resultChan <- result
			}
		}(i)
	}
	
	// 分發工作
	go func() {
		for _, record := range batch.Records {
			workerChan <- record
		}
		close(workerChan)
	}()
	
	// 等待所有 workers 完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()
	
	// 收集結果
	for result := range resultChan {
		results = append(results, result)
	}
	
	return results
}

// processRecord 處理單個記錄
func (k *KDSService) processRecord(ctx context.Context, record types.Record, shardID string) ProcessResult {
	sequenceNumber := *record.SequenceNumber
	
	// 解析事件
	parseEvent, parseErr := k.parseEvent(record.Data)
	if parseErr != nil {
		k.logger.WarnWithContext(ctx, "Failed to parse event",
			k.logger.String("sequence_number", sequenceNumber),
			k.logger.Error("err", parseErr))
		return ProcessResult{
			SequenceNumber: sequenceNumber,
			Success:        false,
			Error:          parseErr,
		}
	}
	
	eventID := parseEvent.ID
	eventType := parseEvent.Type
	
	// 檢查事件類型
	if eventType == "" {
		return ProcessResult{
			EventID:        eventID,
			SequenceNumber: sequenceNumber,
			Success:        false,
			Error:          errors.New("unknown event type"),
		}
	}
	
	// 從資料中取得追蹤上下文
	ctxWithTrace := tracing.ExtractTraceContext(ctx, record.Data)
	eventCtx, eventSpan := tracing.StartSpan(ctxWithTrace, "KDS.EventRecord.Processing")
	defer tracing.SpanEnd(eventSpan)
	
	// 記錄 span 屬性
	tracing.RecordSpanAttributes(eventSpan,
		attribute.String("messaging.shard_id", shardID),
		attribute.String("messaging.sequence_number", sequenceNumber),
		attribute.String("messaging.event_id", eventID),
		attribute.String("messaging.event_type", eventType),
	)
	
	// 將事件ID添加到上下文
	msgCtxWithID := context.WithValue(eventCtx, consts.EventIDKey, eventID)
	
	// 處理事件
	enqueueErr := k.eventEnqueueProcess(msgCtxWithID, eventType, record.Data)
	if enqueueErr != nil {
		tracing.RecordSpanError(eventSpan, enqueueErr)
		return ProcessResult{
			EventID:        eventID,
			EventType:      eventType,
			SequenceNumber: sequenceNumber,
			Success:        false,
			Error:          enqueueErr,
		}
	}
	
	tracing.TraceEvent(eventSpan, "Event processed successfully")
	return ProcessResult{
		EventID:        eventID,
		EventType:      eventType,
		SequenceNumber: sequenceNumber,
		Success:        true,
	}
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
		mutex, mutexErr := k.redisManager.GetMutex(mutexKey, consumerProcessedTTL)
		if mutexErr != nil {
			// Redis連線異常仍執行後面程序
			k.logger.WarnWithContext(ctx, "Failed to create mutex, skipping lock logic",
				k.logger.String("mutex_key", mutexKey),
				k.logger.Error("err", mutexErr))
			mutex = nil
		}

		// 嘗試取得鎖，若失敗則跳過
		if mutex != nil {
			if lockErr := mutex.Lock(); lockErr != nil {
				k.logger.WarnWithContext(ctx, "Shard already locked, skipping",
					k.logger.String("mutex_key", mutexKey),
					k.logger.Error("error", lockErr))
				shardWaiters.Done()
				continue
			}
		}

		// 為每個分片創建一個協程
		go func(shardId, initialIterator string, shardMutex *redsync.Mutex) {
			shardCtx, shardCancel := context.WithCancel(ctx)
			defer func() {
				// 解鎖 shard mutex
				if ok, unlockErr := shardMutex.Unlock(); !ok || err != nil {
					k.logger.WarnWithContext(shardCtx, "Failed to release shard lock",
						k.logger.String("shard_id", shardId),
						k.logger.Error("err", unlockErr))
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
				shardCancel()
				shardWaiters.Done()
			}()

			currentIterator := initialIterator

			// 自適應退避策略參數設置
			backoffDuration := 500 * time.Millisecond
			for {
				select {
				case <-shardCtx.Done():
					return
				default:
					// 獲取記錄
					recordsOutput, getRecordsErr := k.client.GetRecords(
						shardCtx,
						&kinesis.GetRecordsInput{
							ShardIterator: aws.String(currentIterator),
							Limit:         aws.Int32(1000),
						},
					)
					if getRecordsErr != nil {
						k.logger.ErrorWithContext(
							shardCtx,
							"Failed to get records from shard",
							k.logger.String("shard_id", shardId),
							k.logger.Error("err", getRecordsErr),
						)

						// 遇到錯誤時增加退避時間
						backoffDuration = time.Duration(float64(backoffDuration) * 1.5)
						if backoffDuration > maxBackoff {
							backoffDuration = maxBackoff
						}
						time.Sleep(backoffDuration)

						// 重新獲取迭代器
						iterators, getIteratorsErr := k.getShardIterators(shardCtx)
						if getIteratorsErr != nil {
							k.logger.ErrorWithContext(
								shardCtx,
								"Failed to refresh shard iterator",
								k.logger.String("shard_id", shardId),
								k.logger.Error("err", getIteratorsErr),
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
						
						// 批次處理記錄
						batch := RecordBatch{
							Records: recordsOutput.Records,
							ShardID: shardId,
						}
						
						// 批次去重檢查
						eventIDs := make([]string, 0, len(batch.Records))
						for _, record := range batch.Records {
							if parseEvent, err := k.parseEvent(record.Data); err == nil {
								eventIDs = append(eventIDs, parseEvent.ID)
							}
						}
						
						// 批次檢查已處理的事件
						processedMap := k.batchCheckEventsProcessed(shardCtx, eventIDs)
						
						// 過濾未處理的記錄
						unprocessedRecords := make([]types.Record, 0, len(batch.Records))
						for i, record := range batch.Records {
							if i < len(eventIDs) && !processedMap[eventIDs[i]] {
								unprocessedRecords = append(unprocessedRecords, record)
							}
						}
						
						if len(unprocessedRecords) > 0 {
							batch.Records = unprocessedRecords
							
							// 批次處理
							results := k.processBatch(shardCtx, batch)
							
							// 批次標記已處理和更新檢查點
							successfulEvents := make([]string, 0, len(results))
							var lastSuccessSequence string
							
							for _, result := range results {
								if result.Success {
									successfulEvents = append(successfulEvents, result.EventID)
									lastSuccessSequence = result.SequenceNumber
									
									k.logger.InfoWithContext(shardCtx,
										"Processed event successfully",
										k.logger.String("event_type", result.EventType),
										k.logger.String("event_id", result.EventID),
										k.logger.String("sequence_number", result.SequenceNumber))
								} else if result.Error != nil {
									if errors.Is(result.Error, errmsg.ErrUnknownEventType) {
										// 未知事件類型仍需要更新檢查點
										lastSuccessSequence = result.SequenceNumber
									}
									k.logger.ErrorWithContext(shardCtx,
										"Failed to process event",
										k.logger.String("event_id", result.EventID),
										k.logger.Error("err", result.Error))
								}
							}
							
							// 批次標記事件為已處理
							if len(successfulEvents) > 0 {
								k.batchMarkEventsProcessed(shardCtx, successfulEvents)
							}
							
							// 更新檢查點到最後成功的序列號
							if lastSuccessSequence != "" {
								if checkpointErr := k.updateCheckpoint(shardCtx, shardId, lastSuccessSequence); checkpointErr != nil {
									k.logger.WarnWithContext(shardCtx,
										"Failed to update checkpoint",
										k.logger.String("shard_id", shardId),
										k.logger.String("sequence_number", lastSuccessSequence),
										k.logger.Error("err", checkpointErr))
								}
							}
						}
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
	for shardErr := range shardErrs {
		if shardErr != nil {
			k.logger.ErrorLog("Error in shard processing", k.logger.Error("err", shardErr))
			return shardErr
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
	shardsOutput, shardsErr := k.client.ListShards(ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(k.consumeStream),
	})
	if shardsErr != nil {
		return nil, fmt.Errorf("failed to list shards: %w", shardsErr)
	}

	shardIterators := make(map[string]string)

	// 獲取每個分片的迭代器
	for _, shard := range shardsOutput.Shards {
		if shard.ShardId != nil && *shard.ShardId != "" {
			// 檢查DynamoDB中是否有這個分片的checkpoint
			checkpoint, checkpointErr := k.getCheckpoint(ctx, *shard.ShardId)
			k.logger.InfoLog("Current checkpoint",
				k.logger.String("shard_id", *shard.ShardId),
				k.logger.String("checkpoint", checkpoint))

			iteratorType := types.ShardIteratorTypeLatest
			var sequenceNumber *string = nil

			if checkpointErr == nil && checkpoint != "" {
				// 如果找到checkpoint，從該位置開始讀取
				iteratorType = types.ShardIteratorTypeAfterSequenceNumber
				sequenceNumber = &checkpoint
				k.logger.InfoLog("Found checkpoint, starting from after sequence number",
					k.logger.String("shard_id", *shard.ShardId),
					k.logger.String("sequence_number", checkpoint))
			} else {
				k.logger.ErrorLog("No checkpoint found, starting from latest position",
					k.logger.String("shard_id", *shard.ShardId),
					k.logger.Error("err", checkpointErr))
			}

			iterOutput, iteratorErr := k.client.GetShardIterator(
				ctx,
				&kinesis.GetShardIteratorInput{
					StreamName:             aws.String(k.consumeStream),
					ShardId:                shard.ShardId,
					ShardIteratorType:      iteratorType,
					StartingSequenceNumber: sequenceNumber,
				},
			)
			if iteratorErr != nil {
				k.logger.ErrorLog("Failed to get shard iterator",
					k.logger.String("shard_id", *shard.ShardId),
					k.logger.Error("err", iteratorErr))
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

// batchCheckEventsProcessed 批次檢查事件是否已處理
func (k *KDSService) batchCheckEventsProcessed(ctx context.Context, eventIDs []string) map[string]bool {
	processedMap := make(map[string]bool)
	if len(eventIDs) == 0 {
		return processedMap
	}
	
	// 使用 Pipeline 批次執行 EXISTS 命令
	keys := make([]string, 0, len(eventIDs))
	for _, eventID := range eventIDs {
		if eventID != "" {
			keys = append(keys, processedEventKeyPrefix+eventID)
		}
	}
	
	if len(keys) == 0 {
		return processedMap
	}
	
	// 批次檢查 keys 是否存在
	results, err := k.redisManager.MGet(ctx, keys...)
	if err != nil {
		k.logger.WarnWithContext(ctx, "Failed to batch check events processed",
			k.logger.Error("err", err))
		// 錯誤時返回空map，視為都未處理
		return processedMap
	}
	
	// 建立結果映射
	for i, result := range results {
		if i < len(eventIDs) {
			processedMap[eventIDs[i]] = result != nil
		}
	}
	
	return processedMap
}

// batchMarkEventsProcessed 批次標記事件為已處理
func (k *KDSService) batchMarkEventsProcessed(ctx context.Context, eventIDs []string) {
	if len(eventIDs) == 0 {
		return
	}
	
	// 使用 Pipeline 批次設置
	pipeline := k.redisManager.Pipeline()
	
	for _, eventID := range eventIDs {
		if eventID != "" {
			key := processedEventKeyPrefix + eventID
			pipeline.Set(ctx, key, "1", eventProcessedTTL)
		}
	}
	
	// 執行 pipeline
	if _, err := pipeline.Exec(ctx); err != nil {
		k.logger.WarnWithContext(ctx, "Failed to batch mark events as processed",
			k.logger.Error("err", err))
	}
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

func (k *KDSService) parseEvent(data []byte) (EventPayload, error) {
	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return EventPayload{}, err
	}

	eventPayload := EventPayload{
		ID:   jsonData["id"].(string),
		Type: jsonData["type"].(string),
	}

	if eventData, ok := jsonData["data"].(map[string]interface{}); ok {
		eventPayload.GlobalMerchantID, _ = eventData["global_merchant_id"].(string)
		eventPayload.Data = eventData
	}
	return eventPayload, nil
}

func (k *KDSService) composeDynamoDBKey(shardId string) string {
	return fmt.Sprintf("%s_%s_%s", k.consumeStream, shardId, k.config.App.Name)
}
