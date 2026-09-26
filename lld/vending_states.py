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
   