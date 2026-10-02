import unittest
from chess_models import Color, InvalidMoveException, Cell, Move, Piece

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