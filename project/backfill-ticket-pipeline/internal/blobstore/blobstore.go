// Package blobstore is the intermediate store between the LLM extraction
// stage and the embedding stage: one JSONL file per shard, written once,
// read once shortly after (or skipped entirely on resume if it already
// exists).
//
// WHY BLOB STORAGE INSTEAD OF BIGQUERY (see README "Intermediate storage"
// section for the full reasoning):
//   - Each shard writes to its own key -> workers never contend with each
//     other, no locking needed for a distributed worker pool.
//   - Idempotency check is "does this object exist?" - no query engine,
//     no streaming-insert consistency lag.
//   - ~1,800 objects for a 900K-row backfill at shard size 500, not 900K
//     row-level writes - cheap for a one-time job.
//
// THIS IMPLEMENTATION uses the local filesystem so it can run end-to-end in
// this environment without network access to a real object store. The
// interface (Write/Read/Exists keyed by shard ID) is exactly what you'd call
// against a real GCS/S3 client - swapping the two is described at the
// bottom of this file.
package blobstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ticket-pipeline/internal/models"
)

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create blob dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) shardPath(shardID int) string {
	return filepath.Join(s.dir, fmt.Sprintf("shard_%05d.jsonl", shardID))
}

// Exists checks whether this shard's extraction output was already written -
// this is the idempotency check the pipeline uses to skip re-calling the LLM
// on resume.
func (s *Store) Exists(shardID int) bool {
	_, err := os.Stat(s.shardPath(shardID))
	return err == nil
}

// WriteShard writes one JSON object per line (JSONL), atomically: write to a
// temp file in the same directory, then rename - so a crash mid-write never
// leaves a corrupt or half-written shard file that Exists() would wrongly
// treat as complete.
func (s *Store) WriteShard(shardID int, results []models.ExtractionResult) error {
	tmp := s.shardPath(shardID) + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp shard file: %w", err)
	}
	enc := json.NewEncoder(f)
	for _, r := range results {
		if err := enc.Encode(r); err != nil {
			f.Close()
			os.Remove(tmp)
			return fmt.Errorf("encode row: %w", err)
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp shard file: %w", err)
	}
	if err := os.Rename(tmp, s.shardPath(shardID)); err != nil {
		return fmt.Errorf("rename temp shard file: %w", err)
	}
	return nil
}

// ReadShard reads a previously written shard back - used both by the
// embedding stage (normal forward flow) and by the resume path (skip LLM,
// load already-extracted results straight from disk).
func (s *Store) ReadShard(shardID int) ([]models.ExtractionResult, error) {
	f, err := os.Open(s.shardPath(shardID))
	if err != nil {
		return nil, fmt.Errorf("open shard file: %w", err)
	}
	defer f.Close()

	var out []models.ExtractionResult
	dec := json.NewDecoder(f)
	for dec.More() {
		var r models.ExtractionResult
		if err := dec.Decode(&r); err != nil {
			return nil, fmt.Errorf("decode row: %w", err)
		}
		out = append(out, r)
	}
	return out, nil
}

// SizeBytes returns the on-disk size of a shard file - used to report real
// storage-volume numbers (total bytes written across all shards).
func (s *Store) SizeBytes(shardID int) int64 {
	info, err := os.Stat(s.shardPath(shardID))
	if err != nil {
		return 0
	}
	return info.Size()
}

/*
Swapping this for a real GCS client later:

    type GCSStore struct {
        bucket *storage.BucketHandle
        prefix string
    }

    func (s *GCSStore) WriteShard(shardID int, results []models.ExtractionResult) error {
        obj := s.bucket.Object(fmt.Sprintf("%s/shard_%05d.jsonl", s.prefix, shardID))
        w := obj.NewWriter(ctx) // GCS writes are already atomic on Close (no rename needed)
        enc := json.NewEncoder(w)
        for _, r := range results { enc.Encode(r) }
        return w.Close()
    }

    func (s *GCSStore) Exists(shardID int) bool {
        _, err := s.bucket.Object(fmt.Sprintf("%s/shard_%05d.jsonl", s.prefix, shardID)).Attrs(ctx)
        return err == nil
    }

Same interface, same call sites in pipeline.go - only this file changes.
*/
