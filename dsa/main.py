from collections import deque

n = 6
adjList = {    
        0: [1, 3],    
        1: [2],
        2: [],
        3: [1, 4, 5],
        4: [5],
        5: []
    }
    
indegree = [0] * n
for v in adjList.values():
    for k in v:
        indegree[k] += 1    
    
queue = deque([])
for indx, indeg in enumerate(indegree):
    if indeg == 0:
        queue.append(indx)

order = []
while queue:
    front = queue.popleft()
    order.append(front)
    for nbr in adjList[front]:
        indegree[nbr] -= 1
        if indegree[nbr] == 0:
            queue.append(nbr)
print(order)