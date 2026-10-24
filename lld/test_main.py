"""
Rate limiter practice tests.

Run one algorithm at a time:
    pytest test_main.py -k TokenBucket -v
    pytest test_main.py -k FixedWindow -v
    pytest test_main.py -k SlidingWindowLog -v
    pytest test_main.py -k SlidingWindowCounter -v
    pytest test_main.py -k LeakyBucket -v

Classes not yet in main.py are skipped, not failed.

Contract (implement in main.py). Every class has:
    allow(key: str = "default") -> bool
and takes `clock` (a zero-arg callable returning seconds, default time.monotonic).

    FixedWindowCounter(limit, window, clock=...)
    SlidingWindowLog(limit, window, clock=...)
    SlidingWindowCounter(limit, window, clock=...)   # weighted prev + current window
    TokenBucket(capacity, refill_rate, clock=...)    # starts full, refill_rate = tokens/sec
    LeakyBucket(capacity, leak_rate, clock=...)      # meter variant, starts empty, leak_rate = req/sec
"""
import pytest

import main


# ---------- helpers ----------

class FakeClock:
    def __init__(self, start=0.0):
        self.now = start

    def __call__(self):
        return self.now

    def advance(self, seconds):
        self.now += seconds


@pytest.fixture
def clock():
    return FakeClock()


def load(name):
    cls = getattr(main, name, None)
    if cls is None:
        pytest.skip(f"{name} not implemented yet")
    return cls


def hits(limiter, n, key="u1"):
    return [limiter.allow(key) for _ in range(n)]


# ---------- shared behaviour ----------

FACTORIES = {
    "FixedWindowCounter": lambda c, clk: c(limit=2, window=10, clock=clk),
    "SlidingWindowLog": lambda c, clk: c(limit=2, window=10, clock=clk),
    "SlidingWindowCounter": lambda c, clk: c(limit=2, window=10, clock=clk),
    "TokenBucket": lambda c, clk: c(capacity=2, refill_rate=1, clock=clk),
    "LeakyBucket": lambda c, clk: c(capacity=2, leak_rate=1, clock=clk),
}


@pytest.mark.parametrize("name", FACTORIES)
class TestCommon:
    def test_allows_first_request(self, name, clock):
        limiter = FACTORIES[name](load(name), clock)
        assert limiter.allow("u1") is True

    def test_blocks_after_limit(self, name, clock):
        limiter = FACTORIES[name](load(name), clock)
        assert hits(limiter, 3) == [True, True, False]

    def test_keys_are_independent(self, name, clock):
        limiter = FACTORIES[name](load(name), clock)
        hits(limiter, 3, key="u1")
        assert limiter.allow("u2") is True

    def test_default_key(self, name, clock):
        limiter = FACTORIES[name](load(name), clock)
        assert limiter.allow() is True


# ---------- 1. Fixed window counter ----------

class TestFixedWindowCounter:
    @pytest.fixture
    def limiter(self, clock):
        return load("FixedWindowCounter")(limit=3, window=10, clock=clock)

    def test_resets_in_new_window(self, limiter, clock):
        assert hits(limiter, 4) == [True, True, True, False]
        clock.now = 10
        assert limiter.allow("u1") is True

    def test_still_blocked_inside_same_window(self, limiter, clock):
        hits(limiter, 3)
        clock.now = 9.9
        assert limiter.allow("u1") is False

    def test_boundary_burst_is_allowed(self, limiter, clock):
        """Known flaw: 2x limit across a window edge."""
        clock.now = 9
        assert hits(limiter, 3) == [True, True, True]
        clock.now = 10
        assert hits(limiter, 3) == [True, True, True]

    def test_denied_requests_do_not_count(self, limiter, clock):
        hits(limiter, 10)
        clock.now = 10
        assert hits(limiter, 4) == [True, True, True, False]


# ---------- 2. Sliding window log ----------

class TestSlidingWindowLog:
    @pytest.fixture
    def limiter(self, clock):
        return load("SlidingWindowLog")(limit=3, window=10, clock=clock)

    def test_oldest_entry_expires(self, limiter, clock):
        for t in (0, 1, 2):
            clock.now = t
            assert limiter.allow("u1") is True
        clock.now = 3
        assert limiter.allow("u1") is False
        clock.now = 10.1  # t=0 entry expired, t=1 and t=2 remain
        assert limiter.allow("u1") is True
        assert limiter.allow("u1") is False

    def test_no_boundary_burst(self, limiter, clock):
        clock.now = 9
        assert hits(limiter, 3) == [True, True, True]
        clock.now = 10
        assert limiter.allow("u1") is False

    def test_denied_requests_are_not_logged(self, limiter, clock):
        for t in (0, 1, 2):
            clock.now = t
            limiter.allow("u1")
        for t in (3, 4):
            clock.now = t
            assert limiter.allow("u1") is False
        clock.now = 10.1
        assert limiter.allow("u1") is True

    def test_slides_gradually(self, limiter, clock):
        for t in (0, 1, 2):
            clock.now = t
            limiter.allow("u1")
        clock.now = 10.1
        assert limiter.allow("u1") is True
        clock.now = 11.1  # t=1 expired
        assert limiter.allow("u1") is True
        clock.now = 11.2
        assert limiter.allow("u1") is False


# ---------- 3. Sliding window counter ----------

class TestSlidingWindowCounter:
    @pytest.fixture
    def limiter(self, clock):
        return load("SlidingWindowCounter")(limit=10, window=10, clock=clock)

    def test_blocks_at_limit_in_one_window(self, limiter, clock):
        clock.now = 5
        assert hits(limiter, 11) == [True] * 10 + [False]

    def test_previous_window_is_weighted(self, limiter, clock):
        clock.now = 5
        hits(limiter, 10)
        clock.now = 15  # 50% into window 2 -> prev weight 0.5 -> 5 used
        assert hits(limiter, 6) == [True] * 5 + [False]

    def test_weight_decays_over_time(self, limiter, clock):
        clock.now = 5
        hits(limiter, 10)
        clock.now = 19  # 90% in -> prev weight 0.1 -> 1 used
        assert hits(limiter, 10) == [True] * 9 + [False]

    def test_idle_gap_forgets_old_window(self, limiter, clock):
        clock.now = 5
        hits(limiter, 10)
        clock.now = 100
        assert hits(limiter, 11) == [True] * 10 + [False]

    def test_start_of_next_window_still_blocked(self, limiter, clock):
        clock.now = 5
        hits(limiter, 10)
        clock.now = 10  # prev weight 1.0 -> 10 used
        assert limiter.allow("u1") is False


# ---------- 4. Token bucket ----------

class TestTokenBucket:
    @pytest.fixture
    def limiter(self, clock):
        return load("TokenBucket")(capacity=5, refill_rate=1, clock=clock)

    def test_starts_full_allows_burst(self, limiter):
        assert hits(limiter, 6) == [True] * 5 + [False]

    def test_refills_over_time(self, limiter, clock):
        hits(limiter, 5)
        clock.advance(1)
        assert hits(limiter, 2) == [True, False]

    def test_fractional_refill(self, limiter, clock):
        hits(limiter, 5)
        clock.advance(0.5)
        assert limiter.allow("u1") is False
        clock.advance(0.5)
        assert limiter.allow("u1") is True

    def test_refill_is_capped_at_capacity(self, limiter, clock):
        hits(limiter, 5)
        clock.advance(1000)
        assert hits(limiter, 6) == [True] * 5 + [False]

    def test_faster_refill_rate(self, clock):
        limiter = load("TokenBucket")(capacity=4, refill_rate=2, clock=clock)
        hits(limiter, 4)
        clock.advance(1)
        assert hits(limiter, 3) == [True, True, False]

    def test_denied_request_does_not_consume_token(self, limiter, clock):
        hits(limiter, 5)
        assert limiter.allow("u1") is False
        clock.advance(1)
        assert limiter.allow("u1") is True

    def test_per_key_refill_is_independent(self, limiter, clock):
        hits(limiter, 5, key="a")
        clock.advance(1)
        assert hits(limiter, 6, key="b") == [True] * 5 + [False]
        assert limiter.allow("a") is True


# ---------- 5. Leaky bucket (meter) ----------

class TestLeakyBucket:
    @pytest.fixture
    def limiter(self, clock):
        return load("LeakyBucket")(capacity=3, leak_rate=1, clock=clock)

    def test_fills_then_overflows(self, limiter):
        assert hits(limiter, 4) == [True, True, True, False]

    def test_leaks_over_time(self, limiter, clock):
        hits(limiter, 3)
        clock.advance(2)
        assert hits(limiter, 3) == [True, True, False]

    def test_fractional_leak(self, limiter, clock):
        hits(limiter, 3)
        clock.advance(0.5)
        assert limiter.allow("u1") is False
        clock.advance(0.5)
        assert limiter.allow("u1") is True

    def test_level_never_goes_negative(self, limiter, clock):
        limiter.allow("u1")
        clock.advance(1000)
        assert hits(limiter, 4) == [True, True, True, False]

    def test_steady_rate_at_leak_rate_never_blocks(self, limiter, clock):
        for _ in range(50):
            assert limiter.allow("u1") is True
            clock.advance(1)

    def test_denied_request_does_not_add_to_level(self, limiter, clock):
        hits(limiter, 3)
        assert limiter.allow("u1") is False
        clock.advance(1)
        assert limiter.allow("u1") is True