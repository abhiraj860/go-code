from chess_models import Cell, InvalidMoveException, Piece, Move, Color
from chess_pieces import King
class Board:
    def __init__(self):
        self.board: list[list[Cell]] = [[Cell(i, j, None) for j in range(8)] for i in range(8)]
        
    def getCell(self, row: int, col: int) -> Cell:
        if row < 0 or row > 7 or col < 0 or col > 7:
            raise InvalidMoveException("Invalid row or col")
        return self.board[row][col]
    
    def getPiece(self, row: int, col: int) -> Piece:
        if row < 0 or row > 7 or col < 0 or col > 7:
            raise InvalidMoveException("Invalid row or col")
        return self.board[row][col].getPiece()
    
    def setPiece(self, row: int, col: int, piece: Piece):
        self.board[row][col].setPiece(piece)
        
    def movePiece(self, move: Move):
        start, end = move.getStart(), move.getEnd()
        if start.getPiece() is None:
            return False
        piece:Piece = self.getPiece(start.row, start.col)
        if piece.canMove(self, start, end):
            start_temp = start
            end_temp = end
            capture_piece = end.getPiece()
            self.setPiece(end.getRow(), end.getCol(), piece)
            self.setPiece(start.getRow(), start.getCol(), None)
            if self.isKingInCheck(end.getPiece().getColor()):
                self.setPiece(start_temp.getRow(), start_temp.getCol(), piece)
                self.setPiece(end_temp.getRow(), end_temp.getCol(), capture_piece)
                return False 
            return True
        return False
        

    def isKingInCheck(self, color: Color):
        kingPlace = None
        found = False
        for i in range(8):
            for j in range(8):
                piece = self.board[i][j].getPiece()
                if piece is not None and isinstance(piece, King) and piece.getColor() == color:
                    kingPlace = self.board[i][j]
                    found = True
                    break
            if found:
                break
            
        if kingPlace is None:
            return False
        
        for i in range(8):
            for j in range(8):
                piece = self.board[i][j].getPiece()
                if piece is not None and piece.getColor() != color and piece.canMove(self, self.board[i][j], kingPlace):
                    return True                    

        return False
            