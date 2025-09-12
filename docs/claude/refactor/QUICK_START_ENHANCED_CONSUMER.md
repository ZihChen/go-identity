# Enhanced Consumer 快速開始指南

## 🚀 快速啟動

### 1. 確認配置

檢查 `config/config.yaml` 中的 Consumer 配置：

```yaml
consumer:
  batch_size: 100
  worker_pool_size: 10
  max_shard_concurrency: 8
  enable_panic_recovery: true
```

### 2. 啟動增強版消費者

```bash
# 啟動消費者服務 (會自動使用增強版)
go run main.go consumer

# 或使用已編譯的二進制文件
./bin/consumer
```

### 3. 驗證運行狀態

檢查日誌輸出，應該看到：

```
INFO Starting enhanced consumer batch_size=100 worker_pool_size=10
INFO Enhanced consumer started successfully
INFO Starting main consumption loop
```

## 📊 快速性能檢查

### 監控關鍵指標

```bash
# 查看處理速率 (應該看到 records/sec 的日誌)
tail -f logs/consumer.log | grep "records_per_second"

# 查看批次處理統計
tail -f logs/consumer.log | grep "Batch processing completed"

# 查看 Worker 利用率
tail -f logs/consumer.log | grep "Worker pool statistics"
```

### 健康檢查端點

```bash
# 檢查服務健康狀態
curl http://localhost:8080/health

# 預期回應
{"status": "healthy", "consumer": "enhanced", "components": {...}}
```

## ⚡ 效能調優速查表

### 高吞吐量配置
```yaml
consumer:
  batch_size: 200          # ↑ 增大批次
  worker_pool_size: 15     # ↑ 增加 Worker  
  max_shard_concurrency: 12 # ↑ 提高並發
```

### 低延遲配置
```yaml
consumer:
  batch_size: 50           # ↓ 減小批次
  max_batch_wait_time: 200ms # ↓ 降低等待
  worker_pool_size: 8      # ↔ 適中配置
```

### 穩定性優化
```yaml
consumer:
  enable_panic_recovery: true
  max_recovery_attempts: 5   # ↑ 增加恢復嘗試
  min_backoff: 1s           # ↑ 增加退避時間
```

## 🔍 故障排除速查

### 常見問題及解決方案

| 問題症狀 | 可能原因 | 解決方案 |
|---------|---------|---------|
| 吞吐量低 | 批次太小/Worker 不足 | 增加 `batch_size` 和 `worker_pool_size` |
| 延遲過高 | 批次等待時間過長 | 降低 `max_batch_wait_time` |
| 頻繁錯誤 | 分片鎖競爭 | 檢查 `max_shard_concurrency` 設定 |
| Worker Panic | 程式邏輯錯誤 | 查看 PanicRecovery 日誌 |
| 記憶體使用高 | 緩衝區過大 | 調整 `worker_buffer_size` |

### 快速檢查命令

```bash
# 檢查編譯狀態
go build ./cmd/consumer

# 運行性能測試
go test ./internal/infrastructure/kds/ -bench=.

# 檢查配置有效性
go run main.go consumer --validate-config

# 查看詳細指標 (如果有監控端點)
curl http://localhost:8080/metrics | grep consumer_
```

## 📈 性能基準參考

### 典型環境表現

| 環境規格 | 預期吞吐量 | P95 延遲 | 建議配置 |
|----------|-----------|---------|---------|
| 2C4G | 1,500 records/sec | < 800ms | batch_size=80, workers=6 |
| 4C8G | 3,000 records/sec | < 500ms | batch_size=120, workers=10 |
| 8C16G | 5,000+ records/sec | < 300ms | batch_size=200, workers=16 |

### 配置調優步驟

1. **基線測試**: 使用預設配置運行 30 分鐘，記錄基準指標
2. **逐步調優**: 一次只調整一個參數，觀察 10 分鐘
3. **性能驗證**: 確認吞吐量提升且錯誤率不增加
4. **穩定性測試**: 長時間運行驗證穩定性

## 🔧 開發模式

### 本地開發配置

```yaml
consumer:
  batch_size: 10           # 小批次便於測試
  worker_pool_size: 3      # 少量 Worker
  max_shard_concurrency: 2 # 降低並發
  metrics_interval: 10s    # 更頻繁的指標
  enable_panic_recovery: true # 開發時保持穩定
```

### 調試技巧

```go
// 啟用詳細日誌 (在配置中)
log_level: debug

// 或在程式中臨時調整
logger.SetLevel(logger.DebugLevel)
```

## 📚 相關文檔

- [完整技術指南](./ENHANCED_CONSUMER_GUIDE.md) - 詳細架構和使用說明
- [重構計劃](./CONSUMER_REFACTOR_PLAN.md) - 完整的重構進度和計劃
- [CLAUDE.md](../../../CLAUDE.md) - 項目總體說明

## 🆘 支援聯絡

遇到問題時：

1. 檢查本指南的故障排除部分
2. 查看詳細技術指南
3. 檢查系統日誌和指標
4. 參考重構計劃中的測試方案

---

**快速開始版本**: v3.0 Enhanced Consumer  
**更新**: 2025-09-11  
**狀態**: 生產就緒 ✅

🎉 **恭喜！您現在已經開始使用高性能的 Enhanced Consumer！**