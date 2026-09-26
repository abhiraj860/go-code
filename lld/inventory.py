import threading
from vending_models import Item

class Inventory:
    
    def __init__(self):
        self.stock_map: dict[str : int] = {}
        self.item_map: dict[str: Item] = {}
        self._lock = threading.Lock()
        
    def add_item(self, item: Item, count: int):
        with self._lock:
            self.stock_map[item.code] = self.stock_map.get(item.code, 0) + count
            self.item_map[item.code] = item
    
    def get_item(self, code: str):
        with self._lock:
            return self.item_map.get(code, None)
    
    def is_available(self, code: str):
        with self._lock:
            return self.stock_map.get(code, 0) != 0
    
    def reduce_stock(self, code: str):
        with self._lock:
            if self.stock_map.get(code, 0) == 0:
                raise ValueError("Item is out of stock or doesn't exist")
            self.stock_map[code] -= 1