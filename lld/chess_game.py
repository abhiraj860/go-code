from chess_models import Color
from chess_board import Board
from chess_models import Move, Piece

class Player:
    def __init__(self, name: str, color: Color):
        self.name = name
        self.color = color
        
    def getColor(self):
        return self.color
    
    def getName(self):
        return self.name
    
class ChessGame:
    def __init__(self, playerWhiteName: str, playerBlackName: str):
        self.board = Board()
        self.whitePlayer = Player(playerWhiteName, Color.WHITE)
        self.blackPlayer = Player(playerBlackName, Color.BLACK)
        self.currentPlayer = self.whitePlayer

    def switchTurn(self):
        if self.currentPlayer == self.whitePlayer:
            self.currentPlayer = self.blackPlayer
            return
        self.currentPlayer = self.whitePlayer
        return
    
    def getPlayerMove(self, player: Player, move: Move):
        if player != self.currentPlayer:
            return False
        piece: Piece = move.getStart().getPiece()
        if piece.getColor() != player.getColor():
            return False
        if self.board.movePiece(move):
            self.switchTurn()
            return True
        return False
                