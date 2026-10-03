import time, math
from rate_limiter_models import RateLimitResult
from rate_limiter_models import Limiter

class TokenBucket:
    def __init__(self, tokens: float, last_refill_time: float):
        self.tokens = tokens
        self.last_refill_time = last_refill_time

class TokenBucketLimiter(Limiter):
    def __init__(self, capacity: int, refill_rate_per_second: int):
        self.capacity = capacity
        self.refill_rate_per_second = refill_rate_per_second
        self.buckets: dict[str:TokenBucket] = {}
        
    def _get_or_create_bucket(self, key: str)-> TokenBucket:
        bucket = self.buckets.get(key, None)
        if bucket is None:
            bucket = TokenBucket(self.capacity, time.time() * 1000)
            self.buckets[key] = bucket
        return bucket
    
    def allow(self, key: str)-> RateLimitResult:
        bucket = self._get_or_create_bucket(key)
        time_elapsed = time.time() * 1000 - bucket.last_refill_time
        tokens_to_add = (self.refill_rate_per_second * time_elapsed) / 1000
        bucket.tokens = min(bucket.tokens + tokens_to_add, self.capacity)
        bucket.last_refill_time = time.time() * 1000
        if bucket.tokens >= 1:
            bucket.tokens -= 1
            return RateLimitResult(True, math.floor(bucket.tokens), None)
        if bucket.tokens < 1:
            tokens_needed = 1 - bucket.tokens
            retry_after_ms = math.ceil((tokens_needed * 1000) / self.refill_rate_per_second)
            return RateLimitResult(False, 0, retry_after_ms)
        return None
        
            
    
    
        