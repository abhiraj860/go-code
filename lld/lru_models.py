class Node:
    def __init__(self, key, value):
        self.key = key
        self.value = value
        self.next = None
        self.prev = None

class DoublyLinkedList:
    def __init__(self):
        self.head = Node(0, 0)
        self.tail = Node(0, 0)
        self.head.next = self.tail
        self.tail.prev = self.head
        
    def addFirst(self, node: Node):
        node.next = self.head.next
        node.next.prev = node
        self.head.next = node
        node.prev = self.head
        return
    
    def remove(self, node: Node):
        temp = node.next
        temp.prev = node.prev
        node.prev.next = temp
        return    
    
    def moveToFront(self, node: Node):
        self.remove(node)
        self.addFirst(node)
        return
    
    def removeLast(self)->Node:
        node = self.tail.prev
        node.prev.next = node.next
        self.tail.prev = node.prev
        return node
        
        