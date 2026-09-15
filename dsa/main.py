from functools import reduce
numbers = [1, 2, 3, 4, 5]
def mySum(x, y):
    return x + y 
sum = reduce(mySum , numbers)
print(sum)