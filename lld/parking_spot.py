from vehicle import Vehicle, VehicleSize

class ParkingSpot:
    def __init__(self, spot_id: str, spot_size: VehicleSize):
        self.spot_id = spot_id
        self.spot_size = spot_size
        self.parked_vehicle = None
    
    def is_available(self):
        return True if not self.parked_vehicle else False
    
    def can_fit_vehicle(self, vehicle: Vehicle):
        return (vehicle.size == self.spot_size) and (self.is_available())
    
    def park_vehicle(self, vehicle: Vehicle):
        if not self.is_available():
            raise ValueError("Space is not available")
        if vehicle.size != self.spot_size:
            raise ValueError("Vehicle and spot size do not match")
        self.parked_vehicle = vehicle
        return
    
    def unpark_vehicle(self):
        self.parked_vehicle = None
        return