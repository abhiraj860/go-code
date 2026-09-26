from __future__ import annotations
from inventory import Inventory
import threading
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from vending_states import VendingMachineState

class VendingMachine:
    
    _instance = None
    _locks = threading.Lock()
     
    def __init__(self):
        if VendingMachine._instance is not None:
            return ValueError("Please invoke the constructor using get_instance() method")
        self.inventory = Inventory()
        self.current_state = None
        self.balance = 0
        self.selected_item_code = None
        
    @classmethod
    def get_instance(cls):
        if cls._instance is not None:
            return cls._instance
        with cls._locks:
            if cls._instance is None:
                cls._instance = VendingMachine()
                return cls._instance
            return cls._instance
        
    def reset(self):
        self.balance = 0
        self.selected_item_code = None
        
    def set_state(self, state: VendingMachineState):
        self.current_state = state