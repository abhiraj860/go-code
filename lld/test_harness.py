import unittest
from domain_models import (
    ProductCategory, Product, BaseProduct, ProductDecorator, GiftWrapDecorator, 
    CreditCardPaymentStrategy, UPIPaymentStrategy, PaymentStrategy
)

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