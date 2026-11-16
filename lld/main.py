from abc import ABC, abstractmethod

class Handler(ABC):
    def __init__(self):
        self.next = None

    def set_next(self, handler):
        self.next = handler
        return handler
    
    @abstractmethod
    def handle(self, amount): ...
    
    def pass_on(self, amount):
        if self.next:
            self.next.handle(amount)
        elif amount > 0:
            print("cannot dispense remaining ", amount)
            
class NoteHandler(Handler):
    def __init__(self, note):
        super().__init__()
        self.note = note

    def handle(self, amount):
        count, rest = divmod(amount, self.note)
        if count:
            print(f"{count} X {self.note}")
        self.pass_on(rest)
        
head = NoteHandler(500)
head.set_next(NoteHandler(200)).set_next(NoteHandler(100))

head.handle(1800)