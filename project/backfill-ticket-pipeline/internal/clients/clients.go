// Package clients contains the external-system integrations.
//
// These are MOCK implementations that simulate realistic latency and failure
// rates so the pipeline can be run end-to-end and produce real, measurable
// numbers WITHOUT needing live BigQuery/LLM/Milvus credentials. In production,
// swap each client's internals for the real SDK call (cloud.google.com/go/bigquery,
// your LLM provider's SDK, github.com/milvus-io/milvus-sdk-go) - the interfaces
// and the pipeline logic around them (retry, rate limit, checkpoint) stay identical.
package clients

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"ticket-pipeline/internal/models"
)

// ---------- BigQuery source (mocked) ----------

type BigQuerySource struct {
	TotalRows int
	ShardSize int
}

func (b *BigQuerySource) NumShards() int {
	n := b.TotalRows / b.ShardSize
	if b.TotalRows%b.ShardSize != 0 {
		n++
	}
	return n
}

// ReadShard simulates reading one shard of raw tickets from BigQuery.
func (b *BigQuerySource) ReadShard(ctx context.Context, shardID int) ([]models.Ticket, error) {
	start := shardID * b.ShardSize
	end := start + b.ShardSize
	if end > b.TotalRows {
		end = b.TotalRows
	}
	if start >= end {
		return nil, nil
	}
	// Simulate network/read latency proportional to shard size.
	time.Sleep(time.Duration(end-start) * 20 * time.Microsecond)

	rows := make([]models.Ticket, 0, end-start)
	for i := start; i < end; i++ {
		rows = append(rows, models.Ticket{
			ID:      fmt.Sprintf("ticket-%07d", i),
			RawText: fmt.Sprintf("Customer reports issue #%d: service timeout when calling API endpoint X. Steps taken: retried, restarted client.", i),
			ShardID: shardID,
		})
	}
	return rows, nil
}

// Note: writing structured extraction output no longer goes through
// BigQuery - see internal/blobstore. BigQuery here is only the source of
// raw tickets (ReadShard above). If you want the final issue/solution
// dataset queryable later, load the blob store's JSONL files into BigQuery
// as a separate, final step - see README "Intermediate storage" section.

// ---------- LLM extraction client (mocked) ----------

type LLMClient struct {
	FailureRate float64 // simulated transient failure probability, e.g. 0.03
	ThrottleRate float64
	rng         *rand.Rand
}

func NewLLMClient(failureRate, throttleRate float64) *LLMClient {
	return &LLMClient{FailureRate: failureRate, ThrottleRate: throttleRate, rng: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

var ErrThrottled = fmt.Errorf("429: rate limited")
var ErrTransient = fmt.Errorf("transient upstream error")

// ExtractBatch simulates calling an LLM to pull issue/solution out of raw text,
// batched for cost efficiency.
func (c *LLMClient) ExtractBatch(ctx context.Context, tickets []models.Ticket) ([]models.ExtractionResult, error) {
	if c.rng.Float64() < c.ThrottleRate {
		return nil, ErrThrottled
	}
	if c.rng.Float64() < c.FailureRate {
		return nil, ErrTransient
	}
	// simulate LLM latency roughly flat per batch (network + inference), not per row
	time.Sleep(80*time.Millisecond + time.Duration(len(tickets))*2*time.Millisecond)

	out := make([]models.ExtractionResult, 0, len(tickets))
	for _, t := range tickets {
		out = append(out, models.ExtractionResult{
			TicketID: t.ID,
			Issue:    "API timeout on endpoint X",
			Solution: "Increase client timeout and add retry with backoff",
		})
	}
	return out, nil
}

// ---------- Embedding client (mocked) ----------

type EmbeddingClient struct {
	FailureRate  float64
	ThrottleRate float64
	Dim          int
	rng          *rand.Rand
}

func NewEmbeddingClient(failureRate, throttleRate float64, dim int) *EmbeddingClient {
	return &EmbeddingClient{FailureRate: failureRate, ThrottleRate: throttleRate, Dim: dim, rng: rand.New(rand.NewSource(time.Now().UnixNano() + 1))}
}

func (c *EmbeddingClient) EmbedBatch(ctx context.Context, results []models.ExtractionResult) ([]models.EmbeddingRecord, error) {
	if c.rng.Float64() < c.ThrottleRate {
		return nil, ErrThrottled
	}
	if c.rng.Float64() < c.FailureRate {
		return nil, ErrTransient
	}
	time.Sleep(60*time.Millisecond + time.Duration(len(results))*1*time.Millisecond)

	out := make([]models.EmbeddingRecord, 0, len(results))
	for _, r := range results {
		vec := make([]float32, c.Dim)
		for i := range vec {
			vec[i] = c.rng.Float32()
		}
		out = append(out, models.EmbeddingRecord{
			ID:        DeterministicID(r.TicketID), // idempotency key
			TicketID:  r.TicketID,
			Vector:    vec,
			Issue:     r.Issue,
			Solution:  r.Solution,
			CreatedAt: time.Now(),
		})
	}
	return out, nil
}

// DeterministicID hashes the ticket ID so re-processing the same ticket always
// produces the same Milvus primary key -> upsert overwrites, never duplicates.
func DeterministicID(ticketID string) string {
	h := sha256.Sum256([]byte(ticketID))
	return hex.EncodeToString(h[:])[:32]
}

// ---------- Milvus sink (mocked) ----------

type MilvusSink struct {
	FailureRate float64
	rng         *rand.Rand
	written     int
}

func NewMilvusSink(failureRate float64) *MilvusSink {
	return &MilvusSink{FailureRate: failureRate, rng: rand.New(rand.NewSource(time.Now().UnixNano() + 2))}
}

// Upsert simulates an idempotent upsert (insert-or-replace by ID) into Milvus.
func (m *MilvusSink) Upsert(ctx context.Context, records []models.EmbeddingRecord) error {
	if m.rng.Float64() < m.FailureRate {
		return ErrTransient
	}
	time.Sleep(time.Duration(len(records)) * 500 * time.Microsecond)
	m.written += len(records)
	return nil
}

// Delete removes vectors written by a shard whose downstream step failed.
// Not a saga compensating transaction - just cleanup. Correctness comes from
// Upsert being idempotent (deterministic ID), so a retry overwrites safely
// with or without this delete ever running.
func (m *MilvusSink) Delete(ctx context.Context, ids []string) error {
	time.Sleep(time.Duration(len(ids)) * 200 * time.Microsecond)
	return nil
}
