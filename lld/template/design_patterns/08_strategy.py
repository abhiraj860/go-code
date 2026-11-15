# STRATEGY: swap the algorithm at runtime.
from abc import ABC, abstractmethod

# 1. Strategy interface
class Strategy(ABC):
    @abstractmethod
    def execute(self, x): ...

# 2. Concrete strategies
class Normal(Strategy):
    def execute(self, x):
        return x

class Surge(Strategy):
    def execute(self, x):
        return x * 2

# 3. Context: holds a strategy and uses it
class Context:
    def __init__(self, strategy):
        self.strategy = strategy

    def run(self, x):
        return self.strategy.execute(x)


if __name__ == "__main__":
    ctx = Context(Normal())
    print(ctx.run(100))      # 100
    ctx.strategy = Surge()   # swap
    print(ctx.run(100))      # 200
