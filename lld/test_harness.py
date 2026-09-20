import unittest
from abc import ABC
from vehicle import VehicleSize, Vehicle, Motorcycle, Car, Truck
from parking_spot import ParkingSpot
from parking_floor import ParkingFloor
from parking_lot_system import ParkingLotSystem, ParkingTicket
from parking_strategy import NearestFirstStrategy, FarthestFirstStrategy
import time

class TestBenchmark5(unittest.TestCase):
    def setUp(self):
        # Reset singleton for a clean slate
        ParkingLotSystem._instance = None
        self.system = ParkingLotSystem.get_instance()
        
        self.floor1 = ParkingFloor(1)
        self.floor1.add_spot(ParkingSpot("1-M1", VehicleSize.MEDIUM))
        
        self.floor2 = ParkingFloor(2)
        self.floor2.add_spot(ParkingSpot("2-M1", VehicleSize.MEDIUM))
        
        self.system.add_floor(self.floor1)
        self.system.add_floor(self.floor2)
        
        self.car = Car("C-1")

    def test_nearest_first_strategy(self):
        self.system.set_parking_strategy(NearestFirstStrategy())
        ticket = self.system.park_vehicle(self.car)
        
        # Should find the spot on the first floor
        self.assertIsNotNone(ticket)
        self.assertEqual(ticket.spot.spot_id, "1-M1")

    def test_farthest_first_strategy(self):
        self.system.set_parking_strategy(FarthestFirstStrategy())
        ticket = self.system.park_vehicle(self.car)
        
        # Should skip floor 1 and find the spot on the second floor
        self.assertIsNotNone(ticket)
        self.assertEqual(ticket.spot.spot_id, "2-M1")
        

class TestBenchmark4(unittest.TestCase):
    def setUp(self):
        # Reset singleton for testing purposes 
        ParkingLotSystem._instance = None
        self.system = ParkingLotSystem.get_instance()
        
        self.floor1 = ParkingFloor(1)
        self.floor1.add_spot(ParkingSpot("1-S1", VehicleSize.SMALL))
        self.system.add_floor(self.floor1)
        
        self.moto = Motorcycle("M-1")

    def test_singleton(self):
        system2 = ParkingLotSystem.get_instance()
        self.assertIs(self.system, system2)

    def test_parking_ticket_creation(self):
        spot = ParkingSpot("TEST-1", VehicleSize.SMALL)
        ticket = ParkingTicket("TKT-123", self.moto, spot)
        self.assertEqual(ticket.ticket_id, "TKT-123")
        self.assertEqual(ticket.vehicle, self.moto)
        self.assertIsNotNone(ticket.entry_timestamp)
        self.assertIsNone(ticket.exit_timestamp)

    def test_park_vehicle_system(self):
        ticket = self.system.park_vehicle(self.moto)
        self.assertIsNotNone(ticket)
        self.assertEqual(ticket.vehicle, self.moto)
        self.assertIn(ticket.ticket_id, self.system.active_tickets)
        self.assertFalse(ticket.spot.is_available())
        
        # System is full, trying to park a car should return None
        car = Car("C-1")
        no_ticket = self.system.park_vehicle(car)
        self.assertIsNone(no_ticket)


class TestBenchmark3(unittest.TestCase):
    def setUp(self):
        self.floor = ParkingFloor(1)
        self.floor.add_spot(ParkingSpot("1-S1", VehicleSize.SMALL))
        self.floor.add_spot(ParkingSpot("1-M1", VehicleSize.MEDIUM))
        self.floor.add_spot(ParkingSpot("1-M2", VehicleSize.MEDIUM))
        self.moto = Motorcycle("M-1")
        self.car = Car("C-1")

    def test_floor_initialization(self):
        self.assertEqual(self.floor.floor_number, 1)
        self.assertEqual(len(self.floor.spots), 3)
        self.assertIn("1-S1", self.floor.spots)

    def test_find_available_spot(self):
        # Should find an exact match for Car (Medium)
        spot = self.floor.find_available_spot(self.car)
        self.assertIsNotNone(spot)
        self.assertEqual(spot.spot_size, VehicleSize.MEDIUM)
        
        # Park the car to occupy the spot
        spot.park_vehicle(self.car)
        
        # Should find the second medium spot for another car
        car2 = Car("C-2")
        spot2 = self.floor.find_available_spot(car2)
        self.assertIsNotNone(spot2)
        self.assertNotEqual(spot.spot_id, spot2.spot_id)
        
        # Park second car, no medium spots left
        spot2.park_vehicle(car2)
        car3 = Car("C-3")
        self.assertIsNone(self.floor.find_available_spot(car3))
        
        # Small spot should still be available for motorcycle
        self.assertIsNotNone(self.floor.find_available_spot(self.moto))

class TestBenchmark2(unittest.TestCase):
    def setUp(self):
        self.moto = Motorcycle("M-1")
        self.car = Car("C-1")
        self.truck = Truck("T-1")

    def test_parking_spot_initialization(self):
        spot = ParkingSpot("A1", VehicleSize.MEDIUM)
        self.assertEqual(spot.spot_id, "A1")
        self.assertEqual(spot.spot_size, VehicleSize.MEDIUM)
        self.assertTrue(spot.is_available())
        self.assertIsNone(spot.parked_vehicle)

    def test_can_fit_vehicle(self):
        small_spot = ParkingSpot("S1", VehicleSize.SMALL)
        medium_spot = ParkingSpot("M1", VehicleSize.MEDIUM)

        # Strict matching
        self.assertTrue(small_spot.can_fit_vehicle(self.moto))
        self.assertFalse(small_spot.can_fit_vehicle(self.car))
        
        self.assertTrue(medium_spot.can_fit_vehicle(self.car))
        self.assertFalse(medium_spot.can_fit_vehicle(self.truck))

    def test_park_and_unpark(self):
        spot = ParkingSpot("M1", VehicleSize.MEDIUM)
        
        # Successful park
        spot.park_vehicle(self.car)
        self.assertFalse(spot.is_available())
        self.assertEqual(spot.parked_vehicle, self.car)

        # Try to park when already occupied
        with self.assertRaises(ValueError):
            spot.park_vehicle(Car("C-2"))
            
        # Try to park a vehicle that doesn't fit
        spot.unpark_vehicle()
        with self.assertRaises(ValueError):
            spot.park_vehicle(self.truck)
            
        # Unpark and verify
        self.assertTrue(spot.is_available())
        self.assertIsNone(spot.parked_vehicle)
        
        
        
class TestBenchmark1(unittest.TestCase):
    def test_vehicle_enums(self):
        self.assertTrue(hasattr(VehicleSize, 'SMALL'))
        self.assertTrue(hasattr(VehicleSize, 'MEDIUM'))
        self.assertTrue(hasattr(VehicleSize, 'LARGE'))

    def test_vehicle_abstraction(self):
        # Vehicle should be an abstract class and cannot be instantiated
        with self.assertRaises(TypeError):
            Vehicle("TEST-123", VehicleSize.SMALL)
            
    def test_concrete_vehicles(self):
        moto = Motorcycle("MOTO-111")
        car = Car("CAR-222")
        truck = Truck("TRK-333")

        self.assertEqual(moto.license_number, "MOTO-111")
        self.assertEqual(moto.size, VehicleSize.SMALL)
        
        self.assertEqual(car.license_number, "CAR-222")
        self.assertEqual(car.size, VehicleSize.MEDIUM)
        
        self.assertEqual(truck.license_number, "TRK-333")
        self.assertEqual(truck.size, VehicleSize.LARGE)

if __name__ == '__main__':
    unittest.main()