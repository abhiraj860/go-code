package ratelimiter

type Limiter interface {
	Allow(key string) RateLimitResult
}