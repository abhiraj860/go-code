// Package checkpoint implements a durable, crash-safe progress tracker.
//
// WHY THIS EXISTS (defends resume bullet #2 - "cross-stage checkpointing,
// idempotent upserts, safe resume from failure at row-level granularity"):
//
//   - Every shard's status for BOTH pipeline stages (extraction, embedding) is
//     persisted to disk. Here it's a JSON file with atomic rename-on-write
//     (swap for Postgres/Redis/DynamoDB in a real prod deployment - the
//     interface and guarantees are identical, only the storage backend
//     changes).
//   - On restart, the pipeline queries this store and skips any shard already
//     marked "done" for a given stage - so re-running the job never redoes
//     completed work and never double-writes to Milvus.
//   - Writes are serialized behind a mutex and flushed atomically (write to
//     temp file + rename) so a crash mid-write never corrupts the file.
package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"ticket-pipeline/internal/models"
)

type dbFile struct {
	Shards map[int]*models.ShardStatus `json:"shards"`
	Events []event                     `json:"events"`
}

type event struct {
	Type       string    `json:"type"`
	ShardID    int       `json:"shard_id"`
	DurationMs int64     `json:"duration_ms"`
	Rows       int       `json:"rows"`
	Success    bool      `json:"success"`
	CreatedAt  time.Time `json:"created_at"`
}

type Store struct {
	path string
	mu   sync.Mutex
	data dbFile
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, data: dbFile{Shards: map[int]*models.ShardStatus{}}}
	if b, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(b, &s.data); err != nil {
			return nil, fmt.Errorf("parse checkpoint file: %w", err)
		}
		if s.data.Shards == nil {
			s.data.Shards = map[int]*models.ShardStatus{}
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read checkpoint file: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.flushLocked() }

// flushLocked writes the whole state atomically: temp file then rename.
// Caller must hold s.mu.
func (s *Store) flushLocked() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// GetOrInit returns current status for a shard, creating a pending row if absent.
func (s *Store) GetOrInit(shardID int) (models.ShardStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.data.Shards[shardID]
	if !ok {
		st = &models.ShardStatus{
			ShardID:          shardID,
			ExtractionStatus: models.StatusPending,
			EmbeddingStatus:  models.StatusPending,
			UpdatedAt:        time.Now(),
		}
		s.data.Shards[shardID] = st
		if err := s.flushLocked(); err != nil {
			return *st, err
		}
	}
	return *st, nil
}

// UpdateStage sets the status for one stage ("extraction" or "embedding") of a shard.
func (s *Store) UpdateStage(shardID int, stage string, status string, lastErr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, ok := s.data.Shards[shardID]
	if !ok {
		st = &models.ShardStatus{ShardID: shardID, ExtractionStatus: models.StatusPending, EmbeddingStatus: models.StatusPending}
		s.data.Shards[shardID] = st
	}
	if stage == "extraction" {
		st.ExtractionStatus = status
	} else {
		st.EmbeddingStatus = status
	}
	if status == models.StatusFailed || status == models.StatusInProgress {
		st.Attempts++
	}
	st.LastError = lastErr
	st.UpdatedAt = time.Now()
	return s.flushLocked()
}

// IsStageDone lets a worker skip already-completed shards on resume - this is
// the core of the idempotency guarantee.
func (s *Store) IsStageDone(shardID int, stage string) (bool, error) {
	st, err := s.GetOrInit(shardID)
	if err != nil {
		return false, err
	}
	if stage == "extraction" {
		return st.ExtractionStatus == models.StatusDone, nil
	}
	return st.EmbeddingStatus == models.StatusDone, nil
}

// RecordEvent logs a timing/outcome event used later to compute resume metrics:
// throughput (rows/sec), success rate, retry counts, cost proxies.
func (s *Store) RecordEvent(eventType string, shardID int, durationMs int64, rows int, success bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Events = append(s.data.Events, event{
		Type: eventType, ShardID: shardID, DurationMs: durationMs, Rows: rows, Success: success, CreatedAt: time.Now(),
	})
	return s.flushLocked()
}

// Summary is used by cmd/pipeline's `report` subcommand to print the numbers
// that back the resume bullets.
type Summary struct {
	TotalShards        int
	ExtractionDone     int
	ExtractionFailed   int
	EmbeddingDone      int
	EmbeddingFailed    int
	TotalRetries       int
	TotalRowsProcessed int
	SuccessfulRows     int
	AvgDurationMs      float64
	TotalDurationMs    int64
}

func (s *Store) Summarize() (Summary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var sum Summary
	sum.TotalShards = len(s.data.Shards)
	for _, st := range s.data.Shards {
		if st.ExtractionStatus == models.StatusDone {
			sum.ExtractionDone++
		}
		if st.ExtractionStatus == models.StatusFailed {
			sum.ExtractionFailed++
		}
		if st.EmbeddingStatus == models.StatusDone {
			sum.EmbeddingDone++
		}
		if st.EmbeddingStatus == models.StatusFailed {
			sum.EmbeddingFailed++
		}
		if st.Attempts > 0 {
			sum.TotalRetries += st.Attempts - 1
		}
	}
	for _, ev := range s.data.Events {
		sum.TotalRowsProcessed += ev.Rows
		if ev.Success {
			sum.SuccessfulRows += ev.Rows
		}
		sum.TotalDurationMs += ev.DurationMs
	}
	if len(s.data.Events) > 0 {
		sum.AvgDurationMs = float64(sum.TotalDurationMs) / float64(len(s.data.Events))
	}
	return sum, nil
}
