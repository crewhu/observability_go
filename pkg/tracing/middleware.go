package tracing

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel/trace"
)

func TracingMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := SetTraceAtContext(c.UserContext())
		spanCtx := trace.SpanContextFromContext(GetSpanContext(ctx))

		c.Locals("traceCtx", ctx)

		if spanCtx.TraceID().IsValid() {
			c.Set("X-Trace-ID", spanCtx.TraceID().String())
			c.Set("X-Span-ID", spanCtx.SpanID().String())
		}

		err := c.Next()

		return err
	}
}

func GetTraceContext(c *fiber.Ctx) context.Context {
	ctx := c.Locals("traceCtx")
	if ctx == nil {
		return context.Background()
	}
	return ctx.(context.Context)
}
