package logging

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

var otelLogger otellog.Logger

func InitLoggerCollector(name, endpoint string) (*sdklog.LoggerProvider, error) {
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(name),
			semconv.ServiceVersionKey.String("0.1.0"),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	exporter, err := otlploghttp.New(
		context.Background(),
		otlploghttp.WithEndpoint(endpoint),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	otelLogger = lp.Logger(name)

	return lp, nil
}
