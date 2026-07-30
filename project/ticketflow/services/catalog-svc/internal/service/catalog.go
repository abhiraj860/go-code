// Package service holds catalog's business logic and its caching policy.
//
// Caching policy in one place, deliberately: seat maps and events are cached
// aggressively because they are large and change rarely, while nothing that
// touches seat *availability* is cached at all -- that lives in inventory-svc
// and a stale read there means selling one seat twice.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abhiraj860/ticketflow/pkg/cache"
	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/domain"
)

// schemaVersion is baked into every L2 cache key. Bumping it when the cached
// struct layout changes rolls the entire keyspace atomically at deploy time,
// instead of requiring a Redis FLUSH or leaving old-shaped entries to be
// decoded by new code.
const schemaVersion = 1

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// Repo is the data dependency, kept as an interface so the service's caching
// behaviour can be tested against a stub without a database.
type Repo interface {
	GetEvent(ctx context.Context, id string) (domain.Event, error)
	ListEvents(ctx context.Context, f domain.ListFilter) (domain.Page[domain.Event], error)
	GetSeatMap(ctx context.Context, id string) (domain.SeatMap, error)
}

// ContentRepo fetches variable-shape event content from MongoDB. Optional --
// when nil, events are served without their content document.
type ContentRepo interface {
	GetContent(ctx context.Context, eventID string) (map[string]any, error)
}

// Catalog serves catalog reads through a two-tier cache.
type Catalog struct {
	repo    Repo
	content ContentRepo

	events   *cache.Loader[string, domain.Event]
	seatMaps *cache.Loader[string, domain.SeatMap]
}

// Options configures the service.
type Options struct {
	Repo    Repo
	Content ContentRepo
	// L2 is the shared Redis tier. Optional; nil yields an L1-only service,
	// which is what unit tests use.
	L2 cache.Store

	// EventTTL is short by design. Event metadata is invalidated over Kafka
	// when it actually changes, so the TTL is only a safety net for a missed
	// invalidation rather than the primary freshness mechanism.
	EventTTL time.Duration
	// SeatMapTTL is long: seat geometry effectively never changes, and a map
	// is the most expensive object here to rebuild.
	SeatMapTTL time.Duration
	// L2TTL bounds how long Redis holds an entry. Should exceed the L1 TTLs.
	L2TTL time.Duration
	// MaxCachedSeatMaps bounds L1 memory. Seat maps are large (tens of
	// thousands of seats), so this is small on purpose.
	MaxCachedSeatMaps int
	// MaxCachedEvents bounds the event L1 tier.
	MaxCachedEvents int
}

func New(opts Options) *Catalog {
	if opts.EventTTL == 0 {
		opts.EventTTL = 30 * time.Second
	}
	if opts.SeatMapTTL == 0 {
		opts.SeatMapTTL = 10 * time.Minute
	}
	if opts.L2TTL == 0 {
		opts.L2TTL = time.Hour
	}
	if opts.MaxCachedEvents == 0 {
		opts.MaxCachedEvents = 2048
	}
	if opts.MaxCachedSeatMaps == 0 {
		opts.MaxCachedSeatMaps = 64
	}

	c := &Catalog{repo: opts.Repo, content: opts.Content}

	c.events = cache.NewLoader(cache.LoaderOptions[string, domain.Event]{
		L1: cache.New[string, domain.Event](cache.Options{
			MaxEntries: opts.MaxCachedEvents,
			TTL:        opts.EventTTL,
		}),
		L2:    opts.L2,
		Codec: jsonCodec[domain.Event]{},
		KeyFn: func(id string) string {
			return fmt.Sprintf("catalog:event:%s:s%d", id, schemaVersion)
		},
		L2TTL: opts.L2TTL,
		// Short: this exists to blunt ID-scanning traffic, not to remember
		// real state. An event created seconds after a miss must appear quickly.
		NegativeTTL: 10 * time.Second,
		Fetch: func(ctx context.Context, id string) (domain.Event, error) {
			e, err := opts.Repo.GetEvent(ctx, id)
			if errors.Is(err, domain.ErrNotFound) {
				// Translate so the loader can apply negative caching.
				return domain.Event{}, cache.ErrNotFound
			}
			return e, err
		},
	})

	c.seatMaps = cache.NewLoader(cache.LoaderOptions[string, domain.SeatMap]{
		L1: cache.New[string, domain.SeatMap](cache.Options{
			MaxEntries: opts.MaxCachedSeatMaps,
			TTL:        opts.SeatMapTTL,
		}),
		L2:    opts.L2,
		Codec: jsonCodec[domain.SeatMap]{},
		KeyFn: func(id string) string {
			return fmt.Sprintf("catalog:seatmap:%s:s%d", id, schemaVersion)
		},
		L2TTL:       opts.L2TTL,
		NegativeTTL: 10 * time.Second,
		Fetch: func(ctx context.Context, id string) (domain.SeatMap, error) {
			m, err := opts.Repo.GetSeatMap(ctx, id)
			if errors.Is(err, domain.ErrNotFound) {
				return domain.SeatMap{}, cache.ErrNotFound
			}
			return m, err
		},
	})

	return c
}

// GetEvent returns an event, served from cache when possible.
func (c *Catalog) GetEvent(ctx context.Context, id string) (domain.Event, error) {
	if id == "" {
		return domain.Event{}, fmt.Errorf("%w: event id is required", ErrInvalidArgument)
	}

	e, err := c.events.Get(ctx, id)
	if errors.Is(err, cache.ErrNotFound) {
		return domain.Event{}, domain.ErrNotFound
	}
	return e, err
}

// GetSeatMap returns a seat map, served from cache when possible.
func (c *Catalog) GetSeatMap(ctx context.Context, id string) (domain.SeatMap, error) {
	if id == "" {
		return domain.SeatMap{}, fmt.Errorf("%w: seat map id is required", ErrInvalidArgument)
	}

	m, err := c.seatMaps.Get(ctx, id)
	if errors.Is(err, cache.ErrNotFound) {
		return domain.SeatMap{}, domain.ErrNotFound
	}
	return m, err
}

// ListEvents returns a page of browsable events.
//
// Deliberately uncached: the result depends on filters, cursor and page size,
// so the key space is effectively unbounded and the hit ratio would be poor.
// The underlying query is already an index-only range scan. Phase 2 moves this
// to search-svc, where ElasticSearch caches facets properly.
func (c *Catalog) ListEvents(ctx context.Context, f domain.ListFilter) (domain.Page[domain.Event], error) {
	switch {
	case f.PageSize < 0:
		return domain.Page[domain.Event]{}, fmt.Errorf("%w: page size cannot be negative", ErrInvalidArgument)
	case f.PageSize == 0:
		f.PageSize = defaultPageSize
	case f.PageSize > maxPageSize:
		// Clamp rather than reject: a client asking for too much should get a
		// sane page, not an error it has to handle.
		f.PageSize = maxPageSize
	}
	return c.repo.ListEvents(ctx, f)
}

// InvalidateEvent drops an event from both cache tiers. Called by the admin
// write path; Phase 2 also calls it from the catalog.event.updated consumer so
// every replica converges.
func (c *Catalog) InvalidateEvent(ctx context.Context, id string) error {
	return c.events.Invalidate(ctx, id)
}

// InvalidateEventLocal drops only this replica's copy. This is what the Kafka
// consumer calls: the publisher already removed the shared L2 entry, so having
// every replica delete it again would be N redundant Redis round-trips.
func (c *Catalog) InvalidateEventLocal(id string) {
	c.events.InvalidateLocal(id)
}

// CacheStats exposes both loaders' counters for the /metrics endpoint. Hit
// ratio is the number that matters; Phase 7 tunes TTLs against it.
func (c *Catalog) CacheStats() (events, seatMaps cache.LoaderStats) {
	return c.events.Stats(), c.seatMaps.Stats()
}

// ErrInvalidArgument marks a caller error, mapped to gRPC InvalidArgument.
var ErrInvalidArgument = errors.New("catalog: invalid argument")

// jsonCodec serialises cached values for L2.
//
// JSON rather than protobuf or gob: cache entries are inspectable with
// redis-cli during an incident, which is worth more than the bytes saved.
// If Phase 7 shows serialisation is hot, this is a one-line swap.
type jsonCodec[V any] struct{}

func (jsonCodec[V]) Marshal(v V) ([]byte, error) { return json.Marshal(v) }
func (jsonCodec[V]) Unmarshal(b []byte) (V, error) {
	var v V
	err := json.Unmarshal(b, &v)
	return v, err
}
