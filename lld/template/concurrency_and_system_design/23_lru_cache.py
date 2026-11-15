# LRU CACHE: hashmap + doubly linked list.
# get/put move node to front; evict from the tail. O(1) both.
# (In real code: functools.lru_cache or collections.OrderedDict.)
import threading

# 1. Node of the doubly linked list
class Node:
    def __init__(self, key=None, val=None):
        self.key, self.val = key, val
        self.prev = self.next = None

class LRUCache:
    # 2. Map + sentinel head/tail (no None checks)
    def __init__(self, capacity):
        self.capacity = capacity
        self.map = {}
        self.head, self.tail = Node(), Node()
        self.head.next, self.tail.prev = self.tail, self.head
        self.lock = threading.Lock()       # get mutates order -> needs the lock too

    # 3. List helpers
    def _remove(self, node):
        node.prev.next, node.next.prev = node.next, node.prev

    def _add_front(self, node):
        node.prev, node.next = self.head, self.head.next
        self.head.next.prev = node
        self.head.next = node

    # 4. API
    def get(self, key):
        with self.lock:
            node = self.map.get(key)
            if not node:
                return None
            self._remove(node)
            self._add_front(node)
            return node.val

    def put(self, key, val):
        with self.lock:
            if key in self.map:
                self._remove(self.map[key])
            elif len(self.map) >= self.capacity:
                lru = self.tail.prev
                self._remove(lru)
                del self.map[lru.key]
            node = Node(key, val)
            self.map[key] = node
            self._add_front(node)


if __name__ == "__main__":
    c = LRUCache(2)
    c.put("a", 1)
    c.put("b", 2)
    c.get("a")            # a is now most recent
    c.put("c", 3)         # evicts b
    print(c.get("b"), c.get("a"), c.get("c"))   # None 1 3
