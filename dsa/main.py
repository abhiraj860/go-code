class Employee:
    def __init__(self, name):
        self.name = name

    def __len__(self):
        return len(self.name)

    def __call__(self):
        print("Hey I am good")
        
        
e = Employee("Hqrry")
e()
