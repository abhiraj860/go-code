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
    
    def get_absolute_path(self):
        string  = []
        def helper(node: AbstractNode):
             if node.name == "/":
                 return
             helper(node.parent)
             string.append(node.name)
             return
        helper(self)
        str = "/".join(string) 
        return "/" + str
    

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
        
    def add_node(self, node: AbstractNode):
        name = node.get_name()
        if name in self.children:
            raise ValueError("File Systems don't allow duplicate names in the same folder")
        self.children[name] = node
        node.parent = self
    
    def get_node(self, name: str):
        return self.children.get(name, None)
    
    def get_children(self):
        return list(self.children.values())
    
    def get_children_name(self):
        return list(self.children.keys())
    
    def remove_node(self, name: str):
        if name not in self.children:
            raise ValueError('Child does not exist')
        _ = self.children.pop(name, None)
    