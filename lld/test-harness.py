import threading
import time
import sys
import traceback
from main import CountDownLatch

def test_no_wait_if_zero():
    latch = CountDownLatch(0)
    start = time.time()
    result = latch.await_zero(timeout=1.0)
    elapsed = time.time() - start
    
    assert result is True, "Expected True when latch is already zero"
    assert elapsed < 0.1, "await_zero blocked when count was already zero"

def test_timeout_behavior():
    latch = CountDownLatch(1)
    start = time.time()
    result = latch.await_zero(timeout=0.2)
    elapsed = time.time() - start
    
    assert result is False, "Expected False on timeout"
    assert 0.15 <= elapsed <= 0.4, f"Timeout took {elapsed:.2f}s, expected ~0.2s"

def test_count_down_wakes_all():
    latch = CountDownLatch(1)
    waiter_count = 5
    success = [False] * waiter_count
    
    def waiter(idx):
        success[idx] = latch.await_zero(timeout=2.0)
        
    threads = [threading.Thread(target=waiter, args=(i,)) for i in range(waiter_count)]
    for t in threads: t.start()
    
    time.sleep(0.1) # Ensure all waiters are blocked
    
    latch.count_down()
    
    for t in threads:
        t.join(timeout=1.0)
        assert not t.is_alive(), "A thread failed to wake up after count_down"
        
    assert all(success), "Not all threads returned True after being woken"

def test_concurrent_count_down():
    total_count = 100
    latch = CountDownLatch(total_count)
    
    def worker():
        latch.count_down()
        
    threads = [threading.Thread(target=worker) for _ in range(total_count)]
    for t in threads: t.start()
    for t in threads: t.join(timeout=2.0)
    
    result = latch.await_zero(timeout=0.1)
    assert result is True, "Latch did not reach zero after concurrent count_downs"

def test_multiple_steps_to_zero():
    latch = CountDownLatch(3)
    
    def decrementor():
        time.sleep(0.1)
        latch.count_down()
        time.sleep(0.1)
        latch.count_down()
        time.sleep(0.1)
        latch.count_down()

    t = threading.Thread(target=decrementor)
    t.start()
    
    result = latch.await_zero(timeout=1.0)
    t.join()
    assert result is True, "Failed to wait for multiple sequential count_downs"

# --- Test Runner ---
def run_tests():
    tests = [
        test_no_wait_if_zero,
        test_timeout_behavior,
        test_count_down_wakes_all,
        test_concurrent_count_down,
        test_multiple_steps_to_zero
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