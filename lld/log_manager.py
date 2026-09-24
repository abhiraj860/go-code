from logger import Logger

class LogManager:
    _instance = None
    def __init__(self):
        if LogManager._instance is not None:
            raise ValueError("Use LogManager.get_instance() to get the error")
        self.loggers: dict[str, Logger] = {}
    
    @classmethod
    def get_instance(cls):
        if cls._instance is None: 
            cls._instance = LogManager()
        return cls._instance 
    
    def get_logger(self, name: str):
        if name in self.loggers:
            return self.loggers[name] 
        self.loggers[name] = Logger(name)
        return self.loggers[name]