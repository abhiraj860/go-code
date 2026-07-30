package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// fakeClock lets TTL behaviour be tested deterministically instead of with sleeps.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (f *fakeClock) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.t
}

func (f *fakeClock) Advance(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.t = f.t.Add(d)
}

func TestGetSet(t *testing.T) {
	c := New[string, int](Options{MaxEntries: 4})

	if _, ok := c.Get("absent"); ok {
		t.Error("Get on an empty cache reported a hit")
	}

	c.Set("a", 1)
	got, ok := c.Get("a")
	if !ok {
		t.Fatal("Get after Set reported a miss")
	}
	if got != 1 {
		t.Errorf("Get = %d, want 1", got)
	}
}

func TestSetReplacesExistingKey(t *testing.T) {
	c := New[string, int](Options{MaxEntries: 4})

	c.Set("a", 1)
	c.Set("a", 2)

	if got, _ := c.Get("a"); got != 2 {
		t.Errorf("Get after replace = %d, want 2", got)
	}
	if c.Len() != 1 {
		t.Errorf("Len = %d, want 1 -- replace must not add a second entry", c.Len())
	}
}

func TestEvictsLeastRecentlyUsed(t *testing.T) {
	c := New[string, int](Options{MaxEntries: 3})

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	// Touch "a" so "b" becomes the least recently used.
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected a hit on 'a'")
	}

	c.Set("d", 4) // over capacity -> evicts "b"

	if _, ok := c.Get("b"); ok {
		t.Error("'b' survived; expected it to be evicted as least recently used")
	}
	for _, k := range []string{"a", "c", "d"} {
		if _, ok := c.Get(k); !ok {
			t.Errorf("%q was evicted; expected it to survive", k)
		}
	}
	if got := c.Stats().Evictions; got != 1 {
		t.Errorf("Evictions = %d, want 1", got)
	}
}

func TestExpiryIsLazyAndCountsAsMiss(t *testing.T) {
	clock := newFakeClock()
	c := New[string, int](Options{
		MaxEntries: 4,
		TTL:        time.Minute,
		// Disable jitter so the deadline is exactly TTL and the test is precise.
		JitterFraction: 0,
		Now:            clock.Now,
	})
	// JitterFraction 0 with a TTL defaults to 0.1, so force it off explicitly.
	c.jitter = 0

	c.Set("a", 1)
	clock.Advance(59 * time.Second)
	if _, ok := c.Get("a"); !ok {
		t.Fatal("entry expired early")
	}

	clock.Advance(2 * time.Second) // now past the 1-minute deadline
	if _, ok := c.Get("a"); ok {
		t.Fatal("expired entry was served")
	}

	stats := c.Stats()
	if stats.Expirations != 1 {
		t.Errorf("Expirations = %d, want 1", stats.Expirations)
	}
	if stats.Entries != 0 {
		t.Errorf("Entries = %d, want 0 -- expired entry should be reaped on read", stats.Entries)
	}
}

func TestZeroTTLNeverExpires(t *testing.T) {
	clock := newFakeClock()
	c := New[string, int](Options{MaxEntries: 4, Now: clock.Now})

	c.Set("a", 1)
	clock.Advance(1000 * time.Hour)

	if _, ok := c.Get("a"); !ok {
		t.Error("entry expired despite TTL being zero (meaning: no expiry)")
	}
}

// The stampede guard: jitter must pull deadlines strictly earlier than the
// nominal TTL, never later. Serving data past its stated freshness is worse
// than expiring early.
func TestJitterOnlyShortensLifetime(t *testing.T) {
	clock := newFakeClock()
	const ttl = 100 * time.Second

	c := New[string, int](Options{
		MaxEntries:     1000,
		TTL:            ttl,
		JitterFraction: 0.5,
		Now:            clock.Now,
	})

	seen := make(map[time.Time]bool)
	for i := range 200 {
		key := fmt.Sprintf("k%d", i)
		c.Set(key, i)

		c.mu.Lock()
		ent := c.items[key].Value.(*entry[string, int])
		c.mu.Unlock()

		lifetime := ent.expiresAt.Sub(clock.Now())
		if lifetime > ttl {
			t.Fatalf("lifetime %v exceeds TTL %v -- jitter must never extend it", lifetime, ttl)
		}
		if lifetime < ttl/2 {
			t.Fatalf("lifetime %v below the 50%% jitter floor %v", lifetime, ttl/2)
		}
		seen[ent.expiresAt] = true
	}

	// The whole point is spread. If every deadline landed on the same instant,
	// jitter is not doing its job and all replicas would stampede together.
	if len(seen) < 100 {
		t.Errorf("only %d distinct deadlines across 200 entries; jitter is not spreading expiry", len(seen))
	}
}

func TestDelete(t *testing.T) {
	c := New[string, int](Options{MaxEntries: 4})
	c.Set("a", 1)

	if !c.Delete("a") {
		t.Error("Delete returned false for a present key")
	}
	if _, ok := c.Get("a"); ok {
		t.Error("key survived Delete")
	}
	if c.Delete("a") {
		t.Error("Delete returned true for an absent key")
	}
}

func TestPurge(t *testing.T) {
	c := New[string, int](Options{MaxEntries: 4})
	c.Set("a", 1)
	c.Set("b", 2)

	c.Purge()

	if c.Len() != 0 {
		t.Errorf("Len after Purge = %d, want 0", c.Len())
	}
	if _, ok := c.Get("a"); ok {
		t.Error("entry survived Purge")
	}
}

func TestStatsHitRatio(t *testing.T) {
	c := New[string, int](Options{MaxEntries: 4})

	if got := c.Stats().HitRatio(); got != 0 {
		t.Errorf("HitRatio on an unread cache = %v, want 0", got)
	}

	c.Set("a", 1)
	c.Get("a")     // hit
	c.Get("miss1") // miss
	c.Get("miss2") // miss

	stats := c.Stats()
	if stats.Hits != 1 || stats.Misses != 2 {
		t.Fatalf("Hits/Misses = %d/%d, want 1/2", stats.Hits, stats.Misses)
	}
	if got, want := stats.HitRatio(), 1.0/3.0; got != want {
		t.Errorf("HitRatio = %v, want %v", got, want)
	}
}

func TestNewPanicsOnZeroCapacity(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New did not panic on MaxEntries = 0")
		}
	}()
	_ = New[string, int](Options{MaxEntries: 0})
}

func TestJitterFractionIsClamped(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{"negative clamps to zero", -0.5, 0},
		{"at one clamps below one", 1.0, 0.99},
		{"above one clamps below one", 5.0, 0.99},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New[string, int](Options{MaxEntries: 1, TTL: time.Minute, JitterFraction: tt.input})
			if c.jitter != tt.want {
				t.Errorf("jitter = %v, want %v", c.jitter, tt.want)
			}
		})
	}
}

// Guards against the cache being the thing that breaks under a ticket drop.
// Run with -race; a data race here is a production outage.
func TestConcurrentAccessIsSafe(t *testing.T) {
	c := New[int, int](Options{MaxEntries: 64, TTL: time.Millisecond})

	const goroutines = 16
	const opsEach = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := range goroutines {
		go func(g int) {
			defer wg.Done()
			for i := range opsEach {
				key := (g*opsEach + i) % 128
				c.Set(key, i)
				c.Get(key)
				if i%50 == 0 {
					c.Delete(key)
					_ = c.Stats()
				}
			}
		}(g)
	}
	wg.Wait()

	if c.Len() > 64 {
		t.Errorf("Len = %d, exceeds MaxEntries 64 under concurrency", c.Len())
	}
}

func BenchmarkGetHit(b *testing.B) {
	c := New[string, int](Options{MaxEntries: 1024, TTL: time.Minute})
	c.Set("hot", 1)

	b.ResetTimer()
	for range b.N {
		c.Get("hot")
	}
}

func BenchmarkSetWithEviction(b *testing.B) {
	c := New[int, int](Options{MaxEntries: 1024, TTL: time.Minute})

	b.ResetTimer()
	for i := range b.N {
		c.Set(i, i)
	}
}
