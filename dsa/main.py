def search(nums, target):
    low, high = 0, len(nums) - 1
    while low <= high:
        mid = (low + high) // 2
        if nums[mid] == target:
            return mid
        if nums[low] <= nums[mid]:
            if nums[low] <= target and nums[mid] > target:
                high = mid - 1
            else:
                low = mid + 1
        if nums[mid] <= nums[high]:
            if nums[mid] < target and target <= nums[high]:
                low = mid + 1
            else:
                high = mid - 1
    return -1

print(search([8, 9, 10, 12, 16, 17, 1, 2, 3], 3))        