// Package service holds inventory's business rules.
//
// Nothing here caches. Availability is the one thing in TicketFlow that must
// always read through to the source of truth.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/abhiraj860/ticketflow/services/inventory-svc/internal/domain"
)

const (
	// defaultHoldTTL is how long a buyer gets to complete checkout.
	defaultHoldTTL = 2 * time.Minute
	// maxHoldTTL caps what a client may request. Without a ceiling, a caller
	// could park the entire front row for an hour.
	maxHoldTTL = 10 * time.Minute
	// minHoldTTL guards against a TTL so short the buyer cannot possibly pay.
	minHoldTTL = 15 * time.Second

	// maxSeatsPerHold limits one request. Real ticketing systems cap party
	// size; it also bounds the row-lock footprint of a single transaction.
	maxSeatsPerHold = 10
)

// Repo is inventory's data dependency.
type Repo interface {
	GetAvailability(ctx context.Context, eventID string, seatIDs []string) ([]domain.SeatAvailability, error)
	HoldSeats(ctx context.Context, req domain.HoldRequest) (domain.HoldResult, error)
	ReleaseHold(ctx context.Context, holdID string) (int, error)
	ConfirmHold(ctx context.Context, holdID, orderID string) ([]string, error)
	ReapExpiredHolds(ctx context.Context, limit int) (int, error)
	SeedSeats(ctx context.Context, eventID string, seatIDs []string) (int, error)
}

// SeatLocker is the advisory Redis fast path.
//
// Optional: a nil locker means every request goes straight to Postgres, which
// is correct, just less efficient under contention. Implementations must never
// let a lock failure become a grant.
type SeatLocker interface {
	Acquire(ctx context.Context, eventID string, seatIDs []string, owner string, ttl time.Duration) (acquired, rejected []string, err error)
	Release(ctx context.Context, eventID string, seatIDs []string, owner string) error
}

// Inventory serves seat availability and holds.
type Inventory struct {
	repo   Repo
	locker SeatLocker
	logger *slog.Logger

	defaultTTL time.Duration
	maxTTL     time.Duration
}

type Options struct {
	Repo Repo
	// Locker is optional. Without it the service is still correct; Postgres
	// simply absorbs the losing requests too.
	Locker     SeatLocker
	Logger     *slog.Logger
	DefaultTTL time.Duration
	MaxTTL     time.Duration
}

func New(opts Options) *Inventory {
	if opts.DefaultTTL <= 0 {
		opts.DefaultTTL = defaultHoldTTL
	}
	if opts.MaxTTL <= 0 {
		opts.MaxTTL = maxHoldTTL
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Inventory{
		repo:       opts.Repo,
		locker:     opts.Locker,
		logger:     opts.Logger,
		defaultTTL: opts.DefaultTTL,
		maxTTL:     opts.MaxTTL,
	}
}

// GetAvailability returns current seat state, always read through to Postgres.
func (i *Inventory) GetAvailability(ctx context.Context, eventID string, seatIDs []string) ([]domain.SeatAvailability, error) {
	if eventID == "" {
		return nil, fmt.Errorf("%w: event id is required", domain.ErrInvalidArgument)
	}
	return i.repo.GetAvailability(ctx, eventID, seatIDs)
}

// HoldSeats claims seats for a buyer.
func (i *Inventory) HoldSeats(ctx context.Context, req domain.HoldRequest) (domain.HoldResult, error) {
	switch {
	case req.EventID == "":
		return domain.HoldResult{}, fmt.Errorf("%w: event id is required", domain.ErrInvalidArgument)
	case req.UserID == "":
		return domain.HoldResult{}, fmt.Errorf("%w: user id is required", domain.ErrInvalidArgument)
	case len(req.SeatIDs) == 0:
		return domain.HoldResult{}, fmt.Errorf("%w: at least one seat is required", domain.ErrInvalidArgument)
	case len(req.SeatIDs) > maxSeatsPerHold:
		return domain.HoldResult{}, fmt.Errorf("%w: at most %d seats per hold, got %d",
			domain.ErrInvalidArgument, maxSeatsPerHold, len(req.SeatIDs))
	case req.IdempotencyKey == "":
		// Required, not optional. Without it a network retry silently acquires
		// a second set of seats and the buyer is charged twice.
		return domain.HoldResult{}, fmt.Errorf("%w: idempotency key is required", domain.ErrInvalidArgument)
	}

	if dup := firstDuplicate(req.SeatIDs); dup != "" {
		// A duplicate would make RETURNING report fewer rows than requested and
		// look like contention loss. Reject it as the client bug it is.
		return domain.HoldResult{}, fmt.Errorf("%w: seat %q requested twice", domain.ErrInvalidArgument, dup)
	}

	req.TTL = i.clampTTL(req.TTL)

	// Redis fast path. Everything below obeys one rule: Redis may only cause a
	// REJECTION, never a GRANT. Postgres decides who actually owns a seat.
	var (
		lockRejected []string
		// The owner is the logical attempt, not the process, so a retry of the
		// same request re-acquires its own locks instead of colliding with them.
		lockOwner = req.UserID + ":" + req.IdempotencyKey
	)

	if i.locker != nil {
		acquired, rejected, err := i.locker.Acquire(ctx, req.EventID, req.SeatIDs, lockOwner, req.TTL)
		switch {
		case err != nil:
			// Redis is unusable. Fall through to Postgres with the full seat
			// list -- exactly the behaviour with no Redis at all. Never treat a
			// lock failure as a rejection.
			i.logger.WarnContext(ctx, "seat lock unavailable, proceeding without the fast path",
				slog.Any("error", err))
		case len(acquired) == 0:
			// Every seat is locked by someone else. This is the case the fast
			// path exists for: reject without opening a transaction.
			return domain.HoldResult{RejectedSeatIDs: rejected}, domain.ErrNoSeatsAvailable
		default:
			req.SeatIDs = acquired
			lockRejected = rejected
		}
	}

	result, err := i.repo.HoldSeats(ctx, req)
	if err != nil {
		// Postgres said no. Drop the locks we took so the seats are not held
		// out of circulation until their TTL lapses.
		i.releaseLocks(ctx, req.EventID, req.SeatIDs, lockOwner)
		if len(lockRejected) > 0 {
			result.RejectedSeatIDs = append(result.RejectedSeatIDs, lockRejected...)
		}
		return result, err
	}

	// Postgres is the authority: any seat we locked but did not win must have
	// its lock released immediately.
	if lost := difference(req.SeatIDs, result.Hold.SeatIDs); len(lost) > 0 {
		i.releaseLocks(ctx, req.EventID, lost, lockOwner)
	}

	// Report every rejection, whether it came from Redis or Postgres, so the UI
	// can tell the buyer exactly which seats they missed.
	if len(lockRejected) > 0 {
		result.RejectedSeatIDs = append(result.RejectedSeatIDs, lockRejected...)
	}

	if result.Replayed {
		i.logger.InfoContext(ctx, "hold replayed from idempotency key",
			slog.String("hold_id", result.Hold.ID), slog.String("user_id", req.UserID))
	}
	return result, nil
}

// releaseLocks is best effort: a failure only means seats stay locked until
// their TTL lapses, which costs a brief spurious rejection, not correctness.
func (i *Inventory) releaseLocks(ctx context.Context, eventID string, seatIDs []string, owner string) {
	if i.locker == nil || len(seatIDs) == 0 {
		return
	}
	if err := i.locker.Release(ctx, eventID, seatIDs, owner); err != nil {
		i.logger.WarnContext(ctx, "releasing seat locks failed; they will expire on their own",
			slog.Any("error", err))
	}
}

// difference returns items in a that are absent from b.
func difference(a, b []string) []string {
	if len(b) == 0 {
		return a
	}
	inB := make(map[string]struct{}, len(b))
	for _, s := range b {
		inB[s] = struct{}{}
	}
	var out []string
	for _, s := range a {
		if _, ok := inB[s]; !ok {
			out = append(out, s)
		}
	}
	return out
}

// clampTTL bounds a requested lifetime rather than rejecting it, so a client
// asking for something unreasonable still gets a working hold.
func (i *Inventory) clampTTL(ttl time.Duration) time.Duration {
	switch {
	case ttl <= 0:
		return i.defaultTTL
	case ttl < minHoldTTL:
		return minHoldTTL
	case ttl > i.maxTTL:
		return i.maxTTL
	default:
		return ttl
	}
}

// ReleaseHold returns seats to the pool.
//
// Note it does NOT clear the Redis locks: the repo reports only a count, not
// which seats, and the lock owner string is not recoverable from a hold id.
// The locks lapse on their own TTL, which is set to the hold TTL, so the window
// is bounded and the failure mode is a brief spurious rejection -- the safe
// direction. Phase 7 can revisit this if the load test shows it matters.
func (i *Inventory) ReleaseHold(ctx context.Context, holdID string) (int, error) {
	if holdID == "" {
		return 0, fmt.Errorf("%w: hold id is required", domain.ErrInvalidArgument)
	}
	return i.repo.ReleaseHold(ctx, holdID)
}

// ConfirmHold converts a hold into a sale.
func (i *Inventory) ConfirmHold(ctx context.Context, holdID, orderID string) ([]string, error) {
	switch {
	case holdID == "":
		return nil, fmt.Errorf("%w: hold id is required", domain.ErrInvalidArgument)
	case orderID == "":
		return nil, fmt.Errorf("%w: order id is required", domain.ErrInvalidArgument)
	}
	return i.repo.ConfirmHold(ctx, holdID, orderID)
}

// SeedSeats registers an event's seats as available.
func (i *Inventory) SeedSeats(ctx context.Context, eventID string, seatIDs []string) (int, error) {
	if eventID == "" {
		return 0, fmt.Errorf("%w: event id is required", domain.ErrInvalidArgument)
	}
	if len(seatIDs) == 0 {
		return 0, fmt.Errorf("%w: at least one seat is required", domain.ErrInvalidArgument)
	}
	return i.repo.SeedSeats(ctx, eventID, seatIDs)
}

// RunReaper frees lapsed holds until ctx is cancelled.
//
// Correctness does not depend on this: HoldSeats already treats an expired hold
// as claimable. It runs so availability reads reflect reality promptly rather
// than showing phantom holds on the seat map.
func (i *Inventory) RunReaper(ctx context.Context, interval time.Duration, batchSize int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			i.logger.Info("reaper stopped")
			return
		case <-ticker.C:
			// Bound each sweep independently of the parent so a slow query
			// cannot stall shutdown.
			sweepCtx, cancel := context.WithTimeout(ctx, interval)
			n, err := i.repo.ReapExpiredHolds(sweepCtx, batchSize)
			cancel()

			if err != nil {
				// A failed sweep is not fatal; the next tick retries and the
				// hold predicate covers correctness meanwhile.
				i.logger.WarnContext(ctx, "reaper sweep failed", slog.Any("error", err))
				continue
			}
			if n > 0 {
				i.logger.InfoContext(ctx, "reaped expired holds", slog.Int("seats", n))
			}
		}
	}
}

func firstDuplicate(ids []string) string {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			return id
		}
		seen[id] = struct{}{}
	}
	return ""
}
