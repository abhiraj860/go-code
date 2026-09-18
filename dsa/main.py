from abc import ABC, abstractmethod
import math

class Shape(ABC):
    def __init__(self, color:str):
        self.color = color

    @abstractmethod
    def area(self) -> float:
        pass

    @abstractmethod
    def perimeter(self) -> float:
        pass


class Rectangle(Shape):
    def __init__(self, color:str, width: float, height: float):
        super().__init__(color)
        self.width = width
        self.height = height

    def area(self) -> float:
        return self.width * self.height

    def perimeter(self) -> float:
        return 2 * (self.width + self.height)
    
    
class Circle(Shape):
    def __init__(self, color: str, radius: float):
        super().__init__(color)
        self.radius = radius
        
    def area(self) -> float:
        return math.pi * self.radius**2    
    
    def perimeter(self) -> float:
        return 2 * math.pi * self.radius
    
    
# --- INTERVIEW TEST SUITE ---
if __name__ == "__main__":
    # Test 1: Verify Abstract Instantiation Safeguard
    try:
        s = Shape("Red")
        print("FAIL: Shape should not be directly instantiable!")
    except TypeError:
        print("PASS: Abstract base class properly blocked direct instantiation.")

    # Test 2: Rectangle Execution
    rect = Rectangle("Blue", 4.0, 5.0)
    print(f"\n{rect.color} Rectangle Area: {rect.area()}")
    print(f"{rect.color} Rectangle Perimeter: {rect.perimeter()}")

    # Test 3: Circle Execution
    circ = Circle("Red", 3.0)
    print(f"\n{circ.color} Circle Area: {circ.area():.2f}")
    print(f"{circ.color} Circle Perimeter: {circ.perimeter():.2f}")
