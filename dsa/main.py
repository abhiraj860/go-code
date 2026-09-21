from collections import deque

adjList = {    
        0: [1, 3],    
        1: [2],
        2: [],
        3: [1, 4, 5],
        4: [5],
        5: []
} 


def toposort(adjList):
    order = []
    n = len(adjList)
    indegree = [0] * n
    for v in adjList.values():
        for a in v:
            indegree[a] += 1
            
    queue = deque([v for v in indegree if v == 0])
    while queue:
        node = queue.popleft()
        order.append(node)
        for v in adjList[node]:
            indegree[v] -= 1
            if indegree[v] == 0:
                queue.append(v)
    
    return order if n == len(order) else []


print(toposort(adjList))