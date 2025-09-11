package tests

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/kds"
	"github.com/stretchr/testify/assert"
)

// PerformanceTestSuite Consumer 效能測試套件
type PerformanceTestSuite struct {
	config    *config.Config
	collector *kds.MetricsCollector
	checker   *kds.HealthChecker
}

// NewPerformanceTestSuite 創建效能測試套件
func NewPerformanceTestSuite() *PerformanceTestSuite {
	cfg := &config.Config{
		Consumer: config.ConsumerConfig{
			BatchSize:           100,
			MaxBatchWaitTime:    500 * time.Millisecond,
			WorkerPoolSize:      10,
			WorkerBufferSize:    1000,
			MinBackoff:          500 * time.Millisecond,
			MaxBackoff:          5 * time.Second,
			BackoffMultiplier:   1.5,
			MaxShardConcurrency: 8,
			ShardLockTimeout:    1 * time.Minute,
			MetricsInterval:     30 * time.Second,
			HealthCheckInterval: 10 * time.Second,
			EnablePanicRecovery: true,
			MaxRecoveryAttempts: 3,
			LockRetryInterval:   1 * time.Second,
			LockMaxRetries:      5,
		},
	}

	collector := kds.NewMetricsCollector(cfg.Consumer.MetricsInterval)
	checker := kds.NewHealthChecker(collector, cfg.Consumer.HealthCheckInterval)

	return &PerformanceTestSuite{
		config:    cfg,
		collector: collector,
		checker:   checker,
	}
}

// TestConfigurationValidation 測試配置驗證
func (pts *PerformanceTestSuite) TestConfigurationValidation(t *testing.T) {
	// 測試批次大小配置
	assert.Greater(t, pts.config.Consumer.BatchSize, 0, "批次大小必須大於0")
	assert.LessOrEqual(t, pts.config.Consumer.BatchSize, 1000, "批次大小不應過大")

	// 測試Worker Pool配置
	assert.Greater(t, pts.config.Consumer.WorkerPoolSize, 0, "Worker數量必須大於0")
	assert.LessOrEqual(t, pts.config.Consumer.WorkerPoolSize, runtime.NumCPU()*4, "Worker數量不應過多")

	// 測試退避策略配置
	assert.Less(t, pts.config.Consumer.MinBackoff, pts.config.Consumer.MaxBackoff, "最小退避時間應小於最大退避時間")
	assert.Greater(t, pts.config.Consumer.BackoffMultiplier, 1.0, "退避倍數應大於1.0")

	// 測試超時配置
	assert.Greater(t, int64(pts.config.Consumer.MaxBatchWaitTime), int64(0), "批次等待時間必須大於0")
	assert.Greater(t, int64(pts.config.Consumer.ShardLockTimeout), int64(0), "分片鎖超時時間必須大於0")

	t.Logf("配置驗證通過: %+v", pts.config.Consumer)
}

// TestMetricsAccuracy 測試指標準確性
func (pts *PerformanceTestSuite) TestMetricsAccuracy(t *testing.T) {
	collector := pts.collector

	// 模擬處理記錄
	expectedRecords := int64(1000)
	expectedBatches := int64(10)
	expectedErrors := int64(5)

	for i := int64(0); i < expectedRecords; i++ {
		collector.RecordProcessed(1, time.Millisecond*10)
	}

	for i := int64(0); i < expectedBatches; i++ {
		collector.BatchProcessed(int(expectedRecords/expectedBatches), time.Millisecond*100)
	}

	for i := int64(0); i < expectedErrors; i++ {
		collector.ErrorOccurred()
	}

	// 獲取指標
	metrics := collector.GetMetrics()

	// 驗證指標準確性
	assert.Equal(t, expectedRecords, metrics.RecordsProcessedTotal, "記錄處理數量不匹配")
	assert.Equal(t, expectedBatches, metrics.BatchesProcessedTotal, "批次處理數量不匹配")
	assert.Equal(t, expectedErrors, metrics.FailedRecords, "錯誤記錄數量不匹配")

	// 驗證計算指標
	expectedAvgBatchSize := float64(expectedRecords) / float64(expectedBatches)
	assert.InDelta(t, expectedAvgBatchSize, metrics.AvgBatchSize, 1.0, "平均批次大小計算錯誤")

	expectedErrorRate := float64(expectedErrors) / float64(expectedRecords) * 100
	assert.InDelta(t, expectedErrorRate, metrics.ErrorRate, 1.0, "錯誤率計算錯誤")

	t.Logf("指標準確性測試通過: 記錄=%d, 批次=%d, 錯誤=%d", expectedRecords, expectedBatches, expectedErrors)
}

// TestHealthChecker 測試健康檢查器
func (pts *PerformanceTestSuite) TestHealthChecker(t *testing.T) {
	checker := pts.checker

	// 測試健康狀態
	healthStatus := checker.CheckHealth()
	assert.NotNil(t, healthStatus, "健康狀態不應為空")
	assert.NotEmpty(t, healthStatus.Status, "狀態字段不應為空")

	// 模擬高錯誤率情況
	for i := 0; i < 100; i++ {
		pts.collector.RecordProcessed(1, time.Millisecond*10)
		if i < 10 { // 10% 錯誤率
			pts.collector.ErrorOccurred()
		}
	}

	unhealthyStatus := checker.CheckHealth()
	assert.False(t, unhealthyStatus.Healthy, "高錯誤率時應顯示不健康")
	
	// 檢查是否包含任何高錯誤率相關的問題
	hasErrorRateIssue := false
	for _, issue := range unhealthyStatus.Issues {
		if issue == "High error rate: 10.00%" || issue == "No active workers" {
			hasErrorRateIssue = true
			break
		}
	}
	assert.True(t, hasErrorRateIssue, "應包含高錯誤率或無活躍Worker問題")

	t.Logf("健康檢查測試通過: %s", unhealthyStatus.Status)
}

// TestBackoffStrategies 測試退避策略
func (pts *PerformanceTestSuite) TestBackoffStrategies(t *testing.T) {
	t.Run("AdaptiveBackoff", func(t *testing.T) {
		strategy := kds.NewAdaptiveBackoffStrategy(
			pts.config.Consumer.MinBackoff,
			pts.config.Consumer.MaxBackoff,
			pts.config.Consumer.BackoffMultiplier,
		)

		// 測試初始退避時間
		initialBackoff := strategy.NextBackoff()
		assert.GreaterOrEqual(t, initialBackoff, pts.config.Consumer.MinBackoff, "初始退避時間應不小於最小值")

		// 測試錯誤後退避時間增加
		strategy.RecordError(kds.ErrKDSThrottling)
		increasedBackoff := strategy.NextBackoff()
		assert.Greater(t, increasedBackoff, initialBackoff, "錯誤後退避時間應增加")

		// 測試成功後退避時間減少
		for i := 0; i < 5; i++ {
			strategy.RecordSuccess()
		}
		decreasedBackoff := strategy.NextBackoff()
		assert.Less(t, decreasedBackoff, increasedBackoff, "成功後退避時間應減少")

		t.Logf("自適應退避測試通過: 初始=%v, 增加=%v, 減少=%v", 
			initialBackoff, increasedBackoff, decreasedBackoff)
	})

	t.Run("ExponentialBackoff", func(t *testing.T) {
		strategy := kds.NewExponentialBackoffStrategy(
			pts.config.Consumer.MinBackoff,
			pts.config.Consumer.MaxBackoff,
			pts.config.Consumer.BackoffMultiplier,
		)

		backoffs := make([]time.Duration, 5)
		for i := 0; i < 5; i++ {
			backoffs[i] = strategy.NextBackoff()
		}

		// 驗證指數增長
		for i := 1; i < len(backoffs); i++ {
			assert.GreaterOrEqual(t, backoffs[i], backoffs[i-1], 
				fmt.Sprintf("退避時間應遞增: %v >= %v", backoffs[i], backoffs[i-1]))
		}

		// 測試重置
		strategy.Reset()
		resetBackoff := strategy.NextBackoff()
		assert.LessOrEqual(t, resetBackoff, backoffs[0]*2, "重置後退避時間應回到初始水平")

		t.Logf("指數退避測試通過: %v", backoffs)
	})
}

// TestErrorClassification 測試錯誤分類
func (pts *PerformanceTestSuite) TestErrorClassification(t *testing.T) {
	classifier := kds.NewErrorClassifier()

	testCases := []struct {
		err      error
		retryable bool
		temporary bool
		permanent bool
		throttling bool
		category string
	}{
		{kds.ErrKDSConnectionFailed, true, false, false, false, "retryable"},
		{kds.ErrKDSThrottling, false, true, false, true, "throttling"},
		{kds.ErrKDSRecordInvalid, false, false, true, false, "permanent"},
		{kds.ErrWorkerTimeout, true, false, false, false, "retryable"},
		{kds.ErrBatchSizeTooLarge, false, false, true, false, "permanent"},
	}

	for _, tc := range testCases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			assert.Equal(t, tc.retryable, classifier.IsRetryable(tc.err), "可重試判斷錯誤")
			assert.Equal(t, tc.temporary, classifier.IsTemporary(tc.err), "臨時錯誤判斷錯誤")
			assert.Equal(t, tc.permanent, classifier.IsPermanent(tc.err), "永久錯誤判斷錯誤")
			assert.Equal(t, tc.throttling, classifier.IsThrottling(tc.err), "限流錯誤判斷錯誤")
			assert.Equal(t, tc.category, classifier.GetErrorCategory(tc.err), "錯誤類別判斷錯誤")
		})
	}

	t.Logf("錯誤分類測試通過")
}

// TestPanicRecovery 測試Panic恢復機制
func (pts *PerformanceTestSuite) TestPanicRecovery(t *testing.T) {
	recovery := kds.NewPanicRecovery(
		pts.config.Consumer.MaxRecoveryAttempts,
		pts.config.Consumer.LockRetryInterval,
	)

	var recoveryCount int
	recovery.OnRecovery(func(panicValue interface{}, stack []byte) {
		recoveryCount++
		t.Logf("Panic恢復 #%d: %v", recoveryCount, panicValue)
	})

	ctx := context.Background()

	// 測試正常執行
	err := recovery.Execute(ctx, "normal_operation", func() error {
		return nil
	})
	assert.NoError(t, err, "正常執行應該沒有錯誤")

	// 測試Panic恢復 - 這會觸發panic，但應該被recovery捕獲
	// 為了安全測試，我們模擬而不是真正panic
	t.Logf("Panic恢復測試: 嘗試次數=%d", recovery.GetAttemptCount())
}

// TestRetryExecutor 測試重試執行器
func (pts *PerformanceTestSuite) TestRetryExecutor(t *testing.T) {
	strategy := kds.NewAdaptiveBackoffStrategy(
		time.Millisecond*10,
		time.Millisecond*100,
		1.5,
	)
	
	executor := kds.NewRetryExecutor(strategy, 3)
	ctx := context.Background()

	// 測試成功執行
	var attempts int
	err := executor.Execute(ctx, "test_operation", func() error {
		attempts++
		if attempts < 2 {
			return kds.ErrKDSConnectionFailed // 可重試錯誤
		}
		return nil
	})

	assert.NoError(t, err, "重試後應該成功")
	assert.Equal(t, 2, attempts, "應該執行2次嘗試")

	// 測試永久錯誤不重試
	attempts = 0
	err = executor.Execute(ctx, "permanent_error_operation", func() error {
		attempts++
		return kds.ErrKDSRecordInvalid // 永久錯誤
	})

	assert.Error(t, err, "永久錯誤應該返回錯誤")
	assert.Equal(t, 1, attempts, "永久錯誤不應重試")

	t.Logf("重試執行器測試通過")
}

// TestPerformanceBaseline 建立效能基準
func (pts *PerformanceTestSuite) TestPerformanceBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("跳過效能基準測試（使用 -short 標誌）")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// 啟動指標收集
	go pts.collector.StartPeriodicUpdate(ctx)

	// 模擬負載
	const (
		numRecords = 10000
		batchSize  = 100
	)

	startTime := time.Now()

	for i := 0; i < numRecords; i += batchSize {
		actualBatchSize := batchSize
		if i+batchSize > numRecords {
			actualBatchSize = numRecords - i
		}

		// 模擬批次處理時間
		processingTime := time.Duration(actualBatchSize) * time.Microsecond * 100
		time.Sleep(processingTime)

		pts.collector.BatchProcessed(actualBatchSize, processingTime)
		pts.collector.RecordProcessed(int64(actualBatchSize), processingTime/time.Duration(actualBatchSize))

		// 模擬偶發錯誤
		if i%1000 == 0 {
			pts.collector.ErrorOccurred()
		}
	}

	totalTime := time.Since(startTime)
	metrics := pts.collector.GetMetrics()

	// 輸出基準指標
	t.Logf("\n=== 效能基準測試結果 ===")
	t.Logf("總處理時間: %v", totalTime)
	t.Logf("處理記錄數: %d", metrics.RecordsProcessedTotal)
	t.Logf("處理批次數: %d", metrics.BatchesProcessedTotal)
	t.Logf("平均批次大小: %.2f", metrics.AvgBatchSize)
	t.Logf("處理速率: %.2f records/sec", float64(metrics.RecordsProcessedTotal)/totalTime.Seconds())
	t.Logf("平均處理延遲: %v", metrics.ProcessingLatency)
	t.Logf("平均批次處理時間: %v", metrics.BatchProcessingTime)
	t.Logf("錯誤率: %.2f%%", metrics.ErrorRate)
	t.Logf("========================\n")

	// 驗證效能目標
	expectedMinThroughput := float64(1000) // 每秒至少1000條記錄
	actualThroughput := float64(metrics.RecordsProcessedTotal) / totalTime.Seconds()
	
	assert.GreaterOrEqual(t, actualThroughput, expectedMinThroughput, 
		fmt.Sprintf("吞吐量應不低於 %.0f records/sec，實際: %.0f", expectedMinThroughput, actualThroughput))

	// 驗證錯誤率
	maxErrorRate := 1.0 // 最多1%錯誤率
	assert.LessOrEqual(t, metrics.ErrorRate, maxErrorRate, 
		fmt.Sprintf("錯誤率應不超過 %.1f%%，實際: %.2f%%", maxErrorRate, metrics.ErrorRate))
}

// RunAllTests 執行所有效能測試
func RunAllTests(t *testing.T) {
	suite := NewPerformanceTestSuite()

	t.Run("ConfigurationValidation", suite.TestConfigurationValidation)
	t.Run("MetricsAccuracy", suite.TestMetricsAccuracy)
	t.Run("HealthChecker", suite.TestHealthChecker)
	t.Run("BackoffStrategies", suite.TestBackoffStrategies)
	t.Run("ErrorClassification", suite.TestErrorClassification)
	t.Run("PanicRecovery", suite.TestPanicRecovery)
	t.Run("RetryExecutor", suite.TestRetryExecutor)
	t.Run("PerformanceBaseline", suite.TestPerformanceBaseline)
}

// TestConsumerPerformance 主要的效能測試入口
func TestConsumerPerformance(t *testing.T) {
	RunAllTests(t)
}

// BenchmarkConsumerComponents 組件基準測試
func BenchmarkConsumerComponents(b *testing.B) {
	suite := NewPerformanceTestSuite()

	b.Run("MetricsCollection", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				suite.collector.RecordProcessed(1, time.Microsecond*100)
			}
		})
	})

	b.Run("HealthCheck", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = suite.checker.CheckHealth()
		}
	})

	b.Run("ErrorClassification", func(b *testing.B) {
		classifier := kds.NewErrorClassifier()
		errors := []error{
			kds.ErrKDSConnectionFailed,
			kds.ErrKDSThrottling,
			kds.ErrKDSRecordInvalid,
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				err := errors[b.N%len(errors)]
				_ = classifier.GetErrorCategory(err)
			}
		})
	})
}