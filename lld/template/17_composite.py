# COMPOSITE: leaf and container share one interface (files and folders, tree).
from abc import ABC, abstractmethod

# 1. Component interface
class Node(ABC):
    @abstractmethod
    def size(self): ...

# 2. Leaf
class File(Node):
    def __init__(self, size):
        self._size = size

    def size(self):
        return self._size

# 3. Composite: holds children, delegates to them
#    Name lives only as the parent's map key.
class Directory(Node):
    def __init__(self):
        self.children = {}

    def add(self, name, node):
        self.children[name] = node

    def remove(self, name):
        self.children.pop(name, None)

    def size(self):                      # computed, never stored
        return sum(c.size() for c in self.children.values())


if __name__ == "__main__":
    root = Directory()
    docs = Directory()
    docs.add("a.txt", File(10))
    docs.add("b.txt", File(20))
    root.add("docs", docs)
    root.add("c.txt", File(5))
    print("total:", root.size())   # 35
