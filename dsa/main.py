n = 4
edges = [[0, 1], [1, 2], [2, 3], [3, 0], [0, 2]]

adj_List = {i:[] for i in range(n)}

for edge in edges:
    adj_List[edge[0]].append(edge[1])
    adj_List[edge[1]].append(edge[0])
print(adj_List)