from parking_spot import ParkingSpot
from vehicle import Vehicle

class ParkingFloor:
    def __init__(self, floor_number: int):
        self.floor_number = floor_number
        self.spots = {}

    def add_spot(self, spot: ParkingSpot):
        self.spots[spot.spot_id] = spot

    def find_available_spot(self, vehicle: Vehicle):
        for v in self.spots.values():
            if v.can_fit_vehicle(vehicle):
                return v
        return None
            
        