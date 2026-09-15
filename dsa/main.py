# def double(x):
#     return x * 2

# double = lambda x : x * 2
cube = lambda x : x**3
# avg = lambda x, y, z : (x + y + z) / 3

# print(avg(3, 5, 10))
# print(double(5))
# print(cube(5))

def apple(fx, value):
    return 6 + fx(value)

print(apple(lambda x : x * x, 5))

