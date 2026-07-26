// Package relay implements the outbox-relay poller: exactly the same logic
// as outbox-relay-service/main.go from the microservices example, just
// running in-process against the simulated store/broker so the benchmark
// can drive it through many cycles quickly.
package relay

import (
	"arbiter-outbox-demo/internal/broker"
	"arbiter-outbox-demo/internal/model"
	"arbiter-outbox-demo/internal/store"
)

type Relay struct {
	Store  *store.MySQLStore
	Broker *broker.FaultyBroker
}

// RunCycle is one poll-publish-mark cycle, mirroring processPendingEvents()
// from outbox-relay-service. If a publish attempt "crashes"
// (broker.PublishWithCrashChance returns true), the row is simply left
// pending — exactly like a real relay pod dying mid-publish and Kubernetes
// restarting it. The row is NOT lost because it was never deleted; the next
// call to RunCycle will pick it up again.
func (r *Relay) RunCycle(batchSize int, crashProb float64) (published int) {
	pending := r.Store.PendingOutboxEvents(batchSize)
	for _, evt := range pending {
		msg := model.Message{Key: evt.ID, Payload: evt.Payload}
		crashed := r.Broker.PublishWithCrashChance(msg, crashProb)
		if crashed {
			continue // row stays 'pending' -> retried on next cycle
		}
		r.Store.MarkSent(evt.ID)
		published++
	}
	return published
}
