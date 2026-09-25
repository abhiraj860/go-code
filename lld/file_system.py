from fs_nodes import DirectoryNode, FileNode

class FileSystem:
    def __init__(self):
        self.root = DirectoryNode("/")
        
    def traverse(self, path: str, createMissingDirs: bool):
        paths = [v for v in path.split("/") if v]
        parent = self.root
        for name in paths:
            node = parent.get_node(name)
            if isinstance(node, FileNode):
                raise ValueError("You cannot traverse through a file")
            if node:
                parent = node
            else:
                if createMissingDirs:
                    new_node = DirectoryNode(name)
                    parent.add_node(new_node)
                    parent = new_node
                else:
                    raise ValueError("No directories found")
        return parent
                 