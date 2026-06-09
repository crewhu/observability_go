# Crewhu Observability Go

OpenTelemetry instrumentation library for Crewhu Go services. One `Init` call wires the three signals against the same OTLP/HTTP collector (SigNoz) with consistent resource attributes:

- **Traces** (`pkg/tracing`) — OTLP trace exporter, W3C propagation, Fiber middleware
- **Logs** (`pkg/logging`) — structured logs (slog) mirrored to the OTLP log exporter, with trace context
- **Metrics** (`pkg/metrics`) — global `MeterProvider` with a periodic OTLP reader
- **MongoDB pool metrics** (`pkg/metrics/mongo`) — connection-pool monitors for mongo-driver v1 and v2

## Install

```bash
go get github.com/crewhu/observability_go
```

## Quickstart

```go
package main

import (
	"context"
	"os"

	"github.com/crewhu/observability_go/pkg/logging"
	"github.com/crewhu/observability_go/pkg/observability"
)

func main() {
	ctx := context.Background()

	logging.SetLoggingLevel(logging.LogLevelInfo)

	provider, err := observability.Init(ctx, observability.Config{
		ServiceName: os.Getenv("OTEL_PROJECT_NAME"), // service.name
		Endpoint:    os.Getenv("OTEL_ENDPOINT"),     // host:port, no scheme
		Environment: os.Getenv("ENVIRONMENT"),       // deployment.environment
	})
	if err != nil {
		logging.Error(ctx, "failed to initialize observability: %v", err)
		os.Exit(1)
	}
	defer provider.Shutdown(ctx) // flushes traces, logs and metrics

	// Rest of the application...
}
```

`Init` initializes the logger collector, the tracer and the meter provider, and registers the tracer and meter providers globally — no need to call `otel.SetTracerProvider` yourself anymore. On partial failure it shuts down whatever already started and returns the error. `Provider.Shutdown` flushes everything and joins shutdown errors with `errors.Join`.

Optional `Config` fields (zero values keep the package defaults):

| Field | Default |
|---|---|
| `ServiceVersion` | `0.1.0` |
| `Environment` | attribute omitted |
| `MetricsExportInterval` | `60s` |
| `TraceExporterTimeout` | `500ms` |

Underlying providers stay accessible via `provider.LoggerProvider()`, `provider.Tracer()`, `provider.TracerProvider()` and `provider.MeterProvider()`. The individual packages (`tracing.NewTracerWithOptions`, `logging.InitLoggerCollectorWithOptions`, `metrics.InitMeterProvider`) remain usable on their own.

## Environment variable conventions

| Variable | Meaning | Example |
|---|---|---|
| `OTEL_PROJECT_NAME` | Service name → `service.name` on every signal | `contact-api` |
| `OTEL_ENDPOINT` | OTLP/HTTP collector address, `host:port` without scheme | `otel-collector:4318` |
| `ENVIRONMENT` | Deployment environment → `deployment.environment` | `production` |

## HTTP instrumentation (Fiber)

```go
import (
	"github.com/crewhu/observability_go/pkg/tracing/middleware"
	"github.com/gofiber/fiber/v2"
)

app := fiber.New()
app.Use(middleware.OtelMiddleware())
```

## Logging

```go
import "github.com/crewhu/observability_go/pkg/logging"

logging.Debug(ctx, "debug details: %s", detail)
logging.Info(ctx, "operation completed")
logging.Warn(ctx, "high resource usage")
logging.Error(ctx, "failed to process request: %v", err)
logging.Err(ctx, err) // logs the error with the error=true tag
```

Logs carry the active trace/span IDs from `ctx`, so SigNoz links them to the matching trace.

## Manual spans

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

ctx, span := otel.Tracer("contact-api").Start(ctx, "ListContacts")
defer span.End()
span.SetAttributes(attribute.String("company_id", companyID))
```

Name spans as `ResourceAction` (`UserLogin`, `ListContacts`). Use attributes for searchable values (entity IDs, statuses) and events (`span.AddEvent`) for milestones inside the span.

## MongoDB instrumentation

Two complementary monitors attach to the mongo client options:

- **Command monitor** (contrib `otelmongo`) — one span per MongoDB command, with `db.system` and `db.name` attributes
- **Pool monitor** (`pkg/metrics/mongo`) — connection-pool metrics through the global `MeterProvider`

Call `observability.Init` (or `metrics.InitMeterProvider`) **before** constructing the monitors: instruments bind to the global meter provider at construction time.

### Driver v1 (`go.mongodb.org/mongo-driver`)

```go
import (
	mongometrics "github.com/crewhu/observability_go/pkg/metrics/mongo"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

opts := options.Client().
	ApplyURI(uri).
	SetMonitor(otelmongo.NewMonitor()). // command spans
	SetPoolMonitor(mongometrics.NewPoolMonitor( // pool metrics
		mongometrics.WithMaxPoolSize(100),
		mongometrics.WithDatabaseName("crewhu"),
	))

client, err := mongo.Connect(ctx, opts)
```

### Driver v2 (`go.mongodb.org/mongo-driver/v2`)

```go
import (
	mongometrics "github.com/crewhu/observability_go/pkg/metrics/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
)

opts := options.Client().
	ApplyURI(uri).
	SetMonitor(otelmongo.NewMonitor()). // command spans
	SetPoolMonitor(mongometrics.NewPoolMonitorV2( // pool metrics
		mongometrics.WithMaxPoolSize(100),
		mongometrics.WithDatabaseName("crewhu"),
	))

client, err := mongo.Connect(opts)
```

### Emitted pool metrics

All instruments follow OTel semantic conventions (`db.client.connection.*`) and carry `db.system=mongodb`, `db.client.connection.pool.name=<server address>` and, when `WithDatabaseName` is set, `db.name`.

| Metric | Type | Unit | Description |
|---|---|---|---|
| `db.client.connection.count` | UpDownCounter | `{connection}` | Open connections, split by `db.client.connection.state` (`idle` / `used`) |
| `db.client.connection.max` | Gauge | `{connection}` | Maximum allowed pool size (from `WithMaxPoolSize` or the driver's pool options) |
| `db.client.connection.pending_requests` | UpDownCounter | `{request}` | Checkout requests waiting for a connection |
| `db.client.connection.timeouts` | Counter | `{timeout}` | Checkouts that failed with a timeout |
| `db.client.connection.create_time` | Histogram | `s` | Time to establish a new connection |
| `db.client.connection.wait_time` | Histogram | `s` | Time waiting to obtain a connection from the pool |
| `db.client.connection.created` | Counter | `{connection}` | Connections created (supplementary: churn rate) |
| `db.client.connection.closed` | Counter | `{connection}` | Connections closed, by `reason` (supplementary: churn rate) |

### How telemetry is segregated in SigNoz

- **`service.name`** (from `Config.ServiceName` / `OTEL_PROJECT_NAME`) is the resource attribute on every span, log and metric — it is what splits telemetry per service in the SigNoz Services and Dashboards views.
- **`db.system=mongodb`** marks spans/metrics as MongoDB traffic, so database panels can filter out HTTP and other instrumentation.
- **`db.name`** identifies the logical database. The pool monitor emits the same `db.name` attribute as the `otelmongo` command spans, so one `service.name × db.name` group-by correlates command latency with pool saturation for the same database.

## Versioning and releases

The project follows [Semantic Versioning 2.0.0](https://semver.org/). CI derives the release type from commit messages: `feat(major):` / `BREAKING CHANGE` → MAJOR, `feat:` → MINOR, anything else (`fix:`, `docs:`, ...) → PATCH. See [docs/guides/versioning.md](./docs/guides/versioning.md) and [docs/guides/auto-release.md](./docs/guides/auto-release.md).
