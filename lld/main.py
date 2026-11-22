import math
from enum import Enum, auto
from abc import ABC, abstractmethod
from collections import defaultdict

class VehicleType(Enum):
    MOTORCYCLE = auto()
    CAR = auto()
    TRUCK = auto()
    
class Vehicle:
    def __init__(self, plate_number, vehicle_type):
        self.plate_number = plate_number
        self.vehicle_type = vehicle_type
        
class Pricing(ABC):
    @abstractmethod
    def calc_price(self, entry_time, exit_time, vehicle_type): ...

class HourlyPricing(Pricing):
    def __init__(self, rates_by_type):
        self.rates_by_type = rates_by_type

    def calc_price(self, entry_time, exit_time, vehicle_type):
        return math.ceil((exit_time - entry_time) / 60) * self.rates_by_type[vehicle_type]
        
class FlatPricing(Pricing):
    def __init__(self, fees_by_type):
        self.fees_by_type = fees_by_type

    def calc_price(self, entry_time, exit_time, vehicle_type):
        return self.fees_by_type[vehicle_type]
    
class Payment(ABC):
    @abstractmethod
    def pay(price):...
    
class CashPayment(Payment):
    def __init__(self, cash_given):
        self.cash_given = cash_given
    def pay(self, price):
        if self.cash_given >= price:
            return True 
        return None
    
class CardPayment(Payment):
    def __init__(self, card_number):
        self.card_number = card_number
    def pay(self, price):
        return True
    
class ParkingLot:
    def __init__(self, pricing: Pricing):
        self.pricing = pricing
        self.floor_spot: dict[int, list[Spot]] = {}
        self.ticket_code_ticket: dict[str: Ticket] = {}
        self.cnt = 0
            
    def add_floor(self,floor_number: int):
        if floor_number in self.floor_spot:
            return False
        self.floor_spot[floor_number] = []
        return True
    
    def add_spot(self, floor_number, code, vehicle_type):
        if floor_number not in self.floor_spot:
            return False
        self.floor_spot[floor_number].append(Spot(code, vehicle_type))
        return True
    
    def check_vehicle(self, vehicle: Vehicle):
        for ticket in self.ticket_code_ticket.values(): 
            if ticket.get_spot().get_vehicle_code() == vehicle.plate_number:
                return True
        return None
    
    def enter(self, vehicle: Vehicle, entry_time):
        if self.check_vehicle(vehicle):
            return None
        cnt = self.get_availability()[vehicle.vehicle_type]
        if cnt <= 0:
            return None
        spot = self.get_available_spot(vehicle)
        spot.occupy(vehicle) 
        self.cnt += 1
        code = "T" + str(self.cnt) 
        t = Ticket(code, spot, entry_time)    
        self.ticket_code_ticket[code] = t 
        return t 
          
    def exit(self, ticket_id, payment:Payment, exit_time):
        if ticket_id not in self.ticket_code_ticket:
            return None
        ticket:Ticket = self.ticket_code_ticket[ticket_id]
        spot:Spot = ticket.get_spot()
        entry_time = ticket.get_time()
        price = self.pricing.calc_price(entry_time, exit_time, spot.get_vehicle_type())
        if payment.pay(price):
            del self.ticket_code_ticket[ticket_id]
            spot.release()
            r1 = Receipt(price, spot.get_code()) 
            return r1
        return None 

    
    def get_available_spot(self, vehicle: Vehicle):
        for spots in self.floor_spot.values():
            for spot in spots:
                if not spot.is_occupied() and vehicle.vehicle_type == spot.get_vehicle_type():
                    return spot
        return None
                    
    
    def get_availability(self):
        vehicle_type_number = defaultdict(int)
        for spots in self.floor_spot.values():
            for spot in spots:
                if not spot.is_occupied():   
                    vehicle_type_number[spot.get_vehicle_type()] += 1
        return vehicle_type_number 
        
    def set_pricing(self, pricing: Pricing):
        self.pricing = pricing

class Spot:
    def __init__(self, code, vehicle_type):
        self.code = code
        self.vehicle_type = vehicle_type
        self.vehicle = None
    
    def get_code(self):
        return self.code
    
    def get_vehicle_type(self):
        return self.vehicle_type

    def occupy(self, vehicle: Vehicle):
        if self.vehicle:
            return False
        self.vehicle = vehicle
        return True
    
    def get_vehicle_code(self):
        if self.vehicle:
            return self.vehicle.plate_number        
        return None

    def release(self):
        self.vehicle = None
    
    def is_occupied(self):
        return self.vehicle is not None

class Ticket:
    def __init__(self, id, spot, time):
        self.id = id
        self.spot = spot
        self.time = time
    
    def get_ticket_id(self):
        return self.id
    
    def get_spot(self):
        return self.spot
   
    def get_time(self):
        return self.time 
    
class Receipt:
    def __init__(self, fee, spot_code):
        self.fee = fee
        self.spot_code = spot_code
    
    def get_fee(self):
        return self.fee
    
    def get_spot_code(self):
        return self.spot_code
    
lot = ParkingLot(HourlyPricing({VehicleType.MOTORCYCLE: 10, VehicleType.CAR: 20, VehicleType.TRUCK: 50}))
print(lot.add_floor(1))
print(lot.add_floor(1))
lot.add_spot(1, "F1-C1", VehicleType.CAR)
lot.add_spot(1, "F1-C2", VehicleType.CAR)
lot.add_spot(1, "F1-M1", VehicleType.MOTORCYCLE)
print(lot.add_spot(2, "F2-C1", VehicleType.CAR))
print(lot.get_availability()[VehicleType.CAR])
t1 = lot.enter(Vehicle("KA-01", VehicleType.CAR), 0)
t2 = lot.enter(Vehicle("KA-02", VehicleType.CAR), 30)
print(t1.get_ticket_id(), t1.get_spot().get_code())
print(t2.get_ticket_id(), t2.get_spot().get_code())
print(lot.enter(Vehicle("KA-03", VehicleType.CAR), 40))
print(lot.enter(Vehicle("TR-01", VehicleType.TRUCK), 40))
print(lot.get_availability()[VehicleType.CAR])
print(lot.exit("T1", CashPayment(30), 130))
print(lot.get_availability()[VehicleType.CAR])
r1 = lot.exit("T1", CashPayment(100), 130)
print(r1.get_fee(), r1.get_spot_code())
print(lot.exit("T1", CardPayment("4111"), 140))
print(lot.get_availability()[VehicleType.CAR])
t3 = lot.enter(Vehicle("MH-09", VehicleType.MOTORCYCLE), 0)
print(lot.exit(t3.get_ticket_id(), CardPayment("411"), 10).get_fee())
print(lot.enter(Vehicle("KA-02", VehicleType.CAR), 50))
lot.set_pricing(FlatPricing({VehicleType.MOTORCYCLE: 5, VehicleType.CAR: 100, VehicleType.TRUCK: 300}))
print(lot.exit("T2", CardPayment("4111"), 500).get_fee())