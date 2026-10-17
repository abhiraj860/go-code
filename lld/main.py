from abc import ABC, abstractmethod
import copy

class Prototype(ABC):
    @abstractmethod
    def clone(self) -> 'Prototype':
        pass

class NPC(Prototype):
    def __init__(self, name: str, health: int, inventory: list):
        if health < 0:
            raise ValueError("Health cannot be negative")
        self.name = name
        self.health = health
        self.inventory = inventory

    def clone(self) -> 'NPC':
        return copy.deepcopy(self)        

    def add_item(self, item: str) -> None:
        self.inventory.append(item)
        return