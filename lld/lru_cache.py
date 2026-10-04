import threading
from lru_models import DoublyLinkedList, Node

class LRUCache:
    def __init__(self, capacity: int):
        self.capacity = capacity
        self.map = {}
        self.dll = DoublyLinkedList()
        self._lock = threading.Lock()
        
    def get(self, key: int)->int:
        with self._lock:
            node:Node = self.map.get(key, None)
            if node is None:
                return -1
            self.dll.moveToFront(node)
            return node.value
    
    def put(self, key: int, value: int):
        with self._lock:
            node: Node = self.map.get(key, None)
            if node is None:
                if len(self.map) >= self.capacity:
                    node = self.dll.removeLast()
                    del self.map[node.key]
                newNode = Node(key, value)
                self.map[key] = newNode
                self.dll.addFirst(newNode)
                return
            node.value = value
            self.dll.moveToFront(node)
            return
        