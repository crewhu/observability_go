package tracing

import (
	"context"

	"github.com/iamviniciuss/observability_go/pkg/logging"
	"go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
	traceIDKey contextKey = "trace-id"
	spanIDKey  contextKey = "span-id"
)

func SetTraceAtContext(ctx context.Context) context.Context {
	traceID := ""
	spanID := ""
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		traceID = spanCtx.TraceID().String()
		spanID = spanCtx.SpanID().String()
	}
	ctx = context.WithValue(ctx, traceIDKey, traceID)
	ctx = context.WithValue(ctx, spanIDKey, spanID)
	return ctx
}

func GetSpanContext(ctx context.Context) context.Context {
	traceID := getTraceID(ctx)
	spanID := getSpanID(ctx)

	logging.Debug(ctx, "processing trace context", logging.Tags{"traceID": traceID.String(), "spanID": spanID.String()})

	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID,
		SpanID:  spanID,
		Remote:  true,
	})

	return trace.ContextWithSpanContext(ctx, spanContext)
}

func getTraceID(ctx context.Context) trace.TraceID {
	traceIDValue, okTrace := ctx.Value(traceIDKey).(string)

	if !okTrace || traceIDValue == "" {
		return trace.TraceID{}
	}

	parsedTraceID, err := trace.TraceIDFromHex(traceIDValue)
	if err != nil {
		logging.Debug(ctx, "invalid trace ID format", logging.Tags{"error": err, "value": traceIDValue})
		return trace.TraceID{}
	}

	return parsedTraceID
}

func getSpanID(ctx context.Context) trace.SpanID {
	spanIDValue, okTrace := ctx.Value(spanIDKey).(string)

	if !okTrace || spanIDValue == "" {
		return trace.SpanID{}
	}

	parsedSpanID, err := trace.SpanIDFromHex(spanIDValue)
	if err != nil {
		logging.Debug(ctx, "invalid span ID format", logging.Tags{"error": err, "value": spanIDValue})
		return trace.SpanID{}
	}

	return parsedSpanID
}
