package ratelimiter

import "time"

type SlidingWindowLogLimiter struct {
	maxRequests int
	windowMs int64
	logs map[string]*requestLog
}

type requestLog struct {
	timestamps []int64
}

func NewSlidingWindow(maxRequests int, windowMs int64) *SlidingWindowLogLimiter {
	return &SlidingWindowLogLimiter{
		maxRequests: maxRequests,
		windowMs: windowMs,
		logs: make(map[string] *requestLog),
	}
}

func (l *SlidingWindowLogLimiter) Allow(key string) RateLimitResult {
	log := l.getOrCreateLog(key)
	now := time.Now().UnixMilli()
	cutoff := now - l.windowMs

	idx := 0
	for idx < len(log.timestamps) && log.timestamps[idx] < cutoff {
		idx++
	}

	if idx > 0 {
		log.timestamps = log.timestamps[idx:]
	}

	if len(log.timestamps) < l.maxRequests {
		log.timestamps = append(log.timestamps, now)
		remaining := l.maxRequests - len(log.timestamps)
		return RateLimitResult{Allowed: true, Remaining: remaining}
	}

	oldest := log.timestamps[0]
	retryAfter := (oldest + l.windowMs) - now
	return RateLimitResult{Allowed: false, Remaining: 0, RetryAfterMs: &retryAfter}
}

func (l *SlidingWindowLogLimiter) getOrCreateLog(key string) *requestLog {
	log, ok := l.logs[key]
	if !ok {
		log = &requestLog{timestamps: []int64{}}
		l.logs[key] = log
	}
	return log
}