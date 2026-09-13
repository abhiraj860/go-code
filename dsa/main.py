import heapq

arr = []
heapq.heappush(arr, 1)
heapq.heappush(arr, 2)
heapq.heappush(arr, -9)
heapq.heappop(arr)
print(arr[0])