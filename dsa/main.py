import heapq
arr = [(3, 1), (1, 5), (4, 2), (1, 9), (5, 3), (9, 4), (2, 6)]
heapq.heapify(arr)
minEle = heapq.heappop(arr)
print(minEle)
heapq.heappush(arr, (1, 7))