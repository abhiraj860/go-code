package repo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/abhiraj860/ticketflow/pkg/postgres"
	inventory "github.com/abhiraj860/ticketflow/services/inventory-svc"
	"github.com/abhiraj860/ticketflow/services/inventory-svc/internal/domain"
)

const adminDSN = "postgres://ticketflow:ticketflow@localhost:5432/ticketflow?sslmode=disable"

// newTestRepo spins up a scratch database with inventory's schema applied.
// Skips when no stack is running so `go test ./...` stays green without Docker.
func newTestRepo(t *testing.T) *SeatRepo {
	t.Helper()

	admin := os.Getenv("TEST_POSTGRES_ADMIN_DSN")
	if admin == "" {
		admin = adminDSN
	}

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, admin)
	if err != nil {
		t.Skipf("postgres not reachable (run `make up`): %v", err)
	}
	defer adminPool.Close()

	if err := adminPool.Ping(ctx); err != nil {
		t.Skipf("postgres not reachable (run `make up`): %v", err)
	}

	name := fmt.Sprintf("tf_inv_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("creating scratch database: %v", err)
	}

	dsn := fmt.Sprintf("postgres://ticketflow:ticketflow@localhost:5432/%s?sslmode=disable", name)

	if err := postgres.Migrate(dsn, inventory.Migrations, inventory.MigrationsDir); err != nil {
		t.Fatalf("migrating scratch database: %v", err)
	}

	// A pool large enough that concurrency in the test is real contention on
	// rows, not queueing on connections.
	pool, err := postgres.Connect(ctx, postgres.Options{DSN: dsn, MaxConns: 30})
	if err != nil {
		t.Fatalf("connecting to scratch database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		cleanCtx := context.Background()
		p, err := pgxpool.New(cleanCtx, admin)
		if err != nil {
			return
		}
		defer p.Close()
		_, _ = p.Exec(cleanCtx,
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", name)
		_, _ = p.Exec(cleanCtx, "DROP DATABASE IF EXISTS "+name)
	})

	return NewSeatRepo(pool)
}

func seedSeats(t *testing.T, r *SeatRepo, eventID string, n int) []string {
	t.Helper()

	ids := make([]string, n)
	for i := range n {
		ids[i] = fmt.Sprintf("S-%03d", i+1)
	}
	if _, err := r.SeedSeats(context.Background(), eventID, ids); err != nil {
		t.Fatalf("seeding seats: %v", err)
	}
	return ids
}

func hold(eventID, user string, seats []string, ttl time.Duration) domain.HoldRequest {
	return domain.HoldRequest{
		EventID:        eventID,
		SeatIDs:        seats,
		UserID:         user,
		IdempotencyKey: uuid.NewString(),
		TTL:            ttl,
	}
}

func TestHoldAndRelease(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seats := seedSeats(t, r, "evt-1", 10)

	res, err := r.HoldSeats(ctx, hold("evt-1", "user-1", seats[:2], time.Minute))
	if err != nil {
		t.Fatalf("HoldSeats: %v", err)
	}
	if len(res.Hold.SeatIDs) != 2 {
		t.Fatalf("held %d seats, want 2", len(res.Hold.SeatIDs))
	}

	avail, err := r.GetAvailability(ctx, "evt-1", seats[:2])
	if err != nil {
		t.Fatalf("GetAvailability: %v", err)
	}
	for _, sa := range avail {
		if sa.Status != domain.SeatStatusHeld {
			t.Errorf("seat %s status = %v, want Held", sa.SeatID, sa.Status)
		}
	}

	n, err := r.ReleaseHold(ctx, res.Hold.ID)
	if err != nil {
		t.Fatalf("ReleaseHold: %v", err)
	}
	if n != 2 {
		t.Errorf("released %d seats, want 2", n)
	}

	avail, _ = r.GetAvailability(ctx, "evt-1", seats[:2])
	for _, sa := range avail {
		if sa.Status != domain.SeatStatusAvailable {
			t.Errorf("seat %s status = %v after release, want Available", sa.SeatID, sa.Status)
		}
	}
}

// THE test. Many goroutines race for one seat; exactly one must win.
//
// This is what the whole schema design exists to guarantee, and it is why the
// claim is worth testing under real concurrency rather than with sequential
// statements, which cannot exhibit the race at all.
func TestConcurrentHoldsOnOneSeatProduceExactlyOneWinner(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seedSeats(t, r, "evt-drop", 1)

	const contenders = 50

	// A barrier so every goroutine attempts the claim at genuinely the same
	// moment, rather than trickling in and serialising naturally.
	start := make(chan struct{})
	var wg sync.WaitGroup

	type outcome struct {
		holdID string
		err    error
	}
	results := make([]outcome, contenders)

	wg.Add(contenders)
	for i := range contenders {
		go func(i int) {
			defer wg.Done()
			<-start
			res, err := r.HoldSeats(ctx, hold("evt-drop", fmt.Sprintf("user-%d", i), []string{"S-001"}, time.Minute))
			results[i] = outcome{holdID: res.Hold.ID, err: err}
		}(i)
	}

	close(start)
	wg.Wait()

	var winners, losers, unexpected int
	winningHolds := make(map[string]struct{})
	for i, res := range results {
		switch {
		case res.err == nil:
			winners++
			winningHolds[res.holdID] = struct{}{}
		case errors.Is(res.err, domain.ErrNoSeatsAvailable):
			losers++
		default:
			unexpected++
			t.Errorf("contender %d got an unexpected error: %v", i, res.err)
		}
	}

	if winners != 1 {
		t.Fatalf("SEAT SOLD TWICE: %d winners for one seat (losers=%d, errors=%d)",
			winners, losers, unexpected)
	}
	if losers != contenders-1 {
		t.Errorf("losers = %d, want %d", losers, contenders-1)
	}
	if len(winningHolds) != 1 {
		t.Errorf("distinct winning hold ids = %d, want 1", len(winningHolds))
	}

	// The database must agree: exactly one seat row, held by one hold.
	avail, err := r.GetAvailability(ctx, "evt-drop", nil)
	if err != nil {
		t.Fatalf("GetAvailability: %v", err)
	}
	if len(avail) != 1 || avail[0].Status != domain.SeatStatusHeld {
		t.Errorf("final state = %+v, want exactly one Held seat", avail)
	}
}

// Partial success is normal during a drop and must be reported accurately, so
// the UI can tell a buyer which seats they actually got.
func TestPartialHoldReportsRejectedSeats(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seats := seedSeats(t, r, "evt-2", 5)

	// user-1 takes S-001 and S-002.
	if _, err := r.HoldSeats(ctx, hold("evt-2", "user-1", seats[:2], time.Minute)); err != nil {
		t.Fatalf("first hold: %v", err)
	}

	// user-2 asks for S-002..S-004; only S-003 and S-004 are free.
	res, err := r.HoldSeats(ctx, hold("evt-2", "user-2", seats[1:4], time.Minute))
	if err != nil {
		t.Fatalf("second hold: %v", err)
	}

	if len(res.Hold.SeatIDs) != 2 {
		t.Errorf("won %v, want 2 seats", res.Hold.SeatIDs)
	}
	if len(res.RejectedSeatIDs) != 1 || res.RejectedSeatIDs[0] != "S-002" {
		t.Errorf("rejected = %v, want [S-002]", res.RejectedSeatIDs)
	}
}

func TestHoldFailsWhenEverySeatTaken(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seats := seedSeats(t, r, "evt-3", 2)

	if _, err := r.HoldSeats(ctx, hold("evt-3", "user-1", seats, time.Minute)); err != nil {
		t.Fatalf("first hold: %v", err)
	}

	res, err := r.HoldSeats(ctx, hold("evt-3", "user-2", seats, time.Minute))
	if !errors.Is(err, domain.ErrNoSeatsAvailable) {
		t.Fatalf("err = %v, want ErrNoSeatsAvailable", err)
	}
	if len(res.RejectedSeatIDs) != 2 {
		t.Errorf("rejected = %v, want both seats", res.RejectedSeatIDs)
	}
}

// A retried request must return the original hold, never a second set of seats.
func TestIdempotencyKeyReplaysOriginalHold(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seats := seedSeats(t, r, "evt-4", 10)

	req := hold("evt-4", "user-1", seats[:2], time.Minute)

	first, err := r.HoldSeats(ctx, req)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if first.Replayed {
		t.Error("first call reported Replayed")
	}

	second, err := r.HoldSeats(ctx, req)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !second.Replayed {
		t.Error("retry did not report Replayed")
	}
	if second.Hold.ID != first.Hold.ID {
		t.Errorf("retry hold id = %q, want %q", second.Hold.ID, first.Hold.ID)
	}

	// Only the original two seats may be held; a duplicate acquisition would
	// show up as four.
	avail, _ := r.GetAvailability(ctx, "evt-4", nil)
	var held int
	for _, sa := range avail {
		if sa.Status == domain.SeatStatusHeld {
			held++
		}
	}
	if held != 2 {
		t.Errorf("%d seats held after a retry, want 2 -- the retry acquired extra seats", held)
	}
}

// Concurrent retries of one idempotency key must not double-acquire either.
// This exercises the unique-violation path rather than the pre-check.
func TestConcurrentIdempotentRetriesAcquireOnce(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seats := seedSeats(t, r, "evt-5", 10)

	req := hold("evt-5", "user-1", seats[:3], time.Minute)

	const retries = 20
	start := make(chan struct{})
	var wg sync.WaitGroup
	holdIDs := make([]string, retries)
	errs := make([]error, retries)

	wg.Add(retries)
	for i := range retries {
		go func(i int) {
			defer wg.Done()
			<-start
			res, err := r.HoldSeats(ctx, req)
			holdIDs[i], errs[i] = res.Hold.ID, err
		}(i)
	}
	close(start)
	wg.Wait()

	distinct := make(map[string]struct{})
	for i, err := range errs {
		if err != nil {
			t.Fatalf("retry %d errored: %v", i, err)
		}
		distinct[holdIDs[i]] = struct{}{}
	}
	if len(distinct) != 1 {
		t.Errorf("produced %d distinct holds for one idempotency key, want 1", len(distinct))
	}

	avail, _ := r.GetAvailability(ctx, "evt-5", nil)
	var held int
	for _, sa := range avail {
		if sa.Status == domain.SeatStatusHeld {
			held++
		}
	}
	if held != 3 {
		t.Errorf("%d seats held, want 3", held)
	}
}

// An expired hold must be claimable by someone else in the same statement,
// without waiting for the reaper.
func TestExpiredHoldCanBeStolen(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seedSeats(t, r, "evt-6", 1)

	// Shortest TTL the repo accepts; the service clamps higher, but the repo
	// takes what it is given.
	first, err := r.HoldSeats(ctx, hold("evt-6", "user-1", []string{"S-001"}, 50*time.Millisecond))
	if err != nil {
		t.Fatalf("first hold: %v", err)
	}

	if _, err := r.HoldSeats(ctx, hold("evt-6", "user-2", []string{"S-001"}, time.Minute)); !errors.Is(err, domain.ErrNoSeatsAvailable) {
		t.Fatalf("steal before expiry: err = %v, want ErrNoSeatsAvailable", err)
	}

	time.Sleep(120 * time.Millisecond)

	second, err := r.HoldSeats(ctx, hold("evt-6", "user-2", []string{"S-001"}, time.Minute))
	if err != nil {
		t.Fatalf("steal after expiry: %v", err)
	}
	if second.Hold.ID == first.Hold.ID {
		t.Error("steal returned the original hold id")
	}
}

// A lapsed hold must read as available even before the reaper rewrites the row.
func TestAvailabilityReportsExpiredHoldAsAvailable(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seedSeats(t, r, "evt-7", 1)

	if _, err := r.HoldSeats(ctx, hold("evt-7", "user-1", []string{"S-001"}, 50*time.Millisecond)); err != nil {
		t.Fatalf("hold: %v", err)
	}
	time.Sleep(120 * time.Millisecond)

	avail, err := r.GetAvailability(ctx, "evt-7", nil)
	if err != nil {
		t.Fatalf("GetAvailability: %v", err)
	}
	if avail[0].Status != domain.SeatStatusAvailable {
		t.Errorf("status = %v, want Available -- a lapsed hold must not read as Held", avail[0].Status)
	}
	if !avail[0].HoldExpiresAt.IsZero() {
		t.Error("HoldExpiresAt should be cleared for a lapsed hold")
	}
}

func TestConfirmHoldSellsSeats(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seats := seedSeats(t, r, "evt-8", 5)

	res, err := r.HoldSeats(ctx, hold("evt-8", "user-1", seats[:2], time.Minute))
	if err != nil {
		t.Fatalf("hold: %v", err)
	}

	sold, err := r.ConfirmHold(ctx, res.Hold.ID, "order-1")
	if err != nil {
		t.Fatalf("ConfirmHold: %v", err)
	}
	if len(sold) != 2 {
		t.Errorf("sold %d seats, want 2", len(sold))
	}

	avail, _ := r.GetAvailability(ctx, "evt-8", seats[:2])
	for _, sa := range avail {
		if sa.Status != domain.SeatStatusSold {
			t.Errorf("seat %s = %v, want Sold", sa.SeatID, sa.Status)
		}
	}
}

// Confirming after expiry must fail loudly: the seats may already belong to
// someone else, and silently selling them would be the double-sell bug.
func TestConfirmExpiredHoldFails(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seedSeats(t, r, "evt-9", 1)

	res, err := r.HoldSeats(ctx, hold("evt-9", "user-1", []string{"S-001"}, 50*time.Millisecond))
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	time.Sleep(120 * time.Millisecond)

	if _, err := r.ConfirmHold(ctx, res.Hold.ID, "order-1"); !errors.Is(err, domain.ErrHoldExpired) {
		t.Fatalf("err = %v, want ErrHoldExpired", err)
	}
}

func TestConfirmUnknownHoldIsNotFound(t *testing.T) {
	r := newTestRepo(t)
	if _, err := r.ConfirmHold(context.Background(), uuid.NewString(), "order-1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

// Releasing must never un-sell a paid ticket.
func TestReleaseDoesNotRevertSoldSeats(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seedSeats(t, r, "evt-10", 1)

	res, err := r.HoldSeats(ctx, hold("evt-10", "user-1", []string{"S-001"}, time.Minute))
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	if _, err := r.ConfirmHold(ctx, res.Hold.ID, "order-1"); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	n, err := r.ReleaseHold(ctx, res.Hold.ID)
	if err != nil {
		t.Fatalf("ReleaseHold: %v", err)
	}
	if n != 0 {
		t.Errorf("released %d seats, want 0 -- a sold seat must stay sold", n)
	}

	avail, _ := r.GetAvailability(ctx, "evt-10", nil)
	if avail[0].Status != domain.SeatStatusSold {
		t.Errorf("status = %v after release, want Sold", avail[0].Status)
	}
}

func TestReapExpiredHolds(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seats := seedSeats(t, r, "evt-11", 3)

	if _, err := r.HoldSeats(ctx, hold("evt-11", "user-1", seats, 50*time.Millisecond)); err != nil {
		t.Fatalf("hold: %v", err)
	}
	time.Sleep(120 * time.Millisecond)

	n, err := r.ReapExpiredHolds(ctx, 100)
	if err != nil {
		t.Fatalf("ReapExpiredHolds: %v", err)
	}
	if n != 3 {
		t.Errorf("reaped %d, want 3", n)
	}

	// A second sweep has nothing left to do.
	if n, err = r.ReapExpiredHolds(ctx, 100); err != nil || n != 0 {
		t.Errorf("second sweep = %d, %v; want 0, nil", n, err)
	}
}

func TestSeedSeatsIsIdempotent(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()

	n, err := r.SeedSeats(ctx, "evt-12", []string{"A", "B", "C"})
	if err != nil {
		t.Fatalf("SeedSeats: %v", err)
	}
	if n != 3 {
		t.Errorf("inserted %d, want 3", n)
	}

	// Re-seeding must not reset existing state.
	if _, err := r.HoldSeats(ctx, hold("evt-12", "user-1", []string{"A"}, time.Minute)); err != nil {
		t.Fatalf("hold: %v", err)
	}
	if n, err = r.SeedSeats(ctx, "evt-12", []string{"A", "B", "C"}); err != nil || n != 0 {
		t.Errorf("re-seed = %d, %v; want 0, nil", n, err)
	}

	avail, _ := r.GetAvailability(ctx, "evt-12", []string{"A"})
	if avail[0].Status != domain.SeatStatusHeld {
		t.Error("re-seeding reset a held seat back to available")
	}
}

// Two multi-seat requests overlapping in opposite orders would deadlock under a
// SELECT ... FOR UPDATE implementation that did not sort seat ids. The single
// conditional UPDATE has no such hazard; this guards against a regression to
// the two-step approach.
func TestOverlappingMultiSeatHoldsDoNotDeadlock(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	seedSeats(t, r, "evt-13", 4)

	forward := []string{"S-001", "S-002", "S-003"}
	reverse := []string{"S-003", "S-002", "S-001"}

	const rounds = 25
	done := make(chan error, rounds*2)

	for range rounds {
		go func() {
			_, err := r.HoldSeats(ctx, hold("evt-13", "user-fwd", forward, time.Second))
			done <- err
		}()
		go func() {
			_, err := r.HoldSeats(ctx, hold("evt-13", "user-rev", reverse, time.Second))
			done <- err
		}()
	}

	for range rounds * 2 {
		select {
		case err := <-done:
			// Losing the race is fine; a deadlock error is not.
			if err != nil && !errors.Is(err, domain.ErrNoSeatsAvailable) {
				t.Fatalf("unexpected error (deadlock?): %v", err)
			}
		case <-time.After(20 * time.Second):
			t.Fatal("timed out waiting for holds -- likely a deadlock")
		}
	}
}
