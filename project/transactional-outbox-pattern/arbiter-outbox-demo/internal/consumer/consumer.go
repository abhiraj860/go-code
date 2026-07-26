// Package consumer simulates the vector-ingestion consumer reading from the
// Kafka topic, including the redelivery pattern that causes wasted
// embedding compute in the first place.
//
// GenerateStreamWithRedeliveries models realistic at-least-once delivery:
// a consumer only commits its Kafka offset periodically (every
// crashInterval messages). If it crashes before a commit, every message
// since the last commit gets redelivered when it restarts (or when the
// consumer group rebalances). That redeliveryWindow is exactly what shows
// up as duplicate messages in the log.
package consumer

import (
	"fmt"

	"arbiter-outbox-demo/internal/vectorstore"
)

// GenerateStreamWithRedeliveries returns the sequence of ticket IDs a
// consumer actually receives, including duplicates, given numEvents unique
// tickets and a crash/redelivery pattern.
func GenerateStreamWithRedeliveries(numEvents, crashInterval, redeliveryWindow int) []string {
	var stream []string
	var sinceCommit []string

	for i := 0; i < numEvents; i++ {
		id := fmt.Sprintf("T-%d", i)
		stream = append(stream, id)
		sinceCommit = append(sinceCommit, id)

		if crashInterval > 0 && len(sinceCommit) >= crashInterval {
			// Simulate a crash/rebalance right before the offset commit:
			// Kafka redelivers everything processed since the last commit.
			n := redeliveryWindow
			if n > len(sinceCommit) {
				n = len(sinceCommit)
			}
			redelivered := sinceCommit[len(sinceCommit)-n:]
			stream = append(stream, redelivered...)
			sinceCommit = nil
		}
	}
	return stream
}

// RunNaiveConsumer processes every message in the stream and regenerates
// the embedding every time, including duplicates — this is the "before"
// state: no idempotency check.
func RunNaiveConsumer(stream []string, vs *vectorstore.VectorStore) {
	for _, id := range stream {
		vs.GenerateAndInsertEmbedding(id)
	}
}

// RunIdempotentConsumer processes every message but skips the (costly)
// embedding step for any ticket ID already seen, using a dedup set keyed by
// event/ticket ID — this stands in for a `processed_events` table or Redis
// SET checked before calling the embedding model.
func RunIdempotentConsumer(stream []string, vs *vectorstore.VectorStore) {
	seen := make(map[string]bool, len(stream))
	for _, id := range stream {
		if seen[id] {
			continue // duplicate delivery -> skip, no embedding regenerated
		}
		seen[id] = true
		vs.GenerateAndInsertEmbedding(id)
	}
}
