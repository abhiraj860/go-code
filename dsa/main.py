import heapq

def kClosest(nums, target, k):
    arr = [(-1 * abs(target - v), v) for v in nums[0:k]]  
    heapq.heapify(arr)  
    for v in nums[k:]:
        dist = abs(v - target)
        topDist, _ = arr[0]
        if dist < -1 * topDist:
            heapq.heappush(arr, (-1 * dist, v)) 
    arr = list(arr)
    return sorted([-1 * v for _, v in arr])

print(kClosest([-1, 0, 1, 4, 6], 1, 3))
