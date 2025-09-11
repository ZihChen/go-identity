package kds

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"go.opentelemetry.io/otel/attribute"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

// WorkerTask 工作任務結構
type WorkerTask struct {
	ID         string        `json:"id"`          // 任務 ID
	Record     types.Record  `json:"record"`      // KDS 記錄
	ShardID    string        `json:"shard_id"`    // 分片 ID
	Priority   int           `json:"priority"`    // 任務優先級（1-10，數字越大優先級越高）
	Timestamp  time.Time     `json:"timestamp"`   // 任務創建時間
	RetryCount int           `json:"retry_count"` // 重試次數
	Context    context.Context `json:"-"`         // 任務上下文（不序列化）
}

// WorkerResult 工作結果結構
type WorkerResult struct {
	TaskID         string        `json:"task_id"`         // 任務 ID
	WorkerID       int           `json:"worker_id"`       // 工作者 ID
	Success        bool          `json:"success"`         // 是否成功
	ProcessingTime time.Duration `json:"processing_time"` // 處理時間
	Error          error         `json:"error"`           // 錯誤信息
	Result         ProcessResult `json:"result"`          // 處理結果
	Timestamp      time.Time     `json:"timestamp"`       // 完成時間
}

// Worker 工作者結構
type Worker struct {
	ID              int                    // 工作者 ID
	taskChan        <-chan WorkerTask      // 任務接收通道
	resultChan      chan<- WorkerResult    // 結果發送通道
	stopChan        chan struct{}          // 停止通道
	kdsService      *KDSService            // KDS 服務
	logger          infrastructure.Logger          // 日誌記錄器
	isRunning       int32                  // 運行狀態
	
	// 統計信息
	tasksProcessed  int64     // 處理的任務數
	tasksSucceeded  int64     // 成功的任務數
	tasksFailed     int64     // 失敗的任務數
	totalTime       int64     // 總處理時間（納秒）
	lastTaskTime    time.Time // 最後處理任務時間
	startTime       time.Time // 工作者啟動時間
	
	// Panic 恢復配置
	enablePanicRecovery bool // 是否啟用 Panic 恢復
	maxRecoveryAttempts int  // 最大恢復嘗試次數
	recoveryAttempts    int  // 當前恢復嘗試次數
}

// WorkerStats 工作者統計信息
type WorkerStats struct {
	WorkerID        int           `json:"worker_id"`         // 工作者 ID
	IsRunning       bool          `json:"is_running"`        // 是否運行中
	TasksProcessed  int64         `json:"tasks_processed"`   // 處理的任務數
	TasksSucceeded  int64         `json:"tasks_succeeded"`   // 成功的任務數
	TasksFailed     int64         `json:"tasks_failed"`      // 失敗的任務數
	SuccessRate     float64       `json:"success_rate"`      // 成功率
	AverageTime     time.Duration `json:"average_time"`      // 平均處理時間
	LastTaskTime    time.Time     `json:"last_task_time"`    // 最後處理任務時間
	StartTime       time.Time     `json:"start_time"`        // 啟動時間
	Uptime          time.Duration `json:"uptime"`            // 運行時長
	RecoveryAttempts int          `json:"recovery_attempts"` // 恢復嘗試次數
}

// WorkerPool 工作者池介面
type WorkerPool interface {
	// Start 啟動工作者池
	Start(ctx context.Context) error
	
	// Stop 停止工作者池
	Stop() error
	
	// SubmitTask 提交任務
	SubmitTask(task WorkerTask) error
	
	// GetStats 獲取工作者池統計信息
	GetStats() WorkerPoolStats
	
	// GetWorkerStats 獲取所有工作者統計信息
	GetWorkerStats() []WorkerStats
	
	// IsHealthy 檢查工作者池健康狀態
	IsHealthy() bool
	
	// ResizePool 動態調整工作者數量
	ResizePool(newSize int) error
}

// WorkerPoolStats 工作者池統計信息
type WorkerPoolStats struct {
	TotalWorkers      int           `json:"total_workers"`       // 總工作者數
	ActiveWorkers     int           `json:"active_workers"`      // 活躍工作者數
	IdleWorkers       int           `json:"idle_workers"`        // 空閒工作者數
	QueuedTasks       int           `json:"queued_tasks"`        // 等待中的任務數
	ProcessedTasks    int64         `json:"processed_tasks"`     // 總處理任務數
	SuccessfulTasks   int64         `json:"successful_tasks"`    // 成功任務數
	FailedTasks       int64         `json:"failed_tasks"`        // 失敗任務數
	AverageLatency    time.Duration `json:"average_latency"`     // 平均延遲
	Throughput        float64       `json:"throughput"`          // 吞吐量（任務/秒）
	LastActivityTime  time.Time     `json:"last_activity_time"`  // 最後活動時間
	PoolUtilization   float64       `json:"pool_utilization"`    // 池利用率
}

// DefaultWorkerPool 預設工作者池實現
type DefaultWorkerPool struct {
	// 配置
	workerCount   int           // 工作者數量
	bufferSize    int           // 任務緩衝區大小
	kdsService    *KDSService   // KDS 服務
	logger        infrastructure.Logger // 日誌記錄器
	
	// 通道
	taskChan     chan WorkerTask   // 任務通道
	resultChan   chan WorkerResult // 結果通道
	stopChan     chan struct{}     // 停止通道
	
	// 工作者管理
	workers      []*Worker    // 工作者列表
	workerWg     sync.WaitGroup // 工作者等待組
	isRunning    int32        // 運行狀態
	
	// 統計
	totalTasks     int64     // 總任務數
	completedTasks int64     // 完成任務數
	failedTasks    int64     // 失敗任務數
	startTime      time.Time // 啟動時間
	
	// Panic 恢復配置
	enablePanicRecovery bool // 是否啟用 Panic 恢復
	maxRecoveryAttempts int  // 最大恢復嘗試次數
	
	// 同步
	mu sync.RWMutex // 讀寫鎖
}

// NewDefaultWorkerPool 創建預設工作者池
func NewDefaultWorkerPool(
	workerCount int,
	bufferSize int,
	kdsService *KDSService,
	logger infrastructure.Logger,
	enablePanicRecovery bool,
	maxRecoveryAttempts int,
) *DefaultWorkerPool {
	// 參數驗證
	if workerCount <= 0 {
		workerCount = 5
	}
	if bufferSize <= 0 {
		bufferSize = workerCount * 100
	}
	if maxRecoveryAttempts <= 0 {
		maxRecoveryAttempts = 3
	}
	
	return &DefaultWorkerPool{
		workerCount:         workerCount,
		bufferSize:          bufferSize,
		kdsService:          kdsService,
		logger:              logger,
		enablePanicRecovery: enablePanicRecovery,
		maxRecoveryAttempts: maxRecoveryAttempts,
		taskChan:            make(chan WorkerTask, bufferSize),
		resultChan:          make(chan WorkerResult, bufferSize),
		stopChan:            make(chan struct{}),
		workers:             make([]*Worker, 0, workerCount),
	}
}

// Start 啟動工作者池
func (wp *DefaultWorkerPool) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&wp.isRunning, 0, 1) {
		return fmt.Errorf("worker pool is already running")
	}
	
	wp.startTime = time.Now()
	
	wp.logger.InfoWithContext(
		ctx,
		"Starting worker pool",
		wp.logger.Int("worker_count", wp.workerCount),
		wp.logger.Int("buffer_size", wp.bufferSize),
		wp.logger.Bool("panic_recovery", wp.enablePanicRecovery),
	)
	
	// 啟動工作者
	for i := 0; i < wp.workerCount; i++ {
		worker := &Worker{
			ID:                  i + 1,
			taskChan:            wp.taskChan,
			resultChan:          wp.resultChan,
			stopChan:            wp.stopChan,
			kdsService:          wp.kdsService,
			logger:              wp.logger,
			enablePanicRecovery: wp.enablePanicRecovery,
			maxRecoveryAttempts: wp.maxRecoveryAttempts,
			startTime:           time.Now(),
		}
		
		wp.workers = append(wp.workers, worker)
		wp.workerWg.Add(1)
		
		go wp.runWorker(ctx, worker)
	}
	
	// 啟動結果處理 goroutine
	go wp.processResults(ctx)
	
	wp.logger.InfoWithContext(
		ctx,
		"Worker pool started successfully",
		wp.logger.Int("active_workers", len(wp.workers)),
	)
	
	return nil
}

// Stop 停止工作者池
func (wp *DefaultWorkerPool) Stop() error {
	if !atomic.CompareAndSwapInt32(&wp.isRunning, 1, 0) {
		return fmt.Errorf("worker pool is not running")
	}
	
	wp.logger.InfoLog("Stopping worker pool")
	
	// 關閉任務通道，不再接受新任務
	close(wp.taskChan)
	
	// 等待所有工作者完成
	done := make(chan struct{})
	go func() {
		wp.workerWg.Wait()
		close(done)
	}()
	
	// 設置超時等待
	select {
	case <-done:
		wp.logger.InfoLog("All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		wp.logger.WarnLog("Worker pool stop timeout, forcing shutdown")
	}
	
	// 關閉停止通道
	close(wp.stopChan)
	
	// 關閉結果通道
	close(wp.resultChan)
	
	wp.logger.InfoLog("Worker pool stopped")
	return nil
}

// SubmitTask 提交任務
func (wp *DefaultWorkerPool) SubmitTask(task WorkerTask) error {
	if atomic.LoadInt32(&wp.isRunning) == 0 {
		return fmt.Errorf("worker pool is not running")
	}
	
	// 生成任務 ID
	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%d_%d", time.Now().Unix(), atomic.AddInt64(&wp.totalTasks, 1))
	}
	
	if task.Timestamp.IsZero() {
		task.Timestamp = time.Now()
	}
	
	select {
	case wp.taskChan <- task:
		atomic.AddInt64(&wp.totalTasks, 1)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("task submission timeout: worker pool queue is full")
	}
}

// GetStats 獲取工作者池統計信息
func (wp *DefaultWorkerPool) GetStats() WorkerPoolStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	var activeWorkers, idleWorkers int
	var totalProcessed, totalSucceeded, totalFailed int64
	var totalProcessingTime time.Duration
	var lastActivity time.Time
	
	// 統計工作者狀態
	for _, worker := range wp.workers {
		if atomic.LoadInt32(&worker.isRunning) == 1 {
			activeWorkers++
			
			processed := atomic.LoadInt64(&worker.tasksProcessed)
			succeeded := atomic.LoadInt64(&worker.tasksSucceeded)
			failed := atomic.LoadInt64(&worker.tasksFailed)
			
			totalProcessed += processed
			totalSucceeded += succeeded
			totalFailed += failed
			
			if worker.lastTaskTime.After(lastActivity) {
				lastActivity = worker.lastTaskTime
			}
			
			// 計算處理時間
			totalTime := time.Duration(atomic.LoadInt64(&worker.totalTime))
			totalProcessingTime += totalTime
		} else {
			idleWorkers++
		}
	}
	
	// 計算平均延遲
	var averageLatency time.Duration
	if totalProcessed > 0 {
		averageLatency = totalProcessingTime / time.Duration(totalProcessed)
	}
	
	// 計算吞吐量
	var throughput float64
	if !wp.startTime.IsZero() {
		elapsedSeconds := time.Since(wp.startTime).Seconds()
		if elapsedSeconds > 0 {
			throughput = float64(totalProcessed) / elapsedSeconds
		}
	}
	
	// 計算池利用率
	poolUtilization := float64(activeWorkers) / float64(len(wp.workers))
	
	return WorkerPoolStats{
		TotalWorkers:     len(wp.workers),
		ActiveWorkers:    activeWorkers,
		IdleWorkers:      idleWorkers,
		QueuedTasks:      len(wp.taskChan),
		ProcessedTasks:   totalProcessed,
		SuccessfulTasks:  totalSucceeded,
		FailedTasks:      totalFailed,
		AverageLatency:   averageLatency,
		Throughput:       throughput,
		LastActivityTime: lastActivity,
		PoolUtilization:  poolUtilization,
	}
}

// GetWorkerStats 獲取所有工作者統計信息
func (wp *DefaultWorkerPool) GetWorkerStats() []WorkerStats {
	wp.mu.RLock()
	defer wp.mu.RUnlock()
	
	stats := make([]WorkerStats, 0, len(wp.workers))
	
	for _, worker := range wp.workers {
		tasksProcessed := atomic.LoadInt64(&worker.tasksProcessed)
		tasksSucceeded := atomic.LoadInt64(&worker.tasksSucceeded)
		tasksFailed := atomic.LoadInt64(&worker.tasksFailed)
		totalTime := time.Duration(atomic.LoadInt64(&worker.totalTime))
		
		var successRate float64
		if tasksProcessed > 0 {
			successRate = float64(tasksSucceeded) / float64(tasksProcessed)
		}
		
		var averageTime time.Duration
		if tasksProcessed > 0 {
			averageTime = totalTime / time.Duration(tasksProcessed)
		}
		
		stats = append(stats, WorkerStats{
			WorkerID:         worker.ID,
			IsRunning:        atomic.LoadInt32(&worker.isRunning) == 1,
			TasksProcessed:   tasksProcessed,
			TasksSucceeded:   tasksSucceeded,
			TasksFailed:      tasksFailed,
			SuccessRate:      successRate,
			AverageTime:      averageTime,
			LastTaskTime:     worker.lastTaskTime,
			StartTime:        worker.startTime,
			Uptime:           time.Since(worker.startTime),
			RecoveryAttempts: worker.recoveryAttempts,
		})
	}
	
	return stats
}

// IsHealthy 檢查工作者池健康狀態
func (wp *DefaultWorkerPool) IsHealthy() bool {
	if atomic.LoadInt32(&wp.isRunning) == 0 {
		return false
	}
	
	stats := wp.GetStats()
	
	// 健康檢查標準：
	// 1. 至少有一半的工作者在運行
	// 2. 任務隊列不能滿載太久
	// 3. 最近有活動
	
	healthyWorkerRatio := float64(stats.ActiveWorkers) / float64(stats.TotalWorkers)
	queueUtilization := float64(stats.QueuedTasks) / float64(wp.bufferSize)
	
	return healthyWorkerRatio >= 0.5 && 
	       queueUtilization < 0.9 &&
	       time.Since(stats.LastActivityTime) < 5*time.Minute
}

// ResizePool 動態調整工作者數量
func (wp *DefaultWorkerPool) ResizePool(newSize int) error {
	if atomic.LoadInt32(&wp.isRunning) == 0 {
		return fmt.Errorf("worker pool is not running")
	}
	
	if newSize <= 0 {
		return fmt.Errorf("invalid worker count: %d", newSize)
	}
	
	wp.mu.Lock()
	defer wp.mu.Unlock()
	
	currentSize := len(wp.workers)
	
	wp.logger.InfoLog(
		"Resizing worker pool",
		wp.logger.Int("current_size", currentSize),
		wp.logger.Int("new_size", newSize),
	)
	
	if newSize > currentSize {
		// 增加工作者
		for i := currentSize; i < newSize; i++ {
			worker := &Worker{
				ID:                  i + 1,
				taskChan:            wp.taskChan,
				resultChan:          wp.resultChan,
				stopChan:            wp.stopChan,
				kdsService:          wp.kdsService,
				logger:              wp.logger,
				enablePanicRecovery: wp.enablePanicRecovery,
				maxRecoveryAttempts: wp.maxRecoveryAttempts,
				startTime:           time.Now(),
			}
			
			wp.workers = append(wp.workers, worker)
			wp.workerWg.Add(1)
			
			go wp.runWorker(context.Background(), worker)
		}
	} else if newSize < currentSize {
		// 減少工作者（優雅停止多餘的工作者）
		// 這裡簡化實現，實際上可能需要更複雜的邏輯
		wp.logger.WarnLog("Worker pool downsizing not fully implemented in this version")
	}
	
	wp.workerCount = newSize
	
	wp.logger.InfoLog(
		"Worker pool resized",
		wp.logger.Int("new_worker_count", len(wp.workers)),
	)
	
	return nil
}

// 私有方法

// runWorker 運行單個工作者
func (wp *DefaultWorkerPool) runWorker(ctx context.Context, worker *Worker) {
	defer wp.workerWg.Done()
	
	atomic.StoreInt32(&worker.isRunning, 1)
	
	wp.logger.InfoWithContext(
		ctx,
		"Starting worker",
		wp.logger.Int("worker_id", worker.ID),
	)
	
	defer func() {
		if worker.enablePanicRecovery {
			if r := recover(); r != nil {
				worker.recoveryAttempts++
				
				wp.logger.ErrorWithContext(
					ctx,
					"Worker panic recovered",
					wp.logger.Int("worker_id", worker.ID),
					wp.logger.Int("recovery_attempts", worker.recoveryAttempts),
					wp.logger.Any("panic", r),
					wp.logger.String("stack", string(debug.Stack())),
				)
				
				// 如果恢復次數未超過限制，重啟工作者
				if worker.recoveryAttempts < worker.maxRecoveryAttempts {
					wp.logger.InfoWithContext(
						ctx,
						"Restarting worker after panic recovery",
						wp.logger.Int("worker_id", worker.ID),
					)
					
					// 重置狀態並重啟
					atomic.StoreInt32(&worker.isRunning, 0)
					time.Sleep(time.Second) // 短暫延遲
					wp.workerWg.Add(1)
					go wp.runWorker(ctx, worker)
					return
				} else {
					wp.logger.ErrorWithContext(
						ctx,
						"Worker exceeded maximum recovery attempts",
						wp.logger.Int("worker_id", worker.ID),
						wp.logger.Int("max_attempts", worker.maxRecoveryAttempts),
					)
				}
			}
		}
		
		atomic.StoreInt32(&worker.isRunning, 0)
		wp.logger.InfoWithContext(
			ctx,
			"Worker stopped",
			wp.logger.Int("worker_id", worker.ID),
		)
	}()
	
	// 工作者主循環
	for {
		select {
		case task, ok := <-worker.taskChan:
			if !ok {
				// 任務通道已關閉，工作者退出
				return
			}
			
			// 處理任務
			result := wp.processTask(ctx, worker, task)
			
			// 發送結果
			select {
			case worker.resultChan <- result:
			case <-wp.stopChan:
				return
			}
			
		case <-wp.stopChan:
			return
		}
	}
}

// processTask 處理單個任務
func (wp *DefaultWorkerPool) processTask(ctx context.Context, worker *Worker, task WorkerTask) WorkerResult {
	startTime := time.Now()
	
	// 創建任務處理的 tracing context
	taskCtx := task.Context
	if taskCtx == nil {
		taskCtx = ctx
	}
	
	taskCtx, taskSpan := tracing.StartSpan(taskCtx, "WorkerPool.ProcessTask")
	defer tracing.SpanEnd(taskSpan)
	
	// 記錄任務處理屬性
	tracing.RecordSpanAttributes(taskSpan,
		attribute.String("task.id", task.ID),
		attribute.String("task.shard_id", task.ShardID),
		attribute.Int("task.priority", task.Priority),
		attribute.Int("task.retry_count", task.RetryCount),
		attribute.Int("worker.id", worker.ID),
	)
	
	result := WorkerResult{
		TaskID:    task.ID,
		WorkerID:  worker.ID,
		Timestamp: startTime,
	}
	
	wp.logger.DebugWithContext(
		taskCtx,
		"Worker processing task",
		wp.logger.String("task_id", task.ID),
		wp.logger.Int("worker_id", worker.ID),
		wp.logger.String("shard_id", task.ShardID),
		wp.logger.Int("retry_count", task.RetryCount),
	)
	
	// 處理 KDS 記錄
	batchProcessor := wp.getBatchProcessorForWorker()
	if batchProcessor != nil {
		// 使用批次處理器處理單個記錄
		batch := &RecordBatch{
			Records:     []types.Record{task.Record},
			ShardID:     task.ShardID,
			ProcessingTime: startTime,
			ctx:         taskCtx,
		}
		
		results := batchProcessor.ProcessBatch(taskCtx, batch)
		if len(results) > 0 {
			result.Result = results[0]
			result.Success = results[0].Success
			result.Error = results[0].Error
		}
	} else {
		// 直接使用 KDS 服務處理
		result.Error = fmt.Errorf("batch processor not available")
		result.Success = false
	}
	
	// 計算處理時間
	result.ProcessingTime = time.Since(startTime)
	
	// 更新工作者統計
	atomic.AddInt64(&worker.tasksProcessed, 1)
	atomic.AddInt64(&worker.totalTime, int64(result.ProcessingTime))
	worker.lastTaskTime = time.Now()
	
	if result.Success {
		atomic.AddInt64(&worker.tasksSucceeded, 1)
	} else {
		atomic.AddInt64(&worker.tasksFailed, 1)
	}
	
	// 記錄處理結果到 tracing
	tracing.RecordSpanAttributes(taskSpan,
		attribute.Bool("task.success", result.Success),
		attribute.String("task.processing_time", result.ProcessingTime.String()),
	)
	
	if result.Error != nil {
		tracing.RecordSpanError(taskSpan, result.Error)
	}
	
	wp.logger.DebugWithContext(
		taskCtx,
		"Worker completed task",
		wp.logger.String("task_id", task.ID),
		wp.logger.Int("worker_id", worker.ID),
		wp.logger.Bool("success", result.Success),
		wp.logger.String("processing_time", result.ProcessingTime.String()),
	)
	
	return result
}

// processResults 處理工作者結果
func (wp *DefaultWorkerPool) processResults(ctx context.Context) {
	for {
		select {
		case result, ok := <-wp.resultChan:
			if !ok {
				return
			}
			
			// 更新池級統計
			atomic.AddInt64(&wp.completedTasks, 1)
			if !result.Success {
				atomic.AddInt64(&wp.failedTasks, 1)
			}
			
			// 可以在這裡添加結果後處理邏輯，比如重試失敗的任務
			if !result.Success && result.Result.RetryCount < 3 {
				// 簡單的重試邏輯示例
				wp.logger.InfoWithContext(
					ctx,
					"Considering task retry",
					wp.logger.String("task_id", result.TaskID),
					wp.logger.Int("retry_count", result.Result.RetryCount),
				)
			}
			
		case <-wp.stopChan:
			return
		}
	}
}

// getBatchProcessorForWorker 為工作者獲取批次處理器（簡化實現）
func (wp *DefaultWorkerPool) getBatchProcessorForWorker() BatchProcessor {
	// 這裡簡化實現，實際上可能需要注入或創建專用的批次處理器
	// 暫時返回 nil，讓上層調用直接處理
	return nil
}