from abc import ABC, abstractmethod

class PaymentStrategy(ABC):
    @abstractmethod        
    def pay(amount: float):
        pass

class Checkout:
    def __init__(self, paymentMethod: PaymentStrategy):
        self.paymentMethod = paymentMethod
        
    def amount_paid(self, amount):
        return self.paymentMethod.pay(amount)
        
class CreditCard(PaymentStrategy):
    def pay(self, amount):
        return f"Paid amount {amount} through credit card"
    
class PayPal(PaymentStrategy):
    def pay(self, amount):
        return f"Paid amount {amount} through PayPal"
    

ch = Checkout(CreditCard())
print(ch.amount_paid(565.565))

ch2 = Checkout(PayPal())
print(ch2.amount_paid(676.5))
        