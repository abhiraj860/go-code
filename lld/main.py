from collections import defaultdict, OrderedDict

class LFUCache:
    def __init__(self, capacity):
        self.capacity = capacity
        self.kv = {}
        self.freq = defaultdict(OrderedDict)
        self.min_freq = 0

    def _bump(self, k):
        v, f = self.kv[k]
        del self.freq[f][k]
        
        self.freq[f + 1][k] = None
        self.kv[k][1] = (f + 1)
        if len(self.freq[f]) == 0:
            del self.freq[f]
            if self.min_freq == f:
                self.min_freq += 1
        return 
   
       
    def get(self, k):
        if k not in self.kv:
            return -1
        self._bump(k)
        return self.kv[k][0]
    
    def put(self, k, v):
        if self.capacity == 0:
            return 
        if k in self.kv:
           self._bump(k)
           self.kv[k][0] = v
           return
        if len(self.kv) == self.capacity:
            min_freq_key, _ = self.freq[self.min_freq].popitem(last=False) 
            del self.kv[min_freq_key]
        self.kv[k] = [v, 1]
        self.min_freq = 1
        self.freq[self.min_freq][k] = None
        return