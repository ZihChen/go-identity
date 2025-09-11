package kds

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

// EnhancedConsumer 增強版消費者，整合所有新功能
type EnhancedConsumer struct {
	// 基本組件
	kdsService      *KDSService            // KDS 服務
	logger          infrastructure.Logger          // 日誌記錄器
	config          *config.ConsumerConfig // 消費者配置
	
	// 新功能組件
	batchProcessor  BatchProcessor         // 批次處理器
	workerPool      WorkerPool             // 工作者池
	shardManager    ShardManager           // 分片管理器
	metrics         *MetricsCollector      // 指標收集器
	backoffStrategy BackoffStrategy        // 退避策略
	panicRecovery   *PanicRecovery         // Panic 恢復
	
	// 運行狀態
	isRunning       int32              // 運行狀態標記
	stopChan        chan struct{}      // 停止通道
	
	// 同步
	mu sync.RWMutex // 讀寫鎖
}

// EnhancedConsumerOptions 增強消費者選項
type EnhancedConsumerOptions struct {
	KDSService      *KDSService
	Logger          infrastructure.Logger
	Config          *config.ConsumerConfig
	Metrics         *MetricsCollector
	BackoffStrategy BackoffStrategy
}

// NewEnhancedConsumer 創建增強版消費者
func NewEnhancedConsumer(opts EnhancedConsumerOptions) (*EnhancedConsumer, error) {
	if opts.KDSService == nil {
		return nil, fmt.Errorf("KDSService is required")
	}
	if opts.Logger == nil {
		return nil, fmt.Errorf("Logger is required")
	}
	if opts.Config == nil {
		return nil, fmt.Errorf("ConsumerConfig is required")
	}
	
	// 創建默認組件（如果未提供）
	if opts.Metrics == nil {
		opts.Metrics = NewMetricsCollector(opts.Config.MetricsInterval)
	}
	
	if opts.BackoffStrategy == nil {
		opts.BackoffStrategy = NewAdaptiveBackoffStrategy(
			opts.Config.MinBackoff,
			opts.Config.MaxBackoff,
			opts.Config.BackoffMultiplier,
		)
	}
	
	consumer := &EnhancedConsumer{
		kdsService:      opts.KDSService,
		logger:          opts.Logger,
		config:          opts.Config,
		metrics:         opts.Metrics,
		backoffStrategy: opts.BackoffStrategy,
		stopChan:        make(chan struct{}),
	}
	
	// 初始化 Panic 恢復
	if opts.Config.EnablePanicRecovery {
		consumer.panicRecovery = NewPanicRecovery(
			opts.Config.MaxRecoveryAttempts,
			5*time.Second, // 重試間隔
		)
	}
	
	// 初始化工作者池
	consumer.workerPool = NewDefaultWorkerPool(
		opts.Config.WorkerPoolSize,
		opts.Config.WorkerBufferSize,
		opts.KDSService,
		opts.Logger,
		opts.Config.EnablePanicRecovery,
		opts.Config.MaxRecoveryAttempts,
	)
	
	// 初始化批次處理器（使用自適應批次處理器）
	consumer.batchProcessor = NewAdaptiveBatchProcessor(
		opts.Config.BatchSize,                // 初始批次大小
		opts.Config.BatchSize/2,              // 最小批次大小
		opts.Config.BatchSize*2,              // 最大批次大小
		opts.Config.MaxBatchWaitTime,         // 批次等待時間
		100*time.Millisecond,                 // 目標延遲
		30*time.Second,                       // 調整間隔
		opts.KDSService,
		consumer.workerPool,
		opts.Metrics,
		opts.BackoffStrategy,
		opts.Logger,
	)
	
	// 初始化分片管理器
	consumer.shardManager = NewDefaultShardManager(
		opts.KDSService,
		consumer.batchProcessor,
		consumer.workerPool,
		opts.Metrics,
		opts.BackoffStrategy,
		opts.Logger,
		opts.Config.MaxShardConcurrency,
		opts.Config.ShardLockTimeout,
		fmt.Sprintf("consumer-%d", time.Now().Unix()),
	)
	
	return consumer, nil
}

// Start 啟動增強版消費者
func (ec *EnhancedConsumer) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&ec.isRunning, 0, 1) {
		return fmt.Errorf("enhanced consumer is already running")
	}
	
	ec.logger.InfoWithContext(
		ctx,
		"Starting enhanced consumer",
		ec.logger.Int("batch_size", ec.config.BatchSize),
		ec.logger.Int("worker_pool_size", ec.config.WorkerPoolSize),
		ec.logger.Int("max_shard_concurrency", ec.config.MaxShardConcurrency),
		ec.logger.Bool("panic_recovery_enabled", ec.config.EnablePanicRecovery),
	)
	
	// 創建啟動 span
	ctx, startSpan := tracing.StartSpan(ctx, "EnhancedConsumer.Start")
	defer tracing.SpanEnd(startSpan)
	
	// 記錄啟動屬性
	tracing.RecordSpanAttributes(startSpan,
		attribute.Int("consumer.batch_size", ec.config.BatchSize),
		attribute.Int("consumer.worker_pool_size", ec.config.WorkerPoolSize),
		attribute.Int("consumer.max_shard_concurrency", ec.config.MaxShardConcurrency),
		attribute.Bool("consumer.panic_recovery_enabled", ec.config.EnablePanicRecovery),
	)
	
	// 按順序啟動各個組件
	components := []struct {
		name      string
		startFunc func(context.Context) error
	}{
		{"Worker Pool", ec.workerPool.Start},
		{"Batch Processor", ec.batchProcessor.Start},
		{"Shard Manager", ec.shardManager.Initialize},
	}
	
	// 啟動指標收集器
	go ec.metrics.StartPeriodicUpdate(ctx)
	
	for _, component := range components {
		ec.logger.InfoWithContext(
			ctx,
			fmt.Sprintf("Starting %s", component.name),
		)
		
		if err := component.startFunc(ctx); err != nil {
			tracing.RecordSpanError(startSpan, err)
			ec.logger.ErrorWithContext(
				ctx,
				fmt.Sprintf("Failed to start %s", component.name),
				ec.logger.Error("error", err),
			)
			
			// 清理已啟動的組件
			_ = ec.Stop()
			return fmt.Errorf("failed to start %s: %w", component.name, err)
		}
		
		ec.logger.InfoWithContext(
			ctx,
			fmt.Sprintf("%s started successfully", component.name),
		)
	}
	
	// 啟動主消費循環
	go ec.mainConsumptionLoop(ctx)
	
	// 啟動監控和維護任務
	go ec.monitoringLoop(ctx)
	
	ec.logger.InfoWithContext(
		ctx,
		"Enhanced consumer started successfully",
	)
	
	return nil
}

// Stop 停止增強版消費者
func (ec *EnhancedConsumer) Stop() error {
	if !atomic.CompareAndSwapInt32(&ec.isRunning, 1, 0) {
		return fmt.Errorf("enhanced consumer is not running")
	}
	
	ec.logger.InfoLog("Stopping enhanced consumer")
	
	// 發送停止信號
	close(ec.stopChan)
	
	// 按逆序停止各個組件
	components := []struct {
		name     string
		stopFunc func() error
	}{
		{"Shard Manager", func() error { return ec.shardManager.(*DefaultShardManager).Stop() }},
		{"Batch Processor", ec.batchProcessor.Stop},
		{"Worker Pool", ec.workerPool.Stop},
	}
	
	var stopErrors []error
	
	for _, component := range components {
		ec.logger.InfoLog(fmt.Sprintf("Stopping %s", component.name))
		
		if err := component.stopFunc(); err != nil {
			ec.logger.ErrorLog(
				fmt.Sprintf("Failed to stop %s", component.name),
				ec.logger.Error("error", err),
			)
			stopErrors = append(stopErrors, err)
		} else {
			ec.logger.InfoLog(fmt.Sprintf("%s stopped successfully", component.name))
		}
	}
	
	if len(stopErrors) > 0 {
		return fmt.Errorf("errors during shutdown: %v", stopErrors)
	}
	
	ec.logger.InfoLog("Enhanced consumer stopped successfully")
	return nil
}

// IsHealthy 檢查增強版消費者健康狀態
func (ec *EnhancedConsumer) IsHealthy(ctx context.Context) bool {
	if atomic.LoadInt32(&ec.isRunning) == 0 {
		return false
	}
	
	// 檢查各個組件的健康狀態
	checks := []struct {
		name   string
		isHealthy func() bool
	}{
		{"Worker Pool", ec.workerPool.IsHealthy},
		{"Batch Processor", ec.batchProcessor.IsHealthy},
	}
	
	for _, check := range checks {
		if !check.isHealthy() {
			ec.logger.WarnWithContext(
				ctx,
				fmt.Sprintf("Component %s is not healthy", check.name),
			)
			return false
		}
	}
	
	// 檢查分片管理器健康狀態
	if !ec.shardManager.IsHealthy(ctx) {
		ec.logger.WarnWithContext(ctx, "Shard manager is not healthy")
		return false
	}
	
	return true
}

// GetMetrics 獲取增強版消費者指標
func (ec *EnhancedConsumer) GetMetrics() EnhancedConsumerMetrics {
	workerPoolStats := ec.workerPool.GetStats()
	
	var batchMetrics BatchMetrics
	if ec.batchProcessor != nil {
		batchMetrics = ec.batchProcessor.GetMetrics()
	}
	
	shardMetrics, _ := ec.shardManager.GetAllShardMetrics(context.Background())
	
	consumerMetrics := ec.metrics.GetMetrics()
	
	return EnhancedConsumerMetrics{
		ConsumerMetrics:  *consumerMetrics,
		WorkerPoolStats:  workerPoolStats,
		BatchMetrics:     batchMetrics,
		ShardMetrics:     shardMetrics,
		IsHealthy:        ec.IsHealthy(context.Background()),
		StartTime:        time.Now(), // 實際應該記錄真實啟動時間
	}
}

// EnhancedConsumerMetrics 增強版消費者指標
type EnhancedConsumerMetrics struct {
	ConsumerMetrics ConsumerMetrics   `json:"consumer_metrics"`  // 基本消費者指標
	WorkerPoolStats WorkerPoolStats   `json:"worker_pool_stats"` // 工作者池統計
	BatchMetrics    BatchMetrics      `json:"batch_metrics"`     // 批次處理指標
	ShardMetrics    []ShardMetrics    `json:"shard_metrics"`     // 分片指標
	IsHealthy       bool              `json:"is_healthy"`        // 整體健康狀態
	StartTime       time.Time         `json:"start_time"`        // 啟動時間
}

// mainConsumptionLoop 主消費循環
func (ec *EnhancedConsumer) mainConsumptionLoop(ctx context.Context) {
	ec.logger.InfoWithContext(ctx, "Starting main consumption loop")
	
	defer func() {
		if ec.panicRecovery != nil {
			_ = ec.panicRecovery.Execute(ctx, "main_consumption_loop_exit", func() error {
				ec.logger.InfoWithContext(ctx, "Main consumption loop exiting")
				return nil
			})
		} else {
			ec.logger.InfoWithContext(ctx, "Main consumption loop exiting")
		}
	}()
	
	// 初始化分片處理
	for {
		select {
		case <-ec.stopChan:
			return
		case <-ctx.Done():
			return
		default:
			// 執行一輪分片發現和處理
			if err := ec.discoverAndProcessShards(ctx); err != nil {
				ec.logger.ErrorWithContext(
					ctx,
					"Error in shard discovery and processing",
					ec.logger.Error("error", err),
				)
				
				// 使用退避策略
				backoffDuration := ec.backoffStrategy.NextBackoff()
				ec.logger.InfoWithContext(
					ctx,
					"Applying backoff after error",
					ec.logger.String("backoff", backoffDuration.String()),
				)
				time.Sleep(backoffDuration)
			} else {
				// 重置退避策略
				ec.backoffStrategy.Reset()
			}
			
			// 短暫休息避免過度輪詢
			time.Sleep(1 * time.Second)
		}
	}
}

// discoverAndProcessShards 發現和處理分片
func (ec *EnhancedConsumer) discoverAndProcessShards(ctx context.Context) error {
	// 獲取所有分片迭代器
	iterators, err := ec.shardManager.GetShardIterators(ctx)
	if err != nil {
		return fmt.Errorf("failed to get shard iterators: %w", err)
	}
	
	// 處理每個分片（但受並發限制）
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, ec.config.MaxShardConcurrency)
	
	for shardID, iterator := range iterators {
		select {
		case <-ec.stopChan:
			return nil
		case <-ctx.Done():
			return nil
		case semaphore <- struct{}{}: // 獲取並發許可
			wg.Add(1)
			go func(shardID, iterator string) {
				defer func() {
					<-semaphore // 釋放並發許可
					wg.Done()
				}()
				
				// 處理單個分片
				if err := ec.shardManager.ProcessShard(ctx, shardID, iterator); err != nil {
					ec.logger.WarnWithContext(
						ctx,
						"Failed to process shard",
						ec.logger.String("shard_id", shardID),
						ec.logger.Error("error", err),
					)
				}
			}(shardID, iterator)
		}
	}
	
	// 等待所有分片處理完成
	wg.Wait()
	
	return nil
}

// monitoringLoop 監控循環
func (ec *EnhancedConsumer) monitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ec.performHealthChecks(ctx)
			ec.reportMetrics(ctx)
		case <-ec.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performHealthChecks 執行健康檢查
func (ec *EnhancedConsumer) performHealthChecks(ctx context.Context) {
	if !ec.IsHealthy(ctx) {
		ec.logger.WarnWithContext(
			ctx,
			"Enhanced consumer health check failed",
		)
		
		// 可以在這裡添加自動修復邏輯
		// 例如重啟不健康的組件
	}
}

// reportMetrics 報告指標
func (ec *EnhancedConsumer) reportMetrics(ctx context.Context) {
	metrics := ec.GetMetrics()
	
	ec.logger.InfoWithContext(
		ctx,
		"Enhanced consumer metrics",
		ec.logger.Int64("total_records_processed", metrics.ConsumerMetrics.RecordsProcessedTotal),
		ec.logger.Float64("records_per_second", metrics.ConsumerMetrics.RecordsProcessedRate),
		ec.logger.Int("active_workers", metrics.WorkerPoolStats.ActiveWorkers),
		ec.logger.Int64("total_batches", metrics.BatchMetrics.TotalBatches),
		ec.logger.Float64("batch_error_rate", metrics.BatchMetrics.ErrorRate),
		ec.logger.Int("healthy_shards", len(metrics.ShardMetrics)),
		ec.logger.Bool("overall_healthy", metrics.IsHealthy),
	)
	
	// 記錄指標到 tracing
	_, metricsSpan := tracing.StartSpan(ctx, "EnhancedConsumer.Metrics")
	tracing.RecordSpanAttributes(metricsSpan,
		attribute.Int64("metrics.total_records_processed", metrics.ConsumerMetrics.RecordsProcessedTotal),
		attribute.Float64("metrics.records_per_second", metrics.ConsumerMetrics.RecordsProcessedRate),
		attribute.Int("metrics.active_workers", metrics.WorkerPoolStats.ActiveWorkers),
		attribute.Int64("metrics.total_batches", metrics.BatchMetrics.TotalBatches),
		attribute.Float64("metrics.batch_error_rate", metrics.BatchMetrics.ErrorRate),
		attribute.Bool("metrics.overall_healthy", metrics.IsHealthy),
	)
	tracing.SpanEnd(metricsSpan)
}

// ConsumeAllEventsEnhanced 使用增強版消費者消費所有事件
func (ec *EnhancedConsumer) ConsumeAllEventsEnhanced(ctx context.Context) error {
	ec.logger.InfoWithContext(
		ctx,
		"Starting enhanced event consumption",
	)
	
	// 啟動增強版消費者
	if err := ec.Start(ctx); err != nil {
		return fmt.Errorf("failed to start enhanced consumer: %w", err)
	}
	
	// 等待停止信號或上下文取消
	<-ctx.Done()
	
	// 停止消費者
	if err := ec.Stop(); err != nil {
		ec.logger.ErrorWithContext(
			ctx,
			"Error stopping enhanced consumer",
			ec.logger.Error("error", err),
		)
		return err
	}
	
	return nil
}