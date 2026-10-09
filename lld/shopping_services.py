from typing import List, Dict
from domain_models import Product, ProductCategory
from shopping_order import OrderLineItem, Order
from domain_models import PaymentStrategy
from cart_and_users import Customer, ShoppingCart

class InventoryService:
    def __init__(self):
        self.stock: Dict[str, int] = {}

    def addStock(self, product: Product, quantity: int) -> None:
        product_id = product.getId()
        self.stock[product_id] = self.stock.get(product_id, 0) + quantity

    def updateStockForOrder(self, items: List[OrderLineItem]) -> None:
        for item in items:
            product_id = item.productId 
            if product_id in self.stock:
                self.stock[product_id] -= item.quantity
                if self.stock[product_id] < 0:
                    self.stock[product_id] = 0
            else:
                self.stock[product_id] = -item.quantity
                
class SearchService:
    def __init__(self):
        self.productCatalog: List[Product] = []

    def addProduct(self, product: Product) -> None:
        if product not in self.productCatalog:
            self.productCatalog.append(product)

    def searchByCategory(self, category: ProductCategory) -> List[Product]:
        return [product for product in self.productCatalog if product.getCategory() == category]

    def searchByName(self, name: str) -> List[Product]:
        query = name.lower()
        return [product for product in self.productCatalog if query in product.getName().lower()]
    
class PaymentService:
    def processPayment(self, strategy: PaymentStrategy, amount: float) -> bool:
        return strategy.pay(amount)
    
class OrderService:
    def __init__(self, inventoryService: InventoryService):
        self.inventoryService = inventoryService
        self._order_counter = 100  

    def createOrder(self, customer: Customer, cart: ShoppingCart) -> Order:
        order_line_items = []
        for cart_item in cart.getItems().values():
            product = cart_item.product 
            p_id = product.getId() 
            p_name = product.getName() 
            p_price = product.getPrice() 
            
            order_line_item = OrderLineItem(
                productId = p_id,
                quantity = cart_item.quantity,
                productName = p_name,
                priceAtPurchase = p_price
            )
            order_line_items.append(order_line_item)

        total_amount = cart.calculateTotal()
        self._order_counter += 1
        order_id = f"ORD-{self._order_counter}"
        order = Order(
            id=order_id,
            customer=customer,
            items=order_line_items,
            totalAmount=total_amount,
            shippingAddress=customer.shippingAddress
        )
        self.inventoryService.updateStockForOrder(order_line_items)
        return order