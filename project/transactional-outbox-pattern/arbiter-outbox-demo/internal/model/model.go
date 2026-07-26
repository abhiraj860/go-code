// Package model holds the shared data types used across the simulated
// MySQL store, Kafka broker, outbox relay, and Milvus vector store.
//
// These mirror the real Arbiter shapes closely enough to reason about:
//   - Ticket            -> a row in MySQL's `tickets` table
//   - OutboxEvent        -> a row in MySQL's `outbox_events` table
//   - Message            -> a Kafka message published to a topic
//   - EmbeddingRecord    -> a vector row written into Milvus
package model

// Ticket represents a single incident/triage ticket, the source-of-truth
// row that lives in MySQL.
type Ticket struct {
	ID      string
	Payload string
}

// OutboxEventStatus enumerates the lifecycle of an outbox row.
type OutboxEventStatus string

const (
	StatusPending OutboxEventStatus = "pending"
	StatusSent    OutboxEventStatus = "sent"
)

// OutboxEvent represents a row in the outbox_events table — written in the
// SAME transaction as the Ticket it describes.
type OutboxEvent struct {
	ID       string
	TicketID string
	Payload  string
	Status   OutboxEventStatus
}

// Message is what gets published onto the Kafka topic.
type Message struct {
	Key     string // event ID, used for idempotency downstream
	Payload string
}

// EmbeddingRecord is a row written into Milvus by the vector-ingestion
// consumer after it generates an embedding for a ticket.
type EmbeddingRecord struct {
	TicketID string
	Vector   []float32
}
