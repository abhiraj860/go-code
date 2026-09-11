myMap = {}
myMap["alice"] = 88
myMap["bob"] = 77
print(myMap)
print(len(myMap))
myMap["alice"] = 80
print(myMap)
print("alice" in myMap)
myMap = {i : 2 * i for i in range(3)}
print(myMap)
myMap = {"alice": 90, "bob": 70}
for key in myMap:
    print(key, myMap[key])
for val in myMap.values():
    print(val)
for key, val in myMap.items():
    print(key, val)
    
a = [1, 2, 4]
for i, v in enumerate(myMap):
    print(i, v)