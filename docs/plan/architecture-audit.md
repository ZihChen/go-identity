# fat-identity-cat 架構審計報告

> 參考標準：fat-event-cat、fat-guardian-cat（較新的同系列專案）
>
> 審計日期：2026-03-11

---

## 嚴重問題（Critical）

### 1. 基礎設施框架型別洩漏至 Domain Port 層

多個 Domain port 介面直接引用了具體的基礎設施套件，嚴重違反 Clean Architecture：

- `internal/domain/ports/outbound/infrastructure/cache.go`：`CacheManager` 介面方法簽章暴露了 `*redsync.Mutex`、`redis.Pipeliner`、`*redis.Client` 等 Redis/Redsync 具體型別
- `internal/domain/ports/outbound/infrastructure/tracing.go`：`TracingService` 介面引用了 `otel/trace`、`otel/attribute`、`otel/codes` 具體型別
- `internal/domain/ports/outbound/service/queue.go`：`QueueService` 介面暴露了 `asynq.Handler` 型別

相比之下，fat-guardian-cat 的 Cache port 只使用標準庫型別（`Get/Set` 以 `string`/`time.Duration` 為參數）。

### 2. `AsyncLogger` 的非 Context 方法全部為空（靜默資料遺失）

`internal/infrastructure/logger/async_logger.go` 第 340–348 行的 `DebugLog`、`InfoLog`、`ErrorLog`、`WarnLog`、`FatalLog` 五個方法全是空實作。任何使用 `AsyncLogger` 呼叫這些方法的程式碼，日誌將被靜默丟棄。

### 3. 完全沒有 OpenTelemetry Metrics 實作

fat-identity-cat 沒有 `internal/infrastructure/metrics/` 套件。fat-event-cat 和 fat-guardian-cat 都有完整的 OTLP metrics 輸出（request rate、error rate、latency histogram、cache hit/miss 等）。fat-identity-cat 的 `go.mod` 中 `otel/metric` 只是間接依賴。

---

## 高嚴重度問題（High）

### 4. Domain Entity 方法使用 inline 錯誤字串，無法用 `errors.Is()` 判斷

`entity/merchant.go`、`entity/player.go`、`entity/manager.go`、`entity/tag.go`、`entity/level.go`、`entity/agent.go` 中的 `IsValid()` 等方法皆回傳 `errors.New("字串")`，而非在 `domain/consts/errors.go` 定義 sentinel error 再用 `fmt.Errorf("%w", consts.ErrX)` 包裝。fat-event-cat 和 fat-guardian-cat 均使用 sentinel error 模式。

### 5. Handler 在有 Context 的情境下仍使用非 Context 日誌方法

`internal/adapter/inbound/handler/api/http_handler.go` 中大量使用 `h.logger.ErrorLog()`/`InfoLog()`/`WarnLog()`（第 117、160、203、246、287、332、375、451 等行），導致 request 日誌缺失 trace ID 關聯。`internal/application/usecase/player/batch_processor.go` 中也有大量相同問題（第 87、110、122 等行）。

### 6. `TracingService` 被實例化兩次（重複建立 Bug）

`cmd/web/web.go` 第 171 行在 Wire 之外手動呼叫 `tracing.NewTracingService(cfg)`，而 `internal/di/wire.go` 內部的 `provideTracingService` 也會建立一個獨立的 `TracingService`。Handler 使用的 tracing 與 shutdown 管理的 tracing 是兩個不同物件。Worker 和 Consumer 服務也有相同問題。

### 7. 沒有 Domain Value Object

fat-event-cat 定義了 `valueobject.EventID`、`valueobject.MerchantID` 等帶有驗證的獨立型別。fat-identity-cat 完全沒有 `valueobject/` 套件，所有 ID（`globalMerchantID`、`globalPlayerID` 等）都是裸 `string`，無法在型別層級保證有效性。

---

## 中嚴重度問題（Medium）

### 8. Domain Entity 實作 `encoding/json` 序列化介面

`merchant.go`、`player.go`、`manager.go`、`tag.go`、`level.go` 都實作了 `MarshalJSON`/`UnmarshalJSON`，將 JSON 序列化關注點耦合進 Domain 層。fat-guardian-cat 的 entity 不引用 `encoding/json`，序列化由 adapter/infrastructure 層處理。

### 9. `LoggerFiled` 型別名稱有 Typo，且 Logger 介面設計與新專案不一致

- `internal/domain/entity/entity.go` 第 13 行：`LoggerFiled`（應為 `LogField`），此 typo 傳播至所有 logger 相關檔案
- `internal/domain/ports/outbound/infrastructure/logger.go`：Logger 介面缺少 `Duration(key string, value time.Duration)` 方法（fat-guardian-cat 第 29 行有此方法）
- Logger 介面包含 `Close()` 方法，將 lifecycle 關注點洩漏至 domain port

### 10. Tracing 套件的設計問題

- `internal/infrastructure/tracing/tracing.go` 第 82 行：`otlptracehttp.WithEndpointURL(cfg.Tracing.Endpoint)` 的回傳值被丟棄（dead statement），第 84 行才是實際生效的呼叫
- 第 112 行：Service version 硬編碼為 `"1.0.0"`，而非使用 `cfg.App.Version`（fat-event-cat 和 fat-guardian-cat 均使用 config 值）
- 第 18 行：使用 `semconv/v1.4.0`，新專案使用 `v1.26.0`
- `TracingConfig` 沒有 `Enabled bool` 欄位，無法優雅降級（fat-event-cat 和 fat-guardian-cat 都有 enabled 守衛）
- `TracingService` 介面有 10 個方法，其中包含 `TraceWorkerToKDS`、`TraceRedisToWorker` 等業務特定方法，違反 Interface Segregation Principle

### 11. `SendKDSTestEvent` Handler 對 Port 介面做 Runtime Type Assertion

`internal/adapter/inbound/handler/api/http_handler.go` 第 447–456 行，handler 將 `h.eventProducer` 斷言為包含 `SendToConsumeStream` 的匿名介面，隱性依賴具體實作，破壞依賴反轉原則。

### 12. Config 缺少 `mapstructure` 標籤與環境判斷輔助方法

- `internal/infrastructure/config/config.go`：Config struct 欄位無 `mapstructure` 標籤，導致使用大量手動 `viper.GetString()` 呼叫，而非 `viper.Unmarshal()`
- 沒有 `IsProduction()`、`IsDevelopment()`、`IsLocal()` 輔助方法（fat-guardian-cat 第 98–108 行有），環境判斷散落各處
- 使用獨立的 `APP_DEBUG bool` 而非從環境派生 log 模式

### 13. 整合測試缺少 Build Tag 或 Skip Guard

`test/database_test.go` 等整合測試在 `go test ./...` 時直接執行，沒有 `//go:build integration` 標籤或 `testing.Short()` 檢查，若 MySQL/Redis 不可用就會導致 CI 失敗。

### 14. Dead Code

- `internal/adapter/inbound/middleware/auth_middleware.go` 第 50–106 行：`decryptAPIKey` 函式定義但從未被呼叫
- `internal/adapter/inbound/middleware/cors_middleware.go` 第 178–196 行：`parseCorsOrigins` 函式定義但從未被呼叫
- `internal/adapter/inbound/middleware/tracing_middleware.go` 第 51–63 行：`withTraceContext` 的 `if/else` 兩個分支執行完全相同的程式碼

### 15. `HTTPHandler` 設計為巨型單一 Handler

`internal/adapter/inbound/handler/api/http_handler.go` 第 38–46 行，單一 struct 持有 merchantUseCase、playerUseCase、managerUseCase、logger、tracing、eventProducer、jwtService 所有依賴。fat-guardian-cat 按業務域拆分為 `MerchantAdminHandler`、`NDRPHandler`、`BatchHandler`、`HealthHandler`，各自只依賴所需的 use case。

---

## 低嚴重度問題（Low）

### 16. Adapter 缺少編譯時介面合規斷言

所有 `internal/adapter/outbound/repository/` 下的 repository 都沒有 `var _ outbound.X = (*ConcreteType)(nil)` 斷言。fat-guardian-cat 每個 adapter 檔案都有此模式（如 `merchant_repo.go` 第 31 行）。

### 17. `PlayerLevel` 型別在 `entity` 和 `event` 套件中重複定義

`internal/domain/entity/player.go` 和 `internal/domain/event/publish.go` 各自定義了結構相同的 `PlayerLevel struct`，可能靜默地分歧。

### 18. Wire DI 圖不完整，Router 和 Middleware 在 Wire 之外手動組裝

fat-guardian-cat 的 Wire 圖涵蓋到 router（`wire.go` 第 64 行），fat-identity-cat 的 Wire 只到 handler 層，Router 和 Middleware 在 `cmd/web/web.go` 中手動呼叫。

### 19. 依賴版本落後

| 依賴 | fat-identity-cat | fat-guardian-cat |
|------|-----------------|-----------------|
| Go | 1.24.0 | 1.25.6 |
| `otel/otel` | v1.37.0 | v1.40.0 |
| `gin-gonic/gin` | v1.10.1 | v1.11.0 |
| `google/wire` | v0.6.0 | v0.7.0 |
| `gorm.io/gorm` | v1.30.0 | v1.31.1 |

### 20. `fmt.Printf` 用於 Cache 錯誤日誌（熱路徑）

`internal/infrastructure/utils/helper.go` 第 33–53 行的 `QueryWithCache` 函式使用 `fmt.Printf` 輸出 cache 錯誤，完全繞過結構化 logger，導致這些錯誤在日誌聚合系統中消失。

---

## 優先處理建議

| 優先級 | 項目 |
|--------|------|
| P0 | #1 Domain port 框架型別洩漏、#2 AsyncLogger 靜默丟棄日誌、#3 缺少 Metrics |
| P1 | #4 Inline 錯誤字串、#5 非 Context 日誌方法、#6 TracingService 雙重實例化 |
| P2 | #8 Entity JSON 序列化、#9 LoggerFiled typo、#10 Tracing 設計問題 |
| P3 | 其餘技術債（dead code、版本更新、測試改善） |
