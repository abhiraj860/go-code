def employeeFreeTime(schedule):
    interval = [v for inte in schedule for v in inte]
    interval.sort()
    print(interval)
    busy_res = []
    for v in interval:
        if len(busy_res) == 0 or busy_res[-1][1] < v[0]:
            busy_res.append(v)
        else:
            busy_res[-1][1] = max(busy_res[-1][1], v[1])
    res = []
    for i in range(1, len(busy_res)):
        res.append([busy_res[i - 1][1], busy_res[i][0]])
    return res
    
print(employeeFreeTime([[[2,4],[7,10]],[[1,5]],[[6,9]]]))
        