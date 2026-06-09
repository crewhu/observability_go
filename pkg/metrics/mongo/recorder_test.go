package mongo

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv130 "go.opentelemetry.io/otel/semconv/v1.30.0"
)

// feedFunc abstracts the driver-specific monitor so the same scenario
// runs against both the v1 and v2 adapters. Implementations build the
// concrete *event.PoolEvent and pass it to the monitor under test.
type feedFunc func(eventType, address string, duration time.Duration, reason string, maxPoolSize uint64)

const (
	testAddrA = "mongo-a.crewhu.internal:27017"
	testDB    = "crewhu"
)

// setupMeter installs a fresh MeterProvider backed by a ManualReader
// as the global provider and restores the previous one on cleanup.
// Monitors must be constructed after calling it, since instruments
// bind to the global provider at construction time.
func setupMeter(t *testing.T) *sdkmetric.ManualReader {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(provider)
	t.Cleanup(func() {
		otel.SetMeterProvider(prev)
		_ = provider.Shutdown(context.Background())
	})

	return reader
}

func collect(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	return rm
}

func findMetric(t *testing.T, rm metricdata.ResourceMetrics, name string) metricdata.Metrics {
	t.Helper()

	for _, sm := range rm.ScopeMetrics {
		if sm.Scope.Name != meterName {
			continue
		}
		for _, m := range sm.Metrics {
			if m.Name == name {
				return m
			}
		}
	}
	t.Fatalf("metric %q not found", name)
	return metricdata.Metrics{}
}

// sumValue returns the cumulative value of the datapoint matching the
// attribute set on a Sum metric (counter or updowncounter).
func sumValue(t *testing.T, rm metricdata.ResourceMetrics, name string, attrs attribute.Set) int64 {
	t.Helper()

	m := findMetric(t, rm, name)
	sum, ok := m.Data.(metricdata.Sum[int64])
	if !ok {
		t.Fatalf("metric %q: data is %T, want Sum[int64]", name, m.Data)
	}
	for _, dp := range sum.DataPoints {
		if dp.Attributes.Equals(&attrs) {
			return dp.Value
		}
	}
	t.Fatalf("metric %q: no datapoint with attributes %v (got %+v)", name, attrs, sum.DataPoints)
	return 0
}

func gaugeValue(t *testing.T, rm metricdata.ResourceMetrics, name string, attrs attribute.Set) int64 {
	t.Helper()

	m := findMetric(t, rm, name)
	g, ok := m.Data.(metricdata.Gauge[int64])
	if !ok {
		t.Fatalf("metric %q: data is %T, want Gauge[int64]", name, m.Data)
	}
	for _, dp := range g.DataPoints {
		if dp.Attributes.Equals(&attrs) {
			return dp.Value
		}
	}
	t.Fatalf("metric %q: no datapoint with attributes %v", name, attrs)
	return 0
}

func histogramPoint(t *testing.T, rm metricdata.ResourceMetrics, name string, attrs attribute.Set) (count uint64, sum float64) {
	t.Helper()

	m := findMetric(t, rm, name)
	h, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("metric %q: data is %T, want Histogram[float64]", name, m.Data)
	}
	for _, dp := range h.DataPoints {
		if dp.Attributes.Equals(&attrs) {
			return dp.Count, dp.Sum
		}
	}
	t.Fatalf("metric %q: no datapoint with attributes %v", name, attrs)
	return 0, 0
}

// scenarioAttrs returns the expected attribute sets for one pool with
// the db.name attribute present.
func scenarioAttrs(address string) (base, idle, used attribute.Set) {
	common := []attribute.KeyValue{
		semconv.DBSystemMongoDB,
		dbNameKey.String(testDB),
		semconv130.DBClientConnectionPoolName(address),
	}
	base = attribute.NewSet(common...)
	idle = attribute.NewSet(append(common[:3:3], semconv130.DBClientConnectionStateIdle)...)
	used = attribute.NewSet(append(common[:3:3], semconv130.DBClientConnectionStateUsed)...)
	return base, idle, used
}

// runScenario drives the full event→instrument mapping through a
// driver adapter and asserts every metric value and attribute set.
// The monitor must have been built with WithMaxPoolSize(100) and
// WithDatabaseName(testDB).
func runScenario(t *testing.T, reader *sdkmetric.ManualReader, feed feedFunc) {
	t.Helper()

	base, idle, used := scenarioAttrs(testAddrA)

	// Phase 1: pool created, three checkouts queued, two connections
	// created, one finishes its handshake.
	feed(typePoolCreated, testAddrA, 0, "", 80) // option (100) must win over event (80)
	feed(typeCheckOutStarted, testAddrA, 0, "", 0)
	feed(typeCheckOutStarted, testAddrA, 0, "", 0)
	feed(typeCheckOutStarted, testAddrA, 0, "", 0)
	feed(typeConnectionCreated, testAddrA, 0, "", 0)
	feed(typeConnectionCreated, testAddrA, 0, "", 0)
	feed(typeConnectionReady, testAddrA, 250*time.Millisecond, "", 0)

	rm := collect(t, reader)

	if got := gaugeValue(t, rm, semconv.DBClientConnectionMaxName, base); got != 100 {
		t.Errorf("connection.max = %d, want 100 (WithMaxPoolSize must win over the event value)", got)
	}
	if got := sumValue(t, rm, semconv.DBClientConnectionPendingRequestsName, base); got != 3 {
		t.Errorf("pending_requests = %d, want 3", got)
	}
	if got := sumValue(t, rm, connectionCreatedName, base); got != 2 {
		t.Errorf("created = %d, want 2", got)
	}
	if got := sumValue(t, rm, semconv.DBClientConnectionCountName, idle); got != 2 {
		t.Errorf("count{state=idle} = %d, want 2", got)
	}
	if count, sum := histogramPoint(t, rm, semconv.DBClientConnectionCreateTimeName, base); count != 1 || sum != 0.25 {
		t.Errorf("create_time count=%d sum=%v, want count=1 sum=0.25", count, sum)
	}

	// Phase 2: two checkouts succeed, the third times out, one
	// connection is returned, one is closed as stale.
	feed(typeCheckedOut, testAddrA, 10*time.Millisecond, "", 0)
	feed(typeCheckedOut, testAddrA, 30*time.Millisecond, "", 0)
	feed(typeCheckOutFailed, testAddrA, 5*time.Second, reasonTimedOut, 0)
	feed(typeCheckedIn, testAddrA, 0, "", 0)
	feed(typeConnectionClosed, testAddrA, 0, "stale", 0)

	rm = collect(t, reader)

	if got := sumValue(t, rm, semconv.DBClientConnectionPendingRequestsName, base); got != 0 {
		t.Errorf("pending_requests = %d, want 0", got)
	}
	if got := sumValue(t, rm, semconv.DBClientConnectionCountName, used); got != 1 {
		t.Errorf("count{state=used} = %d, want 1", got)
	}
	if got := sumValue(t, rm, semconv.DBClientConnectionCountName, idle); got != 0 {
		t.Errorf("count{state=idle} = %d, want 0", got)
	}
	if got := sumValue(t, rm, semconv.DBClientConnectionTimeoutsName, base); got != 1 {
		t.Errorf("timeouts = %d, want 1", got)
	}
	if count, sum := histogramPoint(t, rm, semconv.DBClientConnectionWaitTimeName, base); count != 2 || sum != 0.04 {
		t.Errorf("wait_time count=%d sum=%v, want count=2 sum=0.04", count, sum)
	}

	closedAttrs := attribute.NewSet(
		semconv.DBSystemMongoDB,
		dbNameKey.String(testDB),
		semconv130.DBClientConnectionPoolName(testAddrA),
		closeReasonKey.String("stale"),
	)
	if got := sumValue(t, rm, connectionClosedName, closedAttrs); got != 1 {
		t.Errorf("closed{reason=stale} = %d, want 1", got)
	}
}

// runMaxFromEventScenario asserts the gauge fallback: without
// WithMaxPoolSize, db.client.connection.max comes from the
// ConnectionPoolCreated event options. The monitor must have been
// built with no options.
func runMaxFromEventScenario(t *testing.T, reader *sdkmetric.ManualReader, feed feedFunc) {
	t.Helper()

	feed(typePoolCreated, testAddrA, 0, "", 80)

	rm := collect(t, reader)
	base := attribute.NewSet(
		semconv.DBSystemMongoDB,
		semconv130.DBClientConnectionPoolName(testAddrA),
	)
	if got := gaugeValue(t, rm, semconv.DBClientConnectionMaxName, base); got != 80 {
		t.Errorf("connection.max = %d, want 80 (from event PoolOptions)", got)
	}
}

// runConcurrentScenario hammers the monitor from many goroutines
// across several pools; meaningful under -race. After every goroutine
// completes a balanced create/checkout/checkin/close cycle, all
// updowncounters must read zero and the wait_time count must match
// the number of checkouts.
func runConcurrentScenario(t *testing.T, reader *sdkmetric.ManualReader, feed feedFunc) {
	t.Helper()

	const (
		goroutines = 32
		iterations = 200
	)
	addresses := []string{
		"mongo-a.crewhu.internal:27017",
		"mongo-b.crewhu.internal:27017",
		"mongo-c.crewhu.internal:27017",
		"mongo-d.crewhu.internal:27017",
	}

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			feed(typeConnectionCreated, addr, 0, "", 0)
			feed(typeConnectionReady, addr, time.Millisecond, "", 0)
			for i := 0; i < iterations; i++ {
				feed(typeCheckOutStarted, addr, 0, "", 0)
				feed(typeCheckedOut, addr, time.Millisecond, "", 0)
				feed(typeCheckedIn, addr, 0, "", 0)
			}
			feed(typeConnectionClosed, addr, 0, "idle", 0)
		}(addresses[g%len(addresses)])
	}
	wg.Wait()

	rm := collect(t, reader)

	perPool := goroutines / len(addresses)
	for _, addr := range addresses {
		base, idle, used := scenarioAttrs(addr)

		if got := sumValue(t, rm, semconv.DBClientConnectionCountName, used); got != 0 {
			t.Errorf("pool %s: count{state=used} = %d, want 0", addr, got)
		}
		if got := sumValue(t, rm, semconv.DBClientConnectionCountName, idle); got != 0 {
			t.Errorf("pool %s: count{state=idle} = %d, want 0", addr, got)
		}
		if got := sumValue(t, rm, semconv.DBClientConnectionPendingRequestsName, base); got != 0 {
			t.Errorf("pool %s: pending_requests = %d, want 0", addr, got)
		}
		wantCheckouts := uint64(perPool * iterations)
		if count, _ := histogramPoint(t, rm, semconv.DBClientConnectionWaitTimeName, base); count != wantCheckouts {
			t.Errorf("pool %s: wait_time count = %d, want %d", addr, count, wantCheckouts)
		}
	}
}
