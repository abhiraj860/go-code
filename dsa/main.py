class TrieNode:
    def __init__(self):
        self.children = {}
        self.isEndOfWord = False

class Solution:
    def create_trie(self, words):
        # === DO NOT MODIFY ===
        self.root = TrieNode()
        for word in words:
            self.insert(word)

    def insert(self, word):
        node = self.root
        for char in word:
            if char not in node.children:
                node.children[char] = TrieNode()
            node = node.children[char]
        node.isEndOfWord = True
    
    def search(self, word):
        node = self.root
        for ch in word:
            if ch in node.children:
                node = node.children[ch]
            else:
                return False
        return node.isEndOfWord
         

    def starts_with(self, prefix):
        node = self.root
        for ch in prefix:
            if ch in node.children:
                node = node.children[ch]
            else:
                return False
        return True

    def delete(self, word):
        node = self.root() 
        def _helper(indx, node: TrieNode):
            if indx == len(word):
               node.isEndOfWord = False
               return len(node.children) == 0
            ch = word[indx]
            if ch not in node.children:
                return False
            can_delete = _helper(indx + 1, node.children[ch])
            if can_delete:
                del node.children[ch] 
            return not node.isEndOfWord and (len(node.children) == 0) 
        return _helper(0, node)
        
        
        
        
        
        
    def trie(self, initialWords, commands):
        # === DO NOT MODIFY ===
        self.create_trie(initialWords)

        output = []
        for command, word in commands:
            if command == "search":
                output.append(self.search(word))
            elif command == "startsWith":
                output.append(self.starts_with(word))
            elif command == "delete":
                self.delete(word)
        return output