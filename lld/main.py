class Node:
    def __init__(self, key = 0, value = 0):
        self.key = key
        self.value = value
        self.prev = None
        self.next = None

class LRUCache:
    def __init__(self, capacity):
        self.capacity = capacity
        self.kv:dict[int, Node] = {}
        self.head = Node()
        self.tail = Node()
        self.head.next = self.tail
        self.tail.prev = self.head
    
    def move_to_front(self, node:Node):
        node.next = self.head.next
        node.next.prev = node
        node.prev = self.head
        self.head.next = node
    
    def remove_node(self, node: Node):
        node.next.prev = node.prev
        node.prev.next = node.next
    
    def get(self, key):
        if key not in self.kv:
            return -1
        if key in self.kv:
            self.remove_node(self.kv[key])
            self.move_to_front(self.kv[key])
            return self.kv[key].value
        return -1
    
    def put(self, key, value): 
        if key in self.kv:
            self.kv[key].value = value
            self.remove_node(self.kv[key])
            self.move_to_front(self.kv[key])
            return
        if len(self.kv) == self.capacity:
            key1 = self.tail.prev.key
            del self.kv[key1]
            self.remove_node(self.tail.prev)
        node = Node(key, value)
        self.kv[key] = node
        self.move_to_front(node)
        return