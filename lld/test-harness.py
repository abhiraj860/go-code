import threading
import sys
import traceback
from main import AtomicInteger

def test_initial_value_and_get():
    atomic = AtomicInteger(5)
    assert atomic.get() == 5, f"Expected 5, got {atomic.get()}"
    atomic2 = AtomicInteger()
    assert atomic2.get() == 0, f"Expected 0, got {atomic2.get()}"

def test_compare_and_set_basic():
    atomic = AtomicInteger(10)
    
    # Successful CAS
    res = atomic.compare_and_set(10, 20)
    assert res is True, "CAS should return True on success"
    assert atomic.get() == 20, "Value should be updated to 20"
    
    # Failed CAS
    res = atomic.compare_and_set(10, 30)
    assert res is False, "CAS should return False on failure"
    assert atomic.get() == 20, "Value should remain 20 after failed CAS"

def test_increment_and_get_basic():
    atomic = AtomicInteger(0)
    res = atomic.increment_and_get()
    assert res == 1, f"Expected 1, got {res}"
    assert atomic.get() == 1, f"Expected 1, got {atomic.get()}"

def test_no_lost_updates_under_contention():
    atomic = AtomicInteger(0)
    threads_count = 10
    increments_per_thread = 10000
    
    barrier = threading.Barrier(threads_count)
    
    def worker():
        barrier.wait(timeout=5.0)
        for _ in range(increments_per_thread):
            atomic.increment_and_get()
            
    threads = [threading.Thread(target=worker) for _ in range(threads_count)]
    for t in threads: t.start()
    for t in threads: t.join(timeout=5.0)
    
    for t in threads:
        assert not t.is_alive(), "A thread timed out"
        
    expected_total = threads_count * increments_per_thread
    actual_total = atomic.get()
    assert actual_total == expected_total, f"Lost updates! Expected {expected_total}, got {actual_total}"

def test_cas_under_contention():
    atomic = AtomicInteger(0)
    threads_count = 10
    updates_per_thread = 1000
    
    barrier = threading.Barrier(threads_count)
    
    def worker():
        barrier.wait(timeout=5.0)
        for _ in range(updates_per_thread):
            while True:
                current = atomic.get()
                # Attempt to update. If it fails, another thread won the race, so try again.
                if atomic.compare_and_set(current, current + 1):
                    break

    threads = [threading.Thread(target=worker) for _ in range(threads_count)]
    for t in threads: t.start()
    for t in threads: t.join(timeout=5.0)
    
    for t in threads:
        assert not t.is_alive(), "A thread timed out or deadlocked in CAS loop"
        
    expected_total = threads_count * updates_per_thread
    actual_total = atomic.get()
    assert actual_total == expected_total, f"CAS contention failed! Expected {expected_total}, got {actual_total}"

# --- Test Runner ---
def run_tests():
    tests = [
        test_initial_value_and_get,
        test_compare_and_set_basic,
        test_increment_and_get_basic,
        test_no_lost_updates_under_contention,
        test_cas_under_contention
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