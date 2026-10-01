intervals = [(10,12),(6,9),(13,15)]
intervals.sort(key = lambda x : x[0])
for v in range(1, len(intervals)):
    if intervals[v][0] < intervals[v - 1][1]:
        print(False)
print(True)
    