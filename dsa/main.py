from typing import Protocol
from functools import singledispatchmethod

class PaymentMethod(Protocol):
    def pay(self, amount: float) -> str:
        pass

    
class CreditCard:
    def __init__(self, cardNumber: str):
        self.cardNumber = cardNumber
    
    def pay(self, amount: float) -> str:
        return f"Paid ${amount:.2f} via Credit Card ****{self.cardNumber[-4:]}"
    
class CryptoWallet:
    def __init__(self, walletAddress: str):
        self.walletAddress = walletAddress

    def pay(self, amount: float) -> str:
        return f"Paid ${amount:.2f} via Crypto Wallet {self.walletAddress}"
    
    
def process_transaction(method: PaymentMethod, amount: float) -> None:
    print(method.pay(amount))
    
class PayloadProcessor:
    @singledispatchmethod
    def parse(self, data):
        raise NotImplementedError("Unsupported payload format")
    
    @parse.register
    def _(self, data: dict):
        return f"Parsed JSON Dict: Status {data['status']}"
    
    @parse.register
    def _(self, data: str):
        return f"Parse Raw String: {data.upper()}"
   
   
   
   
  # --- INTERVIEW TEST SUITE ---
if __name__ == "__main__":
    # Test 1: Protocols & Duck Typing
    card = CreditCard("1234567890123456")
    crypto = CryptoWallet("0xABC123")

    print("--- Testing Protocol Integration ---")
    process_transaction(card, 150.0)
    process_transaction(crypto, 300.0)

    # Test 2: Single-Dispatch Method
    processor = PayloadProcessor()

    print("\n--- Testing Single Dispatch Parsing ---")
    print(processor.parse({"status": "success", "code": 200}))
    print(processor.parse("txn-101,success"))

    # Test 3: Unsupported Payload Error Handling
    try:
        processor.parse([1, 2, 3])
    except NotImplementedError as e:
        print(f"PASS: Handled unsupported payload cleanly -> {e}") 
   
   
   
    