from enum import Enum, auto

class Direction(Enum):
    UP = auto()
    DOWN = auto()
    IDLE = auto()
    
class RequestSource(Enum):
    INTERNAL = auto()
    EXTERNAL = auto()
    
class Request:
    def __init__(self, source_floor, target_floor, request_source, direction = None): 
        self.source_floor = source_floor
        self.target_floor = target_floor
        self.direction = direction if direction is not None else self.direction_calc()
        self.request_source = request_source

    def direction_calc(self): 
        if self.source_floor < self.target_floor:
            return Direction.UP
        elif self.source_floor > self.target_floor:
            return Direction.DOWN
        else:
            return Direction.IDLE