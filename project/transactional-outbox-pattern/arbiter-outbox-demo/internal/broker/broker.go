// Package broker simulates a Kafka broker plus the "process crash" fault
// that both benchmarks inject.
//
// Modeling note on WHERE the crash happens (this is the crux of the whole
// demo): a real dual-write bug happens when the service process dies
// *after* it has done something durable (like a DB commit) but *before* it
// finishes the next step (like a Kafka publish ack). We simulate that exact
// moment: PublishWithCrashChance() has a `crashProb` chance of the calling
// goroutine "dying" right there — meaning the publish never happens and,
// critically, the caller never even gets an error back to retry on (because
// in real life, a dead process can't run its own retry/catch block).
//
// The difference between the naive and outbox benchmarks is NOT in this
// broker code — it's in whether anything durable survives that crash to
// allow a retry later. That's exactly what production incident postmortems
// for dual-write bugs look like.
package broker

import (
	"math/rand"
	"sync"

	"arbiter-outbox-demo/internal/model"
)

type FaultyBroker struct {
	mu        sync.Mutex
	delivered []model.Message
	rng       *rand.Rand
}

func NewFaultyBroker(seed int64) *FaultyBroker {
	return &FaultyBroker{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// PublishWithCrashChance attempts to publish msg. With probability
// crashProb, the process is simulated to crash at this exact point:
// the function returns `crashed=true` and the message is NOT delivered.
// The caller must not treat this like a normal returned error — a crashed
// process can't execute error-handling code, so the two benchmarks handle
// this return value very differently (see cmd/outbox-bench and
// cmd/idempotency-bench).
func (b *FaultyBroker) PublishWithCrashChance(msg model.Message, crashProb float64) (crashed bool) {
	b.mu.Lock()
	roll := b.rng.Float64()
	if roll < crashProb {
		b.mu.Unlock()
		return true
	}
	b.delivered = append(b.delivered, msg)
	b.mu.Unlock()
	return false
}

func (b *FaultyBroker) DeliveredCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.delivered)
}

func (b *FaultyBroker) Delivered() []model.Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]model.Message, len(b.delivered))
	copy(out, b.delivered)
	return out
}
