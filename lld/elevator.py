from elevator_models import Direction
import threading

class Elevator:
    def __init__(self, id, current_floor = 0, capacity = 10):
        self.id = id
        self.current_floor = current_floor
        self.direction = Direction.IDLE
        self.capacity = capacity
        self.passengers = 0
        self.up_stops = set()
        self.down_stops = set()
        self._lock = threading.Lock()
        
    def add_request(self, target_floor):
        with self._lock:
            if self.passengers >= self.capacity or 0 > target_floor or target_floor > 9:
                return False
            if self.current_floor == target_floor:
                return True
            if self.current_floor > target_floor:
                self.down_stops.add(target_floor)
                if self.direction == Direction.IDLE:
                    self.direction = Direction.DOWN
            else:
                self.up_stops.add(target_floor)
                if self.direction == Direction.IDLE:
                    self.direction = Direction.UP
            return True
    
    def step(self):
        with self._lock:
            if self.direction == Direction.UP:
                self.current_floor += 1
                if self.current_floor in self.up_stops:
                    self.up_stops.remove(self.current_floor)
                if not self.up_stops:
                    if self.down_stops:
                        self.direction = Direction.DOWN
                    else:
                        self.direction = Direction.IDLE
            elif self.direction == Direction.DOWN:
                self.current_floor -= 1
                if self.current_floor in self.down_stops:
                    self.down_stops.remove(self.current_floor)
                if not self.down_stops:
                    if self.up_stops:
                        self.direction = Direction.UP
                    else:
                        self.direction = Direction.IDLE
            else:
                if self.up_stops:
                    self.direction = Direction.UP
                if self.down_stops:
                    self.direction = Direction.DOWN
                return