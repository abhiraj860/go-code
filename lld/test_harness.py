import unittest
from abc import ABC
from vehicle import VehicleSize, Vehicle, Motorcycle, Car, Truck

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