class Inventory:
    def reserve(self, item):
        print("reserved", item)
        
class Payment:
    def pay(self, item):
        print("paid for", item)
        
class Shipping:
    def ship(self, item):
        print("shipped", item)
        
class OrderFacade:
    def __init__(self):
        self.inventory = Inventory()
        self.payment = Payment()
        self.shipping = Shipping()
        
    def place_order(self, item):
        self.inventory.reserve(item)
        self.payment.pay(item)
        self.shipping.ship(item)
        
OrderFacade().place_order("book")