from collections import deque

def shortest_path_grid(grid):
    m, n = len(grid), len(grid[0])
    queue = deque([])
    visited = set()
    for i in range(m):
        for j in range(n):
            if grid[i][j] == "*":
                queue.append((i, j, 0))
                visited.add((i, j))
    direct = [(1, 0), (0, 1), (-1, 0), (0, -1)]
    while queue:
        row, col, step = queue.popleft()
        if grid[row][col] == "#":
            return step
        
        for dr, dc in direct:
            nr, nc = dr + row, dc + col
            if 0 <= nr < m and 0 <= nc < n and grid[nr][nc] != "X" and (nr, nc) not in visited: 
                visited.add((nr, nc))
                queue.append((nr, nc, 1 + step))
    return -1


grid = [["X","X","X","X","X","X"],["X","*","O","O","O","X"],["X","O","O","#","O","X"],["X","X","X","X","X","X"]]
print(shortest_path_grid(grid))
grid = [["X","X","X","X","X"],["X","*","X","O","X"],["X","O","X","#","X"],["X","X","X","X","X"]]
print(shortest_path_grid(grid))
grid = [["X","X","X","X","X","X","X","X"],["X","*","O","X","O","#","O","X"],["X","O","O","X","O","O","X","X"],["X","O","O","O","O","#","O","X"],["X","X","X","X","X","X","X","X"]]
print(shortest_path_grid(grid))
grid = [["O","*"],["#","O"]]
print(shortest_path_grid(grid))
grid = [["X","*"],["#","X"]]
print(shortest_path_grid(grid))