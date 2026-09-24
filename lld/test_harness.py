import unittest
from datetime import datetime
import time
from formatters import LogFormatter, SimpleTextFormatter
from logging_core import LogLevel, LogMessage
import io
import os
import tempfile
from unittest.mock import patch
from appenders import LogAppender, ConsoleAppender, FileAppender
from logger import Logger
import io
from unittest.mock import patch
from log_manager import LogManager
from logger import Logger

class TestBenchmark5(unittest.TestCase):
    def setUp(self):
        # Reset the singleton for a clean slate
        LogManager._instance = None
        self.manager = LogManager.get_instance()

    def test_singleton_manager(self):
        manager2 = LogManager.get_instance()
        self.assertIs(self.manager, manager2)

    def test_get_logger_creates_new(self):
        logger = self.manager.get_logger("TestApp")
        self.assertIsInstance(logger, Logger)
        self.assertEqual(logger.name, "TestApp")
        self.assertIn("TestApp", self.manager.loggers)

    def test_get_logger_returns_existing(self):
        logger1 = self.manager.get_logger("Database")
        logger2 = self.manager.get_logger("Database")
        
        # Must return the exact same instance in memory
        self.assertIs(logger1, logger2)
        
        
class TestBenchmark4(unittest.TestCase):
    def setUp(self):
        self.logger = Logger("AppLogger")
        # Default level should be INFO, let's explicitly set to WARNING for the first test
        self.logger.level = LogLevel.WARNING
        
        self.console_appender = ConsoleAppender()
        # Using default formatter for simplicity
        self.logger.add_appender(self.console_appender)

    @patch('sys.stdout', new_callable=io.StringIO)
    def test_logger_filters_by_level(self, mock_stdout):
        # INFO is lower severity than WARNING, so this should be ignored
        self.logger.info("This is an info message")
        self.assertEqual(mock_stdout.getvalue(), "")
        
        # WARNING is equal severity, so it should be processed
        self.logger.warning("This is a warning message")
        self.assertIn("This is a warning message", mock_stdout.getvalue())

    @patch('sys.stdout', new_callable=io.StringIO)
    def test_logger_convenience_methods(self, mock_stdout):
        # Lower threshold to DEBUG to let everything through
        self.logger.level = LogLevel.DEBUG
        
        self.logger.debug("Debug msg")
        self.logger.error("Error msg")
        
        output = mock_stdout.getvalue()
        self.assertIn("Debug msg", output)
        self.assertIn("Error msg", output)
        

class TestBenchmark3(unittest.TestCase):
    def setUp(self):
        self.msg = LogMessage(LogLevel.ERROR, "Database connection failed")
        self.formatter = SimpleTextFormatter()

    def test_appender_interface(self):
        with self.assertRaises(TypeError):
            LogAppender()

    @patch('sys.stdout', new_callable=io.StringIO)
    def test_console_appender(self, mock_stdout):
        appender = ConsoleAppender()
        appender.set_formatter(self.formatter)
        appender.append(self.msg)
        
        output = mock_stdout.getvalue()
        self.assertIn("ERROR", output)
        self.assertIn("Database connection failed", output)

    def test_file_appender(self):
        # Use a temporary file to avoid cluttering the directory
        fd, filepath = tempfile.mkstemp()
        os.close(fd)
        
        try:
            appender = FileAppender(filepath)
            appender.set_formatter(self.formatter)
            appender.append(self.msg)
            appender.close()
            
            with open(filepath, 'r') as f:
                output = f.read()
                
            self.assertIn("ERROR", output)
            self.assertIn("Database connection failed", output)
        finally:
            os.remove(filepath)

class TestBenchmark2(unittest.TestCase):
    def test_formatter_is_abstract(self):
        with self.assertRaises(TypeError):
            LogFormatter()

    def test_simple_text_formatter(self):
        formatter = SimpleTextFormatter()
        msg = LogMessage(LogLevel.WARNING, "Disk space low")
        
        formatted_str = formatter.format(msg)
        
        # Verify the output string is constructed correctly
        self.assertIsInstance(formatted_str, str)
        self.assertIn("WARNING", formatted_str)
        self.assertIn("Disk space low", formatted_str)
        
        # Checking if some form of time was added (length check is a basic proxy)
        self.assertTrue(len(formatted_str) > len("WARNING") + len("Disk space low"))

class TestBenchmark1(unittest.TestCase):
    def test_log_level_ordering(self):
        # Enums should be comparable by severity
        self.assertTrue(LogLevel.DEBUG.value < LogLevel.INFO.value)
        self.assertTrue(LogLevel.INFO.value < LogLevel.WARNING.value)
        self.assertTrue(LogLevel.WARNING.value < LogLevel.ERROR.value)
        self.assertTrue(LogLevel.ERROR.value < LogLevel.FATAL.value)

    def test_log_message_creation(self):
        msg = LogMessage(LogLevel.INFO, "System started")
        self.assertEqual(msg.level, LogLevel.INFO)
        self.assertEqual(msg.message, "System started")
        self.assertIsNotNone(msg.timestamp)
        
        # Ensure timestamp is actually being recorded at creation
        msg2 = LogMessage(LogLevel.ERROR, "Crash")
        self.assertGreaterEqual(msg2.timestamp, msg.timestamp)

if __name__ == '__main__':
    unittest.main()