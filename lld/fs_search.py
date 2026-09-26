from abc import ABC, abstractmethod
from fs_nodes import AbstractNode, FileNode, DirectoryNode

class NodeFilter(ABC):
    @abstractmethod
    def apply(self, node: AbstractNode, params: dict) -> bool:
        pass
    
class FilenameFilter(NodeFilter):
    def apply(self, node: AbstractNode, params: dict):
        para = params.get("name", None)
        if para == node.name or para == None:
            return True
        return False
    
    
class FileSizeFilter(NodeFilter):
    def apply(self, node: AbstractNode, params: dict):
        para = params.get("min_size", None)
        if para is None:
            return True
        if isinstance(node, FileNode) and node.get_size() > para:
            return True
        return False
    
class NodeFilterChain:
    def __init__(self):
        self.filter_objects: list[NodeFilter] = []
        
    def add_filter(self, filter: NodeFilter) -> bool:
        self.filter_objects.append(filter)
        
    def apply_filters(self, node: AbstractNode, params: dict) -> bool:
        for filter in self.filter_objects:
            res = filter.apply(node, params)
            if not res:
                return False
        return True
    
class NodeSearchStrategy(ABC):
    @abstractmethod
    def search(self, directory: DirectoryNode, params: dict) -> list[AbstractNode]:
        pass

class FilenameAndSizeSearchStrategy(NodeSearchStrategy):
    def __init__(self):
        self.filterChain = NodeFilterChain()
        self.filterChain.add_filter(FilenameFilter())
        self.filterChain.add_filter(FileSizeFilter())
        
    def search(self, directory: DirectoryNode, params: dict):
        result = []
        def helper(directory:DirectoryNode):
            if not directory.get_children():
                return
            children_list:AbstractNode = directory.get_children()
            for lst in children_list:
                res = self.filterChain.apply_filters(lst, params)
                if res:
                    result.append(lst)
                if isinstance(lst, DirectoryNode):
                    helper(lst)
            return         
        helper(directory)
        return result