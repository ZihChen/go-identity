package kds

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"go.opentelemetry.io/otel/attribute"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

// AdaptiveBatchProcessor 自適應批次處理器，支援動態調整批次大小
type AdaptiveBatchProcessor struct {
	// 繼承基本批次處理器
	*DefaultBatchProcessor
	
	// 自適應配置
	minBatchSize    int           // 最小批次大小
	maxBatchSize    int           // 最大批次大小
	targetLatency   time.Duration // 目標處理延遲
	adjustInterval  time.Duration // 調整間隔
	
	// 動態統計
	recentLatencies []time.Duration // 最近的處理延遲記錄
	latencyIndex    int            // 延遲記錄索引
	latencyWindow   int            // 延遲統計窗口大小
	
	// 批次大小調整狀態
	currentOptimalSize int32     // 當前最優批次大小
	lastAdjustTime     time.Time // 上次調整時間
	adjustmentCount    int64     // 調整次數
	
	// 效能監控
	throughputHistory    []float64   // 吞吐量歷史記錄
	throughputIndex      int         // 吞吐量記錄索引  
	throughputWindow     int         // 吞吐量統計窗口大小
	lastThroughputCheck  time.Time   // 上次吞吐量檢查時間
	processedInInterval  int64       // 間隔內處理的記錄數
	
	// 同步
	adjustMutex sync.RWMutex // 調整互斥鎖
}

// BatchSizeAdjustment 批次大小調整資訊
type BatchSizeAdjustment struct {
	OldSize           int           `json:"old_size"`            // 舊批次大小
	NewSize           int           `json:"new_size"`            // 新批次大小
	Reason            string        `json:"reason"`              // 調整原因
	AverageLatency    time.Duration `json:"average_latency"`     // 平均延遲
	Throughput        float64       `json:"throughput"`          // 吞吐量
	Timestamp         time.Time     `json:"timestamp"`           // 調整時間
	AdjustmentFactor  float64       `json:"adjustment_factor"`   // 調整係數
}

// AdaptiveMetrics 自適應批次處理指標
type AdaptiveMetrics struct {
	BatchMetrics                     // 繼承基本指標
	
	// 自適應相關指標
	CurrentOptimalSize   int                   `json:"current_optimal_size"`   // 當前最優批次大小
	MinBatchSize         int                   `json:"min_batch_size"`         // 最小批次大小
	MaxBatchSize         int                   `json:"max_batch_size"`         // 最大批次大小
	AverageLatency       time.Duration         `json:"average_latency"`        // 平均延遲
	TargetLatency        time.Duration         `json:"target_latency"`         // 目標延遲
	ThroughputPerSecond  float64               `json:"throughput_per_second"`  // 每秒吞吐量
	AdjustmentCount      int64                 `json:"adjustment_count"`       // 調整次數
	LastAdjustment       *BatchSizeAdjustment  `json:"last_adjustment"`        // 最後一次調整
}

// NewAdaptiveBatchProcessor 創建自適應批次處理器
func NewAdaptiveBatchProcessor(
	initialBatchSize int,
	minBatchSize int,
	maxBatchSize int,
	maxBatchWaitTime time.Duration,
	targetLatency time.Duration,
	adjustInterval time.Duration,
	kdsService *KDSService,
	workerPool WorkerPool,
	metrics *MetricsCollector,
	backoffStrategy BackoffStrategy,
	logger infrastructure.Logger,
) *AdaptiveBatchProcessor {
	
	// 創建基本批次處理器
	baseBatchProcessor := NewDefaultBatchProcessor(
		initialBatchSize,
		maxBatchWaitTime,
		kdsService,
		workerPool,
		metrics,
		backoffStrategy,
		logger,
	)
	
	// 配置參數驗證和調整
	if minBatchSize <= 0 {
		minBatchSize = 10
	}
	if maxBatchSize <= minBatchSize {
		maxBatchSize = minBatchSize * 10
	}
	if initialBatchSize < minBatchSize {
		initialBatchSize = minBatchSize
	}
	if initialBatchSize > maxBatchSize {
		initialBatchSize = maxBatchSize
	}
	if targetLatency <= 0 {
		targetLatency = 100 * time.Millisecond
	}
	if adjustInterval <= 0 {
		adjustInterval = 30 * time.Second
	}
	
	latencyWindow := 50  // 保留最近50次延遲記錄
	throughputWindow := 20 // 保留最近20次吞吐量記錄
	
	abp := &AdaptiveBatchProcessor{
		DefaultBatchProcessor: baseBatchProcessor,
		minBatchSize:          minBatchSize,
		maxBatchSize:          maxBatchSize,
		targetLatency:         targetLatency,
		adjustInterval:        adjustInterval,
		
		recentLatencies:      make([]time.Duration, latencyWindow),
		latencyIndex:         0,
		latencyWindow:        latencyWindow,
		
		currentOptimalSize:   int32(initialBatchSize),
		lastAdjustTime:       time.Now(),
		
		throughputHistory:    make([]float64, throughputWindow),
		throughputIndex:      0,
		throughputWindow:     throughputWindow,
		lastThroughputCheck:  time.Now(),
	}
	
	// 更新基本處理器的批次大小
	abp.DefaultBatchProcessor.batchSize = initialBatchSize
	
	return abp
}

// Start 啟動自適應批次處理器
func (abp *AdaptiveBatchProcessor) Start(ctx context.Context) error {
	// 先啟動基本處理器
	if err := abp.DefaultBatchProcessor.Start(ctx); err != nil {
		return err
	}
	
	abp.logger.InfoWithContext(
		ctx,
		"Starting adaptive batch processor",
		abp.logger.Int("initial_batch_size", int(atomic.LoadInt32(&abp.currentOptimalSize))),
		abp.logger.Int("min_batch_size", abp.minBatchSize),
		abp.logger.Int("max_batch_size", abp.maxBatchSize),
		abp.logger.String("target_latency", abp.targetLatency.String()),
		abp.logger.String("adjust_interval", abp.adjustInterval.String()),
	)
	
	// 啟動自適應調整 goroutine
	go abp.adaptiveSizeAdjustment(ctx)
	
	return nil
}

// ProcessBatch 處理批次（重寫以支援自適應調整）
func (abp *AdaptiveBatchProcessor) ProcessBatch(ctx context.Context, batch *RecordBatch) []ProcessResult {
	// 記錄處理開始時間
	startTime := time.Now()
	
	// 使用基本處理器處理批次
	results := abp.DefaultBatchProcessor.ProcessBatch(ctx, batch)
	
	// 記錄處理延遲
	processingLatency := time.Since(startTime)
	abp.recordLatency(processingLatency)
	
	// 更新吞吐量統計
	abp.updateThroughputStats(len(batch.Records))
	
	abp.logger.DebugWithContext(
		ctx,
		"Batch processed with adaptive monitoring",
		abp.logger.String("shard_id", batch.ShardID),
		abp.logger.Int("record_count", len(batch.Records)),
		abp.logger.String("processing_latency", processingLatency.String()),
		abp.logger.Int("current_optimal_size", int(atomic.LoadInt32(&abp.currentOptimalSize))),
	)
	
	return results
}

// AddRecord 添加記錄（重寫以使用動態批次大小）
func (abp *AdaptiveBatchProcessor) AddRecord(record types.Record, shardID string) error {
	// 使用當前最優批次大小
	currentOptimal := int(atomic.LoadInt32(&abp.currentOptimalSize))
	
	// 臨時更新基本處理器的批次大小（線程安全）
	abp.DefaultBatchProcessor.mu.Lock()
	oldBatchSize := abp.DefaultBatchProcessor.batchSize
	abp.DefaultBatchProcessor.batchSize = currentOptimal
	abp.DefaultBatchProcessor.mu.Unlock()
	
	// 調用基本處理器的 AddRecord
	err := abp.DefaultBatchProcessor.AddRecord(record, shardID)
	
	// 恢復原來的批次大小設定
	abp.DefaultBatchProcessor.mu.Lock()
	abp.DefaultBatchProcessor.batchSize = oldBatchSize
	abp.DefaultBatchProcessor.mu.Unlock()
	
	return err
}

// GetAdaptiveMetrics 獲取自適應批次處理指標
func (abp *AdaptiveBatchProcessor) GetAdaptiveMetrics() AdaptiveMetrics {
	// 獲取基本指標
	baseMetrics := abp.DefaultBatchProcessor.GetMetrics()
	
	// 計算自適應指標
	abp.adjustMutex.RLock()
	defer abp.adjustMutex.RUnlock()
	
	averageLatency := abp.calculateAverageLatency()
	throughput := abp.calculateCurrentThroughput()
	adjustmentCount := atomic.LoadInt64(&abp.adjustmentCount)
	
	return AdaptiveMetrics{
		BatchMetrics:        baseMetrics,
		CurrentOptimalSize:  int(atomic.LoadInt32(&abp.currentOptimalSize)),
		MinBatchSize:        abp.minBatchSize,
		MaxBatchSize:        abp.maxBatchSize,
		AverageLatency:      averageLatency,
		TargetLatency:       abp.targetLatency,
		ThroughputPerSecond: throughput,
		AdjustmentCount:     adjustmentCount,
		LastAdjustment:      nil, // 可以追蹤最後一次調整
	}
}

// 私有方法實現

// adaptiveSizeAdjustment 自適應批次大小調整主循環
func (abp *AdaptiveBatchProcessor) adaptiveSizeAdjustment(ctx context.Context) {
	adjustTicker := time.NewTicker(abp.adjustInterval)
	defer adjustTicker.Stop()
	
	for {
		select {
		case <-adjustTicker.C:
			abp.performSizeAdjustment(ctx)
		case <-abp.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performSizeAdjustment 執行批次大小調整
func (abp *AdaptiveBatchProcessor) performSizeAdjustment(ctx context.Context) {
	abp.adjustMutex.Lock()
	defer abp.adjustMutex.Unlock()
	
	// 計算當前效能指標
	avgLatency := abp.calculateAverageLatency()
	currentThroughput := abp.calculateCurrentThroughput()
	currentSize := int(atomic.LoadInt32(&abp.currentOptimalSize))
	
	// 如果沒有足夠的數據，跳過調整
	if avgLatency == 0 || currentThroughput == 0 {
		abp.logger.DebugWithContext(
			ctx,
			"Insufficient data for batch size adjustment",
			abp.logger.String("avg_latency", avgLatency.String()),
			abp.logger.Float64("throughput", currentThroughput),
		)
		return
	}
	
	// 計算調整係數
	adjustmentFactor := abp.calculateAdjustmentFactor(avgLatency, currentThroughput)
	
	// 計算新的批次大小
	newSize := abp.calculateNewBatchSize(currentSize, adjustmentFactor)
	
	// 檢查是否需要調整
	if newSize != currentSize {
		adjustment := BatchSizeAdjustment{
			OldSize:          currentSize,
			NewSize:          newSize,
			Reason:           abp.getAdjustmentReason(avgLatency, currentThroughput, adjustmentFactor),
			AverageLatency:   avgLatency,
			Throughput:       currentThroughput,
			Timestamp:        time.Now(),
			AdjustmentFactor: adjustmentFactor,
		}
		
		// 應用新的批次大小
		atomic.StoreInt32(&abp.currentOptimalSize, int32(newSize))
		atomic.AddInt64(&abp.adjustmentCount, 1)
		abp.lastAdjustTime = time.Now()
		
		abp.logger.InfoWithContext(
			ctx,
			"Adjusted batch size",
			abp.logger.Int("old_size", adjustment.OldSize),
			abp.logger.Int("new_size", adjustment.NewSize),
			abp.logger.String("reason", adjustment.Reason),
			abp.logger.String("avg_latency", adjustment.AverageLatency.String()),
			abp.logger.Float64("throughput", adjustment.Throughput),
			abp.logger.Float64("adjustment_factor", adjustment.AdjustmentFactor),
		)
		
		// 記錄調整到 tracing
		_, adjustmentSpan := tracing.StartSpan(ctx, "AdaptiveBatchProcessor.SizeAdjustment")
		tracing.RecordSpanAttributes(adjustmentSpan,
			attribute.Int("batch.old_size", adjustment.OldSize),
			attribute.Int("batch.new_size", adjustment.NewSize),
			attribute.String("batch.adjustment_reason", adjustment.Reason),
			attribute.String("batch.avg_latency", adjustment.AverageLatency.String()),
			attribute.Float64("batch.throughput", adjustment.Throughput),
			attribute.Float64("batch.adjustment_factor", adjustment.AdjustmentFactor),
		)
		tracing.SpanEnd(adjustmentSpan)
	}
}

// calculateAdjustmentFactor 計算批次大小調整係數
func (abp *AdaptiveBatchProcessor) calculateAdjustmentFactor(avgLatency time.Duration, throughput float64) float64 {
	// 延遲因子：如果延遲超過目標，則減小批次；否則可能增大
	latencyRatio := float64(avgLatency) / float64(abp.targetLatency)
	
	var latencyFactor float64
	if latencyRatio > 1.2 {
		// 延遲明顯超過目標，需要減小批次
		latencyFactor = 0.8 - math.Min(0.3, (latencyRatio-1.2)*0.2)
	} else if latencyRatio < 0.8 {
		// 延遲明顯低於目標，可以增大批次
		latencyFactor = 1.2 + math.Min(0.3, (0.8-latencyRatio)*0.2)
	} else {
		// 延遲在可接受範圍內
		latencyFactor = 1.0
	}
	
	// 吞吐量因子：基於吞吐量趨勢調整
	throughputFactor := abp.calculateThroughputFactor(throughput)
	
	// 綜合調整係數（加權平均）
	finalFactor := latencyFactor*0.6 + throughputFactor*0.4
	
	// 限制調整幅度，避免劇烈變化
	finalFactor = math.Max(0.5, math.Min(2.0, finalFactor))
	
	return finalFactor
}

// calculateThroughputFactor 計算吞吐量調整因子
func (abp *AdaptiveBatchProcessor) calculateThroughputFactor(currentThroughput float64) float64 {
	// 計算吞吐量趨勢
	recentThroughputs := abp.getRecentThroughputs(5) // 獲取最近5次的吞吐量
	if len(recentThroughputs) < 2 {
		return 1.0
	}
	
	// 計算吞吐量趨勢 (簡單線性趨勢)
	var trend float64
	for i := 1; i < len(recentThroughputs); i++ {
		if recentThroughputs[i-1] > 0 {
			trend += (recentThroughputs[i] - recentThroughputs[i-1]) / recentThroughputs[i-1]
		}
	}
	trend /= float64(len(recentThroughputs) - 1)
	
	// 根據趨勢調整
	if trend > 0.1 {
		// 吞吐量正在增長，可以嘗試增大批次
		return 1.1
	} else if trend < -0.1 {
		// 吞吐量正在下降，減小批次
		return 0.9
	}
	
	return 1.0
}

// calculateNewBatchSize 計算新的批次大小
func (abp *AdaptiveBatchProcessor) calculateNewBatchSize(currentSize int, adjustmentFactor float64) int {
	newSize := int(math.Round(float64(currentSize) * adjustmentFactor))
	
	// 應用邊界限制
	if newSize < abp.minBatchSize {
		newSize = abp.minBatchSize
	} else if newSize > abp.maxBatchSize {
		newSize = abp.maxBatchSize
	}
	
	return newSize
}

// getAdjustmentReason 獲取調整原因的描述
func (abp *AdaptiveBatchProcessor) getAdjustmentReason(avgLatency time.Duration, throughput float64, factor float64) string {
	latencyRatio := float64(avgLatency) / float64(abp.targetLatency)
	
	if factor > 1.1 {
		if latencyRatio < 0.8 {
			return "Low latency allows larger batch size"
		}
		return "Increasing batch size for better throughput"
	} else if factor < 0.9 {
		if latencyRatio > 1.2 {
			return "High latency requires smaller batch size"
		}
		return "Reducing batch size due to performance concerns"
	}
	
	return "Minor adjustment for optimization"
}

// recordLatency 記錄處理延遲
func (abp *AdaptiveBatchProcessor) recordLatency(latency time.Duration) {
	abp.adjustMutex.Lock()
	defer abp.adjustMutex.Unlock()
	
	abp.recentLatencies[abp.latencyIndex] = latency
	abp.latencyIndex = (abp.latencyIndex + 1) % abp.latencyWindow
}

// calculateAverageLatency 計算平均延遲
func (abp *AdaptiveBatchProcessor) calculateAverageLatency() time.Duration {
	var total time.Duration
	var count int
	
	for _, latency := range abp.recentLatencies {
		if latency > 0 {
			total += latency
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	
	return total / time.Duration(count)
}

// updateThroughputStats 更新吞吐量統計
func (abp *AdaptiveBatchProcessor) updateThroughputStats(recordCount int) {
	atomic.AddInt64(&abp.processedInInterval, int64(recordCount))
	
	now := time.Now()
	if now.Sub(abp.lastThroughputCheck) >= time.Second {
		// 計算當前吞吐量 (records per second)
		processed := atomic.SwapInt64(&abp.processedInInterval, 0)
		elapsed := now.Sub(abp.lastThroughputCheck).Seconds()
		throughput := float64(processed) / elapsed
		
		abp.adjustMutex.Lock()
		abp.throughputHistory[abp.throughputIndex] = throughput
		abp.throughputIndex = (abp.throughputIndex + 1) % abp.throughputWindow
		abp.lastThroughputCheck = now
		abp.adjustMutex.Unlock()
	}
}

// calculateCurrentThroughput 計算當前吞吐量
func (abp *AdaptiveBatchProcessor) calculateCurrentThroughput() float64 {
	var total float64
	var count int
	
	for _, throughput := range abp.throughputHistory {
		if throughput > 0 {
			total += throughput
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	
	return total / float64(count)
}

// getRecentThroughputs 獲取最近的吞吐量記錄
func (abp *AdaptiveBatchProcessor) getRecentThroughputs(count int) []float64 {
	if count > abp.throughputWindow {
		count = abp.throughputWindow
	}
	
	result := make([]float64, 0, count)
	for i := 0; i < count; i++ {
		index := (abp.throughputIndex - i - 1 + abp.throughputWindow) % abp.throughputWindow
		if abp.throughputHistory[index] > 0 {
			result = append(result, abp.throughputHistory[index])
		}
	}
	
	return result
}