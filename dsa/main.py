class BankAccount:
    def __init__(self, accountHolder: str, balance: float, accountNumber: str):
        self.accountHolder = accountHolder
        self.balance = balance
        self.accountNumber = accountNumber

    def deposit(self, amount:float) -> None:
        self.balance += amount
    
    def withdraw(self, amount:float) -> None:
        self.balance -= amount
    
    def __repr__(self) -> str:
        return f"BankAccount('{self.accountHolder}', {self.balance}, '{self.accountNumber}')"
    
    def __str__(self) -> str:
        return f"Account #{self.accountNumber} - {self.accountHolder}: ${self.balance:.2f}"
    
    
acc = BankAccount("Alice Smith", 500.0, "ACC-101")
acc.deposit(150.0)
acc.withdraw(50.0)
print(acc)
print(repr(acc))
        