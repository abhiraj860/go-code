from abc import ABC, abstractmethod

class DiscountStrategy(ABC):
    @abstractmethod
    def apply_discount(self, price: float) -> float:
        pass

class PercentageDiscount(DiscountStrategy):
    def __init__(self, percentage: float):
        pass
        
    def apply_discount(self, price: float) -> float:
        pass

class FlatDiscount(DiscountStrategy):
    def __init__(self, flat_amount: float):
        pass
        
    def apply_discount(self, price: float) -> float:
        pass

class NoDiscount(DiscountStrategy):
    def apply_discount(self, price: float) -> float:
        pass

class Checkout:
    def __init__(self, strategy: DiscountStrategy):
        pass

    def set_strategy(self, strategy: DiscountStrategy) -> None:
        pass

    def calculate_total(self, price: float) -> float:
        pass