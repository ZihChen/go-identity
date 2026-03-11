package mocks

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// NilTracingService 提供無操作的TracingService實現，用於測試
type NilTracingService struct{}

// NewNilTracingService 創建無操作的TracingService
func NewNilTracingService() *NilTracingService {
	return &NilTracingService{}
}

func (n *NilTracingService) StartSpan(ctx context.Context, spanName string) (context.Context, entity.Span) {
	return ctx, &nilSpan{}
}
func (n *NilTracingService) SpanEnd(span entity.Span)              {}
func (n *NilTracingService) RecordSpanError(span entity.Span, err error) {}
func (n *NilTracingService) RecordSpanAttributes(span entity.Span, attrs ...entity.SpanAttr) {}
func (n *NilTracingService) TraceEvent(span entity.Span, name string, attrs ...entity.SpanAttr) {}
func (n *NilTracingService) RecordSpanStatus(span entity.Span, ok bool, desc string) {}
func (n *NilTracingService) GetTraceparent(ctx context.Context) string { return "" }
func (n *NilTracingService) InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error) {
	return data, nil
}
func (n *NilTracingService) ExtractTraceContext(ctx context.Context, carrier []byte) context.Context {
	return ctx
}
func (n *NilTracingService) TraceWorkerToKDS(ctx context.Context, eventType, eventID string) (context.Context, entity.Span) {
	return ctx, &nilSpan{}
}
func (n *NilTracingService) TraceRedisToWorker(ctx context.Context, taskType, taskID string) (context.Context, entity.Span) {
	return ctx, &nilSpan{}
}
func (n *NilTracingService) TraceWorkerProcessing(ctx context.Context, taskType, taskID string) (context.Context, entity.Span) {
	return ctx, &nilSpan{}
}

// nilSpan 是無操作的 entity.Span 實作，用於測試
type nilSpan struct{}

func (n *nilSpan) End()                                    {}
func (n *nilSpan) RecordError(_ error)                     {}
func (n *nilSpan) AddEvent(_ string, _ ...entity.SpanAttr) {}
func (n *nilSpan) SetStatus(_ bool, _ string)              {}
