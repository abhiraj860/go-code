// Package repo holds inventory's data access.
//
// This file contains the most important query in the system. Read the comment
// on HoldSeats before changing anything here.
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/abhiraj860/ticketflow/services/inventory-svc/internal/domain"
)

// pgUniqueViolation is the SQLSTATE for a unique constraint breach.
const pgUniqueViolation = "23505"

type SeatRepo struct {
	pool *pgxpool.Pool
}

func NewSeatRepo(pool *pgxpool.Pool) *SeatRepo {
	return &SeatRepo{pool: pool}
}

// GetAvailability returns current seat state. Never cached: during a drop this
// changes thousands of times a second, and a stale answer means showing a buyer
// a seat that is already gone.
//
// An empty seatIDs slice means "every seat for the event".
func (r *SeatRepo) GetAvailability(ctx context.Context, eventID string, seatIDs []string) ([]domain.SeatAvailability, error) {
	// Both branches are served by the (event_id, seat_id) primary key: the
	// first as a range scan, the second as a series of point lookups.
	q := `
		SELECT seat_id, status, hold_expires_at
		FROM seat_allocation
		WHERE event_id = $1`
	args := []any{eventID}

	if len(seatIDs) > 0 {
		q += ` AND seat_id = ANY($2)`
		args = append(args, seatIDs)
	}
	q += ` ORDER BY seat_id`

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("inventory: querying availability for %q: %w", eventID, err)
	}
	defer rows.Close()

	var out []domain.SeatAvailability
	now := time.Now()
	for rows.Next() {
		var (
			sa        domain.SeatAvailability
			expiresAt *time.Time
		)
		if err := rows.Scan(&sa.SeatID, &sa.Status, &expiresAt); err != nil {
			return nil, fmt.Errorf("inventory: scanning availability: %w", err)
		}
		if expiresAt != nil {
			sa.HoldExpiresAt = *expiresAt
		}
		// Present a lapsed hold as available. The row is only rewritten when
		// someone actually takes the seat or the reaper runs, so reporting the
		// stored status verbatim would show seats as held that anyone can claim.
		if sa.Status == domain.SeatStatusHeld && expiresAt != nil && !now.Before(*expiresAt) {
			sa.Status = domain.SeatStatusAvailable
			sa.HoldExpiresAt = time.Time{}
		}
		out = append(out, sa)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inventory: iterating availability: %w", err)
	}
	return out, nil
}

// HoldSeats attempts to claim seats, returning those actually won.
//
// THE CORE GUARANTEE. Two buyers racing for one seat must produce exactly one
// winner, and this is where that is enforced.
//
// The claim is a single conditional UPDATE:
//
//	UPDATE seat_allocation SET status = HELD, hold_id = ...
//	 WHERE event_id = $1 AND seat_id = ANY($2)
//	   AND (status = AVAILABLE OR (status = HELD AND hold_expires_at < now()))
//	RETURNING seat_id
//
// Postgres takes a row lock while evaluating the WHERE clause. When two
// transactions target the same row, the second blocks until the first commits,
// then re-evaluates its predicate against the *committed* row -- where status
// is now HELD with a future expiry, so the condition fails and the row is not
// returned. The winner is whoever RETURNING names.
//
// Why not SELECT ... FOR UPDATE then UPDATE:
//   - two round-trips instead of one, doubling the window under contention
//   - requires sorting seat IDs to avoid deadlocking two multi-seat requests
//     that overlap in opposite orders
//   - the read and the write can drift apart under future refactoring
//
// The expired-hold clause lets a lapsed hold be stolen in the same statement,
// which makes the background reaper an optimisation for freeing seats promptly
// rather than a correctness requirement.
func (r *SeatRepo) HoldSeats(ctx context.Context, req domain.HoldRequest) (domain.HoldResult, error) {
	// Idempotency is checked before opening a transaction so the common replay
	// path costs one indexed lookup rather than a transaction plus a rollback.
	if existing, err := r.findHoldByIdempotencyKey(ctx, req.UserID, req.IdempotencyKey); err == nil {
		return domain.HoldResult{Hold: existing, Replayed: true}, nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return domain.HoldResult{}, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.HoldResult{}, fmt.Errorf("inventory: beginning transaction: %w", err)
	}
	// Rollback on any path that does not commit. A no-op after a successful
	// Commit, so it is safe to defer unconditionally.
	defer func() { _ = tx.Rollback(ctx) }()

	holdID := uuid.NewString()
	expiresAt := time.Now().Add(req.TTL)

	const insertHold = `
		INSERT INTO seat_hold (id, event_id, user_id, idempotency_key, expires_at)
		VALUES ($1, $2, $3, $4, $5)`

	_, err = tx.Exec(ctx, insertHold, holdID, req.EventID, req.UserID, req.IdempotencyKey, expiresAt)
	if err != nil {
		// Two concurrent requests carrying the same idempotency key: one wins
		// the insert, the other lands here. Return the winner's hold rather
		// than acquiring a second set of seats.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			_ = tx.Rollback(ctx)
			existing, findErr := r.findHoldByIdempotencyKey(ctx, req.UserID, req.IdempotencyKey)
			if findErr != nil {
				return domain.HoldResult{}, findErr
			}
			return domain.HoldResult{Hold: existing, Replayed: true}, nil
		}
		return domain.HoldResult{}, fmt.Errorf("inventory: inserting hold: %w", err)
	}

	const claim = `
		UPDATE seat_allocation
		   SET status = 2, hold_id = $3, hold_expires_at = $4, updated_at = now()
		 WHERE event_id = $1
		   AND seat_id = ANY($2)
		   AND (status = 1 OR (status = 2 AND hold_expires_at < now()))
		RETURNING seat_id`

	rows, err := tx.Query(ctx, claim, req.EventID, req.SeatIDs, holdID, expiresAt)
	if err != nil {
		return domain.HoldResult{}, fmt.Errorf("inventory: claiming seats: %w", err)
	}

	won := make([]string, 0, len(req.SeatIDs))
	for rows.Next() {
		var seatID string
		if err := rows.Scan(&seatID); err != nil {
			rows.Close()
			return domain.HoldResult{}, fmt.Errorf("inventory: scanning claimed seat: %w", err)
		}
		won = append(won, seatID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return domain.HoldResult{}, fmt.Errorf("inventory: iterating claimed seats: %w", err)
	}

	// Winning nothing means every seat was taken. Roll back so the hold row
	// does not linger and burn the idempotency key -- the caller should be able
	// to retry the same key against different seats.
	if len(won) == 0 {
		_ = tx.Rollback(ctx)
		return domain.HoldResult{
			RejectedSeatIDs: req.SeatIDs,
		}, domain.ErrNoSeatsAvailable
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.HoldResult{}, fmt.Errorf("inventory: committing hold: %w", err)
	}

	return domain.HoldResult{
		Hold: domain.Hold{
			ID:        holdID,
			EventID:   req.EventID,
			UserID:    req.UserID,
			SeatIDs:   won,
			ExpiresAt: expiresAt,
			CreatedAt: time.Now(),
		},
		RejectedSeatIDs: difference(req.SeatIDs, won),
	}, nil
}

// ReleaseHold returns held seats to the pool. Idempotent: releasing an already
// released or expired hold is not an error, because a client retrying a
// cancellation should not see a failure.
func (r *SeatRepo) ReleaseHold(ctx context.Context, holdID string) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("inventory: beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Only HELD rows are reverted. A seat already SOLD under this hold must
	// stay sold -- releasing it would un-sell a paid ticket.
	const release = `
		UPDATE seat_allocation
		   SET status = 1, hold_id = NULL, hold_expires_at = NULL, updated_at = now()
		 WHERE hold_id = $1 AND status = 2`

	tag, err := tx.Exec(ctx, release, holdID)
	if err != nil {
		return 0, fmt.Errorf("inventory: releasing seats: %w", err)
	}

	const markReleased = `
		UPDATE seat_hold SET released_at = now()
		 WHERE id = $1 AND released_at IS NULL`
	if _, err := tx.Exec(ctx, markReleased, holdID); err != nil {
		return 0, fmt.Errorf("inventory: marking hold released: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("inventory: committing release: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// ConfirmHold converts HELD seats to SOLD. Called by order-svc once payment has
// settled.
//
// The status = 2 AND hold_expires_at > now() predicate is what stops a lapsed
// hold being confirmed after someone else has taken the seats.
func (r *SeatRepo) ConfirmHold(ctx context.Context, holdID, orderID string) ([]string, error) {
	const confirm = `
		UPDATE seat_allocation
		   SET status = 3, order_id = $2, hold_expires_at = NULL, updated_at = now()
		 WHERE hold_id = $1 AND status = 2 AND hold_expires_at > now()
		RETURNING seat_id`

	rows, err := r.pool.Query(ctx, confirm, holdID, orderID)
	if err != nil {
		return nil, fmt.Errorf("inventory: confirming hold: %w", err)
	}
	defer rows.Close()

	var sold []string
	for rows.Next() {
		var seatID string
		if err := rows.Scan(&seatID); err != nil {
			return nil, fmt.Errorf("inventory: scanning sold seat: %w", err)
		}
		sold = append(sold, seatID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inventory: iterating sold seats: %w", err)
	}

	if len(sold) == 0 {
		// Distinguish "never existed" from "too late", because the caller's
		// recovery differs: one is a bug, the other is a retry with new seats.
		exists, err := r.holdExists(ctx, holdID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrHoldExpired
	}
	return sold, nil
}

// ReapExpiredHolds returns lapsed holds' seats to the pool.
//
// Purely an optimisation: the HoldSeats predicate already treats an expired
// hold as claimable, so correctness does not depend on this running. It exists
// so GetAvailability reflects reality promptly and the seat map does not show
// phantom holds.
func (r *SeatRepo) ReapExpiredHolds(ctx context.Context, limit int) (int, error) {
	// Uses seat_allocation_expiry_idx, a partial index over held rows only.
	const reap = `
		UPDATE seat_allocation
		   SET status = 1, hold_id = NULL, hold_expires_at = NULL, updated_at = now()
		 WHERE seat_id IN (
		     SELECT seat_id FROM seat_allocation
		      WHERE status = 2 AND hold_expires_at < now()
		      LIMIT $1
		 ) AND status = 2 AND hold_expires_at < now()`

	tag, err := r.pool.Exec(ctx, reap, limit)
	if err != nil {
		return 0, fmt.Errorf("inventory: reaping expired holds: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// SeedSeats registers seats for an event as AVAILABLE. Called when an event
// goes on sale; idempotent so a retried rollout cannot wipe existing state.
func (r *SeatRepo) SeedSeats(ctx context.Context, eventID string, seatIDs []string) (int, error) {
	const seed = `
		INSERT INTO seat_allocation (event_id, seat_id, status)
		SELECT $1, unnest($2::text[]), 1
		ON CONFLICT (event_id, seat_id) DO NOTHING`

	tag, err := r.pool.Exec(ctx, seed, eventID, seatIDs)
	if err != nil {
		return 0, fmt.Errorf("inventory: seeding seats for %q: %w", eventID, err)
	}
	return int(tag.RowsAffected()), nil
}

func (r *SeatRepo) findHoldByIdempotencyKey(ctx context.Context, userID, key string) (domain.Hold, error) {
	if key == "" {
		return domain.Hold{}, domain.ErrNotFound
	}

	const q = `
		SELECT h.id, h.event_id, h.user_id, h.expires_at, h.created_at,
		       COALESCE(array_agg(a.seat_id ORDER BY a.seat_id)
		                FILTER (WHERE a.seat_id IS NOT NULL), '{}')
		  FROM seat_hold h
		  LEFT JOIN seat_allocation a ON a.hold_id = h.id
		 WHERE h.user_id = $1 AND h.idempotency_key = $2 AND h.released_at IS NULL
		 GROUP BY h.id`

	var h domain.Hold
	err := r.pool.QueryRow(ctx, q, userID, key).Scan(
		&h.ID, &h.EventID, &h.UserID, &h.ExpiresAt, &h.CreatedAt, &h.SeatIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Hold{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Hold{}, fmt.Errorf("inventory: looking up idempotency key: %w", err)
	}
	return h, nil
}

func (r *SeatRepo) holdExists(ctx context.Context, holdID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM seat_hold WHERE id = $1)`, holdID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("inventory: checking hold existence: %w", err)
	}
	return exists, nil
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
