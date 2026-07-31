package kafka

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestEnvelopeRoundTrip(t *testing.T) {
	original := Envelope[OrderCreated]{
		ID:            "evt_1",
		Type:          TopicOrderCreated,
		AggregateID:   "ord_1",
		OccurredAt:    time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
		SchemaVersion: CurrentSchemaVersion,
		TraceID:       "trace-abc",
		Payload: OrderCreated{
			OrderID: "ord_1", UserID: "u1", EventID: "evt-1",
			SeatIDs: []string{"S-1", "S-2"}, TotalMinor: 250000, CurrencyCode: "INR",
		},
	}

	raw, err := Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	got, err := Unmarshal[OrderCreated](raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.ID != original.ID || got.Type != original.Type {
		t.Errorf("metadata lost: %+v", got)
	}
	if !got.OccurredAt.Equal(original.OccurredAt) {
		t.Errorf("OccurredAt = %v, want %v", got.OccurredAt, original.OccurredAt)
	}
	if got.Payload.TotalMinor != 250000 {
		t.Errorf("TotalMinor = %d", got.Payload.TotalMinor)
	}
	if len(got.Payload.SeatIDs) != 2 {
		t.Errorf("SeatIDs = %v", got.Payload.SeatIDs)
	}
}

// During a rolling deploy a newer producer is live before an older consumer has
// restarted. Guessing at an unknown shape is how a consumer silently
// mis-processes a message; rejecting it routes the message to the DLQ instead.
func TestUnmarshalRejectsFutureSchemaVersion(t *testing.T) {
	raw, err := json.Marshal(Envelope[OrderCreated]{
		ID: "evt_1", Type: TopicOrderCreated,
		SchemaVersion: CurrentSchemaVersion + 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Unmarshal[OrderCreated](raw); !errors.Is(err, ErrSchemaTooNew) {
		t.Fatalf("err = %v, want ErrSchemaTooNew", err)
	}
}

// An older message must still decode: a consumer that has been upgraded has to
// keep draining the backlog written by its predecessor.
func TestUnmarshalAcceptsOlderSchemaVersion(t *testing.T) {
	raw, err := json.Marshal(Envelope[OrderCreated]{
		ID: "evt_1", SchemaVersion: CurrentSchemaVersion - 1,
		Payload: OrderCreated{OrderID: "ord_1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := Unmarshal[OrderCreated](raw)
	if err != nil {
		t.Fatalf("an older schema version was rejected: %v", err)
	}
	if got.Payload.OrderID != "ord_1" {
		t.Errorf("payload lost: %+v", got.Payload)
	}
}

func TestUnmarshalRejectsGarbage(t *testing.T) {
	for _, bad := range [][]byte{[]byte("{not json"), []byte(""), []byte("[]")} {
		if _, err := Unmarshal[OrderCreated](bad); err == nil {
			t.Errorf("Unmarshal(%q) succeeded", bad)
		}
	}
}

// The envelope is JSON on purpose: a message sitting in a DLQ at 3am has to be
// readable with kafkacat and human eyes.
func TestEnvelopeIsHumanReadable(t *testing.T) {
	raw, err := Marshal(Envelope[CatalogEventUpdated]{
		ID: "evt_1", Type: TopicCatalogEventUpdated, SchemaVersion: 1,
		Payload: CatalogEventUpdated{EventID: "evt-arijit", Version: 3},
	})
	if err != nil {
		t.Fatal(err)
	}

	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("envelope is not plain JSON: %v", err)
	}
	for _, key := range []string{"id", "type", "schema_version", "payload"} {
		if _, ok := generic[key]; !ok {
			t.Errorf("envelope is missing %q; a DLQ dump would be unreadable", key)
		}
	}
}

func TestPermanentWrapping(t *testing.T) {
	base := errors.New("payload has no seats")
	wrapped := Permanent(base)

	if !errors.Is(wrapped, ErrPermanent) {
		t.Error("Permanent did not mark the error as permanent")
	}
	// The original cause must survive, or a DLQ header says only "permanent
	// failure" and the operator has to guess.
	if !errors.Is(wrapped, base) {
		t.Error("Permanent lost the underlying cause")
	}
}

func TestConsumerValidatesOptions(t *testing.T) {
	tests := []struct {
		name string
		opts ConsumerOptions
	}{
		{"no brokers", ConsumerOptions{Topic: "t", GroupID: "g"}},
		{"no topic", ConsumerOptions{Brokers: []string{"localhost:9092"}, GroupID: "g"}},
		{"no group", ConsumerOptions{Brokers: []string{"localhost:9092"}, Topic: "t"}},
		{"dlq topic without producer", ConsumerOptions{
			Brokers: []string{"localhost:9092"}, Topic: "t", GroupID: "g", DLQTopic: "dlq",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewConsumer(tt.opts); err == nil {
				t.Error("NewConsumer accepted invalid options")
			}
		})
	}
}

func TestProducerRequiresBrokers(t *testing.T) {
	if _, err := NewProducer(ProducerOptions{}); err == nil {
		t.Error("NewProducer accepted an empty broker list")
	}
}
