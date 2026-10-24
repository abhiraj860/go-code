"""
Tests for LRUCache in main.py.

Assumed API (LeetCode style):
    cache = LRUCache(capacity)
    cache.put(key, value)
    cache.get(key) -> value, or -1 if missing

Run:  pytest -v test_lru_cache.py
Coverage:  pytest --cov=main --cov-report=term-missing test_lru_cache.py
"""
import random
from collections import OrderedDict

import pytest

from main import LRUCache

MISS = -1


@pytest.fixture
def cache2():
    return LRUCache(2)


@pytest.fixture
def cache3():
    return LRUCache(3)


# ---------- basic behaviour ----------

def test_get_on_empty_cache_returns_miss(cache2):
    assert cache2.get(1) == MISS


def test_put_then_get(cache2):
    cache2.put(1, 10)
    assert cache2.get(1) == 10


def test_get_missing_key_returns_miss(cache2):
    cache2.put(1, 10)
    assert cache2.get(99) == MISS


def test_multiple_keys_within_capacity(cache3):
    for k in (1, 2, 3):
        cache3.put(k, k * 10)
    assert [cache3.get(k) for k in (1, 2, 3)] == [10, 20, 30]


# ---------- eviction ----------

def test_evicts_lru_when_capacity_exceeded(cache2):
    cache2.put(1, 1)
    cache2.put(2, 2)
    cache2.put(3, 3)          # evicts 1
    assert cache2.get(1) == MISS
    assert cache2.get(2) == 2
    assert cache2.get(3) == 3


def test_only_one_eviction_per_overflow(cache3):
    for k in range(1, 5):     # 1,2,3,4 -> evicts 1 only
        cache3.put(k, k)
    assert cache3.get(1) == MISS
    assert [cache3.get(k) for k in (2, 3, 4)] == [2, 3, 4]


def test_eviction_order_follows_insertion_when_untouched(cache3):
    for k in (1, 2, 3):
        cache3.put(k, k)
    cache3.put(4, 4)          # evicts 1
    cache3.put(5, 5)          # evicts 2
    assert cache3.get(1) == MISS
    assert cache3.get(2) == MISS
    assert [cache3.get(k) for k in (3, 4, 5)] == [3, 4, 5]


# ---------- recency updates ----------

def test_get_refreshes_recency(cache2):
    cache2.put(1, 1)
    cache2.put(2, 2)
    cache2.get(1)             # 1 is now most recent
    cache2.put(3, 3)          # evicts 2, not 1
    assert cache2.get(2) == MISS
    assert cache2.get(1) == 1
    assert cache2.get(3) == 3


def test_put_existing_key_refreshes_recency(cache2):
    cache2.put(1, 1)
    cache2.put(2, 2)
    cache2.put(1, 100)        # 1 is now most recent
    cache2.put(3, 3)          # evicts 2
    assert cache2.get(2) == MISS
    assert cache2.get(1) == 100


def test_failed_get_does_not_change_order(cache2):
    cache2.put(1, 1)
    cache2.put(2, 2)
    cache2.get(99)            # miss, must not affect order
    cache2.put(3, 3)          # still evicts 1
    assert cache2.get(1) == MISS
    assert cache2.get(2) == 2


def test_repeated_gets_keep_key_alive(cache2):
    cache2.put(1, 1)
    cache2.put(2, 2)
    for k in (3, 4, 5):
        assert cache2.get(1) == 1   # keep 1 hot
        cache2.put(k, k)
    assert cache2.get(1) == 1


def test_get_on_most_recent_is_noop(cache2):
    cache2.put(1, 1)
    cache2.put(2, 2)
    cache2.get(2)             # already MRU
    cache2.put(3, 3)          # evicts 1
    assert cache2.get(1) == MISS
    assert cache2.get(2) == 2


def test_get_on_least_recent_moves_it_to_front(cache3):
    for k in (1, 2, 3):
        cache3.put(k, k)
    cache3.get(1)             # LRU -> MRU; order now 2,3,1
    cache3.put(4, 4)          # evicts 2
    assert cache3.get(2) == MISS
    assert [cache3.get(k) for k in (3, 1, 4)] == [3, 1, 4]


def test_get_on_middle_node(cache3):
    for k in (1, 2, 3):
        cache3.put(k, k)
    cache3.get(2)             # middle -> MRU; order now 1,3,2
    cache3.put(4, 4)          # evicts 1
    assert cache3.get(1) == MISS
    assert [cache3.get(k) for k in (3, 2, 4)] == [3, 2, 4]


# ---------- updates / overwrites ----------

def test_overwrite_updates_value(cache2):
    cache2.put(1, 1)
    cache2.put(1, 2)
    assert cache2.get(1) == 2


def test_overwrite_does_not_consume_capacity(cache2):
    cache2.put(1, 1)
    cache2.put(2, 2)
    cache2.put(2, 22)         # update, no growth
    cache2.put(1, 11)         # update, no growth
    assert cache2.get(1) == 11
    assert cache2.get(2) == 22


def test_overwrite_many_times_single_entry(cache2):
    for v in range(100):
        cache2.put("k", v)
    assert cache2.get("k") == 99


# ---------- capacity edge cases ----------

def test_capacity_one():
    c = LRUCache(1)
    c.put(1, 1)
    assert c.get(1) == 1
    c.put(2, 2)               # evicts 1
    assert c.get(1) == MISS
    assert c.get(2) == 2


def test_capacity_one_overwrite_same_key():
    c = LRUCache(1)
    c.put(1, 1)
    c.put(1, 2)
    assert c.get(1) == 2


def test_fills_exactly_to_capacity_no_eviction(cache3):
    for k in (1, 2, 3):
        cache3.put(k, k)
    assert all(cache3.get(k) == k for k in (1, 2, 3))


def test_large_capacity_no_eviction():
    c = LRUCache(1000)
    for k in range(1000):
        c.put(k, k)
    assert all(c.get(k) == k for k in range(1000))


def test_eviction_after_get_in_large_cache():
    c = LRUCache(1000)
    for k in range(1000):
        c.put(k, k)
    c.get(0)                  # protect the oldest
    c.put(1000, 1000)         # evicts 1
    assert c.get(0) == 0
    assert c.get(1) == MISS


# ---------- value / key types ----------

@pytest.mark.parametrize("value", [0, "", False, [], {}, 3.14, "text"])
def test_falsy_and_varied_values_are_stored(value):
    c = LRUCache(2)
    c.put("k", value)
    assert c.get("k") == value


@pytest.mark.parametrize("key", [0, -1, "a", (1, 2), 3.5, 10**12])
def test_varied_hashable_keys(key):
    c = LRUCache(2)
    c.put(key, "v")
    assert c.get(key) == "v"


def test_value_zero_is_not_confused_with_miss(cache2):
    cache2.put(1, 0)
    assert cache2.get(1) == 0


# ---------- classic scenarios ----------

def test_leetcode_example():
    c = LRUCache(2)
    c.put(1, 1)
    c.put(2, 2)
    assert c.get(1) == 1
    c.put(3, 3)               # evicts 2
    assert c.get(2) == MISS
    c.put(4, 4)               # evicts 1
    assert c.get(1) == MISS
    assert c.get(3) == 3
    assert c.get(4) == 4


def test_interleaved_operations(cache3):
    cache3.put(1, 1)
    cache3.put(2, 2)
    cache3.put(3, 3)
    assert cache3.get(2) == 2         # order: 1,3,2
    cache3.put(4, 4)                  # evicts 1 -> 3,2,4
    assert cache3.get(1) == MISS
    cache3.put(3, 33)                 # update   -> 2,4,3
    cache3.put(5, 5)                  # evicts 2 -> 4,3,5
    assert cache3.get(2) == MISS
    assert cache3.get(4) == 4         # -> 3,5,4
    assert cache3.get(3) == 33        # -> 5,4,3
    cache3.put(6, 6)                  # evicts 5
    assert cache3.get(5) == MISS


def test_evicted_key_can_be_reinserted(cache2):
    cache2.put(1, 1)
    cache2.put(2, 2)
    cache2.put(3, 3)          # evicts 1
    cache2.put(1, 111)        # evicts 2
    assert cache2.get(1) == 111
    assert cache2.get(2) == MISS
    assert cache2.get(3) == 3


# ---------- randomized test against a reference model ----------

class ReferenceLRU:
    """Trusted implementation using OrderedDict."""

    def __init__(self, capacity):
        self.cap = capacity
        self.d = OrderedDict()

    def get(self, key):
        if key not in self.d:
            return MISS
        self.d.move_to_end(key)
        return self.d[key]

    def put(self, key, value):
        if key in self.d:
            self.d.move_to_end(key)
        self.d[key] = value
        if len(self.d) > self.cap:
            self.d.popitem(last=False)


@pytest.mark.parametrize("capacity", [1, 2, 3, 5, 10])
@pytest.mark.parametrize("seed", range(5))
def test_random_ops_match_reference(capacity, seed):
    rng = random.Random(seed)
    mine, ref = LRUCache(capacity), ReferenceLRU(capacity)
    key_space = capacity * 2   # forces frequent evictions and hits

    for _ in range(2000):
        key = rng.randint(0, key_space)
        if rng.random() < 0.5:
            val = rng.randint(0, 10_000)
            mine.put(key, val)
            ref.put(key, val)
        else:
            assert mine.get(key) == ref.get(key)

    for key in range(key_space + 1):   # final state must also agree
        assert mine.get(key) == ref.get(key)


# ---------- performance sanity (O(1) ops) ----------

def test_performance_sanity():
    c = LRUCache(1000)
    for i in range(100_000):
        c.put(i, i)
        c.get(i - 500)
    assert c.get(99_999) == 99_999