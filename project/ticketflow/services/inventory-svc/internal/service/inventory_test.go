package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/abhiraj860/ticketflow/services/inventory-svc/internal/domain"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRepo models Postgres: it is the authority, and it is correct on its own.
type fakeRepo struct {
	mu sync.Mutex
	// held maps seatID -> owner. Presence means the seat is taken.
	held map[string]string

	holdCalls atomic.Int64
	// seatsSeen records the seat list Postgres was actually asked about, which
	// is how the fast path's filtering is observed.
	seatsSeen [][]string

	err error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{held: make(map[string]string)}
}

func (f *fakeRepo) HoldSeats(_ context.Context, req domain.HoldRequest) (domain.HoldResult, error) {
	f.holdCalls.Add(1)

	f.mu.Lock()
	defer f.mu.Unlock()
	f.seatsSeen = append(f.seatsSeen, append([]string(nil), req.SeatIDs...))

	if f.err != nil {
		return domain.HoldResult{}, f.err
	}

	var won, rejected []string
	for _, s := range req.SeatIDs {
		if _, taken := f.held[s]; taken {
			rejected = append(rejected, s)
			continue
		}
		f.held[s] = req.UserID
		won = append(won, s)
	}

	if len(won) == 0 {
		return domain.HoldResult{RejectedSeatIDs: rejected}, domain.ErrNoSeatsAvailable
	}
	return domain.HoldResult{
		Hold:            domain.Hold{ID: "hold-" + req.UserID, SeatIDs: won},
		RejectedSeatIDs: rejected,
	}, nil
}

func (f *fakeRepo) postgresCalls() int64 { return f.holdCalls.Load() }

func (f *fakeRepo) GetAvailability(context.Context, string, []string) ([]domain.SeatAvailability, error) {
	return nil, nil
}
func (f *fakeRepo) ReleaseHold(context.Context, string) (int, error)              { return 0, nil }
func (f *fakeRepo) ConfirmHold(context.Context, string, string) ([]string, error) { return nil, nil }
func (f *fakeRepo) ReapExpiredHolds(context.Context, int) (int, error)            { return 0, nil }
func (f *fakeRepo) SeedSeats(context.Context, string, []string) (int, error)      { return 0, nil }

// fakeLocker models Redis, including the ability to break on demand.
type fakeLocker struct {
	mu    sync.Mutex
	locks map[string]string // seatID -> owner

	failAcquire error
	failRelease error

	acquireCalls atomic.Int64
	releaseCalls atomic.Int64
}

func newFakeLocker() *fakeLocker {
	return &fakeLocker{locks: make(map[string]string)}
}

func (f *fakeLocker) Acquire(_ context.Context, _ string, seatIDs []string, owner string, _ time.Duration) ([]string, []string, error) {
	f.acquireCalls.Add(1)
	if f.failAcquire != nil {
		return nil, nil, f.failAcquire
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	var acquired, rejected []string
	for _, s := range seatIDs {
		cur, exists := f.locks[s]
		if !exists || cur == owner { // re-acquiring your own lock is allowed
			f.locks[s] = owner
			acquired = append(acquired, s)
		} else {
			rejected = append(rejected, s)
		}
	}
	return acquired, rejected, nil
}

func (f *fakeLocker) Release(_ context.Context, _ string, seatIDs []string, owner string) error {
	f.releaseCalls.Add(1)
	if f.failRelease != nil {
		return f.failRelease
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	for _, s := range seatIDs {
		if f.locks[s] == owner {
			delete(f.locks, s)
		}
	}
	return nil
}

func (f *fakeLocker) lockCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.locks)
}

func newSvc(repo Repo, locker SeatLocker) *Inventory {
	return New(Options{Repo: repo, Locker: locker, Logger: discardLogger()})
}

func req(user string, seats ...string) domain.HoldRequest {
	return domain.HoldRequest{
		EventID:        "evt-1",
		SeatIDs:        seats,
		UserID:         user,
		IdempotencyKey: "key-" + user,
		TTL:            time.Minute,
	}
}

// The point of the fast path: a losing request must never reach Postgres.
func TestLockRejectionSkipsPostgresEntirely(t *testing.T) {
	repo := newFakeRepo()
	locker := newFakeLocker()
	svc := newSvc(repo, locker)
	ctx := context.Background()

	if _, err := svc.HoldSeats(ctx, req("u1", "S-1")); err != nil {
		t.Fatalf("first hold: %v", err)
	}
	before := repo.postgresCalls()

	_, err := svc.HoldSeats(ctx, req("u2", "S-1"))
	if !errors.Is(err, domain.ErrNoSeatsAvailable) {
		t.Fatalf("err = %v, want ErrNoSeatsAvailable", err)
	}
	if got := repo.postgresCalls(); got != before {
		t.Errorf("Postgres was called %d extra times; the fast path should have rejected without a transaction", got-before)
	}
}

// THE safety property. Redis being broken must never produce a second winner.
// Every degraded mode has to fall through to Postgres, which is the authority.
func TestRedisFailureNeverCausesDoubleSell(t *testing.T) {
	repo := newFakeRepo()
	locker := newFakeLocker()
	svc := newSvc(repo, locker)
	ctx := context.Background()

	// Redis dies before anyone has taken anything.
	locker.failAcquire = errors.New("connection refused")

	const contenders = 30
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, contenders)

	wg.Add(contenders)
	for i := range contenders {
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = svc.HoldSeats(ctx, req(string(rune('a'+i)), "S-1"))
		}(i)
	}
	close(start)
	wg.Wait()

	var winners int
	for i, err := range errs {
		switch {
		case err == nil:
			winners++
		case errors.Is(err, domain.ErrNoSeatsAvailable):
		default:
			t.Errorf("contender %d: unexpected error %v", i, err)
		}
	}

	if winners != 1 {
		t.Fatalf("SEAT SOLD TWICE with Redis down: %d winners, want exactly 1", winners)
	}
	// And it must have degraded by *using* Postgres, not by rejecting everyone.
	if repo.postgresCalls() != contenders {
		t.Errorf("Postgres saw %d of %d requests; a lock failure must fall through, never reject",
			repo.postgresCalls(), contenders)
	}
}

// A lock failure must never be read as "seat taken".
func TestLockFailureFallsThroughRatherThanRejecting(t *testing.T) {
	repo := newFakeRepo()
	locker := newFakeLocker()
	locker.failAcquire = errors.New("redis exploded")

	svc := newSvc(repo, locker)

	result, err := svc.HoldSeats(context.Background(), req("u1", "S-1", "S-2"))
	if err != nil {
		t.Fatalf("hold failed while redis was down: %v", err)
	}
	if len(result.Hold.SeatIDs) != 2 {
		t.Errorf("won %v, want both seats", result.Hold.SeatIDs)
	}
}

// Postgres is the authority: a seat locked in Redis but lost in Postgres must
// have its lock released, or the seat stays out of circulation for the TTL.
func TestLocksReleasedForSeatsPostgresDidNotGrant(t *testing.T) {
	repo := newFakeRepo()
	locker := newFakeLocker()
	ctx := context.Background()

	// Postgres already considers S-2 taken, but Redis knows nothing about it.
	repo.held["S-2"] = "someone-else"

	svc := newSvc(repo, locker)

	result, err := svc.HoldSeats(ctx, req("u1", "S-1", "S-2"))
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	if len(result.Hold.SeatIDs) != 1 || result.Hold.SeatIDs[0] != "S-1" {
		t.Fatalf("won %v, want [S-1]", result.Hold.SeatIDs)
	}

	// Only S-1's lock should remain.
	if n := locker.lockCount(); n != 1 {
		t.Errorf("%d locks remain, want 1 -- S-2's lock was not released", n)
	}
}

// When Postgres rejects everything, no locks may be left behind.
func TestLocksReleasedWhenPostgresRejectsAll(t *testing.T) {
	repo := newFakeRepo()
	locker := newFakeLocker()
	repo.held["S-1"] = "someone-else"

	svc := newSvc(repo, locker)

	if _, err := svc.HoldSeats(context.Background(), req("u1", "S-1")); !errors.Is(err, domain.ErrNoSeatsAvailable) {
		t.Fatalf("err = %v, want ErrNoSeatsAvailable", err)
	}
	if n := locker.lockCount(); n != 0 {
		t.Errorf("%d locks leaked after a full rejection, want 0", n)
	}
}

// A retry of the same logical request must re-acquire its own locks rather than
// colliding with them, or idempotent replay would be falsely rejected.
func TestRetryReacquiresItsOwnLocks(t *testing.T) {
	repo := newFakeRepo()
	locker := newFakeLocker()
	svc := newSvc(repo, locker)
	ctx := context.Background()

	r := req("u1", "S-1")

	if _, err := svc.HoldSeats(ctx, r); err != nil {
		t.Fatalf("first: %v", err)
	}
	// Identical request, same idempotency key: must not be blocked by its own
	// lock from the first attempt.
	if _, err := svc.HoldSeats(ctx, r); !errors.Is(err, domain.ErrNoSeatsAvailable) && err != nil {
		t.Fatalf("retry was rejected by its own lock: %v", err)
	}
	if locker.acquireCalls.Load() != 2 {
		t.Errorf("acquire called %d times, want 2", locker.acquireCalls.Load())
	}
}

// Rejections from Redis and from Postgres must both be reported, so the UI can
// show the buyer every seat they missed.
func TestRejectionsFromBothLayersAreReported(t *testing.T) {
	repo := newFakeRepo()
	locker := newFakeLocker()
	ctx := context.Background()

	// S-1 locked in Redis by someone else; S-2 taken in Postgres; S-3 free.
	locker.locks["S-1"] = "other:key"
	repo.held["S-2"] = "other"

	svc := newSvc(repo, locker)

	result, err := svc.HoldSeats(ctx, req("u1", "S-1", "S-2", "S-3"))
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	if len(result.Hold.SeatIDs) != 1 || result.Hold.SeatIDs[0] != "S-3" {
		t.Fatalf("won %v, want [S-3]", result.Hold.SeatIDs)
	}

	got := map[string]bool{}
	for _, s := range result.RejectedSeatIDs {
		got[s] = true
	}
	if !got["S-1"] || !got["S-2"] {
		t.Errorf("rejected = %v, want both S-1 (redis) and S-2 (postgres)", result.RejectedSeatIDs)
	}
}

// Without a locker the service must still be correct -- that is the whole
// argument for Redis being optional.
func TestNilLockerStillCorrect(t *testing.T) {
	repo := newFakeRepo()
	svc := newSvc(repo, nil)
	ctx := context.Background()

	if _, err := svc.HoldSeats(ctx, req("u1", "S-1")); err != nil {
		t.Fatalf("first: %v", err)
	}
	if _, err := svc.HoldSeats(ctx, req("u2", "S-1")); !errors.Is(err, domain.ErrNoSeatsAvailable) {
		t.Fatalf("err = %v, want ErrNoSeatsAvailable", err)
	}
}

func TestValidation(t *testing.T) {
	svc := newSvc(newFakeRepo(), newFakeLocker())
	ctx := context.Background()

	tests := []struct {
		name string
		req  domain.HoldRequest
	}{
		{"no event", domain.HoldRequest{UserID: "u", SeatIDs: []string{"S"}, IdempotencyKey: "k"}},
		{"no user", domain.HoldRequest{EventID: "e", SeatIDs: []string{"S"}, IdempotencyKey: "k"}},
		{"no seats", domain.HoldRequest{EventID: "e", UserID: "u", IdempotencyKey: "k"}},
		{"no idempotency key", domain.HoldRequest{EventID: "e", UserID: "u", SeatIDs: []string{"S"}}},
		{"duplicate seat", domain.HoldRequest{EventID: "e", UserID: "u", IdempotencyKey: "k",
			SeatIDs: []string{"S-1", "S-1"}}},
		{"too many seats", domain.HoldRequest{EventID: "e", UserID: "u", IdempotencyKey: "k",
			SeatIDs: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.HoldSeats(ctx, tt.req); !errors.Is(err, domain.ErrInvalidArgument) {
				t.Errorf("err = %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestClampTTL(t *testing.T) {
	svc := New(Options{
		Repo: newFakeRepo(), Logger: discardLogger(),
		DefaultTTL: 2 * time.Minute, MaxTTL: 10 * time.Minute,
	})

	tests := []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{"zero becomes default", 0, 2 * time.Minute},
		{"negative becomes default", -time.Second, 2 * time.Minute},
		{"below minimum is raised", time.Second, minHoldTTL},
		{"in range is untouched", 5 * time.Minute, 5 * time.Minute},
		{"above max is clamped", time.Hour, 10 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.clampTTL(tt.in); got != tt.want {
				t.Errorf("clampTTL(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// A release failure must not fail the request; the locks lapse on their TTL.
func TestReleaseFailureIsNotFatal(t *testing.T) {
	repo := newFakeRepo()
	locker := newFakeLocker()
	locker.failRelease = errors.New("redis gone")
	repo.held["S-2"] = "other"

	svc := newSvc(repo, locker)

	result, err := svc.HoldSeats(context.Background(), req("u1", "S-1", "S-2"))
	if err != nil {
		t.Fatalf("a release failure broke the request: %v", err)
	}
	if len(result.Hold.SeatIDs) != 1 {
		t.Errorf("won %v, want [S-1]", result.Hold.SeatIDs)
	}
}
