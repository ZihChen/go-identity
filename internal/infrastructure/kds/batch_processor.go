package kds

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"go.opentelemetry.io/otel/attribute"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

// RecordBatch 批次處理記錄結構
type RecordBatch struct {
	Records        []types.Record     `json:"records"`         // KDS 記錄列表
	ShardID        string             `json:"shard_id"`        // 分片 ID
	ProcessedCount int32              `json:"processed_count"` // 已處理記錄數
	Errors         []ProcessingError  `json:"errors"`          // 處理錯誤列表
	ProcessingTime time.Time          `json:"processing_time"` // 批次處理開始時間
	BatchSize      int                `json:"batch_size"`      // 當前批次大小
	Timeout        time.Duration      `json:"timeout"`         // 批次超時時間
	ctx            context.Context    // 批次處理上下文
	cancel         context.CancelFunc // 取消函數
}

// ProcessingError 處理錯誤結構
type ProcessingError struct {
	RecordIndex    int       `json:"record_index"`    // 記錄索引
	SequenceNumber string    `json:"sequence_number"` // 序列號
	EventID        string    `json:"event_id"`        // 事件 ID
	EventType      string    `json:"event_type"`      // 事件類型
	Error          error     `json:"error"`           // 錯誤信息
	Timestamp      time.Time `json:"timestamp"`       // 錯誤發生時間
	Retryable      bool      `json:"retryable"`       // 是否可重試
}

// ProcessResult 處理結果結構
type ProcessResult struct {
	EventID        string        `json:"event_id"`        // 事件 ID
	EventType      string        `json:"event_type"`      // 事件類型
	SequenceNumber string        `json:"sequence_number"` // 序列號
	Success        bool          `json:"success"`         // 處理是否成功
	Error          error         `json:"error"`           // 錯誤信息（如有）
	ProcessingTime time.Duration `json:"processing_time"` // 處理耗時
	ShardID        string        `json:"shard_id"`        // 分片 ID
	RetryCount     int           `json:"retry_count"`     // 重試次數
	Timestamp      time.Time     `json:"timestamp"`       // 處理時間戳
}

// BatchProcessor 批次處理器介面
type BatchProcessor interface {
	// ProcessBatch 處理一個批次的記錄
	ProcessBatch(ctx context.Context, batch *RecordBatch) []ProcessResult
	
	// GetMetrics 獲取批次處理指標
	GetMetrics() BatchMetrics
	
	// Start 啟動批次處理器
	Start(ctx context.Context) error
	
	// Stop 停止批次處理器
	Stop() error
	
	// AddRecord 添加記錄到當前批次
	AddRecord(record types.Record, shardID string) error
	
	// FlushBatch 強制刷新當前批次
	FlushBatch(shardID string) error
	
	// IsHealthy 檢查批次處理器健康狀態
	IsHealthy() bool
}

// BatchMetrics 批次處理指標
type BatchMetrics struct {
	TotalBatches         int64         `json:"total_batches"`          // 總處理批次數
	TotalRecords         int64         `json:"total_records"`          // 總處理記錄數
	SuccessfulBatches    int64         `json:"successful_batches"`     // 成功批次數
	FailedBatches        int64         `json:"failed_batches"`         // 失敗批次數
	AverageBatchSize     float64       `json:"average_batch_size"`     // 平均批次大小
	AverageProcessingTime time.Duration `json:"average_processing_time"` // 平均處理時間
	ErrorRate            float64       `json:"error_rate"`             // 錯誤率
	LastProcessedAt      time.Time     `json:"last_processed_at"`      // 最後處理時間
	CurrentBatchCount    int           `json:"current_batch_count"`    // 當前批次記錄數
}

// DefaultBatchProcessor 預設批次處理器實現
type DefaultBatchProcessor struct {
	// 配置參數
	batchSize        int           // 批次大小
	maxBatchWaitTime time.Duration // 批次最大等待時間
	
	// 組件依賴
	kdsService       *KDSService       // KDS 服務實例
	workerPool       WorkerPool        // Worker Pool
	metrics          *MetricsCollector // 指標收集器
	backoffStrategy  BackoffStrategy   // 退避策略
	logger           infrastructure.Logger     // 日誌記錄器
	
	// 內部狀態
	currentBatches   map[string]*RecordBatch // 當前分片批次 map
	batchLocks       map[string]*sync.RWMutex // 分片批次鎖 map
	isRunning        int32                   // 運行狀態標記
	stopChan         chan struct{}           // 停止通道
	flushTicker      *time.Ticker            // 刷新定時器
	
	// 指標統計
	totalBatches      int64 // 總批次數
	totalRecords      int64 // 總記錄數
	successfulBatches int64 // 成功批次數
	failedBatches     int64 // 失敗批次數
	totalProcessingTime time.Duration // 總處理時間
	
	// 同步原語
	mu sync.RWMutex // 讀寫鎖
}

// NewDefaultBatchProcessor 創建預設批次處理器
func NewDefaultBatchProcessor(
	batchSize int,
	maxBatchWaitTime time.Duration,
	kdsService *KDSService,
	workerPool WorkerPool,
	metrics *MetricsCollector,
	backoffStrategy BackoffStrategy,
	logger infrastructure.Logger,
) *DefaultBatchProcessor {
	return &DefaultBatchProcessor{
		batchSize:        batchSize,
		maxBatchWaitTime: maxBatchWaitTime,
		kdsService:       kdsService,
		workerPool:       workerPool,
		metrics:          metrics,
		backoffStrategy:  backoffStrategy,
		logger:           logger,
		currentBatches:   make(map[string]*RecordBatch),
		batchLocks:       make(map[string]*sync.RWMutex),
		stopChan:         make(chan struct{}),
	}
}

// Start 啟動批次處理器
func (bp *DefaultBatchProcessor) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&bp.isRunning, 0, 1) {
		return fmt.Errorf("batch processor is already running")
	}
	
	bp.logger.InfoWithContext(
		ctx,
		"Starting batch processor",
		bp.logger.Int("batch_size", bp.batchSize),
		bp.logger.String("max_batch_wait_time", bp.maxBatchWaitTime.String()),
	)
	
	// 啟動定時刷新 goroutine
	bp.flushTicker = time.NewTicker(bp.maxBatchWaitTime / 2) // 每半個超時時間檢查一次
	go bp.periodicFlush(ctx)
	
	return nil
}

// Stop 停止批次處理器
func (bp *DefaultBatchProcessor) Stop() error {
	if !atomic.CompareAndSwapInt32(&bp.isRunning, 1, 0) {
		return fmt.Errorf("batch processor is not running")
	}
	
	bp.logger.InfoLog("Stopping batch processor")
	
	// 停止定時器
	if bp.flushTicker != nil {
		bp.flushTicker.Stop()
	}
	
	// 發送停止信號
	close(bp.stopChan)
	
	// 刷新所有待處理的批次
	bp.flushAllBatches()
	
	bp.logger.InfoLog("Batch processor stopped")
	return nil
}

// AddRecord 添加記錄到批次
func (bp *DefaultBatchProcessor) AddRecord(record types.Record, shardID string) error {
	if atomic.LoadInt32(&bp.isRunning) == 0 {
		return fmt.Errorf("batch processor is not running")
	}
	
	// 獲取或創建分片的批次鎖
	bp.mu.Lock()
	if _, exists := bp.batchLocks[shardID]; !exists {
		bp.batchLocks[shardID] = &sync.RWMutex{}
	}
	batchLock := bp.batchLocks[shardID]
	bp.mu.Unlock()
	
	// 鎖定分片批次
	batchLock.Lock()
	defer batchLock.Unlock()
	
	// 獲取或創建當前批次
	batch := bp.getOrCreateBatch(shardID)
	
	// 添加記錄到批次
	batch.Records = append(batch.Records, record)
	
	// 檢查是否需要立即處理批次
	if len(batch.Records) >= bp.batchSize {
		return bp.processBatchInternal(batch)
	}
	
	return nil
}

// FlushBatch 強制刷新指定分片的批次
func (bp *DefaultBatchProcessor) FlushBatch(shardID string) error {
	bp.mu.RLock()
	batchLock, exists := bp.batchLocks[shardID]
	bp.mu.RUnlock()
	
	if !exists {
		return nil // 沒有該分片的批次
	}
	
	batchLock.Lock()
	defer batchLock.Unlock()
	
	batch, exists := bp.currentBatches[shardID]
	if !exists || len(batch.Records) == 0 {
		return nil // 批次為空或不存在
	}
	
	return bp.processBatchInternal(batch)
}

// ProcessBatch 處理批次記錄
func (bp *DefaultBatchProcessor) ProcessBatch(ctx context.Context, batch *RecordBatch) []ProcessResult {
	startTime := time.Now()
	
	// 創建結果容器
	results := make([]ProcessResult, 0, len(batch.Records))
	
	// 批次處理上下文和 tracing
	batchCtx, batchSpan := tracing.StartSpan(ctx, "BatchProcessor.ProcessBatch")
	defer tracing.SpanEnd(batchSpan)
	
	// 記錄批次處理屬性
	tracing.RecordSpanAttributes(batchSpan,
		attribute.String("batch.shard_id", batch.ShardID),
		attribute.Int("batch.record_count", len(batch.Records)),
		attribute.Int("batch.size_config", bp.batchSize),
	)
	
	bp.logger.InfoWithContext(
		batchCtx,
		"Processing record batch",
		bp.logger.String("shard_id", batch.ShardID),
		bp.logger.Int("record_count", len(batch.Records)),
	)
	
	// 處理批次中的每條記錄
	var processedCount int32
	var errorCount int32
	
	for i, record := range batch.Records {
		// 處理單條記錄
		result := bp.processSingleRecord(batchCtx, record, batch.ShardID, i)
		results = append(results, result)
		
		// 更新計數器
		if result.Success {
			atomic.AddInt32(&processedCount, 1)
		} else {
			atomic.AddInt32(&errorCount, 1)
			
			// 記錄錯誤
			batch.Errors = append(batch.Errors, ProcessingError{
				RecordIndex:    i,
				SequenceNumber: result.SequenceNumber,
				EventID:        result.EventID,
				EventType:      result.EventType,
				Error:          result.Error,
				Timestamp:      time.Now(),
				Retryable:      bp.isRetryableError(result.Error),
			})
		}
	}
	
	// 更新批次統計
	batch.ProcessedCount = processedCount
	processingTime := time.Since(startTime)
	
	// 更新全局指標
	atomic.AddInt64(&bp.totalBatches, 1)
	atomic.AddInt64(&bp.totalRecords, int64(len(batch.Records)))
	
	if errorCount == 0 {
		atomic.AddInt64(&bp.successfulBatches, 1)
	} else {
		atomic.AddInt64(&bp.failedBatches, 1)
	}
	
	// 記錄處理時間（使用原子操作更新總時間）
	atomic.AddInt64((*int64)(&bp.totalProcessingTime), int64(processingTime))
	
	// 更新指標收集器
	if bp.metrics != nil {
		bp.metrics.BatchProcessed(len(batch.Records), processingTime)
		for i := 0; i < int(errorCount); i++ {
			bp.metrics.ErrorOccurred()
		}
	}
	
	bp.logger.InfoWithContext(
		batchCtx,
		"Batch processing completed",
		bp.logger.String("shard_id", batch.ShardID),
		bp.logger.Int("total_records", len(batch.Records)),
		bp.logger.Int("processed_count", int(processedCount)),
		bp.logger.Int("error_count", int(errorCount)),
		bp.logger.String("processing_time", processingTime.String()),
	)
	
	// 記錄批次處理結果到 tracing
	tracing.RecordSpanAttributes(batchSpan,
		attribute.Int("batch.processed_count", int(processedCount)),
		attribute.Int("batch.error_count", int(errorCount)),
		attribute.String("batch.processing_time", processingTime.String()),
	)
	
	return results
}

// processSingleRecord 處理單條記錄
func (bp *DefaultBatchProcessor) processSingleRecord(
	ctx context.Context,
	record types.Record,
	shardID string,
	recordIndex int,
) ProcessResult {
	startTime := time.Now()
	sequenceNumber := ""
	if record.SequenceNumber != nil {
		sequenceNumber = *record.SequenceNumber
	}
	
	// 創建處理結果
	result := ProcessResult{
		SequenceNumber: sequenceNumber,
		ShardID:        shardID,
		Timestamp:      startTime,
		RetryCount:     0,
	}
	
	// 創建記錄處理的 tracing context
	recordCtx, recordSpan := tracing.StartSpan(ctx, "BatchProcessor.ProcessSingleRecord")
	defer func() {
		result.ProcessingTime = time.Since(startTime)
		tracing.RecordSpanAttributes(recordSpan,
			attribute.String("record.sequence_number", result.SequenceNumber),
			attribute.String("record.event_id", result.EventID),
			attribute.String("record.event_type", result.EventType),
			attribute.Bool("record.success", result.Success),
			attribute.String("record.processing_time", result.ProcessingTime.String()),
		)
		
		if result.Error != nil {
			tracing.RecordSpanError(recordSpan, result.Error)
		}
		
		tracing.SpanEnd(recordSpan)
	}()
	
	// 從記錄中提取 trace context
	ctxWithTrace := tracing.ExtractTraceContext(recordCtx, record.Data)
	
	// 解析事件
	parseEvent, parseErr := bp.kdsService.parseEvent(record.Data)
	if parseErr != nil {
		result.Error = fmt.Errorf("failed to parse event: %w", parseErr)
		bp.logger.WarnWithContext(
			ctxWithTrace,
			"Failed to parse event in batch processing",
			bp.logger.String("sequence_number", sequenceNumber),
			bp.logger.Int("record_index", recordIndex),
			bp.logger.Error("error", parseErr),
		)
		return result
	}
	
	result.EventID = parseEvent.ID
	result.EventType = parseEvent.Type
	
	// 檢查事件類型
	if parseEvent.Type == "" {
		result.Error = fmt.Errorf("event type is empty")
		bp.logger.WarnWithContext(
			ctxWithTrace,
			"Skipping event with empty type in batch processing",
			bp.logger.String("sequence_number", sequenceNumber),
			bp.logger.String("event_id", result.EventID),
		)
		return result
	}
	
	// 檢查事件是否已處理（去重）
	processed, processedErr := bp.kdsService.isEventProcessed(ctxWithTrace, result.EventID)
	if processedErr != nil {
		bp.logger.WarnWithContext(
			ctxWithTrace,
			"Failed to check if event is processed in batch, will process anyway",
			bp.logger.String("event_id", result.EventID),
			bp.logger.Error("error", processedErr),
		)
	}
	
	if processed {
		bp.logger.InfoWithContext(
			ctxWithTrace,
			"Skipping already processed event in batch",
			bp.logger.String("event_id", result.EventID),
			bp.logger.String("event_type", result.EventType),
		)
		result.Success = true // 已處理的事件視為成功
		return result
	}
	
	// 將事件 ID 添加到上下文
	msgCtxWithID := context.WithValue(ctxWithTrace, consts.EventIDKey, result.EventID)
	
	// 處理事件入隊
	enqueueErr := bp.kdsService.eventEnqueueProcess(msgCtxWithID, result.EventType, record.Data)
	if enqueueErr != nil {
		result.Error = fmt.Errorf("failed to enqueue event: %w", enqueueErr)
		bp.logger.ErrorWithContext(
			ctxWithTrace,
			"Failed to enqueue event in batch processing",
			bp.logger.String("event_id", result.EventID),
			bp.logger.String("event_type", result.EventType),
			bp.logger.Error("error", enqueueErr),
		)
		return result
	}
	
	// 標記事件為已處理
	if markErr := bp.kdsService.markEventProcessed(ctxWithTrace, result.EventID); markErr != nil {
		bp.logger.WarnWithContext(
			ctxWithTrace,
			"Failed to mark event as processed in batch",
			bp.logger.String("event_id", result.EventID),
			bp.logger.Error("error", markErr),
		)
	}
	
	result.Success = true
	
	return result
}

// IsHealthy 檢查批次處理器健康狀態
func (bp *DefaultBatchProcessor) IsHealthy() bool {
	return atomic.LoadInt32(&bp.isRunning) == 1
}

// GetMetrics 獲取批次處理指標
func (bp *DefaultBatchProcessor) GetMetrics() BatchMetrics {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	
	totalBatches := atomic.LoadInt64(&bp.totalBatches)
	totalRecords := atomic.LoadInt64(&bp.totalRecords)
	successfulBatches := atomic.LoadInt64(&bp.successfulBatches)
	failedBatches := atomic.LoadInt64(&bp.failedBatches)
	totalProcessingTime := time.Duration(atomic.LoadInt64((*int64)(&bp.totalProcessingTime)))
	
	var avgBatchSize float64
	var avgProcessingTime time.Duration
	var errorRate float64
	var currentBatchCount int
	
	if totalBatches > 0 {
		avgBatchSize = float64(totalRecords) / float64(totalBatches)
		avgProcessingTime = time.Duration(int64(totalProcessingTime) / totalBatches)
		errorRate = float64(failedBatches) / float64(totalBatches)
	}
	
	// 計算當前批次記錄總數
	for _, batch := range bp.currentBatches {
		currentBatchCount += len(batch.Records)
	}
	
	return BatchMetrics{
		TotalBatches:          totalBatches,
		TotalRecords:          totalRecords,
		SuccessfulBatches:     successfulBatches,
		FailedBatches:         failedBatches,
		AverageBatchSize:      avgBatchSize,
		AverageProcessingTime: avgProcessingTime,
		ErrorRate:             errorRate,
		LastProcessedAt:       time.Now(), // 實際應該追蹤最後處理時間
		CurrentBatchCount:     currentBatchCount,
	}
}

// 私有輔助方法

// getOrCreateBatch 獲取或創建批次
func (bp *DefaultBatchProcessor) getOrCreateBatch(shardID string) *RecordBatch {
	batch, exists := bp.currentBatches[shardID]
	if !exists {
		ctx, cancel := context.WithTimeout(context.Background(), bp.maxBatchWaitTime)
		batch = &RecordBatch{
			Records:        make([]types.Record, 0, bp.batchSize),
			ShardID:        shardID,
			ProcessingTime: time.Now(),
			BatchSize:      bp.batchSize,
			Timeout:        bp.maxBatchWaitTime,
			ctx:            ctx,
			cancel:         cancel,
		}
		bp.currentBatches[shardID] = batch
	}
	return batch
}

// processBatchInternal 內部批次處理邏輯
func (bp *DefaultBatchProcessor) processBatchInternal(batch *RecordBatch) error {
	if len(batch.Records) == 0 {
		return nil
	}
	
	// 處理批次
	_ = bp.ProcessBatch(batch.ctx, batch)
	
	// 清理批次
	delete(bp.currentBatches, batch.ShardID)
	batch.cancel()
	
	return nil
}

// periodicFlush 定期刷新批次
func (bp *DefaultBatchProcessor) periodicFlush(ctx context.Context) {
	for {
		select {
		case <-bp.flushTicker.C:
			bp.checkAndFlushExpiredBatches()
		case <-bp.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// checkAndFlushExpiredBatches 檢查並刷新過期批次
func (bp *DefaultBatchProcessor) checkAndFlushExpiredBatches() {
	bp.mu.RLock()
	shardIDs := make([]string, 0, len(bp.currentBatches))
	for shardID := range bp.currentBatches {
		shardIDs = append(shardIDs, shardID)
	}
	bp.mu.RUnlock()
	
	for _, shardID := range shardIDs {
		bp.mu.RLock()
		batchLock, exists := bp.batchLocks[shardID]
		bp.mu.RUnlock()
		
		if !exists {
			continue
		}
		
		batchLock.Lock()
		batch, exists := bp.currentBatches[shardID]
		if exists && len(batch.Records) > 0 {
			// 檢查批次是否過期
			if time.Since(batch.ProcessingTime) >= bp.maxBatchWaitTime {
				bp.logger.InfoLog(
					"Flushing expired batch",
					bp.logger.String("shard_id", shardID),
					bp.logger.Int("record_count", len(batch.Records)),
				)
				bp.processBatchInternal(batch)
			}
		}
		batchLock.Unlock()
	}
}

// flushAllBatches 刷新所有待處理批次
func (bp *DefaultBatchProcessor) flushAllBatches() {
	bp.mu.RLock()
	shardIDs := make([]string, 0, len(bp.currentBatches))
	for shardID := range bp.currentBatches {
		shardIDs = append(shardIDs, shardID)
	}
	bp.mu.RUnlock()
	
	for _, shardID := range shardIDs {
		_ = bp.FlushBatch(shardID)
	}
}

// isRetryableError 判斷錯誤是否可重試
func (bp *DefaultBatchProcessor) isRetryableError(err error) bool {
	if bp.kdsService.errorClassifier != nil {
		return bp.kdsService.errorClassifier.IsRetryable(err) || bp.kdsService.errorClassifier.IsTemporary(err)
	}
	return false
}