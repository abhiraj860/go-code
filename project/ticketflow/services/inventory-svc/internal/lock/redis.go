// Package lock provides an advisory seat lock in Redis.
//
// READ THIS BEFORE CHANGING ANYTHING HERE.
//
// This lock is an optimisation and nothing else. Postgres remains the sole
// authority on who owns a seat. The invariant that makes the whole design safe:
//
//	Redis may only ever cause a seat to be REJECTED, never GRANTED.
//
// Every failure mode therefore degrades in the safe direction. If Redis is
// down, unreachable, slow, or returns nonsense, the worst outcome is that a
// request proceeds to Postgres that could have been rejected earlier -- which
// is exactly what happens today with no Redis at all. A stale lock can cause a
// spurious rejection of an available seat; annoying, momentary (the TTL bounds
// it), and never a correctness failure.
//
// What it buys: during a drop, tens of thousands of requests that are going to
// lose can be turned away in ~0.2ms without ever opening a Postgres
// transaction, so the database only does work for requests that can win.
package lock

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// acquireScript claims a seat key, or renews it when the caller already owns it.
//
// The owner check is what makes an idempotent retry work. Without it, a client
// retrying a timed-out request would be blocked by its own earlier lock and see
// a spurious rejection for seats it actually holds.
//
// Runs as a script so the GET and SET are atomic; a read-then-write from Go
// would race with another instance between the two round-trips.
var acquireScript = redis.NewScript(`
local current = redis.call('GET', KEYS[1])
if current == false then
	redis.call('SET', KEYS[1], ARGV[1], 'PX', ARGV[2])
	return 1
elseif current == ARGV[1] then
	redis.call('PEXPIRE', KEYS[1], ARGV[2])
	return 1
else
	return 0
end
`)

// releaseScript deletes a key only when the caller owns it, so a slow request
// whose lock already expired and was re-acquired by someone else cannot delete
// the new owner's lock.
var releaseScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
	return redis.call('DEL', KEYS[1])
else
	return 0
end
`)

// SeatLocker is the advisory fast path.
type SeatLocker struct {
	client redis.UniversalClient
}

func NewSeatLocker(client redis.UniversalClient) *SeatLocker {
	return &SeatLocker{client: client}
}

// loadScripts caches both scripts in Redis so subsequent EVALSHA calls resolve.
//
// This is REQUIRED, not an optimisation, and the reason is subtle: redis.Script
// normally issues EVALSHA and transparently falls back to EVAL when Redis
// replies NOSCRIPT. Inside a pipeline it cannot -- every command is written
// before any reply is read, so there is no opportunity to react to NOSCRIPT and
// retry. Against a Redis that has never seen these scripts (a fresh container,
// a restart, a SCRIPT FLUSH) every pipelined call therefore fails.
//
// That failure is safe -- callers fall through to Postgres -- which is exactly
// what makes it dangerous: the fast path silently does nothing at all, and
// nothing looks broken. It was found by checking that Redis actually contained
// lock keys, not by any test.
func (l *SeatLocker) loadScripts(ctx context.Context) error {
	if err := acquireScript.Load(ctx, l.client).Err(); err != nil {
		return fmt.Errorf("lock: loading acquire script: %w", err)
	}
	if err := releaseScript.Load(ctx, l.client).Err(); err != nil {
		return fmt.Errorf("lock: loading release script: %w", err)
	}
	return nil
}

// isNoScript reports whether an error is Redis's NOSCRIPT reply, meaning the
// script cache was lost (restart or SCRIPT FLUSH) and needs repopulating.
func isNoScript(err error) bool {
	return err != nil && strings.Contains(err.Error(), "NOSCRIPT")
}

// key namespaces locks per event. Locks live in a different logical Redis
// database from the cache, because the cache may run an evicting maxmemory
// policy and evicting a lock would silently disable the fast path.
func key(eventID, seatID string) string {
	return "seatlock:" + eventID + ":" + seatID
}

// Acquire attempts to claim seats, returning those won and those already held
// by somebody else.
//
// owner should identify the logical attempt, not the process -- pass
// userID + idempotency key so a retry of the same request re-acquires its own
// locks rather than colliding with them.
//
// An error means Redis is unusable. Callers MUST treat that as "no fast path
// available" and proceed to Postgres with the full seat list, never as a
// rejection.
func (l *SeatLocker) Acquire(ctx context.Context, eventID string, seatIDs []string, owner string, ttl time.Duration) (acquired, rejected []string, err error) {
	if len(seatIDs) == 0 {
		return nil, nil, nil
	}
	if ttl <= 0 {
		return nil, nil, errors.New("lock: ttl must be positive")
	}

	acquired, rejected, err = l.acquireOnce(ctx, eventID, seatIDs, owner, ttl)
	if isNoScript(err) {
		// Redis lost its script cache. Repopulate and retry once; a second
		// NOSCRIPT means something is genuinely wrong and the caller should
		// fall through to Postgres.
		if loadErr := l.loadScripts(ctx); loadErr != nil {
			return nil, nil, loadErr
		}
		return l.acquireOnce(ctx, eventID, seatIDs, owner, ttl)
	}
	return acquired, rejected, err
}

func (l *SeatLocker) acquireOnce(ctx context.Context, eventID string, seatIDs []string, owner string, ttl time.Duration) (acquired, rejected []string, err error) {
	ttlMillis := ttl.Milliseconds()

	// Pipelined so N seats cost one round-trip rather than N.
	pipe := l.client.Pipeline()
	cmds := make([]*redis.Cmd, len(seatIDs))
	for i, seatID := range seatIDs {
		cmds[i] = acquireScript.Run(ctx, pipe, []string{key(eventID, seatID)}, owner, ttlMillis)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		// Returned unwrapped enough for isNoScript to inspect, so Acquire can
		// repopulate the script cache and retry.
		return nil, nil, fmt.Errorf("lock: acquiring seat locks: %w", err)
	}

	acquired = make([]string, 0, len(seatIDs))
	for i, cmd := range cmds {
		got, err := cmd.Int64()
		if err != nil {
			// One malformed reply must not fail the batch. Treat this seat as
			// unlocked and let Postgres decide -- the safe direction.
			acquired = append(acquired, seatIDs[i])
			continue
		}
		if got == 1 {
			acquired = append(acquired, seatIDs[i])
		} else {
			rejected = append(rejected, seatIDs[i])
		}
	}
	return acquired, rejected, nil
}

// Release drops locks the caller owns. Best-effort: a failure here only means
// seats stay locked until their TTL lapses, which costs a brief spurious
// rejection rather than correctness.
func (l *SeatLocker) Release(ctx context.Context, eventID string, seatIDs []string, owner string) error {
	if len(seatIDs) == 0 {
		return nil
	}

	if err := l.releaseOnce(ctx, eventID, seatIDs, owner); err != nil {
		if !isNoScript(err) {
			return err
		}
		if loadErr := l.loadScripts(ctx); loadErr != nil {
			return loadErr
		}
		return l.releaseOnce(ctx, eventID, seatIDs, owner)
	}
	return nil
}

func (l *SeatLocker) releaseOnce(ctx context.Context, eventID string, seatIDs []string, owner string) error {
	pipe := l.client.Pipeline()
	for _, seatID := range seatIDs {
		releaseScript.Run(ctx, pipe, []string{key(eventID, seatID)}, owner)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("lock: releasing seat locks: %w", err)
	}
	return nil
}
