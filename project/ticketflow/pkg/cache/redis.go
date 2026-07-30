package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStore adapts a go-redis client to the Store interface, providing the L2
// cache tier.
//
// Note on database separation: cache entries are evictable under memory
// pressure, but seat-hold locks are not -- evicting a lock would hand the same
// seat to two buyers. They therefore live in different logical Redis databases
// (see RedisOptions.DB), so a future maxmemory-policy of allkeys-lru can be
// applied to the cache database alone.
type RedisStore struct {
	client redis.UniversalClient
}

// RedisOptions configures the connection.
type RedisOptions struct {
	// Addr is host:port. Required.
	Addr string
	// Password is optional for local development.
	Password string
	// DB selects the logical database. Convention in this project:
	//   0 = cache (evictable)
	//   1 = seat holds and locks (must never be evicted)
	//   2 = sessions
	DB int
	// PoolSize defaults to 10x GOMAXPROCS in go-redis. During a drop the BFF
	// is the bottleneck, so this is worth raising there specifically.
	PoolSize int
	// DialTimeout / ReadTimeout keep a slow Redis from becoming a slow API.
	// The cache is an optimisation; if it cannot answer quickly, falling
	// through to origin is better than making the caller wait.
	DialTimeout time.Duration
	ReadTimeout time.Duration
}

// NewRedisStore connects and verifies reachability with a PING, so a
// misconfigured address fails at startup rather than on first request.
func NewRedisStore(ctx context.Context, opts RedisOptions) (*RedisStore, error) {
	if opts.Addr == "" {
		return nil, errors.New("cache: RedisOptions.Addr is required")
	}
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 2 * time.Second
	}
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 500 * time.Millisecond
	}

	client := redis.NewClient(&redis.Options{
		Addr:         opts.Addr,
		Password:     opts.Password,
		DB:           opts.DB,
		PoolSize:     opts.PoolSize,
		DialTimeout:  opts.DialTimeout,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.ReadTimeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("cache: connecting to redis at %s: %w", opts.Addr, err)
	}

	return &RedisStore{client: client}, nil
}

// NewRedisStoreFromClient wraps an existing client. Used when a service already
// holds a Redis connection for locks or sessions and wants to share the pool.
func NewRedisStoreFromClient(client redis.UniversalClient) *RedisStore {
	return &RedisStore{client: client}
}

// Get returns the raw bytes for key. A missing key is (nil, false, nil), never
// an error -- a cache miss is an ordinary outcome, not a failure.
func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("cache: redis get %q: %w", key, err)
	}
	return v, true, nil
}

// Set writes value with a TTL. A non-positive ttl is rejected rather than
// silently storing the key forever: an entry that never expires in a cache is
// a memory leak waiting for a deploy to notice it.
func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("cache: refusing to set %q with non-positive TTL %v", key, ttl)
	}
	if err := s.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("cache: redis set %q: %w", key, err)
	}
	return nil
}

// Del removes keys. Deleting an absent key is not an error.
func (s *RedisStore) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := s.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache: redis del: %w", err)
	}
	return nil
}

// Client exposes the underlying client for callers needing operations outside
// the Store interface -- seat-hold SET NX PX, rate-limit counters, pub/sub.
func (s *RedisStore) Client() redis.UniversalClient {
	return s.client
}

// Close releases the connection pool.
func (s *RedisStore) Close() error {
	return s.client.Close()
}

// compile-time assertion that RedisStore satisfies Store.
var _ Store = (*RedisStore)(nil)
