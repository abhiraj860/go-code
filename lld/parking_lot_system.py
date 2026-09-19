from vehicle import Vehicle
from parking_spot import ParkingSpot
from parking_floor import ParkingFloor
import time
import uuid

class ParkingTicket:
    def __init__(self, ticket_id: str, vehicle: Vehicle, spot: ParkingSpot):
        self.ticket_id = ticket_id
        self.vehicle = vehicle
        self.spot = spot
        self.entry_timestamp = time.time()
        self.exit_timestamp = None
        
class ParkingLotSystem:
    _instance = None
    
    def __init__(self):
        if ParkingLotSystem._instance is not None:
            raise RuntimeError("Use ParkingLotSystem.get_instance() instead!")
        self.floors = []
        self.active_tickets = {}
    
    @classmethod
    def get_instance(cls):
        if cls._instance is None:
            cls._instance = cls()
        return cls._instance
    
    def add_floor(self, floor: ParkingFloor):
        self.floors.append(floor)
    
    def park_vehicle(self, vehicle: Vehicle):
        for v in self.floors:
            available_spot = v.find_available_spot(vehicle)
            if available_spot:
                available_spot.park_vehicle(vehicle) 
                ticketId = str(uuid.uuid4())
                ticket = ParkingTicket(ticketId, vehicle, available_spot)
                self.active_tickets[ticketId] = ticket
                return ticket         
        return None