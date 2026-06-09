package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/event"
)

// feedV2 adapts the shared scenarios to the mongo-driver v2 event
// package.
func feedV2(monitor *event.PoolMonitor) feedFunc {
	return func(eventType, address string, duration time.Duration, reason string, maxPoolSize uint64) {
		ev := &event.PoolEvent{
			Type:     eventType,
			Address:  address,
			Duration: duration,
			Reason:   reason,
		}
		if maxPoolSize > 0 {
			ev.PoolOptions = &event.MonitorPoolOptions{MaxPoolSize: maxPoolSize}
		}
		monitor.Event(ev)
	}
}

func TestNewPoolMonitorV2Scenario(t *testing.T) {
	reader := setupMeter(t)
	monitor := NewPoolMonitorV2(WithMaxPoolSize(100), WithDatabaseName(testDB))

	runScenario(t, reader, feedV2(monitor))
}

func TestNewPoolMonitorV2MaxFromEvent(t *testing.T) {
	reader := setupMeter(t)
	monitor := NewPoolMonitorV2()

	runMaxFromEventScenario(t, reader, feedV2(monitor))
}

func TestNewPoolMonitorV2Concurrent(t *testing.T) {
	reader := setupMeter(t)
	monitor := NewPoolMonitorV2(WithDatabaseName(testDB))

	runConcurrentScenario(t, reader, feedV2(monitor))
}

func TestNewPoolMonitorV2IgnoresNilAndUnknownEvents(t *testing.T) {
	setupMeter(t)
	monitor := NewPoolMonitorV2()

	monitor.Event(nil)
	monitor.Event(&event.PoolEvent{Type: event.ConnectionPoolCleared, Address: testAddrA})
	monitor.Event(&event.PoolEvent{Type: event.ConnectionPoolReady, Address: testAddrA})
}
