# CHAIN OF RESPONSIBILITY: pass a request along until someone handles it.
# (log levels, approval levels, ATM note denominations)
from abc import ABC, abstractmethod

# 1. Handler: holds the next link, passes along by default
class Handler(ABC):
    def __init__(self):
        self.next = None

    def set_next(self, handler):
        self.next = handler
        return handler          # lets you chain: a.set_next(b).set_next(c)

    @abstractmethod
    def handle(self, amount): ...

    def pass_on(self, amount):
        if self.next:
            self.next.handle(amount)
        elif amount > 0:
            print("cannot dispense remaining", amount)

# 2. Concrete handler: does its part, then passes the rest
class NoteHandler(Handler):
    def __init__(self, note):
        super().__init__()
        self.note = note

    def handle(self, amount):
        count, rest = divmod(amount, self.note)
        if count:
            print(f"{count} x {self.note}")
        self.pass_on(rest)


if __name__ == "__main__":
    # 3. Build the chain largest first, send to the head
    head = NoteHandler(500)
    head.set_next(NoteHandler(200)).set_next(NoteHandler(100))
    head.handle(1800)
