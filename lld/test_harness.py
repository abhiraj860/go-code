import unittest
from abc import ABC
from vehicle import VehicleSize, Vehicle, Motorcycle, Car, Truck
from parking_spot import ParkingSpot

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