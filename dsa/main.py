costs = [[8, 4, 15], [10, 7, 3], [6, 9, 12]]


def min_cost(costs):
    houses = len(costs) 
    if not houses:
        return 0
    if houses == 1:
        return min(costs[0])
    color = len(costs[0])
    if not color:
        return 0 
    
    def helper():
        dp = [[float("inf")] * color for _ in range(houses + 1)]
        for v in range(color):
            dp[houses][v] = 0
            
        for i in reversed(range(houses)):    
            for j in range(color):
                minm = float("inf") 
                for k in range(color):
                    if k != j:
                        minm = min(minm, costs[i][j] + dp[i + 1][k])
                dp[i][j] = minm
        return min(dp[0])
    return helper()  
    
    
print(min_cost(costs))
costs1 = [[5, 8, 6], [19, 14, 13], [7, 5, 12], [14, 5, 9]]
print(min_cost(costs1))