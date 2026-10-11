from abc import ABC, abstractmethod

class DiscountStrategy(ABC):
    @abstractmethod
    def apply_discount(self, price: float) -> float:
        pass

class PercentageDiscount(DiscountStrategy):
    def __init__(self, percentage: float):
        self.percentage = percentage
        if percentage < 0.0 or percentage > 100.0:
            raise ValueError("Negative percentage not allowed") 
    def apply_discount(self, price: float) -> float:
        return price * (1 - self.percentage / 100.0)
        

class FlatDiscount(DiscountStrategy):
    def __init__(self, flat_amount: float):
        self.flat_amount = flat_amount
        
    def apply_discount(self, price: float) -> float:
        price -= self.flat_amount
        return 0.0 if price < 0.0 else price
        
class NoDiscount(DiscountStrategy):
    def apply_discount(self, price: float) -> float:
        return price

class Checkout:
    def __init__(self, strategy: DiscountStrategy):
        self.strategy = strategy

    def set_strategy(self, strategy: DiscountStrategy) -> None:
        self.strategy = strategy

    def calculate_total(self, price: float) -> float:
        if price < 0.0:
            raise ValueError("Negative prices not allowed")
        return self.strategy.apply_discount(price)