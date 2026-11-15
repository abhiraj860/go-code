# Observer Patter
from abc import ABC, abstractmethod

class Observer(ABC):
    @abstractmethod
    def update(self): ...

class EmailObserver(Observer):
    def update(self, event):
        print("Email event received", event)
        
class SMSObserver(Observer):
    def update(self, event):
        print("SMS event received", event)
        
        
class Subject:
    def __init__(self):
        self.observers = []
        
    def subscribe(self, observer):
        self.observers.append(observer)
        
    def unsubscribe(self, observer):
        self.observers.remove(observer)
        
    def notify(self, event):
        for obs in self.observers:
            obs.update(event)
            
a = Subject()
c = EmailObserver()
a.subscribe(c)
b = SMSObserver()
a.subscribe(b)
a.notify("Hello")
a.unsubscribe(b)
a.notify("Herer")
a.unsubscribe(c)
a. notify("rer")