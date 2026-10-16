class Node:
    def __init__(self, key = 0, value = 0):
        self.key = key
        self.value = value
        self.next = None
        self.prev = None

class LRUCache:
    def __init__(self, capacity: int):
        self.cap = capacity
        self.cache = {}
        self.head = Node()
        self.tail = Node()
        self.head.next = self.tail
        self.tail.prev = self.head

    def _remove(self, node: Node):
        node.prev.next = node.next
        node.next.prev = node.prev
        
    def _insert_head(self, node: Node):
        node.prev = self.head
        node.next = self.head.next
        self.head.next = node
        self.head.next.prev = node
        
    def get(self, key: int) -> int:
        if key not in self.cache:
            return -1
        node:Node = self.cache[key]
        self._remove(node)
        self._insert_head(node)
        return node.value
    
    def put(self, key:int, value:int) -> int:
        if key in self.cache:
            self._remove(self._remove(self.cache[key]))
        node = Node(key, value)
        self.cache[key] = node
        self._insert_head(node)
        if len(self.cache) > self.cap:
            lru:Node = self.tail.prev
            self._remove(lru)
            del self.cache[lru.key]