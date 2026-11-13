import threading

class Singleton:
    _instance = None
    _lock = threading.Lock()
    
    def __new__(cls):
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = super().__new__(cls)
                    cls._instance._initialized = False
        return cls._instance
    
    def __init__(self):
        if self._initialized:
            return 
        self._initialized = True
        self.bookings = {}
        
a = Singleton()
a.bookings["B1"] = "data"
b = Singleton()
print(a is b)
print(b.bookings)