import unittest
from datetime import datetime
import time
from formatters import LogFormatter, SimpleTextFormatter
from logging_core import LogLevel, LogMessage

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