costs = [[8, 4, 15], [10, 7, 3], [6, 9, 12]]


def min_cost(costs):
    houses = len(costs) 
    if not houses:
        return 0
    color = len(costs[0])
    dp = {}
    if not color:
        return 0 
    def helper(i, j):
        if i == houses:
            return 0
        if (i, j) in dp:
            return dp[(i, j)]
           
        minm = float("inf") 
        for k in range(color):
            if k != j:
                minm = min(minm, costs[i][j] + helper(i + 1, k))
        dp[(i, j)] = minm
        return minm
    return min(helper(0, 0), helper(0, 1), helper(0, 2))    
    
    
print(min_cost(costs))
costs1 = [[5, 8, 6], [19, 14, 13], [7, 5, 12], [14, 5, 9]]
print(min_cost(costs1))