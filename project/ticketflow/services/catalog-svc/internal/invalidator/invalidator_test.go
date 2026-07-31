package invalidator

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	segkafka "github.com/segmentio/kafka-go"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
)

// fakeCache records which ids were dropped.
type fakeCache struct {
	mu      sync.Mutex
	dropped []string
}

func (f *fakeCache) InvalidateEventLocal(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dropped = append(f.dropped, id)
}

func (f *fakeCache) list() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.dropped...)
}

func msgFor(t *testing.T, eventID string, version int64) segkafka.Message {
	t.Helper()
	env := tfkafka.Envelope[tfkafka.CatalogEventUpdated]{
		ID:            "evt_1",
		Type:          tfkafka.TopicCatalogEventUpdated,
		AggregateID:   eventID,
		OccurredAt:    time.Now(),
		SchemaVersion: tfkafka.CurrentSchemaVersion,
		Payload:       tfkafka.CatalogEventUpdated{EventID: eventID, Version: version},
	}
	raw, err := tfkafka.Marshal(env)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	return segkafka.Message{Value: raw}
}

func TestValidMessageDropsTheLocalCopy(t *testing.T) {
	cache := &fakeCache{}
	inv := New(nil, cache, nil)

	if err := inv.handle(context.Background(), msgFor(t, "evt-1", 2)); err != nil {
		t.Fatalf("handle: %v", err)
	}

	got := cache.list()
	if len(got) != 1 || got[0] != "evt-1" {
		t.Errorf("dropped = %v, want [evt-1]", got)
	}
	if inv.Stats().Invalidated != 1 {
		t.Errorf("Invalidated = %d, want 1", inv.Stats().Invalidated)
	}
}

// Invalidation is idempotent by construction -- deleting a key twice is
// indistinguishable from deleting it once -- so at-least-once redelivery costs
// nothing here.
func TestRedeliveryIsHarmless(t *testing.T) {
	cache := &fakeCache{}
	inv := New(nil, cache, nil)
	msg := msgFor(t, "evt-1", 2)

	for range 3 {
		if err := inv.handle(context.Background(), msg); err != nil {
			t.Fatalf("handle: %v", err)
		}
	}
	if n := len(cache.list()); n != 3 {
		t.Errorf("dropped %d times, want 3 -- each delivery should act", n)
	}
}

// A payload that cannot be parsed will never parse. Retrying wastes the budget
// and blocks the partition, so it must be marked permanent and dead-lettered
// immediately.
func TestMalformedPayloadIsPermanent(t *testing.T) {
	cache := &fakeCache{}
	inv := New(nil, cache, nil)

	err := inv.handle(context.Background(), segkafka.Message{Value: []byte("{not json")})
	if !errors.Is(err, tfkafka.ErrPermanent) {
		t.Fatalf("err = %v, want ErrPermanent", err)
	}
	if len(cache.list()) != 0 {
		t.Error("a malformed message still touched the cache")
	}
	if inv.Stats().Malformed != 1 {
		t.Errorf("Malformed = %d, want 1", inv.Stats().Malformed)
	}
}

func TestEmptyEventIDIsPermanent(t *testing.T) {
	cache := &fakeCache{}
	inv := New(nil, cache, nil)

	raw, _ := json.Marshal(tfkafka.Envelope[tfkafka.CatalogEventUpdated]{
		Type: tfkafka.TopicCatalogEventUpdated, SchemaVersion: 1,
	})

	if err := inv.handle(context.Background(), segkafka.Message{Value: raw}); !errors.Is(err, tfkafka.ErrPermanent) {
		t.Fatalf("err = %v, want ErrPermanent", err)
	}
	if len(cache.list()) != 0 {
		t.Error("an event id-less message still touched the cache")
	}
}

// A message from a newer producer during a rolling deploy must be rejected
// rather than silently mis-decoded.
func TestFutureSchemaVersionIsRejected(t *testing.T) {
	cache := &fakeCache{}
	inv := New(nil, cache, nil)

	raw, _ := json.Marshal(tfkafka.Envelope[tfkafka.CatalogEventUpdated]{
		Type:          tfkafka.TopicCatalogEventUpdated,
		SchemaVersion: tfkafka.CurrentSchemaVersion + 1,
		Payload:       tfkafka.CatalogEventUpdated{EventID: "evt-1"},
	})

	if err := inv.handle(context.Background(), segkafka.Message{Value: raw}); !errors.Is(err, tfkafka.ErrPermanent) {
		t.Fatalf("err = %v, want ErrPermanent", err)
	}
	if len(cache.list()) != 0 {
		t.Error("a future-schema message was acted on")
	}
}
