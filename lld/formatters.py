from abc import ABC, abstractmethod
from logging_core import LogMessage

class LogFormatter(ABC):
    def format(messge: LogMessage):
        