package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"

	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

// TracingMiddleware 紀錄每筆Request的Tracing record
func TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		propagator := propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		)
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := c.Request.Method + " " + c.FullPath()
		ctx, span := tracing.StartSpan(ctx, spanName)
		defer span.End()

		span.SetAttributes(
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.url", c.Request.URL.String()),
			attribute.String("http.path", c.FullPath()),
		)

		c.Request = c.Request.WithContext(ctx)

		c.Next()

		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))

		if len(c.Errors) > 0 {
			span.RecordError(c.Errors.Last().Err)
			fmt.Printf("Trace error %+v", c.Errors)
		}
	}
}
