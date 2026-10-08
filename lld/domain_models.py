from enum import enum, auto
from abc import abc, abstractmethod

class productcategory(enum):
    clothing = auto()
    home_goods = auto()
    grocery = auto()
    electronics = auto()
    books = auto()
    
class product(abc):
    @abstractmethod
    def getid(self)->str:
        pass
    
    @abstractmethod
    def getname(self)->str:
        pass
    
    @abstractmethod
    def getprice(self)->float:
        pass
    
    @abstractmethod
    def getdescription(self)->str:
        pass
    
    @abstractmethod
    def getcategory(self)->productcategory:
        pass
    
class baseproduct(product):
    def __init__(self, id: str, name: str, price: float, description: str, category: productcategory):
        self.id = id
        self.name = name
        self. price = price
        self.description = description
        self.category = category
        
    def getid(self):
        return self.id
    
    def getname(self):
        return self.name
    
    def getprice(self):
        return self.price
    
    def getdescription(self):
        return self.description
    
    def getcategory(self):
        return self.category
    
    
class productdecorator(product):
    def __init__(self, decoratedproduct: product):
        self.decoratedproduct:product = decoratedproduct
    
    def getid(self):
        return self.decoratedproduct.getid()
    
    def getname(self):
        return self.decoratedproduct.getname()
    
    @abstractmethod
    def getprice(self):
        return self.decoratedproduct.getprice()
    
    def getdescription(self):
        return self.decoratedproduct.getdescription()
    
    def getcategory(self):
        return self.decoratedproduct.getcategory()
        
class giftwrapdecorator(productdecorator):
    def __init__(self, product: product, gift_wrap_cost: float = 5.00):
        self.gift_wrap_cost = gift_wrap_cost
        super().__init__(decoratedproduct = product)
    
    def getprice(self):
        return super().getprice() + self.gift_wrap_cost
        
        
class paymentstrategy(abc):
    @abstractmethod
    def pay(self, amount:float) -> bool:
        return true

class creditcardpaymentstrategy(paymentstrategy):
    def __init__(self, cardnumber: str):
        self.cardnumber = cardnumber
    
    def pay(self, amount: float):
        return true
    
class upipaymentstrategy(paymentstrategy):
    def __init__(self, upiid: str):
        self.upiid = upiid
    
    def pay(self, amount: float):
        return true
        
    