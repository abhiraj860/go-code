import threading
import time
import sys
import traceback
from main import AtomicInteger, EventBatcher

def test_atomic_cas_semantics():
    atomic = AtomicInteger(10)
    assert atomic.compare_and_set(10, 20) is True, "CAS should succeed when expected matches"
    assert atomic.get() == 20, "Value should be 20"
    assert atomic.compare_and_set(10, 30) is False, "CAS should fail when expected mismatches"
    assert atomic.get() == 20, "Value should remain 20"

def test_atomic_contention():
    atomic = AtomicInteger(0)
    barrier = threading.Barrier(10)
    
    def worker():
        barrier.wait(timeout=5.0)
        for _ in range(10000):
            atomic.increment_and_get()
            
    threads = [threading.Thread(target=worker) for _ in range(10)]
    for t in threads: t.start()
    for t in threads: t.join(timeout=5.0)
    
    assert atomic.get() == 100000, f"Expected 100000, got {atomic.get()}. Lost updates occurred!"

def test_batcher_size_trigger():
    batcher = EventBatcher(batch_size=3)
    
    def producer():
        time.sleep(0.1)
        batcher.submit("A")
        batcher.submit("B")
        time.sleep(0.1)
        batcher.submit("C") # This should wake the extractor

    t = threading.Thread(target=producer)
    t.start()
    
    start = time.time()
    batch = batcher.extract_batch(timeout=2.0)
    elapsed = time.time() - start
    t.join(timeout=1.0)
    
    assert batch == ["A", "B", "C"], f"Expected full batch, got {batch}"
    assert elapsed < 1.0, "Extractor took too long to wake up after condition was met"

def test_batcher_timeout_trigger():
    batcher = EventBatcher(batch_size=5)
    batcher.submit("A")
    batcher.submit("B")
    
    start = time.time()
    batch = batcher.extract_batch(timeout=0.3)
    elapsed = time.time() - start
    
    assert batch == ["A", "B"], f"Expected partial batch on timeout, got {batch}"
    assert 0.25 <= elapsed <= 0.45, f"Timeout block took {elapsed:.2f}s, expected ~0.3s"

def test_end_to_end_integration():
    batcher = EventBatcher(batch_size=10)
    producers_count = 10
    events_per_producer = 1000
    
    extracted_events = []
    extractor_lock = threading.Lock()
    barrier = threading.Barrier(producers_count + 2) # Producers + 2 extractors
    
    stop_event = threading.Event()

    def producer(pid):
        barrier.wait(timeout=5.0)
        for i in range(events_per_producer):
            batcher.submit(f"p{pid}-{i}")

    def extractor():
        barrier.wait(timeout=5.0)
        while not stop_event.is_set():
            batch = batcher.extract_batch(timeout=0.1)
            if batch:
                with extractor_lock:
                    extracted_events.extend(batch)

    threads = []
    # Start extractors
    for _ in range(2):
        t = threading.Thread(target=extractor)
        t.start()
        threads.append(t)
        
    # Start producers
    for i in range(producers_count):
        t = threading.Thread(target=producer, args=(i,))
        t.start()
        threads.append(t)

    # Wait for producers to finish
    for t in threads[2:]:
        t.join(timeout=5.0)
        
    # Let extractors drain the remaining
    time.sleep(0.5)
    stop_event.set()
    
    for t in threads[:2]:
        t.join(timeout=2.0)

    total_expected = producers_count * events_per_producer
    
    assert batcher.total_submitted.get() == total_expected, \
        f"Atomic tracking failed! Expected {total_expected}, got {batcher.total_submitted.get()}"
    
    assert len(extracted_events) == total_expected, \
        f"Event corruption or loss! Expected {total_expected} events, got {len(extracted_events)}"
        
    assert len(set(extracted_events)) == total_expected, "Duplicate events detected in extraction!"

# --- Test Runner ---
def run_tests():
    tests = [
        test_atomic_cas_semantics,
        test_atomic_contention,
        test_batcher_size_trigger,
        test_batcher_timeout_trigger,
        test_end_to_end_integration
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