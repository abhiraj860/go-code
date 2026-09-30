from abc import ABC, abstractmethod
from elevator import Elevator
from elevator_models import Request

class ElevatorSelectionStrategy(ABC):
    @abstractmethod
    def select_elevator(elevators: list[Elevator], request: Request) -> None | Elevator:
       pass 
   
class NearestElevatorStrategy(ElevatorSelectionStrategy):
    def select_elevator(self, elevators: list[Elevator], request: Request):
        difference = float("inf")
        ele = None
        for elevator in elevators:
            if elevator.passengers >= elevator.capacity:
                continue
            diff = abs(elevator.current_floor - request.source_floor)
            if diff < difference:
                difference = diff
                ele = elevator
        return ele
