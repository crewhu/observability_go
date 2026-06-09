package metrics

import (
	"context"
	"testing"
	"time"

	obsresource "github.com/crewhu/observability_go/pkg/resource"
	"go.opentelemetry.io/otel"
)

func defaultTestConfig(opts ...Option) config {
	cfg := config{
		serviceVersion:  obsresource.DefaultServiceVersion,
		exportInterval:  DefaultExportInterval,
		exporterTimeout: DefaultExporterTimeout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

func TestDefaults(t *testing.T) {
	cfg := defaultTestConfig()

	if cfg.serviceVersion != "0.1.0" {
		t.Errorf("serviceVersion = %q, want %q", cfg.serviceVersion, "0.1.0")
	}
	if cfg.environment != "" {
		t.Errorf("environment = %q, want empty", cfg.environment)
	}
	if cfg.exportInterval != 60*time.Second {
		t.Errorf("exportInterval = %v, want 60s", cfg.exportInterval)
	}
	if cfg.exporterTimeout != 10*time.Second {
		t.Errorf("exporterTimeout = %v, want 10s", cfg.exporterTimeout)
	}
}

func TestOptions(t *testing.T) {
	cfg := defaultTestConfig(
		WithServiceVersion("2.0.0"),
		WithEnvironment("production"),
		WithExportInterval(15*time.Second),
		WithExporterTimeout(3*time.Second),
	)

	if cfg.serviceVersion != "2.0.0" {
		t.Errorf("serviceVersion = %q, want %q", cfg.serviceVersion, "2.0.0")
	}
	if cfg.environment != "production" {
		t.Errorf("environment = %q, want %q", cfg.environment, "production")
	}
	if cfg.exportInterval != 15*time.Second {
		t.Errorf("exportInterval = %v, want 15s", cfg.exportInterval)
	}
	if cfg.exporterTimeout != 3*time.Second {
		t.Errorf("exporterTimeout = %v, want 3s", cfg.exporterTimeout)
	}
}

func TestNonPositiveDurationsKeepDefaults(t *testing.T) {
	cfg := defaultTestConfig(
		WithExportInterval(0),
		WithExporterTimeout(-1*time.Second),
	)

	if cfg.exportInterval != DefaultExportInterval {
		t.Errorf("exportInterval = %v, want default %v", cfg.exportInterval, DefaultExportInterval)
	}
	if cfg.exporterTimeout != DefaultExporterTimeout {
		t.Errorf("exporterTimeout = %v, want default %v", cfg.exporterTimeout, DefaultExporterTimeout)
	}
}

func TestInitMeterProvider(t *testing.T) {
	// "localhost:0" is never dialed during construction: the OTLP/HTTP
	// exporter only connects when exporting, so the test stays offline.
	provider, err := InitMeterProvider(
		"test-service",
		"localhost:0",
		WithEnvironment("test"),
		WithExportInterval(time.Minute),
	)
	if err != nil {
		t.Fatalf("InitMeterProvider returned error: %v", err)
	}
	if provider == nil {
		t.Fatal("InitMeterProvider returned nil provider")
	}

	if got := otel.GetMeterProvider(); got != provider {
		t.Error("InitMeterProvider did not register the global meter provider")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// Shutdown flushes to an unreachable endpoint; the export error is
	// expected here — the test only ensures shutdown returns promptly.
	_ = provider.Shutdown(ctx)
}
