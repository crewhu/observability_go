// Package resource is the single source of OpenTelemetry resource
// attributes for the library. Traces, logs and metrics all build their
// *resource.Resource through New so every signal carries the same
// service.name, service.version and deployment.environment attributes.
package resource

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// DefaultServiceVersion is used when callers do not provide a version.
// It matches the value historically hardcoded by pkg/tracing and
// pkg/logging, keeping behavior stable for existing consumers.
const DefaultServiceVersion = "0.1.0"

// New builds a *resource.Resource with the standard Crewhu service
// attributes. Empty values are skipped gracefully, so callers can omit
// any attribute they do not have.
func New(serviceName, serviceVersion, environment string) (*resource.Resource, error) {
	attrs := make([]attribute.KeyValue, 0, 3)

	if serviceName != "" {
		attrs = append(attrs, semconv.ServiceName(serviceName))
	}
	if serviceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(serviceVersion))
	}
	if environment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironment(environment))
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the resource: %w", err)
	}

	return res, nil
}
