edges = [(0, 1), (1, 2), (1, 3), (3, 2), (3, 4)]
n = 5

indegree = [0] * n
for u, v in edges:
    indegree[v] += 1
    
print(indegree)


edges = {0: [1], 1: [2, 3], 2: [], 3: [2, 4], 4: []}
n = 5

indegree = [0] * n
for nbrs in edges.values():
    for nbr in nbrs:
        indegree[nbr] += 1
        
print(indegree)