import threading

class Singleton:
    _instance = None              # holds the one and only instance
    _lock = threading.Lock()      # makes creation thread-safe

    def __new__(cls):
        # 1st check: skip the lock once the instance exists (fast path)
        if cls._instance is None:
            with cls._lock:
                # 2nd check: another thread may have created it while we waited
                if cls._instance is None:
                    cls._instance = super().__new__(cls)
                    cls._instance._initialized = False   # init flag, starts False
        return cls._instance      # always the same object

    def __init__(self):
        # __init__ runs on every Singleton() call, so guard it
        if self._initialized:     # already set up, don't reset the data
            return
        self._initialized = True
        self.bookings = {}        # set up only once


if __name__ == "__main__":
    a = Singleton()
    a.bookings["B1"] = "data"
    b = Singleton()
    print(a is b)        # True  (same object)
    print(b.bookings)    # {'B1': 'data'}  (not reset)