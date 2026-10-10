arr = [1, 3, 4, 6, 2, 5, 8]

n = len(arr)
prefix = [0] * (n + 1)

for i in range(1, n + 1):
    prefix[i] = arr[i - 1] + prefix[i - 1]

print(prefix)

print(prefix[5] - prefix[1])