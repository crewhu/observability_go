package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	otellog "go.opentelemetry.io/otel/log"
)

func ConfigureLogger(level LogLevel) {
	ConfigureLoggerWithWriter(os.Stdout, level)
}

func ConfigureLoggerWithWriter(w io.Writer, level LogLevel) {
	opts := &slog.HandlerOptions{
		Level: slog.Level(level),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			return a
		},
	}

	handler := slog.NewJSONHandler(w, opts)

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// GetLoggerFromContext returns a logger carrying the ctx-accumulated tags
// merged with extra (call-site Tags win on a key collision). Both must be
// folded into a single Tags map before the *one* With() call below: slog
// doesn't dedupe attributes by key, so applying ctx tags and extra as two
// separate With() calls would emit a colliding key twice instead of letting
// the call-site value override it.
func GetLoggerFromContext(ctx context.Context, extra Tags) *slog.Logger {
	traceInfo := ExtractTraceInfo(ctx)

	loggerWithTrace := logger
	if traceInfo.TraceID != "" {
		loggerWithTrace = loggerWithTrace.With(
			slog.String("trace_id", traceInfo.TraceID),
			slog.String("span_id", traceInfo.SpanID),
		)
	}

	tags := mergeCtxTags(ctx, extra)
	if len(tags) > 0 {
		attrs := make([]any, 0, len(tags)*2)
		for k, v := range tags {
			attrs = append(attrs, slog.Any(k, v))
		}
		loggerWithTrace = loggerWithTrace.With(attrs...)
	}

	return loggerWithTrace
}

// GetOtelLoggerFromContext mirrors GetLoggerFromContext's merge-before-emit
// rule for the otel Record: AddAttributes doesn't dedupe by key either.
func GetOtelLoggerFromContext(ctx context.Context, extra Tags) otellog.Record {
	traceInfo := ExtractTraceInfo(ctx)
	otelRecord := otellog.Record{}

	if traceInfo.TraceID != "" {
		otelRecord.AddAttributes(otellog.KeyValue{
			Key:   "trace_id",
			Value: otellog.StringValue(traceInfo.TraceID),
		}, otellog.KeyValue{
			Key:   "span_id",
			Value: otellog.StringValue(traceInfo.SpanID),
		})
	} 

	tags := mergeCtxTags(ctx, extra)
	if len(tags) > 0 {
		for k, v := range tags {
			otelRecord.AddAttributes(otellog.KeyValue{
				Key:   k,
				Value: otellog.StringValue(fmt.Sprintf("%v", v)),
			})
		}
	}

	return otelRecord
}
