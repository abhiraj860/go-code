# Graph configuration
n = 5
start = 0

# Edge list: (u, v, weight)
edges = [
    (0, 1, -1),
    (0, 2, 4),
    (1, 2, 3),
    (1, 3, 2),
    (1, 4, 2),
    (3, 2, 5),
    (3, 1, 1),
    (4, 3, -3)
]

def bellman_ford(edges, n, start):
    dist = [float("inf")] * n
    dist[start]= 0
    for v in range(n - 1):
        for u, v, w in edges:
            if dist[u] + w < dist[v]:
                dist[v] = dist[u] + w
    for u, v, w in edges:
        if dist[u] + w < dist[v]:
            return None
    return dist

# Running your function
result = bellman_ford(edges, n, start)
print("Distances from start node:", result)