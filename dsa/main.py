import heapq
arr = [3, 1, 4, 1, 5, 9, 2]
arr = [-x for x in arr]
heapq.heapify(arr)
heapq.heappush(arr, -11)
print(-1 * arr[0])
maxElem = -1 * heapq.heappop(arr)