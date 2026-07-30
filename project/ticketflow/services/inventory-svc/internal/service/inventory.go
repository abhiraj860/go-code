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

// Inventory serves seat availability and holds.
type Inventory struct {
	repo   Repo
	logger *slog.Logger

	defaultTTL time.Duration
	maxTTL     time.Duration
}

type Options struct {
	Repo       Repo
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

	result, err := i.repo.HoldSeats(ctx, req)
	if err != nil {
		return result, err
	}

	if result.Replayed {
		i.logger.InfoContext(ctx, "hold replayed from idempotency key",
			slog.String("hold_id", result.Hold.ID), slog.String("user_id", req.UserID))
	}
	return result, nil
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
