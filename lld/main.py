import threading

class ConnectionPool:
    def __init__(self, max_connections: int):
        """Initialize the pool with a maximum number of available connections."""
        self._max_connections = max_connections
        self._semaphore = threading.BoundedSemaphore(max_connections)

    def acquire(self, timeout: float = None) -> bool:
        """
        Attempt to acquire a connection. 
        Blocks until one is available or the timeout (in seconds) expires.
        Returns True if acquired, False if the timeout was reached.
        """
        return self._semaphore.acquire(timeout=timeout)

    def release(self) -> None:
        """
        Return a connection to the pool.
        Raises ValueError if releasing would exceed the initial max_connections.
        """
        self._semaphore.release()