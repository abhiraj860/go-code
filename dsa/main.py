edges = [(0, 1), (1, 2), (1, 3), (3, 2), (3, 4)]
n = 5
adj_list = {i:[] for i in range(n)}
indegree = [0] * n
for edge in edges:
    start, end = edge[0], edge[1]
    adj_list[start].append(end)
    indegree[end] += 1
    
print(indegree)
