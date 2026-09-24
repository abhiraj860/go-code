from logger import Logger
import threading

class LogManager:
    _instance = None
    _lock = threading.Lock()
    
    def __init__(self):
        if LogManager._instance is not None:
            raise ValueError("Use LogManager.get_instance() to get the error")
        self.lock = threading.Lock()
        self.loggers: dict[str, Logger] = {}
        
    @classmethod
    def get_instance(cls):
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None: 
                    cls._instance = cls()
        return cls._instance 
    
    def get_logger(self, name: str):
        if name in self.loggers:
            return self.loggers[name]
        with self.lock:
            if name in self.loggers:
                return self.loggers[name] 
            self.loggers[name] = Logger(name)
            return self.loggers[name]