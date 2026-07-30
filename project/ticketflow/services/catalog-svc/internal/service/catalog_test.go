package service

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/domain"
)

// stubRepo is an in-memory Repo, so the service's caching policy can be tested
// without a database. Call counts are what the assertions are actually about:
// the policy is only observable through how often the origin is consulted.
type stubRepo struct {
	mu sync.Mutex

	events   map[string]domain.Event
	seatMaps map[string]domain.SeatMap

	eventCalls   atomic.Int64
	seatMapCalls atomic.Int64
	listCalls    atomic.Int64

	// lastFilter records what ListEvents received, to assert clamping.
	lastFilter domain.ListFilter

	err error // when set, every call fails
}

func newStubRepo() *stubRepo {
	return &stubRepo{
		events:   make(map[string]domain.Event),
		seatMaps: make(map[string]domain.SeatMap),
	}
}

func (s *stubRepo) GetEvent(_ context.Context, id string) (domain.Event, error) {
	s.eventCalls.Add(1)
	if s.err != nil {
		return domain.Event{}, s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.events[id]
	if !ok {
		return domain.Event{}, domain.ErrNotFound
	}
	return e, nil
}

func (s *stubRepo) GetSeatMap(_ context.Context, id string) (domain.SeatMap, error) {
	s.seatMapCalls.Add(1)
	if s.err != nil {
		return domain.SeatMap{}, s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.seatMaps[id]
	if !ok {
		return domain.SeatMap{}, domain.ErrNotFound
	}
	return m, nil
}

func (s *stubRepo) ListEvents(_ context.Context, f domain.ListFilter) (domain.Page[domain.Event], error) {
	s.listCalls.Add(1)
	s.mu.Lock()
	s.lastFilter = f
	s.mu.Unlock()
	if s.err != nil {
		return domain.Page[domain.Event]{}, s.err
	}
	return domain.Page[domain.Event]{}, nil
}

func (s *stubRepo) filter() domain.ListFilter {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastFilter
}

// stubContent is an in-memory ContentRepo.
type stubContent struct {
	docs  map[string]domain.EventContent
	calls atomic.Int64
}

func (s *stubContent) GetContent(_ context.Context, id string) (domain.EventContent, error) {
	s.calls.Add(1)
	c, ok := s.docs[id]
	if !ok {
		return domain.EventContent{}, domain.ErrNotFound
	}
	return c, nil
}

func newTestCatalog(t *testing.T, repo Repo, content ContentRepo) *Catalog {
	t.Helper()
	// No L2: the Redis tier has its own tests in pkg/cache. What is under test
	// here is catalog's policy, not the cache implementation.
	return New(Options{
		Repo:       repo,
		Content:    content,
		EventTTL:   time.Minute,
		SeatMapTTL: time.Minute,
	})
}

func TestGetEventIsCachedAfterFirstRead(t *testing.T) {
	repo := newStubRepo()
	repo.events["e1"] = domain.Event{ID: "e1", Title: "Concert", Version: 1}

	svc := newTestCatalog(t, repo, nil)
	ctx := context.Background()

	for i := range 5 {
		got, err := svc.GetEvent(ctx, "e1")
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		if got.Title != "Concert" {
			t.Fatalf("read %d: Title = %q", i, got.Title)
		}
	}

	if n := repo.eventCalls.Load(); n != 1 {
		t.Errorf("repo consulted %d times for 5 reads, want 1", n)
	}
}

// domain.ErrNotFound must survive translation through cache.ErrNotFound and
// come back out as domain.ErrNotFound, or callers cannot distinguish a missing
// event from a broken one.
func TestGetEventNotFoundRoundTripsAsDomainError(t *testing.T) {
	repo := newStubRepo()
	svc := newTestCatalog(t, repo, nil)

	_, err := svc.GetEvent(context.Background(), "nope")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want domain.ErrNotFound", err)
	}
}

// Negative caching: repeated lookups of a missing event must not repeatedly hit
// the database. Without an L2 the L1 holds no entry, but the loader's
// single-flight still collapses concurrent misses -- so this asserts the
// concurrent case, which is the one that matters under bot traffic.
func TestConcurrentMissesCollapseToOneRepoCall(t *testing.T) {
	repo := newStubRepo()
	repo.events["hot"] = domain.Event{ID: "hot", Title: "Sold Out"}

	svc := newTestCatalog(t, repo, nil)

	const readers = 100
	start := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(readers)
	for range readers {
		go func() {
			defer wg.Done()
			<-start
			_, _ = svc.GetEvent(context.Background(), "hot")
		}()
	}
	close(start)
	wg.Wait()

	if n := repo.eventCalls.Load(); n != 1 {
		t.Errorf("repo consulted %d times for %d concurrent readers, want 1", n, readers)
	}
}

func TestGetSeatMapIsCached(t *testing.T) {
	repo := newStubRepo()
	repo.seatMaps["m1"] = domain.SeatMap{ID: "m1", Seats: []domain.Seat{{ID: "A-1"}}}

	svc := newTestCatalog(t, repo, nil)
	ctx := context.Background()

	for range 3 {
		if _, err := svc.GetSeatMap(ctx, "m1"); err != nil {
			t.Fatal(err)
		}
	}
	if n := repo.seatMapCalls.Load(); n != 1 {
		t.Errorf("repo consulted %d times, want 1", n)
	}
}

func TestInvalidateForcesRefetch(t *testing.T) {
	repo := newStubRepo()
	repo.events["e1"] = domain.Event{ID: "e1", Title: "v1"}

	svc := newTestCatalog(t, repo, nil)
	ctx := context.Background()

	if _, err := svc.GetEvent(ctx, "e1"); err != nil {
		t.Fatal(err)
	}

	// Simulate an admin edit landing in the database.
	repo.mu.Lock()
	repo.events["e1"] = domain.Event{ID: "e1", Title: "v2"}
	repo.mu.Unlock()

	// Without invalidation the cache still serves the stale copy.
	if got, _ := svc.GetEvent(ctx, "e1"); got.Title != "v1" {
		t.Fatalf("expected the stale cached copy, got %q", got.Title)
	}

	if err := svc.InvalidateEvent(ctx, "e1"); err != nil {
		t.Fatalf("InvalidateEvent: %v", err)
	}

	got, err := svc.GetEvent(ctx, "e1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "v2" {
		t.Errorf("Title = %q after invalidation, want v2", got.Title)
	}
}

// InvalidateEventLocal is what each replica's Kafka consumer will call in
// Phase 2. It must drop the in-process copy.
func TestInvalidateLocalDropsCachedCopy(t *testing.T) {
	repo := newStubRepo()
	repo.events["e1"] = domain.Event{ID: "e1", Title: "v1"}

	svc := newTestCatalog(t, repo, nil)
	ctx := context.Background()

	if _, err := svc.GetEvent(ctx, "e1"); err != nil {
		t.Fatal(err)
	}
	svc.InvalidateEventLocal("e1")

	if _, err := svc.GetEvent(ctx, "e1"); err != nil {
		t.Fatal(err)
	}
	if n := repo.eventCalls.Load(); n != 2 {
		t.Errorf("repo consulted %d times, want 2 -- the local copy should have been dropped", n)
	}
}

// A client asking for too much should get a sane page, not an error.
func TestListEventsClampsPageSize(t *testing.T) {
	tests := []struct {
		name string
		in   int32
		want int32
	}{
		{"zero becomes the default", 0, defaultPageSize},
		{"within range is untouched", 50, 50},
		{"oversized is clamped", 5000, maxPageSize},
		{"exactly at the max", maxPageSize, maxPageSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newStubRepo()
			svc := newTestCatalog(t, repo, nil)

			if _, err := svc.ListEvents(context.Background(), domain.ListFilter{PageSize: tt.in}); err != nil {
				t.Fatalf("ListEvents: %v", err)
			}
			if got := repo.filter().PageSize; got != tt.want {
				t.Errorf("repo received PageSize %d, want %d", got, tt.want)
			}
		})
	}
}

func TestListEventsRejectsNegativePageSize(t *testing.T) {
	svc := newTestCatalog(t, newStubRepo(), nil)

	_, err := svc.ListEvents(context.Background(), domain.ListFilter{PageSize: -1})
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("err = %v, want ErrInvalidArgument", err)
	}
}

// Listing must not be cached: filters and cursors make the key space unbounded,
// so the hit ratio would be poor and staleness would show on every browse.
func TestListEventsIsNotCached(t *testing.T) {
	repo := newStubRepo()
	svc := newTestCatalog(t, repo, nil)
	ctx := context.Background()

	for range 3 {
		if _, err := svc.ListEvents(ctx, domain.ListFilter{}); err != nil {
			t.Fatal(err)
		}
	}
	if n := repo.listCalls.Load(); n != 3 {
		t.Errorf("repo consulted %d times, want 3 -- listing must not be cached", n)
	}
}

func TestEmptyIDsAreRejected(t *testing.T) {
	svc := newTestCatalog(t, newStubRepo(), nil)
	ctx := context.Background()

	if _, err := svc.GetEvent(ctx, ""); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("GetEvent(\"\") = %v, want ErrInvalidArgument", err)
	}
	if _, err := svc.GetSeatMap(ctx, ""); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("GetSeatMap(\"\") = %v, want ErrInvalidArgument", err)
	}
	if _, err := svc.GetEventContent(ctx, ""); !errors.Is(err, ErrInvalidArgument) {
		t.Errorf("GetEventContent(\"\") = %v, want ErrInvalidArgument", err)
	}
}

// Mongo is optional. With no ContentRepo the service must report NotFound
// rather than panicking on a nil loader -- content is enrichment, and an event
// page without a setlist is still a usable page.
func TestGetEventContentWithoutMongoReportsNotFound(t *testing.T) {
	svc := newTestCatalog(t, newStubRepo(), nil)

	_, err := svc.GetEventContent(context.Background(), "e1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want domain.ErrNotFound", err)
	}
}

func TestGetEventContentIsCached(t *testing.T) {
	content := &stubContent{docs: map[string]domain.EventContent{
		"e1": {EventID: "e1", Kind: domain.EventKindConcert, Body: map[string]any{"summary": "hi"}},
	}}

	svc := newTestCatalog(t, newStubRepo(), content)
	ctx := context.Background()

	for range 4 {
		got, err := svc.GetEventContent(ctx, "e1")
		if err != nil {
			t.Fatal(err)
		}
		if got.Body["summary"] != "hi" {
			t.Fatalf("Body = %v", got.Body)
		}
	}
	if n := content.calls.Load(); n != 1 {
		t.Errorf("mongo consulted %d times for 4 reads, want 1", n)
	}
}

func TestGetEventContentMissingDocument(t *testing.T) {
	content := &stubContent{docs: map[string]domain.EventContent{}}
	svc := newTestCatalog(t, newStubRepo(), content)

	if _, err := svc.GetEventContent(context.Background(), "absent"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want domain.ErrNotFound", err)
	}
}

// Every cache must appear in CacheStats. The content cache was initially
// missing from the metrics output, which is exactly the failure this guards:
// an uninstrumented cache is a guess.
func TestCacheStatsCoversEveryConfiguredCache(t *testing.T) {
	t.Run("without mongo", func(t *testing.T) {
		svc := newTestCatalog(t, newStubRepo(), nil)
		stats := svc.CacheStats()

		for _, want := range []string{"event", "seatmap"} {
			if _, ok := stats[want]; !ok {
				t.Errorf("CacheStats is missing %q", want)
			}
		}
		if _, ok := stats["content"]; ok {
			t.Error("CacheStats reports a content cache that was never configured")
		}
	})

	t.Run("with mongo", func(t *testing.T) {
		svc := newTestCatalog(t, newStubRepo(), &stubContent{docs: map[string]domain.EventContent{}})
		stats := svc.CacheStats()

		for _, want := range []string{"event", "seatmap", "content"} {
			if _, ok := stats[want]; !ok {
				t.Errorf("CacheStats is missing %q", want)
			}
		}
	})
}

func TestCacheStatsCountsHitsAndFetches(t *testing.T) {
	repo := newStubRepo()
	repo.events["e1"] = domain.Event{ID: "e1"}

	svc := newTestCatalog(t, repo, nil)
	ctx := context.Background()

	for range 3 {
		_, _ = svc.GetEvent(ctx, "e1")
	}

	stats := svc.CacheStats()["event"]
	if stats.Fetches != 1 {
		t.Errorf("Fetches = %d, want 1", stats.Fetches)
	}
	if stats.L1Hits != 2 {
		t.Errorf("L1Hits = %d, want 2", stats.L1Hits)
	}
	if got, want := stats.HitRatio(), 2.0/3.0; got != want {
		t.Errorf("HitRatio = %v, want %v", got, want)
	}
}

// A repo failure that is not ErrNotFound must propagate, not be swallowed into
// an empty result -- otherwise a database outage looks like an empty catalogue.
func TestRepoErrorPropagates(t *testing.T) {
	repo := newStubRepo()
	repo.err = errors.New("connection reset")

	svc := newTestCatalog(t, repo, nil)
	ctx := context.Background()

	if _, err := svc.GetEvent(ctx, "e1"); err == nil || errors.Is(err, domain.ErrNotFound) {
		t.Errorf("GetEvent err = %v, want the underlying failure", err)
	}
	if _, err := svc.ListEvents(ctx, domain.ListFilter{}); err == nil {
		t.Error("ListEvents swallowed a repo failure")
	}
}
