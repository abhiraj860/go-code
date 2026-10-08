from collections import deque

adjList = {
    "1": ["2", "4"],
    "2": ["1", "3"],
    "3": ["2", "4"],
    "4": ["1", "3", "5"],
    "5": ["4"]
}

def bfs(adjList, n, node):
    visited = set()
    queue = deque([node])
    visited.add(node)
    while queue:
        front = queue.popleft()
        print(front)
        for nbr in adjList[front]:
            if nbr not in visited:
                visited.add(nbr) 
                queue.append(nbr)
    return
    
bfs(adjList, 5, "1")