class HitCounter:
    def __init__(self):
        self.time = [0] * 300
        self.hits = [0] * 300
        
    def hit(self, timestamp):
        indx_time = timestamp % 300
        if timestamp != self.time[indx_time]:
            self.hits[indx_time] = 1
            self.time[indx_time] = timestamp
        else:
            self.hits[indx_time] += 1
    
    def get_hits(self, timestamp):
        return sum(v for indx, v in enumerate(self.hits) if timestamp - self.time[indx] < 300)
    
    
counter = HitCounter()
counter.hit(1)
counter.hit(2)
counter.hit(3)
print(counter.get_hits(4))
counter.hit(300)
print(counter.get_hits(300))
print(counter.get_hits(301))