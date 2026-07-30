package lock

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/abhiraj860/ticketflow/pkg/testsupport"
)

// newTestLocker uses DB 9, reserved for lock tests, and flushes it.
//
// These tests deliberately run against a REAL Redis rather than a fake. A fake
// cannot reproduce the NOSCRIPT behaviour that made the fast path silently
// dead: redis.Script falls back from EVALSHA to EVAL automatically, but not
// inside a pipeline, so against a cold Redis every call failed. Unit tests with
// a stub locker passed the whole time.
func newTestLocker(t *testing.T) (*SeatLocker, *redis.Client) {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", DB: 9, DialTimeout: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		testsupport.SkipOrFail(t, "redis not reachable (run `make up`): %v", err)
	}
	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flushing test db: %v", err)
	}

	t.Cleanup(func() { _ = client.Close() })
	return NewSeatLocker(client), client
}

// The regression test for the bug. A Redis with no cached scripts -- a fresh
// container, a restart, a SCRIPT FLUSH -- must still work.
func TestAcquireWorksAgainstAColdRedis(t *testing.T) {
	locker, client := newTestLocker(t)
	ctx := context.Background()

	// Exactly the state a freshly started Redis is in.
	if err := client.ScriptFlush(ctx).Err(); err != nil {
		t.Fatalf("flushing script cache: %v", err)
	}

	acquired, rejected, err := locker.Acquire(ctx, "e1", []string{"S-1"}, "owner-1", time.Minute)
	if err != nil {
		t.Fatalf("Acquire against a cold Redis failed: %v", err)
	}
	if len(acquired) != 1 || acquired[0] != "S-1" {
		t.Fatalf("acquired = %v, want [S-1]", acquired)
	}
	if len(rejected) != 0 {
		t.Errorf("rejected = %v, want none", rejected)
	}

	// The whole point: the lock must actually exist in Redis. Asserting only on
	// the return value would have passed even while the fast path did nothing.
	keys := client.Keys(ctx, "seatlock:*").Val()
	if len(keys) != 1 {
		t.Fatalf("redis holds %v, want exactly one lock key -- the fast path is not actually locking", keys)
	}
}

// SCRIPT FLUSH mid-flight must self-heal rather than disabling the fast path
// until the next deploy.
func TestRecoversFromScriptCacheLoss(t *testing.T) {
	locker, client := newTestLocker(t)
	ctx := context.Background()

	if _, _, err := locker.Acquire(ctx, "e1", []string{"S-1"}, "owner-1", time.Minute); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	// Simulate a Redis restart or an operator running SCRIPT FLUSH.
	if err := client.ScriptFlush(ctx).Err(); err != nil {
		t.Fatalf("script flush: %v", err)
	}

	acquired, _, err := locker.Acquire(ctx, "e1", []string{"S-2"}, "owner-2", time.Minute)
	if err != nil {
		t.Fatalf("acquire after script cache loss: %v", err)
	}
	if len(acquired) != 1 {
		t.Errorf("acquired = %v, want [S-2]", acquired)
	}
}

func TestSecondOwnerIsRejected(t *testing.T) {
	locker, _ := newTestLocker(t)
	ctx := context.Background()

	if _, _, err := locker.Acquire(ctx, "e1", []string{"S-1"}, "owner-1", time.Minute); err != nil {
		t.Fatal(err)
	}

	acquired, rejected, err := locker.Acquire(ctx, "e1", []string{"S-1"}, "owner-2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(acquired) != 0 {
		t.Errorf("acquired = %v, want none", acquired)
	}
	if len(rejected) != 1 || rejected[0] != "S-1" {
		t.Errorf("rejected = %v, want [S-1]", rejected)
	}
}

// An idempotent retry must re-acquire its own lock, not be blocked by it.
func TestSameOwnerReacquires(t *testing.T) {
	locker, _ := newTestLocker(t)
	ctx := context.Background()

	for i := range 3 {
		acquired, _, err := locker.Acquire(ctx, "e1", []string{"S-1"}, "owner-1", time.Minute)
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		if len(acquired) != 1 {
			t.Fatalf("attempt %d: acquired = %v, want [S-1] -- a retry was blocked by its own lock", i, acquired)
		}
	}
}

func TestPartialAcquire(t *testing.T) {
	locker, _ := newTestLocker(t)
	ctx := context.Background()

	if _, _, err := locker.Acquire(ctx, "e1", []string{"S-2"}, "owner-1", time.Minute); err != nil {
		t.Fatal(err)
	}

	acquired, rejected, err := locker.Acquire(ctx, "e1", []string{"S-1", "S-2", "S-3"}, "owner-2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(acquired) != 2 {
		t.Errorf("acquired = %v, want S-1 and S-3", acquired)
	}
	if len(rejected) != 1 || rejected[0] != "S-2" {
		t.Errorf("rejected = %v, want [S-2]", rejected)
	}
}

func TestLocksExpire(t *testing.T) {
	locker, _ := newTestLocker(t)
	ctx := context.Background()

	if _, _, err := locker.Acquire(ctx, "e1", []string{"S-1"}, "owner-1", 150*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)

	acquired, _, err := locker.Acquire(ctx, "e1", []string{"S-1"}, "owner-2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if len(acquired) != 1 {
		t.Errorf("acquired = %v; a lapsed lock must be claimable", acquired)
	}
}

// A slow request whose lock already expired and was re-acquired by someone else
// must not delete the new owner's lock.
func TestReleaseOnlyAffectsOwnLocks(t *testing.T) {
	locker, client := newTestLocker(t)
	ctx := context.Background()

	if _, _, err := locker.Acquire(ctx, "e1", []string{"S-1"}, "owner-1", time.Minute); err != nil {
		t.Fatal(err)
	}

	if err := locker.Release(ctx, "e1", []string{"S-1"}, "someone-else"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if n := len(client.Keys(ctx, "seatlock:*").Val()); n != 1 {
		t.Error("a non-owner deleted somebody else's lock")
	}

	if err := locker.Release(ctx, "e1", []string{"S-1"}, "owner-1"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if n := len(client.Keys(ctx, "seatlock:*").Val()); n != 0 {
		t.Error("the owner's own release did not delete the lock")
	}
}

func TestEmptyInputsAreNoOps(t *testing.T) {
	locker, _ := newTestLocker(t)
	ctx := context.Background()

	if a, r, err := locker.Acquire(ctx, "e1", nil, "o", time.Minute); err != nil || a != nil || r != nil {
		t.Errorf("Acquire(nil) = %v, %v, %v", a, r, err)
	}
	if err := locker.Release(ctx, "e1", nil, "o"); err != nil {
		t.Errorf("Release(nil) = %v", err)
	}
}

func TestAcquireRejectsNonPositiveTTL(t *testing.T) {
	locker, _ := newTestLocker(t)

	if _, _, err := locker.Acquire(context.Background(), "e1", []string{"S-1"}, "o", 0); err == nil {
		t.Error("Acquire accepted a zero TTL")
	}
}
