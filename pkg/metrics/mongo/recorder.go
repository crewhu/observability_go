// Package mongo translates MongoDB driver connection-pool events into
// OpenTelemetry metrics so SigNoz can show pool saturation per service
// and per database (REQ-3 / PD-2605).
//
// The package supports both driver majors through thin adapters over a
// shared recorder:
//
//   - NewPoolMonitor   → go.mongodb.org/mongo-driver/event      (v1)
//   - NewPoolMonitorV2 → go.mongodb.org/mongo-driver/v2/event   (v2)
//
// Naming decision: metric names follow the NEW singular semconv naming
// (db.client.connection.*), taken from the semconv v1.26.0 constants
// (DBClientConnectionCountName, ...). The matching singular attribute
// keys (db.client.connection.state / db.client.connection.pool.name)
// are not exported by the semconv v1.26.0 Go package (it only ships the
// deprecated plural keys), so those two helpers are imported from
// semconv v1.30.0, where the singular keys landed with identical metric
// names. db.system="mongodb" comes from semconv v1.26.0.
package mongo

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv130 "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// meterName identifies the instrumentation scope of every pool metric.
const meterName = "github.com/crewhu/observability_go/pkg/metrics/mongo"

// Supplementary instruments with no semconv equivalent: connection
// churn counters. High created/closed rates signal pool thrashing
// (e.g. server selection errors or aggressive idle reaping) that the
// gauge-style count metric cannot show.
const (
	connectionCreatedName = "db.client.connection.created"
	connectionClosedName  = "db.client.connection.closed"
)

// dbNameKey is the legacy db.name attribute required by REQ-9 so the
// SigNoz UI can group by service × database. Modern semconv replaced it
// with db.namespace, but db.name is kept on purpose: it matches the
// attribute emitted by the otelmongo command spans the services use.
const dbNameKey = attribute.Key("db.name")

// closeReasonKey carries the driver-provided reason on the closed
// counter (idle, stale, error, poolClosed, connectionError, timeout).
const closeReasonKey = attribute.Key("reason")

// Pool event type and checkout-failure reason strings. Both driver
// majors emit the exact same strings (verified against mongo-driver
// v1.17.9 and v2.3.0); only the Go constant names differ between the
// two event packages, so the recorder matches on the strings directly.
const (
	typePoolCreated       = "ConnectionPoolCreated"
	typeConnectionCreated = "ConnectionCreated"
	typeConnectionReady   = "ConnectionReady"
	typeConnectionClosed  = "ConnectionClosed"
	typeCheckOutStarted   = "ConnectionCheckOutStarted"
	typeCheckOutFailed    = "ConnectionCheckOutFailed"
	typeCheckedOut        = "ConnectionCheckedOut"
	typeCheckedIn         = "ConnectionCheckedIn"

	reasonTimedOut = "timeout"
)

type config struct {
	maxPoolSize  uint64
	databaseName string
}

// Option customizes the pool monitors created by NewPoolMonitor and
// NewPoolMonitorV2.
type Option func(*config)

// WithMaxPoolSize reports the configured maximum pool size on the
// db.client.connection.max gauge. It takes precedence over the value
// the driver attaches to the ConnectionPoolCreated event.
func WithMaxPoolSize(size uint64) Option {
	return func(c *config) {
		c.maxPoolSize = size
	}
}

// WithDatabaseName adds the db.name attribute to every instrument so
// SigNoz can differentiate databases (REQ-9). Empty values are ignored.
func WithDatabaseName(name string) Option {
	return func(c *config) {
		c.databaseName = name
	}
}

// poolEvent is the driver-version-neutral projection of a PoolEvent.
// Both event packages have the same field shapes; the adapters in
// pool_v1.go / pool_v2.go fill this struct.
type poolEvent struct {
	eventType   string
	address     string
	duration    time.Duration
	reason      string
	maxPoolSize uint64 // from PoolOptions on ConnectionPoolCreated, 0 when absent
}

// poolAttrs caches the pre-computed attribute sets for one pool
// (one server address), avoiding per-event allocations on the hot
// checkout path.
type poolAttrs struct {
	base attribute.Set // db.system [+ db.name] + pool.name
	idle attribute.Set // base + state=idle
	used attribute.Set // base + state=used
}

// recorder owns the OTel instruments and the event→measurement
// mapping shared by the v1 and v2 monitors.
//
// State model for db.client.connection.count (documented decision):
//
//   - state=used is exact: CheckedOut +1, CheckedIn -1. The driver
//     emits CheckedIn even for connections that perished in use, so
//     this never drifts.
//   - state=idle is "open and not in use": Created +1, CheckedOut -1,
//     CheckedIn +1, Closed -1. Tracking strictly-idle from events alone
//     is unreliable (a pool clear with interruptInUseConnections emits
//     Closed before CheckedIn, which would leave a permanent skew), so
//     idle is defined as open(created-closed) minus used. Every event
//     appears exactly once per connection lifecycle, making the model
//     drift-free; the only approximation is that a connection counts as
//     idle during its brief handshake window (Created→Ready).
//
// Durations: both installed drivers (v1.17.9, v2.3.0) populate
// PoolEvent.Duration on ConnectionReady (establishment time) and
// ConnectionCheckedOut (checkout wait), so create_time and wait_time
// come straight from the events — no stopwatch map is needed.
// wait_time is recorded only for successful checkouts; failures are
// visible through the timeouts counter and pending_requests.
type recorder struct {
	connCount       metric.Int64UpDownCounter
	connMax         metric.Int64Gauge
	pendingRequests metric.Int64UpDownCounter
	timeouts        metric.Int64Counter
	createTime      metric.Float64Histogram
	waitTime        metric.Float64Histogram
	created         metric.Int64Counter
	closed          metric.Int64Counter

	baseAttrs   []attribute.KeyValue
	maxPoolSize uint64

	mu    sync.RWMutex
	pools map[string]*poolAttrs
}

func newRecorder(cfg config) (*recorder, error) {
	meter := otel.Meter(meterName)

	r := &recorder{
		maxPoolSize: cfg.maxPoolSize,
		pools:       make(map[string]*poolAttrs),
	}

	r.baseAttrs = []attribute.KeyValue{semconv.DBSystemMongoDB}
	if cfg.databaseName != "" {
		r.baseAttrs = append(r.baseAttrs, dbNameKey.String(cfg.databaseName))
	}

	var err error
	if r.connCount, err = meter.Int64UpDownCounter(
		semconv.DBClientConnectionCountName,
		metric.WithUnit(semconv.DBClientConnectionCountUnit),
		metric.WithDescription(semconv.DBClientConnectionCountDescription),
	); err != nil {
		return nil, err
	}
	if r.connMax, err = meter.Int64Gauge(
		semconv.DBClientConnectionMaxName,
		metric.WithUnit(semconv.DBClientConnectionMaxUnit),
		metric.WithDescription(semconv.DBClientConnectionMaxDescription),
	); err != nil {
		return nil, err
	}
	if r.pendingRequests, err = meter.Int64UpDownCounter(
		semconv.DBClientConnectionPendingRequestsName,
		metric.WithUnit(semconv.DBClientConnectionPendingRequestsUnit),
		metric.WithDescription(semconv.DBClientConnectionPendingRequestsDescription),
	); err != nil {
		return nil, err
	}
	if r.timeouts, err = meter.Int64Counter(
		semconv.DBClientConnectionTimeoutsName,
		metric.WithUnit(semconv.DBClientConnectionTimeoutsUnit),
		metric.WithDescription(semconv.DBClientConnectionTimeoutsDescription),
	); err != nil {
		return nil, err
	}
	if r.createTime, err = meter.Float64Histogram(
		semconv.DBClientConnectionCreateTimeName,
		metric.WithUnit(semconv.DBClientConnectionCreateTimeUnit),
		metric.WithDescription(semconv.DBClientConnectionCreateTimeDescription),
	); err != nil {
		return nil, err
	}
	if r.waitTime, err = meter.Float64Histogram(
		semconv.DBClientConnectionWaitTimeName,
		metric.WithUnit(semconv.DBClientConnectionWaitTimeUnit),
		metric.WithDescription(semconv.DBClientConnectionWaitTimeDescription),
	); err != nil {
		return nil, err
	}
	if r.created, err = meter.Int64Counter(
		connectionCreatedName,
		metric.WithUnit("{connection}"),
		metric.WithDescription("The number of connections created (supplementary, no semconv equivalent)"),
	); err != nil {
		return nil, err
	}
	if r.closed, err = meter.Int64Counter(
		connectionClosedName,
		metric.WithUnit("{connection}"),
		metric.WithDescription("The number of connections closed, by reason (supplementary, no semconv equivalent)"),
	); err != nil {
		return nil, err
	}

	return r, nil
}

// attrsFor returns the cached attribute sets for a pool, building them
// on first sight of the address.
func (r *recorder) attrsFor(address string) *poolAttrs {
	r.mu.RLock()
	pa, ok := r.pools[address]
	r.mu.RUnlock()
	if ok {
		return pa
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if pa, ok = r.pools[address]; ok {
		return pa
	}

	base := make([]attribute.KeyValue, 0, len(r.baseAttrs)+2)
	base = append(base, r.baseAttrs...)
	base = append(base, semconv130.DBClientConnectionPoolName(address))

	pa = &poolAttrs{
		base: attribute.NewSet(base...),
		idle: attribute.NewSet(append(base[:len(base):len(base)], semconv130.DBClientConnectionStateIdle)...),
		used: attribute.NewSet(append(base[:len(base):len(base)], semconv130.DBClientConnectionStateUsed)...),
	}
	r.pools[address] = pa
	return pa
}

// handle maps one pool event onto the instruments. It is safe for
// concurrent use: the driver fires pool events from many goroutines.
func (r *recorder) handle(ev poolEvent) {
	ctx := context.Background()
	pa := r.attrsFor(ev.address)
	base := metric.WithAttributeSet(pa.base)
	idle := metric.WithAttributeSet(pa.idle)
	used := metric.WithAttributeSet(pa.used)

	switch ev.eventType {
	case typePoolCreated:
		// Option takes precedence; the event's PoolOptions is the
		// fallback. Zero means "not configured" and is not reported.
		maxConns := r.maxPoolSize
		if maxConns == 0 {
			maxConns = ev.maxPoolSize
		}
		if maxConns > 0 {
			r.connMax.Record(ctx, int64(maxConns), base)
		}

	case typeConnectionCreated:
		r.created.Add(ctx, 1, base)
		r.connCount.Add(ctx, 1, idle)

	case typeConnectionReady:
		if ev.duration > 0 {
			r.createTime.Record(ctx, ev.duration.Seconds(), base)
		}

	case typeConnectionClosed:
		r.closed.Add(ctx, 1, base, metric.WithAttributes(closeReasonKey.String(ev.reason)))
		r.connCount.Add(ctx, -1, idle)

	case typeCheckOutStarted:
		r.pendingRequests.Add(ctx, 1, base)

	case typeCheckedOut:
		r.pendingRequests.Add(ctx, -1, base)
		r.connCount.Add(ctx, 1, used)
		r.connCount.Add(ctx, -1, idle)
		r.waitTime.Record(ctx, ev.duration.Seconds(), base)

	case typeCheckOutFailed:
		r.pendingRequests.Add(ctx, -1, base)
		if ev.reason == reasonTimedOut {
			r.timeouts.Add(ctx, 1, base)
		}

	case typeCheckedIn:
		r.connCount.Add(ctx, -1, used)
		r.connCount.Add(ctx, 1, idle)
	}
}

// noopHandler is returned when instrument creation fails so monitors
// never panic; the error is reported through the global error handler.
func noopHandler(err error) func(poolEvent) {
	otel.Handle(err)
	return func(poolEvent) {}
}

// newHandler builds the shared event handler used by both adapters.
func newHandler(opts ...Option) func(poolEvent) {
	var cfg config
	for _, opt := range opts {
		opt(&cfg)
	}

	r, err := newRecorder(cfg)
	if err != nil {
		return noopHandler(err)
	}
	return r.handle
}
