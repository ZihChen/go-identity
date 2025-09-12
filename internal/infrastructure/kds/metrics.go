package kds

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ConsumerMetrics Consumer 效能指標
type ConsumerMetrics struct {
	// 吞吐量指標
	RecordsProcessedTotal int64   `json:"records_processed_total"` // 總處理記錄數
	RecordsProcessedRate  float64 `json:"records_processed_rate"`  // 每秒處理記錄數
	BatchesProcessedTotal int64   `json:"batches_processed_total"` // 總處理批次數
	BatchesProcessedRate  float64 `json:"batches_processed_rate"`  // 每秒處理批次數

	// 效能指標
	ProcessingLatency   time.Duration `json:"processing_latency"`    // 處理延遲
	BatchProcessingTime time.Duration `json:"batch_processing_time"` // 批次處理時間
	WorkerUtilization   float64       `json:"worker_utilization"`    // Worker 利用率
	AvgBatchSize        float64       `json:"avg_batch_size"`        // 平均批次大小

	// 錯誤指標
	ErrorRate          float64 `json:"error_rate"`           // 錯誤率
	PanicRecoveryCount int64   `json:"panic_recovery_count"` // Panic 恢復次數
	RetryAttempts      int64   `json:"retry_attempts"`       // 重試次數
	FailedRecords      int64   `json:"failed_records"`       // 失敗記錄數

	// 資源指標
	ActiveWorkers  int32 `json:"active_workers"`  // 活躍 Worker 數
	QueuedRecords  int32 `json:"queued_records"`  // 佇列中記錄數
	MemoryUsage    int64 `json:"memory_usage"`    // 記憶體使用量
	GoroutineCount int32 `json:"goroutine_count"` // Goroutine 數量

	// 分片指標
	ActiveShards          int32         `json:"active_shards"`           // 活躍分片數
	ShardsWithLock        int32         `json:"shards_with_lock"`        // 持有鎖的分片數
	ShardProcessingTime   time.Duration `json:"shard_processing_time"`   // 分片處理時間
	CheckpointUpdateCount int64         `json:"checkpoint_update_count"` // Checkpoint 更新次數

	// 時間相關
	StartTime      time.Time     `json:"start_time"`       // 啟動時間
	LastUpdateTime time.Time     `json:"last_update_time"` // 最後更新時間
	UptimeDuration time.Duration `json:"uptime_duration"`  // 運行時間
}

// MetricsCollector 指標收集器
type MetricsCollector struct {
	metrics        *ConsumerMetrics
	startTime      time.Time
	lastResetTime  time.Time
	updateInterval time.Duration
	mu             sync.RWMutex

	// 內部計數器
	recordsCounter    int64
	batchesCounter    int64
	errorsCounter     int64
	retriesCounter    int64
	panicCounter      int64
	checkpointCounter int64

	// 時間窗口統計
	timeWindow       time.Duration
	recordTimestamps []time.Time
	batchTimestamps  []time.Time
	errorTimestamps  []time.Time

	// 延遲統計
	latencySum     int64
	latencyCount   int64
	batchTimeSum   int64
	batchTimeCount int64
	shardTimeSum   int64
	shardTimeCount int64

	// 實時狀態
	activeWorkers  int32
	queuedRecords  int32
	activeShards   int32
	shardsWithLock int32
}

// NewMetricsCollector 創建指標收集器
func NewMetricsCollector(updateInterval time.Duration) *MetricsCollector {
	now := time.Now()
	return &MetricsCollector{
		metrics: &ConsumerMetrics{
			StartTime:      now,
			LastUpdateTime: now,
		},
		startTime:      now,
		lastResetTime:  now,
		updateInterval: updateInterval,
		timeWindow:     5 * time.Minute, // 5分鐘時間窗口
	}
}

// RecordProcessed 記錄處理的記錄
func (c *MetricsCollector) RecordProcessed(count int64, processingTime time.Duration) {
	atomic.AddInt64(&c.recordsCounter, count)
	atomic.AddInt64(&c.latencySum, int64(processingTime))
	atomic.AddInt64(&c.latencyCount, count)

	c.mu.Lock()
	now := time.Now()
	c.recordTimestamps = append(c.recordTimestamps, now)
	c.cleanOldTimestamps(&c.recordTimestamps, now)
	c.mu.Unlock()
}

// BatchProcessed 記錄處理的批次
func (c *MetricsCollector) BatchProcessed(batchSize int, processingTime time.Duration) {
	atomic.AddInt64(&c.batchesCounter, 1)
	atomic.AddInt64(&c.batchTimeSum, int64(processingTime))
	atomic.AddInt64(&c.batchTimeCount, 1)

	c.mu.Lock()
	now := time.Now()
	c.batchTimestamps = append(c.batchTimestamps, now)
	c.cleanOldTimestamps(&c.batchTimestamps, now)
	c.mu.Unlock()
}

// ErrorOccurred 記錄錯誤
func (c *MetricsCollector) ErrorOccurred() {
	atomic.AddInt64(&c.errorsCounter, 1)

	c.mu.Lock()
	now := time.Now()
	c.errorTimestamps = append(c.errorTimestamps, now)
	c.cleanOldTimestamps(&c.errorTimestamps, now)
	c.mu.Unlock()
}

// RetryAttempted 記錄重試嘗試
func (c *MetricsCollector) RetryAttempted() {
	atomic.AddInt64(&c.retriesCounter, 1)
}

// PanicRecovered 記錄 Panic 恢復
func (c *MetricsCollector) PanicRecovered() {
	atomic.AddInt64(&c.panicCounter, 1)
}

// CheckpointUpdated 記錄 Checkpoint 更新
func (c *MetricsCollector) CheckpointUpdated() {
	atomic.AddInt64(&c.checkpointCounter, 1)
}

// ShardProcessed 記錄分片處理時間
func (c *MetricsCollector) ShardProcessed(processingTime time.Duration) {
	atomic.AddInt64(&c.shardTimeSum, int64(processingTime))
	atomic.AddInt64(&c.shardTimeCount, 1)
}

// SetActiveWorkers 設置活躍 Worker 數
func (c *MetricsCollector) SetActiveWorkers(count int32) {
	atomic.StoreInt32(&c.activeWorkers, count)
}

// SetQueuedRecords 設置佇列中記錄數
func (c *MetricsCollector) SetQueuedRecords(count int32) {
	atomic.StoreInt32(&c.queuedRecords, count)
}

// SetActiveShards 設置活躍分片數
func (c *MetricsCollector) SetActiveShards(count int32) {
	atomic.StoreInt32(&c.activeShards, count)
}

// SetShardsWithLock 設置持有鎖的分片數
func (c *MetricsCollector) SetShardsWithLock(count int32) {
	atomic.StoreInt32(&c.shardsWithLock, count)
}

// cleanOldTimestamps 清理過期的時間戳
func (c *MetricsCollector) cleanOldTimestamps(timestamps *[]time.Time, now time.Time) {
	cutoff := now.Add(-c.timeWindow)
	i := 0
	for _, t := range *timestamps {
		if t.After(cutoff) {
			(*timestamps)[i] = t
			i++
		}
	}
	*timestamps = (*timestamps)[:i]
}

// UpdateMetrics 更新指標統計
func (c *MetricsCollector) UpdateMetrics() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	timeSinceStart := now.Sub(c.startTime)

	// 更新基本計數器
	c.metrics.RecordsProcessedTotal = atomic.LoadInt64(&c.recordsCounter)
	c.metrics.BatchesProcessedTotal = atomic.LoadInt64(&c.batchesCounter)
	c.metrics.PanicRecoveryCount = atomic.LoadInt64(&c.panicCounter)
	c.metrics.RetryAttempts = atomic.LoadInt64(&c.retriesCounter)
	c.metrics.CheckpointUpdateCount = atomic.LoadInt64(&c.checkpointCounter)
	c.metrics.FailedRecords = atomic.LoadInt64(&c.errorsCounter)

	// 計算速率（基於時間窗口）
	if len(c.recordTimestamps) > 0 {
		c.metrics.RecordsProcessedRate = float64(len(c.recordTimestamps)) / c.timeWindow.Seconds()
	}
	if len(c.batchTimestamps) > 0 {
		c.metrics.BatchesProcessedRate = float64(len(c.batchTimestamps)) / c.timeWindow.Seconds()
	}

	// 計算錯誤率
	totalOperations := c.metrics.RecordsProcessedTotal
	if totalOperations > 0 {
		c.metrics.ErrorRate = float64(c.metrics.FailedRecords) / float64(totalOperations) * 100
	}

	// 計算平均延遲
	latencyCount := atomic.LoadInt64(&c.latencyCount)
	if latencyCount > 0 {
		latencySum := atomic.LoadInt64(&c.latencySum)
		c.metrics.ProcessingLatency = time.Duration(latencySum / latencyCount)
	}

	// 計算平均批次處理時間
	batchTimeCount := atomic.LoadInt64(&c.batchTimeCount)
	if batchTimeCount > 0 {
		batchTimeSum := atomic.LoadInt64(&c.batchTimeSum)
		c.metrics.BatchProcessingTime = time.Duration(batchTimeSum / batchTimeCount)
	}

	// 計算平均分片處理時間
	shardTimeCount := atomic.LoadInt64(&c.shardTimeCount)
	if shardTimeCount > 0 {
		shardTimeSum := atomic.LoadInt64(&c.shardTimeSum)
		c.metrics.ShardProcessingTime = time.Duration(shardTimeSum / shardTimeCount)
	}

	// 計算平均批次大小
	if c.metrics.BatchesProcessedTotal > 0 {
		c.metrics.AvgBatchSize = float64(
			c.metrics.RecordsProcessedTotal,
		) / float64(
			c.metrics.BatchesProcessedTotal,
		)
	}

	// 更新實時狀態
	c.metrics.ActiveWorkers = atomic.LoadInt32(&c.activeWorkers)
	c.metrics.QueuedRecords = atomic.LoadInt32(&c.queuedRecords)
	c.metrics.ActiveShards = atomic.LoadInt32(&c.activeShards)
	c.metrics.ShardsWithLock = atomic.LoadInt32(&c.shardsWithLock)

	// 計算 Worker 利用率（假設最大 Worker 數已知）
	if c.metrics.ActiveWorkers > 0 {
		// 這裡需要知道最大 Worker 數，可以從配置中獲取
		// 暫時使用活躍 Worker 數作為基準
		c.metrics.WorkerUtilization = float64(
			c.metrics.ActiveWorkers,
		) / float64(
			c.metrics.ActiveWorkers,
		) * 100
	}

	// 更新時間相關指標
	c.metrics.LastUpdateTime = now
	c.metrics.UptimeDuration = timeSinceStart
}

// GetMetrics 獲取當前指標
func (c *MetricsCollector) GetMetrics() *ConsumerMetrics {
	c.UpdateMetrics()

	c.mu.RLock()
	defer c.mu.RUnlock()

	// 創建副本以避免併發問題
	metricsCopy := *c.metrics
	return &metricsCopy
}

// ResetCounters 重置計數器
func (c *MetricsCollector) ResetCounters() {
	atomic.StoreInt64(&c.recordsCounter, 0)
	atomic.StoreInt64(&c.batchesCounter, 0)
	atomic.StoreInt64(&c.errorsCounter, 0)
	atomic.StoreInt64(&c.retriesCounter, 0)
	atomic.StoreInt64(&c.panicCounter, 0)
	atomic.StoreInt64(&c.checkpointCounter, 0)
	atomic.StoreInt64(&c.latencySum, 0)
	atomic.StoreInt64(&c.latencyCount, 0)
	atomic.StoreInt64(&c.batchTimeSum, 0)
	atomic.StoreInt64(&c.batchTimeCount, 0)
	atomic.StoreInt64(&c.shardTimeSum, 0)
	atomic.StoreInt64(&c.shardTimeCount, 0)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastResetTime = time.Now()
	c.recordTimestamps = nil
	c.batchTimestamps = nil
	c.errorTimestamps = nil
}

// StartPeriodicUpdate 啟動定期更新
func (c *MetricsCollector) StartPeriodicUpdate(ctx context.Context) {
	ticker := time.NewTicker(c.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.UpdateMetrics()
		}
	}
}

// GetSummaryString 獲取指標摘要字符串
func (c *MetricsCollector) GetSummaryString() string {
	metrics := c.GetMetrics()

	return fmt.Sprintf(
		"Consumer Metrics Summary:\n"+
			"  Records: %d total, %.2f/sec\n"+
			"  Batches: %d total, %.2f/sec, avg size: %.1f\n"+
			"  Latency: %v avg processing, %v avg batch\n"+
			"  Errors: %d failed records, %.2f%% error rate, %d retries\n"+
			"  Workers: %d active, %d queued records\n"+
			"  Shards: %d active, %d with locks\n"+
			"  Uptime: %v, Last Update: %v",
		metrics.RecordsProcessedTotal, metrics.RecordsProcessedRate,
		metrics.BatchesProcessedTotal, metrics.BatchesProcessedRate, metrics.AvgBatchSize,
		metrics.ProcessingLatency, metrics.BatchProcessingTime,
		metrics.FailedRecords, metrics.ErrorRate, metrics.RetryAttempts,
		metrics.ActiveWorkers, metrics.QueuedRecords,
		metrics.ActiveShards, metrics.ShardsWithLock,
		metrics.UptimeDuration, metrics.LastUpdateTime.Format(time.RFC3339),
	)
}

// HealthStatus 健康狀態
type HealthStatus struct {
	Healthy       bool              `json:"healthy"`
	Status        string            `json:"status"`
	LastCheckTime time.Time         `json:"last_check_time"`
	Issues        []string          `json:"issues,omitempty"`
	Details       map[string]string `json:"details,omitempty"`
}

// HealthChecker 健康檢查器
type HealthChecker struct {
	metricsCollector *MetricsCollector
	checkInterval    time.Duration
	thresholds       HealthThresholds
	mu               sync.RWMutex
	lastStatus       *HealthStatus
}

// HealthThresholds 健康檢查閾值
type HealthThresholds struct {
	MaxErrorRate         float64       // 最大錯誤率 (%)
	MaxProcessingLatency time.Duration // 最大處理延遲
	MinWorkerUtilization float64       // 最小 Worker 利用率 (%)
	MaxQueuedRecords     int32         // 最大佇列記錄數
}

// NewHealthChecker 創建健康檢查器
func NewHealthChecker(
	metricsCollector *MetricsCollector,
	checkInterval time.Duration,
) *HealthChecker {
	return &HealthChecker{
		metricsCollector: metricsCollector,
		checkInterval:    checkInterval,
		thresholds: HealthThresholds{
			MaxErrorRate:         5.0, // 5% 錯誤率
			MaxProcessingLatency: 10 * time.Second,
			MinWorkerUtilization: 10.0, // 10% 最小利用率
			MaxQueuedRecords:     10000,
		},
	}
}

// CheckHealth 執行健康檢查
func (h *HealthChecker) CheckHealth() *HealthStatus {
	metrics := h.metricsCollector.GetMetrics()
	issues := make([]string, 0)
	details := make(map[string]string)

	// 檢查錯誤率
	if metrics.ErrorRate > h.thresholds.MaxErrorRate {
		issues = append(issues, fmt.Sprintf("High error rate: %.2f%%", metrics.ErrorRate))
	}
	details["error_rate"] = fmt.Sprintf("%.2f%%", metrics.ErrorRate)

	// 檢查處理延遲
	if metrics.ProcessingLatency > h.thresholds.MaxProcessingLatency {
		issues = append(
			issues,
			fmt.Sprintf("High processing latency: %v", metrics.ProcessingLatency),
		)
	}
	details["processing_latency"] = metrics.ProcessingLatency.String()

	// 檢查 Worker 利用率
	if metrics.WorkerUtilization < h.thresholds.MinWorkerUtilization {
		issues = append(
			issues,
			fmt.Sprintf("Low worker utilization: %.2f%%", metrics.WorkerUtilization),
		)
	}
	details["worker_utilization"] = fmt.Sprintf("%.2f%%", metrics.WorkerUtilization)

	// 檢查佇列長度
	if metrics.QueuedRecords > h.thresholds.MaxQueuedRecords {
		issues = append(issues, fmt.Sprintf("High queue length: %d", metrics.QueuedRecords))
	}
	details["queued_records"] = fmt.Sprintf("%d", metrics.QueuedRecords)

	// 檢查活躍 Worker
	if metrics.ActiveWorkers == 0 {
		issues = append(issues, "No active workers")
	}
	details["active_workers"] = fmt.Sprintf("%d", metrics.ActiveWorkers)

	// 確定整體健康狀態
	healthy := len(issues) == 0
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	healthStatus := &HealthStatus{
		Healthy:       healthy,
		Status:        status,
		LastCheckTime: time.Now(),
		Issues:        issues,
		Details:       details,
	}

	h.mu.Lock()
	h.lastStatus = healthStatus
	h.mu.Unlock()

	return healthStatus
}

// GetLastHealthStatus 獲取最後的健康狀態
func (h *HealthChecker) GetLastHealthStatus() *HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.lastStatus == nil {
		return h.CheckHealth()
	}

	// 創建副本
	status := *h.lastStatus
	return &status
}

// StartPeriodicHealthCheck 啟動定期健康檢查
func (h *HealthChecker) StartPeriodicHealthCheck(ctx context.Context) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.CheckHealth()
		}
	}
}
