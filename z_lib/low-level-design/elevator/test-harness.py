import unittest
from elevator_models import Direction, RequestSource, Request
from elevator import Elevator
from elevator_models import Direction

from dispatch_strategy import NearestElevatorStrategy
from elevator_models import Request, RequestSource, Direction

from elevator_system import ElevatorSystem
import concurrent.futures

class TestBenchmark5(unittest.TestCase):
    def setUp(self):
        self.system = ElevatorSystem()

    def test_concurrent_rush_hour(self):
        def simulate_rush_hour_user():
            # User hits the hall call button at floor 0
            self.system.request_elevator(0, Direction.UP)
            # User gets into elevator 1 and selects floor 8
            self.system.select_floor(1, 8)
            # User gets into elevator 2 and selects floor 5
            self.system.select_floor(2, 5)

        # Fire 100 simultaneous requests
        with concurrent.futures.ThreadPoolExecutor(max_workers=20) as executor:
            futures = [executor.submit(simulate_rush_hour_user) for _ in range(100)]
            concurrent.futures.wait(futures)

        # If thread-safe, no sets should be corrupted and stops should be registered
        self.assertIn(8, self.system.elevators[1].up_stops)
        self.assertIn(5, self.system.elevators[2].up_stops)
        
        # Simulate 10 time ticks concurrently with another batch of requests
        def simulate_time():
            self.system.step()
            
        with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
            step_futures = [executor.submit(simulate_time) for _ in range(10)]
            concurrent.futures.wait(step_futures)

        # Elevators should have moved without crashing from Race Conditions
        self.assertGreater(self.system.elevators[1].current_floor, 0)
        self.assertGreater(self.system.elevators[2].current_floor, 0)
        
        
class TestBenchmark4(unittest.TestCase):
    def setUp(self):
        self.system = ElevatorSystem()

    def test_initialization(self):
        self.assertEqual(len(self.system.elevators), 3)
        self.assertIsNotNone(self.system.selection_strategy)

    def test_invalid_floor_requests(self):
        self.assertFalse(self.system.request_elevator(10, Direction.UP))
        self.assertFalse(self.system.request_elevator(-1, Direction.DOWN))
        
        # Valid elevator ID, but invalid target floor
        self.assertFalse(self.system.select_floor(1, 15))

    def test_request_elevator_dispatch(self):
        # Move elevator 2 to floor 5
        self.system.elevators[2].current_floor = 5
        
        # Request from floor 4
        success = self.system.request_elevator(4, Direction.UP)
        self.assertTrue(success)
        
        # Elevator 2 should have intercepted this request
        self.assertIn(4, self.system.elevators[2].down_stops)
        self.assertEqual(self.system.elevators[2].direction, Direction.DOWN)

    def test_select_floor_internal(self):
        success = self.system.select_floor(1, 7)
        self.assertTrue(success)
        self.assertIn(7, self.system.elevators[1].up_stops)

    def test_system_step_advances_all(self):
        self.system.select_floor(1, 3) # E1 moving UP to 3
        self.system.elevators[2].current_floor = 5
        self.system.select_floor(2, 2) # E2 moving DOWN to 2
        
        self.system.step()
        
        self.assertEqual(self.system.elevators[1].current_floor, 1)
        self.assertEqual(self.system.elevators[2].current_floor, 4)
        self.assertEqual(self.system.elevators[3].current_floor, 0) # E3 remains IDLE
class TestBenchmark3(unittest.TestCase):
    def setUp(self):
        self.e1 = Elevator(id=1, current_floor=0)
        self.e2 = Elevator(id=2, current_floor=5)
        self.e3 = Elevator(id=3, current_floor=9)
        self.elevators = [self.e1, self.e2, self.e3]
        self.strategy = NearestElevatorStrategy()

    def test_select_nearest_elevator(self):
        # Request from floor 4 -> Elevator 2 (at floor 5) is closest
        req = Request(source_floor=4, target_floor=1, request_source=RequestSource.EXTERNAL, direction=Direction.DOWN)
        selected = self.strategy.select_elevator(self.elevators, req)
        self.assertEqual(selected.id, 2)

        # Request from floor 8 -> Elevator 3 (at floor 9) is closest
        req2 = Request(source_floor=8, target_floor=2, request_source=RequestSource.EXTERNAL, direction=Direction.DOWN)
        selected2 = self.strategy.select_elevator(self.elevators, req2)
        self.assertEqual(selected2.id, 3)

    def test_ignores_full_elevators(self):
        # Fill up elevator 2
        self.e2.passengers = 10
        
        # Request from floor 4. Normally e2 (at 5) is closest, but it's full.
        # e1 (at 0, distance 4) should be chosen over e3 (at 9, distance 5).
        req = Request(source_floor=4, target_floor=1, request_source=RequestSource.EXTERNAL, direction=Direction.DOWN)
        selected = self.strategy.select_elevator(self.elevators, req)
        self.assertEqual(selected.id, 1)

    def test_returns_none_if_all_full(self):
        for e in self.elevators:
            e.passengers = 10
            
        req = Request(source_floor=4, target_floor=1, request_source=RequestSource.EXTERNAL, direction=Direction.DOWN)
        selected = self.strategy.select_elevator(self.elevators, req)
        self.assertIsNone(selected)




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