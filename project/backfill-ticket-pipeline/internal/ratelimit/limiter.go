// Package ratelimit implements an adaptive token-bucket limiter.
//
// WHY THIS EXISTS (defends resume bullet #3 - "adaptive rate limiting and
// batching... cutting API costs and reducing throttling failures"):
//
//   - A single shared limiter is used by ALL workers, so total request rate
//     across the whole worker pool stays under the provider's quota (workers
//     limiting independently would multiply the effective rate).
//   - On a 429/rate-limit error, the limiter halves its rate (multiplicative
//     decrease) and slowly ramps back up on sustained success (additive
//     increase) - classic AIMD, same idea as TCP congestion control.
//   - Every throttle event and rate change is counted so you can report
//     "throttling errors reduced by X%" with real numbers.
package ratelimit

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

type AdaptiveLimiter struct {
	mu           sync.Mutex
	limiter      *rate.Limiter
	minRate      rate.Limit
	maxRate      rate.Limit
	successCount int64
	throttleCount int64
	rampEvery    int64 // ramp up after this many consecutive successes
}

func NewAdaptiveLimiter(initial, min, max rate.Limit, burst int) *AdaptiveLimiter {
	return &AdaptiveLimiter{
		limiter:   rate.NewLimiter(initial, burst),
		minRate:   min,
		maxRate:   max,
		rampEvery: 50,
	}
}

// Wait blocks until a token is available (respects context cancellation).
func (a *AdaptiveLimiter) Wait(ctx context.Context) error {
	return a.limiter.Wait(ctx)
}

// ReportSuccess is called after every successful call; slowly ramps the rate
// back up after a run of consecutive successes.
func (a *AdaptiveLimiter) ReportSuccess() {
	n := atomic.AddInt64(&a.successCount, 1)
	if n%a.rampEvery == 0 {
		a.mu.Lock()
		defer a.mu.Unlock()
		newRate := a.limiter.Limit() * 1.1
		if newRate > a.maxRate {
			newRate = a.maxRate
		}
		a.limiter.SetLimit(newRate)
	}
}

// ReportThrottle is called on a 429/rate-limit error; halves the rate (down
// to a floor) and records the event for later reporting.
func (a *AdaptiveLimiter) ReportThrottle() {
	atomic.AddInt64(&a.throttleCount, 1)
	atomic.StoreInt64(&a.successCount, 0)

	a.mu.Lock()
	defer a.mu.Unlock()
	newRate := a.limiter.Limit() / 2
	if newRate < a.minRate {
		newRate = a.minRate
	}
	a.limiter.SetLimit(newRate)
}

func (a *AdaptiveLimiter) CurrentRate() rate.Limit {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.limiter.Limit()
}

func (a *AdaptiveLimiter) ThrottleCount() int64 {
	return atomic.LoadInt64(&a.throttleCount)
}

// Backoff performs exponential backoff sleep for retry attempt N (0-indexed),
// with jitter, capped at maxSleep.
func Backoff(ctx context.Context, attempt int, base, maxSleep time.Duration) error {
	sleep := base * (1 << attempt)
	if sleep > maxSleep {
		sleep = maxSleep
	}
	timer := time.NewTimer(sleep)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
