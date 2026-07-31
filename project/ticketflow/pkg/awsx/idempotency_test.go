package awsx

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/abhiraj860/ticketflow/pkg/testsupport"
)

// newTestStore talks to a real DynamoDB (LocalStack). A fake cannot reproduce
// what this package is for: the atomicity of a conditional write under
// concurrency is a property of DynamoDB, not of the Go code wrapping it.
func newTestStore(t *testing.T) *Store {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := Load(ctx, Config{
		Region: "us-east-1", Endpoint: "http://localhost:4566",
		AccessKey: "test", SecretKey: "test",
	})
	if err != nil {
		testsupport.SkipOrFail(t, "loading aws config: %v", err)
	}

	table := fmt.Sprintf("tf_idem_test_%d", time.Now().UnixNano())
	store, err := NewStore(Options{Client: NewDynamoDB(cfg), Table: table, TTL: time.Hour})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	if err := store.EnsureTable(ctx); err != nil {
		testsupport.SkipOrFail(t, "dynamodb not reachable (run `make up-aws`): %v", err)
	}
	return store
}

func TestFirstCallWinsTheClaim(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	rec, err := store.Begin(ctx, "key-1", "hash-a")
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if rec != nil {
		t.Fatal("a fresh key returned an existing record")
	}
}

// A retry arriving after the work finished must replay the stored response
// rather than doing the work again.
func TestCompletedRequestIsReplayed(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Begin(ctx, "key-1", "hash-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(ctx, "key-1", 201, []byte(`{"hold_id":"h1"}`)); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	rec, err := store.Begin(ctx, "key-1", "hash-a")
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if rec == nil {
		t.Fatal("a completed key did not return its record; the work would be redone")
	}
	if string(rec.Response) != `{"hold_id":"h1"}` {
		t.Errorf("response = %q", rec.Response)
	}
	if rec.StatusCode != 201 {
		t.Errorf("status code = %d, want 201", rec.StatusCode)
	}
}

// A retry arriving while the original is still running must NOT proceed --
// proceeding is exactly how two tickets get issued for one request.
func TestInFlightRequestIsRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Begin(ctx, "key-1", "hash-a"); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Begin(ctx, "key-1", "hash-a"); !errors.Is(err, ErrInFlight) {
		t.Fatalf("err = %v, want ErrInFlight", err)
	}
}

// THE test. Many concurrent requests with one key: exactly one may proceed.
// This is the edge-layer twin of the no-double-sell test in inventory.
func TestConcurrentClaimsProduceExactlyOneWinner(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	const contenders = 25
	start := make(chan struct{})
	var wg sync.WaitGroup
	var winners, inFlight, other atomic.Int64

	wg.Add(contenders)
	for range contenders {
		go func() {
			defer wg.Done()
			<-start
			rec, err := store.Begin(ctx, "hot-key", "hash-a")
			switch {
			case err == nil && rec == nil:
				winners.Add(1)
			case errors.Is(err, ErrInFlight):
				inFlight.Add(1)
			default:
				other.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if w := winners.Load(); w != 1 {
		t.Fatalf("REQUEST PROCESSED TWICE: %d winners for one key (in-flight=%d, other=%d)",
			w, inFlight.Load(), other.Load())
	}
	if inFlight.Load() != contenders-1 {
		t.Errorf("in-flight rejections = %d, want %d", inFlight.Load(), contenders-1)
	}
}

// Reusing a key with a different body is a client bug. Replaying the first
// response would answer a question that was never asked.
func TestKeyReuseWithDifferentBodyIsRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Begin(ctx, "key-1", "hash-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(ctx, "key-1", 201, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Begin(ctx, "key-1", "hash-DIFFERENT"); !errors.Is(err, ErrKeyMismatch) {
		t.Fatalf("err = %v, want ErrKeyMismatch", err)
	}
}

// A failed request should free its key immediately, so the client can retry
// rather than waiting out the TTL.
func TestAbandonReleasesAnInFlightClaim(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Begin(ctx, "key-1", "hash-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.Abandon(ctx, "key-1"); err != nil {
		t.Fatalf("Abandon: %v", err)
	}

	rec, err := store.Begin(ctx, "key-1", "hash-a")
	if err != nil {
		t.Fatalf("Begin after Abandon: %v", err)
	}
	if rec != nil {
		t.Error("the abandoned claim was not released")
	}
}

// Abandon must never delete a completed record: replaying it is the entire
// point of storing it.
func TestAbandonLeavesCompletedRecords(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Begin(ctx, "key-1", "hash-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.Complete(ctx, "key-1", 200, []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if err := store.Abandon(ctx, "key-1"); err != nil {
		t.Fatalf("Abandon on a completed record errored: %v", err)
	}

	rec, err := store.Get(ctx, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	if rec == nil || rec.Status != StatusComplete {
		t.Error("Abandon destroyed a completed record; its retry would redo the work")
	}
}

func TestGetOnMissingKey(t *testing.T) {
	store := newTestStore(t)

	rec, err := store.Get(context.Background(), "never-existed")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if rec != nil {
		t.Errorf("got %+v, want nil", rec)
	}
}

func TestStoreValidatesOptions(t *testing.T) {
	if _, err := NewStore(Options{Table: "t"}); err == nil {
		t.Error("NewStore accepted a nil client")
	}
	if _, err := NewStore(Options{Client: dynamodb.New(dynamodb.Options{})}); err == nil {
		t.Error("NewStore accepted an empty table name")
	}
}
