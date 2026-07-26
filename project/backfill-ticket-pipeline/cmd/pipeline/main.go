package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/time/rate"

	"ticket-pipeline/internal/checkpoint"
	"ticket-pipeline/internal/metrics"
	"ticket-pipeline/internal/pipeline"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: pipeline <run|report|reset> [flags]")
		os.Exit(1)
	}
	cmd := os.Args[1]

	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	totalRows := fs.Int("rows", 9000, "total rows to process (use 900000 for the real run)")
	shardSize := fs.Int("shard-size", 100, "rows per shard")
	concurrency := fs.Int("concurrency", 20, "max shards processed in parallel")
	maxRetries := fs.Int("max-retries", 4, "max retries per stage per shard")
	dbPath := fs.String("db", "checkpoint.db", "path to checkpoint sqlite db")
	blobDir := fs.String("blob-dir", "data/extractions", "local dir acting as the blob store for extraction JSONL shards (swap for a GCS/S3 bucket path in prod)")
	baseline := fs.Bool("baseline", false, "also run with concurrency=1 to compute speedup")
	adaptiveRate := fs.Bool("adaptive-rate", true, "use AIMD adaptive rate limiting (false = fixed-rate limiter, for comparison)")
	fs.Parse(os.Args[2:])

	switch cmd {
	case "reset":
		os.Remove(*dbPath)
		os.RemoveAll(*blobDir)
		fmt.Println("checkpoint db and blob dir removed:", *dbPath, *blobDir)

	case "report":
		store, err := checkpoint.NewStore(*dbPath)
		must(err)
		defer store.Close()
		sum, err := store.Summarize()
		must(err)
		printReport(sum)

	case "run":
		store, err := checkpoint.NewStore(*dbPath)
		must(err)
		defer store.Close()

		cfg := pipeline.Config{
			TotalRows:          *totalRows,
			ShardSize:          *shardSize,
			Concurrency:        *concurrency,
			MaxRetries:         *maxRetries,
			InitialRate:        rate.Limit(50),
			MinRate:            rate.Limit(5),
			MaxRate:            rate.Limit(200),
			Burst:              10,
			ExtractFailPct:     0.03,
			ExtractThrottlePct: 0.05,
			EmbedFailPct:       0.02,
			EmbedThrottlePct:   0.05,
			MilvusFailPct:      0.01,
			AdaptiveRateEnabled: *adaptiveRate,
			BlobDir:            *blobDir,
		}

		if *baseline {
			// Run once single-threaded to get a real baseline for the
			// "speedup vs single-threaded" resume number, then reset and
			// run the real distributed version.
			log.Println("=== BASELINE RUN (concurrency=1) ===")
			os.Remove(*dbPath)
			os.RemoveAll(*blobDir)
			baseStore, _ := checkpoint.NewStore(*dbPath)
			baseCfg := cfg
			baseCfg.Concurrency = 1
			start := time.Now()
			p, err := pipeline.New(baseCfg, baseStore)
			must(err)
			must(p.Run(context.Background()))
			baseElapsed := time.Since(start)
			baseStore.Close()
			os.Remove(*dbPath)
			os.RemoveAll(*blobDir)
			log.Printf("baseline elapsed: %s\n", baseElapsed)

			log.Println("=== DISTRIBUTED RUN (concurrency=", *concurrency, ") ===")
			store2, _ := checkpoint.NewStore(*dbPath)
			defer store2.Close()
			start2 := time.Now()
			p2, err := pipeline.New(cfg, store2)
			must(err)
			must(p2.Run(context.Background()))
			distElapsed := time.Since(start2)

			speedup := float64(baseElapsed) / float64(distElapsed)
			fmt.Printf("\n=== SPEEDUP SUMMARY ===\nbaseline (1 worker):     %s\ndistributed (%d workers): %s\nspeedup:                 %.2fx\n",
				baseElapsed, *concurrency, distElapsed, speedup)

			snap := p2.Metrics().Snapshot()
			printSnapshot(snap)
			printBlobStats(*blobDir)
			return
		}

		start := time.Now()
		p, err := pipeline.New(cfg, store)
		must(err)
		must(p.Run(context.Background()))
		elapsed := time.Since(start)

		snap := p.Metrics().Snapshot()
		fmt.Printf("\n=== RUN COMPLETE in %s ===\n", elapsed)
		printSnapshot(snap)
		printBlobStats(*blobDir)

		sum, _ := store.Summarize()
		printReport(sum)

	default:
		fmt.Println("unknown command:", cmd)
		os.Exit(1)
	}
}

func printSnapshot(s metrics.Snapshot) {
	fmt.Println("\n=== RUN METRICS (source for resume numbers) ===")
	fmt.Printf("elapsed seconds:       %.2f\n", s.ElapsedSeconds)
	fmt.Printf("rows extracted:        %d\n", s.RowsExtracted)
	fmt.Printf("rows embedded:         %d\n", s.RowsEmbedded)
	fmt.Printf("rows written to milvus:%d\n", s.RowsWritten)
	fmt.Printf("throughput (rows/sec): %.2f\n", s.ThroughputRowsSec)
	fmt.Printf("extraction API calls:  %d\n", s.ExtractionCalls)
	fmt.Printf("embedding API calls:   %d\n", s.EmbeddingCalls)
	fmt.Printf("extraction retries:    %d\n", s.ExtractionRetries)
	fmt.Printf("embedding retries:     %d\n", s.EmbeddingRetries)
	fmt.Printf("milvus retries:        %d\n", s.MilvusRetries)
	fmt.Printf("shards done:           %d\n", s.ShardsDone)
	fmt.Printf("shards failed:         %d\n", s.ShardsFailed)
	fmt.Printf("p50 shard latency ms:  %d\n", s.P50LatencyMs)
	fmt.Printf("p95 shard latency ms:  %d\n", s.P95LatencyMs)
}

func printReport(sum checkpoint.Summary) {
	fmt.Println("\n=== CHECKPOINT DB SUMMARY (resume-safe state) ===")
	fmt.Printf("total shards:        %d\n", sum.TotalShards)
	fmt.Printf("extraction done:     %d\n", sum.ExtractionDone)
	fmt.Printf("extraction failed:   %d\n", sum.ExtractionFailed)
	fmt.Printf("embedding done:      %d\n", sum.EmbeddingDone)
	fmt.Printf("embedding failed:    %d\n", sum.EmbeddingFailed)
	fmt.Printf("total retries:       %d\n", sum.TotalRetries)
	fmt.Printf("rows processed:      %d\n", sum.TotalRowsProcessed)
	fmt.Printf("rows successful:     %d\n", sum.SuccessfulRows)
	if sum.TotalRowsProcessed > 0 {
		fmt.Printf("success rate:        %.3f%%\n", 100*float64(sum.SuccessfulRows)/float64(sum.TotalRowsProcessed))
	}
}

func printBlobStats(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var totalBytes int64
	var count int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		totalBytes += info.Size()
		count++
	}
	fmt.Println("\n=== BLOB STORE STATS (real files on disk, dir:", dir, ") ===")
	fmt.Printf("shard files:           %d\n", count)
	fmt.Printf("total bytes:           %d (%.2f MB)\n", totalBytes, float64(totalBytes)/1e6)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
