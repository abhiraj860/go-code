marks = [3, 5, 6, "Abhira", True, 34, 34, 34, 12, 23,14, 56]
# print(marks)
# print(type(marks))
# print(marks[0])
# print(marks[1])
# print(marks[2])
# print(marks[3])
# print(marks[4])
# print(marks[5-3])
# print(marks[-3])
# if "6" in marks:
#     print("Yes")
# else:
#     print("No")

# if "Ha" in "Harry":
#     print("Yes")

# print(marks[:7])
# print(marks[1:len(marks) + 10:3])

lst = [i * i for i in range(11) if i % 2 == 0]
lst = [i * i for i in range(10) for k in range(3)]
print(lst)