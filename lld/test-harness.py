import sys
import traceback

try:
    from main import (
        Checkout, DiscountStrategy, PercentageDiscount, 
        FlatDiscount, NoDiscount
    )
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_no_discount():
    checkout = Checkout(NoDiscount())
    assert checkout.calculate_total(100.0) == 100.0, "NoDiscount should return original price"

def test_percentage_discount():
    checkout = Checkout(PercentageDiscount(20.0))
    assert checkout.calculate_total(100.0) == 80.0, "20% off 100 should be 80.0"
    
def test_percentage_discount_validation():
    try:
        PercentageDiscount(150.0)
        assert False, "Should have raised ValueError for > 100%"
    except ValueError:
        pass
        
    try:
        PercentageDiscount(-10.0)
        assert False, "Should have raised ValueError for < 0%"
    except ValueError:
        pass

def test_flat_discount_standard():
    checkout = Checkout(FlatDiscount(15.0))
    assert checkout.calculate_total(50.0) == 35.0, "Flat discount of 15 off 50 should be 35.0"

def test_flat_discount_exceeds_price():
    checkout = Checkout(FlatDiscount(100.0))
    assert checkout.calculate_total(50.0) == 0.0, "Discount exceeding price should result in 0.0"

def test_dynamic_strategy_swap():
    checkout = Checkout(PercentageDiscount(50.0))
    assert checkout.calculate_total(100.0) == 50.0, "Initial strategy failed"
    
    checkout.set_strategy(FlatDiscount(10.0))
    assert checkout.calculate_total(100.0) == 90.0, "Strategy swap failed"

def test_negative_price_validation():
    checkout = Checkout(NoDiscount())
    try:
        checkout.calculate_total(-50.0)
        assert False, "Should have raised ValueError for negative price"
    except ValueError:
        pass

if __name__ == "__main__":
    tests = [
        test_no_discount,
        test_percentage_discount,
        test_percentage_discount_validation,
        test_flat_discount_standard,
        test_flat_discount_exceeds_price,
        test_dynamic_strategy_swap,
        test_negative_price_validation
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