from elevator import Elevator
from dispatch_strategy import NearestElevatorStrategy
from elevator_models import Request, RequestSource
import threading


class ElevatorSystem:
    def __init__(self):
        self.elevators: dict[int, Elevator] = { 1: Elevator(1), 2: Elevator(2), 3: Elevator(3)}
        self.selection_strategy = NearestElevatorStrategy()
        self._lock = threading.Lock()
        
    def request_elevator(self, source_floor, direction):
        with self._lock:
            if source_floor < 0 or source_floor > 9:
                return False
            request = Request(source_floor = source_floor, target_floor = source_floor, request_source  = RequestSource.EXTERNAL, direction = direction)
            elevators = [elev for elev in self.elevators.values()]
            elevator = self.selection_strategy.select_elevator(elevators, request)
            if elevator:
                elevator.add_request(source_floor)
                return True
            return False
       
    def select_floor(self, elevator_id, target_floor):
        with self._lock:
            if target_floor < 0 or target_floor > 9:
                return False
            elevator = self.elevators.get(elevator_id, None)
            
            if elevator is None:
                return None
            return elevator.add_request(target_floor)
 
    
    def step(self):
        for elevator in self.elevators.values():
            elevator.step()
