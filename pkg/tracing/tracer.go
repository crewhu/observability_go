package tracing

import (
	"context"
	"fmt"
	"time"

	obsresource "github.com/crewhu/observability_go/pkg/resource"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// DefaultExporterTimeout is the OTLP trace exporter timeout applied when
// no override is provided. It matches the value historically hardcoded
// by this package.
const DefaultExporterTimeout = 500 * time.Millisecond

type config struct {
	serviceVersion  string
	environment     string
	exporterTimeout time.Duration
}

// Option customizes the tracer created by NewTracerWithOptions.
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

// WithExporterTimeout overrides the OTLP exporter timeout
// (default 500ms). Non-positive values keep the default.
func WithExporterTimeout(timeout time.Duration) Option {
	return func(c *config) {
		if timeout > 0 {
			c.exporterTimeout = timeout
		}
	}
}

type Tracer struct {
	provider *sdktrace.TracerProvider
	endpoint string
	name     string
}

func NewTracer(name, endpoint string) (*Tracer, error) {
	return NewTracerWithOptions(name, endpoint)
}

// NewTracerWithOptions creates a Tracer like NewTracer, additionally
// accepting options for the service version, deployment environment and
// OTLP exporter timeout.
func NewTracerWithOptions(name, endpoint string, opts ...Option) (*Tracer, error) {
	cfg := config{
		serviceVersion:  obsresource.DefaultServiceVersion,
		exporterTimeout: DefaultExporterTimeout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	provider, err := initTracer(name, endpoint, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	return &Tracer{
		provider: provider,
		endpoint: endpoint,
		name:     name,
	}, nil
}

func (t *Tracer) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

func (t *Tracer) GetProvider() *sdktrace.TracerProvider {
	return t.provider
}

func initTracer(name, endpoint string, cfg config) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithTimeout(cfg.exporterTimeout),
	)

	if err != nil {
		return nil, err
	}

	res, err := obsresource.New(name, cfg.serviceVersion, cfg.environment)
	if err != nil {
		return nil, err
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)

	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return traceProvider, nil
}
