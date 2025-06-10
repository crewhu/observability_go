package tracing

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/iamviniciuss/observability_go/pkg/logging"
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

        var span trace.Span
        if !spanCtx.IsValid() || !trace.SpanFromContext(otelCtx).IsRecording(){
            otelCtx, newSpan := GetTracer("fiber").Start(
                otelCtx, 
                c.Path(),
                trace.WithAttributes(
					attribute.String("custom", "yes"),
                    attribute.String("http.method", c.Method()),
                    attribute.String("http.url", c.OriginalURL()),
                ),
            )
			span = newSpan
			c.SetUserContext(otelCtx)
            defer span.End()

        } else {
            span = trace.SpanFromContext(otelCtx)
        }

        items := 0
        args := c.Request().URI().QueryArgs()
        args.VisitAll(func(key, value []byte) {
            items++
            span.SetAttributes(
                attribute.String("http.query."+string(key), string(value)),
            )
        })

        span.SetAttributes(
            attribute.Int("http.query.items", items),
        )

        if span.SpanContext().TraceID().IsValid() {
            log.Println("TraceID:", span.SpanContext().TraceID().String())
        }

		logging.Info(ctx, "URL: %s", c.OriginalURL())

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
