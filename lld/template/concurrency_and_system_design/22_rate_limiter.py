# RATE LIMITER: know all 3 algorithms.
#   fixed window   -> simple, burst allowed at window boundary
#   sliding log    -> exact, stores every timestamp (memory)
#   token bucket   -> tokens refill = elapsed * rate, capped at capacity
import threading
import time
from collections import defaultdict, deque

# 1. Token bucket (per key)
class TokenBucket:
    def __init__(self, capacity, rate):
        self.capacity = capacity
        self.rate = rate                       # tokens per second
        self.tokens = capacity
        self.last = time.monotonic()
        self.lock = threading.Lock()

    def try_acquire(self):
        with self.lock:
            now = time.monotonic()
            self.tokens = min(self.capacity, self.tokens + (now - self.last) * self.rate)
            self.last = now
            if self.tokens >= 1:
                self.tokens -= 1
                return True
            return False

# 2. Sliding window log (per key)
class SlidingWindowLog:
    def __init__(self, limit, window):
        self.limit = limit
        self.window = window
        self.hits = deque()
        self.lock = threading.Lock()

    def try_acquire(self):
        with self.lock:
            now = time.monotonic()
            while self.hits and now - self.hits[0] >= self.window:
                self.hits.popleft()
            if len(self.hits) < self.limit:
                self.hits.append(now)
                return True
            return False

# 3. Fixed window (per key)
class FixedWindow:
    def __init__(self, limit, window):
        self.limit = limit
        self.window = window
        self.start = time.monotonic()
        self.count = 0
        self.lock = threading.Lock()

    def try_acquire(self):
        with self.lock:
            now = time.monotonic()
            if now - self.start >= self.window:
                self.start, self.count = now, 0
            if self.count < self.limit:
                self.count += 1
                return True
            return False

# 4. Limiter: algorithm is a Strategy (a factory that makes one tracker per key)
class RateLimiter:
    def __init__(self, make_tracker):
        self.trackers = defaultdict(make_tracker)
        self.lock = threading.Lock()

    def allow(self, key):
        with self.lock:
            tracker = self.trackers[key]
        return tracker.try_acquire()


if __name__ == "__main__":
    rl = RateLimiter(lambda: TokenBucket(capacity=3, rate=1))
    print([rl.allow("user1") for _ in range(5)])   # T T T F F
    print(rl.allow("user2"))                       # separate bucket
