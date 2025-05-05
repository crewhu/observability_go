package logging

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type TraceInfo struct {
	TraceID    string `json:"trace_id"`
	SpanID     string `json:"span_id"`
	TraceFlags string `json:"trace_flags"`
}

func ExtractTraceInfo(ctx context.Context) TraceInfo {
	var traceInfo TraceInfo

	spanCtx := trace.SpanContextFromContext(ctx)
	if !spanCtx.IsValid() {
		return traceInfo
	}

	traceInfo.TraceID = spanCtx.TraceID().String()
	traceInfo.SpanID = spanCtx.SpanID().String()
	traceInfo.TraceFlags = spanCtx.TraceFlags().String()

	return traceInfo
}
