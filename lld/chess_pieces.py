from chess_models import Piece, Color

class King(Piece):
    def canMove(self, board, start, end):
        if end.isOccupied() and end.getPiece().getColor() == self.getColor():
            return False
        if abs(end.getCol() - start.getCol()) <= 1 and abs(end.getRow() - start.getRow()) <= 1:
            return True
        return False
            
class Queen(Piece):
    def canMove(self, board, start, end):
        if end.isOccupied() and end.getPiece().getColor() == self.getColor():
            return False
        if abs(end.getCol() - start.getCol()) == abs(end.getRow() - start.getRow()):
            return True
        if abs(end.getCol() - start.getCol()) == 0 or abs(end.getRow() - start.getRow()) == 0:
            return True
        return False

class Rook(Piece):
    def canMove(self, board, start, end):
        if end.isOccupied() and end.getPiece().getColor() == self.getColor():
            return False
        if abs(end.getCol() - start.getCol()) == 0 or abs(end.getRow() - start.getRow()) == 0:
            return True
        return False

class Bishop(Piece):
    def canMove(self, board, start, end):
        if end.isOccupied() and end.getPiece().getColor() == self.getColor():
            return False
        if abs(end.getCol() - start.getCol()) == abs(end.getRow() - start.getRow()):
            return True
        return False

class Knight(Piece):
    def canMove(self, board, start, end):
        if end.isOccupied() and end.getPiece().getColor() == self.getColor():
            return False
        if abs(end.getCol() - start.getCol()) == 1 and abs(end.getRow() - start.getRow()) == 2:
            return True
        if abs(end.getCol() - start.getCol()) == 2 and abs(end.getRow() - start.getRow()) == 1:
            return True
        return False
    
    
class Pawn(Piece):
    def canMove(self, board, start, end):
        if end.isOccupied() and end.getPiece().getColor() == self.getColor():
            return False
        row_diff = end.getRow() - start.getRow()
        col_diff = end.getCol() - start.getCol()
        if not end.isOccupied() and self.getColor() == Color.WHITE and row_diff == 1 and col_diff == 0:
            return True
        if not end.isOccupied() and self.getColor() == Color.BLACK and row_diff == -1 and col_diff == 0:
            return True
        if end.isOccupied() and self.getColor() == Color.WHITE and row_diff == 1 and abs(col_diff) == 1:
            return True
        if end.isOccupied() and self.getColor() == Color.BLACK and row_diff == -1 and abs(col_diff) == 1:
            return True
        return False
                