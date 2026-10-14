import threading
import time
import sys
import traceback
from main import ReentrantLock

def test_mutual_exclusion():
    rlock = ReentrantLock()
    shared_resource = 0
    
    def worker():
        nonlocal shared_resource
        if rlock.acquire(timeout=2.0):
            # Hold lock long enough to force contention
            val = shared_resource
            time.sleep(0.05)
            shared_resource = val + 1
            rlock.release()

    threads = [threading.Thread(target=worker) for _ in range(10)]
    for t in threads: t.start()
    for t in threads: t.join(timeout=2.0)
    
    assert shared_resource == 10, f"Expected 10, got {shared_resource}. Mutual exclusion failed."

def test_reentrancy():
    rlock = ReentrantLock()
    
    # Same thread acquires 3 times
    assert rlock.acquire(timeout=1.0) is True
    assert rlock.acquire(timeout=1.0) is True
    assert rlock.acquire(timeout=1.0) is True
    
    # Should still be locked to other threads
    def other_thread():
        assert rlock.acquire(timeout=0.1) is False, "Other thread acquired lock while held!"
        
    t = threading.Thread(target=other_thread)
    t.start()
    t.join()
    
    # Release 3 times
    rlock.release()
    rlock.release()
    rlock.release()
    
    # Now other thread should be able to acquire
    success = [False]
    def other_thread_success():
        if rlock.acquire(timeout=1.0):
            success[0] = True
            rlock.release()
            
    t2 = threading.Thread(target=other_thread_success)
    t2.start()
    t2.join()
    assert success[0] is True, "Lock did not free properly after matching releases"

def test_unowned_release_raises():
    rlock = ReentrantLock()
    rlock.acquire()
    
    def bad_releaser():
        try:
            rlock.release()
            return False # Should not get here
        except RuntimeError:
            return True # Expected
            
    # Use a mutable list to get the result from the thread
    result = [False]
    def wrapper():
        result[0] = bad_releaser()
        
    t = threading.Thread(target=wrapper)
    t.start()
    t.join()
    
    rlock.release() # Cleanup
    assert result[0] is True, "release() from non-owner did not raise RuntimeError"

def test_timeout_behavior():
    rlock = ReentrantLock()
    rlock.acquire() # Main thread takes the lock
    
    start = time.time()
    def waiter():
        return rlock.acquire(timeout=0.2)
        
    result = [True]
    def wrapper():
        result[0] = waiter()
        
    t = threading.Thread(target=wrapper)
    t.start()
    t.join()
    elapsed = time.time() - start
    
    rlock.release()
    assert result[0] is False, "Expected False on timeout"
    assert 0.15 <= elapsed <= 0.4, f"Timeout took {elapsed:.2f}s, expected ~0.2s"

# --- Test Runner ---
def run_tests():
    tests = [
        test_mutual_exclusion,
        test_reentrancy,
        test_unowned_release_raises,
        test_timeout_behavior
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