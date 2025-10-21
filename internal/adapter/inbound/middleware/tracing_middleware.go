package middleware

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
)

// NewTracingMiddleware 創建新的追蹤中間件
func NewTracingMiddleware(tracingService infrastructure.TracingService) gin.HandlerFunc {
	return func(c *gin.Context) {
		propagator := propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := c.Request.Method + " " + c.FullPath()
		ctx, span := tracingService.StartSpan(ctx, spanName)
		defer tracingService.SpanEnd(span)

		tracingService.RecordSpanAttributes(span,
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.path", c.FullPath()),
		)

		ctx = withTraceContext(ctx, tracingService)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		tracingService.RecordSpanAttributes(
			span,
			attribute.Int("http.status_code", c.Writer.Status()),
		)

		if len(c.Errors) > 0 {
			tracingService.RecordSpanError(span, c.Errors.Last().Err)
			fmt.Printf("Trace error %+v", c.Errors)
		}
	}
}

func withTraceContext(
	ctx context.Context,
	tracingService infrastructure.TracingService,
) context.Context {
	if tracingService != nil {
		// 使用TracingService获取trace信息
		c := context.WithValue(ctx, consts.TraceIDKey, tracing.GetTraceID(ctx))
		return context.WithValue(c, consts.SpanIDKey, tracing.GetSpanID(c))
	}
	// 在旧版本中使用的fallback
	c := context.WithValue(ctx, consts.TraceIDKey, tracing.GetTraceID(ctx))
	return context.WithValue(c, consts.SpanIDKey, tracing.GetSpanID(c))
}
