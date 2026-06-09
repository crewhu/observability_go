package observability

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/crewhu/observability_go/pkg/logging"
	"github.com/crewhu/observability_go/pkg/metrics"
	"github.com/crewhu/observability_go/pkg/tracing"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// saveGlobals restores the global tracer and meter providers after a
// test that runs Init, which mutates both.
func saveGlobals(t *testing.T) {
	t.Helper()

	prevTracer := otel.GetTracerProvider()
	prevMeter := otel.GetMeterProvider()
	t.Cleanup(func() {
		otel.SetTracerProvider(prevTracer)
		otel.SetMeterProvider(prevMeter)
	})
}

// restoreSeams resets the constructor seams stubbed by failure tests.
func restoreSeams(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		initLoggerCollector = logging.InitLoggerCollectorWithOptions
		newTracer = tracing.NewTracerWithOptions
		initMeterProvider = metrics.InitMeterProvider
	})
}

// recordingProcessor is an sdklog.Processor that records whether it was
// shut down and returns a configurable shutdown error.
type recordingProcessor struct {
	mu          sync.Mutex
	shutdown    bool
	shutdownErr error
}

func (p *recordingProcessor) OnEmit(context.Context, *sdklog.Record) error { return nil }
func (p *recordingProcessor) ForceFlush(context.Context) error             { return nil }

func (p *recordingProcessor) Shutdown(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.shutdown = true
	return p.shutdownErr
}

func (p *recordingProcessor) wasShutdown() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.shutdown
}

// recordingMetricExporter is an sdkmetric.Exporter that records whether
// it was shut down and returns a configurable shutdown error.
type recordingMetricExporter struct {
	mu          sync.Mutex
	shutdown    bool
	shutdownErr error
}

func (e *recordingMetricExporter) Temporality(k sdkmetric.InstrumentKind) metricdata.Temporality {
	return sdkmetric.DefaultTemporalitySelector(k)
}

func (e *recordingMetricExporter) Aggregation(k sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(k)
}

func (e *recordingMetricExporter) Export(context.Context, *metricdata.ResourceMetrics) error {
	return nil
}

func (e *recordingMetricExporter) ForceFlush(context.Context) error { return nil }

func (e *recordingMetricExporter) Shutdown(context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.shutdown = true
	return e.shutdownErr
}

func (e *recordingMetricExporter) wasShutdown() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.shutdown
}

func TestInit(t *testing.T) {
	saveGlobals(t)

	// "localhost:0" is never dialed during construction: the OTLP/HTTP
	// exporters only connect when exporting, so the test stays offline.
	provider, err := Init(context.Background(), Config{
		ServiceName:           "test-service",
		Endpoint:              "localhost:0",
		ServiceVersion:        "1.2.3",
		Environment:           "test",
		MetricsExportInterval: time.Minute,
		TraceExporterTimeout:  time.Second,
	})
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	if provider.LoggerProvider() == nil {
		t.Error("LoggerProvider returned nil")
	}
	if provider.Tracer() == nil {
		t.Error("Tracer returned nil")
	}
	if provider.TracerProvider() == nil {
		t.Error("TracerProvider returned nil")
	}
	if provider.MeterProvider() == nil {
		t.Error("MeterProvider returned nil")
	}

	if got := otel.GetTracerProvider(); got != provider.TracerProvider() {
		t.Error("Init did not register the global tracer provider")
	}
	if got := otel.GetMeterProvider(); got != provider.MeterProvider() {
		t.Error("Init did not register the global meter provider")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Shutdown flushes to an unreachable endpoint; export errors are
	// expected here — the test only ensures shutdown returns promptly.
	_ = provider.Shutdown(ctx)
}

func TestInitZeroValuesUsePackageDefaults(t *testing.T) {
	cfg := Config{}

	if got := len(cfg.loggingOptions()); got != 0 {
		t.Errorf("loggingOptions() returned %d options, want 0", got)
	}
	if got := len(cfg.tracingOptions()); got != 0 {
		t.Errorf("tracingOptions() returned %d options, want 0", got)
	}
	if got := len(cfg.metricsOptions()); got != 0 {
		t.Errorf("metricsOptions() returned %d options, want 0", got)
	}
}

func TestInitTracerFailureShutsDownLogger(t *testing.T) {
	restoreSeams(t)

	proc := &recordingProcessor{}
	initLoggerCollector = func(name, endpoint string, opts ...logging.CollectorOption) (*sdklog.LoggerProvider, error) {
		return sdklog.NewLoggerProvider(sdklog.WithProcessor(proc)), nil
	}
	errTracer := errors.New("tracer boom")
	newTracer = func(name, endpoint string, opts ...tracing.Option) (*tracing.Tracer, error) {
		return nil, errTracer
	}

	provider, err := Init(context.Background(), Config{ServiceName: "test-service", Endpoint: "localhost:0"})
	if provider != nil {
		t.Error("Init returned a provider on failure")
	}
	if !errors.Is(err, errTracer) {
		t.Errorf("Init error = %v, want wrapping %v", err, errTracer)
	}
	if !proc.wasShutdown() {
		t.Error("logger provider was not shut down after tracer failure")
	}
}

func TestInitMeterFailureShutsDownTracerAndLogger(t *testing.T) {
	saveGlobals(t)
	restoreSeams(t)

	proc := &recordingProcessor{}
	initLoggerCollector = func(name, endpoint string, opts ...logging.CollectorOption) (*sdklog.LoggerProvider, error) {
		return sdklog.NewLoggerProvider(sdklog.WithProcessor(proc)), nil
	}

	tracer, err := tracing.NewTracerWithOptions("test-service", "localhost:0")
	if err != nil {
		t.Fatalf("NewTracerWithOptions returned error: %v", err)
	}
	newTracer = func(name, endpoint string, opts ...tracing.Option) (*tracing.Tracer, error) {
		return tracer, nil
	}

	errMeter := errors.New("meter boom")
	initMeterProvider = func(name, endpoint string, opts ...metrics.Option) (*sdkmetric.MeterProvider, error) {
		return nil, errMeter
	}

	provider, err := Init(context.Background(), Config{ServiceName: "test-service", Endpoint: "localhost:0"})
	if provider != nil {
		t.Error("Init returned a provider on failure")
	}
	if !errors.Is(err, errMeter) {
		t.Errorf("Init error = %v, want wrapping %v", err, errMeter)
	}
	if !proc.wasShutdown() {
		t.Error("logger provider was not shut down after meter failure")
	}

	// A shut-down SDK tracer provider hands out non-recording spans.
	_, span := tracer.GetProvider().Tracer("test").Start(context.Background(), "op")
	defer span.End()
	if span.IsRecording() {
		t.Error("tracer provider was not shut down after meter failure")
	}
}

func TestShutdownAggregatesErrors(t *testing.T) {
	errLogger := errors.New("logger shutdown boom")
	errMeter := errors.New("meter shutdown boom")

	proc := &recordingProcessor{shutdownErr: errLogger}
	exporter := &recordingMetricExporter{shutdownErr: errMeter}

	tracer, err := tracing.NewTracerWithOptions("test-service", "localhost:0")
	if err != nil {
		t.Fatalf("NewTracerWithOptions returned error: %v", err)
	}

	provider := &Provider{
		loggerProvider: sdklog.NewLoggerProvider(sdklog.WithProcessor(proc)),
		tracer:         tracer,
		meterProvider: sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		),
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = provider.Shutdown(ctx)

	if !errors.Is(err, errMeter) {
		t.Errorf("Shutdown error = %v, want it to wrap %v", err, errMeter)
	}
	if !errors.Is(err, errLogger) {
		t.Errorf("Shutdown error = %v, want it to wrap %v", err, errLogger)
	}
	if !exporter.wasShutdown() {
		t.Error("meter exporter was not shut down")
	}
	if !proc.wasShutdown() {
		t.Error("logger processor was not shut down despite meter error")
	}
}

func TestShutdownNilSafe(t *testing.T) {
	var nilProvider *Provider
	if err := nilProvider.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown on nil provider returned error: %v", err)
	}

	if err := (&Provider{}).Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown on empty provider returned error: %v", err)
	}
}
