package mocks

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// TracingServiceMock TracingService的mock實現
type TracingServiceMock struct {
	*BaseMock
}

// NewTracingServiceMock 創建新的TracingService mock
func NewTracingServiceMock(t *testing.T) *TracingServiceMock {
	return &TracingServiceMock{
		BaseMock: NewBaseMock(t),
	}
}

// StartSpan 開始一個新的span
func (m *TracingServiceMock) StartSpan(
	ctx context.Context,
	spanName string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	args := m.Called(ctx, spanName, opts)
	return args.Get(0).(context.Context), args.Get(1).(trace.Span)
}

// RecordSpanError 記錄span錯誤
func (m *TracingServiceMock) RecordSpanError(span trace.Span, err error) {
	m.Called(span, err)
}

// RecordSpanAttributes 記錄span屬性
func (m *TracingServiceMock) RecordSpanAttributes(span trace.Span, attrs ...attribute.KeyValue) {
	m.Called(span, attrs)
}

// TraceEvent 追蹤事件
func (m *TracingServiceMock) TraceEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	m.Called(span, name, attrs)
}

// SpanEnd 結束span
func (m *TracingServiceMock) SpanEnd(span trace.Span) {
	m.Called(span)
}

// GetTraceparent 從上下文中獲取traceparent
func (m *TracingServiceMock) GetTraceparent(ctx context.Context) string {
	args := m.Called(ctx)
	return args.String(0)
}

// InjectTraceparentToJSON 將traceparent注入到JSON數據中
func (m *TracingServiceMock) InjectTraceparentToJSON(
	ctx context.Context,
	data []byte,
) ([]byte, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]byte), args.Error(1)
}

// RecordSpanStatus 記錄span狀態
func (m *TracingServiceMock) RecordSpanStatus(span trace.Span, code codes.Code, desc string) {
	m.Called(span, code, desc)
}

// TraceWorkerToKDS 從Worker到KDS的追蹤封裝
func (m *TracingServiceMock) TraceWorkerToKDS(
	ctx context.Context,
	eventType, eventID string,
) (context.Context, trace.Span) {
	args := m.Called(ctx, eventType, eventID)
	return args.Get(0).(context.Context), args.Get(1).(trace.Span)
}

// ExtractTraceContext 從數據中提取追蹤上下文
func (m *TracingServiceMock) ExtractTraceContext(
	ctx context.Context,
	carrier []byte,
) context.Context {
	args := m.Called(ctx, carrier)
	return args.Get(0).(context.Context)
}

// TraceRedisToWorker 從Redis到Worker的追蹤封裝
func (m *TracingServiceMock) TraceRedisToWorker(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	args := m.Called(ctx, taskType, taskID)
	return args.Get(0).(context.Context), args.Get(1).(trace.Span)
}

// TraceWorkerProcessing Worker處理任務的追蹤封裝
func (m *TracingServiceMock) TraceWorkerProcessing(
	ctx context.Context,
	taskType, taskID string,
) (context.Context, trace.Span) {
	args := m.Called(ctx, taskType, taskID)
	return args.Get(0).(context.Context), args.Get(1).(trace.Span)
}
