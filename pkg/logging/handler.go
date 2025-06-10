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

func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	traceInfo := ExtractTraceInfo(ctx)

	loggerWithTrace := logger
	if traceInfo.TraceID != "" {
		loggerWithTrace = loggerWithTrace.With(
			slog.String("trace_id", traceInfo.TraceID),
			slog.String("span_id", traceInfo.SpanID),
		)
	}

	ctxTags := getTags(ctx)
	if len(ctxTags) > 0 {
		attrs := make([]any, 0, len(ctxTags)*2)
		for k, v := range ctxTags {
			attrs = append(attrs, slog.Any(k, v))
		}
		loggerWithTrace = loggerWithTrace.With(attrs...)
	}

	return loggerWithTrace
}

func GetOtelLoggerFromContext(ctx context.Context) otellog.Record {
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

	ctxTags := getTags(ctx)
	if len(ctxTags) > 0 {
		for k, v := range ctxTags {
			otelRecord.AddAttributes(otellog.KeyValue{
				Key:   k,
				Value: otellog.StringValue(fmt.Sprintf("%v", v)),
			})
		}
	}

	return otelRecord
}
