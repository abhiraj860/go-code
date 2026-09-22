from collections import deque
import heapq

def my_bfs(graph, start, target):
    """
    Input: graph (dict of lists), start (str), target (str)
    Return: The shortest distance (number of edges) as an integer.
    """
    queue = deque([start])
    visited = set([start])
    dist = {start: 0}
    while queue:
        front = queue.popleft()
        for nbr in graph[front]:
            if nbr not in visited:
                queue.append(nbr)
                visited.add(nbr)
                dist[nbr] = dist[front] + 1
    
    return dist[target]

def my_dijkstra(graph, start, target):
    """
    Input: graph (dict of dicts), start (str), target (str)
    Return: The shortest distance as an integer or float.
    """
    dist = {i:float("inf") for i in graph}
    dist[start] = 0
    pq = [(0, start)]
    heapq.heapify(pq)
    while pq:
        currDist, node = heapq.heappop(pq)
        for nb, wt in graph[node].items():     
            if currDist + wt > dist[nb]:
                continue
            dist[nb] = currDist + wt
            heapq.heappush(pq, (dist[nb], nb))
    
    return dist[target]

def my_bellman_ford(vertices, edges, start, target):
    """
    Input: vertices (list), edges (list of tuples: (u, v, weight)), start, target
    Return: The shortest distance as an integer or float.
    """
    dist = {i:float("inf") for i in vertices} 
    dist[start] = 0 
    for _ in range(len(dist) - 1):
        
        for edge in edges:
            
            u, v, wt = edge
            if dist[u] + wt < dist[v]:
                dist[v] = dist[u] + wt
    return dist[target]
    
def my_floyd_warshall(matrix):
    """
    Input: 2D list (adjacency matrix) with float('infinity') for no edge
    Return: 2D list of shortest distances between all pairs
    """
    n = len(matrix)
    for k in range(n):
        for i in range(n):
            for j in range(n):
                if matrix[i][j] > matrix[i][k] + matrix[k][j]:
                    matrix[i][j] = matrix[i][k] + matrix[k][j]
    return matrix

def my_dag_shortest_path(graph, start, target):
    """
    Input: graph (dict of dicts), start (str), target (str)
    Return: The shortest distance as an integer or float.
    """
    pass


# ==========================================
# 2. TEST HARNESS (DO NOT MODIFY BELOW)
# ==========================================

def run_tests():
    INF = float('infinity')
    score = 0
    total = 5

    print("--- Running Tests ---\n")

    # 1. BFS Test
    bfs_graph = {
        'A': ['B', 'C'], 'B': ['A', 'D', 'E'], 'C': ['A', 'F'],
        'D': ['B'], 'E': ['B', 'F'], 'F': ['C', 'E', 'G'], 'G': ['F']
    }
    try:
        # Path is A -> C -> F -> G (3 edges)
        ans = my_bfs(bfs_graph, 'A', 'G')
        assert ans == 3
        print("✅ BFS: PASS")
        score += 1
    except AssertionError:
        print(f"❌ BFS: FAIL (Expected 3, got {ans})")
    except Exception as e:
        print(f"❌ BFS: ERROR ({e})")

    # 2. Dijkstra Test
    dijkstra_graph = {
        'A': {'B': 4, 'C': 2}, 'B': {'C': 5, 'D': 10}, 
        'C': {'E': 3}, 'E': {'D': 4}, 'D': {}
    }
    try:
        ans = my_dijkstra(dijkstra_graph, 'A', 'D')
        assert ans == 9
        print("✅ Dijkstra: PASS")
        score += 1
    except AssertionError:
        print(f"❌ Dijkstra: FAIL (Expected 9, got {ans})")
    except Exception as e:
        print(f"❌ Dijkstra: ERROR ({e})")

    # 3. Bellman-Ford Test
    bf_vertices = ['A', 'B', 'C', 'D', 'E']
    bf_edges = [
        ('A', 'B', -1), ('A', 'C', 4), ('B', 'C', 3), 
        ('B', 'D', 2), ('B', 'E', 2), ('D', 'B', 1), 
        ('D', 'C', 5), ('E', 'D', -3)
    ]
    try:
        ans = my_bellman_ford(bf_vertices, bf_edges, 'A', 'D')
        assert ans == -2
        print("✅ Bellman-Ford: PASS")
        score += 1
    except AssertionError:
        print(f"❌ Bellman-Ford: FAIL (Expected -2, got {ans})")
    except Exception as e:
        print(f"❌ Bellman-Ford: ERROR ({e})")

    # 4. Floyd-Warshall Test
    fw_matrix = [
        [0,   5,  INF, 10],
        [INF, 0,    3, INF],
        [INF, INF,  0,   1],
        [INF, INF, INF,  0]
    ]
    expected_fw = [
        [0, 5, 8, 9],
        [INF, 0, 3, 4],
        [INF, INF, 0, 1],
        [INF, INF, INF, 0]
    ]
    try:
        ans = my_floyd_warshall(fw_matrix)
        assert ans == expected_fw
        print("✅ Floyd-Warshall: PASS")
        score += 1
    except AssertionError:
        print("❌ Floyd-Warshall: FAIL (Matrix did not match expected output)")
    except Exception as e:
        print(f"❌ Floyd-Warshall: ERROR ({e})")

    # 5. DAG Shortest Path Test
    dag_graph = {
        'A': {'B': 2, 'C': 6}, 'B': {'D': 4, 'E': 2}, 
        'C': {'E': 1}, 'D': {'F': 2}, 'E': {'F': 5}, 'F': {}
    }
    try:
        ans = my_dag_shortest_path(dag_graph, 'A', 'F')
        assert ans == 8
        print("✅ DAG Shortest Path: PASS")
        score += 1
    except AssertionError:
        print(f"❌ DAG Shortest Path: FAIL (Expected 8, got {ans})")
    except Exception as e:
        print(f"❌ DAG Shortest Path: ERROR ({e})")

    print(f"\n--- Final Score: {score}/{total} ---")

if __name__ == "__main__":
    run_tests()