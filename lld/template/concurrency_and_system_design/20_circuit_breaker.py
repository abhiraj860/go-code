# CIRCUIT BREAKER: stop calling an endpoint that keeps failing.
# CLOSED --(N failures)--> OPEN --(cooldown)--> HALF_OPEN --(ok)--> CLOSED
#                                                     \--(fail)--> OPEN
import threading
import time

class CircuitOpenError(Exception):
    pass

class CircuitBreaker:
    # 1. Threshold, cooldown, state, lock
    def __init__(self, threshold=3, cooldown=1.0):
        self.threshold = threshold
        self.cooldown = cooldown
        self.state = "CLOSED"
        self.failures = 0
        self.opened_at = 0.0
        self.lock = threading.Lock()

    # 2. Before the call: refuse if OPEN, allow a trial after cooldown
    def _before(self):
        with self.lock:
            if self.state == "OPEN":
                if time.monotonic() - self.opened_at >= self.cooldown:
                    self.state = "HALF_OPEN"
                else:
                    raise CircuitOpenError("circuit open")

    # 3. After the call: success closes, failure counts / re-opens
    def _on_success(self):
        with self.lock:
            self.state = "CLOSED"
            self.failures = 0

    def _on_failure(self):
        with self.lock:
            self.failures += 1
            if self.state == "HALF_OPEN" or self.failures >= self.threshold:
                self.state = "OPEN"
                self.opened_at = time.monotonic()

    def call(self, fn, *args):
        self._before()
        try:
            result = fn(*args)
        except Exception:
            self._on_failure()
            raise
        self._on_success()
        return result


if __name__ == "__main__":
    cb = CircuitBreaker(threshold=2, cooldown=0.2)
    healthy = False

    def endpoint():
        if not healthy:
            raise IOError("down")
        return "ok"

    for i in range(4):
        try:
            print(cb.call(endpoint))
        except Exception as e:
            print(i, type(e).__name__, cb.state)

    time.sleep(0.25)
    healthy = True
    print(cb.call(endpoint), cb.state)   # trial succeeds -> CLOSED
