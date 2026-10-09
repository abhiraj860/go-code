from domain_models import Product
from abc import ABC, abstractmethod

class CartItem:
    def __init__(self, product: Product, quantity: int):
        self.product = product
        self.quantity = quantity
    
    def incrementQuantity(self, amount: int):
        self.quantity += amount

    def getPrice(self):
        return self.quantity * self.product.getPrice()
    
    
class ShoppingCart:
    def __init__(self):
        self.items: dict[str, CartItem] = {}
    
    def addItem(self, product: Product, quantity: int):
        cartitem = self.items.get(product.getId(), None)
        if cartitem is not None:
            self.items[product.getId()].incrementQuantity(quantity) 
        else:
            self.items[product.getId()] = CartItem(product, quantity)
        return
    
    def removeItem(self, productId: str):
        del self.items[productId]
        
    def getItems(self):
        return self.items
    
    def clearCart(self):
        self.items.clear()
        
    def calculateTotal(self):
        return sum([v.getPrice() for v in self.items.values()])
    
    
class Address:
    def __init__(self, street: str, city: str, state: str, zipCode: str):
        self.street = street
        self.city = city
        self.state = state
        self.zipCode = zipCode

class Account:
    def __init__(self, username: str, password: str):
        self.username = username
        self.password = password    
        self.cart: ShoppingCart = ShoppingCart()
        
class OrderObserver(ABC):
    @abstractmethod
    def update(self, order):
        pass        
    
class Subject(ABC):
    @abstractmethod
    def addObserver(self, observer: OrderObserver):
        pass
    
    @abstractmethod 
    def removeObserver(self, observer: OrderObserver):
        pass
    
    @abstractmethod 
    def notifyObservers(self, order):
        pass
    
class Customer(OrderObserver):
    def __init__(self, id: str, name: str, email: str, account: Account, shippingAddress: Address):
        self.id = id
        self.name = name
        self.email = email
        self.account = account
        self.shippingAddress = shippingAddress

    def updateShippingAddress(self, shippingAddress: Address):
        self.shippingAddress = shippingAddress
        
    def update(self, order):
        return f"Customer {self.name} notified of order update."