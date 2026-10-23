import pytest 
from main import LFUCache


@pytest.fixture
def cache():
    return LFUCache(2)


# --- basic behavior ---
def test_get_missing_returns_minus_one(cache):
    assert cache.get(1) == -1


def test_put_then_get(cache):
    cache.put(1, 10)
    assert cache.get(1) == 10


def test_update_value_does_not_evict(cache):
    cache.put(1, 1)
    cache.put(2, 2)
    cache.put(1, 10)                 # update, cache not over capacity
    assert cache.get(1) == 10
    assert cache.get(2) == 2


# --- eviction rules ---
def test_evicts_lowest_frequency(cache):
    cache.put(1, 1)
    cache.put(2, 2)
    cache.get(1)                     # freq: 1=2, 2=1
    cache.put(3, 3)                  # evict 2
    assert cache.get(2) == -1
    assert cache.get(1) == 1
    assert cache.get(3) == 3


def test_tie_on_frequency_evicts_least_recent(cache):
    cache.put(1, 1)
    cache.put(2, 2)
    cache.put(3, 3)                  # both freq 1, evict older = 1
    assert cache.get(1) == -1
    assert cache.get(2) == 2
    assert cache.get(3) == 3


def test_frequency_beats_recency(cache):
    cache.put(1, 1)
    cache.get(1)
    cache.get(1)                     # key 1: freq 3, but old
    cache.put(2, 2)                  # key 2: freq 1, newest
    cache.put(3, 3)                  # evict 2, not 1
    assert cache.get(2) == -1
    assert cache.get(1) == 1


# --- updates affect ordering ---
def test_put_update_bumps_frequency(cache):
    cache.put(1, 1)
    cache.put(2, 2)
    cache.put(1, 10)                 # freq: 1=2, 2=1
    cache.put(3, 3)                  # evict 2
    assert cache.get(2) == -1
    assert cache.get(1) == 10


def test_get_refreshes_recency(cache):
    cache.put(1, 1)
    cache.put(2, 2)
    cache.get(1)
    cache.get(2)                     # both freq 2, key 1 is now older
    cache.put(3, 3)                  # evict 1
    assert cache.get(1) == -1
    assert cache.get(2) == 2


def test_evicted_key_restarts_at_frequency_one(cache):
    cache.put(1, 1)
    cache.get(1)
    cache.get(1)
    cache.put(2, 2)
    cache.put(3, 3)                  # evict 2
    cache.put(2, 2)                  # 2 returns with freq 1; evicts 3 (freq 1, older)
    assert cache.get(3) == -1
    assert cache.get(2) == 2
    assert cache.get(1) == 1


# --- edge cases ---
def test_capacity_zero_stores_nothing():
    c = LFUCache(0)
    c.put(1, 1)
    assert c.get(1) == -1


def test_capacity_one_keeps_latest():
    c = LFUCache(1)
    c.put(1, 1)
    c.put(2, 2)
    assert c.get(1) == -1
    assert c.get(2) == 2