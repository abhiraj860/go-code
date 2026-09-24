from abc import ABC, abstractmethod
from logging_core import LogMessage

class LogFormatter(ABC):
    @abstractmethod
    def format(self, message: LogMessage) -> str:
        pass
    
class SimpleTextFormatter(LogFormatter):
    def format(self, message: LogMessage):
        time = message.timestamp.strftime("%Y-%m-%d %H:%M:%S")
        return f"{time} {message.level} {message.message}"