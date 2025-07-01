// 檔案: tests/openobserve_test.go
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
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
		logger, _ = zap.NewProduction()
	}

	// 測試日誌輸出
	testID := time.Now().Format("20060102150405")
	logger.Info("Test log to OpenObserve",
		zap.String("test_id", testID),
		zap.String("component", "test"),
		zap.String("message", "This is a test log entry for OpenObserve"),
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

	// 初始化追蹤器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		t.Fatalf("Failed to create tracer: %v", err)
	}
	defer func() {
		_ = tracer.Shutdown(context.Background())
	}()

	// 創建測試跟蹤
	ctx, rootSpan := tracing.StartSpan(context.Background(), "TestOpenObserveTracing")
	defer rootSpan.End()

	testID := time.Now().Format("20060102150405")
	rootSpan.SetAttributes(
		attribute.String("test_id", testID),
		attribute.String("component", "test"),
	)

	// 添加事件
	tracing.TraceEvent(rootSpan, "Test event",
		attribute.String("detail", "This is a test event for OpenObserve tracing"),
	)

	// 創建子 span
	_, childSpan := tracing.StartSpan(ctx, "TestOpenObserveTracingChild")
	childSpan.SetAttributes(
		attribute.String("test_id", testID),
		attribute.String("component", "test-child"),
	)
	tracing.TraceEvent(childSpan, "Test child event")
	childSpan.End()

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
