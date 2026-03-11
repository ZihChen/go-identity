package mocks

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/stretchr/testify/mock"
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

func (m *TracingServiceMock) StartSpan(ctx context.Context, spanName string) (context.Context, entity.Span) {
	args := m.Called(ctx, spanName)
	return args.Get(0).(context.Context), args.Get(1).(entity.Span)
}
func (m *TracingServiceMock) SpanEnd(span entity.Span) { m.Called(span) }
func (m *TracingServiceMock) RecordSpanError(span entity.Span, err error) { m.Called(span, err) }
func (m *TracingServiceMock) RecordSpanAttributes(span entity.Span, attrs ...entity.SpanAttr) {
	m.Called(span, attrs)
}
func (m *TracingServiceMock) TraceEvent(span entity.Span, name string, attrs ...entity.SpanAttr) {
	m.Called(span, name, attrs)
}
func (m *TracingServiceMock) RecordSpanStatus(span entity.Span, ok bool, desc string) {
	m.Called(span, ok, desc)
}
func (m *TracingServiceMock) GetTraceparent(ctx context.Context) string {
	args := m.Called(ctx)
	return args.String(0)
}
func (m *TracingServiceMock) InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error) {
	args := m.Called(ctx, data)
	return args.Get(0).([]byte), args.Error(1)
}
func (m *TracingServiceMock) ExtractTraceContext(ctx context.Context, carrier []byte) context.Context {
	args := m.Called(ctx, carrier)
	return args.Get(0).(context.Context)
}
func (m *TracingServiceMock) TraceWorkerToKDS(ctx context.Context, eventType, eventID string) (context.Context, entity.Span) {
	args := m.Called(ctx, eventType, eventID)
	return args.Get(0).(context.Context), args.Get(1).(entity.Span)
}
func (m *TracingServiceMock) TraceRedisToWorker(ctx context.Context, taskType, taskID string) (context.Context, entity.Span) {
	args := m.Called(ctx, taskType, taskID)
	return args.Get(0).(context.Context), args.Get(1).(entity.Span)
}
func (m *TracingServiceMock) TraceWorkerProcessing(ctx context.Context, taskType, taskID string) (context.Context, entity.Span) {
	args := m.Called(ctx, taskType, taskID)
	return args.Get(0).(context.Context), args.Get(1).(entity.Span)
}

// Ensure mock.Mock is imported to avoid "imported and not used" errors
var _ = mock.Anything
