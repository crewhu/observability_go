// Package observability is the single composition entrypoint for the
// library: one Init call wires logs (pkg/logging), traces (pkg/tracing)
// and metrics (pkg/metrics) with the same service resource attributes,
// and one Provider.Shutdown flushes everything on exit.
package observability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/crewhu/observability_go/pkg/logging"
	"github.com/crewhu/observability_go/pkg/metrics"
	"github.com/crewhu/observability_go/pkg/tracing"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Config carries the settings shared by every signal. Zero values fall
// back to the defaults of the underlying packages:
//
//   - ServiceVersion        → resource.DefaultServiceVersion ("0.1.0")
//   - Environment           → deployment.environment attribute omitted
//   - MetricsExportInterval → metrics.DefaultExportInterval (60s)
//   - TraceExporterTimeout  → tracing.DefaultExporterTimeout (500ms)
type Config struct {
	// ServiceName becomes the service.name resource attribute on every
	// signal (convention: the OTEL_PROJECT_NAME env var).
	ServiceName string

	// Endpoint is the OTLP/HTTP collector address as host:port, without
	// scheme (convention: the OTEL_ENDPOINT env var).
	Endpoint string

	// ServiceVersion sets the service.version resource attribute.
	ServiceVersion string

	// Environment sets the deployment.environment resource attribute
	// (convention: the ENVIRONMENT env var).
	Environment string

	// MetricsExportInterval is how often the periodic reader pushes
	// metrics. Non-positive values keep the package default.
	MetricsExportInterval time.Duration

	// TraceExporterTimeout is the OTLP trace exporter timeout.
	// Non-positive values keep the package default.
	TraceExporterTimeout time.Duration
}

// Provider holds the three signal providers created by Init and shuts
// them down together.
type Provider struct {
	loggerProvider *sdklog.LoggerProvider
	tracer         *tracing.Tracer
	meterProvider  *sdkmetric.MeterProvider
}

// Seams for the partial-failure unit tests; production code always uses
// the real constructors.
var (
	initLoggerCollector = logging.InitLoggerCollectorWithOptions
	newTracer           = tracing.NewTracerWithOptions
	initMeterProvider   = metrics.InitMeterProvider
)

// Init initializes, in order, the logger collector, the tracer (also
// registered as the global TracerProvider, so callers no longer need to
// call otel.SetTracerProvider themselves) and the meter provider
// (registered as the global MeterProvider by pkg/metrics).
//
// On partial failure every component already started is shut down with
// ctx before the error is returned, so Init never leaks exporters.
//
//	provider, err := observability.Init(ctx, observability.Config{
//		ServiceName: os.Getenv("OTEL_PROJECT_NAME"),
//		Endpoint:    os.Getenv("OTEL_ENDPOINT"),
//		Environment: os.Getenv("ENVIRONMENT"),
//	})
//	if err != nil { ... }
//	defer provider.Shutdown(ctx)
func Init(ctx context.Context, cfg Config) (*Provider, error) {
	loggerProvider, err := initLoggerCollector(cfg.ServiceName, cfg.Endpoint, cfg.loggingOptions()...)
	if err != nil {
		return nil, fmt.Errorf("observability: init logger collector: %w", err)
	}

	tracer, err := newTracer(cfg.ServiceName, cfg.Endpoint, cfg.tracingOptions()...)
	if err != nil {
		err = fmt.Errorf("observability: init tracer: %w", err)
		return nil, errors.Join(err, loggerProvider.Shutdown(ctx))
	}
	otel.SetTracerProvider(tracer.GetProvider())

	meterProvider, err := initMeterProvider(cfg.ServiceName, cfg.Endpoint, cfg.metricsOptions()...)
	if err != nil {
		err = fmt.Errorf("observability: init meter provider: %w", err)
		return nil, errors.Join(err, tracer.Shutdown(ctx), loggerProvider.Shutdown(ctx))
	}

	return &Provider{
		loggerProvider: loggerProvider,
		tracer:         tracer,
		meterProvider:  meterProvider,
	}, nil
}

func (c Config) loggingOptions() []logging.CollectorOption {
	opts := make([]logging.CollectorOption, 0, 2)
	if c.ServiceVersion != "" {
		opts = append(opts, logging.WithServiceVersion(c.ServiceVersion))
	}
	if c.Environment != "" {
		opts = append(opts, logging.WithEnvironment(c.Environment))
	}
	return opts
}

func (c Config) tracingOptions() []tracing.Option {
	opts := make([]tracing.Option, 0, 3)
	if c.ServiceVersion != "" {
		opts = append(opts, tracing.WithServiceVersion(c.ServiceVersion))
	}
	if c.Environment != "" {
		opts = append(opts, tracing.WithEnvironment(c.Environment))
	}
	if c.TraceExporterTimeout > 0 {
		opts = append(opts, tracing.WithExporterTimeout(c.TraceExporterTimeout))
	}
	return opts
}

func (c Config) metricsOptions() []metrics.Option {
	opts := make([]metrics.Option, 0, 3)
	if c.ServiceVersion != "" {
		opts = append(opts, metrics.WithServiceVersion(c.ServiceVersion))
	}
	if c.Environment != "" {
		opts = append(opts, metrics.WithEnvironment(c.Environment))
	}
	if c.MetricsExportInterval > 0 {
		opts = append(opts, metrics.WithExportInterval(c.MetricsExportInterval))
	}
	return opts
}

// Shutdown flushes and stops every provider in reverse initialization
// order (metrics, traces, logs). All shutdowns are attempted even when
// earlier ones fail; the errors are joined with errors.Join.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil {
		return nil
	}

	var errs []error
	if p.meterProvider != nil {
		errs = append(errs, p.meterProvider.Shutdown(ctx))
	}
	if p.tracer != nil {
		errs = append(errs, p.tracer.Shutdown(ctx))
	}
	if p.loggerProvider != nil {
		errs = append(errs, p.loggerProvider.Shutdown(ctx))
	}
	return errors.Join(errs...)
}

// LoggerProvider returns the underlying OTel logger provider.
func (p *Provider) LoggerProvider() *sdklog.LoggerProvider {
	return p.loggerProvider
}

// Tracer returns the pkg/tracing Tracer created by Init.
func (p *Provider) Tracer() *tracing.Tracer {
	return p.tracer
}

// TracerProvider returns the underlying SDK tracer provider. Init has
// already registered it globally via otel.SetTracerProvider, so most
// consumers never need this accessor.
func (p *Provider) TracerProvider() *sdktrace.TracerProvider {
	return p.tracer.GetProvider()
}

// MeterProvider returns the underlying SDK meter provider. Init has
// already registered it globally via otel.SetMeterProvider, so most
// consumers never need this accessor.
func (p *Provider) MeterProvider() *sdkmetric.MeterProvider {
	return p.meterProvider
}
