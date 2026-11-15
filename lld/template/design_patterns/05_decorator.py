# DECORATOR: add behaviour by wrapping an object.
from abc import ABC, abstractmethod

# 1. Component interface
class Item(ABC):
    @abstractmethod
    def cost(self): ...

# 2. Base component
class Coffee(Item):
    def cost(self):
        return 100

# 3. Decorator: is an Item AND holds an Item
class Decorator(Item):
    def __init__(self, item):
        self.item = item

# 4. Concrete decorators: add on top of the wrapped item
class Milk(Decorator):
    def cost(self):
        return self.item.cost() + 20

class Sugar(Decorator):
    def cost(self):
        return self.item.cost() + 10


if __name__ == "__main__":
    print(Sugar(Milk(Coffee())).cost())   # 130
