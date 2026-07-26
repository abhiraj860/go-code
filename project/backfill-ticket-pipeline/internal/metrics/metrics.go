// Package metrics tracks the counters that back every resume bullet number.
package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

type Collector struct {
	StartTime time.Time

	RowsExtracted   int64
	RowsEmbedded    int64
	RowsWritten     int64

	ExtractionCalls int64 // number of BATCH calls (proxy for API cost)
	EmbeddingCalls  int64

	ExtractionRetries int64
	EmbeddingRetries  int64
	MilvusRetries     int64

	ShardsFailed int64
	ShardsDone   int64

	mu          sync.Mutex
	latenciesMs []int64 // per-shard end-to-end latency, for p50/p95
}

func NewCollector() *Collector {
	return &Collector{StartTime: time.Now()}
}

func (c *Collector) RecordShardLatency(ms int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latenciesMs = append(c.latenciesMs, ms)
}

func (c *Collector) Snapshot() Snapshot {
	elapsed := time.Since(c.StartTime).Seconds()
	rowsWritten := atomic.LoadInt64(&c.RowsWritten)
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(rowsWritten) / elapsed
	}

	c.mu.Lock()
	lat := append([]int64(nil), c.latenciesMs...)
	c.mu.Unlock()

	p50, p95 := percentile(lat, 0.50), percentile(lat, 0.95)

	return Snapshot{
		ElapsedSeconds:    elapsed,
		RowsExtracted:     atomic.LoadInt64(&c.RowsExtracted),
		RowsEmbedded:      atomic.LoadInt64(&c.RowsEmbedded),
		RowsWritten:       rowsWritten,
		ExtractionCalls:   atomic.LoadInt64(&c.ExtractionCalls),
		EmbeddingCalls:    atomic.LoadInt64(&c.EmbeddingCalls),
		ExtractionRetries: atomic.LoadInt64(&c.ExtractionRetries),
		EmbeddingRetries:  atomic.LoadInt64(&c.EmbeddingRetries),
		MilvusRetries:     atomic.LoadInt64(&c.MilvusRetries),
		ShardsFailed:      atomic.LoadInt64(&c.ShardsFailed),
		ShardsDone:        atomic.LoadInt64(&c.ShardsDone),
		ThroughputRowsSec: throughput,
		P50LatencyMs:      p50,
		P95LatencyMs:      p95,
	}
}

type Snapshot struct {
	ElapsedSeconds    float64
	RowsExtracted     int64
	RowsEmbedded      int64
	RowsWritten       int64
	ExtractionCalls   int64
	EmbeddingCalls    int64
	ExtractionRetries int64
	EmbeddingRetries  int64
	MilvusRetries     int64
	ShardsFailed      int64
	ShardsDone        int64
	ThroughputRowsSec float64
	P50LatencyMs      int64
	P95LatencyMs      int64
}

func percentile(vals []int64, p float64) int64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := append([]int64(nil), vals...)
	// simple insertion sort is fine; shard counts are in the hundreds/thousands
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1] > sorted[j]; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}
