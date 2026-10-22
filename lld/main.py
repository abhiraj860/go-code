import threading
from collections import deque

class ThrottledQueryDispatcher:
    def __init__(self, queue_capacity: int, max_concurrent_queries: int):
        """
        Initialize the dispatcher.
        queue_capacity: Max items the pending queue can hold.
        max_concurrent_queries: Max queries that can execute simultaneously.
        """
        self.queue_capacity = queue_capacity
        self.dispatcher = {}
        self._conditions = threading.Condition()
        self._semaphore = threading.BoundedSemaphore(max_concurrent_queries)
        self._lock = threading.Lock()
        self.queue = deque([])
        self._cv_lock = threading.Lock()
        self._not_empty = threading.Condition(self._cv_lock)
        self._not_full = threading.Condition(self._cv_lock)

    def submit_query(self, query_id: str, timeout: float = None) -> bool:
        """
        Add a query to the pending queue.
        Block if the queue is full until space is available or timeout expires.
        Return True if successfully submitted, False if the timeout expired.
        """
        with self._not_full:
            if not self._not_full.wait_for(lambda: len(self.queue) < self.queue_capacity, timeout = timeout):
                return False
            self.queue.append(query_id)
            self._not_empty.notify()
            return True

    def fetch_result(self, query_id: str) -> str:
        """
        Return the processed result from the internal dictionary.
        Return None if the query is not yet processed or doesn't exist.
        """
        with self._lock:
            return self.dispatcher.get(query_id)

    def worker_step(self, execute_fn, timeout: float = None) -> bool:
        """
        A single cycle for a worker thread:
        1. Pull ONE query from the queue (block if empty, up to `timeout`).
           If timeout expires and no query is retrieved, return False.
        2. Acquire the execution semaphore.
        3. result = execute_fn(query_id)
        4. Release the execution semaphore.
        5. Store the result in the thread-safe dictionary (query_id -> result).
        6. Return True.
        """
        with self._not_empty:
            if not self._not_empty.wait_for(lambda: len(self.queue) > 0, timeout = timeout):
                return False
            query_id = self.queue.popleft()
            self._not_full.notify()
            
        with self._semaphore:
            result = execute_fn(query_id) 
        
        with self._lock:
            self.dispatcher[query_id] = result
        return True