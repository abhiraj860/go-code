from shopping_domain import Product

class OrderItem:
    def __init__(self, product: Product, quantity: int):
        self.product = product
        self.quantity = quantity
        
class ShoppingCart:
    def __init__(self):
        self.items: dict[str, OrderItem] = {}
        
    def add_item(self, product: Product, quantity: int):
        id = product.product_id
        order = self.items.get(id, None)
        if order is not None:
            order.quantity += quantity
        else:
            self.items[id] = OrderItem(product, quantity)
        return
    
    def remove_item(self, product_id: str):
        del self.items[product_id]
        return
    
    def update_quantity(self, product_id: str, quantity: int):
        if self.items.get(product_id) is not None:
            self.items[product_id].quantity = quantity
            if self.items[product_id].quantity <= 0:
                del self.items[product_id]
        return
    
    def get_items(self):
        order = [v for v in self.items.values()]
        return order
    
    def clear_cart(self):
        self.items.clear()
        return
    
    def calculate_total(self):
        total = sum([(v.product.price * v.quantity) for v in self.items.values()])
        return total
        