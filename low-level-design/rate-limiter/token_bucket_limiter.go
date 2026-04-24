package ratelimiter

import (
	"math"
	"time"
)

type TokenBucketLimiter struct {
	capacity int
	refillRatePerSecond int
	buckets map[string]*tokenBucket
}

type tokenBucket struct {
	tokens float64
	lastRefillTime int64
}

func NewTokenBucketLimiter(capacity int, refillRatePerSecond int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity: capacity,
		refillRatePerSecond: refillRatePerSecond,
		buckets: make(map[string] *tokenBucket),
	}
}

func(l *TokenBucketLimiter) Allow(key string) RateLimitResult {
	bucket := l.getOrCreateBucket(key)	
	now := time.Now().UnixMilli()
	elapsed := now - bucket.lastRefillTime
	tokensToAdd := (float64(elapsed) * float64(l.refillRatePerSecond)) / 1000
	bucket.tokens = math.Min(float64(l.capacity), bucket.tokens + tokensToAdd)
	bucket.lastRefillTime = now
	if bucket.tokens >= 1 {
		bucket.tokens -= 1
		remaining := int(math.Floor(bucket.tokens))
		return RateLimitResult{Allowed: true, Remaining: remaining}
	}

	tokensNeeded := 1 - bucket.tokens
	retryAfter := int64(math.Ceil((tokensNeeded * 1000) / float64(l.refillRatePerSecond)))
	return RateLimitResult{Allowed:false, Remaining: 0, RetryAfterMs: &retryAfter}
}



func(l *TokenBucketLimiter) getOrCreateBucket(key string) *tokenBucket {
	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &tokenBucket{
			tokens : float64(l.capacity),
			lastRefillTime: time.Now().UnixMilli(),
		}
		l.buckets[key] = bucket
	}
	return bucket
}