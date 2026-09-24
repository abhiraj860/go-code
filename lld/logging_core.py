from enum import Enum, auto
import time

class LogLevel(Enum):
    DEBUG = auto()
    INFO = auto()
    WARNING = auto()
    ERROR = auto()
    FATAL = auto()
    
class LogMessage:
    def __init__(self, level, message):
        self.level: LogLevel = level
        self.message: str = message
        self.timestamp = time.time()
        
    