import unittest
from chess_models import Color, InvalidMoveException, Cell, Move, Piece
from chess_models import Color, Cell
from chess_pieces import King, Knight, Rook, Pawn

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