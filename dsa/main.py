matrix = [[0,1,0], [1, 0, 1], [0, 1, 0]]
def dfs(matrix):
    directions = [(1, 0), (0, 1), (-1, 0), (0, -1)]
    visited = set()
    def helper(row, col):
        if (row, col) in visited:
            return
        visited.add((row, col))
        print(row, col)
        for dir in directions:
            nx = row + dir[0]
            ny = col + dir[1]
            if nx >= 0 and ny >= 0 and nx < len(matrix) and ny < len(matrix[0]):
                helper(nx, ny)
    
    helper(0, 0)
dfs(matrix)