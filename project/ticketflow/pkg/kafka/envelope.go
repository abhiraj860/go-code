// Package kafka wraps the Kafka client with the conventions every TicketFlow
// service shares: a typed envelope, at-least-once consumption with a worker
// pool, bounded retry, and a dead-letter topic.
package kafka

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Topics published in this system. Declared here rather than as string
// literals at call sites so a typo is a compile error.
const (
	TopicCatalogEventUpdated = "catalog.event.updated"
	TopicOrderCreated        = "order.created"
	TopicTicketIssued        = "ticket.issued"
	TopicSeatReleased        = "seat.released"
	TopicDLQ                 = "dlq"
)

// Envelope is the shape of every message on the bus.
//
// Generic over the payload so producers and consumers agree on the type at
// compile time instead of each hand-rolling a struct with the same five
// metadata fields and then disagreeing about their names.
type Envelope[T any] struct {
	// ID is stable across redeliveries and is what makes at-least-once
	// delivery survivable: a consumer that has already processed this ID skips
	// it. Without it, the outbox's "publish then crash before marking sent"
	// path would mint duplicate tickets.
	ID string `json:"id"`

	// Type names the event, e.g. "order.created". Carried in the body as well
	// as the topic so a message dumped from a DLQ is self-describing.
	Type string `json:"type"`

	// AggregateID is the entity the event concerns. Used as the partition key,
	// so all events for one order stay ordered relative to each other.
	AggregateID string `json:"aggregate_id"`

	// OccurredAt is when the state change happened, not when it was published.
	// The two differ by however long the message sat in the outbox.
	OccurredAt time.Time `json:"occurred_at"`

	// SchemaVersion lets a consumer reject a payload shape it predates rather
	// than silently mis-decoding it.
	SchemaVersion int `json:"schema_version"`

	// TraceID correlates the message with the HTTP request that caused it.
	TraceID string `json:"trace_id,omitempty"`

	Payload T `json:"payload"`
}

// CurrentSchemaVersion is bumped when an envelope payload changes shape
// incompatibly. Consumers compare against it and route mismatches to the DLQ
// rather than guessing.
const CurrentSchemaVersion = 1

// ErrSchemaTooNew is returned when a message declares a schema version this
// binary does not understand -- normal during a rolling deploy, when a newer
// producer is already live and an older consumer has not restarted yet.
var ErrSchemaTooNew = errors.New("kafka: message schema version is newer than this consumer supports")

// Marshal serialises an envelope.
func Marshal[T any](e Envelope[T]) ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("kafka: marshalling %s envelope: %w", e.Type, err)
	}
	return b, nil
}

// Unmarshal parses an envelope and rejects payloads from the future.
//
// JSON rather than protobuf on the wire: a message sitting in a DLQ at 3am
// needs to be readable with kafkacat and human eyes, and the bytes saved do
// not justify losing that. The gRPC APIs stay protobuf, where the contract is
// enforced by buf.
func Unmarshal[T any](data []byte) (Envelope[T], error) {
	var e Envelope[T]
	if err := json.Unmarshal(data, &e); err != nil {
		return e, fmt.Errorf("kafka: unmarshalling envelope: %w", err)
	}
	if e.SchemaVersion > CurrentSchemaVersion {
		return e, fmt.Errorf("%w: message is v%d, this consumer understands v%d",
			ErrSchemaTooNew, e.SchemaVersion, CurrentSchemaVersion)
	}
	return e, nil
}

// OrderCreated is the payload of TopicOrderCreated.
type OrderCreated struct {
	OrderID      string   `json:"order_id"`
	UserID       string   `json:"user_id"`
	EventID      string   `json:"event_id"`
	HoldID       string   `json:"hold_id"`
	SeatIDs      []string `json:"seat_ids"`
	TotalMinor   int64    `json:"total_minor"`
	CurrencyCode string   `json:"currency_code"`
}

// CatalogEventUpdated is the payload of TopicCatalogEventUpdated. It carries no
// event body on purpose: a consumer that needs the new state reads it back from
// catalog, so the message cannot go stale in a queue. Its only job is to say
// "your cached copy of this id is wrong".
type CatalogEventUpdated struct {
	EventID string `json:"event_id"`
	Version int64  `json:"version"`
}

// TicketIssued is the payload of TopicTicketIssued.
type TicketIssued struct {
	TicketID string `json:"ticket_id"`
	OrderID  string `json:"order_id"`
	EventID  string `json:"event_id"`
	SeatID   string `json:"seat_id"`
	UserID   string `json:"user_id"`
	PDFKey   string `json:"pdf_key"`
}
