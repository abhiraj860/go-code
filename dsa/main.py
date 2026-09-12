def myFunc(m, n):
    return m * n
print(myFunc(3, 4))

def outer(a, b):
    c = "c"
    def inner():
        return a + b + c
    return inner()
print(outer("a", "b"))

def double(arr, val):
    def helper():
        for i, n in enumerate(arr):
            arr[i] *= 2
        # nonlocal val
        nonlocal val
        val *= 2
    helper()
    print(arr, val)
nums = [1, 2]
val = 3
print(double(nums, val))