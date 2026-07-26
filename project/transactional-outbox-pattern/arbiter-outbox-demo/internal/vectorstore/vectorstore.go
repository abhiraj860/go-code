// Package vectorstore simulates Milvus. The metric that matters for the
// idempotency benchmark isn't "how many rows are in Milvus" (duplicates
// would just overwrite the same key) — it's how many times the expensive
// embedding-generation step ran. That's the wasted compute the resume bullet
// is about, so EmbedCallCount is the real thing being measured, and
// InsertCount is tracked alongside just to show final vector-store state.
package vectorstore

import (
	"sync"

	"arbiter-outbox-demo/internal/model"
)

type VectorStore struct {
	mu             sync.Mutex
	rows           map[string]model.EmbeddingRecord
	embedCallCount int
	insertCount    int
}

func NewVectorStore() *VectorStore {
	return &VectorStore{rows: make(map[string]model.EmbeddingRecord)}
}

// GenerateAndInsertEmbedding simulates calling an embedding model (the
// costly step) and then upserting into Milvus. Every call increments
// embedCallCount regardless of whether the row already existed — because in
// production, the compute cost is spent the moment you call the embedding
// API, before you ever get to the Milvus upsert.
func (v *VectorStore) GenerateAndInsertEmbedding(ticketID string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.embedCallCount++
	if _, exists := v.rows[ticketID]; !exists {
		v.insertCount++
	}
	v.rows[ticketID] = model.EmbeddingRecord{TicketID: ticketID, Vector: []float32{0}} // vector contents irrelevant to this benchmark
}

func (v *VectorStore) EmbedCallCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.embedCallCount
}

func (v *VectorStore) InsertCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.insertCount
}
