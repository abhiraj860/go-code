import heapq

minHeap = []
heapq.heappush(minHeap, 3)
heapq.heappush(minHeap, 2)
heapq.heappush(minHeap, 4)
# print(minHeap[0])

maxHeap = []
heapq.heappush(maxHeap, -3)
heapq.heappush(maxHeap, -2)
heapq.heappush(maxHeap, -4)

# while len(maxHeap):
#     print(heapq.heappop(maxHeap) * -1)

arr = [2, 1, 8, 4, 5]
arr = [-x for x in arr]    

heapq.heapify(arr)
while arr:
    print(-1 * heapq.heappop(arr))