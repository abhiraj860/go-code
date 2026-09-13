a = input("Enter a number: ")

try:
    num = [5, 6]
    print(num[int(a)])
except ValueError:
    print("Input is not an integer")
except IndexError:
    print("Index error")
    

