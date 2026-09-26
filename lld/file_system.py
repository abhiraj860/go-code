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
    
    def mkdir(self, path:str):
        self.traverse(path, True)
        
    def addFile(self, filepath: str, content: str):
        pathSplit = filepath.rsplit("/", 1)
        file = pathSplit[-1]
        node = self.traverse(pathSplit[-2], True)
        fileNode = FileNode(file)
        fileNode.append_content(content)
        node.add_node(fileNode)
        return
    
    def ls(self, path: str):
        parent = self.traverse(path, False)
        return parent.get_children_name()
    
    def delete(self, path: str):
        if path == "/":
            raise ValueError("Cannot delete root file")
        path_split = path.rsplit("/", 1)
        node = self.traverse(path_split[-2], False)
        node.remove_node(path_split[-1])
    
        
                 