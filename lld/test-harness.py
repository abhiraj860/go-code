import unittest
from rate_limiter_models import RateLimitResult, Limiter

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