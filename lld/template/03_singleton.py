# SINGLETON: only one instance in the whole program.
import threading

class Singleton:
    # 1. Class-level slot + lock
    _instance = None
    _lock = threading.Lock()

    # 2. Check -> lock -> check again -> create
    @classmethod
    def get_instance(cls):
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = cls()
        return cls._instance


if __name__ == "__main__":
    # 3. Always use get_instance()
    print(Singleton.get_instance() is Singleton.get_instance())  # True
