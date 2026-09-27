import unittest
from vending_models import Item, Coin, Note
import concurrent.futures
from inventory import Inventory

from vending_machine import VendingMachine
from vending_states import VendingMachineState

from vending_machine import VendingMachine
from vending_states import IdleState, ItemSelectedState
from vending_models import Item, Coin

from vending_states import HasMoneyState, DispensingState
from vending_models import Item, Coin, Note

class TestBenchmark5(unittest.TestCase):
    def setUp(self):
        VendingMachine._instance = None
        self.machine = VendingMachine.get_instance()
        self.machine.inventory.add_item(Item("Soda", 150, "A1"), 5)
        self.machine.set_state(IdleState())

    def test_has_money_insufficient_funds(self):
        self.machine.current_state.select_item("A1", self.machine)
        self.machine.current_state.insert_coin(Coin.QUARTER, self.machine) # 25 cents
        
        # Balance is 25, price is 150
        with self.assertRaises(ValueError):
            self.machine.current_state.dispense(self.machine)

    def test_has_money_refund(self):
        self.machine.current_state.select_item("A1", self.machine)
        self.machine.current_state.insert_note(Note.ONE, self.machine) # 100 cents
        
        self.machine.current_state.refund(self.machine)
        
        self.assertEqual(self.machine.balance, 0)
        self.assertIsNone(self.machine.selected_item_code)
        self.assertIsInstance(self.machine.current_state, IdleState)

    def test_successful_dispense_and_change(self):
        self.machine.current_state.select_item("A1", self.machine)
        self.machine.current_state.insert_note(Note.ONE, self.machine) # 100 cents
        self.machine.current_state.insert_note(Note.ONE, self.machine) # 100 cents (Total 200)
        
        # This should transition to DispensingState and return the product
        product_name, change = self.machine.current_state.dispense(self.machine)
        
        self.assertEqual(product_name, "Soda")
        self.assertEqual(change, 50)
        self.assertEqual(self.machine.inventory.stock_map["A1"], 4)
        self.assertIsInstance(self.machine.current_state, IdleState)
        self.assertEqual(self.machine.balance, 0)






class TestBenchmark4(unittest.TestCase):
    def setUp(self):
        VendingMachine._instance = None
        self.machine = VendingMachine.get_instance()
        self.machine.inventory.add_item(Item("Soda", 150, "A1"), 5)
        # Manually set initial state
        self.machine.set_state(IdleState())

    def test_idle_state_select_valid_item(self):
        self.machine.current_state.select_item("A1", self.machine)
        self.assertEqual(self.machine.selected_item_code, "A1")
        self.assertIsInstance(self.machine.current_state, ItemSelectedState)

    def test_idle_state_select_invalid_item(self):
        with self.assertRaises(ValueError):
            self.machine.current_state.select_item("B2", self.machine)

    def test_idle_state_invalid_actions(self):
        with self.assertRaises(ValueError):
            self.machine.current_state.insert_coin(Coin.QUARTER, self.machine)
            
    def test_item_selected_state_insert_money(self):
        # Force transition to ItemSelectedState
        self.machine.current_state.select_item("A1", self.machine)
        
        self.machine.current_state.insert_coin(Coin.QUARTER, self.machine)
        self.assertEqual(self.machine.balance, 25)
        # It should transition to HasMoneyState (which we stubbed out)
        self.assertNotIsInstance(self.machine.current_state, ItemSelectedState)

    def test_item_selected_invalid_actions(self):
        self.machine.current_state.select_item("A1", self.machine)
        with self.assertRaises(ValueError):
            self.machine.current_state.dispense(self.machine)


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