// Package domain holds inventory's internal types.
//
// Everything here is correctness-critical. Unlike catalog, none of it may be
// cached at any tier: a stale availability read means selling one seat twice.
package domain

import (
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when a hold or event does not exist.
	ErrNotFound = errors.New("inventory: not found")

	// ErrHoldExpired is returned when confirming a hold whose TTL already
	// lapsed. The caller must re-acquire; the seats may since have been taken.
	ErrHoldExpired = errors.New("inventory: hold expired")

	// ErrNoSeatsAvailable is returned when none of the requested seats could be
	// held, i.e. every one was lost to another buyer.
	ErrNoSeatsAvailable = errors.New("inventory: no seats available")

	// ErrInvalidArgument marks a caller error.
	ErrInvalidArgument = errors.New("inventory: invalid argument")
)

// SeatStatus mirrors ticketflow.inventory.v1.SeatStatus and the smallint stored
// in seat_allocation.status.
type SeatStatus int16

const (
	SeatStatusUnspecified SeatStatus = 0
	SeatStatusAvailable   SeatStatus = 1
	SeatStatusHeld        SeatStatus = 2
	SeatStatusSold        SeatStatus = 3
	SeatStatusBlocked     SeatStatus = 4
)

// SeatAvailability is the current state of one seat. Always read through to
// Postgres -- never served from cache.
type SeatAvailability struct {
	SeatID string
	Status SeatStatus
	// HoldExpiresAt is set only when Status is Held, letting a client render a
	// countdown without polling again.
	HoldExpiresAt time.Time
}

// Hold is a time-limited claim on a set of seats.
type Hold struct {
	ID        string
	EventID   string
	UserID    string
	SeatIDs   []string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Expired reports whether the hold has lapsed as of now.
func (h Hold) Expired(now time.Time) bool {
	return !now.Before(h.ExpiresAt)
}

// HoldRequest asks for a set of seats.
type HoldRequest struct {
	EventID string
	SeatIDs []string
	UserID  string
	// IdempotencyKey makes a retried request return the original hold rather
	// than acquiring a second set of seats. Enforced by a unique constraint in
	// Postgres, not by application logic -- the database is the only place a
	// check-then-act race cannot slip through.
	IdempotencyKey string
	TTL            time.Duration
}

// HoldResult reports what was won and what was lost.
type HoldResult struct {
	Hold Hold
	// RejectedSeatIDs were requested but lost to another buyer. A partial
	// result is normal during a drop and is surfaced rather than hidden, so the
	// UI can tell the user which seats it actually got.
	RejectedSeatIDs []string
	// Replayed is true when an idempotency key matched an existing hold, so no
	// new seats were acquired.
	Replayed bool
}
