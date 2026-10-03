import unittest
from rate_limiter_models import RateLimitResult, Limiter

import math
from unittest.mock import patch
from token_bucket import TokenBucketLimiter

from sliding_window_log import SlidingWindowLogLimiter

from limiter_factory import LimiterFactory
from token_bucket import TokenBucketLimiter
from sliding_window_log import SlidingWindowLogLimiter

class TestBenchmark4(unittest.TestCase):
    def setUp(self):
        self.factory = LimiterFactory()

    def test_create_token_bucket(self):
        config = {
            "endpoint": "/search",
            "algorithm": "TokenBucket",
            "algoConfig": {
                "capacity": 100,
                "refillRatePerSecond": 10
            }
        }
        limiter = self.factory.create(config)
        self.assertIsInstance(limiter, TokenBucketLimiter)
        self.assertEqual(limiter.capacity, 100)
        self.assertEqual(limiter.refill_rate_per_second, 10)

    def test_create_sliding_window_log(self):
        config = {
            "endpoint": "/upload",
            "algorithm": "SlidingWindowLog",
            "algoConfig": {
                "maxRequests": 5,
                "windowMs": 10000
            }
        }
        limiter = self.factory.create(config)
        self.assertIsInstance(limiter, SlidingWindowLogLimiter)
        self.assertEqual(limiter.max_request, 5) # Note: match this to your actual class attribute name
        self.assertEqual(limiter.window_ms, 10000)

    def test_unknown_algorithm_raises_error(self):
        config = {
            "endpoint": "/health",
            "algorithm": "QuantumBucket",
            "algoConfig": {}
        }
        with self.assertRaises(ValueError):
            self.factory.create(config)




class TestBenchmark3(unittest.TestCase):
    def setUp(self):
        # 3 requests allowed per 60,000 ms (1 minute)
        self.limiter = SlidingWindowLogLimiter(max_requests=3, window_ms=60000)

    @patch('time.time')
    def test_allow_within_limit_and_deny(self, mock_time):
        # t = 10,000 ms (10 sec)
        mock_time.return_value = 10.0
        
        for i in range(3):
            result = self.limiter.allow("user1")
            self.assertTrue(result.is_allowed())
            self.assertEqual(result.get_remaining(), 2 - i)

        # 4th request in the same window should be denied
        result = self.limiter.allow("user1")
        self.assertFalse(result.is_allowed())
        self.assertEqual(result.get_remaining(), 0)
        
        # The oldest request was at 10,000ms. Window is 60,000ms. 
        # It expires at 70,000ms. Current time is 10,000ms.
        # Retry after = 60,000ms.
        self.assertEqual(result.get_retry_after_ms(), 60000)

    # @patch('time.time')
    # def test_sliding_window_cleanup(self, mock_time):
    #     # Make 3 requests at t=0
    #     mock_time.return_value = 0.0
    #     for _ in range(3):
    #         self.limiter.allow("user2")
            
    #     # Attempt at t=30s (denied)
    #     mock_time.return_value = 30.0
    #     result = self.limiter.allow("user2")
    #     self.assertFalse(result.is_allowed())
    #     # Oldest expires at 60,000ms. Current time is 30,000ms. Retry in 30,000ms.
    #     self.assertEqual(result.get_retry_after_ms(), 30000)

    #     # Attempt at t=65s (allowed, since the first 3 requests are now outside the 60s window)
    #     mock_time.return_value = 65.0
    #     result = self.limiter.allow("user2")
    #     self.assertTrue(result.is_allowed())
    #     self.assertEqual(result.get_remaining(), 2)



class TestBenchmark2(unittest.TestCase):
    def setUp(self):
        self.limiter = TokenBucketLimiter(capacity=10, refill_rate_per_second=1)

    @patch('time.time')
    def test_allow_initial_burst_and_deny(self, mock_time):
        mock_time.return_value = 0.0  # 0 ms
        
        # Client should be able to make exactly 10 requests instantly
        for _ in range(10):
            result = self.limiter.allow("user123")
            self.assertTrue(result.is_allowed())
        
        # 11th request must be denied
        result = self.limiter.allow("user123")
        self.assertFalse(result.is_allowed())
        self.assertEqual(result.get_remaining(), 0)
        # Needs 1 full token at 1 token/sec = 1000ms wait
        self.assertEqual(result.get_retry_after_ms(), 1000)

    @patch('time.time')
    def test_refill_logic_partial_token(self, mock_time):
        mock_time.return_value = 0.0
        self.limiter.allow("user456")  # Bucket drops from 10 to 9
        
        # Advance time by 500ms (0.5 seconds)
        mock_time.return_value = 0.5
        result = self.limiter.allow("user456")
        
        self.assertTrue(result.is_allowed())
        # Tokens: 9 + 0.5 (refilled) - 1 (consumed) = 8.5. Floored remaining = 8.
        self.assertEqual(result.get_remaining(), 8)

    @patch('time.time')
    def test_refill_cap_at_capacity(self, mock_time):
        mock_time.return_value = 0.0
        self.limiter.allow("user789")  # Bucket drops to 9
        
        # Advance time by 50 seconds (refills 50 tokens, but should cap at 10)
        mock_time.return_value = 50.0
        result = self.limiter.allow("user789")
        
        self.assertTrue(result.is_allowed())
        # Capped at 10, then consumed 1. Remaining = 9.
        self.assertEqual(result.get_remaining(), 9)
        
class DummyLimiter(Limiter):
    def allow(self, key: str) -> RateLimitResult:
        return RateLimitResult(True, 10, None)

class TestBenchmark1(unittest.TestCase):
    def test_rate_limit_result_allowed(self):
        result = RateLimitResult(allowed=True, remaining=97, retry_after_ms=None)
        self.assertTrue(result.is_allowed())
        self.assertEqual(result.get_remaining(), 97)
        self.assertIsNone(result.get_retry_after_ms())

    def test_rate_limit_result_denied(self):
        result = RateLimitResult(allowed=False, remaining=0, retry_after_ms=1500)
        self.assertFalse(result.is_allowed())
        self.assertEqual(result.get_remaining(), 0)
        self.assertEqual(result.get_retry_after_ms(), 1500)

    def test_limiter_is_abstract(self):
        with self.assertRaises(TypeError):
            Limiter()  # Should fail because it has an abstract method

    def test_dummy_limiter(self):
        limiter = DummyLimiter()
        result = limiter.allow("user123")
        self.assertTrue(result.is_allowed())

if __name__ == '__main__':
    unittest.main()