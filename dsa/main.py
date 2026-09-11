mySet = set()
mySet.add(1)
mySet.add(2)
mySet.add(3)

print(mySet)
print(len(mySet))
print(1 in mySet)
print(2 in mySet)
print(4 in mySet)

mySet.remove(2)
print(2 in mySet)

print(set([1, 2, 3, 4, 4, 4]))
mySet = {i for i in range(7)}
print(mySet)