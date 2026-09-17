class Employee:
    def __init__(self, name, id):
        self.name = name
        self.id = id

class Programmer(Employee):
    def __init__(self, name, id, lang):
        self.lang = lang
        super().__init__(name, id)
        self.lang
        
rohan = Employee("Rohan Das", "420")
harry = Programmer("Harry", "32345", "javascript")
print(harry.name)
print(harry.id)
print(harry.lang)