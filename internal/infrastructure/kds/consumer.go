package kds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const (
	// eventProcessedTTL 處理過的事件在Redis中保留的時間
	// 用於防止重複處理相同事件，24小時後自動過期
	eventProcessedTTL = 24 * time.Hour

	// processedEventKeyPrefix 處理過的事件在Redis中的key前綴
	// 格式：kds:processed:{eventId}
	processedEventKeyPrefix = "kds:processed:"

	// consumerProcessedTTL 分片處理的分佈式鎖最大持有時間
	// 防止多個消費者同時處理同一個分片
	consumerProcessedTTL = 1 * time.Minute
)

// EventPayload 代表從Kinesis中解析出的事件負載
// 包含了事件的核心識別資訊和數據內容
type EventPayload struct {
	// ID 事件的唯一識別符，用於去重和追蹤
	ID string

	// Type 事件類型，用於決定處理策略
	Type string

	// GlobalMerchantID 全局商戶ID，用於商戶級別的數據隔離
	GlobalMerchantID string

	// Data 事件的具體數據內容
	Data map[string]interface{}
}

// RecordBatch 代表一批待處理的Kinesis記錄
// 用於批次處理以提高效率
type RecordBatch struct {
	// Records Kinesis原始記錄列表
	Records []types.Record

	// ShardID 所屬分片ID，用於日誌記錄和錯誤追蹤
	ShardID string

	// ProcessedCount 已處理的記錄數量（留作未來統計使用）
	ProcessedCount int

	// Errors 處理過程中的錯誤列表（留作未來錯誤處理使用）
	Errors []error
}

// ProcessResult 代表單個事件處理的結果
// 用於統計成功率和錯誤追蹤
type ProcessResult struct {
	// EventID 事件唯一識別符
	EventID string

	// EventType 事件類型
	EventType string

	// SequenceNumber Kinesis序列號，用於checkpoint更新
	SequenceNumber string

	// Success 是否處理成功
	Success bool

	// Error 處理失敗時的錯誤資訊
	Error error
}

// ConsumeAllEvents 消費所有事件類型
//
// 這是 KDS Consumer 的主入口點，負責：
// 1. 獲取所有分片的迭代器
// 2. 為每個分片創建獨立的消費者goroutine
// 3. 使用分佈式鎖防止多個消費者同時處理同一分片
// 4. 統一管理所有分片的生命週期和錯誤處理
//
// 參數：
//   - ctx: 上下文，用於取消和超時控制
//
// 返回值：
//   - error: 若初始化失敗或有分片處理出現不可恢復錯誤
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

		// 獲取分片鎖
		shardMutex := k.acquireShardLock(ctx, shardId)
		if shardMutex == nil {
			shardWaiters.Done()
			continue
		}

		// 為每個分片創建一個協程
		go k.consumeShardEvents(ctx, shardId, iterator, shardMutex, &shardWaiters, shardErrs)
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

// acquireShardLock 獲取分片鎖
//
// 使用Redis分佈式鎖確保只有一個消費者實例處理特定分片。
// 這在水平擴展情境下特別重要，防止重複處理和競爭情況。
//
// 參數：
//   - ctx: 上下文，用於日誌記錄
//   - shardId: 分片ID
//
// 返回值：
//   - *redsync.Mutex: 成功獲取的鎖，失敗時返回 nil
func (k *KDSService) acquireShardLock(ctx context.Context, shardId string) *redsync.Mutex {
	mutexKey := fmt.Sprintf(consts.ShardMutexRedisKey, k.consumeStream, shardId)
	mutex, mutexErr := k.redisManager.GetMutex(mutexKey, consumerProcessedTTL)
	if mutexErr != nil {
		// Redis連線異常仍執行後面程序
		k.logger.WarnWithContext(ctx, "Failed to create mutex, skipping lock logic",
			k.logger.String("mutex_key", mutexKey),
			k.logger.Error("err", mutexErr))
		return nil
	}

	// 嘗試取得鎖，若失敗則跳過
	if lockErr := mutex.Lock(); lockErr != nil {
		k.logger.WarnWithContext(ctx, "Shard already locked, skipping",
			k.logger.String("mutex_key", mutexKey),
			k.logger.Error("error", lockErr))
		return nil
	}

	return mutex
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
			k.logger.InfoWithContext(ctx, "Current checkpoint",
				k.logger.String("shard_id", *shard.ShardId),
				k.logger.String("checkpoint", checkpoint))

			iteratorType := types.ShardIteratorTypeLatest
			var sequenceNumber *string = nil

			if checkpointErr == nil && checkpoint != "" {
				// 如果找到checkpoint，從該位置開始讀取
				iteratorType = types.ShardIteratorTypeAfterSequenceNumber
				sequenceNumber = &checkpoint
				k.logger.InfoWithContext(
					ctx,
					"Found checkpoint, starting from after sequence number",
					k.logger.String("shard_id", *shard.ShardId),
					k.logger.String("sequence_number", checkpoint),
				)
			} else {
				k.logger.ErrorWithContext(ctx, "No checkpoint found, starting from latest position",
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
				k.logger.ErrorWithContext(ctx, "Failed to get shard iterator",
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
		k.logger.ErrorWithContext(ctx, "Failed to get checkpoint from DynamoDB",
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
	k.logger.InfoWithContext(ctx,
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
func (k *KDSService) batchCheckEventsProcessed(
	ctx context.Context,
	eventIDs []string,
) map[string]bool {
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

// processBatch 批次處理記錄
func (k *KDSService) processBatch(ctx context.Context, batch RecordBatch) []ProcessResult {
	results := make([]ProcessResult, 0, len(batch.Records))
	resultChan := make(chan ProcessResult, len(batch.Records))

	// 使用 worker pool 並行處理
	var wg sync.WaitGroup
	workerChan := make(chan types.Record, k.config.Consumer.WorkerBufferSize)

	// 啟動 workers
	numWorkers := k.config.Consumer.WorkerPoolSize
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
func (k *KDSService) processRecord(
	ctx context.Context,
	record types.Record,
	shardID string,
) ProcessResult {
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

// consumeShardEvents 消費單個分片事件
// 這是單個分片的主處理循環，負責：
// 1. 持續從Kinesis獲取記錄
// 2. 管理自適應退避策略
// 3. 處理獲取錯誤和連線中斷
// 4. 在分片關閉或上下文取消時清理資源
//
// 參數：
//   - ctx: 上下文，用於取消控制
//   - shardId: 分片ID
//   - initialIterator: 初始迭代器
//   - shardMutex: 分片鎖，用於防止多個消費者同時處理
//   - shardWaiters: 用於等待所有分片完成
//   - shardErrs: 用於報告不可恢復的錯誤
func (k *KDSService) consumeShardEvents(
	ctx context.Context,
	shardId, initialIterator string,
	shardMutex *redsync.Mutex,
	shardWaiters *sync.WaitGroup,
	shardErrs chan<- error,
) {
	shardCtx, shardCancel := context.WithCancel(ctx)
	defer func() {
		// 解鎖 shard mutex
		if shardMutex != nil {
			if ok, unlockErr := shardMutex.Unlock(); !ok || unlockErr != nil {
				k.logger.WarnWithContext(shardCtx, "Failed to release shard lock",
					k.logger.String("shard_id", shardId),
					k.logger.Error("err", unlockErr))
			}
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
	backoffManager := k.createBackoffManager()

	for {
		select {
		case <-shardCtx.Done():
			return
		default:
			// 獲取和處理記錄
			newIterator, shouldContinue := k.processShardRecords(
				shardCtx, shardId, currentIterator, backoffManager, shardErrs)
			if !shouldContinue {
				return
			}
			currentIterator = newIterator
			time.Sleep(backoffManager.GetCurrentBackoff())
		}
	}
}

// processShardRecords 處理分片記錄
func (k *KDSService) processShardRecords(
	ctx context.Context,
	shardId, currentIterator string,
	backoffManager *BackoffManager,
	shardErrs chan<- error,
) (string, bool) {
	// 獲取記錄
	recordsOutput, getRecordsErr := k.client.GetRecords(
		ctx,
		&kinesis.GetRecordsInput{
			ShardIterator: aws.String(currentIterator),
			Limit:         aws.Int32(int32(k.config.Consumer.KDSRecordLimit)),
		},
	)
	if getRecordsErr != nil {
		return k.handleGetRecordsError(ctx, shardId, getRecordsErr, backoffManager, shardErrs)
	}

	// 調整退避時間
	recordsCount := len(recordsOutput.Records)
	backoffManager.AdjustBackoffByRecordCount(recordsCount)

	// 處理記錄批次
	if recordsCount > 0 {
		k.processRecordBatches(ctx, shardId, recordsOutput.Records)
	}

	// 獲取下一個迭代器
	if recordsOutput.NextShardIterator != nil {
		return *recordsOutput.NextShardIterator, true
	}

	// 分片已關閉
	k.logger.InfoWithContext(ctx, "Shard has been closed", k.logger.String("shard_id", shardId))
	return "", false
}

// handleGetRecordsError 處理獲取記錄錯誤
func (k *KDSService) handleGetRecordsError(
	ctx context.Context,
	shardId string,
	getRecordsErr error,
	backoffManager *BackoffManager,
	shardErrs chan<- error,
) (string, bool) {
	k.logger.ErrorWithContext(
		ctx,
		"Failed to get records from shard",
		k.logger.String("shard_id", shardId),
		k.logger.Error("err", getRecordsErr),
	)

	// 增加退避時間
	backoffManager.IncreaseBackoff()
	time.Sleep(backoffManager.GetCurrentBackoff())

	// 重新獲取迭代器
	iterators, getIteratorsErr := k.getShardIterators(ctx)
	if getIteratorsErr != nil {
		k.logger.ErrorWithContext(
			ctx,
			"Failed to refresh shard iterator",
			k.logger.String("shard_id", shardId),
			k.logger.Error("err", getIteratorsErr),
		)

		// 只有在上下文未被取消時才報告錯誤
		if ctx.Err() == nil {
			select {
			case shardErrs <- fmt.Errorf("failed to refresh shard iterator for shard %s: %w", shardId, getIteratorsErr):
			default:
			}
		}
		return "", false
	}

	if newIterator, ok := iterators[shardId]; ok {
		return newIterator, true
	}

	k.logger.ErrorWithContext(
		ctx,
		"Shard no longer available",
		k.logger.String("shard_id", shardId),
	)
	return "", false
}

// processRecordBatches 處理記錄批次
func (k *KDSService) processRecordBatches(
	ctx context.Context,
	shardId string,
	records []types.Record,
) {
	// 批次去重檢查
	eventIDs := k.extractEventIDs(records)
	duplicationCheckMap := k.batchCheckEventsProcessed(ctx, eventIDs)

	// 過濾未處理的記錄
	unprocessedRecords := k.filterUnprocessedRecords(records, eventIDs, duplicationCheckMap)

	if len(unprocessedRecords) == 0 {
		return
	}

	// 收集所有批次的處理結果
	allSuccessfulEvents := make([]string, 0)
	var lastSuccessSequence string

	// 按照配置的 BatchSize 分割記錄
	batchSize := k.config.Consumer.BatchSize
	for i := 0; i < len(unprocessedRecords); i += batchSize {
		end := i + batchSize
		if end > len(unprocessedRecords) {
			end = len(unprocessedRecords)
		}

		batch := RecordBatch{
			Records: unprocessedRecords[i:end],
			ShardID: shardId,
		}

		// 批次處理
		results := k.processBatch(ctx, batch)

		// 處理每個批次的結果
		successfulEvents, lastSequence := k.handleBatchResults(ctx, results)
		allSuccessfulEvents = append(allSuccessfulEvents, successfulEvents...)
		if lastSequence != "" {
			lastSuccessSequence = lastSequence
		}
	}

	// 批次標記所有成功的事件為已處理
	if len(allSuccessfulEvents) > 0 {
		k.batchMarkEventsProcessed(ctx, allSuccessfulEvents)
	}

	// 更新檢查點到最後成功的序列號
	if lastSuccessSequence != "" {
		if checkpointErr := k.updateCheckpoint(ctx, shardId, lastSuccessSequence); checkpointErr != nil {
			k.logger.WarnWithContext(ctx,
				"Failed to update checkpoint",
				k.logger.String("shard_id", shardId),
				k.logger.String("sequence_number", lastSuccessSequence),
				k.logger.Error("err", checkpointErr))
		}
	}
}

// extractEventIDs 提取事件ID列表
func (k *KDSService) extractEventIDs(records []types.Record) []string {
	eventIDs := make([]string, 0, len(records))
	for _, record := range records {
		if parseEvent, err := k.parseEvent(record.Data); err == nil {
			eventIDs = append(eventIDs, parseEvent.ID)
		}
	}
	return eventIDs
}

// filterUnprocessedRecords 過濾未處理的記錄
func (k *KDSService) filterUnprocessedRecords(
	records []types.Record,
	eventIDs []string,
	processedMap map[string]bool,
) []types.Record {
	unprocessedRecords := make([]types.Record, 0, len(records))
	for i, record := range records {
		if i < len(eventIDs) && !processedMap[eventIDs[i]] {
			unprocessedRecords = append(unprocessedRecords, record)
		}
	}
	return unprocessedRecords
}

// handleBatchResults 處理批次結果
func (k *KDSService) handleBatchResults(
	ctx context.Context,
	results []ProcessResult,
) ([]string, string) {
	successfulEvents := make([]string, 0, len(results))
	var lastSuccessSequence string

	for _, result := range results {
		if result.Success {
			successfulEvents = append(successfulEvents, result.EventID)
			lastSuccessSequence = result.SequenceNumber

			k.logger.InfoWithContext(ctx,
				"Processed event successfully",
				k.logger.String("event_type", result.EventType),
				k.logger.String("event_id", result.EventID),
				k.logger.String("sequence_number", result.SequenceNumber))
		} else if result.Error != nil {
			if errors.Is(result.Error, errmsg.ErrUnknownEventType) {
				// 未知事件類型仍需要更新檢查點
				lastSuccessSequence = result.SequenceNumber
			}
			k.logger.ErrorWithContext(ctx,
				"Failed to process event",
				k.logger.String("event_id", result.EventID),
				k.logger.Error("err", result.Error))
		}
	}

	return successfulEvents, lastSuccessSequence
}
