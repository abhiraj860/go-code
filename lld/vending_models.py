from enum import Enum

class Item:
    def __init__(self, name: str, price: int, code: str):
        self.name = name
        self.price = price
        self.code = code

class Coin(Enum):
    PENNY = 1
    NICKEL = 5
    DIME = 10
    QUARTER = 25
    
class Note(Enum):
    ONE = 100
    FIVE = 500 