import unittest
from lru_models import Node, DoublyLinkedList

from lru_cache import LRUCache

class TestBenchmark2(unittest.TestCase):
    def test_lru_cache_behavior(self):
        cache = LRUCache(2)
        
        cache.put(1, 1) # Cache is {1=1}
        cache.put(2, 2) # Cache is {1=1, 2=2}
        
        # Accessing 1 moves it to the front
        self.assertEqual(cache.get(1), 1)
        
        # Capacity is 2. Inserting 3 should evict the least recently used key (2).
        cache.put(3, 3) 
        self.assertEqual(cache.get(2), -1)
        
        # Inserting 4 should evict key 1
        cache.put(4, 4)
        self.assertEqual(cache.get(1), -1)
        
        # Keys 3 and 4 should still be present
        self.assertEqual(cache.get(3), 3)
        self.assertEqual(cache.get(4), 4)

    def test_lru_cache_update_existing(self):
        cache = LRUCache(2)
        cache.put(1, 10)
        cache.put(2, 20)
        
        # Update existing key 1. This should also mark it as most recently used.
        cache.put(1, 100)
        
        cache.put(3, 30) # This should evict 2, not 1
        
        self.assertEqual(cache.get(1), 100)
        self.assertEqual(cache.get(2), -1)



class TestBenchmark1(unittest.TestCase):
    def setUp(self):
        self.dll = DoublyLinkedList()
        self.node1 = Node(1, "A")
        self.node2 = Node(2, "B")
        self.node3 = Node(3, "C")

    def test_add_first(self):
        self.dll.addFirst(self.node1)
        self.dll.addFirst(self.node2)
        
        # Head -> node2 -> node1 -> Tail
        first_real_node = self.dll.head.next
        self.assertEqual(first_real_node.key, 2)
        self.assertEqual(first_real_node.next.key, 1)

    def test_remove(self):
        self.dll.addFirst(self.node1)
        self.dll.addFirst(self.node2)
        self.dll.addFirst(self.node3)
        
        self.dll.remove(self.node2)
        
        # Head -> node3 -> node1 -> Tail
        self.assertEqual(self.dll.head.next.key, 3)
        self.assertEqual(self.dll.head.next.next.key, 1)
        self.assertEqual(self.dll.tail.prev.key, 1)

    def test_move_to_front(self):
        self.dll.addFirst(self.node1)
        self.dll.addFirst(self.node2)
        
        self.dll.moveToFront(self.node1)
        
        # node1 should now be at the front
        self.assertEqual(self.dll.head.next.key, 1)
        self.assertEqual(self.dll.tail.prev.key, 2)

    def test_remove_last(self):
        self.dll.addFirst(self.node1)
        self.dll.addFirst(self.node2)
        
        removed_node = self.dll.removeLast()
        
        # node1 was added first, so it was pushed to the back. It should be removed.
        self.assertEqual(removed_node.key, 1)
        self.assertEqual(self.dll.tail.prev.key, 2)

if __name__ == '__main__':
    unittest.main()