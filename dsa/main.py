from typing import List
costs = [[8, 4, 15], [10, 7, 3], [6, 9, 12]]
costs1 = [[5, 8, 6], [19, 14, 13], [7, 5, 12], [14, 5, 9]]

def min_cost(costs: List[List[int]]) -> int:
    if not costs:
        return 0    
    houses = len(costs)
    color = len(costs[0])
    
    def helper(col, indx):
        if indx == houses:
            return 0
        curr = costs[indx][col]
        next = float('inf')
        for k in range(color):
            if k != col:
                next = min(next, helper(k, indx + 1))
        return curr + next
        
    result = float("inf")
    
    for col in range(color):
        result = min(result, helper(col, 0))
    return result


print(min_cost(costs))
print(min_cost(costs1))