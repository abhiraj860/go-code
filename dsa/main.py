import math
apples = [3, 6, 7]
h = 8

def minRate(apples, h):
   
    def possible(mid, apples, h):
        total = 0
        for app in apples:
            total += (app + mid - 1) // mid
        return total <= h
    
    low, high = 1, max(apples)
    ans = -1
    while low <= high:
        mid = (low + high) // 2
        if possible(mid, apples, h):
            high = mid - 1
            ans = mid
        else:
            low = mid + 1
    return ans


print(minRate( [30,11,23,4,20], 6))