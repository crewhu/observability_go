package logging

import (
	"context"
	"fmt"

	obsresource "github.com/crewhu/observability_go/pkg/resource"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

type collectorConfig struct {
	serviceVersion string
	environment    string
}

// CollectorOption customizes the logger provider created by
// InitLoggerCollectorWithOptions.
type CollectorOption func(*collectorConfig)

// WithServiceVersion sets the service.version resource attribute
// (default "0.1.0").
func WithServiceVersion(version string) CollectorOption {
	return func(c *collectorConfig) {
		c.serviceVersion = version
	}
}

// WithEnvironment sets the deployment.environment resource attribute.
// When empty, the attribute is omitted.
func WithEnvironment(environment string) CollectorOption {
	return func(c *collectorConfig) {
		c.environment = environment
	}
}

var otelLogger otellog.Logger

func InitLoggerCollector(name, endpoint string) (*sdklog.LoggerProvider, error) {
	return InitLoggerCollectorWithOptions(name, endpoint)
}

// InitLoggerCollectorWithOptions creates a LoggerProvider like
// InitLoggerCollector, additionally accepting options for the service
// version and deployment environment resource attributes.
func InitLoggerCollectorWithOptions(name, endpoint string, opts ...CollectorOption) (*sdklog.LoggerProvider, error) {
	cfg := collectorConfig{
		serviceVersion: obsresource.DefaultServiceVersion,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	res, err := obsresource.New(name, cfg.serviceVersion, cfg.environment)
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
