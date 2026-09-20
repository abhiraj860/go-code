from abc import ABC, abstractmethod
from vehicle import Vehicle
from parking_floor import ParkingFloor
from typing import List

class ParkingStrategy(ABC):
    @abstractmethod
    def find_spot(self, floors: List[ParkingFloor], vehicle: Vehicle):
        pass
    
class NearestFirstStrategy(ParkingStrategy):
    def find_spot(self, floors: List[ParkingFloor], vehicle: Vehicle):
        for floor in floors:
            spot = floor.find_available_spot(vehicle)
            if spot:
                return spot
        return None
    
class FarthestFirstStrategy(ParkingStrategy):
    def find_spot(self, floors: List[ParkingFloor], vehicle: Vehicle):
        for floor in reversed(floors):
            spot = floor.find_available_spot(vehicle)
            if spot:
                return spot
        return None
        
        