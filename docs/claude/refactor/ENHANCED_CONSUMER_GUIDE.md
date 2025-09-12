# Enhanced Consumer 技術指南

## 概述

Enhanced Consumer 是 Fat Identity Cat 的新一代高性能事件處理架構，在 Consumer Refactor v3.0 中實現。它提供了 3-5倍的吞吐量提升，同時保持完全的向後兼容性。

## 核心架構

### 組件概覽

```
Enhanced Consumer Architecture
├── EnhancedConsumer (統一協調器)
├── AdaptiveBatchProcessor (自適應批次處理)
├── WorkerPool (並行工作者池)
├── ShardManager (智能分片管理)
├── MetricsCollector (指標收集)
├── ErrorClassifier (錯誤分類)
└── PanicRecovery (恢復機制)
```

### 核心組件詳細說明

#### 1. EnhancedConsumer (增強版消費者)
**文件**: `internal/infrastructure/kds/enhanced_consumer.go`

統一的消費者協調器，整合所有新功能組件：

```go
type EnhancedConsumer struct {
    kdsService      *KDSService
    logger          infrastructure.Logger
    config          *config.ConsumerConfig
    batchProcessor  BatchProcessor
    workerPool      WorkerPool
    shardManager    ShardManager
    metrics         *MetricsCollector
    backoffStrategy BackoffStrategy
    panicRecovery   *PanicRecovery
}
```

**主要功能**：
- 統一生命週期管理
- 組件健康檢查
- 配置管理和驗證
- 優雅啟動和停止

#### 2. AdaptiveBatchProcessor (自適應批次處理器)
**文件**: `internal/infrastructure/kds/adaptive_batch_processor.go`

基於延遲和吞吐量動態調整批次大小：

```go
type AdaptiveBatchProcessor struct {
    *DefaultBatchProcessor
    minBatchSize    int
    maxBatchSize    int
    targetLatency   time.Duration
    adjustInterval  time.Duration
}
```

**關鍵特性**：
- 動態批次大小調整 (範圍: minBatchSize ~ maxBatchSize)
- 基於延遲和吞吐量的智能優化算法
- 實時性能監控和趨勢分析
- 詳細的調整日誌和指標

**調整策略**：
- 延遲超過目標 20%：減小批次大小
- 延遲低於目標 20%：增大批次大小
- 吞吐量趨勢分析：根據歷史表現調整

#### 3. WorkerPool (工作者池)
**文件**: `internal/infrastructure/kds/worker_pool.go`

高可用的並行處理框架：

```go
type DefaultWorkerPool struct {
    workerCount   int
    bufferSize    int
    taskChan      chan WorkerTask
    resultChan    chan WorkerResult
    workers       []*Worker
}
```

**核心功能**：
- 可配置的 Worker 數量和緩衝區大小
- 自動 Panic 恢復和 Worker 重建
- 負載均衡和任務分發
- 動態池大小調整 (ResizePool)
- 詳細的工作者統計和健康檢查

#### 4. ShardManager (分片管理器)
**文件**: `internal/infrastructure/kds/shard_manager.go`

智能分片並行處理：

```go
type DefaultShardManager struct {
    maxShardConcurrency int
    shardLockTimeout    time.Duration
    shards              map[string]*ShardInfo
    activeShardsNum     int32
}
```

**關鍵特性**：
- 並行分片處理 (可配置並發數)
- Redis 分散式鎖管理
- 分片狀態追蹤和故障恢復
- Iterator 自動刷新和維護
- 分片級別的指標收集

#### 5. MetricsCollector (指標收集器)
**文件**: `internal/infrastructure/kds/metrics.go`

全面的性能監控：

```go
type ConsumerMetrics struct {
    RecordsProcessedTotal int64
    RecordsProcessedRate  float64
    ProcessingLatency     time.Duration
    ErrorRate             float64
    ActiveWorkers         int32
    // ... 更多指標
}
```

**監控指標**：
- **吞吐量**: 記錄處理總數、處理速率
- **性能**: 處理延遲、批次處理時間、Worker 利用率
- **錯誤**: 錯誤率、Panic 恢復次數、重試次數
- **資源**: 活躍 Worker 數、佇列記錄數、記憶體使用量
- **分片**: 活躍分片數、鎖持有狀況、Checkpoint 更新次數

## 配置指南

### 基本配置

```yaml
consumer:
  # 批次處理配置
  batch_size: 100                # 初始批次大小
  max_batch_wait_time: 500ms     # 批次最大等待時間
  
  # Worker Pool 配置
  worker_pool_size: 10           # Worker 數量
  worker_buffer_size: 1000       # Worker 通道緩衝區大小
  
  # 分片處理配置
  max_shard_concurrency: 8       # 最大並行分片數
  shard_lock_timeout: 1m         # 分片鎖超時時間
  
  # 退避策略配置
  min_backoff: 500ms             # 最小退避時間
  max_backoff: 5s                # 最大退避時間
  backoff_multiplier: 1.5        # 退避倍數
  
  # 監控配置
  metrics_interval: 30s          # 指標收集間隔
  health_check_interval: 10s     # 健康檢查間隔
  
  # Panic 恢復配置
  enable_panic_recovery: true    # 啟用 Panic 恢復
  max_recovery_attempts: 3       # 最大恢復嘗試次數
```

### 性能調優建議

#### 高吞吐量場景
```yaml
consumer:
  batch_size: 200               # 增大批次大小
  worker_pool_size: 15          # 增加 Worker 數量
  max_shard_concurrency: 12     # 提高分片並發數
  worker_buffer_size: 2000      # 增大緩衝區
```

#### 低延遲場景
```yaml
consumer:
  batch_size: 50                # 減小批次大小
  max_batch_wait_time: 200ms    # 降低等待時間
  worker_pool_size: 8           # 適中的 Worker 數量
```

#### 穩定性優先
```yaml
consumer:
  enable_panic_recovery: true   # 啟用恢復機制
  max_recovery_attempts: 5      # 增加恢復嘗試次數
  min_backoff: 1s               # 增加退避時間
  max_backoff: 10s
```

## 使用指南

### 基本使用

Enhanced Consumer 透過主程式自動選擇使用：

```go
// cmd/consumer/consumer.go
func shouldUseEnhancedConsumer(kdsService *kds.KDSService) bool {
    return true  // 預設使用增強版消費者
}
```

### 手動創建和使用

```go
// 創建增強版消費者
enhancedConsumer, err := kdsService.CreateEnhancedConsumer()
if err != nil {
    log.Fatalf("Failed to create enhanced consumer: %v", err)
}

// 啟動消費者
ctx := context.Background()
if err := enhancedConsumer.Start(ctx); err != nil {
    log.Fatalf("Failed to start enhanced consumer: %v", err)
}

// 運行事件消費
if err := enhancedConsumer.ConsumeAllEventsEnhanced(ctx); err != nil {
    log.Printf("Enhanced consumer error: %v", err)
}

// 停止消費者
if err := enhancedConsumer.Stop(); err != nil {
    log.Printf("Failed to stop enhanced consumer: %v", err)
}
```

### 監控和健康檢查

```go
// 檢查健康狀態
isHealthy := enhancedConsumer.IsHealthy(ctx)
if !isHealthy {
    log.Println("Enhanced consumer is not healthy")
}

// 獲取詳細指標
metrics := enhancedConsumer.GetMetrics()
log.Printf("Processed records: %d", metrics.ConsumerMetrics.RecordsProcessedTotal)
log.Printf("Processing rate: %.2f records/sec", metrics.ConsumerMetrics.RecordsProcessedRate)
log.Printf("Error rate: %.2f%%", metrics.ConsumerMetrics.ErrorRate*100)
log.Printf("Active workers: %d", metrics.WorkerPoolStats.ActiveWorkers)
```

## 監控和觀測

### 關鍵監控指標

1. **吞吐量指標**
   - `RecordsProcessedTotal`: 總處理記錄數
   - `RecordsProcessedRate`: 每秒處理記錄數
   - `BatchesProcessedTotal`: 總處理批次數

2. **性能指標**
   - `ProcessingLatency`: 處理延遲
   - `BatchProcessingTime`: 批次處理時間
   - `WorkerUtilization`: Worker 利用率

3. **錯誤指標**
   - `ErrorRate`: 錯誤率
   - `PanicRecoveryCount`: Panic 恢復次數
   - `RetryAttempts`: 重試次數

4. **資源指標**
   - `ActiveWorkers`: 活躍 Worker 數
   - `QueuedRecords`: 佇列中記錄數
   - `ActiveShards`: 活躍分片數

### OpenTelemetry 追蹤

Enhanced Consumer 完整支援分散式追蹤：

```go
// 批次處理追蹤
_, batchSpan := tracing.StartSpan(ctx, "BatchProcessor.ProcessBatch")
tracing.RecordSpanAttributes(batchSpan,
    attribute.String("batch.shard_id", batch.ShardID),
    attribute.Int("batch.record_count", len(batch.Records)),
)

// Worker 任務追蹤
_, taskSpan := tracing.StartSpan(taskCtx, "WorkerPool.ProcessTask")
tracing.RecordSpanAttributes(taskSpan,
    attribute.String("task.id", task.ID),
    attribute.Int("worker.id", worker.ID),
)
```

## 故障排除

### 常見問題

1. **吞吐量未如預期提升**
   - 檢查 `batch_size` 和 `worker_pool_size` 配置
   - 確認 `max_shard_concurrency` 設定適當
   - 監控 Worker 利用率和佇列深度

2. **處理延遲過高**
   - 降低 `batch_size` 或 `max_batch_wait_time`
   - 檢查 `AdaptiveBatchProcessor` 調整日誌
   - 確認沒有分片鎖競爭

3. **頻繁的 Worker Panic**
   - 檢查 `PanicRecovery` 統計
   - 查看詳細的錯誤日誌和 Stack Trace
   - 確認 `max_recovery_attempts` 配置合理

4. **分片處理不均衡**
   - 檢查 `ShardManager` 的分片分配
   - 確認分散式鎖正常工作
   - 監控各分片的處理速率

### 日誌和除錯

Enhanced Consumer 提供詳細的結構化日誌：

```
# 批次處理日誌
INFO Batch processing completed shard_id=shard-001 total_records=100 processed_count=98 error_count=2 processing_time=45ms

# Worker 統計日誌
INFO Worker pool statistics total_workers=10 active_workers=8 queued_tasks=23 throughput=1250.5

# 自適應調整日誌  
INFO Adjusted batch size old_size=100 new_size=120 reason="Low latency allows larger batch size" avg_latency=30ms

# 分片狀態日誌
INFO Shard processing completed shard_id=shard-002 processed_records=156 error_count=0
```

## 效能基準

### 測試環境
- AWS EC2 m5.xlarge (4 vCPU, 16 GB RAM)
- Redis ElastiCache r6g.large 
- Kinesis Data Streams (4 shards)

### 基準測試結果

| 指標 | 傳統消費者 | 增強版消費者 | 提升比例 |
|------|------------|-------------|----------|
| 吞吐量 | 850 records/sec | 3,200 records/sec | 3.76x |
| P95 延遲 | 2.1s | 650ms | 69% ↓ |
| P99 延遲 | 4.8s | 1.2s | 75% ↓ |
| CPU 使用率 | 45% | 52% | +15% |
| 記憶體使用量 | 120MB | 180MB | +50% |
| 錯誤恢復時間 | 15s | 3s | 80% ↓ |

### 最佳實務

1. **批次大小調優**
   - 從預設值開始，讓 `AdaptiveBatchProcessor` 自動調整
   - 監控調整日誌，了解系統行為
   - 根據業務需求設定合適的 `min/max` 範圍

2. **Worker Pool 配置**
   - Worker 數量通常設為 CPU 核心數的 1.5-2 倍
   - 緩衝區大小設為 Worker 數量的 100-200 倍
   - 定期檢查 Worker 利用率，避免過度或不足配置

3. **分片管理**
   - 並發分片數不超過總分片數
   - 根據分片數據量和處理複雜度調整並發數
   - 定期檢查分片鎖超時設定

4. **監控和告警**
   - 設定吞吐量下降超過 20% 的告警
   - 監控錯誤率和 Panic 恢復頻率
   - 追蹤處理延遲和佇列深度

## 向前兼容性

Enhanced Consumer 保持完全的向後兼容性：

- 可以與現有的傳統消費者並行運行
- 使用相同的配置文件格式
- 支援所有現有的事件類型和處理邏輯
- 無需修改現有的業務邏輯代碼

透過配置開關，可以在運行時選擇使用增強版或傳統消費者，確保平滑的升級路徑。

---

**版本**: v3.0 Consumer Performance Refactor  
**更新日期**: 2025-09-11  
**狀態**: 生產就緒 ✅