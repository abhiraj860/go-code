from abc import ABC, abstractmethod

class Strategy(ABC):
    @abstractmethod
    def execute(self, x): ...
    
class Normal(Strategy):
    def execute(self, x):
        return x
    
class Surge(Strategy):
    def execute(self, x):
        return x * 2
    
    
class Context:
    def __init__(self, strategy):
        self.strategy = strategy

    def run(self, x):
        return self.strategy.execute(x)
    
ctx = Context(Normal())
print(ctx.run(100))
ctx.strategy = Surge()
print(ctx.run(100))