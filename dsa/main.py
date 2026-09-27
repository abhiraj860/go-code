# initialWords = ["apple", "app", "apartment", "ap", "apricot"]
initialWords = ["ball", "bath", "bat", "batter"]

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
    
    def delete(self, word: str):
        def _helper(node: Node, indx):
            if indx == len(word):
                node.eow = False
                return len(node.children) == 0
            char = word[indx]
            child = node.children.get(char, None)
            if child is None:
                return False
            should_delete = _helper(child, indx + 1)
            if should_delete:
                del node.children[char]
                
            return len(node.children) == 0 and not node.eow
        return _helper(self.root, 0)            
    
    def _helper(self, curr_node: Node, result: list[str], word: str):
        if curr_node.eow:
            result.append(word)
        for ch in curr_node.children:
            node = curr_node.children[ch]
            self._helper(node, result, word + ch)
        return
    
    def prefix(self, word: str):
        curr_node = self.root
        for ch in word:
            if ch not in curr_node.children:
                return []
            curr_node = curr_node.children[ch]
        result = []
        self._helper(curr_node, result, word)
        return result
    
trie = Trie()
for word in initialWords:
    trie.insert(word)
# print(trie.search("apartment"))
print(trie.prefix("bat"))