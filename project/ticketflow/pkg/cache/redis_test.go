package cache

import (
	"context"
	"errors"
	"github.com/abhiraj860/ticketflow/pkg/testsupport"
	"os"
	"testing"
	"time"
)

// redisAddr returns the address to test against. Skips the test when Redis is
// not reachable, so `go test ./...` stays green on a machine with no stack up
// while still exercising the real client when one is running.
func redisAddr(t *testing.T) string {
	t.Helper()

	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	store, err := NewRedisStore(ctx, RedisOptions{Addr: addr, DB: 15})
	if err != nil {
		testsupport.SkipOrFail(t, "redis not reachable at %s (run `make up`): %v", addr, err)
	}
	_ = store.Close()
	return addr
}

// newTestRedis connects to DB 15, reserved for tests, and flushes it so runs
// cannot contaminate each other.
func newTestRedis(t *testing.T) *RedisStore {
	t.Helper()
	addr := redisAddr(t)

	ctx := context.Background()
	store, err := NewRedisStore(ctx, RedisOptions{Addr: addr, DB: 15})
	if err != nil {
		t.Fatalf("connecting to redis: %v", err)
	}
	if err := store.Client().FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flushing test db: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestRedisStoreRoundTrip(t *testing.T) {
	store := newTestRedis(t)
	ctx := context.Background()

	if _, found, err := store.Get(ctx, "absent"); err != nil {
		t.Fatalf("Get on absent key errored: %v", err)
	} else if found {
		t.Error("absent key reported as found")
	}

	if err := store.Set(ctx, "k1", []byte("hello"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, found, err := store.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("key written by Set was not found")
	}
	if string(got) != "hello" {
		t.Errorf("value = %q, want hello", got)
	}
}

func TestRedisStoreTTLIsApplied(t *testing.T) {
	store := newTestRedis(t)
	ctx := context.Background()

	if err := store.Set(ctx, "expiring", []byte("v"), 150*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := store.Get(ctx, "expiring"); !found {
		t.Fatal("key missing immediately after Set")
	}

	time.Sleep(250 * time.Millisecond)

	if _, found, err := store.Get(ctx, "expiring"); err != nil {
		t.Fatalf("Get after expiry errored: %v", err)
	} else if found {
		t.Error("key survived its TTL")
	}
}

// A cache entry with no expiry is a memory leak, so this is rejected outright
// rather than silently stored forever.
func TestRedisStoreRejectsNonPositiveTTL(t *testing.T) {
	store := newTestRedis(t)
	ctx := context.Background()

	for _, ttl := range []time.Duration{0, -time.Second} {
		if err := store.Set(ctx, "k", []byte("v"), ttl); err == nil {
			t.Errorf("Set with ttl=%v was accepted; want an error", ttl)
		}
	}
}

func TestRedisStoreDel(t *testing.T) {
	store := newTestRedis(t)
	ctx := context.Background()

	_ = store.Set(ctx, "a", []byte("1"), time.Minute)
	_ = store.Set(ctx, "b", []byte("2"), time.Minute)

	if err := store.Del(ctx, "a", "b"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	for _, k := range []string{"a", "b"} {
		if _, found, _ := store.Get(ctx, k); found {
			t.Errorf("key %q survived Del", k)
		}
	}

	// Deleting nothing, and deleting an absent key, are both fine.
	if err := store.Del(ctx); err != nil {
		t.Errorf("Del with no keys errored: %v", err)
	}
	if err := store.Del(ctx, "never-existed"); err != nil {
		t.Errorf("Del on an absent key errored: %v", err)
	}
}

func TestNewRedisStoreRequiresAddr(t *testing.T) {
	if _, err := NewRedisStore(context.Background(), RedisOptions{}); err == nil {
		t.Error("NewRedisStore accepted an empty Addr")
	}
}

func TestNewRedisStoreFailsFastOnBadAddr(t *testing.T) {
	// Port 1 is reserved and never listening; connecting must fail at startup
	// rather than on the first cache read.
	_, err := NewRedisStore(context.Background(), RedisOptions{
		Addr:        "127.0.0.1:1",
		DialTimeout: 200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("NewRedisStore succeeded against a dead address")
	}
}

// The end-to-end shape the services actually use: Loader over a real Redis L2.
func TestLoaderOverRealRedis(t *testing.T) {
	store := newTestRedis(t)

	var fetches int
	l := NewLoader(LoaderOptions[string, event]{
		L1:          New[string, event](Options{MaxEntries: 10, TTL: time.Minute}),
		L2:          store,
		Codec:       jsonCodec[event]{},
		KeyFn:       func(k string) string { return "event:" + k + ":v1" },
		L2TTL:       time.Minute,
		NegativeTTL: 5 * time.Second,
		Fetch: func(_ context.Context, k string) (event, error) {
			fetches++
			if k == "ghost" {
				return event{}, ErrNotFound
			}
			return event{ID: k, Title: "Live From Redis"}, nil
		},
	})
	ctx := context.Background()

	got, err := l.Get(ctx, "e1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Live From Redis" {
		t.Errorf("Title = %q", got.Title)
	}

	// Drop L1 to force the next read through Redis.
	l.opts.L1.Purge()
	if _, err := l.Get(ctx, "e1"); err != nil {
		t.Fatal(err)
	}
	if fetches != 1 {
		t.Errorf("fetches = %d, want 1 -- Redis should have served the second read", fetches)
	}
	if l.Stats().L2Hits != 1 {
		t.Errorf("L2Hits = %d, want 1", l.Stats().L2Hits)
	}

	// Negative caching must survive a round-trip through real Redis too.
	if _, err := l.Get(ctx, "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	l.opts.L1.Purge()
	if _, err := l.Get(ctx, "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second ghost read: err = %v, want ErrNotFound", err)
	}
	if fetches != 2 {
		t.Errorf("fetches = %d, want 2 -- the negative marker should have absorbed the retry", fetches)
	}
}
