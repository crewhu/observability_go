package tracing

import (
	"runtime/debug"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)


func RecordCustomException(span trace.Span, name string, err error, options ...attribute.KeyValue) {
	options = append(options, attribute.String("exception.type", name))
	options = append(options, attribute.String("exception.message", err.Error()))
	options = append(options, attribute.String("exception.stacktrace", string(debug.Stack())))

	span.AddEvent("exception", trace.WithAttributes(options...))
	span.SetStatus(codes.Error, name+": "+err.Error())
}