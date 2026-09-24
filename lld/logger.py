from logging_core import LogLevel
from appenders import LogAppender
from logging_core import LogMessage

class Logger:
    def __init__(self, name):
        self.name = name
        self.level = LogLevel.INFO
        self.appender:list[LogAppender] = []
        
    def add_appender(self, appendType: LogAppender):
        self.appender.append(appendType)
        return
    
    def log(self, level, msg: str):
        if self.level.value <= level.value:
            message = LogMessage(level, msg) 
            for app in self.appender:
                app.append(message)
            
    def debug(self, message: str):
        self.log(LogLevel.DEBUG, message)

    def info(self, message: str):
        self.log(LogLevel.INFO, message)
    
    def warning(self, message: str):
        self.log(LogLevel.WARNING, message)

    def error(self, message: str):
        self.log(LogLevel.ERROR, message)    
        
    def fatal(self, message: str):
        self.log(LogLevel.FATAL, message)