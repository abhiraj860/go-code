import threading

class AtomicInteger:
    def __init__(self, initial_value: int = 0):
        """Initialize the atomic integer."""
        self._atomic_integer = initial_value
        self._threading_lock = threading.Lock()
    
    def get(self) -> int:
        """Return the current value."""
        with self._threading_lock:
            return self._atomic_integer

    def compare_and_set(self, expected: int, new_value: int) -> bool:
        """
        Atomically set the value to new_value IF the current value equals expected.
        Returns True if the update succeeded, False if the current value was not 'expected'.
        """
        with self._threading_lock:
            if self._atomic_integer == expected:
                self._atomic_integer = new_value
                return True
            else:
                return False 

    def increment_and_get(self) -> int:
        """
        Atomically increment the value by 1 and return the NEW value.
        (Hint: You can implement this using a loop and compare_and_set, or directly 
        if you prefer, but it must be completely thread-safe).
        """
        with self._threading_lock:
            self._atomic_integer += 1
            return self._atomic_integer
        