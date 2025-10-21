package infrastructure

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracingService 定義分散式追蹤服務的抽象介面
//
//go:generate mockery --name=TracingService --output=../../../../../../test/mocks --outpkg=mocks
type TracingService interface {
	// StartSpan 開始一個新的span
	StartSpan(
		ctx context.Context,
		spanName string,
		opts ...trace.SpanStartOption,
	) (context.Context, trace.Span)

	// RecordSpanError 記錄span錯誤
	RecordSpanError(span trace.Span, err error)

	// RecordSpanAttributes 記錄span屬性
	RecordSpanAttributes(span trace.Span, attrs ...attribute.KeyValue)

	// TraceEvent 追蹤事件
	TraceEvent(span trace.Span, name string, attrs ...attribute.KeyValue)

	// SpanEnd 結束span
	SpanEnd(span trace.Span)

	// GetTraceparent 從上下文中獲取traceparent
	GetTraceparent(ctx context.Context) string

	// InjectTraceparentToJSON 將traceparent注入到JSON數據中
	InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error)

	// RecordSpanStatus 記錄span狀態
	RecordSpanStatus(span trace.Span, code codes.Code, desc string)

	// TraceWorkerToKDS 從Worker到KDS的追蹤封裝
	TraceWorkerToKDS(ctx context.Context, eventType, eventID string) (context.Context, trace.Span)

	// ExtractTraceContext 從數據中提取追蹤上下文
	ExtractTraceContext(ctx context.Context, carrier []byte) context.Context

	// TraceRedisToWorker 從Redis到Worker的追蹤封裝
	TraceRedisToWorker(ctx context.Context, taskType, taskID string) (context.Context, trace.Span)

	// TraceWorkerProcessing Worker處理任務的追蹤封裝
	TraceWorkerProcessing(
		ctx context.Context,
		taskType, taskID string,
	) (context.Context, trace.Span)
}
