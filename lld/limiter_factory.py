from rate_limiter_models import Limiter
from token_bucket import TokenBucketLimiter
from sliding_window_log import SlidingWindowLogLimiter

class LimiterFactory:
    def create(self, config_data: dict)->Limiter:
        algo: str = config_data.get("algorithm", None) 
        algoconfig: dict[str, str | int] = config_data.get("algoConfig")
        if algo == "TokenBucket":
            return TokenBucketLimiter(capacity = algoconfig["capacity"], refill_rate_per_second = algoconfig["refillRatePerSecond"])
        if algo == "SlidingWindowLog":
            return SlidingWindowLogLimiter(max_requests = algoconfig["maxRequests"], window_ms = algoconfig["windowMs"]) 
        raise ValueError("Input algorithm is not supported")