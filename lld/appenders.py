from abc import ABC, abstractmethod
from logging_core import LogMessage
from formatters import LogFormatter, SimpleTextFormatter
import threading

class LogAppender(ABC):
    def __init__(self):
        self.formatter = SimpleTextFormatter()
        self.lock = threading.Lock()
        
    @abstractmethod
    def append(self, message: LogMessage):
        pass
    
    @abstractmethod
    def close(self):
        pass
    
    def set_formatter(self, formatter: LogFormatter):
        self.formatter = formatter

class ConsoleAppender(LogAppender):
    def __init__(self):
        super().__init__() 
    
    def append(self, message:LogMessage):
        with self.lock:
            print(self.formatter.format(message))
        
    def close(self):
        pass

class FileAppender(LogAppender):
    def __init__(self, filepath: str):
        super().__init__()
        self.filepath = filepath

    def append(self, message: LogMessage):
        with self.lock:
            with open(self.filepath, 'a') as file:
                file.write(self.formatter.format(message) + "\n")
    
    def close(self):
        pass        
        