from abc import ABC
import time

class AbstractNode(ABC):
    def __init__(self, name = ""):
       if type(self) is AbstractNode:
           raise TypeError("Cannot instantiate the base class") 
       self.name = name
       self.created_at = time.time()
       self.parent = None
    
    def get_name(self):
        return self.name
    
    def get_created_at(self):
        return self.created_at
    

class FileNode(AbstractNode):
    def __init__(self, name, content = "", size = 0):
       self.content = content
       self.size = size
       super().__init__(name)
       
    def append_content(self, new_content):
        self.content += new_content 
        self.size += len(new_content)
        
    def read_content(self):
        return self.content
    
    def get_size(self):
        return self.size
    
class DirectoryNode(AbstractNode):
    def __init__(self, name):
        super().__init__(name)
        self.children : dict[str, AbstractNode] = {}
        