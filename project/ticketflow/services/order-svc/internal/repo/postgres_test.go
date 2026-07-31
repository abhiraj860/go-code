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

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/pkg/postgres"
	"github.com/abhiraj860/ticketflow/pkg/testsupport"
	order "github.com/abhiraj860/ticketflow/services/order-svc"
	"github.com/abhiraj860/ticketflow/services/order-svc/internal/domain"
)

const adminDSN = "postgres://ticketflow:ticketflow@localhost:5432/ticketflow?sslmode=disable"

func newTestRepo(t *testing.T) *OrderRepo {
	t.Helper()

	admin := os.Getenv("TEST_POSTGRES_ADMIN_DSN")
	if admin == "" {
		admin = adminDSN
	}

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, admin)
	if err != nil {
		testsupport.SkipOrFail(t, "postgres not reachable (run `make up`): %v", err)
	}
	defer adminPool.Close()
	if err := adminPool.Ping(ctx); err != nil {
		testsupport.SkipOrFail(t, "postgres not reachable (run `make up`): %v", err)
	}

	name := fmt.Sprintf("tf_ord_test_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatalf("creating scratch database: %v", err)
	}
	dsn := fmt.Sprintf("postgres://ticketflow:ticketflow@localhost:5432/%s?sslmode=disable", name)

	if err := postgres.Migrate(dsn, order.Migrations, order.MigrationsDir); err != nil {
		t.Fatalf("migrating: %v", err)
	}
	pool, err := postgres.Connect(ctx, postgres.Options{DSN: dsn, MaxConns: 20})
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		p, err := pgxpool.New(context.Background(), admin)
		if err != nil {
			return
		}
		defer p.Close()
		_, _ = p.Exec(context.Background(),
			"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1", name)
		_, _ = p.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name)
	})

	return NewOrderRepo(pool)
}

func placeReq(user, hold string) domain.PlaceOrderRequest {
	return domain.PlaceOrderRequest{
		UserID:         user,
		EventID:        "evt-1",
		HoldID:         hold,
		SeatIDs:        []string{"S-1", "S-2"},
		TotalMinor:     250000,
		CurrencyCode:   "INR",
		IdempotencyKey: "key-" + user + "-" + hold,
	}
}

// THE core guarantee: order and message land together or not at all.
func TestOrderAndOutboxAreWrittenAtomically(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()

	o, _, err := r.PlaceOrder(ctx, placeReq("u1", uuid.NewString()))
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	got, err := r.GetOrder(ctx, o.ID)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if got.Status != domain.OrderStatusPending {
		t.Errorf("status = %v, want Pending", got.Status)
	}

	pending, err := r.PendingCount(ctx)
	if err != nil {
		t.Fatalf("PendingCount: %v", err)
	}
	if pending != 1 {
		t.Fatalf("outbox holds %d rows, want 1 -- the message must be written with the order", pending)
	}

	// And the message must actually be a decodable envelope naming this order,
	// not merely a row that exists.
	tx, err := r.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	records, err := r.ClaimOutbox(ctx, tx, 10)
	if err != nil {
		t.Fatalf("ClaimOutbox: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("claimed %d rows, want 1", len(records))
	}

	env, err := tfkafka.Unmarshal[tfkafka.OrderCreated](records[0].Payload)
	if err != nil {
		t.Fatalf("outbox payload is not a valid envelope: %v", err)
	}
	if env.Payload.OrderID != o.ID {
		t.Errorf("envelope names order %q, want %q", env.Payload.OrderID, o.ID)
	}
	if env.ID == "" {
		t.Error("envelope has no id -- consumers cannot dedup redeliveries")
	}
	if records[0].MessageKey != o.ID {
		t.Errorf("message key = %q, want the order id so partitions preserve order", records[0].MessageKey)
	}
}

// A failed order must leave NO outbox row. This is the half of atomicity that
// is easy to get wrong: publishing after the commit would leave a message for
// an order that does not exist.
func TestFailedOrderLeavesNoOutboxRow(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()

	hold := uuid.NewString()
	if _, _, err := r.PlaceOrder(ctx, placeReq("u1", hold)); err != nil {
		t.Fatalf("first order: %v", err)
	}

	// A different user checking out the SAME hold violates the hold unique
	// index, so the whole transaction rolls back.
	req := placeReq("u2", hold)
	if _, _, err := r.PlaceOrder(ctx, req); !errors.Is(err, domain.ErrHoldAlreadyOrdered) {
		t.Fatalf("err = %v, want ErrHoldAlreadyOrdered", err)
	}

	pending, _ := r.PendingCount(ctx)
	if pending != 1 {
		t.Errorf("outbox holds %d rows, want 1 -- the rolled-back order must not have left a message", pending)
	}
}

func TestIdempotentReplayReturnsSameOrder(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	req := placeReq("u1", uuid.NewString())

	first, firstReplayed, err := r.PlaceOrder(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	second, secondReplayed, err := r.PlaceOrder(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	if first.ID != second.ID {
		t.Errorf("retry produced order %q, want %q", second.ID, first.ID)
	}
	if firstReplayed {
		t.Error("the first call reported replayed")
	}
	if !secondReplayed {
		t.Error("the retry did not report replayed")
	}
	// Crucially, the retry must NOT enqueue a second message, or the buyer
	// would receive two sets of tickets.
	if pending, _ := r.PendingCount(ctx); pending != 1 {
		t.Errorf("outbox holds %d rows after a retry, want 1", pending)
	}
}

func TestConcurrentIdempotentRetriesProduceOneOrder(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()
	req := placeReq("u1", uuid.NewString())

	const retries = 15
	start := make(chan struct{})
	var wg sync.WaitGroup
	ids := make([]string, retries)
	errs := make([]error, retries)

	wg.Add(retries)
	for i := range retries {
		go func(i int) {
			defer wg.Done()
			<-start
			o, _, err := r.PlaceOrder(ctx, req)
			ids[i], errs[i] = o.ID, err
		}(i)
	}
	close(start)
	wg.Wait()

	distinct := map[string]struct{}{}
	for i, err := range errs {
		if err != nil {
			t.Fatalf("retry %d: %v", i, err)
		}
		distinct[ids[i]] = struct{}{}
	}
	if len(distinct) != 1 {
		t.Errorf("produced %d distinct orders for one idempotency key, want 1", len(distinct))
	}
	if pending, _ := r.PendingCount(ctx); pending != 1 {
		t.Errorf("outbox holds %d rows, want 1", pending)
	}
}

// SKIP LOCKED is what lets multiple relay replicas run. Two concurrent claims
// must return disjoint sets rather than the second blocking on the first.
func TestConcurrentClaimsAreDisjoint(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()

	for i := range 10 {
		if _, _, err := r.PlaceOrder(ctx, placeReq(fmt.Sprintf("u%d", i), uuid.NewString())); err != nil {
			t.Fatal(err)
		}
	}

	tx1, _ := r.Begin(ctx)
	defer func() { _ = tx1.Rollback(ctx) }()
	tx2, _ := r.Begin(ctx)
	defer func() { _ = tx2.Rollback(ctx) }()

	first, err := r.ClaimOutbox(ctx, tx1, 5)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	second, err := r.ClaimOutbox(ctx, tx2, 5)
	if err != nil {
		t.Fatalf("second claim blocked or failed -- SKIP LOCKED is not working: %v", err)
	}

	if len(first) != 5 || len(second) != 5 {
		t.Fatalf("claims returned %d and %d rows, want 5 each", len(first), len(second))
	}

	seen := map[int64]struct{}{}
	for _, rec := range append(first, second...) {
		if _, dup := seen[rec.ID]; dup {
			t.Errorf("row %d was claimed by both relays", rec.ID)
		}
		seen[rec.ID] = struct{}{}
	}
}

func TestMarkPublishedRemovesFromBacklog(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()

	if _, _, err := r.PlaceOrder(ctx, placeReq("u1", uuid.NewString())); err != nil {
		t.Fatal(err)
	}

	tx, _ := r.Begin(ctx)
	records, _ := r.ClaimOutbox(ctx, tx, 10)
	ids := []int64{records[0].ID}
	if err := r.MarkPublished(ctx, tx, ids); err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if pending, _ := r.PendingCount(ctx); pending != 0 {
		t.Errorf("backlog = %d, want 0", pending)
	}

	// A published row must never be claimed again.
	tx2, _ := r.Begin(ctx)
	defer func() { _ = tx2.Rollback(ctx) }()
	if again, _ := r.ClaimOutbox(ctx, tx2, 10); len(again) != 0 {
		t.Errorf("claimed %d already-published rows", len(again))
	}
}

// THE at-least-once test: a crash after publishing but before marking must
// redeliver, never lose. A lost order.created means a paying customer never
// receives a ticket.
func TestCrashBetweenPublishAndMarkRedelivers(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()

	if _, _, err := r.PlaceOrder(ctx, placeReq("u1", uuid.NewString())); err != nil {
		t.Fatal(err)
	}

	// Sweep one: claim and "publish", then die before committing the mark.
	tx, _ := r.Begin(ctx)
	records, err := r.ClaimOutbox(ctx, tx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("claimed %d, want 1", len(records))
	}
	publishedID := records[0].ID
	if err := r.MarkPublished(ctx, tx, []int64{publishedID}); err != nil {
		t.Fatal(err)
	}
	// The crash: rollback instead of commit.
	_ = tx.Rollback(ctx)

	// Sweep two, after restart. The row must still be pending.
	if pending, _ := r.PendingCount(ctx); pending != 1 {
		t.Fatalf("backlog = %d, want 1 -- the message was LOST by the crash", pending)
	}

	tx2, _ := r.Begin(ctx)
	defer func() { _ = tx2.Rollback(ctx) }()
	again, err := r.ClaimOutbox(ctx, tx2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 1 || again[0].ID != publishedID {
		t.Fatalf("redelivery claimed %v, want the same row %d", again, publishedID)
	}

	// The redelivered envelope must carry the SAME id, or consumers cannot
	// recognise it as a duplicate and would issue a second ticket.
	firstEnv, _ := tfkafka.Unmarshal[tfkafka.OrderCreated](records[0].Payload)
	secondEnv, _ := tfkafka.Unmarshal[tfkafka.OrderCreated](again[0].Payload)
	if firstEnv.ID != secondEnv.ID {
		t.Errorf("redelivery changed the envelope id (%q -> %q); dedup is impossible",
			firstEnv.ID, secondEnv.ID)
	}
}

func TestRecordFailureIncrementsAttempts(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()

	if _, _, err := r.PlaceOrder(ctx, placeReq("u1", uuid.NewString())); err != nil {
		t.Fatal(err)
	}

	tx, _ := r.Begin(ctx)
	records, _ := r.ClaimOutbox(ctx, tx, 10)
	if err := r.RecordFailure(ctx, tx, []int64{records[0].ID}, "broker unreachable"); err != nil {
		t.Fatalf("RecordFailure: %v", err)
	}
	_ = tx.Commit(ctx)

	tx2, _ := r.Begin(ctx)
	defer func() { _ = tx2.Rollback(ctx) }()
	again, _ := r.ClaimOutbox(ctx, tx2, 10)
	if len(again) != 1 {
		t.Fatalf("claimed %d, want 1 -- a failed row must stay pending", len(again))
	}
	if again[0].Attempts != 1 {
		t.Errorf("attempts = %d, want 1", again[0].Attempts)
	}
}

func TestMarkPaid(t *testing.T) {
	r := newTestRepo(t)
	ctx := context.Background()

	o, _, err := r.PlaceOrder(ctx, placeReq("u1", uuid.NewString()))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.MarkPaid(ctx, o.ID); err != nil {
		t.Fatalf("MarkPaid: %v", err)
	}

	got, _ := r.GetOrder(ctx, o.ID)
	if got.Status != domain.OrderStatusPaid {
		t.Errorf("status = %v, want Paid", got.Status)
	}

	// Only a PENDING order transitions; a second call is a no-op, not a
	// silent success.
	if err := r.MarkPaid(ctx, o.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second MarkPaid = %v, want ErrNotFound", err)
	}
}

func TestGetUnknownOrder(t *testing.T) {
	r := newTestRepo(t)
	if _, err := r.GetOrder(context.Background(), "nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
