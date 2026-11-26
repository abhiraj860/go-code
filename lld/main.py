# Problem - 02 - BookMyShow
def nextPermutation(nums: list[int]) -> None:
    """
    Do not return anything, modify nums in-place instead.
    """
    n = len(nums)
    if n == 1:
        return 
    i, j = n - 1, n - 2
    while i >= 0 and j >= 0 and nums[j] >= nums[i]:
        i -= 1
        j -= 1
    if j == -1:
        nums.reverse()
        return 
    temp = nums[j]
    k = n - 1
    while k > 0 and temp >= nums[k]:
        k -= 1
    nums[j], nums[k] = nums[k], nums[j]
    nums[j + 1:] = nums[j + 1:].reverse()
    return  

a = [1, 1]    
nextPermutation(a)
print(a)