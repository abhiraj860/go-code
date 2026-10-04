import heapq

nums = [5, 3, 2, 1, 4]
k = 2

def kthLargest(nums, k):
    arr = nums[:k] 
    heapq.heapify(arr)
    for v in nums[k:]:
        heapq.heappushpop(arr, v)   
        
    return arr[0]

print(kthLargest(nums, k))