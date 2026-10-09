from __future__ import annotations
from typing import Dict, List
from shopping_services import SearchService, InventoryService, PaymentService, OrderService
from cart_and_users import Customer, Product, Address, Account, ShoppingCart
from shopping_order import Order 
from domain_models import PaymentStrategy

class OnlineShoppingSystem:
    _instance = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(OnlineShoppingSystem, cls).__new__(cls)
            cls._instance._initialized = False
        return cls._instance

    def __init__(self):
        if self._initialized:
            return
        self._initialized = True

        self.customers: Dict[str, Customer] = {}
        self.orders: Dict[str, Order] = {}
        self.products: Dict[str, Product] = {}

        self.inventoryService = InventoryService()
        self.searchService = SearchService()
        self.paymentService = PaymentService()
        self.orderService = OrderService(self.inventoryService)
        
    @classmethod
    def getInstance(cls) -> OnlineShoppingSystem:
        if cls._instance is None:
            cls()
        return cls._instance

    def registerCustomer(self, id: str, name: str, email: str, address: Address) -> Customer:
        account = Account(username = name, password="****")
        customer = Customer(id=id, name=name, email = email, shippingAddress=address, account=account)
        self.customers[id] = customer
        return customer

    def addProduct(self, product: Product, quantity: int) -> None:
        self.products[product.getId()] = product
        self.searchService.addProduct(product)
        self.inventoryService.addStock(product, quantity)

    def addToCart(self, customerId: str, productId: str, quantity: int) -> None:
        customer = self.customers.get(customerId)
        product = self.products.get(productId)
        
        if not customer or not product:
            raise ValueError("Customer or Product not found.")
            
        customer.account.cart.addItem(product, quantity)

    def getCustomerCart(self, customerId: str) -> ShoppingCart:
        customer = self.customers.get(customerId)
        if not customer:
            raise ValueError("Customer not found.")
        return customer.account.cart

    def searchProducts(self, name: str) -> List[Product]:
        """Delegate search directly to SearchService."""
        return self.searchService.searchByName(name)

    def placeOrder(self, customerId: str, paymentStrategy: PaymentStrategy) -> Order:
        """Handle checkout: validate cart, process payment, create order, and clean up."""
        customer = self.customers.get(customerId)
        if not customer:
            raise ValueError("Customer not found.")
            
        cart = customer.account.cart
        
        if not cart.getItems():
            raise Exception("Cannot place order: Shopping cart is empty.")
            
        total_amount = cart.calculateTotal()
        
        payment_success = self.paymentService.processPayment(paymentStrategy, total_amount)
        if not payment_success:
            raise Exception("Payment processing failed.")
            
        order = self.orderService.createOrder(customer, cart)
        
        self.orders[order.id] = order
        
        if hasattr(cart, 'clear'):
            cart.clearCart()
        else:
            cart.items = {}  
            
        return order