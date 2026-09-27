from __future__ import annotations
from abc import ABC, abstractmethod
from vending_models import Coin, Note
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from vending_machine import VendingMachine

class VendingMachineState(ABC):
    @abstractmethod
    def select_item(self, code: str, machine: VendingMachine):
        pass

    @abstractmethod
    def insert_coin(self, coin: Coin, machine: VendingMachine):
        pass

    @abstractmethod
    def insert_note(self, node: Note, machine: VendingMachine):
        pass

    @abstractmethod
    def dispense(self, machine: VendingMachine):
        pass
    
    @abstractmethod
    def refund(self, machine: VendingMachine):
        pass


class IdleState(VendingMachineState):   
    def select_item(self, code: str, machine: VendingMachine):
        if machine.inventory.is_available(code):
            machine.selected_item_code = code
            machine.set_state(ItemSelectedState())
        else:
            raise ValueError("Item is out of stock and does not exist")
        
    def insert_coin(self, coin: Coin, machine: VendingMachine):
        raise ValueError("Please select item first")

    def insert_note(self, note: Note, machine: VendingMachine):
        raise ValueError("Please select item first")

    def dispense(self, machine: VendingMachine):
        raise ValueError("Please select item first")
    
    def refund(self, machine: VendingMachine):
        raise ValueError("Please select item first")
    
    
class ItemSelectedState(VendingMachineState):
     def select_item(self, code: str, machine: VendingMachine):
        raise ValueError("Item already selected")   
    
     def insert_coin(self, coin: Coin, machine: VendingMachine):
        machine.balance += coin.value
        machine.set_state(HasMoneyState()) 

     def insert_note(self, note: Note, machine: VendingMachine):
        machine.balance += note.value
        machine.set_state(HasMoneyState())

     def dispense(self, machine: VendingMachine):
        raise ValueError("Please insert payment")
    
     def refund(self, machine: VendingMachine):
        raise ValueError("Please insert payment")

class HasMoneyState(VendingMachineState):
    def select_item(self, code: str, machine: VendingMachine):
        pass

    def insert_coin(self, coin: Coin, machine: VendingMachine):
        pass

    def insert_note(self, node: Note, machine: VendingMachine):
        pass

    def dispense(self, machine: VendingMachine):
        pass
    
    def refund(self, machine: VendingMachine):
        pass