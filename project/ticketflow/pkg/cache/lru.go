// Package cache provides the L1 (in-process) tier of TicketFlow's two-tier cache.
//
// The tiers exist for different reasons. L2 (Redis) keeps the database from
// being hammered across the fleet. L1 keeps *Redis* from being hammered by a
// single hot key: during a ticket drop, 50k requests/sec for the same event
// would otherwise become 50k Redis round-trips from one pod.
//
// What belongs in here is anything expensive and slow-changing -- seat map
// geometry, event metadata, pricing tiers. Seat *availability* must never be
// cached at any tier; it changes thousands of times a second and a stale read
// means selling a seat twice.
package cache

import (
	"container/list"
	"math/rand/v2"
	"sync"
	"time"
)

// Options configures an LRU. Only MaxEntries is required.
type Options struct {
	// MaxEntries bounds the cache. Once exceeded, the least recently used
	// entry is evicted. Must be > 0.
	MaxEntries int

	// TTL is how long an entry stays fresh. Zero means entries never expire
	// on their own and are only removed by eviction or explicit Delete.
	TTL time.Duration

	// JitterFraction spreads expiry times to prevent a synchronised stampede.
	//
	// Without it, N pods that warmed their caches from the same deploy expire
	// the same key at the same instant, and every one of them stampedes the
	// upstream together. With 0.1, each entry's real TTL is drawn uniformly
	// from [0.9*TTL, 1.0*TTL], so expiries smear out instead of aligning.
	//
	// Valid range [0, 1); values outside it are clamped. Defaults to 0.1 when
	// TTL is set and this is left at zero.
	JitterFraction float64

	// Now is injectable for tests. Defaults to time.Now.
	Now func() time.Time
}

// Stats is a point-in-time snapshot of cache behaviour. Hit ratio is the number
// that matters: an L1 cache below roughly 80% during a drop is not earning the
// memory it occupies, and the TTL should go up.
type Stats struct {
	Hits      uint64
	Misses    uint64
	Evictions uint64
	// Expirations counts entries found stale on read. High expirations with
	// low hits means the TTL is shorter than the access interval.
	Expirations uint64
	Entries     int
}

// HitRatio returns hits/(hits+misses), or 0 when the cache has never been read.
func (s Stats) HitRatio() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

type entry[K comparable, V any] struct {
	key K
	val V
	// expiresAt is the jittered deadline. Zero means no expiry.
	expiresAt time.Time
}

// LRU is a fixed-capacity, TTL'd, goroutine-safe cache.
//
// It is generic over both key and value so services share one implementation
// rather than hand-rolling a map+mutex per cached type: catalog caches
// LRU[string, *Event], the seat map cache is LRU[string, *SeatMap], and neither
// gives up type safety at the call site.
type LRU[K comparable, V any] struct {
	mu       sync.Mutex
	ll       *list.List               // front = most recently used
	items    map[K]*list.Element       // key -> element holding *entry[K,V]
	maxEntry int
	ttl      time.Duration
	jitter   float64
	now      func() time.Time

	hits, misses, evictions, expirations uint64
}

// New builds an LRU. It panics if MaxEntries is not positive, because a
// zero-capacity cache is always a configuration bug and failing at startup
// beats silently caching nothing in production.
func New[K comparable, V any](opts Options) *LRU[K, V] {
	if opts.MaxEntries <= 0 {
		panic("cache: MaxEntries must be > 0")
	}

	jitter := opts.JitterFraction
	switch {
	case jitter < 0:
		jitter = 0
	case jitter >= 1:
		// A full-TTL jitter could produce a zero lifetime; cap below 1.
		jitter = 0.99
	case jitter == 0 && opts.TTL > 0:
		jitter = 0.1 // sensible default rather than perfectly-aligned expiry
	}

	now := opts.Now
	if now == nil {
		now = time.Now
	}

	return &LRU[K, V]{
		ll:       list.New(),
		items:    make(map[K]*list.Element, opts.MaxEntries),
		maxEntry: opts.MaxEntries,
		ttl:      opts.TTL,
		jitter:   jitter,
		now:      now,
	}
}

// Get returns the cached value and whether it was a usable hit. An entry past
// its deadline is removed and reported as a miss.
//
// Expiry is lazy -- checked on read rather than swept by a background
// goroutine. A sweeper would cost a goroutine per cache and still not bound
// memory any better, since capacity is already enforced by eviction.
func (c *LRU[K, V]) Get(key K) (V, bool) {
	var zero V

	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		c.misses++
		return zero, false
	}

	ent := el.Value.(*entry[K, V])
	if !ent.expiresAt.IsZero() && !c.now().Before(ent.expiresAt) {
		c.removeElement(el)
		c.expirations++
		c.misses++
		return zero, false
	}

	c.ll.MoveToFront(el)
	c.hits++
	return ent.val, true
}

// Set inserts or replaces a value, evicting the least recently used entry if
// the cache is over capacity. Replacing an existing key refreshes its deadline.
func (c *LRU[K, V]) Set(key K, val V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := c.deadline()

	if el, ok := c.items[key]; ok {
		ent := el.Value.(*entry[K, V])
		ent.val = val
		ent.expiresAt = expiresAt
		c.ll.MoveToFront(el)
		return
	}

	c.items[key] = c.ll.PushFront(&entry[K, V]{key: key, val: val, expiresAt: expiresAt})

	if c.ll.Len() > c.maxEntry {
		if oldest := c.ll.Back(); oldest != nil {
			c.removeElement(oldest)
			c.evictions++
		}
	}
}

// deadline computes a jittered expiry. Caller must hold the lock.
func (c *LRU[K, V]) deadline() time.Time {
	if c.ttl <= 0 {
		return time.Time{}
	}
	ttl := c.ttl
	if c.jitter > 0 {
		// Subtract up to jitter*TTL so entries expire early, never late --
		// serving data past its stated freshness would be the worse bug.
		reduction := time.Duration(rand.Float64() * c.jitter * float64(c.ttl))
		ttl -= reduction
	}
	return c.now().Add(ttl)
}

// Delete removes a key. It is the hook Kafka-driven invalidation calls when
// catalog-svc publishes catalog.event.updated, so every replica drops its stale
// copy instead of waiting out the TTL.
func (c *LRU[K, V]) Delete(key K) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return false
	}
	c.removeElement(el)
	return true
}

// Purge drops every entry, leaving counters intact. Used when a schema version
// rolls and the whole keyspace is suspect.
func (c *LRU[K, V]) Purge() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ll.Init()
	c.items = make(map[K]*list.Element, c.maxEntry)
}

// Len returns the current entry count, including any not-yet-reaped expired
// entries.
func (c *LRU[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// Stats snapshots counters for export to CloudWatch.
func (c *LRU[K, V]) Stats() Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	return Stats{
		Hits:        c.hits,
		Misses:      c.misses,
		Evictions:   c.evictions,
		Expirations: c.expirations,
		Entries:     c.ll.Len(),
	}
}

// removeElement unlinks an element. Caller must hold the lock.
func (c *LRU[K, V]) removeElement(el *list.Element) {
	c.ll.Remove(el)
	delete(c.items, el.Value.(*entry[K, V]).key)
}
