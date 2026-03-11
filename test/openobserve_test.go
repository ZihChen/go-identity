// 檔案: tests/openobserve_test.go
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
)

func TestOpenObserveLogging(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 確保配置中包含 OpenObserve 端點
	if cfg.Tracing.Endpoint == "" {
		t.Skip("Skipping OpenObserve test because OPENOBSERVE_TRACE_API_ENDPOINT is not set")
	}

	// 初始化日誌
	logger := cmd.GetLogger()
	if logger == nil {
		// 如果 cmd.GetLogger() 返回 nil，則手動初始化
		logger = mocks.NewMockLogger(t)
	}

	// 測試日誌輸出
	testID := time.Now().Format("20060102150405")
	logger.InfoLog("Test log to OpenObserve",
		logger.String("test_id", testID),
		logger.String("component", "test"),
		logger.String("message", "This is a test log entry for OpenObserve"),
	)

	// 給日誌一些時間發送到 OpenObserve
	time.Sleep(1 * time.Second)

	// 注意: 我們無法在測試代碼中自動驗證日誌是否真的到達 OpenObserve
	// 這需要手動在 OpenObserve UI 中檢查
	// 以下僅輸出測試日誌的標識符，以便在 UI 中搜尋
	t.Logf(
		"Sent test log with test_id: %s to OpenObserve. Please check the OpenObserve UI.",
		testID,
	)
}

func TestOpenObserveTracing(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 確保配置中包含 OpenObserve 端點
	if cfg.Tracing.Endpoint == "" {
		t.Skip("Skipping OpenObserve test because OPENOBSERVE_TRACE_API_ENDPOINT is not set")
	}

	// 創建 TracingService 實例
	tracingService, err := tracing.NewTracingService(cfg)
	if err != nil {
		t.Fatalf("Failed to create tracing service: %v", err)
	}
	defer func() {
		_ = tracingService.Shutdown(context.Background())
	}()

	// 創建測試跟蹤
	ctx, rootSpan := tracingService.StartSpan(context.Background(), "TestOpenObserveTracing")
	defer tracingService.SpanEnd(rootSpan)

	testID := time.Now().Format("20060102150405")
	tracingService.RecordSpanAttributes(rootSpan,
		entity.StringAttr("test_id", testID),
		entity.StringAttr("component", "test"),
	)

	// 添加事件
	tracingService.TraceEvent(rootSpan, "Test event",
		entity.StringAttr("detail", "This is a test event for OpenObserve tracing"),
	)

	// 創建子 span
	_, childSpan := tracingService.StartSpan(ctx, "TestOpenObserveTracingChild")
	tracingService.RecordSpanAttributes(childSpan,
		entity.StringAttr("test_id", testID),
		entity.StringAttr("component", "test-child"),
	)
	tracingService.TraceEvent(childSpan, "Test child event")
	tracingService.SpanEnd(childSpan)

	// 給追蹤一些時間發送到 OpenObserve
	time.Sleep(1 * time.Second)

	// 注意: 我們無法在測試代碼中自動驗證追蹤是否真的到達 OpenObserve
	// 這需要手動在 OpenObserve UI 中檢查
	// 以下僅輸出測試追蹤的標識符，以便在 UI 中搜尋
	t.Logf(
		"Sent test trace with test_id: %s to OpenObserve. Please check the OpenObserve UI.",
		testID,
	)
}
