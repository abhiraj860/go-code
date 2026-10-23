class SimpleLFU:
    def __init__(self, capacity):
        self.capacity = capacity
        self.kv = {}
        self.time = 0
        
    def get(self, key):
        if key not in self.kv:
            return -1
        self.time += 1
        self.kv[key][1] += 1
        self.kv[key][2] = self.time
        return self.kv[key][0]
    
    def put(self, key, value):
        self.time += 1
        if self.capacity == 0:
            return
        if key in self.kv:
            self.kv[key][1] += 1
            self.kv[key][2] = self.time 
            self.kv[key][0] = value
            return
        if len(self.kv) == self.capacity:
            key1 = min(self.kv, key = lambda k: (self.kv[k][1], self.kv[k][2]), default = None)
            self.kv.pop(key1, None)
        self.kv[key] = [value, 1, self.time]            
        return
    