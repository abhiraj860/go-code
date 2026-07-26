// Package pipeline wires together checkpointing, rate limiting, retries, and
// the two processing stages (extraction, embedding) into a single distributed,
// idempotent, resumable backfill job.
package pipeline

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"ticket-pipeline/internal/blobstore"
	"ticket-pipeline/internal/checkpoint"
	"ticket-pipeline/internal/clients"
	"ticket-pipeline/internal/metrics"
	"ticket-pipeline/internal/models"
	"ticket-pipeline/internal/ratelimit"
)

type Config struct {
	TotalRows      int
	ShardSize      int
	Concurrency    int // max shards processed in parallel (bounded worker pool)
	MaxRetries     int
	InitialRate    rate.Limit // requests/sec, shared across all workers per stage
	MinRate        rate.Limit
	MaxRate        rate.Limit
	Burst          int
	ExtractFailPct float64
	ExtractThrottlePct float64
	EmbedFailPct   float64
	EmbedThrottlePct float64
	MilvusFailPct  float64
	AdaptiveRateEnabled bool // when false, limiter never adjusts - used as the "before" comparison arm
	BlobDir        string   // local dir acting as the blob store (swap for a GCS/S3 bucket path in prod)
}

type Pipeline struct {
	cfg   Config
	store *checkpoint.Store
	src   *clients.BigQuerySource
	blob  *blobstore.Store
	llm   *clients.LLMClient
	embed *clients.EmbeddingClient
	milvus *clients.MilvusSink

	extractLimiter *ratelimit.AdaptiveLimiter
	embedLimiter   *ratelimit.AdaptiveLimiter

	metrics *metrics.Collector
}

func New(cfg Config, store *checkpoint.Store) (*Pipeline, error) {
	blob, err := blobstore.NewStore(cfg.BlobDir)
	if err != nil {
		return nil, err
	}
	return &Pipeline{
		cfg:   cfg,
		store: store,
		src:   &clients.BigQuerySource{TotalRows: cfg.TotalRows, ShardSize: cfg.ShardSize},
		blob:  blob,
		llm:   clients.NewLLMClient(cfg.ExtractFailPct, cfg.ExtractThrottlePct),
		embed: clients.NewEmbeddingClient(cfg.EmbedFailPct, cfg.EmbedThrottlePct, 128),
		milvus: clients.NewMilvusSink(cfg.MilvusFailPct),

		extractLimiter: ratelimit.NewAdaptiveLimiter(cfg.InitialRate, cfg.MinRate, cfg.MaxRate, cfg.Burst),
		embedLimiter:   ratelimit.NewAdaptiveLimiter(cfg.InitialRate, cfg.MinRate, cfg.MaxRate, cfg.Burst),

		metrics: metrics.NewCollector(),
	}, nil
}

func (p *Pipeline) Metrics() *metrics.Collector { return p.metrics }

// Run processes all shards through both stages using a bounded worker pool.
// It is SAFE TO RE-RUN: any shard already marked "done" for a stage is skipped,
// which is what makes the whole job idempotent and crash-resumable.
func (p *Pipeline) Run(ctx context.Context) error {
	numShards := p.src.NumShards()
	log.Printf("total shards: %d (shard size %d, total rows %d)", numShards, p.cfg.ShardSize, p.cfg.TotalRows)

	sem := make(chan struct{}, p.cfg.Concurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, numShards)

	for shardID := 0; shardID < numShards; shardID++ {
		sem <- struct{}{}
		wg.Add(1)
		go func(sid int) {
			defer wg.Done()
			defer func() { <-sem }()

			start := time.Now()
			err := p.processShard(ctx, sid)
			elapsedMs := time.Since(start).Milliseconds()
			p.metrics.RecordShardLatency(elapsedMs)

			if err != nil {
				p.metrics.ShardsFailed++
				p.store.RecordEvent("shard", sid, elapsedMs, p.cfg.ShardSize, false)
				errCh <- fmt.Errorf("shard %d: %w", sid, err)
				return
			}
			p.metrics.ShardsDone++
			p.store.RecordEvent("shard", sid, elapsedMs, p.cfg.ShardSize, true)
		}(shardID)
	}

	wg.Wait()
	close(errCh)

	var failCount int
	for err := range errCh {
		log.Printf("SHARD FAILURE (dead-lettered, pipeline continues): %v", err)
		failCount++
	}
	if failCount > 0 {
		log.Printf("%d/%d shards failed after retries - see checkpoint DB for dead-letter list", failCount, numShards)
	}
	return nil
}

// processShard runs stage 1 (extraction) then stage 2 (embedding+milvus write)
// for one shard, with per-stage retry+backoff and idempotency-based cleanup
// on partial failure (see clients.MilvusSink.Delete doc comment).
func (p *Pipeline) processShard(ctx context.Context, shardID int) error {
	// ---- Stage 1: LLM extraction ----
	done, err := p.store.IsStageDone(shardID, "extraction")
	if err != nil {
		return err
	}
	var extracted []models.ExtractionResult
	if done && p.blob.Exists(shardID) {
		// Resume path: the shard's JSONL file is already sitting in blob
		// storage from a prior run - read it back directly instead of
		// re-calling the LLM. This is the idempotency guarantee in action,
		// and the main cost/time saver on a re-run after a crash.
		extracted, err = p.blob.ReadShard(shardID)
		if err != nil {
			return fmt.Errorf("resume read from blob store: %w", err)
		}
	} else {
		extracted, err = p.runExtraction(ctx, shardID)
		if err != nil {
			p.store.UpdateStage(shardID, "extraction", models.StatusFailed, err.Error())
			return fmt.Errorf("extraction: %w", err)
		}
		p.store.UpdateStage(shardID, "extraction", models.StatusDone, "")
	}

	// ---- Stage 2: embedding + Milvus upsert ----
	done, err = p.store.IsStageDone(shardID, "embedding")
	if err != nil {
		return err
	}
	if done {
		return nil // already fully processed - true no-op on resume
	}

	records, err := p.runEmbeddingAndWrite(ctx, shardID, extracted)
	if err != nil {
		// Cleanup on partial failure: if embedding succeeded but the Milvus
		// write failed partway through the batch, delete any vectors this
		// shard did manage to write before retrying/failing the shard.
		// This is NOT a saga/distributed-transaction pattern - it's a simple
		// idempotency guard. Milvus upserts are already keyed by a
		// deterministic ID (hash of ticket ID), so a retry naturally
		// overwrites the same rows rather than duplicating them. The delete
		// here just avoids leaving a half-written batch sitting around
		// between now and the next retry; it is not required for
		// correctness, only for tidiness.
		if len(records) > 0 {
			ids := make([]string, len(records))
			for i, r := range records {
				ids[i] = r.ID
			}
			if compErr := p.milvus.Delete(ctx, ids); compErr != nil {
				log.Printf("shard %d: partial-write cleanup also failed (safe to ignore - retry will overwrite by ID): %v", shardID, compErr)
			}
		}
		p.store.UpdateStage(shardID, "embedding", models.StatusFailed, err.Error())
		return fmt.Errorf("embedding: %w", err)
	}

	p.store.UpdateStage(shardID, "embedding", models.StatusDone, "")
	return nil
}

func (p *Pipeline) runExtraction(ctx context.Context, shardID int) ([]models.ExtractionResult, error) {
	tickets, err := p.src.ReadShard(ctx, shardID)
	if err != nil {
		return nil, fmt.Errorf("bigquery read: %w", err)
	}
	if len(tickets) == 0 {
		return nil, nil
	}

	var results []models.ExtractionResult
	op := func() error {
		if err := p.extractLimiter.Wait(ctx); err != nil {
			return err
		}
		p.metrics.ExtractionCalls++
		r, err := p.llm.ExtractBatch(ctx, tickets)
		if err == clients.ErrThrottled {
			if p.cfg.AdaptiveRateEnabled {
				p.extractLimiter.ReportThrottle()
			}
			return err
		}
		if err != nil {
			return err
		}
		if p.cfg.AdaptiveRateEnabled {
			p.extractLimiter.ReportSuccess()
		}
		results = r
		return nil
	}

	if err := p.retry(ctx, op, &p.metrics.ExtractionRetries); err != nil {
		return nil, err
	}
	// Write to the blob store (real file I/O here; a real GCS/S3 object
	// write in prod - see blobstore.go). Atomic write-then-rename means a
	// crash here never leaves a shard file that Exists() would wrongly
	// treat as a completed, resumable shard.
	if err := p.blob.WriteShard(shardID, results); err != nil {
		return nil, fmt.Errorf("blob store write: %w", err)
	}
	p.metrics.RowsExtracted += int64(len(results))
	return results, nil
}

func (p *Pipeline) runEmbeddingAndWrite(ctx context.Context, shardID int, extracted []models.ExtractionResult) ([]models.EmbeddingRecord, error) {
	if len(extracted) == 0 {
		return nil, nil
	}

	var records []models.EmbeddingRecord
	op := func() error {
		if err := p.embedLimiter.Wait(ctx); err != nil {
			return err
		}
		p.metrics.EmbeddingCalls++
		r, err := p.embed.EmbedBatch(ctx, extracted)
		if err == clients.ErrThrottled {
			if p.cfg.AdaptiveRateEnabled {
				p.embedLimiter.ReportThrottle()
			}
			return err
		}
		if err != nil {
			return err
		}
		if p.cfg.AdaptiveRateEnabled {
			p.embedLimiter.ReportSuccess()
		}
		records = r
		return nil
	}
	if err := p.retry(ctx, op, &p.metrics.EmbeddingRetries); err != nil {
		return nil, err
	}
	p.metrics.RowsEmbedded += int64(len(records))

	// Milvus upsert - deterministic IDs mean re-running this is idempotent.
	writeOp := func() error {
		return p.milvus.Upsert(ctx, records)
	}
	if err := p.retry(ctx, writeOp, &p.metrics.MilvusRetries); err != nil {
		return records, err // return records so caller can compensate
	}
	p.metrics.RowsWritten += int64(len(records))
	return records, nil
}

// retry wraps an operation with exponential backoff up to MaxRetries.
func (p *Pipeline) retry(ctx context.Context, op func() error, counter *int64) error {
	var err error
	for attempt := 0; attempt <= p.cfg.MaxRetries; attempt++ {
		if err = op(); err == nil {
			return nil
		}
		if attempt < p.cfg.MaxRetries {
			*counter++
			ratelimit.Backoff(ctx, attempt, 50*time.Millisecond, 2*time.Second)
		}
	}
	return err
}
