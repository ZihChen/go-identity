# Design: 修復 Domain Port 層框架型別洩漏

**日期**：2026-03-11
**問題來源**：`docs/plan/architecture-audit.md` 問題 #1

---

## 問題陳述

三個 Domain Port 介面直接引用了具體的基礎設施框架型別，違反 Clean Architecture 的依賴規則（外層依賴內層，而非內層依賴外層）：

| Port 檔案 | 洩漏型別 |
|-----------|---------|
| `domain/ports/outbound/infrastructure/cache.go` | `redis.Pipeliner`、`*redis.Client`、`*redsync.Mutex`、`*redsync.Redsync`、`redsync.Option` |
| `domain/ports/outbound/infrastructure/tracing.go` | `trace.Span`、`trace.SpanStartOption`、`attribute.KeyValue`、`codes.Code` |
| `domain/ports/outbound/service/queue.go` | `asynq.Handler` |

---

## 解決方案：方案 B（務實分層修正）

### 原則

1. Domain port 只允許引用 `context`、`time`、`errors` 等標準庫型別，以及 domain 自身定義的型別
2. 框架型別由 infrastructure adapter 實作層處理
3. 最小化對現有業務邏輯的影響

---

## 架構變更

### 新增檔案

```
internal/domain/entity/span.go          — domain Span interface + SpanAttr
internal/domain/entity/cache.go         — CacheSetEntry 輔助型別
internal/domain/ports/outbound/infrastructure/lock.go  — DistributedLockService port
```

### 修改的 Port 介面

#### 1. `cache.go` — 移除 Redis 型別

移除：
- `Pipeline() (redis.Pipeliner, error)`
- `GetClient() (*redis.Client, error)`
- `GetMutex(key string, expireTime time.Duration) (*redsync.Mutex, error)`
- `GetMutexWithOption(key string, options ...redsync.Option) (*redsync.Mutex, error)`
- `GetRedsync() (*redsync.Redsync, error)`

新增：
- `BatchSet(ctx context.Context, entries []entity.CacheSetEntry, ttl time.Duration) error`
- `BatchDelete(ctx context.Context, keys []string) error`

#### 2. `lock.go` — 新增獨立的分散式鎖 Port

```go
type DistributedMutex interface {
    Lock() error
    Unlock() (bool, error)
}

type DistributedLockService interface {
    GetLock(key string, ttl time.Duration) (DistributedMutex, error)
    GetLockWithRetry(key string, ttl time.Duration, tries int, delay time.Duration) (DistributedMutex, error)
}
```

#### 3. `tracing.go` — 回傳 domain Span

移除所有 `trace.Span`、`attribute.KeyValue`、`codes.Code`、`trace.SpanStartOption` 引用，改用：
- `entity.Span` interface
- `entity.SpanAttr` struct

#### 4. `queue.go` — 移除 `WrapHandlerWithTracing`

此方法是 adapter 層關注點：`WorkerHandler` 本身已在 adapter 層，可直接使用 asynq 型別。改由 `worker_handler.go` 在 `RegisterHandlers` 中自行用 closure 包裝 tracing 邏輯。

---

## 新增 Domain 型別

### `domain/entity/span.go`

```go
// Span 代表一個分散式追蹤的 span，不依賴 OTel 型別
type Span interface {
    End()
    RecordError(err error)
    AddEvent(name string, attrs ...SpanAttr)
    SetStatus(ok bool, desc string)
}

// SpanAttr 代表 span 的屬性鍵值對
type SpanAttr struct {
    Key   string
    Value any
}
```

### `domain/entity/cache.go`

```go
// CacheSetEntry 代表批次寫入 cache 的單一條目
type CacheSetEntry struct {
    Key   string
    Value string
}
```

---

## Infrastructure 層變更

### `infrastructure/cache/redis/manager.go`

新增實作：
- `BatchSet` → 用 Redis Pipeline 批次執行 SET
- `BatchDelete` → 用 Redis Pipeline 批次執行 DEL

### `infrastructure/cache/redis/lock.go`（新檔案）

實作 `DistributedLockService`，內部使用 `redsync`。

### `infrastructure/tracing/span.go`（新檔案）

定義 `otelSpan` struct 實作 `entity.Span`，包裝 `trace.Span`：

```go
type otelSpan struct {
    span trace.Span
}

func (s *otelSpan) End()                              { s.span.End() }
func (s *otelSpan) RecordError(err error)             { s.span.RecordError(err) }
func (s *otelSpan) AddEvent(name string, ...)         { ... }
func (s *otelSpan) SetStatus(ok bool, desc string)    { ... }
```

`TracingService.StartSpan` 回傳 `*otelSpan`（對外暴露為 `entity.Span`）。

---

## 呼叫端變更清單

| 檔案 | 變更 |
|------|------|
| `application/usecase/tag/tag_usecase.go` | `Pipeline()` → `cache.BatchSet()` |
| `application/usecase/player/batch_processor.go` | `Pipeline()` → `cache.BatchDelete()` |
| `infrastructure/utils/helper.go` | `GetMutexWithOption()` → 注入 `DistributedLockService`，呼叫 `GetLockWithRetry()` |
| `infrastructure/kds/consumer.go` | `GetMutex()` → 注入 `DistributedLockService`，呼叫 `GetLock()` |
| `infrastructure/cache/redis/cleanup_service.go` | `GetClient()` → 改注入 `*redis.Client` 具體型別（基礎設施內部） |
| `di/wire.go` | `GetClient()` → 直接使用 `*redis.Manager` 具體型別取得 client |
| `adapter/inbound/handler/worker/worker_handler.go` | 移除 `WrapHandlerWithTracing`、`trace.Span` → `entity.Span` |
| 所有使用 `trace.Span` 的 use case 方法 | 型別改為 `entity.Span` |
| `test/mocks/` | 更新 `NilCacheManager`、`TracingServiceMock`、`QueueServiceMock` 等 |

---

## 測試策略

- Mock 不再需要引入 `go-redis`、`redsync`、`go.opentelemetry.io/otel` 等依賴
- `NilTracingService` mock 改用 `entity.Span` 的 no-op 實作
- `NilCacheManager` mock 的 `BatchSet`/`BatchDelete` 直接回傳 nil
- Infrastructure 層新增 `BatchSet`/`BatchDelete` 的單元測試

---

## 不在本次範圍內

- `LoggerFiled` typo 修正（獨立問題 #9）
- `AsyncLogger` 空方法問題（問題 #2）
- Metrics 實作（問題 #3）
