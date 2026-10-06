n = 4
edges = [[0, 1], [1, 2], [2, 3], [3, 0], [0, 2]]

adj_List = {
    "1": ["2", "4"],
    "2": ["1", "3"],
    "3": ["2", "4"],
    "4": ["1", "3", "5"],
    "5": ["4"]
}

    
visited = set()
def dfs(node):
    print(node)
    visited.add(node)
    for nbr in adj_List[node]:
        if nbr not in visited:
            dfs(nbr)
    
dfs("1")