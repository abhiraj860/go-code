import unittest
import time
from fs_nodes import AbstractNode, FileNode, DirectoryNode
from file_system import FileSystem

from fs_search import (
    NodeFilter, FilenameFilter, FileSizeFilter, NodeFilterChain,
    NodeSearchStrategy, FilenameAndSizeSearchStrategy
)

class TestBenchmark6(unittest.TestCase):
    def setUp(self):
        self.fs = FileSystem()
        self.fs.mkdir("/home/user/docs")
        # Sizes will be based on content length
        self.fs.addFile("/home/user/docs/report.txt", "Quarterly Report Data") # size: 21
        self.fs.addFile("/home/user/docs/notes.txt", "Meeting notes")         # size: 13
        self.fs.addFile("/home/user/docs/report_v2.txt", "Q")                  # size: 1
        
        self.fs.mkdir("/home/user/archive")
        self.fs.addFile("/home/user/archive/report.txt", "Old Data")           # size: 8

    def test_filename_filter(self):
        f_filter = FilenameFilter()
        node = self.fs.traverse("/home/user", False)
        self.assertTrue(f_filter.apply(node, {"name": "user"}))
        self.assertFalse(f_filter.apply(node, {"name": "docs"}))

    def test_filesize_filter(self):
        s_filter = FileSizeFilter()
        file_node = self.fs.traverse("/home/user/docs", False).get_node("report.txt")
        
        # report.txt size is 21
        self.assertTrue(s_filter.apply(file_node, {"min_size": 10}))
        self.assertFalse(s_filter.apply(file_node, {"min_size": 50}))
        
        # Should safely ignore directories (return False) when filtering by size
        dir_node = self.fs.traverse("/home/user", False)
        self.assertFalse(s_filter.apply(dir_node, {"min_size": 1}))

    def test_search_nodes_by_name(self):
        strategy = FilenameAndSizeSearchStrategy()
        
        # Search by name only (recursively from /home/user)
        params = {"name": "report.txt"}
        results = self.fs.search_nodes("/home/user", strategy, params)
        
        # Should find docs/report.txt and archive/report.txt
        self.assertEqual(len(results), 2)
        names = [n.get_name() for n in results]
        self.assertEqual(names.count("report.txt"), 2)

    # def test_search_nodes_by_name_and_size(self):
    #     strategy = FilenameAndSizeSearchStrategy()
        
    #     # Search by name AND min_size
    #     params = {"name": "report.txt", "min_size": 10}
    #     results = self.fs.search_nodes("/home/user", strategy, params)
        
    #     # Should ONLY find docs/report.txt (size 21). archive/report.txt is size 8.
    #     self.assertEqual(len(results), 1)
    #     self.assertIsInstance(results[0], FileNode)
    #     self.assertEqual(results[0].read_content(), "Quarterly Report Data")

class TestBenchmark5(unittest.TestCase):
    def setUp(self):
        self.fs = FileSystem()
        self.fs.addFile("/home/user/docs/readme.txt", "Data")
        self.fs.mkdir("/home/user/archive")

    def test_get_absolute_path(self):
        node = self.fs.traverse("/home/user/docs", False)
        self.assertEqual(node.get_absolute_path(), "/home/user/docs")
        
        file_node = node.get_node("readme.txt")
        self.assertEqual(file_node.get_absolute_path(), "/home/user/docs/readme.txt")
        
        self.assertEqual(self.fs.root.get_absolute_path(), "/")

    def test_delete(self):
        self.fs.delete("/home/user/docs/readme.txt")
        docs_dir = self.fs.traverse("/home/user/docs", False)
        self.assertIsNone(docs_dir.get_node("readme.txt"))
        
        with self.assertRaises(ValueError):
            self.fs.delete("/") # Should not allow deleting root

    def test_rename(self):
        self.fs.rename("/home/user/docs", "documents")
        
        # Old path should fail
        with self.assertRaises(ValueError):
            self.fs.traverse("/home/user/docs", False)
            
        # New path should work and still contain the file
        new_docs = self.fs.traverse("/home/user/documents", False)
        self.assertIsNotNone(new_docs.get_node("readme.txt"))

    def test_move(self):
        # Move readme.txt from docs to archive
        self.fs.move("/home/user/docs/readme.txt", "/home/user/archive")
        
        docs_dir = self.fs.traverse("/home/user/docs", False)
        archive_dir = self.fs.traverse("/home/user/archive", False)
        
        self.assertIsNone(docs_dir.get_node("readme.txt"))
        self.assertIsNotNone(archive_dir.get_node("readme.txt"))


class TestBenchmark4(unittest.TestCase):
    def setUp(self):
        self.fs = FileSystem()

    def test_mkdir(self):
        self.fs.mkdir("/home/user/music")
        # Verify it was created
        target = self.fs.traverse("/home/user/music", False)
        self.assertIsNotNone(target)
        self.assertEqual(target.get_name(), "music")

    def test_add_file(self):
        self.fs.addFile("/home/user/docs/resume.txt", "My Resume Content")
        
        # Verify the directory was created
        docs_dir = self.fs.traverse("/home/user/docs", False)
        self.assertIsNotNone(docs_dir)
        
        # Verify the file was created inside
        file_node = docs_dir.get_node("resume.txt")
        self.assertIsInstance(file_node, FileNode)
        self.assertEqual(file_node.read_content(), "My Resume Content")

    def test_ls(self):
        self.fs.mkdir("/folder/subfolder")
        self.fs.addFile("/folder/file1.txt", "Data")
        self.fs.addFile("/folder/file2.txt", "Data")
        
        contents = self.fs.ls("/folder")
        
        self.assertEqual(len(contents), 3)
        self.assertIn("subfolder", contents)
        self.assertIn("file1.txt", contents)
        self.assertIn("file2.txt", contents)

    def test_ls_nonexistent_path(self):
        with self.assertRaises(ValueError):
            self.fs.ls("/fake/path")


class TestBenchmark3(unittest.TestCase):
    def setUp(self):
        self.fs = FileSystem()

    def test_filesystem_initialization(self):
        self.assertEqual(self.fs.root.get_name(), "/")
        self.assertIsInstance(self.fs.root, DirectoryNode)

    def test_traverse_creates_missing_dirs(self):
        # Should create "home", then "user", then "docs"
        target_dir = self.fs.traverse("/home/user/docs", True)
        self.assertIsNotNone(target_dir)
        self.assertEqual(target_dir.get_name(), "docs")
        
        # Verify the tree structure from the root
        home_dir = self.fs.root.get_node("home")
        self.assertIsNotNone(home_dir)
        user_dir = home_dir.get_node("user")
        self.assertIsNotNone(user_dir)
        self.assertIs(user_dir.get_node("docs"), target_dir)

    def test_traverse_without_create_missing_fails(self):
        with self.assertRaises(ValueError):
            self.fs.traverse("/var/logs", False)

    def test_traverse_into_file_fails(self):
        self.fs.traverse("/home", True)
        home = self.fs.root.get_node("home")
        home.add_node(FileNode("readme.txt"))
        
        with self.assertRaises(ValueError):
            # Attempting to treat a file as a directory
            self.fs.traverse("/home/readme.txt/nested", True)
            
            
            
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