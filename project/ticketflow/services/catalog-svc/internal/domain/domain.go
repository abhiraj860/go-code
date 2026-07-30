// Package domain holds catalog's internal types.
//
// These are deliberately separate from the generated protobuf types. The proto
// is a wire contract that outside callers depend on and `buf breaking` freezes;
// the domain model is ours to refactor freely. Collapsing the two would mean
// every internal rename became a breaking API change.
package domain

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a record does not exist. The service layer maps
// it to cache.ErrNotFound for negative caching and to gRPC NotFound at the edge.
var ErrNotFound = errors.New("catalog: not found")

// EventKind mirrors ticketflow.catalog.v1.EventKind. Stored as a smallint.
type EventKind int16

const (
	EventKindUnspecified EventKind = 0
	EventKindConcert     EventKind = 1
	EventKindSports      EventKind = 2
	EventKindTheatre     EventKind = 3
	EventKindConference  EventKind = 4
)

// EventStatus mirrors ticketflow.catalog.v1.EventStatus.
type EventStatus int16

const (
	EventStatusUnspecified EventStatus = 0
	EventStatusAnnounced   EventStatus = 1
	EventStatusOnSale      EventStatus = 2
	EventStatusSoldOut     EventStatus = 3
	EventStatusCancelled   EventStatus = 4
)

// Money is an exact amount. Minor units only -- floats are never used for money
// because 0.1 + 0.2 != 0.3 in binary floating point, and ticket prices get
// summed.
type Money struct {
	AmountMinor  int64
	CurrencyCode string
}

type Venue struct {
	ID          string
	Name        string
	City        string
	CountryCode string
	Address     string
	Latitude    float64
	Longitude   float64
}

type PricingTier struct {
	ID    string
	Name  string
	Price Money
}

// Event is the cacheable half of an event: descriptive, slow-changing data.
// Seat availability deliberately lives in inventory-svc and is never cached.
type Event struct {
	ID           string
	Title        string
	Kind         EventKind
	Status       EventStatus
	Venue        Venue
	SeatMapID    string
	StartsAt     time.Time
	EndsAt       time.Time
	SaleOpensAt  time.Time
	PosterURL    string
	Tags         []string
	PricingTiers []PricingTier

	// Version increments on every mutation. It is both the L2 cache-key suffix
	// and the optimistic-locking token for admin edits.
	Version   int64
	UpdatedAt time.Time
}

// Seat is fixed geometry -- where a seat physically is. Its availability is
// inventory-svc's concern. That separation is what makes a seat map safe to
// cache for hours while availability changes thousands of times a second.
type Seat struct {
	ID            string
	Section       string
	Row           string
	Number        string
	PricingTierID string
	X             float32
	Y             float32
}

type SeatMap struct {
	ID            string
	VenueID       string
	Seats         []Seat
	ViewboxWidth  float32
	ViewboxHeight float32
	Version       int64
}

// ListFilter narrows a browse query. A zero value lists upcoming on-sale
// events across every city.
type ListFilter struct {
	City string
	Kind EventKind
	// PageSize is clamped by the service; see service.maxPageSize.
	PageSize int32
	// PageToken is an opaque cursor produced by a previous call.
	PageToken string
}

// Page carries a slice of results plus the cursor for the next call.
//
// Generic so catalog, search and order all use one pagination shape rather
// than each defining their own EventPage / OrderPage struct.
type Page[T any] struct {
	Items         []T
	NextPageToken string
}
