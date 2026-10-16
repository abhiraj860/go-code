import heapq

class MedianFinder:
    def __init__(self):
        self.lo = []
        self.high = []
        
    def add_num(self, num: int) -> None:
        heapq.heappush(self.lo, -num)
        heapq.heappush(self.high, -heapq.heappop(self.lo))
        if len(self.high) > len(self.lo):
            heapq.heappush(self.lo, -heapq.heappop(self.high))
            
    def find_median(self) -> float:
        if len(self.lo) > len(self.high):
            return -self.lo[0]
        return (-self.lo[0] + self.high[0]) / 2