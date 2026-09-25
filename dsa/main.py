from typing import List
costs = [[8, 4, 15], [10, 7, 3], [6, 9, 12]]
costs1 = [[5, 8, 6], [19, 14, 13], [7, 5, 12], [14, 5, 9]]
costs3 = [[4, 2, 8], [7, 1, 5], [3, 9, 6]]
costs4 = [[8, 3, 12, 5], [15, 9, 4, 7]]

def min_cost(costs: List[List[int]]) -> int:
    if not costs:
        return 0    
    houses = len(costs)
    color = len(costs[0])
    
    def helper():
        dp = [[float("inf")] * color for _ in range(houses + 1)]  
        for indx in range(color):
            dp[houses][indx] = 0   
        for indx in reversed(range(houses)):
            min1 = float("inf")
            min2 = float("inf")
            pre_col = -1
            for col in range(color):
                next_val = dp[indx + 1][col]
                if next_val < min1:
                    min2 = min1
                    min1 = next_val
                    pre_col = col
                elif next_val < min2:
                    min2 = next_val
            for k in range(color):
                if k == pre_col:
                    next_val = min2
                else:
                    next_val = min1
                if next_val != float("inf"):
                    dp[indx][k] = costs[indx][k] + next_val
                else:
                    dp[indx][k] = costs[indx][k] 
        return min(dp[0])
    return helper()

print(min_cost(costs))
print(min_cost(costs1))
print(min_cost(costs3))
print(min_cost(costs4))
print(min_cost([[8]]))