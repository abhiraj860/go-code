// Package notify publishes seat-state changes for the realtime gateway.
//
// This closes the loop the gateway was built for. Without it the gateway is
// architecturally complete and functionally dead: it subscribes to a channel
// nothing writes to, so browsers fall back to the seat map's 15-second poll and
// the WebSocket delivers nothing. The failure is silent -- every service is
// healthy, every metric is green, and seat maps are simply always a little
// stale.
package notify

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

// SeatStatus is the wire vocabulary the browser understands. Deliberately
// lower-case strings rather than the protobuf enum names: this crosses into
// TypeScript, where SEAT_STATUS_AVAILABLE would be noise.
type SeatStatus string

const (
	StatusAvailable SeatStatus = "available"
	StatusHeld      SeatStatus = "held"
	StatusSold      SeatStatus = "sold"
	StatusBlocked   SeatStatus = "blocked"
)

// Update is one seat changing state.
type Update struct {
	SeatID string     `json:"seatId"`
	Status SeatStatus `json:"status"`
	// Sequence is monotonic per event. The gateway drops frames whose sequence
	// is not greater than the last one it forwarded, which is what makes
	// out-of-order delivery across replicas safe.
	Sequence      int64  `json:"sequence"`
	HoldExpiresAt string `json:"holdExpiresAt,omitempty"`
}

// Publisher pushes seat changes onto Redis pub/sub.
type Publisher struct {
	client redis.UniversalClient
	logger *slog.Logger

	published, failed atomic.Uint64
}

func New(client redis.UniversalClient, logger *slog.Logger) *Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Publisher{client: client, logger: logger}
}

func channel(eventID string) string { return "seat.updates:" + eventID }

// sequenceKey holds the per-event counter.
//
// Redis INCR rather than a timestamp or an in-process counter: it is atomic
// across every inventory replica, so two pods updating the same event still
// produce a strictly increasing sequence. A per-process counter would restart
// at zero on deploy and make every subsequent frame look stale to the gateway.
func sequenceKey(eventID string) string { return "seat.seq:" + eventID }

// Publish announces a batch of seat changes for one event.
//
// Best effort by design. This is a UX optimisation on top of a system that is
// already correct: the seat map polls as a backstop and inventory remains the
// authority, so a failed publish costs a few seconds of freshness, never
// correctness. It must therefore never fail the caller's hold or release.
func (p *Publisher) Publish(ctx context.Context, eventID string, updates []Update) {
	if len(updates) == 0 || p.client == nil {
		return
	}

	// One INCR for the whole batch: every seat in a single hold changes at the
	// same logical moment, so they share a sequence and the gateway forwards
	// them together rather than dropping all but the first.
	seq, err := p.client.Incr(ctx, sequenceKey(eventID)).Result()
	if err != nil {
		p.failed.Add(1)
		p.logger.WarnContext(ctx, "could not allocate a seat-update sequence; skipping publish",
			slog.String("event_id", eventID), slog.Any("error", err))
		return
	}

	pipe := p.client.Pipeline()
	for _, u := range updates {
		u.Sequence = seq
		payload, err := json.Marshal(u)
		if err != nil {
			p.failed.Add(1)
			continue
		}
		pipe.Publish(ctx, channel(eventID), payload)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		p.failed.Add(1)
		p.logger.WarnContext(ctx, "publishing seat updates failed; browsers will refresh on their next poll",
			slog.String("event_id", eventID), slog.Any("error", err))
		return
	}

	p.published.Add(uint64(len(updates)))
}

// PublishStatus is the common single-status case.
func (p *Publisher) PublishStatus(ctx context.Context, eventID string, seatIDs []string, status SeatStatus, holdExpiresAt string) {
	if len(seatIDs) == 0 {
		return
	}
	updates := make([]Update, 0, len(seatIDs))
	for _, id := range seatIDs {
		updates = append(updates, Update{SeatID: id, Status: status, HoldExpiresAt: holdExpiresAt})
	}
	p.Publish(ctx, eventID, updates)
}

// Stats reports publishing activity.
type Stats struct {
	Published uint64
	Failed    uint64
}

func (p *Publisher) Stats() Stats {
	return Stats{Published: p.published.Load(), Failed: p.failed.Load()}
}

// Channel exposes the channel name, so tests and tooling do not hard-code it.
func Channel(eventID string) string { return channel(eventID) }
