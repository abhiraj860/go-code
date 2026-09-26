import unittest
from vending_models import Item, Coin, Note

class TestBenchmark1(unittest.TestCase):
    def test_item_initialization(self):
        item = Item(name="Soda", price=150, code="A1")
        self.assertEqual(item.name, "Soda")
        self.assertEqual(item.price, 150)
        self.assertEqual(item.code, "A1")

    def test_coin_enums(self):
        self.assertEqual(Coin.PENNY.value, 1)
        self.assertEqual(Coin.NICKEL.value, 5)
        self.assertEqual(Coin.DIME.value, 10)
        self.assertEqual(Coin.QUARTER.value, 25)

    def test_note_enums(self):
        self.assertEqual(Note.ONE.value, 100)
        self.assertEqual(Note.FIVE.value, 500)

if __name__ == '__main__':
    unittest.main()