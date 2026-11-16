from abc import ABC, abstractmethod

class Node(ABC):
    @abstractmethod
    def size(self): ...
    
class File(Node):
    def __init__(self, size):
        self._size = size
        
    def size(self):
        return self._size
    
    
class Directory(Node):
    def __init__(self):
        self.children = {}
    
    def add(self, name, node):
        self.children[name] = node

    def remove(self, name):
        self.children.pop(name, None)
        
    def size(self):
        return sum(c.size() for c in self.children.values())
    
root = Directory()
docs = Directory()
docs.add("a.txt", File(10))
docs.add("b.txt", File(20))
root.add("docs", docs)
root.add("c.txt", File(5))
print(root.size())
    