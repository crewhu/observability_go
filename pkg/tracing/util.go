package tracing

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
