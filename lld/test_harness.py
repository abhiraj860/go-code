import unittest
import time
from fs_nodes import AbstractNode, FileNode, DirectoryNode

class TestBenchmark2(unittest.TestCase):
    def setUp(self):
        self.root = DirectoryNode("root")
        self.doc_dir = DirectoryNode("docs")
        self.readme = FileNode("readme.txt")
        
    def test_add_and_get_node(self):
        self.root.add_node(self.doc_dir)
        self.root.add_node(self.readme)
        
        # Verify retrieval
        self.assertIs(self.root.get_node("docs"), self.doc_dir)
        self.assertIs(self.root.get_node("readme.txt"), self.readme)
        self.assertIsNone(self.root.get_node("nonexistent"))
        
        # Verify parent assignment
        self.assertIs(self.doc_dir.parent, self.root)
        self.assertIs(self.readme.parent, self.root)

    def test_add_duplicate_node_raises_error(self):
        self.root.add_node(self.doc_dir)
        duplicate_dir = DirectoryNode("docs")
        with self.assertRaises(ValueError):
            self.root.add_node(duplicate_dir)

    def test_get_children(self):
        self.root.add_node(self.doc_dir)
        self.root.add_node(self.readme)
        
        children = self.root.get_children()
        self.assertEqual(len(children), 2)
        self.assertIn(self.doc_dir, children)
        self.assertIn(self.readme, children)
        
        
class TestBenchmark1(unittest.TestCase):
    def test_abstract_node_cannot_be_instantiated(self):
        # Assuming you use the abc module for AbstractNode
        with self.assertRaises(TypeError):
            AbstractNode("test")

    def test_file_node_initialization_and_content(self):
        file_node = FileNode("readme.txt")
        self.assertEqual(file_node.get_name(), "readme.txt")
        self.assertIsNotNone(file_node.get_created_at())
        self.assertIsNone(file_node.parent)
        
        # Test content operations
        self.assertEqual(file_node.read_content(), "")
        self.assertEqual(file_node.get_size(), 0)
        
        file_node.append_content("Hello World")
        self.assertEqual(file_node.read_content(), "Hello World")
        self.assertEqual(file_node.get_size(), 11)
        
        file_node.append_content("!")
        self.assertEqual(file_node.read_content(), "Hello World!")
        self.assertEqual(file_node.get_size(), 12)

    def test_directory_node_initialization(self):
        dir_node = DirectoryNode("documents")
        self.assertEqual(dir_node.get_name(), "documents")
        self.assertIsInstance(dir_node.children, dict)
        self.assertEqual(len(dir_node.children), 0)
        self.assertIsNone(dir_node.parent)

if __name__ == '__main__':
    unittest.main()