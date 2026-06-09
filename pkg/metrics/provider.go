// Package metrics provides the OpenTelemetry metrics pipeline for
// Crewhu Go services: an OTLP/HTTP exporter with a periodic reader,
// registered as the global MeterProvider.
package metrics

import (
	"context"
	"fmt"
	"time"

	obsresource "github.com/crewhu/observability_go/pkg/resource"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	// DefaultExportInterval is how often the periodic reader pushes
	// metrics when no override is provided.
	DefaultExportInterval = 60 * time.Second

	// DefaultExporterTimeout is the OTLP metrics exporter timeout
	// applied when no override is provided.
	DefaultExporterTimeout = 10 * time.Second
)

type config struct {
	serviceVersion  string
	environment     string
	exportInterval  time.Duration
	exporterTimeout time.Duration
}

// Option customizes the meter provider created by InitMeterProvider.
type Option func(*config)

// WithServiceVersion sets the service.version resource attribute
// (default "0.1.0").
func WithServiceVersion(version string) Option {
	return func(c *config) {
		c.serviceVersion = version
	}
}

// WithEnvironment sets the deployment.environment resource attribute.
// When empty, the attribute is omitted.
func WithEnvironment(environment string) Option {
	return func(c *config) {
		c.environment = environment
	}
}

// WithExportInterval overrides the periodic reader export interval
// (default 60s). Non-positive values keep the default.
func WithExportInterval(interval time.Duration) Option {
	return func(c *config) {
		if interval > 0 {
			c.exportInterval = interval
		}
	}
}

// WithExporterTimeout overrides the OTLP exporter timeout
// (default 10s). Non-positive values keep the default.
func WithExporterTimeout(timeout time.Duration) Option {
	return func(c *config) {
		if timeout > 0 {
			c.exporterTimeout = timeout
		}
	}
}

// InitMeterProvider creates a MeterProvider that exports metrics via
// OTLP/HTTP to the given endpoint (same convention as pkg/tracing) and
// registers it as the global meter provider.
//
// Callers own the returned provider and must flush it on exit:
//
//	mp, err := metrics.InitMeterProvider("contact-api", endpoint)
//	if err != nil { ... }
//	defer mp.Shutdown(context.Background())
func InitMeterProvider(serviceName, endpoint string, opts ...Option) (*sdkmetric.MeterProvider, error) {
	cfg := config{
		serviceVersion:  obsresource.DefaultServiceVersion,
		exportInterval:  DefaultExportInterval,
		exporterTimeout: DefaultExporterTimeout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	exporter, err := otlpmetrichttp.New(
		context.Background(),
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithTimeout(cfg.exporterTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	res, err := obsresource.New(serviceName, cfg.serviceVersion, cfg.environment)
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				exporter,
				sdkmetric.WithInterval(cfg.exportInterval),
			),
		),
	)

	otel.SetMeterProvider(meterProvider)

	return meterProvider, nil
}
