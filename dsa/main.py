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
        prev = [0] * color
        for i in reversed(range(houses)):
            min1 = float("inf")
            min2 = float("inf")
            min_idx = -1
            for c in range(color):
                if prev[c] < min1:
                    min2 = min1
                    min1 = prev[c]
                    min_idx = c
                elif prev[c] < min2:
                    min2 = prev[c]
            curr = [0] * color
            for j in range(color):
                if j != min_idx:
                    curr[j] = costs[i][j] + min1
                else:
                    curr[j] = costs[i][j] + min2
            prev = curr
        return min(prev)
   

    return helper()  
    
    
print(min_cost(costs))
costs1 = [[5, 8, 6], [19, 14, 13], [7, 5, 12], [14, 5, 9]]
print(min_cost(costs1))