package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeStore is an in-memory Store so the loader's coalescing and negative
// caching can be tested without a Redis container.
type fakeStore struct {
	mu      sync.Mutex
	data    map[string][]byte
	getErr  error // when set, every Get fails -- simulates Redis being down
	getCall atomic.Uint64
	setCall atomic.Uint64
	delCall atomic.Uint64
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: make(map[string][]byte)}
}

func (f *fakeStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	f.getCall.Add(1)
	if f.getErr != nil {
		return nil, false, f.getErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.data[key]
	return v, ok, nil
}

func (f *fakeStore) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	f.setCall.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = value
	return nil
}

func (f *fakeStore) Del(_ context.Context, keys ...string) error {
	f.delCall.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range keys {
		delete(f.data, k)
	}
	return nil
}

type jsonCodec[V any] struct{}

func (jsonCodec[V]) Marshal(v V) ([]byte, error) { return json.Marshal(v) }
func (jsonCodec[V]) Unmarshal(b []byte) (V, error) {
	var v V
	err := json.Unmarshal(b, &v)
	return v, err
}

type event struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

func newTestLoader(t *testing.T, store Store, fetch Fetch[string, event]) *Loader[string, event] {
	t.Helper()
	return NewLoader(LoaderOptions[string, event]{
		L1:          New[string, event](Options{MaxEntries: 100, TTL: time.Minute}),
		L2:          store,
		Codec:       jsonCodec[event]{},
		KeyFn:       func(k string) string { return "event:" + k },
		L2TTL:       5 * time.Minute,
		NegativeTTL: 10 * time.Second,
		Fetch:       fetch,
	})
}

func TestLoaderReadThroughPopulatesBothTiers(t *testing.T) {
	store := newFakeStore()
	var fetches atomic.Uint64

	l := newTestLoader(t, store, func(_ context.Context, k string) (event, error) {
		fetches.Add(1)
		return event{ID: k, Title: "Concert"}, nil
	})

	ctx := context.Background()

	got, err := l.Get(ctx, "e1")
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if got.Title != "Concert" {
		t.Errorf("Title = %q, want Concert", got.Title)
	}
	if n := fetches.Load(); n != 1 {
		t.Errorf("fetches = %d, want 1", n)
	}

	// Second read must be served by L1 with no further fetch.
	if _, err := l.Get(ctx, "e1"); err != nil {
		t.Fatalf("second Get: %v", err)
	}
	if n := fetches.Load(); n != 1 {
		t.Errorf("fetches after L1 hit = %d, want 1", n)
	}

	stats := l.Stats()
	if stats.L1Hits != 1 || stats.Fetches != 1 {
		t.Errorf("L1Hits/Fetches = %d/%d, want 1/1", stats.L1Hits, stats.Fetches)
	}
	if store.setCall.Load() != 1 {
		t.Errorf("L2 writes = %d, want 1", store.setCall.Load())
	}
}

func TestLoaderServesFromL2WhenL1Cold(t *testing.T) {
	store := newFakeStore()
	var fetches atomic.Uint64

	l := newTestLoader(t, store, func(_ context.Context, k string) (event, error) {
		fetches.Add(1)
		return event{ID: k, Title: "Match"}, nil
	})
	ctx := context.Background()

	if _, err := l.Get(ctx, "e1"); err != nil {
		t.Fatal(err)
	}

	// Simulate a fresh replica: L1 empty, L2 warm.
	l.opts.L1.Purge()

	got, err := l.Get(ctx, "e1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Match" {
		t.Errorf("Title = %q, want Match", got.Title)
	}
	if n := fetches.Load(); n != 1 {
		t.Errorf("fetches = %d, want 1 -- L2 should have served this", n)
	}
	if l.Stats().L2Hits != 1 {
		t.Errorf("L2Hits = %d, want 1", l.Stats().L2Hits)
	}
}

// The headline behaviour: a hot key expiring mid-drop must not turn N
// concurrent requests into N database queries.
func TestStampedeIsCollapsedToOneFetch(t *testing.T) {
	store := newFakeStore()

	var fetches atomic.Uint64
	release := make(chan struct{})

	l := newTestLoader(t, store, func(_ context.Context, k string) (event, error) {
		fetches.Add(1)
		<-release // hold the fetch open so every goroutine piles up behind it
		return event{ID: k, Title: "Sold Out Show"}, nil
	})

	const goroutines = 200
	var wg sync.WaitGroup
	errs := make([]error, goroutines)
	vals := make([]event, goroutines)

	wg.Add(goroutines)
	for i := range goroutines {
		go func(i int) {
			defer wg.Done()
			vals[i], errs[i] = l.Get(context.Background(), "hot-event")
		}(i)
	}

	// Let every goroutine reach the loader before the fetch returns.
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if n := fetches.Load(); n != 1 {
		t.Errorf("origin was called %d times for %d concurrent readers; want exactly 1", n, goroutines)
	}
	for i := range goroutines {
		if errs[i] != nil {
			t.Fatalf("goroutine %d errored: %v", i, errs[i])
		}
		if vals[i].Title != "Sold Out Show" {
			t.Fatalf("goroutine %d got %q, want the shared result", i, vals[i].Title)
		}
	}
	if c := l.Stats().Coalesced; c == 0 {
		t.Error("Coalesced = 0; the stampede guard did not record any waiters")
	}
}

func TestNegativeCachingStopsRepeatOriginCalls(t *testing.T) {
	store := newFakeStore()
	var fetches atomic.Uint64

	l := newTestLoader(t, store, func(_ context.Context, _ string) (event, error) {
		fetches.Add(1)
		return event{}, ErrNotFound
	})
	ctx := context.Background()

	for i := range 5 {
		_, err := l.Get(ctx, "does-not-exist")
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("attempt %d: err = %v, want ErrNotFound", i, err)
		}
		// L1 holds no entry for a miss, so each attempt re-reads L2; the
		// marker there is what must prevent a second origin call.
		l.opts.L1.Purge()
	}

	if n := fetches.Load(); n != 1 {
		t.Errorf("origin called %d times for a known-absent key; want 1 (negative cache should absorb the rest)", n)
	}
	if l.Stats().NegativeHits == 0 {
		t.Error("NegativeHits = 0; the negative marker was never read back")
	}
}

// Redis being unavailable must degrade latency, not availability.
func TestL2FailureFallsThroughToOrigin(t *testing.T) {
	store := newFakeStore()
	store.getErr = errors.New("connection refused")

	var fetches atomic.Uint64
	l := newTestLoader(t, store, func(_ context.Context, k string) (event, error) {
		fetches.Add(1)
		return event{ID: k, Title: "Still Served"}, nil
	})

	got, err := l.Get(context.Background(), "e1")
	if err != nil {
		t.Fatalf("request failed while L2 was down: %v", err)
	}
	if got.Title != "Still Served" {
		t.Errorf("Title = %q, want Still Served", got.Title)
	}
	if l.Stats().Errors == 0 {
		t.Error("Errors = 0; the L2 failure should still have been counted")
	}
}

func TestInvalidateClearsBothTiers(t *testing.T) {
	store := newFakeStore()
	var fetches atomic.Uint64

	l := newTestLoader(t, store, func(_ context.Context, k string) (event, error) {
		n := fetches.Add(1)
		return event{ID: k, Title: fmt.Sprintf("v%d", n)}, nil
	})
	ctx := context.Background()

	if _, err := l.Get(ctx, "e1"); err != nil {
		t.Fatal(err)
	}
	if err := l.Invalidate(ctx, "e1"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}

	got, err := l.Get(ctx, "e1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "v2" {
		t.Errorf("Title = %q, want v2 -- a re-fetch should have happened", got.Title)
	}
}

// InvalidateLocal is what each replica's Kafka consumer calls. It must drop the
// in-process copy without issuing a redundant Redis delete, since the publisher
// already removed the shared entry.
func TestInvalidateLocalDoesNotTouchL2(t *testing.T) {
	store := newFakeStore()
	l := newTestLoader(t, store, func(_ context.Context, k string) (event, error) {
		return event{ID: k}, nil
	})
	ctx := context.Background()

	if _, err := l.Get(ctx, "e1"); err != nil {
		t.Fatal(err)
	}
	before := store.delCall.Load()

	l.InvalidateLocal("e1")

	if got := store.delCall.Load(); got != before {
		t.Errorf("L2 Del calls = %d, want %d -- InvalidateLocal must not hit Redis", got, before)
	}
	if _, ok := l.opts.L1.Get("e1"); ok {
		t.Error("entry survived InvalidateLocal in L1")
	}
}

func TestCorruptL2EntryIsEvictedAndRefetched(t *testing.T) {
	store := newFakeStore()
	var fetches atomic.Uint64

	l := newTestLoader(t, store, func(_ context.Context, k string) (event, error) {
		fetches.Add(1)
		return event{ID: k, Title: "Recovered"}, nil
	})
	ctx := context.Background()

	// Plant an entry L2 can return but the codec cannot decode -- e.g. a value
	// written by a previous release with a different schema.
	_ = store.Set(ctx, "event:e1", []byte("{not json"), time.Minute)

	got, err := l.Get(ctx, "e1")
	if err != nil {
		t.Fatalf("corrupt L2 entry broke the read: %v", err)
	}
	if got.Title != "Recovered" {
		t.Errorf("Title = %q, want Recovered", got.Title)
	}
	if store.delCall.Load() == 0 {
		t.Error("corrupt entry was not evicted from L2")
	}
}

func TestLoaderWithoutL2IsL1Only(t *testing.T) {
	var fetches atomic.Uint64
	l := NewLoader(LoaderOptions[string, event]{
		L1: New[string, event](Options{MaxEntries: 10, TTL: time.Minute}),
		Fetch: func(_ context.Context, k string) (event, error) {
			fetches.Add(1)
			return event{ID: k}, nil
		},
	})
	ctx := context.Background()

	for range 3 {
		if _, err := l.Get(ctx, "e1"); err != nil {
			t.Fatal(err)
		}
	}
	if n := fetches.Load(); n != 1 {
		t.Errorf("fetches = %d, want 1", n)
	}
}

func TestNewLoaderPanicsOnMissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		opts LoaderOptions[string, event]
	}{
		{"no L1", LoaderOptions[string, event]{
			Fetch: func(context.Context, string) (event, error) { return event{}, nil },
		}},
		{"no Fetch", LoaderOptions[string, event]{
			L1: New[string, event](Options{MaxEntries: 1}),
		}},
		{"L2 without Codec", LoaderOptions[string, event]{
			L1:    New[string, event](Options{MaxEntries: 1}),
			L2:    newFakeStore(),
			KeyFn: func(k string) string { return k },
			Fetch: func(context.Context, string) (event, error) { return event{}, nil },
		}},
		{"L2 without KeyFn", LoaderOptions[string, event]{
			L1:    New[string, event](Options{MaxEntries: 1}),
			L2:    newFakeStore(),
			Codec: jsonCodec[event]{},
			Fetch: func(context.Context, string) (event, error) { return event{}, nil },
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("NewLoader did not panic")
				}
			}()
			_ = NewLoader(tt.opts)
		})
	}
}
