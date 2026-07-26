package models

import "time"

// Ticket is a raw support ticket as read from BigQuery (Stage 1 input).
type Ticket struct {
	ID          string `json:"id"`
	RawText     string `json:"raw_text"`
	ShardID     int    `json:"shard_id"`
}

// ExtractionResult is the structured issue/solution pulled out by the LLM (Stage 1 output).
type ExtractionResult struct {
	TicketID string `json:"ticket_id"`
	Issue    string `json:"issue"`
	Solution string `json:"solution"`
}

// EmbeddingRecord is what eventually gets upserted into Milvus (Stage 2 output).
type EmbeddingRecord struct {
	ID        string    `json:"id"` // deterministic hash of TicketID -> idempotent upsert key
	TicketID  string    `json:"ticket_id"`
	Vector    []float32 `json:"vector"`
	Issue     string    `json:"issue"`
	Solution  string    `json:"solution"`
	CreatedAt time.Time `json:"created_at"`
}

// ShardStatus tracks per-shard progress for BOTH stages independently.
// This is what makes the pipeline resumable/idempotent across a crash at any point.
type ShardStatus struct {
	ShardID          int       `json:"shard_id"`
	ExtractionStatus string    `json:"extraction_status"` // pending | in_progress | done | failed
	EmbeddingStatus  string    `json:"embedding_status"`  // pending | in_progress | done | failed
	Attempts         int       `json:"attempts"`
	LastError        string    `json:"last_error,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
	StatusFailed     = "failed"
)
