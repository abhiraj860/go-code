from abc import ABC, abstractmethod
from logging_core import LogMessage
from formatters import LogFormatter, SimpleTextFormatter


class LogAppender(ABC):
    def __init__(self):
        self.formatter = SimpleTextFormatter()
     
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
        print(self.formatter.format(message))
        
    def close(self):
        pass

class FileAppender(LogAppender):
    def __init__(self, filepath: str):
        super().__init__()
        self.filepath = filepath

    def append(self, message: LogMessage):
        with open(self.filepath, 'a') as file:
            file.write(self.formatter.format(message))
    
    def close(self):
        pass        
        