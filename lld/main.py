from enum import Enum, auto

class Status(Enum):
    PENDING = auto()
    IN_PROGRESS = auto()
    COMPLETED = auto()
    FAILED = auto()
    
current = Status.FAILED

print(current.name)
    