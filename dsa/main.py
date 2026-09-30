nums = [2, 1, 5, 1, 3, 2]
k = 3

start = 0
total = 0
max_total = float("-inf")
for end in range(len(nums)):
    total += nums[end]
    if end - start + 1 == k:
        max_total = max(max_total, total)
        total -= nums[start]
        start += 1
print(max_total)