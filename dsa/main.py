# Number of vertices
n = 4

# Edge list in the format (u, v, weight)
graph = [
    (0, 1, 5),
    (0, 3, 10),
    (1, 2, 3),
    (2, 3, 1),
    (3, 1, -10)  # Negative edge weight creates a negative cycle between 1, 2, and 3
]


def floyd_warshall(graph, n):
    dist = [[float('inf')] * n for _ in range(n)]
    for i in range(n):
        dist[i][i]  = 0
    for u, v, w in graph:
        dist[u][v] = w

    for k in range(n):
        for i in range(n):
            for j in range(n):
                if dist[i][j] > dist[i][k] + dist[k][j]:
                    dist[i][j] = dist[i][k] + dist[k][j]
                    
    for v in range(n):
        if dist[v][v] < 0:
            print("Negtive")
            return None

    return dist



# Running your function with this test input
result = floyd_warshall(graph, n)
print(result)