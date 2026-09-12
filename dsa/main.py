tup1 = (1, 2, 3)
tup2 = (4, 5, 6, 8, 9, 9, 9)
tot = tup1 + tup2
print(tot.count(2))
# print(tot.index(9, 0, len(tot)))
print(tot.index(6, 0, -1))
# print(tot)