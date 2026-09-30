import unittest
from elevator_models import Direction, RequestSource, Request
from elevator import Elevator
from elevator_models import Direction



class TestBenchmark2(unittest.TestCase):
    def setUp(self):
        self.elevator = Elevator(id=1, current_floor=0, capacity=2)

    def test_add_request_and_capacity_limit(self):
        self.assertTrue(self.elevator.add_request(5))
        self.assertEqual(self.elevator.direction, Direction.UP)
        self.assertIn(5, self.elevator.up_stops)

        # Fill capacity manually
        self.elevator.passengers = 2
        self.assertFalse(self.elevator.add_request(3))  # Should reject due to capacity

    def test_step_movement_up(self):
        self.elevator.add_request(2)  # Floor 2 stop
        
        # Step 1: Moves from 0 to 1
        self.elevator.step()
        self.assertEqual(self.elevator.current_floor, 1)
        self.assertEqual(self.elevator.direction, Direction.UP)

        # Step 2: Moves from 1 to 2 (arrives at stop)
        self.elevator.step()
        self.assertEqual(self.elevator.current_floor, 2)
        self.assertNotIn(2, self.elevator.up_stops)
        self.assertEqual(self.elevator.direction, Direction.IDLE)

def test_direction_reversal(self):
        # Starting at floor 0
        self.elevator.add_destination(2)  # Added to up_stops
        
        self.elevator.step()  # Moves to Floor 1 (direction: UP)
        self.assertEqual(self.elevator.current_floor, 1)
        
        # Now while at floor 1, add a destination below current floor
        self.elevator.add_destination(0)  # Added to down_stops because 0 < 1
        
        self.elevator.step()  # Moves to Floor 2 (reaches top request, up_stops cleared)
        self.assertEqual(self.elevator.current_floor, 2)
        # Having cleared up_stops, it sees down_stops has [0], so it switches direction to DOWN!
        self.assertEqual(self.elevator.direction, Direction.DOWN)

        self.elevator.step()  # Moves to Floor 1 (direction: DOWN)
        self.assertEqual(self.elevator.current_floor, 1)

        self.elevator.step()  # Moves to Floor 0 (reaches request, down_stops cleared)
        self.assertEqual(self.elevator.current_floor, 0)
        self.assertEqual(self.elevator.direction, Direction.IDLE)



class TestBenchmark1(unittest.TestCase):
    def test_enums(self):
        self.assertEqual(Direction.UP.name, "UP")
        self.assertEqual(Direction.DOWN.name, "DOWN")
        self.assertEqual(Direction.IDLE.name, "IDLE")
        
        self.assertEqual(RequestSource.INTERNAL.name, "INTERNAL")
        self.assertEqual(RequestSource.EXTERNAL.name, "EXTERNAL")

    def test_external_request(self):
        req = Request(source_floor=3, target_floor=3, direction=Direction.UP, request_source=RequestSource.EXTERNAL)
        self.assertEqual(req.source_floor, 3)
        self.assertEqual(req.direction, Direction.UP)
        self.assertEqual(req.request_source, RequestSource.EXTERNAL)

    def test_internal_request(self):
        req = Request(source_floor=2, target_floor=7, request_source=RequestSource.INTERNAL)
        self.assertEqual(req.source_floor, 2)
        self.assertEqual(req.target_floor, 7)
        self.assertEqual(req.direction, Direction.UP)
        self.assertEqual(req.request_source, RequestSource.INTERNAL)

if __name__ == '__main__':
    unittest.main()