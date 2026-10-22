import threading
import time
import sys
import traceback
from main import ThrottledQueryDispatcher

def test_queue_blocking_and_timeout():
    dispatcher = ThrottledQueryDispatcher(queue_capacity=2, max_concurrent_queries=1)
    assert dispatcher.submit_query("q1", timeout=0.1) is True
    assert dispatcher.submit_query("q2", timeout=0.1) is True
    
    start = time.time()
    assert dispatcher.submit_query("q3", timeout=0.2) is False
    elapsed = time.time() - start
    assert 0.15 <= elapsed <= 0.3, f"Queue block timeout took {elapsed:.2f}s, expected ~0.2s"

def test_semaphore_throttling():
    dispatcher = ThrottledQueryDispatcher(queue_capacity=10, max_concurrent_queries=2)
    for i in range(10):
        dispatcher.submit_query(f"q{i}")
        
    active = 0
    max_active = 0
    track_lock = threading.Lock()
    
    def execute_fn(qid):
        nonlocal active, max_active
        with track_lock:
            active += 1
            if active > max_active:
                max_active = active
        time.sleep(0.05) # simulate execution
        with track_lock:
            active -= 1
        return f"{qid}-done"
        
    def worker():
        dispatcher.worker_step(execute_fn, timeout=1.0)
        
    # Start 10 workers simultaneously
    threads = [threading.Thread(target=worker) for _ in range(10)]
    for t in threads: t.start()
    for t in threads: t.join(timeout=2.0)
    
    assert max_active == 2, f"Semaphore failed: expected max 2 concurrent executions, saw {max_active}"

def test_lock_protection_and_results():
    dispatcher = ThrottledQueryDispatcher(queue_capacity=5, max_concurrent_queries=5)
    dispatcher.submit_query("test-1")
    
    def execute_fn(qid):
        return "success"
        
    dispatcher.worker_step(execute_fn, timeout=0.1)
    
    assert dispatcher.fetch_result("test-1") == "success", "Failed to retrieve stored result"
    assert dispatcher.fetch_result("test-2") is None, "Expected None for unknown query"

def test_end_to_end_contention():
    total_queries = 100
    dispatcher = ThrottledQueryDispatcher(queue_capacity=20, max_concurrent_queries=5)
    
    def execute_fn(qid):
        time.sleep(0.01)
        return f"{qid}-processed"
        
    stop_workers = threading.Event()
    def worker():
        while not stop_workers.is_set():
            dispatcher.worker_step(execute_fn, timeout=0.1)
            
    workers = [threading.Thread(target=worker) for _ in range(10)]
    for w in workers: w.start()
    
    def producer(start_idx):
        for i in range(start_idx, start_idx + 25):
            # Block until submitted
            while not dispatcher.submit_query(f"q{i}", timeout=0.1):
                pass
                
    producers = [threading.Thread(target=producer, args=(i*25,)) for i in range(4)]
    for p in producers: p.start()
    for p in producers: p.join(timeout=5.0)
    
    # Wait for all results to populate
    start_wait = time.time()
    while time.time() - start_wait < 5.0:
        count = sum(1 for i in range(100) if dispatcher.fetch_result(f"q{i}") is not None)
        if count == 100:
            break
        time.sleep(0.1)
        
    stop_workers.set()
    for w in workers: w.join(timeout=2.0)
    
    final_count = sum(1 for i in range(100) if dispatcher.fetch_result(f"q{i}") == f"q{i}-processed")
    assert final_count == 100, f"Expected 100 processed results, got {final_count}"

# --- Test Runner ---
def run_tests():
    tests = [
        test_queue_blocking_and_timeout,
        test_semaphore_throttling,
        test_lock_protection_and_results,
        test_end_to_end_contention
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