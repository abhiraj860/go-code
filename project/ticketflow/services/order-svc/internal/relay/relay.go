// Package relay delivers outbox rows to Kafka.
//
// It is the second half of the transactional outbox: order-svc writes messages
// atomically with the orders they describe, and this moves them onto the bus.
package relay

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/domain"
)

// Store is the outbox side of the repository.
type Store interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	ClaimOutbox(ctx context.Context, tx pgx.Tx, limit int) ([]domain.OutboxRecord, error)
	MarkPublished(ctx context.Context, tx pgx.Tx, ids []int64) error
	RecordFailure(ctx context.Context, tx pgx.Tx, ids []int64, cause string) error
	PendingCount(ctx context.Context) (int, error)
}

// Publisher sends messages to the bus.
type Publisher interface {
	PublishBatch(ctx context.Context, msgs []tfkafka.Message) error
}

// Relay polls the outbox and publishes.
type Relay struct {
	store  Store
	pub    Publisher
	logger *slog.Logger

	interval  time.Duration
	batchSize int

	published, failedBatches, sweeps atomic.Uint64
}

type Options struct {
	Store     Store
	Publisher Publisher
	Logger    *slog.Logger
	// Interval between sweeps when the outbox is empty. Kept short: a buyer is
	// waiting for a ticket, and this delay is added to their wait.
	Interval time.Duration
	// BatchSize bounds one sweep. Larger batches amortise the round-trip but
	// hold the row locks longer.
	BatchSize int
}

func New(opts Options) *Relay {
	if opts.Interval <= 0 {
		opts.Interval = 200 * time.Millisecond
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Relay{
		store:     opts.Store,
		pub:       opts.Publisher,
		logger:    opts.Logger,
		interval:  opts.Interval,
		batchSize: opts.BatchSize,
	}
}

// Run sweeps until ctx is cancelled.
//
// When a sweep publishes a full batch it loops immediately rather than waiting
// for the next tick, so a backlog drains at full speed instead of at one batch
// per interval.
func (r *Relay) Run(ctx context.Context) {
	r.logger.Info("outbox relay starting",
		slog.Duration("interval", r.interval), slog.Int("batch_size", r.batchSize))

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("outbox relay stopped")
			return
		default:
		}

		n, err := r.sweep(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			r.logger.Error("outbox sweep failed", slog.Any("error", err))
		}

		// A full batch means there is probably more waiting; keep going.
		if n == r.batchSize {
			continue
		}

		select {
		case <-time.After(r.interval):
		case <-ctx.Done():
			r.logger.Info("outbox relay stopped")
			return
		}
	}
}

// sweep claims, publishes and marks one batch, returning how many were sent.
//
// The ordering here is deliberate and is where at-least-once comes from:
//
//	BEGIN -> claim (FOR UPDATE SKIP LOCKED) -> publish to Kafka -> mark -> COMMIT
//
// A crash after the publish but before the commit leaves the rows unmarked, so
// the next sweep republishes them. That is a duplicate, which consumers dedup
// on the envelope id. The alternative -- mark first, then publish -- would turn
// the same crash into a lost message, and a lost order.created means a customer
// paid and never received a ticket. Duplicates are recoverable; losses are not.
func (r *Relay) sweep(ctx context.Context) (int, error) {
	r.sweeps.Add(1)

	tx, err := r.store.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	records, err := r.store.ClaimOutbox(ctx, tx, r.batchSize)
	if err != nil {
		return 0, err
	}
	if len(records) == 0 {
		return 0, nil
	}

	msgs := make([]tfkafka.Message, 0, len(records))
	ids := make([]int64, 0, len(records))
	for _, rec := range records {
		msgs = append(msgs, tfkafka.Message{
			Topic:   rec.Topic,
			Key:     rec.MessageKey,
			Value:   rec.Payload,
			Headers: rec.Headers,
		})
		ids = append(ids, rec.ID)
	}

	if err := r.pub.PublishBatch(ctx, msgs); err != nil {
		// Record the failure and COMMIT that bookkeeping, so the attempt count
		// survives. Rolling back instead would lose the evidence and a
		// permanently failing row would look healthy forever.
		if recErr := r.store.RecordFailure(ctx, tx, ids, err.Error()); recErr != nil {
			r.logger.Error("recording outbox failure failed", slog.Any("error", recErr))
		} else if commitErr := tx.Commit(ctx); commitErr != nil {
			r.logger.Error("committing failure bookkeeping failed", slog.Any("error", commitErr))
		}
		r.failedBatches.Add(1)
		return 0, err
	}

	if err := r.store.MarkPublished(ctx, tx, ids); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		// Published but not marked. The next sweep resends; consumers dedup.
		r.logger.Warn("outbox marked-published commit failed; messages will be resent",
			slog.Int("count", len(ids)), slog.Any("error", err))
		return 0, err
	}

	r.published.Add(uint64(len(records)))
	r.logger.Debug("outbox batch published", slog.Int("count", len(records)))
	return len(records), nil
}

// Stats reports relay activity.
type Stats struct {
	Published     uint64
	FailedBatches uint64
	Sweeps        uint64
	// Pending is the backlog. The metric that matters: if it climbs, orders
	// are being accepted that nothing is fulfilling.
	Pending int
}

func (r *Relay) Stats(ctx context.Context) Stats {
	pending, err := r.store.PendingCount(ctx)
	if err != nil {
		pending = -1 // distinguishable from a genuine zero
	}
	return Stats{
		Published:     r.published.Load(),
		FailedBatches: r.failedBatches.Load(),
		Sweeps:        r.sweeps.Load(),
		Pending:       pending,
	}
}

// ErrNoPublisher guards a misconfigured relay.
var ErrNoPublisher = errors.New("relay: publisher is required")
