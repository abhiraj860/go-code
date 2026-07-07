package ratelimiter

import "fmt"

type LimiterFactory struct{}

func (f *LimiterFactory) Create(config map[string]interface{}) Limiter {
	algorithm, _ := config["algorithm"].(string)
	algoConfig, _ := config["algoConfig"].(map[string]interface{})
	if algoConfig == nil {
		algoConfig = map[string]interface{}{}
	}

	switch algorithm {
	case "TokenBucket":
		capacity := toInt(algoConfig["capacity"])
		refillRate := toInt(algoConfig["refillRatePerSecond"])
		return NewTokenBucketLimiter(capacity, refillRate)
	case "SlidingWindowLog":
		maxRequests := toInt(algoConfig["maxRequests"])
		windowMs := toInt64(algoConfig["windowMs"])
		return NewSlidingWindowLogLimiter(maxRequests, windowMs)
	default:
		panic(fmt.Sprintf("unknown algorithm: %s", algorithm))
	}
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func toInt64(value interface{}) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}