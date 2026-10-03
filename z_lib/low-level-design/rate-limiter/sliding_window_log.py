from collections import deque
from rate_limiter_models import Limiter, RateLimitResult
import time

class RequestLog:
    def __init__(self):
        self.timestamps = deque()
        
class SlidingWindowLogLimiter(Limiter):
    def __init__(self, max_requests: int, window_ms: int):
        self.max_request = max_requests
        self.window_ms = window_ms
        self.logs: dict[str, RequestLog] = {}
        
    def _get_or_create_log(self, key: str)->RequestLog:
        log = self.logs.get(key, None)
        if log is None:
            self.logs[key] = RequestLog()
        return self.logs[key]
    
    def allow(self, key: str):
        log = self._get_or_create_log(key)
        now = time.time() * 1000
        cutoff = now - self.window_ms        
        queue = log.timestamps
        while queue and queue[0] <= cutoff:
            queue.popleft()
        if len(queue) < self.max_request:
            queue.append(now * 1000)
            remaining = self.max_request - len(queue)
            return RateLimitResult(True, remaining, None)
        if len(queue) >= self.max_request:
            oldest_ts = queue[0]
            retry_after = (oldest_ts + self.window_ms) - now * 1000
            return RateLimitResult(False, 0, retry_after)  
        return None