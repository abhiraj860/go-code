# FACTORY: one place decides which class to create.
from abc import ABC, abstractmethod

# 1. Interface
class Product(ABC):
    @abstractmethod
    def run(self): ...

# 2. Concrete products
class ProductA(Product):
    def run(self):
        print("A running")

class ProductB(Product):
    def run(self):
        print("B running")

# 3. Factory: maps type -> class
class Factory:
    @staticmethod
    def create(kind):
        if kind == "a":
            return ProductA()
        if kind == "b":
            return ProductB()
        raise Exception("Unknown type")


if __name__ == "__main__":
    Factory.create("a").run()
