arr = [3, 2, 1, 5]
b = [[3, 4], [5, 6]]
m = list(reversed(b.pop()))
print(m)
arr.extend(b)
print(arr)