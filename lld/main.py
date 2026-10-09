from abc import ABC, abstractmethod

class Coffee(ABC):
    @abstractmethod
    def get_description(self):
        pass

    @abstractmethod
    def get_cost(self):
        pass

class Espresso(Coffee):
    def __init__(self):
        pass
    
    def get_cost(self):
        return 2.00

    def get_description(self):
        return "Espresso"
    
class HouseBlend(Coffee):
    def __init__(self):
        pass
    
    def get_cost(self):
        return 1.50
    
    def get_description(self):
        return "House Blend"
    
class Milk(Coffee):
    def __init__(self, coffee: Coffee):
        self.coffee = coffee

    def get_cost(self):
        return self.coffee.get_cost() + 0.50
    
    def get_description(self):
        return self.coffee.get_description() + ", Milk"
    
class Mocha(Coffee):
    def __init__(self, coffee: Coffee):
        self.coffee = coffee
    
    def get_cost(self):
        return self.coffee.get_cost() + 0.75
    
    def get_description(self):
        return self.coffee.get_description() + ", Mocha"
            
