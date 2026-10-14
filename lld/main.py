import threading

class CountDownLatch:
    def __init__(self, count: int):
        """Initialize the latch with the given count."""
        self._condition = threading.Condition()
        self._latch_count = count

    def await_zero(self, timeout: float = None) -> bool:
        """
        Block the calling thread until the internal count reaches zero.
        If the count is already zero, return True immediately.
        If the timeout is reached before the count reaches zero, return False.
        Otherwise, return True.
        """
        with self._condition:
            return self._condition.wait_for(lambda: self._latch_count == 0, timeout = timeout)
        

    def count_down(self) -> None:
        """
        Decrement the internal count. 
        If the count reaches zero, wake up ALL waiting threads.
        If the count is already zero, do nothing (or cap at zero).
        """
        with self._condition:
            if self._latch_count == 0:
                return
            self._latch_count -= 1
            if self._latch_count == 0:
                self._condition.notify_all()