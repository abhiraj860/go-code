import unittest
from domain_models import (
    ProductCategory, Product, BaseProduct, ProductDecorator, GiftWrapDecorator, 
    CreditCardPaymentStrategy, UPIPaymentStrategy, PaymentStrategy
)

from domain_models import BaseProduct, ProductCategory

from cart_and_users import (
    CartItem, ShoppingCart, Address, Account, 
    OrderObserver, Subject, Customer
)
from datetime import datetime
from cart_and_users import Address, Account, Customer
from shopping_order import (
    OrderLineItem, OrderStatus, OrderState, PlacedState, 
    ShippedState, DeliveredState, CancelledState, Order
)

class TestPhase3_Comprehensive(unittest.TestCase):
    def setUp(self):
        self.address = Address("123 Main St", "Tech City", "TS", "10001")
        self.account = Account("dev_user", "secure123")
        self.customer = Customer("C1", "Alice", "alice@example.com", self.account, self.address)
        
        self.item1 = OrderLineItem("P1", 2, "Laptop", 1000.0)
        self.item2 = OrderLineItem("P2", 1, "Mouse", 50.0)
        
        self.order = Order("O-100", self.customer, [self.item1, self.item2], 2050.0, self.address)

    def test_order_initialization(self):
        self.assertEqual(self.order.id, "O-100")
        self.assertEqual(self.order.status, OrderStatus.PLACED)
        self.assertIsInstance(self.order.currentState, PlacedState)
        self.assertIsInstance(self.order.orderDate, datetime)
        self.assertEqual(len(self.order.items), 2)

    def test_valid_state_transitions_and_notifications(self):
        # We can intercept stdout or just trust the state mutation
        # Placed -> Shipped
        self.order.shipOrder()
        self.assertEqual(self.order.status, OrderStatus.SHIPPED)
        self.assertIsInstance(self.order.currentState, ShippedState)
        
        # Shipped -> Delivered
        self.order.deliverOrder()
        self.assertEqual(self.order.status, OrderStatus.DELIVERED)
        self.assertIsInstance(self.order.currentState, DeliveredState)

    def test_invalid_state_transitions(self):
        # Cannot deliver an order that is only PLACED
        with self.assertRaises(Exception):
            self.order.deliverOrder()
            
        # Cancel the order
        self.order.cancelOrder()
        self.assertEqual(self.order.status, OrderStatus.CANCELLED)
        self.assertIsInstance(self.order.currentState, CancelledState)
        
        # Cannot ship a cancelled order
        with self.assertRaises(Exception):
            self.order.shipOrder()

    def test_observer_wiring(self):
        # The customer should be automatically added to the observers list
        self.assertIn(self.customer, self.order.observers)
        
        # Removing the observer
        self.order.removeObserver(self.customer)
        self.assertNotIn(self.customer, self.order.observers)






class TestPhase2_Comprehensive(unittest.TestCase):
    def setUp(self):
        self.mouse = BaseProduct("P1", "Mouse", 25.0, "Wireless", ProductCategory.ELECTRONICS)
        self.keyboard = BaseProduct("P2", "Keyboard", 75.0, "Mechanical", ProductCategory.ELECTRONICS)
        
    def test_cart_item_mechanics(self):
        item = CartItem(self.mouse, 2)
        self.assertEqual(item.getPrice(), 50.0)
        
        item.incrementQuantity(3)
        self.assertEqual(item.quantity, 5)
        self.assertEqual(item.getPrice(), 125.0)

    def test_shopping_cart_exhaustive(self):
        cart = ShoppingCart()
        
        # Test Adding
        cart.addItem(self.mouse, 2)
        cart.addItem(self.keyboard, 1)
        self.assertEqual(len(cart.getItems()), 2)
        self.assertEqual(cart.calculateTotal(), 125.0)
        
        # Test Incrementing existing item
        cart.addItem(self.mouse, 2)
        self.assertEqual(cart.getItems()["P1"].quantity, 4)
        self.assertEqual(cart.calculateTotal(), 175.0)
        
        # Test Removing
        cart.removeItem("P2")
        self.assertNotIn("P2", cart.getItems())
        self.assertEqual(cart.calculateTotal(), 100.0)
        
        # Test Clearing
        cart.clearCart()
        self.assertEqual(len(cart.getItems()), 0)
        self.assertEqual(cart.calculateTotal(), 0.0)

    def test_user_composition_and_observer(self):
        # Ensure Interfaces are abstract
        with self.assertRaises(TypeError):
            OrderObserver()
        with self.assertRaises(TypeError):
            Subject()

        # Build Customer dependencies
        address = Address("123 Main St", "Tech City", "TS", "10001")
        account = Account("dev_user", "secure123")
        
        # We can directly access the instantiated ShoppingCart inside the account
        self.assertIsInstance(account.cart, ShoppingCart)
        
        customer = Customer("C1", "Alice", "alice@example.com", account, address)
        
        self.assertEqual(customer.shippingAddress.city, "Tech City")
        
        # Test Address Update
        new_address = Address("456 Broad St", "New City", "NS", "20002")
        customer.updateShippingAddress(new_address)
        self.assertEqual(customer.shippingAddress.street, "456 Broad St")
        
        # Test Observer implementation
        notification = customer.update("DummyOrderObject")
        self.assertEqual(notification, "Customer Alice notified of order update.")

class TestPhase1_Comprehensive(unittest.TestCase):
    def test_base_product_exhaustive(self):
        prod = BaseProduct("P1", "Laptop", 1000.0, "Gaming Laptop", ProductCategory.ELECTRONICS)
        self.assertEqual(prod.getId(), "P1")
        self.assertEqual(prod.getName(), "Laptop")
        self.assertEqual(prod.getPrice(), 1000.0)
        self.assertEqual(prod.getDescription(), "Gaming Laptop")
        self.assertEqual(prod.getCategory(), ProductCategory.ELECTRONICS)

    def test_product_decorator_full_delegation(self):
        base_prod = BaseProduct("P2", "Book", 20.0, "Novel", ProductCategory.BOOKS)
        wrapped_prod = GiftWrapDecorator(base_prod)
        
        # The abstract decorator MUST delegate all these methods down to the base_prod
        self.assertEqual(wrapped_prod.getId(), "P2")
        self.assertEqual(wrapped_prod.getName(), "Book")
        self.assertEqual(wrapped_prod.getDescription(), "Novel")
        self.assertEqual(wrapped_prod.getCategory(), ProductCategory.BOOKS)
        
        # Only the price should be overridden by the concrete GiftWrapDecorator
        self.assertEqual(wrapped_prod.getPrice(), 25.0)

    def test_interfaces_are_abstract(self):
        # Ensure abstract classes cannot be instantiated directly
        with self.assertRaises(TypeError):
            Product()
        with self.assertRaises(TypeError):
            ProductDecorator(BaseProduct("P", "N", 0, "D", ProductCategory.BOOKS))
        with self.assertRaises(TypeError):
            PaymentStrategy()
            
    def test_payment_strategies(self):
        cc_pay = CreditCardPaymentStrategy("1234-5678-9012-3456")
        self.assertTrue(cc_pay.pay(1025.0))
        
        upi_pay = UPIPaymentStrategy("user@upi")
        self.assertTrue(upi_pay.pay(25.0))

if __name__ == '__main__':
    unittest.main()