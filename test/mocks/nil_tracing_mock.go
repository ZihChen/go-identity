package mocks

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// NilTracingService 提供無操作的TracingService實現，用於測試
type NilTracingService struct{}

// NewNilTracingService 創建無操作的TracingService
func NewNilTracingService() *NilTracingService {
	return &NilTracingService{}
}

// StartSpan 無操作實現
func (n *NilTracingService) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

// RecordSpanError 無操作實現
func (n *NilTracingService) RecordSpanError(span trace.Span, err error) {}

// RecordSpanAttributes 無操作實現
func (n *NilTracingService) RecordSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {}

// TraceEvent 無操作實現
func (n *NilTracingService) TraceEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {}

// SpanEnd 無操作實現
func (n *NilTracingService) SpanEnd(span trace.Span) {}

// GetTraceparent 無操作實現
func (n *NilTracingService) GetTraceparent(ctx context.Context) string {
	return ""
}

// InjectTraceparentToJSON 無操作實現
func (n *NilTracingService) InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error) {
	return data, nil
}

// RecordSpanStatus 無操作實現
func (n *NilTracingService) RecordSpanStatus(span trace.Span, code codes.Code, desc string) {}

// TraceWorkerToKDS 無操作實現
func (n *NilTracingService) TraceWorkerToKDS(ctx context.Context, eventType, eventID string) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

// ExtractTraceContext 無操作實現
func (n *NilTracingService) ExtractTraceContext(ctx context.Context, carrier []byte) context.Context {
	return ctx
}

// TraceRedisToWorker 無操作實現
func (n *NilTracingService) TraceRedisToWorker(ctx context.Context, taskType, taskID string) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

// TraceWorkerProcessing 無操作實現
func (n *NilTracingService) TraceWorkerProcessing(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	return ctx, trace.SpanFromContext(ctx)
}

