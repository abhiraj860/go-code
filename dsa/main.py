import heapq

def topK(nums):
    heap = nums[:3]
    heapq.heapify(heap)
    for num in nums[3:]:
        heapq.heappop(heap)
        heapq.heappush(heap, num)
    return heap
    

nums = [9, 3, 7, 1, -2, 6, 8]
print(topK(nums))
