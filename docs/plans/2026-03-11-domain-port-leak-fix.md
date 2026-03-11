# Domain Port 框架型別洩漏修復 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 移除 Domain Port 介面中的 Redis、OTel、asynq 框架型別，改以 domain-defined 抽象型別取代。

**Architecture:** 新增 `entity.Span`、`entity.SpanAttr`、`entity.CacheSetEntry` 以及 `infrastructure.DistributedLockService` port。Infrastructure 層新增 `otelSpan` wrapper 實作 `entity.Span`，Redis `Manager` 新增 `BatchSet`/`BatchDelete`，並新增獨立的 `RedisLockService`。所有呼叫端（use cases、middleware、handler、wire）同步更新。

**Tech Stack:** Go 1.24、google/wire、go-redis/v9、go-redsync/v4、go.opentelemetry.io/otel、hibiken/asynq

---

## Task 1: 新增 domain entity 型別（Span、SpanAttr、CacheSetEntry）

**Files:**
- Create: `internal/domain/entity/span.go`
- Create: `internal/domain/entity/cache_entry.go`

**Step 1: 建立 `internal/domain/entity/span.go`**

```go
package entity

// Span 代表一個分散式追蹤的 span，不依賴任何 OTel 具體型別。
// 由 infrastructure 層的 otelSpan wrapper 實作。
type Span interface {
	End()
	RecordError(err error)
	AddEvent(name string, attrs ...SpanAttr)
	SetStatus(ok bool, desc string)
}

// SpanAttr 代表 span 的屬性鍵值對。
type SpanAttr struct {
	Key   string
	Value any
}

// 下列輔助函數讓呼叫端無需手動建構 SpanAttr struct，
// 對應原 go.opentelemetry.io/otel/attribute 的常用建構子。

func StringAttr(key, value string) SpanAttr       { return SpanAttr{Key: key, Value: value} }
func IntAttr(key string, value int) SpanAttr      { return SpanAttr{Key: key, Value: value} }
func Int64Attr(key string, value int64) SpanAttr  { return SpanAttr{Key: key, Value: value} }
func BoolAttr(key string, value bool) SpanAttr    { return SpanAttr{Key: key, Value: value} }
func Float64Attr(key string, value float64) SpanAttr { return SpanAttr{Key: key, Value: value} }
```

**Step 2: 建立 `internal/domain/entity/cache_entry.go`**

```go
package entity

// CacheSetEntry 代表批次寫入 cache 的單一條目。
type CacheSetEntry struct {
	Key   string
	Value string
}
```

**Step 3: 確認編譯**

```bash
cd /Users/winstonlin/go/src/fatcat/fat-identity-cat && go build ./internal/domain/entity/...
```

Expected: 成功，無錯誤。

**Step 4: Commit**

```bash
git add internal/domain/entity/span.go internal/domain/entity/cache_entry.go
git commit -m "feat(domain): add Span interface and CacheSetEntry domain types"
```

---

## Task 2: 新增 DistributedLockService domain port

**Files:**
- Create: `internal/domain/ports/outbound/infrastructure/lock.go`

**Step 1: 建立 `internal/domain/ports/outbound/infrastructure/lock.go`**

```go
package infrastructure

import "time"

// DistributedMutex 代表一個分散式互斥鎖，不依賴 redsync 具體型別。
type DistributedMutex interface {
	Lock() error
	Unlock() (bool, error)
}

// LockOptions 配置分散式鎖的取得選項。
type LockOptions struct {
	Expiry     time.Duration
	Tries      int
	RetryDelay time.Duration
}

// DistributedLockService 提供分散式鎖的抽象，供 domain/application 層使用。
// 由 infrastructure/cache/redis/lock.go 的 RedisLockService 實作。
type DistributedLockService interface {
	// GetLock 取得一個簡單的分散式鎖（使用預設重試策略）。
	GetLock(key string, ttl time.Duration) (DistributedMutex, error)
	// GetLockWithOptions 取得一個帶自訂選項的分散式鎖。
	GetLockWithOptions(key string, opts LockOptions) (DistributedMutex, error)
}
```

**Step 2: 確認編譯**

```bash
go build ./internal/domain/ports/...
```

Expected: 成功。

**Step 3: Commit**

```bash
git add internal/domain/ports/outbound/infrastructure/lock.go
git commit -m "feat(domain): add DistributedLockService port interface"
```

---

## Task 3: 更新 CacheManager port — 移除 Redis 型別

**Files:**
- Modify: `internal/domain/ports/outbound/infrastructure/cache.go`

**Step 1: 完整替換 `cache.go` 內容**

```go
package infrastructure

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// CacheManager 通用緩存管理器介面。
// 所有方法只使用標準庫型別或 domain entity 型別，不依賴任何 Redis 具體型別。
type CacheManager interface {
	// 連接管理
	Connect(ctx context.Context) error
	Close() error

	// 基本單筆操作
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error)
	Del(ctx context.Context, key string) error
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)

	// 批次操作（取代 Pipeline）
	BatchSet(ctx context.Context, entries []entity.CacheSetEntry, ttl time.Duration) error
	BatchDelete(ctx context.Context, keys []string) error

	// 健康檢查
	HealthCheck(ctx context.Context) error
}
```

**Step 2: 嘗試編譯，預期出現多個 compile error**

```bash
go build ./... 2>&1 | grep -v "^#" | head -40
```

Expected: `GetClient`, `GetMutex`, `GetMutexWithOption`, `GetRedsync`, `Pipeline` 相關錯誤出現。這些錯誤會在後續 Task 中逐一修復。

**Step 3: Commit**

```bash
git add internal/domain/ports/outbound/infrastructure/cache.go
git commit -m "refactor(domain): remove Redis framework types from CacheManager port"
```

---

## Task 4: 更新 TracingService port — 移除 OTel 型別

**Files:**
- Modify: `internal/domain/ports/outbound/infrastructure/tracing.go`

**Step 1: 完整替換 `tracing.go` 內容**

```go
package infrastructure

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// TracingService 定義分散式追蹤服務的抽象介面。
// 所有方法只使用標準庫型別或 domain entity 型別，不依賴任何 OTel 具體型別。
//
//go:generate mockery --name=TracingService --output=../../../../../../test/mocks --outpkg=mocks
type TracingService interface {
	// StartSpan 開始一個新的 span，回傳帶有新 span 的 context 以及 domain Span。
	StartSpan(ctx context.Context, spanName string) (context.Context, entity.Span)

	// SpanEnd 結束 span。
	SpanEnd(span entity.Span)

	// RecordSpanError 記錄 span 錯誤。
	RecordSpanError(span entity.Span, err error)

	// RecordSpanAttributes 記錄 span 屬性。
	RecordSpanAttributes(span entity.Span, attrs ...entity.SpanAttr)

	// TraceEvent 為 span 新增事件。
	TraceEvent(span entity.Span, name string, attrs ...entity.SpanAttr)

	// RecordSpanStatus 記錄 span 的最終狀態（ok=true 代表成功）。
	RecordSpanStatus(span entity.Span, ok bool, desc string)

	// GetTraceparent 從 context 中取得 W3C traceparent 字串。
	GetTraceparent(ctx context.Context) string

	// InjectTraceparentToJSON 將 traceparent 注入至 JSON payload 中。
	InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error)

	// ExtractTraceContext 從 JSON payload 中提取 trace context 並注入 context。
	ExtractTraceContext(ctx context.Context, carrier []byte) context.Context

	// TraceWorkerToKDS Worker 發布至 KDS 的便利追蹤包裝。
	TraceWorkerToKDS(ctx context.Context, eventType, eventID string) (context.Context, entity.Span)

	// TraceRedisToWorker Redis 佇列至 Worker 消費的便利追蹤包裝。
	TraceRedisToWorker(ctx context.Context, taskType, taskID string) (context.Context, entity.Span)

	// TraceWorkerProcessing Worker 處理任務的便利追蹤包裝。
	TraceWorkerProcessing(ctx context.Context, taskType, taskID string) (context.Context, entity.Span)
}
```

注意：移除了 `opts ...trace.SpanStartOption` 參數（呼叫端無需傳入 OTel 選項），以及 `RecordSpanAttributes` 改用 `entity.SpanAttr`。`RecordSpanStatus` 改用 `ok bool` 取代 `codes.Code`。

**Step 2: Commit**

```bash
git add internal/domain/ports/outbound/infrastructure/tracing.go
git commit -m "refactor(domain): remove OTel framework types from TracingService port"
```

---

## Task 5: 更新 QueueService port — 移除 WrapHandlerWithTracing

**Files:**
- Modify: `internal/domain/ports/outbound/service/queue.go`

**Step 1: 移除 `WrapHandlerWithTracing` 方法，移除 `asynq` import**

```go
package service

import "context"

// QueueService 隊列服務接口
type QueueService interface {
	EnqueueMerchantSync(ctx context.Context, data []byte) error
	EnqueuePlayerSync(ctx context.Context, data []byte) error
	EnqueueManagerSync(ctx context.Context, data []byte) error
	EnqueueLevelSync(ctx context.Context, data []byte) error
	EnqueueTagSync(ctx context.Context, data []byte) error
	EnqueueAgentSync(ctx context.Context, data []byte) error
	Close() error
}
```

**Step 2: Commit**

```bash
git add internal/domain/ports/outbound/service/queue.go
git commit -m "refactor(domain): remove asynq.Handler from QueueService port"
```

---

## Task 6: Infrastructure — 新增 otelSpan wrapper

**Files:**
- Create: `internal/infrastructure/tracing/span.go`

**Step 1: 建立 `span.go`**

```go
package tracing

import (
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// otelSpan 包裝 trace.Span 以實作 entity.Span interface。
type otelSpan struct {
	span trace.Span
}

// newOtelSpan 將 OTel trace.Span 包裝為 entity.Span。
// 若傳入 nil，回傳 noopSpan 避免 nil pointer dereference。
func newOtelSpan(s trace.Span) entity.Span {
	if s == nil {
		return &noopSpan{}
	}
	return &otelSpan{span: s}
}

func (s *otelSpan) End() { s.span.End() }

func (s *otelSpan) RecordError(err error) {
	if s.span.IsRecording() {
		s.span.RecordError(err)
	}
}

func (s *otelSpan) AddEvent(name string, attrs ...entity.SpanAttr) {
	if !s.span.IsRecording() {
		return
	}
	otelAttrs := toOtelAttrs(attrs)
	s.span.AddEvent(name, trace.WithAttributes(otelAttrs...))
}

func (s *otelSpan) SetStatus(ok bool, desc string) {
	if !s.span.IsRecording() {
		return
	}
	if ok {
		s.span.SetStatus(codes.Ok, desc)
	} else {
		s.span.SetStatus(codes.Error, desc)
	}
}

// toOtelAttrs 將 []entity.SpanAttr 轉換為 []attribute.KeyValue。
func toOtelAttrs(attrs []entity.SpanAttr) []attribute.KeyValue {
	kvs := make([]attribute.KeyValue, 0, len(attrs))
	for _, a := range attrs {
		switch v := a.Value.(type) {
		case string:
			kvs = append(kvs, attribute.String(a.Key, v))
		case int:
			kvs = append(kvs, attribute.Int(a.Key, v))
		case int64:
			kvs = append(kvs, attribute.Int64(a.Key, v))
		case bool:
			kvs = append(kvs, attribute.Bool(a.Key, v))
		case float64:
			kvs = append(kvs, attribute.Float64(a.Key, v))
		default:
			kvs = append(kvs, attribute.String(a.Key, fmt.Sprintf("%v", v)))
		}
	}
	return kvs
}

// noopSpan 是無操作的 Span 實作，用於 tracing 未啟用或 span 為 nil 的場景。
type noopSpan struct{}

func (n *noopSpan) End()                                  {}
func (n *noopSpan) RecordError(_ error)                   {}
func (n *noopSpan) AddEvent(_ string, _ ...entity.SpanAttr) {}
func (n *noopSpan) SetStatus(_ bool, _ string)            {}
```

注意：需要在 `span.go` import `"fmt"`。

**Step 2: Commit**

```bash
git add internal/infrastructure/tracing/span.go
git commit -m "feat(infra): add otelSpan wrapper implementing domain entity.Span"
```

---

## Task 7: Infrastructure — 更新 TracingService 實作

**Files:**
- Modify: `internal/infrastructure/tracing/tracing.go`

**Step 1: 更新所有方法簽章使用 `entity.Span` 並移除 OTel 型別**

在 `tracing.go` 中做以下修改：

1. import 新增 `entity` 套件，移除 `attribute`、`codes` 從公開方法的使用：
   ```go
   import (
       ...
       "github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
       "go.opentelemetry.io/otel/attribute"   // 保留，供內部使用
       "go.opentelemetry.io/otel/codes"       // 保留，供內部使用
       "go.opentelemetry.io/otel/trace"       // 保留，供內部使用
   )
   ```

2. 更新 `StartSpan` — 移除 `opts ...trace.SpanStartOption`，回傳 `entity.Span`：
   ```go
   func (s *TracingService) StartSpan(ctx context.Context, spanName string) (context.Context, entity.Span) {
       ctx, span := s.tracer.Start(ctx, spanName)
       return ctx, newOtelSpan(span)
   }
   ```

3. 更新 `SpanEnd`：
   ```go
   func (s *TracingService) SpanEnd(span entity.Span) {
       if span != nil {
           span.End()
       }
   }
   ```

4. 更新 `RecordSpanError`：
   ```go
   func (s *TracingService) RecordSpanError(span entity.Span, err error) {
       if span != nil {
           span.RecordError(err)
       }
   }
   ```

5. 更新 `RecordSpanAttributes`：
   ```go
   func (s *TracingService) RecordSpanAttributes(span entity.Span, attrs ...entity.SpanAttr) {
       if span != nil {
           span.AddEvent("attributes", attrs...)
           // 注意：OTel SetAttributes 需在 otelSpan 層處理
           // 改用 otelSpan 內部的 SetAttributes：
       }
   }
   ```

   **重要**：`RecordSpanAttributes` 實際上應呼叫 `span.SetAttributes`（OTel 的屬性設定），而非 AddEvent。需在 `entity.Span` interface 或 `otelSpan` 中處理。修改方式：在 `otelSpan` 新增內部 helper，然後讓 `TracingService.RecordSpanAttributes` 透過 type assertion 取得 `otelSpan`：

   ```go
   func (s *TracingService) RecordSpanAttributes(span entity.Span, attrs ...entity.SpanAttr) {
       if span == nil {
           return
       }
       if os, ok := span.(*otelSpan); ok && os.span.IsRecording() {
           os.span.SetAttributes(toOtelAttrs(attrs)...)
       }
   }
   ```

6. 更新 `TraceEvent`：
   ```go
   func (s *TracingService) TraceEvent(span entity.Span, name string, attrs ...entity.SpanAttr) {
       if span != nil {
           span.AddEvent(name, attrs...)
       }
   }
   ```

7. 更新 `RecordSpanStatus` — 使用 `ok bool` 取代 `codes.Code`：
   ```go
   func (s *TracingService) RecordSpanStatus(span entity.Span, ok bool, desc string) {
       if span != nil {
           span.SetStatus(ok, desc)
       }
   }
   ```

8. 更新 `TraceWorkerToKDS`、`TraceRedisToWorker`、`TraceWorkerProcessing`，回傳 `entity.Span`：
   ```go
   func (s *TracingService) TraceWorkerToKDS(ctx context.Context, eventType, eventID string) (context.Context, entity.Span) {
       ctx, span := s.StartSpan(ctx, "Worker.PublishToKDS")
       s.RecordSpanAttributes(span,
           entity.StringAttr("messaging.system", "kds"),
           entity.StringAttr("messaging.operation", "publish"),
           entity.StringAttr("messaging.event_type", eventType),
           entity.StringAttr("messaging.event_id", eventID))
       return ctx, span
   }

   func (s *TracingService) TraceRedisToWorker(ctx context.Context, taskType, taskID string) (context.Context, entity.Span) {
       ctx, span := s.StartSpan(ctx, "Redis.WorkerConsume")
       s.RecordSpanAttributes(span,
           entity.StringAttr("messaging.system", "redis"),
           entity.StringAttr("messaging.destination", "worker"),
           entity.StringAttr("messaging.task_type", taskType),
           entity.StringAttr("messaging.task_id", taskID))
       return ctx, span
   }

   func (s *TracingService) TraceWorkerProcessing(ctx context.Context, taskType, taskID string) (context.Context, entity.Span) {
       ctx, span := s.StartSpan(ctx, "Worker.ProcessTask")
       s.RecordSpanAttributes(span,
           entity.StringAttr("processing.task_type", taskType),
           entity.StringAttr("processing.task_id", taskID))
       return ctx, span
   }
   ```

**Step 2: 確認此檔案編譯**

```bash
go build ./internal/infrastructure/tracing/...
```

**Step 3: Commit**

```bash
git add internal/infrastructure/tracing/tracing.go
git commit -m "refactor(infra): update TracingService impl to use entity.Span"
```

---

## Task 8: Infrastructure — Redis Manager 新增 BatchSet/BatchDelete

**Files:**
- Modify: `internal/infrastructure/cache/redis/manager.go`

**Step 1: 新增 `BatchSet` 和 `BatchDelete` 方法**

在 `manager.go` 末尾新增：

```go
// BatchSet 使用 Pipeline 批次寫入多個 key-value 對。
func (m *Manager) BatchSet(ctx context.Context, entries []entity.CacheSetEntry, ttl time.Duration) error {
    client, err := m.GetClient()
    if err != nil {
        return fmt.Errorf("failed to get redis client for batch set: %w", err)
    }
    pipe := client.Pipeline()
    for _, e := range entries {
        pipe.Set(ctx, e.Key, e.Value, ttl)
    }
    _, err = pipe.Exec(ctx)
    return err
}

// BatchDelete 使用 Pipeline 批次刪除多個 key。
func (m *Manager) BatchDelete(ctx context.Context, keys []string) error {
    if len(keys) == 0 {
        return nil
    }
    client, err := m.GetClient()
    if err != nil {
        return fmt.Errorf("failed to get redis client for batch delete: %w", err)
    }
    pipe := client.Pipeline()
    for _, key := range keys {
        pipe.Del(ctx, key)
    }
    _, err = pipe.Exec(ctx)
    return err
}
```

同時在 import 中確保 `entity` 套件已引入：
```go
"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
```

**Step 2: 確認編譯時介面合規斷言仍成立**

第 19 行已有 `var _ infrastructure.CacheManager = (*Manager)(nil)` — 此行在 compile 時會驗證 `Manager` 是否完整實作新的 `CacheManager` 介面。

```bash
go build ./internal/infrastructure/cache/redis/...
```

Expected: 成功（`Pipeline`/`GetClient`/`GetMutex` 等方法仍存在於 `Manager` struct 上作為內部方法，不需要移除）。

**Step 3: Commit**

```bash
git add internal/infrastructure/cache/redis/manager.go
git commit -m "feat(infra): add BatchSet and BatchDelete to Redis Manager"
```

---

## Task 9: Infrastructure — 新增 Redis DistributedLockService

**Files:**
- Create: `internal/infrastructure/cache/redis/lock.go`

**Step 1: 建立 `lock.go`**

```go
package redis

import (
    "time"

    "github.com/go-redsync/redsync/v4"
    "github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
)

// 確保編譯時介面合規
var _ infrastructure.DistributedLockService = (*RedisLockService)(nil)

// RedisLockService 實作 infrastructure.DistributedLockService，使用 redsync。
type RedisLockService struct {
    manager *Manager
}

// NewRedisLockService 建立 RedisLockService 實例。
func NewRedisLockService(manager *Manager) *RedisLockService {
    return &RedisLockService{manager: manager}
}

func (s *RedisLockService) GetLock(key string, ttl time.Duration) (infrastructure.DistributedMutex, error) {
    rs, err := s.manager.GetRedsync()
    if err != nil {
        return nil, err
    }
    return rs.NewMutex(key, redsync.WithExpiry(ttl)), nil
}

func (s *RedisLockService) GetLockWithOptions(key string, opts infrastructure.LockOptions) (infrastructure.DistributedMutex, error) {
    rs, err := s.manager.GetRedsync()
    if err != nil {
        return nil, err
    }
    rsOpts := []redsync.Option{redsync.WithExpiry(opts.Expiry)}
    if opts.Tries > 0 {
        rsOpts = append(rsOpts, redsync.WithTries(opts.Tries))
    }
    if opts.RetryDelay > 0 {
        rsOpts = append(rsOpts, redsync.WithRetryDelay(opts.RetryDelay))
    }
    return rs.NewMutex(key, rsOpts...), nil
}
```

**Step 2: 確認編譯**

```bash
go build ./internal/infrastructure/cache/redis/...
```

**Step 3: Commit**

```bash
git add internal/infrastructure/cache/redis/lock.go
git commit -m "feat(infra): add RedisLockService implementing DistributedLockService"
```

---

## Task 10: Infrastructure — 更新 QueueService 移除 WrapHandlerWithTracing

**Files:**
- Modify: `internal/infrastructure/queue/queue.go`

**Step 1: 移除 `WrapHandlerWithTracing` 方法及其 import**

從 `queue.go` 刪除整個 `WrapHandlerWithTracing` 方法（約第 195–240 行）。

同時移除不再需要的 import：
- `"go.opentelemetry.io/otel/attribute"` （若其他地方仍用到則保留）
- 確認 `"github.com/hibiken/asynq"` 仍被其他方法使用

**Step 2: 確認 QueueService struct 仍實作 service.QueueService**

在 `queue.go` 加入（若尚無）：
```go
var _ service.QueueService = (*QueueService)(nil)
```

**Step 3: 確認編譯**

```bash
go build ./internal/infrastructure/queue/...
```

**Step 4: Commit**

```bash
git add internal/infrastructure/queue/queue.go
git commit -m "refactor(infra): remove WrapHandlerWithTracing from QueueService"
```

---

## Task 11: Infrastructure — 更新 utils/helper.go 使用 DistributedLockService

**Files:**
- Modify: `internal/infrastructure/utils/helper.go`

**Step 1: 更新 `ExecuteWithLock` 函數簽章**

移除 `cache infrastructure.CacheManager` 參數，改為 `lockService infrastructure.DistributedLockService`。移除 `redsync` 和 `redis` imports。

新函數：
```go
package utils

import (
    "context"
    "fmt"
    "time"

    "github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
)

// ExecuteWithLock 使用分佈式鎖執行函數的共用方法。
// 使用智能重試機制：3 次嘗試，每次遞增鎖配置參數。
func ExecuteWithLock(
    ctx context.Context,
    lockService infrastructure.DistributedLockService,
    logger infrastructure.Logger,
    mutexKey string,
    entityID uint64,
    entityType string,
    fn func() error,
) error {
    maxAttempts := 3
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        expiry := time.Duration(10+attempt*5) * time.Second
        tries := 5 + attempt*2
        baseDelay := time.Duration(50*attempt) * time.Millisecond

        mutex, err := lockService.GetLockWithOptions(mutexKey, infrastructure.LockOptions{
            Expiry:     expiry,
            Tries:      tries,
            RetryDelay: baseDelay,
        })
        if err != nil {
            logger.ErrorWithContext(ctx, "Failed to create mutex",
                logger.String("mutex_key", mutexKey),
                logger.String("entity_type", entityType),
                logger.UInt64("entity_id", entityID),
                logger.Int("attempt", attempt),
                logger.Error("err", err),
            )
            if attempt < maxAttempts {
                backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
                time.Sleep(backoffTime)
                continue
            }
            return fmt.Errorf("failed to create mutex for %s %d after %d attempts: %w",
                entityType, entityID, maxAttempts, err)
        }

        if err = mutex.Lock(); err != nil {
            logger.WarnWithContext(ctx, "Failed to acquire lock",
                logger.String("mutex_key", mutexKey),
                logger.String("entity_type", entityType),
                logger.UInt64("entity_id", entityID),
                logger.Int("attempt", attempt),
                logger.String("expiry", expiry.String()),
                logger.Int("tries", tries),
                logger.Error("err", err),
            )
            if attempt < maxAttempts {
                backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
                logger.InfoWithContext(ctx, "Retrying after backoff",
                    logger.String("entity_type", entityType),
                    logger.UInt64("entity_id", entityID),
                    logger.String("backoff_time", backoffTime.String()),
                    logger.Int("next_attempt", attempt+1),
                )
                time.Sleep(backoffTime)
                continue
            }
            return fmt.Errorf("failed to acquire lock for %s %d after %d attempts: %w",
                entityType, entityID, maxAttempts, err)
        }

        logger.InfoWithContext(ctx, "Successfully acquired lock",
            logger.String("entity_type", entityType),
            logger.UInt64("entity_id", entityID),
            logger.Int("attempt", attempt),
            logger.String("expiry", expiry.String()),
            logger.Int("tries", tries),
        )

        defer func() {
            ok, unlockErr := mutex.Unlock()
            if !ok || unlockErr != nil {
                logger.ErrorWithContext(ctx, "Failed to unlock mutex",
                    logger.String("mutex_key", mutexKey),
                    logger.String("entity_type", entityType),
                    logger.UInt64("entity_id", entityID),
                    logger.Error("err", unlockErr),
                    logger.Bool("unlock_success", ok),
                )
            }
        }()

        return fn()
    }
    return fmt.Errorf("failed to execute with lock for %s %d after %d attempts",
        entityType, entityID, maxAttempts)
}
```

注意：`QueryWithCache` 函數保持不變（不使用 lock，只使用 `cache.Get`/`Set`）。

**Step 2: 確認編譯**

```bash
go build ./internal/infrastructure/utils/...
```

**Step 3: Commit**

```bash
git add internal/infrastructure/utils/helper.go
git commit -m "refactor(infra): replace GetMutexWithOption with DistributedLockService in ExecuteWithLock"
```

---

## Task 12: Infrastructure — 更新 KDS Consumer 使用 DistributedLockService + BatchSet

**Files:**
- Modify: `internal/infrastructure/kds/consumer.go`

**Step 1: 更新 `KDSService` struct 和 constructor**

找到 `KDSService` struct 定義，新增 `lockService infrastructure.DistributedLockService` 欄位（保留 `redisManager`）。

更新 `NewKDSService`（或相關 constructor）接受 `lockService infrastructure.DistributedLockService` 參數。

**Step 2: 更新 `acquireShardLock`**

```go
func (k *KDSService) acquireShardLock(ctx context.Context, shardId string) infrastructure.DistributedMutex {
    mutexKey := fmt.Sprintf(consts.ShardMutexRedisKey, k.consumeStream, shardId)
    mutex, mutexErr := k.lockService.GetLock(mutexKey, k.config.Consumer.ShardLockTimeout)
    if mutexErr != nil {
        k.logger.WarnWithContext(ctx, "Failed to create mutex, skipping lock logic",
            k.logger.String("mutex_key", mutexKey),
            k.logger.Error("err", mutexErr))
        return nil
    }
    if lockErr := mutex.Lock(); lockErr != nil {
        k.logger.WarnWithContext(ctx, "Shard already locked, skipping",
            k.logger.String("mutex_key", mutexKey),
            k.logger.Error("error", lockErr))
        return nil
    }
    return mutex
}
```

更新呼叫 `acquireShardLock` 的地方（釋放鎖從 `mutex.Unlock()` → `lockMutex.Unlock()`，型別從 `*redsync.Mutex` → `infrastructure.DistributedMutex`）。

**Step 3: 更新 `batchMarkEventsProcessed` 使用 `BatchSet`**

```go
func (k *KDSService) batchMarkEventsProcessed(ctx context.Context, eventIDs []string) {
    if len(eventIDs) == 0 {
        return
    }
    entries := make([]entity.CacheSetEntry, 0, len(eventIDs))
    for _, eventID := range eventIDs {
        if eventID != "" {
            entries = append(entries, entity.CacheSetEntry{
                Key:   processedEventKeyPrefix + eventID,
                Value: "1",
            })
        }
    }
    if err := k.redisManager.BatchSet(ctx, entries, eventProcessedTTL); err != nil {
        k.logger.WarnWithContext(ctx, "Failed to batch mark events as processed",
            k.logger.Error("err", err))
    }
}
```

**Step 4: 移除 `redsync` import（若不再使用）**

**Step 5: 確認 KDS 相關 `trace.Span` 變數型別更新**

在 `consumer.go` 和 `producer.go` 中，所有 `span trace.Span` 區域變數改為 `span entity.Span`。所有 `attribute.String/Int(...)` 呼叫改為 `entity.StringAttr/IntAttr(...)`。移除 `otel/attribute` import（若仍需 OTel 內部使用則保留，但通常 infrastructure 層可繼續引用）。

**Step 6: 確認編譯**

```bash
go build ./internal/infrastructure/kds/...
```

**Step 7: Commit**

```bash
git add internal/infrastructure/kds/consumer.go internal/infrastructure/kds/producer.go
git commit -m "refactor(infra): update KDS to use DistributedLockService and BatchSet"
```

---

## Task 13: Infrastructure — 更新其他 infrastructure 檔案的 span 型別

**Files:**
- Modify: `internal/infrastructure/cache/redis/cleanup_service.go`
- Modify: `internal/infrastructure/queue/queue.go`（`WrapHandlerWithTracing` 已在 Task 10 移除，此步驟更新剩餘的 span 型別）

**Step 1: 更新 `cleanup_service.go`**

將 `span trace.Span` → `span entity.Span`，`attribute.String(...)` → `entity.StringAttr(...)`，`attribute.Int(...)` → `entity.IntAttr(...)`。

**Step 2: 確認編譯**

```bash
go build ./internal/infrastructure/...
```

**Step 3: Commit**

```bash
git add internal/infrastructure/cache/redis/cleanup_service.go
git commit -m "refactor(infra): update cleanup_service span types to entity.Span"
```

---

## Task 14: Application — 更新所有 Use Cases

**Files:**
- Modify: `internal/application/usecase/agent/agent_usecase.go`
- Modify: `internal/application/usecase/manager/manager_usecase.go`
- Modify: `internal/application/usecase/merchant/merchant_usecase.go`
- Modify: `internal/application/usecase/player/player_usecase.go`
- Modify: `internal/application/usecase/player/batch_processor.go`
- Modify: `internal/application/usecase/level/level_usecase.go`
- Modify: `internal/application/usecase/tag/tag_usecase.go`
- Modify: `internal/application/usecase/failed_task_event/failed_task_event_usecase.go`

**每個 use case 的變更模式相同：**

1. 移除 import `"go.opentelemetry.io/otel/attribute"` 和 `"go.opentelemetry.io/otel/codes"`
2. 移除 import `"github.com/redis/go-redis/v9"`（若只因 Pipeline 而引入）
3. 將所有 `span trace.Span` 區域變數改為 `span entity.Span`（加入 `entity` import）
4. 將 `attribute.String(key, val)` → `entity.StringAttr(key, val)`
5. 將 `attribute.Int(key, val)` → `entity.IntAttr(key, val)`
6. 將 `attribute.Int64(key, val)` → `entity.Int64Attr(key, val)`
7. 將 `attribute.Bool(key, val)` → `entity.BoolAttr(key, val)`
8. 將 `RecordSpanStatus(span, codes.Ok, "")` → `RecordSpanStatus(span, true, "")`
9. 將 `RecordSpanStatus(span, codes.Error, msg)` → `RecordSpanStatus(span, false, msg)`
10. `StartSpan(ctx, name, opts...)` → `StartSpan(ctx, name)`（移除 opts 參數）

**tag_usecase.go 額外變更：**

a. 新增 `lockService infrastructure.DistributedLockService` 欄位到 `TagUseCase` struct

b. 更新 `NewTagUseCase` constructor 加入 `lockService` 參數

c. 在 `updateTagCache` 中，將：
   ```go
   pipeline, err := u.cache.Pipeline()
   ...
   pipeline.Set(ctx, cacheKey, string(tagData), 10*time.Minute)
   ...
   _, err = pipeline.Exec(ctx)
   ```
   改為：
   ```go
   entries := make([]entity.CacheSetEntry, 0, len(tags))
   for _, tag := range tags {
       cacheKey := fmt.Sprintf(consts.RedisTagGlobalIDKey, tag.GetGlobalTagID())
       tagData, err := json.Marshal(tag)
       if err != nil { ... continue }
       entries = append(entries, entity.CacheSetEntry{Key: cacheKey, Value: string(tagData)})
   }
   if err := u.cache.BatchSet(ctx, entries, 10*time.Minute); err != nil {
       u.logger.ErrorWithContext(ctx, "Failed to batch update tag cache", ...)
       u.updateTagCacheFallback(ctx, tags)
   }
   ```

d. 更新 `ExecuteWithLock` 呼叫：
   ```go
   // 舊：utils.ExecuteWithLock(ctx, u.cache, u.logger, ...)
   // 新：
   utils.ExecuteWithLock(ctx, u.lockService, u.logger, mutexKey, player.GetID(), "player_tags", func() error { ... })
   ```

**batch_processor.go 額外變更：**

在 `batchInvalidateCache` 中，將 Pipeline 操作改為 `BatchDelete`：
```go
func (p *PlayerBatchProcessor) batchInvalidateCache(ctx context.Context, players []*entity.Player) error {
    if len(players) == 0 {
        return nil
    }
    keys := make([]string, 0, len(players))
    for _, player := range players {
        keys = append(keys, fmt.Sprintf(consts.RedisPlayerGlobalIDKey, player.GetGlobalPlayerID()))
    }
    if err := p.cache.BatchDelete(ctx, keys); err != nil {
        p.logger.WarnWithContext(ctx, "Batch cache deletion failed, falling back",
            p.logger.Error("error", err))
        return p.fallbackInvalidateCache(ctx, players)
    }
    p.logger.InfoWithContext(ctx, "Batch cache invalidation completed",
        p.logger.Int("cache_keys_deleted", len(keys)))
    return nil
}
```

**Step 1: 逐一修改每個 use case 檔案，每改完一個即可嘗試 build**

```bash
go build ./internal/application/...
```

**Step 2: Commit（可一次 commit 所有 use case 變更）**

```bash
git add internal/application/
git commit -m "refactor(usecase): replace OTel/Redis types with domain entity types"
```

---

## Task 15: Adapter — 更新 worker_handler.go

**Files:**
- Modify: `internal/adapter/inbound/handler/worker/worker_handler.go`

**Step 1: 移除 `WrapHandlerWithTracing` 使用，新增 `wrapWithTracing` 私有方法**

在 `WorkerHandler` struct 中確保持有 `tracing infrastructure.TracingService`（應已存在）。

新增私有方法（取代 `queueService.WrapHandlerWithTracing`）：

```go
// wrapWithTracing 從 asynq 任務 payload 中提取 trace context，並建立追蹤 span。
func (h *WorkerHandler) wrapWithTracing(handler asynq.Handler) asynq.Handler {
    return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
        if task == nil || len(task.Payload()) == 0 || task.Type() == "" {
            return asynq.SkipRetry
        }
        data := task.Payload()
        ctxWithTrace := h.tracing.ExtractTraceContext(ctx, data)
        ctxWithTrace, span := h.tracing.TraceRedisToWorker(
            ctxWithTrace,
            task.Type(),
            task.ResultWriter().TaskID(),
        )
        defer h.tracing.SpanEnd(span)
        h.tracing.TraceEvent(span, "Starting worker task processing")
        h.tracing.RecordSpanAttributes(span, entity.IntAttr("task.payload_size_bytes", len(data)))

        var jsonData map[string]interface{}
        if err := json.Unmarshal(data, &jsonData); err == nil {
            if id, ok := jsonData["id"].(string); ok {
                h.tracing.RecordSpanAttributes(span, entity.StringAttr("messaging.event_id", id))
            }
        }
        return handler.ProcessTask(ctxWithTrace, task)
    })
}
```

**Step 2: 更新 `RegisterHandlers` 使用 `h.wrapWithTracing` 替代 `h.queueService.WrapHandlerWithTracing`**

```go
mux.Handle(queue.TypeMerchantSync, h.wrapWithTracing(asynq.HandlerFunc(h.HandleMerchantSync)))
// ... 同樣處理其他 6 個
```

**Step 3: 更新所有 handler 方法的 `span trace.Span` → `span entity.Span`，`attribute.*` → `entity.*Attr`**

**Step 4: 更新 `tracing_middleware.go`**

同樣更新 `attribute.String(...)` → `entity.StringAttr(...)`，`span trace.Span` → `span entity.Span`。

**Step 5: 確認編譯**

```bash
go build ./internal/adapter/...
```

**Step 6: Commit**

```bash
git add internal/adapter/
git commit -m "refactor(adapter): use entity.Span and add wrapWithTracing to WorkerHandler"
```

---

## Task 16: cmd 層 — 更新 StartSpan/TraceEvent 呼叫

**Files:**
- Modify: `cmd/web/web.go`
- Modify: `cmd/worker/worker.go`（若有）
- Modify: `cmd/consumer/consumer.go`（若有）

**Step 1: 更新 `cmd/web/web.go`**

找到所有 `tracingService.StartSpan(...)` 呼叫，移除第三個 opts 參數（若有）。
`tracingService.TraceEvent(rootSpan, ...)` 的 `rootSpan` 型別從 `trace.Span` → `entity.Span`（加入 import）。

**Step 2: 確認編譯**

```bash
go build ./cmd/...
```

**Step 3: Commit**

```bash
git add cmd/
git commit -m "refactor(cmd): update span types to entity.Span in service entrypoints"
```

---

## Task 17: DI — 更新 wire.go 加入 DistributedLockService 和修正 provideRedisClient

**Files:**
- Modify: `internal/di/wire.go`

**Step 1: 修改所有 Initialize 函數，接受 `*redisCache.Manager` 取代 `infrastructure.CacheManager`**

新增 import：
```go
redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
```

更新 `InitializeWebServer`、`InitializeWorkerServer`、`InitializeWorkerComponents`、`InitializeConsumerHandler` 的參數：
```go
func InitializeWebServer(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*api.HTTPHandler, error) {
    wire.Build(
        wire.Bind(new(infrastructure.CacheManager), new(*redisCache.Manager)),
        baseSet,
        kds.NewKDSService,
        api.NewHTTPHandler,
    )
    return nil, nil
}
```
（其他三個 Initialize 函數做相同修改）

**Step 2: 更新 `provideRedisClient`**

```go
func provideRedisClient(manager *redisCache.Manager) (*redis.Client, error) {
    return manager.GetClient()
}
```

**Step 3: 新增 `provideDistributedLockService` 並加入 `baseSet`**

```go
func provideDistributedLockService(manager *redisCache.Manager) infrastructure.DistributedLockService {
    return redisCache.NewRedisLockService(manager)
}
```

在 `baseSet` 中新增：
```go
provideDistributedLockService,
```

**Step 4: 更新 `provideWorkerServer` 若參數改變**

若 `queue.NewWorkerServer` 簽章變更，同步更新。

**Step 5: 更新 `cmd/web/web.go`、`cmd/worker/worker.go`、`cmd/consumer/consumer.go` 傳入 `*redis.Manager` 而非介面**

這些 cmd 檔案本就有 `*redis.Manager` 實例，只需不做型別斷言直接傳入。

**Step 6: 確認編譯**

```bash
go build ./internal/di/... ./cmd/...
```

**Step 7: Commit**

```bash
git add internal/di/wire.go cmd/
git commit -m "refactor(di): add DistributedLockService provider and use concrete Manager type"
```

---

## Task 18: 更新所有 Mocks

**Files:**
- Modify: `test/mocks/cache_manager_mock.go`
- Modify: `test/mocks/nil_tracing_mock.go`
- Modify: `test/mocks/tracing_service_mock.go`
- Modify: `test/mocks/service_mocks.go`（QueueService mock 移除 WrapHandlerWithTracing）

**Step 1: 更新 `cache_manager_mock.go`**

移除所有 Redis 型別方法（`Pipeline`、`GetClient`、`GetMutex`、`GetMutexWithOption`、`GetRedsync`）。
新增：
```go
func (n *NilCacheManager) BatchSet(ctx context.Context, entries []entity.CacheSetEntry, ttl time.Duration) error {
    return nil
}
func (n *NilCacheManager) BatchDelete(ctx context.Context, keys []string) error {
    return nil
}
```
移除 `redsync` 和 `redis` imports。

**Step 2: 更新 `nil_tracing_mock.go`**

將所有方法更新為使用 `entity.Span`，移除 OTel imports：
```go
func (n *NilTracingService) StartSpan(ctx context.Context, spanName string) (context.Context, entity.Span) {
    return ctx, &noopEntitySpan{}
}
func (n *NilTracingService) SpanEnd(span entity.Span) {}
func (n *NilTracingService) RecordSpanError(span entity.Span, err error) {}
func (n *NilTracingService) RecordSpanAttributes(span entity.Span, attrs ...entity.SpanAttr) {}
func (n *NilTracingService) TraceEvent(span entity.Span, name string, attrs ...entity.SpanAttr) {}
func (n *NilTracingService) RecordSpanStatus(span entity.Span, ok bool, desc string) {}
// TraceWorkerToKDS, TraceRedisToWorker, TraceWorkerProcessing 同樣更新
```

新增 `noopEntitySpan`（實作 `entity.Span` 的 no-op struct）。

**Step 3: 更新 `tracing_service_mock.go`**

同樣更新所有方法簽章使用 `entity.Span`。

**Step 4: 更新 `service_mocks.go`**

移除 `QueueServiceMock.WrapHandlerWithTracing` 方法及相關 `asynq` import。

**Step 5: 確認編譯**

```bash
go build ./test/...
```

**Step 6: Commit**

```bash
git add test/mocks/
git commit -m "refactor(mocks): update mocks to use entity.Span and new CacheManager interface"
```

---

## Task 19: 重新產生 wire_gen.go 並全面驗證

**Step 1: 重新產生 wire**

```bash
cd /Users/winstonlin/go/src/fatcat/fat-identity-cat && go generate ./internal/di/...
# 或直接執行 wire：
wire ./internal/di/
```

**Step 2: 全面建置**

```bash
go build ./...
```

Expected: 0 errors。

**Step 3: 執行現有測試**

```bash
go test ./... 2>&1 | tail -30
```

Expected: 所有原本通過的測試仍通過（integration tests 若需要 DB 連線可跳過）。

**Step 4: 最終 commit**

```bash
git add internal/di/wire_gen.go
git commit -m "chore: regenerate wire_gen.go after domain port refactoring"
```
