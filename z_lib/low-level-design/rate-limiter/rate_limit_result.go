package ratelimiter

type RateLimitResult struct {
	Allowed bool
	Remaining int
	RetryAfterMs *int64
}