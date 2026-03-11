package entity

import "fmt"

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

func StringAttr(key, value string) SpanAttr          { return SpanAttr{Key: key, Value: value} }
func IntAttr(key string, value int) SpanAttr         { return SpanAttr{Key: key, Value: value} }
func Int64Attr(key string, value int64) SpanAttr     { return SpanAttr{Key: key, Value: value} }
func BoolAttr(key string, value bool) SpanAttr       { return SpanAttr{Key: key, Value: value} }
func Float64Attr(key string, value float64) SpanAttr { return SpanAttr{Key: key, Value: value} }

// AnyAttr creates a SpanAttr with any value type.
func AnyAttr(key string, value any) SpanAttr { return SpanAttr{Key: key, Value: value} }

// FormatValue formats the SpanAttr value as a string for display/logging.
func (a SpanAttr) FormatValue() string {
	return fmt.Sprintf("%v", a.Value)
}
