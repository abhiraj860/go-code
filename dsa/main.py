def minMeetingRooms(start, end):
    intervals = [(start[i], end[i]) for i in range(len(start))]
    intervals.sort(key = lambda x : x[1])
    n = len(start)
    cnt = 0
    for v in range(1, n):
        curr = intervals[v][0]
        prev = intervals[v - 1][1]
        if curr < prev:
            cnt += 1
    return cnt + 1
    
minMeetingRooms([1, 10, 7], [4, 15, 10])
minMeetingRooms([2, 9, 6], [4, 12, 10])