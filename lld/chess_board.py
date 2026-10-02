from chess_models import Cell, InvalidMoveException, Piece, Move
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
            self.setPiece(end.getRow(), end.getCol(), piece)
            self.setPiece(start.getRow(), start.getCol(), None)
            return True
        return False
        

