def fibo(n):
    if n <= 1:
        return n
    return fibo(n - 1) + fibo(n - 2)

# print(fibo(6))

lst = [0, 1]
for k in range(2, 10):
    lst.append(lst[-1] + lst[-2])
print(lst)