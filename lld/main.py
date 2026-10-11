from abc import ABC, abstractmethod
from typing import List

class Observer(ABC):
    @abstractmethod
    def update(self, job_title: str) -> None:
        pass

class Subject(ABC):
    @abstractmethod
    def attach(self, observer: Observer) -> None:
        pass
        
    @abstractmethod
    def detach(self, observer: Observer) -> None:
        pass
        
    @abstractmethod
    def notify(self, job_title: str) -> None:
        pass

class JobSeeker(Observer):
    def __init__(self, name: str):
        # Must initialize a public list to track received jobs
        self.name = name
        self.notifications: List[str] = []
        
    def update(self, job_title: str) -> None:
        self.notifications.append(job_title)

class JobBoard(Subject):
    def __init__(self):
        self.observers : List[Observer] = []
        self.job_list: List[str] = []

    def attach(self, observer: Observer) -> None:
        if observer in self.observers:
            raise ValueError("Observer already present")
        self.observers.append(observer)
        
    def detach(self, observer: Observer) -> None:
        if observer not in self.observers:
            raise KeyError("Observer not in list")     
        self.observers.remove(observer)
        
    def notify(self, job_title: str) -> None:
        for observer in self.observers:
           observer.update(job_title) 
        
    def add_job(self, job_title: str) -> None:
        self.job_list.append(job_title)
        self.notify(job_title)