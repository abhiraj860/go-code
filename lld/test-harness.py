import unittest
from elevator_models import Direction, RequestSource, Request

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