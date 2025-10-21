package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	ServiceName = "ms-identity-cat"
)

// TracingService 統一的追踪服務實現，包含provider管理和全局tracer
type TracingService struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// NewTracingService 創建統一的追踪服務實例
func NewTracingService(cfg *config.Config) (*TracingService, error) {
	ctx := context.Background()

	// 創建OTLP導出器
	traceExporter, err := createExporter(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// 創建資源
	res := createResource(cfg)

	// 創建追踪提供者
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)

	// 設置全局追踪提供者
	otel.SetTracerProvider(provider)

	// 設置全局傳播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 創建tracer實例
	tracer := otel.Tracer(ServiceName)

	return &TracingService{
		provider: provider,
		tracer:   tracer,
	}, nil
}

// Shutdown 關閉追踪服務
func (s *TracingService) Shutdown(ctx context.Context) error {
	if s.provider != nil {
		if err := s.provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown trace provider: %w", err)
		}
	}
	return nil
}

// 創建OTLP導出器
func createExporter(ctx context.Context, cfg *config.Config) (*otlptrace.Exporter, error) {
	otlptracehttp.WithEndpointURL(cfg.Tracing.Endpoint)
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.Tracing.Endpoint),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": cfg.Tracing.APIKey,
			"stream-name":   cfg.Tracing.StreamName,
		}),
		otlptracehttp.WithTimeout(30 * time.Second), // 增加超時時間
		otlptracehttp.WithRetry(otlptracehttp.RetryConfig{
			Enabled:         true,
			InitialInterval: 1 * time.Second,
			MaxInterval:     5 * time.Second,
			MaxElapsedTime:  30 * time.Second,
		}),
	}

	client := otlptracehttp.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	return exporter, nil
}

// 創建資源
func createResource(cfg *config.Config) *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(cfg.App.Name),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.DeploymentEnvironmentKey.String(cfg.App.Env),
	)
}

// ExtractTraceparent 從 JSON 字符串中提取 traceparent
func extractTraceparent(jsonStr string) string {
	// 簡單的字符串匹配，實際應用中可能需要更健壯的方式
	traceparentStart := `"traceparent":"`
	traceparentEnd := `"`

	traceparentStartIndex := findIndex(jsonStr, traceparentStart)
	if traceparentStartIndex == -1 {
		return ""
	}

	traceparentStartIndex += len(traceparentStart)
	traceparentEndIndex := findIndex(jsonStr[traceparentStartIndex:], traceparentEnd)
	if traceparentEndIndex == -1 {
		return ""
	}

	return jsonStr[traceparentStartIndex : traceparentStartIndex+traceparentEndIndex]
}

// findIndex 在字符串中查找子字符串的索引
func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func GetTraceID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	return spanCtx.TraceID().String()
}

func GetSpanID(ctx context.Context) string {
	spanCtx := trace.SpanContextFromContext(ctx)
	return spanCtx.SpanID().String()
}

// StartSpan 開始一個新的span
func (s *TracingService) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return s.tracer.Start(ctx, spanName, opts...)
}

// RecordSpanError 記錄span錯誤
func (s *TracingService) RecordSpanError(span trace.Span, err error) {
	if span != nil && span.IsRecording() {
		span.RecordError(err)
	}
}

// RecordSpanAttributes 記錄span屬性
func (s *TracingService) RecordSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	if span != nil && span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// TraceEvent 追蹤事件
func (s *TracingService) TraceEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span != nil && span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attrs...))
	}
}

// SpanEnd 結束span
func (s *TracingService) SpanEnd(span trace.Span) {
	if span != nil {
		span.End()
	}
}

// GetTraceparent 從上下文中獲取traceparent
func (s *TracingService) GetTraceparent(ctx context.Context) string {
	carrier := make(propagation.MapCarrier)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

// InjectTraceparentToJSON 將traceparent注入到JSON數據中
func (s *TracingService) InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error) {
	traceparent := s.GetTraceparent(ctx)
	if traceparent == "" {
		return data, nil
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return data, err
	}

	jsonData["traceparent"] = traceparent

	return json.Marshal(jsonData)
}

// RecordSpanStatus 記錄span狀態
func (s *TracingService) RecordSpanStatus(span trace.Span, code codes.Code, desc string) {
	if span != nil && span.IsRecording() {
		span.SetStatus(code, desc)
	}
}

// TraceWorkerToKDS 從Worker到KDS的追蹤封裝
func (s *TracingService) TraceWorkerToKDS(
	ctx context.Context,
	eventType, eventID string,
) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "Worker.PublishToKDS")
	s.RecordSpanAttributes(span,
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "publish"),
		attribute.String("messaging.event_type", eventType),
		attribute.String("messaging.event_id", eventID))
	return ctx, span
}

// ExtractTraceContext 從數據中提取追蹤上下文
func (s *TracingService) ExtractTraceContext(ctx context.Context, carrier []byte) context.Context {
	// 嘗試解析 JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal(carrier, &jsonData); err == nil {
		// 如果包含 traceparent
		if traceparent, ok := jsonData["traceparent"].(string); ok && traceparent != "" {
			mapCarrier := make(propagation.MapCarrier)
			mapCarrier.Set("traceparent", traceparent)
			return otel.GetTextMapPropagator().Extract(ctx, mapCarrier)
		}
	}

	// 無法解析 JSON 或沒有找到 traceparent，直接解析
	traceparent := extractTraceparent(string(carrier))
	if traceparent != "" {
		mapCarrier := make(propagation.MapCarrier)
		mapCarrier.Set("traceparent", traceparent)
		return otel.GetTextMapPropagator().Extract(ctx, mapCarrier)
	}

	return ctx
}

// TraceRedisToWorker 從Redis到Worker的追蹤封裝
func (s *TracingService) TraceRedisToWorker(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "Redis.WorkerConsume")
	s.RecordSpanAttributes(span,
		attribute.String("messaging.system", "redis"),
		attribute.String("messaging.destination", "worker"),
		attribute.String("messaging.task_type", taskType),
		attribute.String("messaging.task_id", taskID))
	return ctx, span
}

// TraceWorkerProcessing Worker處理任務的追蹤封裝
func (s *TracingService) TraceWorkerProcessing(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	ctx, span := s.StartSpan(ctx, "Worker.ProcessTask")
	s.RecordSpanAttributes(span,
		attribute.String("processing.task_type", taskType),
		attribute.String("processing.task_id", taskID))
	return ctx, span
}
