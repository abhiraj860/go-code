class SimpleLFU:
    def __init__(self, size):
        self.capacity = size
        self.kv = {}
        self.time = 0
        
    