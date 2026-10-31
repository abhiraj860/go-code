# ADAPTER: make an old/3rd-party class fit the interface you want.
from abc import ABC, abstractmethod

# 1. Target: what our code expects
class Target(ABC):
    @abstractmethod
    def request(self): ...

# 2. Adaptee: existing class with a different method
class OldService:
    def old_request(self):
        print("old service called")

# 3. Adapter: implements Target, calls Adaptee
class Adapter(Target):
    def __init__(self, old):
        self.old = old

    def request(self):
        self.old.old_request()


if __name__ == "__main__":
    Adapter(OldService()).request()
