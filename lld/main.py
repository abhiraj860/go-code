from abc import ABC, abstractmethod

class Coffee(ABC):
    @abstractmethod
    def get_cost(self) -> float:
        pass
        
    @abstractmethod
    def get_description(self) -> str:
        pass

class SimpleCoffee(Coffee):
    def get_cost(self) -> float:
        return 2.0
        
    def get_description(self) -> str:
        return "Simple Coffee"

class CoffeeDecorator(Coffee):
    def __init__(self, coffee: Coffee):
        self.coffee = coffee

    @abstractmethod
    def get_cost(self) -> float:
        pass
        
    @abstractmethod
    def get_description(self) -> str:
        pass

class MilkDecorator(CoffeeDecorator):
        
    def get_cost(self) -> float:
        return self.coffee.get_cost() + 0.5 
        
    def get_description(self) -> str:
        return self.coffee.get_description() + ", Milk"

class WhipDecorator(CoffeeDecorator):
    def get_cost(self) -> float:
        return self.coffee.get_cost() + 0.7
        
    def get_description(self) -> str:
        return self.coffee.get_description() + ", Whip"

class VanillaSyrupDecorator(CoffeeDecorator):
    def get_cost(self) -> float:
        return self.coffee.get_cost() + 0.6
        
    def get_description(self) -> str:
        return self.coffee.get_description() + ", Vanilla Syrup"