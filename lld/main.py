import threading
from collections import deque

class MessageBroker:
    def __init__(self, capacity: int):
        """Initialize the broker with a maximum capacity."""
        self.queue = deque([])
        self.capacity = capacity
        self.lock = threading.Lock()
        self.not_full = threading.Condition(self.lock)
        self.not_empty = threading.Condition(self.lock)

    def publish(self, message: str, timeout: float = None) -> bool:
        """
        Add a message to the broker. 
        If the broker is full, block until space is available or the timeout expires.
        Return True if successfully published, False if the timeout expired.
        """
        with self.not_full:
            if not self.not_full.wait_for(lambda: len(self.queue) < self.capacity, timeout = timeout):
                    return False
            self.queue.append(message)
            self.not_empty.notify()
            return True

    def consume(self, timeout: float = None) -> str:
        """
        Retrieve and remove the oldest message from the broker (FIFO).
        If the broker is empty, block until a message is available or the timeout expires.
        Raise a TimeoutError if the timeout expires before a message can be consumed.
        """
        with self.not_empty:
            if not self.not_empty.wait_for(lambda: len(self.queue), timeout=timeout):
                    raise TimeoutError("Expired")
            self.not_full.notify_all() 
            return self.queue.popleft()