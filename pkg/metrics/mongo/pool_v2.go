package mongo

import (
	"go.mongodb.org/mongo-driver/v2/event"
)

// NewPoolMonitorV2 returns an event.PoolMonitor for mongo-driver v2
// that records connection-pool metrics through the global
// MeterProvider (register it first via metrics.InitMeterProvider).
//
// Wire it into the client options:
//
//	opts := options.Client().
//		ApplyURI(uri).
//		SetPoolMonitor(mongometrics.NewPoolMonitorV2(
//			mongometrics.WithMaxPoolSize(maxPoolSize),
//			mongometrics.WithDatabaseName(dbName),
//		))
func NewPoolMonitorV2(opts ...Option) *event.PoolMonitor {
	handle := newHandler(opts...)

	return &event.PoolMonitor{
		Event: func(ev *event.PoolEvent) {
			if ev == nil {
				return
			}

			pe := poolEvent{
				eventType: ev.Type,
				address:   ev.Address,
				duration:  ev.Duration,
				reason:    ev.Reason,
			}
			if ev.PoolOptions != nil {
				pe.maxPoolSize = ev.PoolOptions.MaxPoolSize
			}

			handle(pe)
		},
	}
}
