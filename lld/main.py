import threading

class ReentrantLock:
    def __init__(self):
        """Initialize the reentrant lock using a basic threading.Lock()."""
        self._lock = threading.Lock()
        self._id = None
        self._recursion = 0

    def acquire(self, timeout: float = -1.0) -> bool:
        """
        Acquire the lock.
        If the current thread already owns the lock, increment the recursion count and return True.
        If another thread owns it, block until it's available or the timeout expires.
        Return True if acquired, False if the timeout expired.
        (Note: timeout=-1.0 means block indefinitely in standard Python lock semantics, 
        but you can interpret it as blocking forever if timeout is None or negative).
        """
        current_thread = threading.get_ident()
        if self._id == current_thread:
            self._recursion += 1
            return True
        acquired = self._lock.acquire(timeout = timeout) 
        if acquired:
            self._id = current_thread
            self._recursion += 1
            return True
        return False

    def release(self) -> None:
        """
        Release the lock.
        Decrement the recursion count. If it reaches zero, fully release the lock 
        so other threads can acquire it.
        Raise a RuntimeError if the lock is not currently owned by the calling thread.
        """
        current_thread = threading.get_ident()
        if current_thread != self._id:
            raise RuntimeError("Lock currently not owned by the current thread")
        self._recursion -= 1
        if self._recursion == 0:
            self._id = None
            self._lock.release()