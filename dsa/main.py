s = "abc"
print(s[0:2])

s += "def"
print(s)

print(int("123") + int("565"))
print(str(123) + str(123))
print(ord("a"))
print(ord("A"))

strings = ["abu", "cd", "eddf"]
strings.sort(key=lambda x : len(x), reverse = True)
print(strings)