import unittest
from chess_models import Color, InvalidMoveException, Cell, Move, Piece
from chess_models import Color, Cell
from chess_pieces import King, Knight, Rook, Pawn
from chess_board import Board
from chess_models import Move, Color
from chess_pieces import Rook, Pawn

from chess_game import Player, ChessGame
from chess_pieces import Rook
from chess_models import Color, Move

from chess_pieces import King, Rook

class TestBenchmark5(unittest.TestCase):
    def setUp(self):
        self.board = Board()
        self.white_king = King(Color.WHITE)
        self.black_rook = Rook(Color.BLACK)
        
        self.board.setPiece(0, 0, self.white_king)
        self.board.setPiece(7, 7, self.black_rook)

    def test_is_king_in_check(self):
        # Move rook to the same row as the King
        self.board.setPiece(0, 7, self.black_rook)
        self.board.setPiece(7, 7, None)
        
        # Black Rook at 0,7 can attack White King at 0,0
        self.assertTrue(self.board.isKingInCheck(Color.WHITE))

    def test_cannot_move_into_check(self):
        # Move rook to file 1
        self.board.setPiece(7, 1, self.black_rook)
        self.board.setPiece(7, 7, None)
        
        start = self.board.getCell(0, 0)
        end = self.board.getCell(0, 1) # Moving into the Rook's line of fire
        move = Move(start, end)
        
        # The move should be rejected
        success = self.board.movePiece(move)
        self.assertFalse(success)
        
        # The board state must be rolled back to original
        self.assertEqual(self.board.getCell(0, 0).getPiece(), self.white_king)
        self.assertFalse(self.board.getCell(0, 1).isOccupied())


class TestBenchmark4(unittest.TestCase):
    def setUp(self):
        self.game = ChessGame("Alice", "Bob")
        # Manually place pieces for testing turn logic
        self.white_rook = Rook(Color.WHITE)
        self.black_rook = Rook(Color.BLACK)
        self.game.board.setPiece(0, 0, self.white_rook)
        self.game.board.setPiece(7, 7, self.black_rook)

    def test_game_initialization(self):
        self.assertEqual(self.game.whitePlayer.getName(), "Alice")
        self.assertEqual(self.game.blackPlayer.getName(), "Bob")
        self.assertEqual(self.game.currentPlayer, self.game.whitePlayer)

    def test_wrong_turn_fails(self):
        start = self.game.board.getCell(7, 7)
        end = self.game.board.getCell(7, 0)
        move = Move(start, end)
        
        # Bob tries to move Black Rook on Alice's (White's) turn
        success = self.game.getPlayerMove(self.game.blackPlayer, move)
        self.assertFalse(success)
        self.assertEqual(self.game.currentPlayer, self.game.whitePlayer)

    def test_wrong_piece_fails(self):
        start = self.game.board.getCell(7, 7)
        end = self.game.board.getCell(7, 0)
        move = Move(start, end)
        
        # Alice tries to move Black Rook on her turn
        success = self.game.getPlayerMove(self.game.whitePlayer, move)
        self.assertFalse(success)
        self.assertEqual(self.game.currentPlayer, self.game.whitePlayer)

    def test_successful_move_switches_turn(self):
        start = self.game.board.getCell(0, 0)
        end = self.game.board.getCell(0, 7)
        move = Move(start, end)
        
        # Alice moves White Rook legally
        success = self.game.getPlayerMove(self.game.whitePlayer, move)
        self.assertTrue(success)
        self.assertEqual(self.game.currentPlayer, self.game.blackPlayer)



class TestBenchmark3(unittest.TestCase):
    def setUp(self):
        self.board = Board()

    def test_board_initialization(self):
        # Check grid boundaries
        cell = self.board.getCell(0, 0)
        self.assertEqual(cell.getRow(), 0)
        self.assertEqual(cell.getCol(), 0)
        
        cell_top_right = self.board.getCell(7, 7)
        self.assertEqual(cell_top_right.getRow(), 7)
        self.assertEqual(cell_top_right.getCol(), 7)
        
        # Out of bounds
        with self.assertRaises(Exception):
            self.board.getCell(8, 0)

    def test_set_and_get_piece(self):
        rook = Rook(Color.WHITE)
        self.board.setPiece(0, 0, rook)
        
        self.assertEqual(self.board.getPiece(0, 0), rook)
        self.assertTrue(self.board.getCell(0, 0).isOccupied())

    def test_move_piece_success(self):
        rook = Rook(Color.WHITE)
        self.board.setPiece(0, 0, rook)
        
        start_cell = self.board.getCell(0, 0)
        end_cell = self.board.getCell(0, 5)
        move = Move(start_cell, end_cell)
        
        success = self.board.movePiece(move)
        
        self.assertTrue(success)
        self.assertFalse(start_cell.isOccupied())
        self.assertEqual(end_cell.getPiece(), rook)

    def test_move_piece_failure(self):
        pawn = Pawn(Color.WHITE)
        self.board.setPiece(1, 1, pawn)
        
        start_cell = self.board.getCell(1, 1)
        # Pawns cannot move backwards
        end_cell = self.board.getCell(0, 1) 
        move = Move(start_cell, end_cell)
        
        success = self.board.movePiece(move)
        
        self.assertFalse(success)
        # Piece should remain in original spot
        self.assertEqual(start_cell.getPiece(), pawn)
        self.assertFalse(end_cell.isOccupied())
        
        
class DummyBoard:
    # We will pass this as a mock board for now
    pass

class TestBenchmark2(unittest.TestCase):
    def setUp(self):
        self.board = DummyBoard()

    def test_knight_movement(self):
        knight = Knight(Color.WHITE)
        start = Cell(0, 1, knight)
        
        # Valid L-shapes
        self.assertTrue(knight.canMove(self.board, start, Cell(2, 2)))
        self.assertTrue(knight.canMove(self.board, start, Cell(2, 0)))
        
        # Invalid moves
        self.assertFalse(knight.canMove(self.board, start, Cell(1, 1))) # 1 step forward
        self.assertFalse(knight.canMove(self.board, start, Cell(0, 3))) # Straight line

    def test_cannot_capture_own_piece(self):
        rook = Rook(Color.WHITE)
        start = Cell(0, 0, rook)
        
        # Friendly piece at destination
        friendly_pawn = Pawn(Color.WHITE)
        end = Cell(0, 5, friendly_pawn)
        
        self.assertFalse(rook.canMove(self.board, start, end))

        # Enemy piece at destination
        enemy_pawn = Pawn(Color.BLACK)
        end_enemy = Cell(0, 5, enemy_pawn)
        
        self.assertTrue(rook.canMove(self.board, start, end_enemy))

    def test_pawn_movement(self):
        white_pawn = Pawn(Color.WHITE)
        black_pawn = Pawn(Color.BLACK)
        
        # White moves up (row + 1)
        start_w = Cell(1, 1, white_pawn)
        self.assertTrue(white_pawn.canMove(self.board, start_w, Cell(2, 1)))
        self.assertFalse(white_pawn.canMove(self.board, start_w, Cell(0, 1))) # Can't move backwards
        
        # Black moves down (row - 1)
        start_b = Cell(6, 1, black_pawn)
        self.assertTrue(black_pawn.canMove(self.board, start_b, Cell(5, 1)))
        
        # Diagonal capture
        enemy_piece = Rook(Color.BLACK)
        end_capture = Cell(2, 2, enemy_piece)
        self.assertTrue(white_pawn.canMove(self.board, start_w, end_capture))
        
        # Cannot move diagonal if empty
        self.assertFalse(white_pawn.canMove(self.board, start_w, Cell(2, 0)))


class DummyPiece(Piece):
    def canMove(self, board, start, end):
        return True

class TestBenchmark1(unittest.TestCase):
    def test_enums_and_exceptions(self):
        self.assertEqual(Color.WHITE.name, "WHITE")
        self.assertEqual(Color.BLACK.name, "BLACK")
        
        with self.assertRaises(InvalidMoveException):
            raise InvalidMoveException("Illegal move")

    def test_cell_operations(self):
        cell = Cell(0, 0)
        self.assertEqual(cell.getRow(), 0)
        self.assertEqual(cell.getCol(), 0)
        self.assertFalse(cell.isOccupied())
        self.assertIsNone(cell.getPiece())

        piece = DummyPiece(Color.WHITE)
        cell.setPiece(piece)
        self.assertTrue(cell.isOccupied())
        self.assertEqual(cell.getPiece().getColor(), Color.WHITE)

    def test_move_object(self):
        start_cell = Cell(1, 1)
        end_cell = Cell(2, 2)
        move = Move(start_cell, end_cell)
        
        self.assertEqual(move.getStart(), start_cell)
        self.assertEqual(move.getEnd(), end_cell)

    def test_piece_is_abstract(self):
        with self.assertRaises(TypeError):
            Piece(Color.WHITE)  # Should fail because it has an abstract method

if __name__ == '__main__':
    unittest.main()