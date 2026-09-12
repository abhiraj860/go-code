tup = (1, 2, 3, "green", True)
# tup[0] = 40
print(type(tup), tup)
# print(tup[0])
# print(tup[1])
print(tup[-1])



if 3 in tup:
    print("yes prensent in tup")
else:
    print("Not present")
    
tup2 = tup[1:4:2]
print(tup2)
print(tup2[1:])