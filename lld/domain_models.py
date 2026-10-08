from enum import Enum, auto
from abc import ABC, abstractmethod

class ProductCategory(Enum):
    CLOTHING = auto()
    HOME_GOODS = auto()
    GROCERY = auto()
    ELECTRONICS = auto()
    BOOKS = auto()
    
class Product(ABC):
    @abstractmethod
    def getId(self)->str:
        pass
    
    @abstractmethod
    def getName(self)->str:
        pass
    
    @abstractmethod
    def getPrice(self)->float:
        pass
    
    @abstractmethod
    def getDescription(self)->str:
        pass
    
    @abstractmethod
    def getCategory(self)->ProductCategory:
        pass
    
class BaseProduct(Product):
    def __init__(self, id: str, name: str, price: float, description: str, category: ProductCategory):
        self.id = id
        self.name = name
        self. price = price
        self.description = description
        self.category = category
        
    def getId(self):
        return self.id
    
    def getName(self):
        return self.name
    
    def getPrice(self):
        return self.price
    
    def getDescription(self):
        return self.description
    
    def getCategory(self):
        return self.category
    
    
class ProductDecorator(Product):
    def __init__(self, decoratedProduct: Product):
        self.decoratedProduct:Product = decoratedProduct
    
    def getId(self):
        return self.decoratedProduct.getId()
    
    def getName(self):
        return self.decoratedProduct.getName()
    
    @abstractmethod
    def getPrice(self):
        return self.decoratedProduct.getPrice()
    
    def getDescription(self):
        return self.decoratedProduct.getDescription()
    
    def getCategory(self):
        return self.decoratedProduct.getCategory()
        
class GiftWrapDecorator(ProductDecorator):
    def __init__(self, product: Product, GIFT_WRAP_COST: float = 5.00):
        self.GIFT_WRAP_COST = GIFT_WRAP_COST
        super().__init__(decoratedProduct = product)
    
    def getPrice(self):
        return super().getPrice() + self.GIFT_WRAP_COST
        
        
class PaymentStrategy(ABC):
    @abstractmethod
    def pay(self, amount:float) -> bool:
        return True

class CreditCardPaymentStrategy(PaymentStrategy):
    def __init__(self, cardNumber: str):
        self.cardNumber = cardNumber
    
    def pay(self, amount: float):
        return True
    
class UPIPaymentStrategy(PaymentStrategy):
    def __init__(self, upiId: str):
        self.upiId = upiId
    
    def pay(self, amount: float):
        return True
        
    