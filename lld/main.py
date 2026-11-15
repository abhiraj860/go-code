from abc import ABC, abstractmethod

class Target(ABC):
    @abstractmethod
    def request(self): ...
    
class OldService:
    def old_request(self):
        print("old service called")
        
class Adapter(Target):
    def __init__(self, old):
        self.old = old

    def request(self):
        self.old.old_request()
        
Adapter(OldService()).request()