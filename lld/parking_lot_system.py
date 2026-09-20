from vehicle import Vehicle
from parking_spot import ParkingSpot
from parking_floor import ParkingFloor
from parking_strategy import NearestFirstStrategy, ParkingStrategy
from fee_strategy import FlatRateFeeStrategy, FeeStrategy
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
        self.parking_strategy = NearestFirstStrategy()
        self.fee_strategy = FlatRateFeeStrategy()
    
    @classmethod
    def get_instance(cls):
        if cls._instance is None:
            cls._instance = cls()
        return cls._instance
    
    def set_parking_strategy(self, strategy: ParkingStrategy):
        self.parking_strategy = strategy
        return
    
    def set_fee_strategy(self, strategy: FeeStrategy):
        self.fee_strategy = strategy
        return
    
    def add_floor(self, floor: ParkingFloor):
        self.floors.append(floor)
    
    def park_vehicle(self, vehicle: Vehicle):
        available_spot = self.parking_strategy.find_spot(self.floors, vehicle) 
        if available_spot:
            available_spot.park_vehicle(vehicle) 
            ticketId = str(uuid.uuid4())
            ticket = ParkingTicket(ticketId, vehicle, available_spot)
            self.active_tickets[ticketId] = ticket
            return ticket         
        return None
    
    def unpark_vehicle(self, ticket_id: ParkingTicket):
        if ticket_id in self.active_tickets:
            ticket = self.active_tickets[ticket_id]
            ticket.exit_timestamp = time.time()
            cost = self.fee_strategy.calculate_fee(ticket) 
            ticket.spot.unpark_vehicle()
            self.active_tickets.pop(ticket_id, None) 
            return cost
            
        else:
            raise LookupError("Ticket not found")