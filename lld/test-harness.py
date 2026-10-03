import unittest
from rate_limiter_models import RateLimitResult, Limiter

import math
from unittest.mock import patch
from token_bucket import TokenBucketLimiter

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