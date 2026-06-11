package main

import (
	"fmt"
	"math"
	"sync"
	"time"
)

// ─────────────────────────────────────────────
// 1. TOKEN BUCKET
// Tokens refilled at fixed rate, max capacity = burst limit.
// Request allowed if token available, else rejected.
// ─────────────────────────────────────────────

type TokenBucket struct {
	mu           sync.Mutex
	tokens       float64
	maxTokens    float64
	refillRate   float64 // tokens per second
	lastRefillAt time.Time
}

func NewTokenBucket(maxTokens, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:       maxTokens,
		maxTokens:    maxTokens,
		refillRate:   refillRate,
		lastRefillAt: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefillAt).Seconds()
	tb.tokens = math.Min(tb.maxTokens, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefillAt = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// ─────────────────────────────────────────────
// 2. LEAKY BUCKET
// Requests queued; processed at fixed rate.
// Queue overflow = drop. Smooths bursts into constant output rate.
// ─────────────────────────────────────────────

type LeakyBucket struct {
	mu          sync.Mutex
	queue       int
	maxQueue    int
	leakRate    float64 // requests leaked per second
	lastLeakAt  time.Time
}

func NewLeakyBucket(maxQueue int, leakRate float64) *LeakyBucket {
	return &LeakyBucket{
		maxQueue:   maxQueue,
		leakRate:   leakRate,
		lastLeakAt: time.Now(),
	}
}

func (lb *LeakyBucket) Allow() bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(lb.lastLeakAt).Seconds()
	leaked := int(elapsed * lb.leakRate)
	if leaked > 0 {
		lb.queue = max(0, lb.queue-leaked)
		lb.lastLeakAt = now
	}

	if lb.queue < lb.maxQueue {
		lb.queue++
		return true
	}
	return false
}

// ─────────────────────────────────────────────
// 3. FIXED WINDOW COUNTER
// Time split into fixed windows. Counter resets each window.
// Problem: boundary burst — 2x traffic possible at window edges.
// ─────────────────────────────────────────────

type FixedWindow struct {
	mu           sync.Mutex
	count        int
	limit        int
	windowSize   time.Duration
	windowStart  time.Time
}

func NewFixedWindow(limit int, windowSize time.Duration) *FixedWindow {
	return &FixedWindow{
		limit:       limit,
		windowSize:  windowSize,
		windowStart: time.Now(),
	}
}

func (fw *FixedWindow) Allow() bool {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	now := time.Now()
	if now.Sub(fw.windowStart) >= fw.windowSize {
		// New window: reset counter
		fw.count = 0
		fw.windowStart = now
	}

	if fw.count < fw.limit {
		fw.count++
		return true
	}
	return false
}

// ─────────────────────────────────────────────
// 4. SLIDING WINDOW LOG
// Store timestamp of every request. On each request:
//   - evict timestamps outside the window
//   - count remaining; allow if under limit
// Accurate but memory = O(requests in window).
// ─────────────────────────────────────────────

type SlidingWindowLog struct {
	mu         sync.Mutex
	timestamps []time.Time
	limit      int
	windowSize time.Duration
}

func NewSlidingWindowLog(limit int, windowSize time.Duration) *SlidingWindowLog {
	return &SlidingWindowLog{
		limit:      limit,
		windowSize: windowSize,
	}
}

func (swl *SlidingWindowLog) Allow() bool {
	swl.mu.Lock()
	defer swl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-swl.windowSize)

	// Evict timestamps older than the window
	valid := swl.timestamps[:0]
	for _, t := range swl.timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	swl.timestamps = valid

	if len(swl.timestamps) < swl.limit {
		swl.timestamps = append(swl.timestamps, now)
		return true
	}
	return false
}

// ─────────────────────────────────────────────
// 5. SLIDING WINDOW COUNTER
// Hybrid of fixed window + sliding approximation.
// Formula: count = curr_count + prev_count * overlap_ratio
// Memory efficient, near-accurate. Used by Cloudflare.
// ─────────────────────────────────────────────

type SlidingWindowCounter struct {
	mu          sync.Mutex
	currCount   int
	prevCount   int
	limit       int
	windowSize  time.Duration
	windowStart time.Time
}

func NewSlidingWindowCounter(limit int, windowSize time.Duration) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		limit:       limit,
		windowSize:  windowSize,
		windowStart: time.Now(),
	}
}

func (swc *SlidingWindowCounter) Allow() bool {
	swc.mu.Lock()
	defer swc.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(swc.windowStart)

	if elapsed >= swc.windowSize {
		// Slide: current becomes previous, reset current
		swc.prevCount = swc.currCount
		swc.currCount = 0
		swc.windowStart = now
		elapsed = 0
	}

	// How far into the current window we are (0.0 - 1.0)
	ratio := float64(elapsed) / float64(swc.windowSize)
	// Weighted estimate of requests in the sliding window
	estimated := float64(swc.prevCount)*(1-ratio) + float64(swc.currCount)

	if int(estimated) < swc.limit {
		swc.currCount++
		return true
	}
	return false
}

// ─────────────────────────────────────────────
// DEMO
// ─────────────────────────────────────────────

func simulate(name string, allow func() bool, requests int, delay time.Duration) {
	allowed, denied := 0, 0
	for i := 0; i < requests; i++ {
		if allow() {
			allowed++
		} else {
			denied++
		}
		if delay > 0 {
			time.Sleep(delay)
		}
	}
	fmt.Printf("[%-26s] allowed=%d  denied=%d\n", name, allowed, denied)
}

func main() {
	requests := 20
	delay := 10 * time.Millisecond

	fmt.Println("=== Rate Limiter Comparison ===")
	fmt.Printf("Sending %d requests with %v delay between each\n\n", requests, delay)

	// Token Bucket: max 10 tokens, refill 5/sec
	tb := NewTokenBucket(10, 5)
	simulate("Token Bucket", tb.Allow, requests, delay)

	// Leaky Bucket: queue=10, leak 5/sec
	lb := NewLeakyBucket(10, 5)
	simulate("Leaky Bucket", lb.Allow, requests, delay)

	// Fixed Window: 10 req per 200ms window
	fw := NewFixedWindow(10, 200*time.Millisecond)
	simulate("Fixed Window Counter", fw.Allow, requests, delay)

	// Sliding Window Log: 10 req per 200ms
	swl := NewSlidingWindowLog(10, 200*time.Millisecond)
	simulate("Sliding Window Log", swl.Allow, requests, delay)

	// Sliding Window Counter: 10 req per 200ms
	swc := NewSlidingWindowCounter(10, 200*time.Millisecond)
	simulate("Sliding Window Counter", swc.Allow, requests, delay)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
