package kds

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// BenchmarkMetricsCollector 測試 MetricsCollector 效能
func BenchmarkMetricsCollector(b *testing.B) {
	collector := NewMetricsCollector(time.Second)
	ctx := context.Background()

	b.Run("RecordProcessed", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				collector.RecordProcessed(1, time.Millisecond*10)
			}
		})
	})

	b.Run("BatchProcessed", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				collector.BatchProcessed(100, time.Millisecond*50)
			}
		})
	})

	b.Run("ErrorOccurred", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				collector.ErrorOccurred()
			}
		})
	})

	b.Run("UpdateMetrics", func(b *testing.B) {
		// 先添加一些數據
		for i := 0; i < 1000; i++ {
			collector.RecordProcessed(1, time.Millisecond*10)
			collector.BatchProcessed(100, time.Millisecond*50)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			collector.UpdateMetrics()
		}
	})

	b.Run("GetMetrics", func(b *testing.B) {
		// 先添加一些數據
		for i := 0; i < 1000; i++ {
			collector.RecordProcessed(1, time.Millisecond*10)
		}

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = collector.GetMetrics()
			}
		})
	})

	// 清理
	go collector.StartPeriodicUpdate(ctx)
	time.Sleep(time.Millisecond * 100)
}

// BenchmarkBackoffStrategy 測試退避策略效能
func BenchmarkBackoffStrategy(b *testing.B) {
	b.Run("AdaptiveBackoff", func(b *testing.B) {
		strategy := NewAdaptiveBackoffStrategy(
			time.Millisecond*100,
			time.Second*5,
			1.5,
		)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = strategy.NextBackoff()
				if rand.Intn(10) > 7 {
					strategy.RecordError(ErrKDSConnectionFailed)
				} else {
					strategy.RecordSuccess()
				}
			}
		})
	})

	b.Run("ExponentialBackoff", func(b *testing.B) {
		strategy := NewExponentialBackoffStrategy(
			time.Millisecond*100,
			time.Second*5,
			1.5,
		)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = strategy.NextBackoff()
				if rand.Intn(10) > 8 {
					strategy.Reset()
				}
			}
		})
	})
}

// BenchmarkErrorClassifier 測試錯誤分類器效能
func BenchmarkErrorClassifier(b *testing.B) {
	classifier := NewErrorClassifier()
	errors := []error{
		ErrKDSConnectionFailed,
		ErrKDSThrottling,
		ErrKDSRecordInvalid,
		ErrBatchProcessingFailed,
		ErrWorkerTimeout,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := errors[rand.Intn(len(errors))]
			_ = classifier.IsRetryable(err)
			_ = classifier.IsTemporary(err)
			_ = classifier.IsPermanent(err)
			_ = classifier.IsThrottling(err)
		}
	})
}

// BenchmarkPanicRecovery 測試 Panic 恢復機制效能
func BenchmarkPanicRecovery(b *testing.B) {
	recovery := NewPanicRecovery(3, time.Millisecond*10)
	ctx := context.Background()

	b.Run("NormalExecution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := recovery.Execute(ctx, "test", func() error {
				return nil
			})
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// LoadTest 負載測試結構
type LoadTest struct {
	duration        time.Duration
	concurrency     int
	recordsPerSec   int
	batchSize       int
	errorRate       float64
	metricsInterval time.Duration
}

// RunLoadTest 執行負載測試
func (lt *LoadTest) RunLoadTest(b *testing.B) {
	collector := NewMetricsCollector(lt.metricsInterval)
	ctx, cancel := context.WithTimeout(context.Background(), lt.duration)
	defer cancel()

	// 啟動指標收集
	go collector.StartPeriodicUpdate(ctx)

	var wg sync.WaitGroup
	recordChan := make(chan struct{}, lt.recordsPerSec)

	// 記錄生成器
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(lt.recordsPerSec))
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				close(recordChan)
				return
			case <-ticker.C:
				select {
				case recordChan <- struct{}{}:
				default:
					// 通道滿了，跳過
				}
			}
		}
	}()

	// 處理 workers
	for i := 0; i < lt.concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			batch := make([]struct{}, 0, lt.batchSize)
			batchTimer := time.NewTimer(time.Millisecond * 500)
			defer batchTimer.Stop()

			for {
				select {
				case <-ctx.Done():
					// 處理剩餘的批次
					if len(batch) > 0 {
						lt.processBatch(collector, batch, workerID)
					}
					return
				case record, ok := <-recordChan:
					if !ok {
						return
					}
					batch = append(batch, record)

					// 批次滿了或超時，處理批次
					if len(batch) >= lt.batchSize {
						lt.processBatch(collector, batch, workerID)
						batch = batch[:0]
						batchTimer.Reset(time.Millisecond * 500)
					}
				case <-batchTimer.C:
					if len(batch) > 0 {
						lt.processBatch(collector, batch, workerID)
						batch = batch[:0]
					}
					batchTimer.Reset(time.Millisecond * 500)
				}
			}
		}(i)
	}

	wg.Wait()

	// 獲取最終指標
	finalMetrics := collector.GetMetrics()
	b.Logf("LoadTest Results:")
	b.Logf("  Duration: %v", lt.duration)
	b.Logf("  Concurrency: %d", lt.concurrency)
	b.Logf("  Target Records/sec: %d", lt.recordsPerSec)
	b.Logf("  Batch Size: %d", lt.batchSize)
	b.Logf("  Actual Records Processed: %d", finalMetrics.RecordsProcessedTotal)
	b.Logf("  Actual Records/sec: %.2f", finalMetrics.RecordsProcessedRate)
	b.Logf("  Batches Processed: %d", finalMetrics.BatchesProcessedTotal)
	b.Logf("  Average Batch Size: %.2f", finalMetrics.AvgBatchSize)
	b.Logf("  Processing Latency: %v", finalMetrics.ProcessingLatency)
	b.Logf("  Batch Processing Time: %v", finalMetrics.BatchProcessingTime)
	b.Logf("  Error Rate: %.2f%%", finalMetrics.ErrorRate)
	b.Logf("  Failed Records: %d", finalMetrics.FailedRecords)
}

// processBatch 模擬批次處理
func (lt *LoadTest) processBatch(collector *MetricsCollector, batch []struct{}, workerID int) {
	startTime := time.Now()

	// 模擬處理時間（10-50ms）
	processingTime := time.Duration(rand.Intn(40)+10) * time.Millisecond
	time.Sleep(processingTime)

	actualProcessingTime := time.Since(startTime)
	batchSize := len(batch)

	// 模擬錯誤
	shouldError := rand.Float64() < lt.errorRate
	if shouldError {
		collector.ErrorOccurred()
	}

	// 記錄指標
	collector.BatchProcessed(batchSize, actualProcessingTime)
	collector.RecordProcessed(int64(batchSize), actualProcessingTime/time.Duration(batchSize))
}

// BenchmarkLoadTest 執行不同負載場景的測試
func BenchmarkLoadTest(b *testing.B) {
	tests := []struct {
		name string
		test LoadTest
	}{
		{
			name: "LightLoad",
			test: LoadTest{
				duration:        time.Second * 10,
				concurrency:     2,
				recordsPerSec:   100,
				batchSize:       10,
				errorRate:       0.01, // 1% 錯誤率
				metricsInterval: time.Second,
			},
		},
		{
			name: "MediumLoad",
			test: LoadTest{
				duration:        time.Second * 10,
				concurrency:     5,
				recordsPerSec:   500,
				batchSize:       50,
				errorRate:       0.02, // 2% 錯誤率
				metricsInterval: time.Second,
			},
		},
		{
			name: "HighLoad",
			test: LoadTest{
				duration:        time.Second * 10,
				concurrency:     10,
				recordsPerSec:   1000,
				batchSize:       100,
				errorRate:       0.05, // 5% 錯誤率
				metricsInterval: time.Second,
			},
		},
		{
			name: "BurstLoad",
			test: LoadTest{
				duration:        time.Second * 5,
				concurrency:     20,
				recordsPerSec:   2000,
				batchSize:       200,
				errorRate:       0.03, // 3% 錯誤率
				metricsInterval: time.Millisecond * 500,
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			// 只執行一次負載測試
			b.N = 1
			for i := 0; i < b.N; i++ {
				tt.test.RunLoadTest(b)
			}
		})
	}
}

// BenchmarkMemoryUsage 測試記憶體使用情況
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("MetricsCollectorMemory", func(b *testing.B) {
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		collector := NewMetricsCollector(time.Second)

		// 模擬大量操作
		for i := 0; i < 10000; i++ {
			collector.RecordProcessed(1, time.Millisecond*10)
			collector.BatchProcessed(100, time.Millisecond*50)
			if i%100 == 0 {
				collector.ErrorOccurred()
			}
		}

		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		b.Logf("Memory allocated: %d KB", (m2.Alloc-m1.Alloc)/1024)
		b.Logf("Total allocations: %d", m2.TotalAlloc-m1.TotalAlloc)
		b.Logf("GC cycles: %d", m2.NumGC-m1.NumGC)
	})
}

// BenchmarkConcurrency 測試併發安全性
func BenchmarkConcurrency(b *testing.B) {
	collector := NewMetricsCollector(time.Second)
	const numGoroutines = 100
	const operationsPerGoroutine = 1000

	b.Run("ConcurrentOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var wg sync.WaitGroup

			// 多個 goroutine 同時操作
			for g := 0; g < numGoroutines; g++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for op := 0; op < operationsPerGoroutine; op++ {
						switch op % 5 {
						case 0:
							collector.RecordProcessed(1, time.Microsecond*100)
						case 1:
							collector.BatchProcessed(10, time.Millisecond)
						case 2:
							collector.ErrorOccurred()
						case 3:
							collector.RetryAttempted()
						case 4:
							_ = collector.GetMetrics()
						}
					}
				}()
			}
			wg.Wait()
		}
	})
}

// StressTest 壓力測試結構
type StressTest struct {
	name             string
	duration         time.Duration
	maxGoroutines    int
	rampUpDuration   time.Duration
	operationsPerSec int
}

// RunStressTest 執行壓力測試
func (st *StressTest) RunStressTest(b *testing.B) {
	collector := NewMetricsCollector(time.Millisecond * 100)
	ctx, cancel := context.WithTimeout(context.Background(), st.duration)
	defer cancel()

	go collector.StartPeriodicUpdate(ctx)

	var wg sync.WaitGroup
	var activeGoroutines int32

	// 逐漸增加負載
	rampUpTicker := time.NewTicker(st.rampUpDuration / time.Duration(st.maxGoroutines))
	defer rampUpTicker.Stop()

	go func() {
		for i := 0; i < st.maxGoroutines; i++ {
			select {
			case <-ctx.Done():
				return
			case <-rampUpTicker.C:
				wg.Add(1)
				activeGoroutines++

				go func(workerID int) {
					defer wg.Done()
					defer func() { activeGoroutines-- }()

					ticker := time.NewTicker(time.Second / time.Duration(st.operationsPerSec))
					defer ticker.Stop()

					for {
						select {
						case <-ctx.Done():
							return
						case <-ticker.C:
							// 執行操作
							processingTime := time.Duration(rand.Intn(50)+10) * time.Millisecond
							collector.RecordProcessed(1, processingTime)

							if rand.Intn(10) == 0 { // 10% 的機會處理批次
								collector.BatchProcessed(rand.Intn(100)+1, processingTime)
							}

							if rand.Intn(100) < 2 { // 2% 錯誤率
								collector.ErrorOccurred()
							}
						}
					}
				}(i)
			}
		}
	}()

	wg.Wait()

	finalMetrics := collector.GetMetrics()
	b.Logf("Stress Test [%s] Results:", st.name)
	b.Logf("  Duration: %v", st.duration)
	b.Logf("  Max Goroutines: %d", st.maxGoroutines)
	b.Logf("  Ramp Up Duration: %v", st.rampUpDuration)
	b.Logf("  Target Operations/sec per goroutine: %d", st.operationsPerSec)
	b.Logf("  Total Records Processed: %d", finalMetrics.RecordsProcessedTotal)
	b.Logf("  Actual Records/sec: %.2f", finalMetrics.RecordsProcessedRate)
	b.Logf("  Average Processing Latency: %v", finalMetrics.ProcessingLatency)
	b.Logf("  Error Rate: %.2f%%", finalMetrics.ErrorRate)
	b.Logf("  Panic Recoveries: %d", finalMetrics.PanicRecoveryCount)
	b.Logf("  Memory Usage: %d KB", finalMetrics.MemoryUsage/1024)
}

// BenchmarkStressTest 執行壓力測試
func BenchmarkStressTest(b *testing.B) {
	tests := []StressTest{
		{
			name:             "RapidRampUp",
			duration:         time.Second * 30,
			maxGoroutines:    50,
			rampUpDuration:   time.Second * 5,
			operationsPerSec: 10,
		},
		{
			name:             "SustainedLoad",
			duration:         time.Second * 60,
			maxGoroutines:    20,
			rampUpDuration:   time.Second * 10,
			operationsPerSec: 20,
		},
		{
			name:             "HighConcurrency",
			duration:         time.Second * 20,
			maxGoroutines:    100,
			rampUpDuration:   time.Second * 2,
			operationsPerSec: 5,
		},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.N = 1 // 壓力測試只執行一次
			test.RunStressTest(b)
		})
	}
}

// TestBaseline 建立基準測試，用於比較重構前後的效能
func TestBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("跳過基準測試（使用 -short 標誌）")
	}

	// 建立當前實現的基準
	collector := NewMetricsCollector(time.Second)

	// 模擬當前 Consumer 的工作負載
	baselineTest := LoadTest{
		duration:        time.Second * 30,
		concurrency:     1,  // 當前實現是單線程
		recordsPerSec:   50, // 保守估計當前吞吐量
		batchSize:       1,  // 當前是單記錄處理
		errorRate:       0.02,
		metricsInterval: time.Second * 5,
	}

	t.Logf("執行基準測試...")
	baselineTest.RunLoadTest(&testing.B{})

	finalMetrics := collector.GetMetrics()

	// 輸出基準數據
	t.Logf("\n=== 基準測試結果 ===")
	t.Logf("吞吐量: %.2f records/sec", finalMetrics.RecordsProcessedRate)
	t.Logf("處理延遲: %v", finalMetrics.ProcessingLatency)
	t.Logf("錯誤率: %.2f%%", finalMetrics.ErrorRate)
	t.Logf("========================\n")

	// 保存基準數據到文件或環境變數，供後續比較使用
	// 這裡可以擴展為將結果寫入文件
}
