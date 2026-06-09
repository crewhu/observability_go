package mongo

import (
	"go.mongodb.org/mongo-driver/event"
)

// NewPoolMonitor returns an event.PoolMonitor for mongo-driver v1 that
// records connection-pool metrics through the global MeterProvider
// (register it first via metrics.InitMeterProvider).
//
// Wire it into the client options:
//
//	opts := options.Client().
//		ApplyURI(uri).
//		SetPoolMonitor(mongometrics.NewPoolMonitor(
//			mongometrics.WithMaxPoolSize(maxPoolSize),
//			mongometrics.WithDatabaseName(dbName),
//		))
func NewPoolMonitor(opts ...Option) *event.PoolMonitor {
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
