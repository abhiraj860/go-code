import unittest
from main import Espresso, HouseBlend, Milk, Mocha
# ---------------------------------------------------------
# WRITE YOUR CODE HERE
# Implement Coffee, Espresso, HouseBlend, Milk, and Mocha
# ---------------------------------------------------------



# ---------------------------------------------------------
# TEST HARNESS
# ---------------------------------------------------------
class TestDecoratorPattern(unittest.TestCase):
    def test_plain_espresso(self):
        beverage = Espresso()
        self.assertEqual(beverage.get_description(), "Espresso")
        self.assertEqual(beverage.get_cost(), 2.00)

    def test_house_blend_with_milk(self):
        beverage = HouseBlend()
        beverage = Milk(beverage)
        
        self.assertEqual(beverage.get_description(), "House Blend, Milk")
        self.assertEqual(beverage.get_cost(), 2.00) # 1.50 + 0.50

    def test_double_mocha_milk_espresso(self):
        beverage = Espresso()
        beverage = Milk(beverage)
        beverage = Mocha(beverage)
        beverage = Mocha(beverage)
        
        self.assertEqual(beverage.get_description(), "Espresso, Milk, Mocha, Mocha")
        self.assertEqual(beverage.get_cost(), 4.00) # 2.00 + 0.50 + 0.75 + 0.75

if __name__ == '__main__':
    unittest.main(argv=['first-arg-is-ignored'], exit=False)