package tracing

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TracingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := SetTraceAtContext(c.UserContext())
		otelCtx := GetSpanContext(ctx)

		spanCtx := trace.SpanContextFromContext(otelCtx)

		c.Locals("traceCtx", ctx)

		if spanCtx.TraceID().IsValid() {
			c.Set("X-Trace-ID", spanCtx.TraceID().String())
			c.Set("X-Span-ID", spanCtx.SpanID().String())
		}

		_, span := GetTracer("fiber").Start(c.Context(), "middleware")
		defer span.End()

		for key, value := range c.AllParams() {
			span.SetAttributes(
				attribute.String("http.query."+key, value),
			)
		}

		return c.Next()
	}
}

func GetTraceContext(c *fiber.Ctx) context.Context {
	ctx := c.Locals("traceCtx")
	if ctx == nil {
		return context.Background()
	}
	return ctx.(context.Context)
}
