import time
from collections import defaultdict, deque

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
    
class SlidingWindowLog:
    def __init__(self, limit, window, clock = time.monotonic):
        self.limit = limit
        self.clock = clock
        self.state = defaultdict(deque)
        self.window = window
    
    def allow(self, key = "default"):
        current_time = self.clock()
        cut_off = current_time - self.window
        timestamps = self.state[key]
        while timestamps and timestamps[0] <= cut_off:
            timestamps.popleft() 
        count = len(timestamps)
        if count >= self.limit:
            return False
        timestamps.append(current_time)
        return True
        
        
            