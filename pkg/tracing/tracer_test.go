package tracing

import (
	"context"
	"testing"
	"time"
)

func TestNewTracerKeepsLegacyBehavior(t *testing.T) {
	tracer, err := NewTracer("test-service", "localhost:0")
	if err != nil {
		t.Fatalf("NewTracer returned error: %v", err)
	}
	if tracer.GetProvider() == nil {
		t.Fatal("GetProvider returned nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := tracer.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}

func TestNewTracerWithOptions(t *testing.T) {
	tracer, err := NewTracerWithOptions(
		"test-service",
		"localhost:0",
		WithServiceVersion("2.0.0"),
		WithEnvironment("test"),
		WithExporterTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("NewTracerWithOptions returned error: %v", err)
	}
	if tracer.GetProvider() == nil {
		t.Fatal("GetProvider returned nil")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := tracer.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}
}

func TestExporterTimeoutOptionIgnoresNonPositive(t *testing.T) {
	cfg := config{exporterTimeout: DefaultExporterTimeout}
	WithExporterTimeout(0)(&cfg)
	if cfg.exporterTimeout != DefaultExporterTimeout {
		t.Errorf("exporterTimeout = %v, want default %v", cfg.exporterTimeout, DefaultExporterTimeout)
	}

	WithExporterTimeout(time.Second)(&cfg)
	if cfg.exporterTimeout != time.Second {
		t.Errorf("exporterTimeout = %v, want 1s", cfg.exporterTimeout)
	}
}
