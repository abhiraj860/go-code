# OBSERVER: when subject changes, notify all subscribers.
from abc import ABC, abstractmethod

# 1. Observer interface
class Observer(ABC):
    @abstractmethod
    def update(self, event): ...

# 2. Concrete observers
class EmailObserver(Observer):
    def update(self, event):
        print("Email got:", event)

class SmsObserver(Observer):
    def update(self, event):
        print("SMS got:", event)

# 3. Subject: keeps a list, notifies everyone
class Subject:
    def __init__(self):
        self.observers = []

    def subscribe(self, obs):
        self.observers.append(obs)

    def unsubscribe(self, obs):
        self.observers.remove(obs)

    def notify(self, event):
        for obs in self.observers:
            obs.update(event)


if __name__ == "__main__":
    s = Subject()
    s.subscribe(EmailObserver())
    s.subscribe(SmsObserver())
    s.notify("price changed")
