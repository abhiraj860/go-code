import heapq

nums = [9, 3, 7, 1, -2, 6, 8]

heap = nums[:3]
heapq.heapify(heap)

for i in nums[3:]:
    heapq.heappushpop(heap, i)

print(heap)        


