import sys
import traceback
import math

try:
    from main import (
        SimpleCoffee, MilkDecorator, WhipDecorator, VanillaSyrupDecorator
    )
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def is_close(a, b, rel_tol=1e-09, abs_tol=0.0):
    return abs(a-b) <= max(rel_tol * max(abs(a), abs(b)), abs_tol)

def test_simple_coffee():
    coffee = SimpleCoffee()
    assert is_close(coffee.get_cost(), 2.0), f"Expected 2.0, got {coffee.get_cost()}"
    assert coffee.get_description() == "Simple Coffee", f"Got: {coffee.get_description()}"

def test_single_decorator():
    coffee = SimpleCoffee()
    coffee = MilkDecorator(coffee)
    assert is_close(coffee.get_cost(), 2.5), f"Expected 2.5, got {coffee.get_cost()}"
    assert coffee.get_description() == "Simple Coffee, Milk", f"Got: {coffee.get_description()}"

def test_multiple_mixed_decorators():
    coffee = SimpleCoffee()
    coffee = VanillaSyrupDecorator(coffee)
    coffee = WhipDecorator(coffee)
    coffee = MilkDecorator(coffee)
    
    # 2.0 + 0.6 + 0.7 + 0.5 = 3.8
    expected_cost = 3.8
    expected_desc = "Simple Coffee, Vanilla Syrup, Whip, Milk"
    
    assert is_close(coffee.get_cost(), expected_cost), f"Expected {expected_cost}, got {coffee.get_cost()}"
    assert coffee.get_description() == expected_desc, f"Got: {coffee.get_description()}"

def test_duplicate_decorators():
    coffee = SimpleCoffee()
    coffee = MilkDecorator(coffee)
    coffee = MilkDecorator(coffee) # Double milk
    
    # 2.0 + 0.5 + 0.5 = 3.0
    expected_cost = 3.0
    expected_desc = "Simple Coffee, Milk, Milk"
    
    assert is_close(coffee.get_cost(), expected_cost), f"Expected {expected_cost}, got {coffee.get_cost()}"
    assert coffee.get_description() == expected_desc, f"Got: {coffee.get_description()}"

if __name__ == "__main__":
    tests = [
        test_simple_coffee,
        test_single_decorator,
        test_multiple_mixed_decorators,
        test_duplicate_decorators
    ]
    
    passed = 0
    for test in tests:
        try:
            test()
            print(f"PASS: {test.__name__}")
            passed += 1
        except Exception as e:
            print(f"FAIL: {test.__name__}")
            print(f"      {type(e).__name__}: {e}")
            traceback.print_exc(limit=-1, file=sys.stdout)
            
    print(f"\nSummary: {passed}/{len(tests)} passed.")
    if passed < len(tests):
        sys.exit(1)
    else:
        sys.exit(0)