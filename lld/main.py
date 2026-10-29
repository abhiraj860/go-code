from typing import Callable
from collections import defaultdict, deque
import time
import threading

class RateLimiter:
    def __init__(self, max_requests: int, window_sec: float,
                 clock: Callable[[], float] = time.monotonic):
        self.max_requests = max_requests
        self.window_sec = window_sec
        self.clock = clock
        self.state = defaultdict(deque)
        self.lock = threading.Lock()
        self.last_sweep = clock()

    def allow(self, user_id: str) -> bool:
        with self.lock:
            now = self.clock()
            if now - self.last_sweep >= self.window_sec:
                lists = []
                for key, val in self.state.items():
                    if len(val) == 0 or val[-1] <= now - self.window_sec:
                        lists.append(key)
                self.last_sweep = now
                for key in lists:
                    del self.state[key]
            queue = self.state[user_id]
            while queue and queue[0] <= now - self.window_sec:
                queue.popleft()
            if self.max_requests <= len(queue):
                return False
            queue.append(now)
            return True