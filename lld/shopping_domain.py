from enum import Enum, auto
from abc import ABC, abstractmethod

class OrderStatus(Enum):
    PENDING = auto()
    PROCESSING = auto()
    SHIPPED = auto()
    DELIVERED = auto()
    CANCELLED = auto()


class Product:
    def __init__(self, product_id: str, name: str, description: str, price: float, quantity: int):
        self.product_id = product_id
        self.name = name
        self.description = description
        self.price = price
        self.quantity = quantity
    
    def is_available(self, required_amount: int):
        return self.quantity >= required_amount

    def update_quantity(self, amount: int):
        self.quantity += amount
        return 

class Payment(ABC):
    @abstractmethod
    def process_payment(self, amount: float):
        pass

class CreditCardPayment(Payment):
    def __init__(self, card_number: str):
        self.card_number = card_number

    def process_payment(self, amount: float):
        return True
