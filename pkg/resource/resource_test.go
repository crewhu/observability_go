package resource_test

import (
	"testing"

	obsresource "github.com/crewhu/observability_go/pkg/resource"
	"go.opentelemetry.io/otel/attribute"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
)

func attributeValue(t *testing.T, res *sdkresource.Resource, key attribute.Key) (string, bool) {
	t.Helper()

	value, ok := res.Set().Value(key)
	if !ok {
		return "", false
	}
	return value.AsString(), true
}

func TestNewWithAllAttributes(t *testing.T) {
	res, err := obsresource.New("contact-api", "1.2.3", "staging")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	tests := []struct {
		key  attribute.Key
		want string
	}{
		{"service.name", "contact-api"},
		{"service.version", "1.2.3"},
		{"deployment.environment", "staging"},
	}

	for _, tt := range tests {
		got, ok := attributeValue(t, res, tt.key)
		if !ok {
			t.Errorf("attribute %q not found", tt.key)
			continue
		}
		if got != tt.want {
			t.Errorf("attribute %q = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestNewSkipsEmptyValues(t *testing.T) {
	res, err := obsresource.New("contact-api", "", "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if got, ok := attributeValue(t, res, "service.name"); !ok || got != "contact-api" {
		t.Errorf("service.name = %q (found=%v), want %q", got, ok, "contact-api")
	}

	for _, key := range []attribute.Key{"service.version", "deployment.environment"} {
		if got, ok := attributeValue(t, res, key); ok {
			t.Errorf("attribute %q = %q, want it absent", key, got)
		}
	}
}

func TestNewAllEmpty(t *testing.T) {
	res, err := obsresource.New("", "", "")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if got := res.Set().Len(); got != 0 {
		t.Errorf("attribute count = %d, want 0", got)
	}
}

func TestDefaultServiceVersion(t *testing.T) {
	if obsresource.DefaultServiceVersion != "0.1.0" {
		t.Errorf("DefaultServiceVersion = %q, want %q (legacy hardcoded value)", obsresource.DefaultServiceVersion, "0.1.0")
	}
}
