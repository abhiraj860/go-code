from __future__ import annotations
from enum import Enum, auto
from abc import ABC, abstractmethod

class Color(Enum):
    WHITE = auto()
    BLACK = auto()
    
    
class Piece(ABC):
    def __init__(self, color: Color):
        self.color = color
        
    def getColor(self):
        return self.color
    
    @abstractmethod
    def canMove(self, board, start: Cell, end: Cell):
        pass
    
    
class InvalidMoveException(Exception):
    def __init__(self, message: str):
        super().__init__(message)
        self.message = message
        
class Cell:
    def __init__(self, row: int, col: int, piece: Piece = None):
        self.row = row
        self.col = col
        self.piece = piece
        
    def isOccupied(self):
        return self.piece != None

    def getPiece(self):
        return self.piece

    def setPiece(self, piece):
        self.piece = piece
        
    def getRow(self):
        return self.row    
    
    def getCol(self):
        return self.col 
    
class Move:
    def __init__(self, start: Cell, end: Cell):
        self.start = start
        self.end = end
    
    def getStart(self):
        return self.start
    
    def getEnd(self):
        return self.end
    
