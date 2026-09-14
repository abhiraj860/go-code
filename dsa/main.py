import heapq

def kClosest(nums, target, k):
    arr = [(-abs(target - v), v) for v in nums[0:k]]  
    heapq.heapify(arr)  
    for v in nums[k:]:
        dist = abs(v - target)
        if dist < -arr[0][0]:
            heapq.heappushpop(arr, (-dist, v))
    return sorted([v for _, v in arr])

print(kClosest([-1, 0, 1, 4, 6], 1, 3))
