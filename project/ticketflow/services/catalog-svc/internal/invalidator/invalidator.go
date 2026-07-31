// Package invalidator keeps every catalog replica's in-process cache coherent.
//
// THE PROBLEM IT SOLVES.
//
// L2 (Redis) is shared, so deleting a key there fixes every replica at once.
// L1 is not: each pod holds its own copy in its own memory. When a promoter
// edits an event, the writer can delete the Redis key, but the other N-1
// replicas keep serving their stale local copies until their TTL lapses. With
// N replicas the same request can return different answers depending on which
// pod it lands on -- and it stays that way for the whole TTL window.
//
// Shortening the L1 TTL is the obvious fix and the wrong one: it trades
// staleness for cache hit ratio, which is the exact thing L1 exists to provide.
//
// Instead, catalog publishes catalog.event.updated on every mutation, and every
// replica consumes it and drops its own copy. Correctness comes from the fan-out,
// not from the clock.
//
// Note each replica needs its OWN consumer group. A shared group would give the
// message to exactly one replica -- the opposite of what is needed here, since
// every replica must hear it.
package invalidator

import (
	"context"
	"log/slog"
	"sync/atomic"

	segkafka "github.com/segmentio/kafka-go"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
)

// Cache is the part of the catalog service this package drives.
type Cache interface {
	// InvalidateEventLocal drops only the in-process copy. The publisher
	// already removed the shared L2 entry, so having every replica delete it
	// again would be N redundant round-trips to Redis.
	InvalidateEventLocal(id string)
}

// Invalidator consumes invalidation events.
type Invalidator struct {
	consumer *tfkafka.Consumer
	cache    Cache
	logger   *slog.Logger

	invalidated, malformed atomic.Uint64
}

func New(consumer *tfkafka.Consumer, cache Cache, logger *slog.Logger) *Invalidator {
	if logger == nil {
		logger = slog.Default()
	}
	return &Invalidator{consumer: consumer, cache: cache, logger: logger}
}

// Run consumes until ctx is cancelled.
func (i *Invalidator) Run(ctx context.Context) error {
	return i.consumer.Run(ctx, i.handle)
}

// handle drops one event from the local cache.
//
// Idempotent by construction: deleting a cache key twice is indistinguishable
// from deleting it once, so at-least-once redelivery costs nothing here. That
// is a happy accident of what invalidation is, not something arranged.
func (i *Invalidator) handle(_ context.Context, msg segkafka.Message) error {
	env, err := tfkafka.Unmarshal[tfkafka.CatalogEventUpdated](msg.Value)
	if err != nil {
		// A payload we cannot parse will never parse. Retrying wastes the
		// budget and blocks the partition; dead-letter it immediately.
		i.malformed.Add(1)
		return tfkafka.Permanent(err)
	}

	if env.Payload.EventID == "" {
		i.malformed.Add(1)
		return tfkafka.Permanent(errEmptyEventID)
	}

	i.cache.InvalidateEventLocal(env.Payload.EventID)
	i.invalidated.Add(1)

	i.logger.Debug("local cache invalidated",
		slog.String("event_id", env.Payload.EventID),
		slog.Int64("version", env.Payload.Version))
	return nil
}

// Stats reports invalidation activity. A replica whose count lags the others
// is not receiving events and is serving stale data.
type Stats struct {
	Invalidated uint64
	Malformed   uint64
}

func (i *Invalidator) Stats() Stats {
	return Stats{
		Invalidated: i.invalidated.Load(),
		Malformed:   i.malformed.Load(),
	}
}

type errString string

func (e errString) Error() string { return string(e) }

const errEmptyEventID errString = "invalidator: message carries no event id"
