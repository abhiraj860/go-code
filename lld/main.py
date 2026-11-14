from abc import ABC, abstractmethod

class Item(ABC):
    @abstractmethod
    def cost(self): ...
    
class Coffee(Item):
    def cost(self):
        return 100
    
class Decorator(Item):
    def __init__(self, item):
        self.item = item

class Milk(Decorator):
    def cost(self):
        return self.item.cost() + 20
    
class Sugar(Decorator):
    def cost(self):
        return self.item.cost() + 10
    
print(Milk(Sugar(Coffee())).cost())