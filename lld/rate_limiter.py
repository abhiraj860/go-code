from limiter_factory import LimiterFactory
from rate_limiter_models import Limiter, RateLimitResult



class RateLimiter:
    def __init__(self, configs: list[dict], default_config: dict):
        self.factory = LimiterFactory()
        self.limiters: dict[str: Limiter] = {}
        self.default_limiter = None
        for config in configs:
            endpoint = config.get("endpoint")
            if endpoint:
                self.limiters[endpoint] = self.factory.create(config)
        self.default_limiter = self.factory.create(default_config)
            
    def allow(self, client_id: str, endpoint: str) -> RateLimitResult:
       limiter: Limiter = self.limiters.get(endpoint, None)
       if limiter is None:
            limiter: Limiter = self.default_limiter
       return limiter.allow(client_id)
        