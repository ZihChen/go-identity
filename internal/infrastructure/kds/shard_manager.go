package kds

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/go-redsync/redsync/v4"
	"go.opentelemetry.io/otel/attribute"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

// ShardState 分片狀態
type ShardState int

const (
	ShardStateIdle       ShardState = iota // 空閒
	ShardStateProcessing                   // 處理中
	ShardStateLocked                       // 已鎖定
	ShardStateError                        // 錯誤狀態
	ShardStateClosed                       // 已關閉
)

// ShardInfo 分片信息
type ShardInfo struct {
	ID                string                `json:"id"`                  // 分片 ID
	Iterator          string                `json:"iterator"`            // 當前迭代器
	NextIterator      string                `json:"next_iterator"`       // 下一個迭代器
	State             ShardState            `json:"state"`               // 分片狀態
	LastSequenceNum   string                `json:"last_sequence_num"`   // 最後處理的序列號
	ProcessedRecords  int64                 `json:"processed_records"`   // 處理的記錄數
	ErrorCount        int64                 `json:"error_count"`         // 錯誤計數
	LastProcessedAt   time.Time             `json:"last_processed_at"`   // 最後處理時間
	Lock              *redsync.Mutex        `json:"-"`                   // 分散式鎖（不序列化）
	LockAcquiredAt    time.Time             `json:"lock_acquired_at"`    // 鎖獲取時間
	LockTimeout       time.Duration         `json:"lock_timeout"`        // 鎖超時時間
	IteratorExpiry    time.Time             `json:"iterator_expiry"`     // 迭代器過期時間
	BackoffStrategy   BackoffStrategy       `json:"-"`                   // 退避策略（不序列化）
	WorkerAssignment  int                   `json:"worker_assignment"`   // 分配的工作者 ID
	ConsumerInstance  string                `json:"consumer_instance"`   // 消費者實例 ID
}

// ShardMetrics 分片指標
type ShardMetrics struct {
	ShardID           string        `json:"shard_id"`            // 分片 ID
	State             ShardState    `json:"state"`               // 狀態
	ProcessedRecords  int64         `json:"processed_records"`   // 處理記錄數
	ErrorCount        int64         `json:"error_count"`         // 錯誤計數
	ProcessingRate    float64       `json:"processing_rate"`     // 處理速率
	LastProcessedAt   time.Time     `json:"last_processed_at"`   // 最後處理時間
	LockDuration      time.Duration `json:"lock_duration"`       // 鎖持有時間
	IteratorAge       time.Duration `json:"iterator_age"`        // 迭代器年齡
	WorkerAssignment  int           `json:"worker_assignment"`   // 工作者分配
	ConsumerInstance  string        `json:"consumer_instance"`   // 消費者實例
}

// ShardManager 分片管理器介面
type ShardManager interface {
	// Initialize 初始化分片管理器
	Initialize(ctx context.Context) error
	
	// GetShardIterators 獲取所有分片迭代器
	GetShardIterators(ctx context.Context) (map[string]string, error)
	
	// ProcessShard 處理指定分片
	ProcessShard(ctx context.Context, shardID string, iterator string) error
	
	// UpdateCheckpoint 更新分片檢查點
	UpdateCheckpoint(ctx context.Context, shardID, sequenceNumber string) error
	
	// RefreshIterator 刷新分片迭代器
	RefreshIterator(ctx context.Context, shardID string) (string, error)
	
	// AcquireShardLock 獲取分片鎖
	AcquireShardLock(ctx context.Context, shardID string) (*redsync.Mutex, error)
	
	// ReleaseShardLock 釋放分片鎖
	ReleaseShardLock(ctx context.Context, shardID string, mutex *redsync.Mutex) error
	
	// GetShardInfo 獲取分片詳細信息
	GetShardInfo(ctx context.Context, shardID string) (*ShardInfo, error)
	
	// GetAllShardMetrics 獲取所有分片指標
	GetAllShardMetrics(ctx context.Context) ([]ShardMetrics, error)
	
	// IsHealthy 檢查分片管理器健康狀態
	IsHealthy(ctx context.Context) bool
	
	// RebalanceShards 重新平衡分片分配
	RebalanceShards(ctx context.Context) error
}

// DefaultShardManager 預設分片管理器實現
type DefaultShardManager struct {
	// 依賴組件
	kdsService      *KDSService            // KDS 服務
	batchProcessor  BatchProcessor         // 批次處理器
	workerPool      WorkerPool             // 工作者池
	metrics         *MetricsCollector      // 指標收集器
	backoffStrategy BackoffStrategy        // 退避策略
	logger          infrastructure.Logger          // 日誌記錄器
	
	// 配置參數
	maxShardConcurrency int           // 最大並行分片數
	shardLockTimeout    time.Duration // 分片鎖超時時間
	iteratorRefreshRate time.Duration // 迭代器刷新頻率
	consumerInstanceID  string        // 消費者實例 ID
	
	// 內部狀態
	shards          map[string]*ShardInfo // 分片信息 map
	activeShardsNum int32                 // 活躍分片數
	processingWg    sync.WaitGroup        // 處理等待組
	stopChan        chan struct{}         // 停止通道
	isRunning       int32                 // 運行狀態
	
	// 同步
	mu sync.RWMutex // 讀寫鎖
}

// NewDefaultShardManager 創建預設分片管理器
func NewDefaultShardManager(
	kdsService *KDSService,
	batchProcessor BatchProcessor,
	workerPool WorkerPool,
	metrics *MetricsCollector,
	backoffStrategy BackoffStrategy,
	logger infrastructure.Logger,
	maxShardConcurrency int,
	shardLockTimeout time.Duration,
	consumerInstanceID string,
) *DefaultShardManager {
	if maxShardConcurrency <= 0 {
		maxShardConcurrency = 8
	}
	if shardLockTimeout <= 0 {
		shardLockTimeout = 1 * time.Minute
	}
	if consumerInstanceID == "" {
		consumerInstanceID = fmt.Sprintf("consumer-%d", time.Now().Unix())
	}
	
	return &DefaultShardManager{
		kdsService:          kdsService,
		batchProcessor:      batchProcessor,
		workerPool:          workerPool,
		metrics:             metrics,
		backoffStrategy:     backoffStrategy,
		logger:              logger,
		maxShardConcurrency: maxShardConcurrency,
		shardLockTimeout:    shardLockTimeout,
		iteratorRefreshRate: 5 * time.Minute,
		consumerInstanceID:  consumerInstanceID,
		shards:              make(map[string]*ShardInfo),
		stopChan:            make(chan struct{}),
	}
}

// Initialize 初始化分片管理器
func (sm *DefaultShardManager) Initialize(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&sm.isRunning, 0, 1) {
		return fmt.Errorf("shard manager is already running")
	}
	
	sm.logger.InfoWithContext(
		ctx,
		"Initializing shard manager",
		sm.logger.Int("max_shard_concurrency", sm.maxShardConcurrency),
		sm.logger.String("shard_lock_timeout", sm.shardLockTimeout.String()),
		sm.logger.String("consumer_instance_id", sm.consumerInstanceID),
	)
	
	// 獲取初始分片信息
	iterators, err := sm.GetShardIterators(ctx)
	if err != nil {
		atomic.StoreInt32(&sm.isRunning, 0)
		return fmt.Errorf("failed to get initial shard iterators: %w", err)
	}
	
	// 初始化分片信息
	sm.mu.Lock()
	for shardID, iterator := range iterators {
		shardInfo := &ShardInfo{
			ID:               shardID,
			Iterator:         iterator,
			State:            ShardStateIdle,
			BackoffStrategy:  sm.backoffStrategy,
			ConsumerInstance: sm.consumerInstanceID,
			LockTimeout:      sm.shardLockTimeout,
			IteratorExpiry:   time.Now().Add(sm.iteratorRefreshRate),
		}
		sm.shards[shardID] = shardInfo
	}
	sm.mu.Unlock()
	
	// 啟動分片監控和維護 goroutine
	go sm.shardMaintenanceLoop(ctx)
	
	sm.logger.InfoWithContext(
		ctx,
		"Shard manager initialized successfully",
		sm.logger.Int("total_shards", len(sm.shards)),
	)
	
	return nil
}

// GetShardIterators 獲取所有分片迭代器
func (sm *DefaultShardManager) GetShardIterators(ctx context.Context) (map[string]string, error) {
	// 使用 KDSService 的現有實現
	return sm.kdsService.getShardIterators(ctx)
}

// ProcessShard 處理指定分片
func (sm *DefaultShardManager) ProcessShard(ctx context.Context, shardID string, iterator string) error {
	// 檢查並發限制
	activeShards := atomic.LoadInt32(&sm.activeShardsNum)
	if activeShards >= int32(sm.maxShardConcurrency) {
		return fmt.Errorf("maximum shard concurrency reached: %d", sm.maxShardConcurrency)
	}
	
	// 獲取分片信息
	sm.mu.RLock()
	shardInfo, exists := sm.shards[shardID]
	sm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("shard %s not found", shardID)
	}
	
	// 嘗試獲取分片鎖
	mutex, err := sm.AcquireShardLock(ctx, shardID)
	if err != nil {
		sm.logger.WarnWithContext(
			ctx,
			"Failed to acquire shard lock",
			sm.logger.String("shard_id", shardID),
			sm.logger.Error("error", err),
		)
		return err
	}
	
	// 更新分片狀態
	sm.mu.Lock()
	shardInfo.State = ShardStateLocked
	shardInfo.Lock = mutex
	shardInfo.LockAcquiredAt = time.Now()
	atomic.AddInt32(&sm.activeShardsNum, 1)
	sm.mu.Unlock()
	
	// 啟動分片處理 goroutine
	sm.processingWg.Add(1)
	go sm.processShardRecords(ctx, shardInfo)
	
	return nil
}

// processShardRecords 處理分片記錄的主循環
func (sm *DefaultShardManager) processShardRecords(ctx context.Context, shardInfo *ShardInfo) {
	defer func() {
		// 清理工作
		atomic.AddInt32(&sm.activeShardsNum, -1)
		sm.processingWg.Done()
		
		// 釋放鎖
		if shardInfo.Lock != nil {
			_ = sm.ReleaseShardLock(ctx, shardInfo.ID, shardInfo.Lock)
		}
		
		// 更新狀態
		sm.mu.Lock()
		shardInfo.State = ShardStateIdle
		shardInfo.Lock = nil
		sm.mu.Unlock()
		
		sm.logger.InfoWithContext(
			ctx,
			"Shard processing completed",
			sm.logger.String("shard_id", shardInfo.ID),
			sm.logger.Int64("processed_records", shardInfo.ProcessedRecords),
		)
	}()
	
	// 更新分片狀態為處理中
	sm.mu.Lock()
	shardInfo.State = ShardStateProcessing
	sm.mu.Unlock()
	
	currentIterator := shardInfo.Iterator
	consecutiveErrors := int64(0)
	
	sm.logger.InfoWithContext(
		ctx,
		"Starting shard processing",
		sm.logger.String("shard_id", shardInfo.ID),
	)
	
	// 分片處理主循環
	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		default:
			// 檢查迭代器是否需要刷新
			if time.Now().After(shardInfo.IteratorExpiry) {
				newIterator, err := sm.RefreshIterator(ctx, shardInfo.ID)
				if err != nil {
					sm.logger.ErrorWithContext(
						ctx,
						"Failed to refresh iterator",
						sm.logger.String("shard_id", shardInfo.ID),
						sm.logger.Error("error", err),
					)
					
					// 使用退避策略
					backoffDuration := sm.backoffStrategy.NextBackoff()
					time.Sleep(backoffDuration)
					continue
				}
				currentIterator = newIterator
			}
			
			// 獲取記錄
			records, nextIterator, err := sm.getRecordsFromShard(ctx, currentIterator)
			if err != nil {
				consecutiveErrors++
				atomic.AddInt64(&shardInfo.ErrorCount, 1)
				
				sm.logger.ErrorWithContext(
					ctx,
					"Failed to get records from shard",
					sm.logger.String("shard_id", shardInfo.ID),
					sm.logger.Int64("consecutive_errors", consecutiveErrors),
					sm.logger.Error("error", err),
				)
				
				// 如果連續錯誤過多，暫停處理
				if consecutiveErrors >= 5 {
					sm.logger.ErrorWithContext(
						ctx,
						"Too many consecutive errors, pausing shard processing",
						sm.logger.String("shard_id", shardInfo.ID),
					)
					
					sm.mu.Lock()
					shardInfo.State = ShardStateError
					sm.mu.Unlock()
					
					// 長時間退避
					time.Sleep(5 * time.Minute)
					consecutiveErrors = 0
					
					sm.mu.Lock()
					shardInfo.State = ShardStateProcessing
					sm.mu.Unlock()
				} else {
					// 使用退避策略
					backoffDuration := sm.backoffStrategy.NextBackoff()
					time.Sleep(backoffDuration)
				}
				continue
			}
			
			// 重置錯誤計數
			consecutiveErrors = 0
			sm.backoffStrategy.Reset()
			
			// 如果沒有記錄，短暫休息
			if len(records) == 0 {
				time.Sleep(500 * time.Millisecond)
				currentIterator = nextIterator
				continue
			}
			
			// 處理記錄批次
			err = sm.processBatchOfRecords(ctx, shardInfo, records)
			if err != nil {
				sm.logger.ErrorWithContext(
					ctx,
					"Failed to process batch of records",
					sm.logger.String("shard_id", shardInfo.ID),
					sm.logger.Int("record_count", len(records)),
					sm.logger.Error("error", err),
				)
				
				// 記錄錯誤但繼續處理
				atomic.AddInt64(&shardInfo.ErrorCount, 1)
			} else {
				// 更新處理統計
				atomic.AddInt64(&shardInfo.ProcessedRecords, int64(len(records)))
				shardInfo.LastProcessedAt = time.Now()
				
				// 更新檢查點
				if len(records) > 0 {
					lastRecord := records[len(records)-1]
					if lastRecord.SequenceNumber != nil {
						_ = sm.UpdateCheckpoint(ctx, shardInfo.ID, *lastRecord.SequenceNumber)
					}
				}
			}
			
			// 更新迭代器
			currentIterator = nextIterator
			if currentIterator == "" {
				// 分片已關閉
				sm.logger.InfoWithContext(
					ctx,
					"Shard has been closed",
					sm.logger.String("shard_id", shardInfo.ID),
				)
				
				sm.mu.Lock()
				shardInfo.State = ShardStateClosed
				sm.mu.Unlock()
				return
			}
		}
	}
}

// getRecordsFromShard 從分片獲取記錄
func (sm *DefaultShardManager) getRecordsFromShard(
	ctx context.Context, 
	iterator string,
) ([]types.Record, string, error) {
	
	ctx, span := tracing.StartSpan(ctx, "ShardManager.GetRecordsFromShard")
	defer tracing.SpanEnd(span)
	
	recordsOutput, err := sm.kdsService.client.GetRecords(ctx, &kinesis.GetRecordsInput{
		ShardIterator: aws.String(iterator),
		Limit:         aws.Int32(1000),
	})
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, "", fmt.Errorf("failed to get records: %w", err)
	}
	
	nextIterator := ""
	if recordsOutput.NextShardIterator != nil {
		nextIterator = *recordsOutput.NextShardIterator
	}
	
	tracing.RecordSpanAttributes(span,
		attribute.Int("records.count", len(recordsOutput.Records)),
		attribute.Bool("records.has_next_iterator", nextIterator != ""),
	)
	
	return recordsOutput.Records, nextIterator, nil
}

// processBatchOfRecords 處理記錄批次
func (sm *DefaultShardManager) processBatchOfRecords(
	ctx context.Context, 
	shardInfo *ShardInfo, 
	records []types.Record,
) error {
	
	if sm.batchProcessor == nil {
		return fmt.Errorf("batch processor not available")
	}
	
	// 創建批次
	batch := &RecordBatch{
		Records:        records,
		ShardID:        shardInfo.ID,
		ProcessingTime: time.Now(),
		BatchSize:      len(records),
		ctx:            ctx,
	}
	
	// 處理批次
	results := sm.batchProcessor.ProcessBatch(ctx, batch)
	
	// 檢查處理結果
	var errors []error
	for _, result := range results {
		if !result.Success && result.Error != nil {
			errors = append(errors, result.Error)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("batch processing had %d errors: %v", len(errors), errors[0])
	}
	
	return nil
}

// UpdateCheckpoint 更新分片檢查點
func (sm *DefaultShardManager) UpdateCheckpoint(ctx context.Context, shardID, sequenceNumber string) error {
	// 使用 KDSService 的現有實現
	err := sm.kdsService.updateCheckpoint(ctx, shardID, sequenceNumber)
	if err != nil {
		return err
	}
	
	// 更新本地分片信息
	sm.mu.Lock()
	if shardInfo, exists := sm.shards[shardID]; exists {
		shardInfo.LastSequenceNum = sequenceNumber
	}
	sm.mu.Unlock()
	
	return nil
}

// RefreshIterator 刷新分片迭代器
func (sm *DefaultShardManager) RefreshIterator(ctx context.Context, shardID string) (string, error) {
	sm.mu.RLock()
	shardInfo, exists := sm.shards[shardID]
	sm.mu.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("shard %s not found", shardID)
	}
	
	// 獲取新的迭代器（基於最後的檢查點）
	iterators, err := sm.GetShardIterators(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to refresh iterators: %w", err)
	}
	
	newIterator, exists := iterators[shardID]
	if !exists {
		return "", fmt.Errorf("shard %s no longer available", shardID)
	}
	
	// 更新分片信息
	sm.mu.Lock()
	shardInfo.Iterator = newIterator
	shardInfo.IteratorExpiry = time.Now().Add(sm.iteratorRefreshRate)
	sm.mu.Unlock()
	
	sm.logger.InfoWithContext(
		ctx,
		"Refreshed shard iterator",
		sm.logger.String("shard_id", shardID),
	)
	
	return newIterator, nil
}

// AcquireShardLock 獲取分片鎖
func (sm *DefaultShardManager) AcquireShardLock(ctx context.Context, shardID string) (*redsync.Mutex, error) {
	mutexKey := fmt.Sprintf(consts.ShardMutexRedisKey, sm.kdsService.consumeStream, shardID)
	mutex, err := sm.kdsService.redisManager.GetMutex(mutexKey, sm.shardLockTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create mutex: %w", err)
	}
	
	if err := mutex.Lock(); err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	
	return mutex, nil
}

// ReleaseShardLock 釋放分片鎖
func (sm *DefaultShardManager) ReleaseShardLock(ctx context.Context, shardID string, mutex *redsync.Mutex) error {
	if mutex == nil {
		return nil
	}
	
	ok, err := mutex.Unlock()
	if !ok || err != nil {
		sm.logger.WarnWithContext(
			ctx,
			"Failed to release shard lock",
			sm.logger.String("shard_id", shardID),
			sm.logger.Error("error", err),
		)
		return fmt.Errorf("failed to release lock: %w", err)
	}
	
	return nil
}

// GetShardInfo 獲取分片詳細信息
func (sm *DefaultShardManager) GetShardInfo(ctx context.Context, shardID string) (*ShardInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	shardInfo, exists := sm.shards[shardID]
	if !exists {
		return nil, fmt.Errorf("shard %s not found", shardID)
	}
	
	// 創建副本以避免併發修改
	infoCopy := *shardInfo
	return &infoCopy, nil
}

// GetAllShardMetrics 獲取所有分片指標
func (sm *DefaultShardManager) GetAllShardMetrics(ctx context.Context) ([]ShardMetrics, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	metrics := make([]ShardMetrics, 0, len(sm.shards))
	
	for _, shardInfo := range sm.shards {
		// 計算處理速率
		var processingRate float64
		if !shardInfo.LastProcessedAt.IsZero() {
			elapsed := time.Since(shardInfo.LastProcessedAt).Minutes()
			if elapsed > 0 {
				processingRate = float64(shardInfo.ProcessedRecords) / elapsed
			}
		}
		
		// 計算鎖持有時間
		var lockDuration time.Duration
		if !shardInfo.LockAcquiredAt.IsZero() {
			lockDuration = time.Since(shardInfo.LockAcquiredAt)
		}
		
		// 計算迭代器年齡
		var iteratorAge time.Duration
		if !shardInfo.IteratorExpiry.IsZero() {
			iteratorAge = sm.iteratorRefreshRate - time.Until(shardInfo.IteratorExpiry)
		}
		
		metrics = append(metrics, ShardMetrics{
			ShardID:          shardInfo.ID,
			State:            shardInfo.State,
			ProcessedRecords: shardInfo.ProcessedRecords,
			ErrorCount:       shardInfo.ErrorCount,
			ProcessingRate:   processingRate,
			LastProcessedAt:  shardInfo.LastProcessedAt,
			LockDuration:     lockDuration,
			IteratorAge:      iteratorAge,
			WorkerAssignment: shardInfo.WorkerAssignment,
			ConsumerInstance: shardInfo.ConsumerInstance,
		})
	}
	
	return metrics, nil
}

// IsHealthy 檢查分片管理器健康狀態
func (sm *DefaultShardManager) IsHealthy(ctx context.Context) bool {
	if atomic.LoadInt32(&sm.isRunning) == 0 {
		return false
	}
	
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	totalShards := len(sm.shards)
	healthyShards := 0
	
	for _, shardInfo := range sm.shards {
		// 健康標準：
		// 1. 不在錯誤狀態
		// 2. 最近有活動或者是新創建的分片
		// 3. 錯誤率不能太高
		
		if shardInfo.State != ShardStateError {
			recentActivity := time.Since(shardInfo.LastProcessedAt) < 10*time.Minute
			lowErrorRate := shardInfo.ProcessedRecords == 0 || 
							float64(shardInfo.ErrorCount)/float64(shardInfo.ProcessedRecords) < 0.1
			
			if (recentActivity || shardInfo.LastProcessedAt.IsZero()) && lowErrorRate {
				healthyShards++
			}
		}
	}
	
	// 至少70%的分片健康才認為整體健康
	return float64(healthyShards)/float64(totalShards) >= 0.7
}

// RebalanceShards 重新平衡分片分配
func (sm *DefaultShardManager) RebalanceShards(ctx context.Context) error {
	sm.logger.InfoWithContext(ctx, "Starting shard rebalancing")
	
	// 簡化的重新平衡邏輯
	// 實際實現可能需要考慮更多因素，如工作者負載、分片處理速率等
	
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// 統計當前分配情況
	workerAssignments := make(map[int][]string)
	for shardID, shardInfo := range sm.shards {
		workerID := shardInfo.WorkerAssignment
		workerAssignments[workerID] = append(workerAssignments[workerID], shardID)
	}
	
	// 重新分配分片（簡單輪詢策略）
	shardList := make([]string, 0, len(sm.shards))
	for shardID := range sm.shards {
		shardList = append(shardList, shardID)
	}
	
	// 假設有固定數量的工作者
	numWorkers := 5 // 可以從工作者池獲取實際數量
	
	for i, shardID := range shardList {
		newWorkerID := (i % numWorkers) + 1
		sm.shards[shardID].WorkerAssignment = newWorkerID
	}
	
	sm.logger.InfoWithContext(
		ctx,
		"Shard rebalancing completed",
		sm.logger.Int("total_shards", len(shardList)),
		sm.logger.Int("num_workers", numWorkers),
	)
	
	return nil
}

// Stop 停止分片管理器
func (sm *DefaultShardManager) Stop() error {
	if !atomic.CompareAndSwapInt32(&sm.isRunning, 1, 0) {
		return fmt.Errorf("shard manager is not running")
	}
	
	sm.logger.InfoLog("Stopping shard manager")
	
	// 發送停止信號
	close(sm.stopChan)
	
	// 等待所有分片處理完成
	done := make(chan struct{})
	go func() {
		sm.processingWg.Wait()
		close(done)
	}()
	
	// 設置超時等待
	select {
	case <-done:
		sm.logger.InfoLog("All shard processing stopped gracefully")
	case <-time.After(30 * time.Second):
		sm.logger.WarnLog("Shard manager stop timeout")
	}
	
	// 釋放所有分片鎖
	sm.mu.Lock()
	for _, shardInfo := range sm.shards {
		if shardInfo.Lock != nil {
			_ = sm.ReleaseShardLock(context.Background(), shardInfo.ID, shardInfo.Lock)
		}
	}
	sm.mu.Unlock()
	
	sm.logger.InfoLog("Shard manager stopped")
	return nil
}

// 私有維護方法

// shardMaintenanceLoop 分片維護循環
func (sm *DefaultShardManager) shardMaintenanceLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			sm.performMaintenance(ctx)
		case <-sm.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performMaintenance 執行維護任務
func (sm *DefaultShardManager) performMaintenance(ctx context.Context) {
	// 檢查迭代器過期
	sm.checkExpiredIterators(ctx)
	
	// 檢查長時間無活動的分片
	sm.checkStaleShards(ctx)
	
	// 更新指標
	if sm.metrics != nil {
		sm.updateShardMetrics(ctx)
	}
}

// checkExpiredIterators 檢查過期的迭代器
func (sm *DefaultShardManager) checkExpiredIterators(ctx context.Context) {
	sm.mu.RLock()
	expiredShards := make([]string, 0)
	
	for shardID, shardInfo := range sm.shards {
		if time.Now().After(shardInfo.IteratorExpiry) && 
		   shardInfo.State == ShardStateIdle {
			expiredShards = append(expiredShards, shardID)
		}
	}
	sm.mu.RUnlock()
	
	// 刷新過期的迭代器
	for _, shardID := range expiredShards {
		_, err := sm.RefreshIterator(ctx, shardID)
		if err != nil {
			sm.logger.WarnWithContext(
				ctx,
				"Failed to refresh expired iterator during maintenance",
				sm.logger.String("shard_id", shardID),
				sm.logger.Error("error", err),
			)
		}
	}
}

// checkStaleShards 檢查長時間無活動的分片
func (sm *DefaultShardManager) checkStaleShards(ctx context.Context) {
	sm.mu.RLock()
	staleThreshold := time.Now().Add(-10 * time.Minute)
	staleShards := make([]string, 0)
	
	for shardID, shardInfo := range sm.shards {
		if !shardInfo.LastProcessedAt.IsZero() && 
		   shardInfo.LastProcessedAt.Before(staleThreshold) &&
		   shardInfo.State == ShardStateProcessing {
			staleShards = append(staleShards, shardID)
		}
	}
	sm.mu.RUnlock()
	
	// 記錄長時間無活動的分片
	for _, shardID := range staleShards {
		sm.logger.WarnWithContext(
			ctx,
			"Detected stale shard",
			sm.logger.String("shard_id", shardID),
		)
	}
}

// updateShardMetrics 更新分片指標
func (sm *DefaultShardManager) updateShardMetrics(ctx context.Context) {
	metrics, err := sm.GetAllShardMetrics(ctx)
	if err != nil {
		return
	}
	
	for _, metric := range metrics {
		if sm.metrics != nil {
			sm.metrics.RecordProcessed(metric.ProcessedRecords, 0)
		}
	}
}