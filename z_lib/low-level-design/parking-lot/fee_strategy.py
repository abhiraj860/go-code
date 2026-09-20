from abc import ABC, abstractmethod
from vehicle import VehicleSize
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from parking_lot_system import ParkingTicket

class FeeStrategy(ABC):
    @abstractmethod
    def calculate_fee(self, ticket: 'ParkingTicket'):
        pass

class FlatRateFeeStrategy(FeeStrategy):
    def calculate_fee(self, ticket: "ParkingTicket"):
        hours_parked =  (ticket.exit_timestamp - ticket.entry_timestamp) / 3600
        return 10 * hours_parked
    
class VehicleBasedFeeStrategy(FeeStrategy):
    def calculate_fee(self, ticket: "ParkingTicket"):
        hours_parked = (ticket.exit_timestamp - ticket.entry_timestamp) / 3600
        vehicle_size = ticket.vehicle.size
        match vehicle_size:
            case VehicleSize.SMALL:
                return 10 * hours_parked
            case VehicleSize.MEDIUM:
                return 20 * hours_parked
            case VehicleSize.LARGE:
                return 30 * hours_parked
            case _:
                raise ValueError("No a correct size")
        return
