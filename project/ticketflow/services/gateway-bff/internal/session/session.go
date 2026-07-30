// Package session stores browser sessions in Redis.
//
// Redis rather than in-process memory because the BFF runs on N replicas behind
// a load balancer with no affinity: a buyer whose second request lands on a
// different pod must still be the same buyer. It also means a blue/green flip
// does not log everyone out.
//
// Sessions live in a different logical Redis database from the cache (DB 2 vs
// DB 0). The cache database can safely run an evicting maxmemory policy;
// evicting a session would sign users out at random.
package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a session id is unknown or expired.
var ErrNotFound = errors.New("session: not found")

// Session is the server-side state behind a session cookie.
type Session struct {
	ID     string    `json:"id"`
	UserID string    `json:"user_id"`
	Create time.Time `json:"created_at"`
}

// Store persists sessions.
type Store struct {
	client redis.UniversalClient
	ttl    time.Duration
}

type Options struct {
	Client redis.UniversalClient
	// TTL is the idle lifetime. Refreshed on each successful Get, so an active
	// buyer is not logged out mid-checkout.
	TTL time.Duration
}

func NewStore(opts Options) *Store {
	if opts.TTL <= 0 {
		opts.TTL = 24 * time.Hour
	}
	return &Store{client: opts.Client, ttl: opts.TTL}
}

func key(id string) string { return "session:" + id }

// Create issues a new session with a cryptographically random id.
//
// An empty userID yields an anonymous identity derived from the session id.
// Deriving it here, before the session is persisted, is deliberate: a caller
// that set the field afterwards would hold a correct copy in memory while
// Redis kept one with an empty UserID, so the identity would silently vanish
// on the next request. A stored session always has a usable UserID.
func (s *Store) Create(ctx context.Context, userID string) (Session, error) {
	id, err := newID()
	if err != nil {
		return Session{}, err
	}
	if userID == "" {
		userID = anonUserID(id)
	}

	sess := Session{ID: id, UserID: userID, Create: time.Now().UTC()}
	raw, err := json.Marshal(sess)
	if err != nil {
		return Session{}, fmt.Errorf("session: marshalling: %w", err)
	}
	if err := s.client.Set(ctx, key(id), raw, s.ttl).Err(); err != nil {
		return Session{}, fmt.Errorf("session: storing: %w", err)
	}
	return sess, nil
}

// Get loads a session and extends its TTL, so activity keeps it alive.
func (s *Store) Get(ctx context.Context, id string) (Session, error) {
	if id == "" {
		return Session{}, ErrNotFound
	}

	raw, err := s.client.Get(ctx, key(id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("session: loading: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		// A session written by an older format is unusable; treat it as absent
		// so the caller mints a fresh one instead of erroring the request.
		return Session{}, ErrNotFound
	}

	// Sliding expiry. Failure here is not fatal -- the session is still valid,
	// it just expires on its original schedule.
	_ = s.client.Expire(ctx, key(id), s.ttl).Err()

	return sess, nil
}

// Destroy removes a session (logout).
func (s *Store) Destroy(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	if err := s.client.Del(ctx, key(id)).Err(); err != nil {
		return fmt.Errorf("session: destroying: %w", err)
	}
	return nil
}

// anonUserID derives a stable pseudonymous identity from a session id, so an
// unauthenticated browser can still own a seat hold. Phase 4 replaces this with
// the JWT subject supplied by the API Gateway authorizer.
//
// A prefix of the session id is used rather than the whole thing so the user id
// -- which appears in logs and in the seat_hold table -- is not itself a usable
// bearer token.
func anonUserID(sessionID string) string {
	const idLen = 12
	if len(sessionID) < idLen {
		return "anon-" + sessionID
	}
	return "anon-" + sessionID[:idLen]
}

// newID returns 256 bits of entropy, URL-safe. Session ids are bearer tokens,
// so they must not be guessable -- a counter or UUIDv1 would be.
func newID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("session: generating id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
