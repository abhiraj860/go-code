initialWords = ["apple", "app", "apartment"]

class Node:
    def __init__(self, char: str):
        self.char = char
        self.eow = False
        self.children: dict[str, Node] = {}
        
class Trie:
    def __init__(self):
        self.root = Node("")
        
    def insert(self, string: str):
        curr_node = self.root
        for ch in string:
            if ch in curr_node.children:
                curr_node = curr_node.children[ch]
            else:
                new_node = Node(ch)
                curr_node.children[ch] = new_node
                curr_node = curr_node.children[ch]
        curr_node.eow = True
        return
    
    def search(self, string: str):
        curr_node = self.root
        for ch in string:
            if ch in curr_node.children:
                curr_node = curr_node.children[ch]
            else:
                return False
        return curr_node.eow == True
    
    def starts_with(self, string: str):
        curr_node = self.root
        for ch in string:
            if ch in curr_node.children:
                curr_node = curr_node.children[ch]
            else:
                return False
        return True
                
    
trie = Trie()
for word in initialWords:
    trie.insert(word)
print(trie.search("app"))
print(trie.search("apple"))
print(trie.search("a"))
print(trie.starts_with("a"))