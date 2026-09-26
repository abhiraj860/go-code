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
              
    def rename(self, path: str, new_name: str):
        path_split = path.rsplit("/", 1)
        parent_node = self.traverse(path_split[0], False)
        node = parent_node.get_node(path_split[1])
        if not node:
            raise ValueError("Node not found")
        parent_node.remove_node(node.name)
        node.name = new_name
        parent_node.add_node(node)
    
    def move(self, source_path: str, destination_dir_path: str):
        path_split = source_path.rsplit("/", 1)
        parent_node = self.traverse(path_split[0], False)
        node = parent_node.get_node(path_split[1])
        if not node:
            raise ValueError("Source node not found")
        target_parent = self.traverse(destination_dir_path, False)
        parent_node.remove_node(node.name)
        target_parent.add_node(node)