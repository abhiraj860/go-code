import unittest
from vending_models import Item, Coin, Note
import concurrent.futures
from inventory import Inventory

from vending_machine import VendingMachine
from vending_states import VendingMachineState

class TestBenchmark3(unittest.TestCase):
    def setUp(self):
        # Reset singleton for testing
        VendingMachine._instance = None
        self.machine = VendingMachine.get_instance()

    def test_state_is_abstract(self):
        with self.assertRaises(TypeError):
            VendingMachineState()

    def test_machine_is_singleton(self):
        machine2 = VendingMachine.get_instance()
        self.assertIs(self.machine, machine2)

    def test_machine_initialization_and_reset(self):
        self.assertIsNotNone(self.machine.inventory)
        self.assertEqual(self.machine.balance, 0)
        self.assertIsNone(self.machine.selected_item_code)
        
        # Modify and reset
        self.machine.balance = 100
        self.machine.selected_item_code = "A1"
        self.machine.reset()
        
        self.assertEqual(self.machine.balance, 0)
        self.assertIsNone(self.machine.selected_item_code)

class TestBenchmark2(unittest.TestCase):
    def setUp(self):
        self.inventory = Inventory()
        self.soda = Item("Soda", 150, "A1")
        self.chips = Item("Chips", 100, "A2")

    def test_add_and_get_item(self):
        self.inventory.add_item(self.soda, 5)
        self.assertEqual(self.inventory.get_item("A1"), self.soda)
        self.assertIsNone(self.inventory.get_item("B1"))

    def test_is_available_and_reduce_stock(self):
        self.inventory.add_item(self.chips, 1)
        self.assertTrue(self.inventory.is_available("A2"))
        
        self.inventory.reduce_stock("A2")
        self.assertFalse(self.inventory.is_available("A2"))
        
        with self.assertRaises(ValueError):
            self.inventory.reduce_stock("A2")

    def test_concurrent_stock_reduction(self):
        # Add 100 sodas
        self.inventory.add_item(self.soda, 100)
        
        # Try to buy 120 sodas concurrently
        def buy_soda():
            try:
                self.inventory.reduce_stock("A1")
                return True
            except ValueError:
                return False
                
        successful_buys = 0
        with concurrent.futures.ThreadPoolExecutor(max_workers=20) as executor:
            futures = [executor.submit(buy_soda) for _ in range(120)]
            for future in concurrent.futures.as_completed(futures):
                if future.result():
                    successful_buys += 1
                    
        # Should only successfully buy exactly 100 sodas, no negative stock
        self.assertEqual(successful_buys, 100)
        self.assertFalse(self.inventory.is_available("A1"))




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