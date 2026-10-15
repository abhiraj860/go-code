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
        # === DO NOT MODIFY ===
        node = self.root
        for char in word:
            if char not in node.children:
                node.children[char] = TrieNode()
            node = node.children[char]
        node.isEndOfWord = True
    
    def prefix(self, word):
        """
        Return a list of all words in the trie that start with the given prefix.
        """
        result = []
        node = self.root 
        for ch in word:
           if ch in node.children:
               node = node.children[ch]
           else:
               return [] 
               
        def helper(node:TrieNode, string):
            if node.isEndOfWord:
                result.append(str(string))
            if len(node.children) == 0:
                return
            for ch in node.children:
                helper(node.children[ch], string + ch)
            
        helper(node, word)
        return result

    def trie(self, words, prefix):
        # === DO NOT MODIFY ===
        self.create_trie(words)
        return self.prefix(prefix)