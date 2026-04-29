package correctness

import "sync"

type RateLimiter struct {
	mu            sync.Mutex
	requestCounts map[string]int
	maxRequests   int
}

func NewRateLimiter(maxRequests int) *RateLimiter {
	return &RateLimiter{
		requestCounts: make(map[string]int),
		maxRequests:   maxRequests,
	}
}

func (rl *RateLimiter) AllowRequest(userId string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	count := rl.requestCounts[userId]
	if count < rl.maxRequests {
		rl.requestCounts[userId] = count + 1
		return true
	}
	return false
}
