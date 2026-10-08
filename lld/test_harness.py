import unittest
from shopping_domain import OrderStatus, Product, Payment, CreditCardPayment
from shopping_domain import Product

from shopping_cart import OrderItem, ShoppingCart

class TestBenchmark2(unittest.TestCase):
    def setUp(self):
        self.mouse = Product("P2", "Mouse", "Wireless Mouse", 25.00, 50)
        self.keyboard = Product("P3", "Keyboard", "Mechanical Keyboard", 75.00, 30)
        self.cart = ShoppingCart()

    def test_add_and_get_items(self):
        self.cart.add_item(self.mouse, 2)
        self.cart.add_item(self.keyboard, 1)
        
        items = self.cart.get_items()
        self.assertEqual(len(items), 2)
        
        # Adding an existing item should increment its quantity
        self.cart.add_item(self.mouse, 3)
        self.assertEqual(self.cart.items["P2"].quantity, 5)

    def test_update_and_remove_items(self):
        self.cart.add_item(self.mouse, 2)
        self.cart.update_quantity("P2", 10)
        self.assertEqual(self.cart.items["P2"].quantity, 10)
        
        # Updating to 0 or less should remove the item
        self.cart.update_quantity("P2", 0)
        self.assertNotIn("P2", self.cart.items)

        self.cart.add_item(self.keyboard, 1)
        self.cart.remove_item("P3")
        self.assertNotIn("P3", self.cart.items)

    def test_calculate_total_and_clear(self):
        self.cart.add_item(self.mouse, 2)       # 2 * 25.00 = 50.00
        self.cart.add_item(self.keyboard, 1)    # 1 * 75.00 = 75.00
        
        self.assertEqual(self.cart.calculate_total(), 125.00)
        
        self.cart.clear_cart()
        self.assertEqual(len(self.cart.get_items()), 0)
        self.assertEqual(self.cart.calculate_total(), 0.0)

class TestBenchmark1(unittest.TestCase):
    def test_order_status_enum(self):
        self.assertEqual(OrderStatus.PENDING.name, "PENDING")
        self.assertEqual(OrderStatus.DELIVERED.name, "DELIVERED")

    def test_product_availability_and_updates(self):
        laptop = Product(product_id="P1", name="Laptop", description="High-end laptop", price=1500.00, quantity=10)
        
        # Check availability
        self.assertTrue(laptop.is_available(5))
        self.assertTrue(laptop.is_available(10))
        self.assertFalse(laptop.is_available(11))
        
        # Update quantity (simulating a purchase)
        laptop.update_quantity(-5)
        self.assertEqual(laptop.quantity, 5)
        self.assertFalse(laptop.is_available(10))
        
        # Restock
        laptop.update_quantity(20)
        self.assertEqual(laptop.quantity, 25)

    def test_payment_strategy(self):
        # Ensure Payment is abstract
        with self.assertRaises(TypeError):
            Payment()
            
        cc_payment = CreditCardPayment(card_number="1234-5678-9012-3456")
        self.assertTrue(cc_payment.process_payment(1500.00))

if __name__ == '__main__':
    unittest.main()