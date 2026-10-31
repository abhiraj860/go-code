# FACADE: one simple method in front of many classes.

# 1. Subsystems
class Inventory:
    def reserve(self, item):
        print("reserved", item)

class Payment:
    def pay(self, item):
        print("paid for", item)

class Shipping:
    def ship(self, item):
        print("shipped", item)

# 2. Facade: holds subsystems, calls them in order
class OrderFacade:
    def __init__(self):
        self.inventory = Inventory()
        self.payment = Payment()
        self.shipping = Shipping()

    def place_order(self, item):
        self.inventory.reserve(item)
        self.payment.pay(item)
        self.shipping.ship(item)


if __name__ == "__main__":
    OrderFacade().place_order("book")
