import threading
import time
import sys
import traceback
from main import MessageBroker

def test_fifo_ordering():
    broker = MessageBroker(5)
    for i in range(5):
        broker.publish(f"msg-{i}")
    
    for i in range(5):
        msg = broker.consume()
        assert msg == f"msg-{i}", f"FIFO broken: expected msg-{i}, got {msg}"

def test_consume_blocks_when_empty():
    broker = MessageBroker(5)
    
    start = time.time()
    try:
        broker.consume(timeout=0.2)
        assert False, "Expected TimeoutError when consuming from empty broker"
    except TimeoutError:
        pass # Expected
    elapsed = time.time() - start
    assert 0.15 <= elapsed <= 0.4, f"Timeout took {elapsed:.2f}s, expected ~0.2s"

def test_publish_blocks_when_full():
    broker = MessageBroker(1)
    assert broker.publish("first") is True
    
    start = time.time()
    result = broker.publish("second", timeout=0.2)
    elapsed = time.time() - start
    
    assert result is False, "Expected publish to return False when full"
    assert 0.15 <= elapsed <= 0.4, f"Timeout took {elapsed:.2f}s, expected ~0.2s"

def test_producer_consumer_wakeup():
    broker = MessageBroker(1)
    
    # Test producer waking up consumer
    result = []
    def consumer():
        try:
            result.append(broker.consume(timeout=2.0))
        except TimeoutError:
            pass
            
    t = threading.Thread(target=consumer)
    t.start()
    time.sleep(0.1)
    broker.publish("hello")
    t.join(timeout=1.0)
    assert result == ["hello"], "Consumer did not wake up and receive message"
    
    # Test consumer waking up producer
    def producer():
        broker.publish("blocked-msg", timeout=2.0)
        
    broker.publish("filling-msg") # fill the capacity
    t2 = threading.Thread(target=producer)
    t2.start()
    time.sleep(0.1)
    
    msg1 = broker.consume()
    t2.join(timeout=1.0)
    msg2 = broker.consume()
    
    assert msg1 == "filling-msg"
    assert msg2 == "blocked-msg", "Producer did not wake up and publish message"

def test_high_concurrency():
    broker = MessageBroker(10)
    producers_count = 10
    consumers_count = 10
    messages_per_producer = 100
    
    consumed_messages = []
    consumed_lock = threading.Lock()
    barrier = threading.Barrier(producers_count + consumers_count)
    
    def producer(p_id):
        barrier.wait(timeout=5.0)
        for i in range(messages_per_producer):
            broker.publish(f"p{p_id}-m{i}")
            
    def consumer():
        barrier.wait(timeout=5.0)
        for _ in range(messages_per_producer):
            try:
                msg = broker.consume(timeout=5.0)
                with consumed_lock:
                    consumed_messages.append(msg)
            except TimeoutError:
                break

    threads = []
    for i in range(producers_count):
        t = threading.Thread(target=producer, args=(i,))
        threads.append(t)
        t.start()
        
    for _ in range(consumers_count):
        t = threading.Thread(target=consumer)
        threads.append(t)
        t.start()
        
    for t in threads:
        t.join(timeout=10.0)
        assert not t.is_alive(), "A thread deadlocked or timed out"

    assert len(consumed_messages) == producers_count * messages_per_producer, \
        f"Expected {producers_count * messages_per_producer} messages, got {len(consumed_messages)}"
    
    assert len(set(consumed_messages)) == len(consumed_messages), "Duplicate messages detected!"

# --- Test Runner ---
def run_tests():
    tests = [
        test_fifo_ordering,
        test_consume_blocks_when_empty,
        test_publish_blocks_when_full,
        test_producer_consumer_wakeup,
        test_high_concurrency
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