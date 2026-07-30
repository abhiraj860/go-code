package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// ErrNotFound is what a Fetch function returns when the underlying record does
// not exist. The Loader caches this outcome (negative caching) so a bot walking
// random event IDs cannot turn every miss into a database query.
var ErrNotFound = errors.New("cache: record not found")

// Store is the L2 tier. Redis satisfies this in production; tests use an
// in-memory fake. Keeping it an interface means the stampede and negative-
// caching logic can be tested without a Redis container.
type Store interface {
	// Get returns the raw value. found is false when the key is absent.
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	// Set stores value with the given TTL.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	// Del removes keys. Invalidation calls this.
	Del(ctx context.Context, keys ...string) error
}

// Codec converts between a domain value and the bytes held in L2.
type Codec[V any] interface {
	Marshal(V) ([]byte, error)
	Unmarshal([]byte) (V, error)
}

// Fetch loads a value from the system of record on a full cache miss.
type Fetch[K comparable, V any] func(ctx context.Context, key K) (V, error)

// LoaderOptions configures a two-tier loader.
type LoaderOptions[K comparable, V any] struct {
	// L1 is the in-process tier. Required.
	L1 *LRU[K, V]
	// L2 is the shared tier (Redis). Optional -- when nil the loader is L1-only,
	// which is what unit tests and single-instance deployments use.
	L2 Store
	// Codec serialises for L2. Required when L2 is set.
	Codec Codec[V]
	// KeyFn maps a domain key to its L2 string key. Include a version suffix
	// (event:{id}:v{n}) so a schema change rolls the keyspace without a flush.
	KeyFn func(K) string
	// L2TTL is how long values live in Redis. Should exceed the L1 TTL --
	// L1 is a latency optimisation, L2 is the database shield.
	L2TTL time.Duration
	// NegativeTTL is how long a "not found" result is remembered. Keep it short
	// (seconds): it exists to blunt scanning traffic, not to cache real state.
	// Zero disables negative caching.
	NegativeTTL time.Duration
	// Fetch loads from the system of record. Required.
	Fetch Fetch[K, V]
}

// LoaderStats reports where reads were served from. The ratio between these is
// the number to watch: heavy l2_hits with few l1_hits means the L1 TTL is too
// short; heavy fetches means the whole cache is not earning its memory.
type LoaderStats struct {
	L1Hits       uint64
	L2Hits       uint64
	Fetches      uint64
	NegativeHits uint64
	// Coalesced counts requests that waited on another goroutine's in-flight
	// fetch instead of issuing their own. During a drop this should be large;
	// it is the stampede guard doing its job.
	Coalesced uint64
	Errors    uint64
}

// Loader is a read-through cache over two tiers with single-flight protection.
//
// Read path: L1 -> L2 -> Fetch. A miss at every tier issues exactly one Fetch
// per key no matter how many goroutines ask concurrently; the rest block on
// that call and share its result.
//
// Why the stampede guard matters here specifically: when a hot event's key
// expires mid-drop, tens of thousands of in-flight requests miss simultaneously.
// Without coalescing, every one of them becomes a database query and the
// database -- not the cache -- becomes the outage.
type Loader[K comparable, V any] struct {
	opts LoaderOptions[K, V]

	// inflight tracks one call per key. This is a hand-rolled singleflight
	// rather than golang.org/x/sync/singleflight so the loader stays
	// dependency-free and can report Coalesced counts, which the upstream
	// package does not expose.
	mu       sync.Mutex
	inflight map[K]*call[V]

	l1Hits, l2Hits, fetches, negHits, coalesced, errs atomic.Uint64
}

type call[V any] struct {
	wg  sync.WaitGroup
	val V
	err error
}

// NewLoader builds a two-tier loader. It panics on a missing required field,
// since every one of them is a wiring bug that should fail at startup.
func NewLoader[K comparable, V any](opts LoaderOptions[K, V]) *Loader[K, V] {
	switch {
	case opts.L1 == nil:
		panic("cache: LoaderOptions.L1 is required")
	case opts.Fetch == nil:
		panic("cache: LoaderOptions.Fetch is required")
	case opts.L2 != nil && opts.Codec == nil:
		panic("cache: LoaderOptions.Codec is required when L2 is set")
	case opts.L2 != nil && opts.KeyFn == nil:
		panic("cache: LoaderOptions.KeyFn is required when L2 is set")
	}

	return &Loader[K, V]{
		opts:     opts,
		inflight: make(map[K]*call[V]),
	}
}

// Get returns the value for key, consulting L1, then L2, then the system of
// record. It returns ErrNotFound when the record does not exist.
func (l *Loader[K, V]) Get(ctx context.Context, key K) (V, error) {
	// L1: no I/O, no lock contention beyond the cache's own mutex.
	if v, ok := l.opts.L1.Get(key); ok {
		l.l1Hits.Add(1)
		return v, nil
	}
	return l.loadShared(ctx, key)
}

// loadShared performs the L2 lookup and fetch under single-flight, so N
// concurrent missers cause one downstream call rather than N.
func (l *Loader[K, V]) loadShared(ctx context.Context, key K) (V, error) {
	l.mu.Lock()
	if c, ok := l.inflight[key]; ok {
		// Someone else is already loading this key. Wait for their result
		// instead of duplicating the work.
		l.mu.Unlock()
		l.coalesced.Add(1)
		c.wg.Wait()
		return c.val, c.err
	}

	c := new(call[V])
	c.wg.Add(1)
	l.inflight[key] = c
	l.mu.Unlock()

	c.val, c.err = l.load(ctx, key)

	l.mu.Lock()
	delete(l.inflight, key)
	l.mu.Unlock()
	c.wg.Done()

	return c.val, c.err
}

// load runs the actual L2-then-origin lookup for a single caller.
func (l *Loader[K, V]) load(ctx context.Context, key K) (V, error) {
	var zero V

	// Re-check L1: while we waited for the inflight lock, another goroutine's
	// completed load may have populated it.
	if v, ok := l.opts.L1.Get(key); ok {
		l.l1Hits.Add(1)
		return v, nil
	}

	if l.opts.L2 != nil {
		v, ok, err := l.fromL2(ctx, key)
		switch {
		case errors.Is(err, ErrNotFound):
			// A negative-cache hit is an answer, not a failure. Returning here
			// is the entire point of negative caching -- falling through to
			// origin would leave the database exposed to the scanning traffic
			// the marker exists to absorb.
			return zero, ErrNotFound
		case err != nil:
			// A real L2 failure must not fail the request. Redis being down
			// should degrade latency, never availability: fall through to origin.
			l.errs.Add(1)
		case ok:
			l.l2Hits.Add(1)
			l.opts.L1.Set(key, v)
			return v, nil
		}
	}

	l.fetches.Add(1)
	v, err := l.opts.Fetch(ctx, key)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			l.cacheNegative(ctx, key)
		} else {
			l.errs.Add(1)
		}
		return zero, err
	}

	l.opts.L1.Set(key, v)
	l.toL2(ctx, key, v)
	return v, nil
}

// negativeMarker is the sentinel byte written to L2 for a known-absent record.
// A single byte that cannot be valid JSON keeps it unambiguous.
var negativeMarker = []byte{0x00}

func (l *Loader[K, V]) fromL2(ctx context.Context, key K) (V, bool, error) {
	var zero V

	raw, found, err := l.opts.L2.Get(ctx, l.opts.KeyFn(key))
	if err != nil || !found {
		return zero, false, err
	}

	if len(raw) == 1 && raw[0] == negativeMarker[0] {
		l.negHits.Add(1)
		return zero, false, ErrNotFound
	}

	v, err := l.opts.Codec.Unmarshal(raw)
	if err != nil {
		// Corrupt or stale-format entry: drop it and fall through to origin
		// rather than serving garbage.
		_ = l.opts.L2.Del(ctx, l.opts.KeyFn(key))
		return zero, false, err
	}
	return v, true, nil
}

func (l *Loader[K, V]) toL2(ctx context.Context, key K, v V) {
	if l.opts.L2 == nil {
		return
	}
	raw, err := l.opts.Codec.Marshal(v)
	if err != nil {
		l.errs.Add(1)
		return
	}
	if err := l.opts.L2.Set(ctx, l.opts.KeyFn(key), raw, l.opts.L2TTL); err != nil {
		l.errs.Add(1)
	}
}

func (l *Loader[K, V]) cacheNegative(ctx context.Context, key K) {
	if l.opts.L2 == nil || l.opts.NegativeTTL <= 0 {
		return
	}
	if err := l.opts.L2.Set(ctx, l.opts.KeyFn(key), negativeMarker, l.opts.NegativeTTL); err != nil {
		l.errs.Add(1)
	}
}

// Invalidate drops a key from both tiers. This is what the Kafka consumer for
// catalog.event.updated calls: the publisher deletes the shared L2 entry, and
// every replica independently drops its own L1 copy. Without that fan-out,
// in-process caches on N replicas would each serve stale data until their TTL
// elapsed.
func (l *Loader[K, V]) Invalidate(ctx context.Context, key K) error {
	l.opts.L1.Delete(key)
	if l.opts.L2 == nil {
		return nil
	}
	return l.opts.L2.Del(ctx, l.opts.KeyFn(key))
}

// InvalidateLocal drops only the in-process copy. Used by the Kafka consumer on
// each replica: the publisher already removed the shared L2 entry, so replicas
// deleting it again would be N redundant round-trips to Redis.
func (l *Loader[K, V]) InvalidateLocal(key K) {
	l.opts.L1.Delete(key)
}

// Stats snapshots counters for export to CloudWatch.
func (l *Loader[K, V]) Stats() LoaderStats {
	return LoaderStats{
		L1Hits:       l.l1Hits.Load(),
		L2Hits:       l.l2Hits.Load(),
		Fetches:      l.fetches.Load(),
		NegativeHits: l.negHits.Load(),
		Coalesced:    l.coalesced.Load(),
		Errors:       l.errs.Load(),
	}
}
