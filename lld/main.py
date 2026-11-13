from abc import ABC, abstractmethod

class Product(ABC):
    @abstractmethod
    def run(self): ...
    
class ProductA(Product):
    def run(self):
        print("A running")
        
class ProductB(Product):
    def run(self):
        print("B running")
        
class Factory:
    @staticmethod
    def create(kind):
        if kind == "a":
            return ProductA()
        if kind == "b":
            return ProductB()
        raise Exception("Unknown type")
    

Factory.create("a").run()
Factory.create("b").run()