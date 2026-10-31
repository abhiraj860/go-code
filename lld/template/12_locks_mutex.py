# LOCK: only one thread at a time in the critical section.
import threading

class Account:
    def __init__(self, id, balance):
        self.id = id
        self.balance = balance
        self.lock = threading.Lock()      # 1. one lock per resource

    def deposit(self, amount):
        with self.lock:                   # 2. lock -> work -> auto unlock
            self.balance += amount

# 3. Two locks: always lock in the same order (by id) to avoid deadlock
def transfer(src, dst, amount):
    first, second = (src, dst) if src.id < dst.id else (dst, src)
    with first.lock:
        with second.lock:
            if src.balance < amount:
                return False
            src.balance -= amount
            dst.balance += amount
            return True


if __name__ == "__main__":
    a, b = Account(1, 100), Account(2, 100)
    transfer(a, b, 30)
    print(a.balance, b.balance)   # 70 130
