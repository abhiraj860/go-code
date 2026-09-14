# import math
# print(dir(math))
def kClosest(nums, target, k):
    dis = [(abs(target - v), v) for v in nums]
    dis.sort()
    return sorted([v for _, v in dis[0:k]])



print(kClosest([5, 6, 7, 8, 9], 10, 2)) 