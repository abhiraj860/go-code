from enum import Enum

class State(Enum):
    START = 0
    INTEGER = 1
    DOT = 2
    DECIMAL = 3
    INVALID = 5