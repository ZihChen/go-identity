package tracing

import (
	"fmt"

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

// setAttributes sets OTel attributes directly on the underlying span.
// Used internally by TracingService.RecordSpanAttributes.
func (s *otelSpan) setAttributes(attrs ...entity.SpanAttr) {
	if s.span.IsRecording() {
		s.span.SetAttributes(toOtelAttrs(attrs)...)
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

func (n *noopSpan) End()                                    {}
func (n *noopSpan) RecordError(_ error)                     {}
func (n *noopSpan) AddEvent(_ string, _ ...entity.SpanAttr) {}
func (n *noopSpan) SetStatus(_ bool, _ string)              {}
