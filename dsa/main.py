class Employee:
    def __init__(self, name, age):
        self.name = name
        self.age = age

e = Employee("abhiraj", 45)
print(getattr(e, "TTT", "454"))