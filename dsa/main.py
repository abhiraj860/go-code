import bisect
arr1 = [1, 2, 3]
arr2 = [4, 5 ,6]
t = [2, 3, 4]
z = list(zip(arr1, arr2, t))
print(list(z))
arr1 = list(map(list, zip(*z)))
print(arr1)
ind = bisect.bisect_left(arr2, 3)
print(ind)
