from typing import List
schedule = [[[2,4],[7,10]],[[1,5]],[[6,9]]]
def employeeFreeTime(schedule: List[List[List[int]]]) -> List[List[int]]:
    intervals = [item for sublist in schedule for item in sublist]
    intervals.sort(key = lambda x : x[0])
    n = len(intervals)
    result = []
    for i in range(n):
        if not result or result[-1][1] < intervals[i][0]:
            result.append(intervals[i])
        else:
            result[-1][1] = max(result[-1][1], intervals[i][1])
            
    ans = []
    for k in range(1, len(result)):
        ans.append([result[k - 1][1], result[k][0]])
    return ans


print(employeeFreeTime(schedule))