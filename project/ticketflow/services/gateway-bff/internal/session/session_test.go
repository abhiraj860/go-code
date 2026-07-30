package session

import (
	"context"
	"errors"
	"github.com/abhiraj860/ticketflow/pkg/testsupport"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// newTestStore connects to Redis DB 14, reserved for session tests, and flushes
// it. Skips when no stack is running.
func newTestStore(t *testing.T, ttl time.Duration) *Store {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr:        "localhost:6379",
		DB:          14,
		DialTimeout: 500 * time.Millisecond,
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
	return NewStore(Options{Client: client, TTL: ttl})
}

// Regression test for a real bug: the middleware used to assign the anonymous
// user id *after* Create persisted the session, so the identity existed only in
// memory. The first request worked and every subsequent one loaded a session
// with an empty UserID and was refused permission to hold seats.
//
// The invariant now: a session read back from Redis always has a usable UserID.
func TestUserIDSurvivesRoundTrip(t *testing.T) {
	store := newTestStore(t, time.Minute)
	ctx := context.Background()

	created, err := store.Create(ctx, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.UserID == "" {
		t.Fatal("Create returned a session with an empty UserID")
	}

	loaded, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if loaded.UserID == "" {
		t.Fatal("session loaded from Redis has an empty UserID -- identity did not persist")
	}
	if loaded.UserID != created.UserID {
		t.Errorf("UserID = %q after round trip, want %q", loaded.UserID, created.UserID)
	}
}

func TestCreateHonoursExplicitUserID(t *testing.T) {
	store := newTestStore(t, time.Minute)
	ctx := context.Background()

	sess, err := store.Create(ctx, "user-42")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sess.UserID != "user-42" {
		t.Errorf("UserID = %q, want user-42", sess.UserID)
	}

	loaded, _ := store.Get(ctx, sess.ID)
	if loaded.UserID != "user-42" {
		t.Errorf("UserID = %q after round trip, want user-42", loaded.UserID)
	}
}

func TestAnonymousIDIsNotTheFullSessionID(t *testing.T) {
	store := newTestStore(t, time.Minute)

	sess, err := store.Create(context.Background(), "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if !strings.HasPrefix(sess.UserID, "anon-") {
		t.Errorf("UserID = %q, want an anon- prefix", sess.UserID)
	}
	// The user id lands in logs and in seat_hold rows; if it were the whole
	// session id it would be a leaked bearer token.
	if strings.Contains(sess.UserID, sess.ID) {
		t.Error("anonymous user id contains the full session id -- that leaks a bearer token")
	}
}

func TestSessionIDsAreUnpredictable(t *testing.T) {
	store := newTestStore(t, time.Minute)
	ctx := context.Background()

	seen := make(map[string]struct{})
	for range 100 {
		sess, err := store.Create(ctx, "")
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if _, dup := seen[sess.ID]; dup {
			t.Fatal("duplicate session id generated")
		}
		// 32 random bytes, RawURLEncoding -> 43 characters.
		if len(sess.ID) < 40 {
			t.Errorf("session id length %d is too short to be unguessable", len(sess.ID))
		}
		seen[sess.ID] = struct{}{}
	}
}

func TestGetUnknownSession(t *testing.T) {
	store := newTestStore(t, time.Minute)

	if _, err := store.Get(context.Background(), "no-such-session"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	if _, err := store.Get(context.Background(), ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("empty id: err = %v, want ErrNotFound", err)
	}
}

func TestDestroy(t *testing.T) {
	store := newTestStore(t, time.Minute)
	ctx := context.Background()

	sess, _ := store.Create(ctx, "user-1")
	if err := store.Destroy(ctx, sess.ID); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if _, err := store.Get(ctx, sess.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("session survived Destroy: %v", err)
	}
	// Destroying an absent session is not an error.
	if err := store.Destroy(ctx, "never-existed"); err != nil {
		t.Errorf("Destroy on absent session errored: %v", err)
	}
}

func TestExpiry(t *testing.T) {
	store := newTestStore(t, 150*time.Millisecond)
	ctx := context.Background()

	sess, _ := store.Create(ctx, "user-1")
	time.Sleep(250 * time.Millisecond)

	if _, err := store.Get(ctx, sess.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("session survived its TTL: %v", err)
	}
}

// Activity must extend the session, so a buyer is not logged out mid-checkout.
func TestGetSlidesExpiry(t *testing.T) {
	store := newTestStore(t, 400*time.Millisecond)
	ctx := context.Background()

	sess, _ := store.Create(ctx, "user-1")

	// Three reads spaced under the TTL; each should push the deadline out.
	for range 3 {
		time.Sleep(200 * time.Millisecond)
		if _, err := store.Get(ctx, sess.ID); err != nil {
			t.Fatalf("session expired despite activity: %v", err)
		}
	}
}

// A session written in an older format must be treated as absent so the caller
// mints a fresh one, rather than failing the request.
func TestCorruptSessionTreatedAsMissing(t *testing.T) {
	store := newTestStore(t, time.Minute)
	ctx := context.Background()

	if err := store.client.Set(ctx, key("corrupt"), []byte("{not json"), time.Minute).Err(); err != nil {
		t.Fatalf("planting corrupt session: %v", err)
	}

	if _, err := store.Get(ctx, "corrupt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
