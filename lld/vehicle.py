from enum import Enum, auto
from abc import ABC, abstractmethod

class VehicleSize(Enum):
    SMALL = auto()
    MEDIUM = auto()
    LARGE = auto()
    
class Vehicle(ABC):
    @abstractmethod
    def __init__(self, license_number: str, size: VehicleSize):
        self.license_number = license_number
        self.size = size

class Motorcycle(Vehicle):
    def __init__(self, license_number: str):
        super().__init__(license_number, VehicleSize.SMALL)


class Car(Vehicle):
    def __init__(self, license_number: str):
        super().__init__(license_number, VehicleSize.MEDIUM)

class Truck(Vehicle):
    def __init__(self, license_number: str):
        super().__init__(license_number, VehicleSize.LARGE)
        
