import threading
import time
import sys
import traceback
from main import ConnectionPool

def test_initial_capacity():
    pool = ConnectionPool(3)
    assert pool.acquire(timeout=1.0) is True, "Failed to acquire 1st connection"
    assert pool.acquire(timeout=1.0) is True, "Failed to acquire 2nd connection"
    assert pool.acquire(timeout=1.0) is True, "Failed to acquire 3rd connection"
    assert pool.acquire(timeout=0.1) is False, "Acquired 4th connection but max is 3"

def test_concurrency_limit():
    max_conn = 5
    pool = ConnectionPool(max_conn)
    active_count = 0
    max_observed = 0
    lock = threading.Lock()
    barrier = threading.Barrier(20)
    
    def worker():
        nonlocal active_count, max_observed
        barrier.wait(timeout=5.0)
        if pool.acquire(timeout=5.0):
            with lock:
                active_count += 1
                if active_count > max_observed:
                    max_observed = active_count
            time.sleep(0.05) # Hold the connection briefly
            with lock:
                active_count -= 1
            pool.release()

    threads = [threading.Thread(target=worker) for _ in range(20)]
    for t in threads: t.start()
    for t in threads: t.join(timeout=5.0)
    
    for t in threads:
        assert not t.is_alive(), "A thread deadlocked or missed a signal"
    assert max_observed == max_conn, f"Expected max concurrency {max_conn}, got {max_observed}"

def test_blocking_and_wakeup():
    pool = ConnectionPool(1)
    pool.acquire() # Pool is now empty
    
    success = [False]
    def waiter():
        success[0] = pool.acquire(timeout=5.0)
        
    t = threading.Thread(target=waiter)
    t.start()
    
    time.sleep(0.1) # Give waiter time to block
    assert t.is_alive(), "Thread should be blocked waiting for connection"
    
    pool.release() # Should wake up the waiter
    t.join(timeout=1.0)
    
    assert not t.is_alive(), "Waiter thread did not wake up upon release"
    assert success[0] is True, "Waiter failed to acquire connection after being woken"

def test_timeout_behavior():
    pool = ConnectionPool(1)
    pool.acquire() # Empty the pool
    
    start = time.time()
    result = pool.acquire(timeout=0.2)
    elapsed = time.time() - start
    
    assert result is False, "Expected acquire to return False on timeout"
    assert 0.15 <= elapsed <= 0.4, f"Timeout took {elapsed:.2f}s, expected ~0.2s"

def test_over_release_raises():
    pool = ConnectionPool(2)
    pool.acquire()
    pool.release()
    
    try:
        pool.release()
        assert False, "Expected ValueError when over-releasing, but no exception was raised"
    except ValueError:
        pass # Expected

# --- Test Runner ---
def run_tests():
    tests = [
        test_initial_capacity,
        test_concurrency_limit,
        test_blocking_and_wakeup,
        test_timeout_behavior,
        test_over_release_raises
    ]
    
    passed = 0
    for test in tests:
        try:
            test()
            print(f"✅ {test.__name__}")
            passed += 1
        except AssertionError as e:
            print(f"❌ {test.__name__} FAILED: {e}")
        except Exception as e:
            print(f"❌ {test.__name__} CRASHED:")
            traceback.print_exc()

    print(f"\n{passed}/{len(tests)} passed.")
    if passed < len(tests):
        sys.exit(1)
    sys.exit(0)

if __name__ == "__main__":
    run_tests()