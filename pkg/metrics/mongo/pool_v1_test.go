package mongo

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/event"
)

// feedV1 adapts the shared scenarios to the mongo-driver v1 event
// package.
func feedV1(monitor *event.PoolMonitor) feedFunc {
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

func TestNewPoolMonitorScenario(t *testing.T) {
	reader := setupMeter(t)
	monitor := NewPoolMonitor(WithMaxPoolSize(100), WithDatabaseName(testDB))

	runScenario(t, reader, feedV1(monitor))
}

func TestNewPoolMonitorMaxFromEvent(t *testing.T) {
	reader := setupMeter(t)
	monitor := NewPoolMonitor()

	runMaxFromEventScenario(t, reader, feedV1(monitor))
}

func TestNewPoolMonitorConcurrent(t *testing.T) {
	reader := setupMeter(t)
	monitor := NewPoolMonitor(WithDatabaseName(testDB))

	runConcurrentScenario(t, reader, feedV1(monitor))
}

func TestNewPoolMonitorIgnoresNilAndUnknownEvents(t *testing.T) {
	setupMeter(t)
	monitor := NewPoolMonitor()

	monitor.Event(nil)
	monitor.Event(&event.PoolEvent{Type: event.PoolCleared, Address: testAddrA})
	monitor.Event(&event.PoolEvent{Type: event.PoolReady, Address: testAddrA})
}
