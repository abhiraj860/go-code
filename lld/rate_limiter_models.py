from __future__ import annotations
from abc import ABC, abstractmethod

class Limiter(ABC):
    @abstractmethod
    def allow(self, key: str) -> RateLimitResult:
        pass

class RateLimitResult:
    def __init__(self, allowed: bool, remaining: int, retry_after_ms: int | None):
        self.allowed = allowed
        self.remaining = remaining
        self.retry_after_ms = retry_after_ms

    def is_allowed(self) -> bool:
        return self.allowed
    
    def get_remaining(self) -> int:
        return self.remaining
    
    def get_retry_after_ms(self) -> int | None:
        return self.retry_after_ms
