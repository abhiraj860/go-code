import heapq
arr = [3, 1, 4, 1, 5, 9, 2]
heapq.heapify(arr)
heapq.heappush(arr, 0)
print(arr[0])
min_element = heapq.heappop(arr)
print(arr[0])
