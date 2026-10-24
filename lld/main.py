import time
from collections import defaultdict

class FixedWindowCounter:
    def __init__(self, limit, window, clock = time.monotonic):
        self.limit = limit
        self.window = window
        self.clock = clock
        self.counts = defaultdict(int) 
        
    def allow(self, key:str = "default"):
        current_number =  self.clock() // self.window
        current = self.counts[(key, current_number)]
        if current >= self.limit:
            return False
        self.counts[(key, current_number)] += 1
        return True
        
            