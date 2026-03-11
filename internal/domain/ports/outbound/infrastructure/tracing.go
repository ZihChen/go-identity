package infrastructure

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// TracingService 定義分散式追蹤服務的抽象介面。
// 所有方法只使用標準庫型別或 domain entity 型別，不依賴任何 OTel 具體型別。
//
//go:generate mockery --name=TracingService --output=../../../../../../test/mocks --outpkg=mocks
type TracingService interface {
	// StartSpan 開始一個新的 span，回傳帶有新 span 的 context 以及 domain Span。
	StartSpan(ctx context.Context, spanName string) (context.Context, entity.Span)

	// SpanEnd 結束 span。
	SpanEnd(span entity.Span)

	// RecordSpanError 記錄 span 錯誤。
	RecordSpanError(span entity.Span, err error)

	// RecordSpanAttributes 記錄 span 屬性。
	RecordSpanAttributes(span entity.Span, attrs ...entity.SpanAttr)

	// TraceEvent 為 span 新增事件。
	TraceEvent(span entity.Span, name string, attrs ...entity.SpanAttr)

	// RecordSpanStatus 記錄 span 的最終狀態（ok=true 代表成功）。
	RecordSpanStatus(span entity.Span, ok bool, desc string)

	// GetTraceparent 從 context 中取得 W3C traceparent 字串。
	GetTraceparent(ctx context.Context) string

	// InjectTraceparentToJSON 將 traceparent 注入至 JSON payload 中。
	InjectTraceparentToJSON(ctx context.Context, data []byte) ([]byte, error)

	// ExtractTraceContext 從 JSON payload 中提取 trace context 並注入 context。
	ExtractTraceContext(ctx context.Context, carrier []byte) context.Context

	// TraceWorkerToKDS Worker 發布至 KDS 的便利追蹤包裝。
	TraceWorkerToKDS(ctx context.Context, eventType, eventID string) (context.Context, entity.Span)

	// TraceRedisToWorker Redis 佇列至 Worker 消費的便利追蹤包裝。
	TraceRedisToWorker(ctx context.Context, taskType, taskID string) (context.Context, entity.Span)

	// TraceWorkerProcessing Worker 處理任務的便利追蹤包裝。
	TraceWorkerProcessing(
		ctx context.Context,
		taskType, taskID string,
	) (context.Context, entity.Span)
}
