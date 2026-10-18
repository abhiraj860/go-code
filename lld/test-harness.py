import sys
import traceback

try:
    from main import (
        BaseAlert, PhoneAlertDecorator, PoliceAlertDecorator,
        SecuritySystem, SecuritySystemBuilder
    )
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_decorator_chaining():
    alert = BaseAlert()
    alert = PoliceAlertDecorator(alert)
    alert = PhoneAlertDecorator(alert)
    assert alert.trigger() == "Siren, Police, Phone", f"Decorator output incorrect: {alert.trigger()}"

def test_state_machine_transitions():
    system = SecuritySystem(BaseAlert())
    
    # Initial state is Disarmed
    assert system.disarm() == "Already disarmed"
    assert system.trip_sensor() == "Ignored"
    
    # Disarmed -> Armed
    assert system.arm() == "System armed"
    assert system.arm() == "Already armed"
    
    # Armed -> Triggered
    assert system.trip_sensor() == "Siren"
    assert system.trip_sensor() == "Already triggered"
    assert system.arm() == "Cannot arm, system triggered"
    
    # Triggered -> Disarmed
    assert system.disarm() == "System disarmed"
    assert system.trip_sensor() == "Ignored"

def test_builder_validation():
    builder = SecuritySystemBuilder()
    
    try:
        builder.add_phone_alert()
        assert False, "Should raise ValueError when adding decorator without base alert"
    except ValueError:
        pass
        
    try:
        builder.build()
        assert False, "Should raise ValueError when building without base alert"
    except ValueError:
        pass

def test_end_to_end_integration():
    builder = SecuritySystemBuilder()
    
    system = (builder.set_base_alert()
                     .add_phone_alert()
                     .add_police_alert()
                     .build())
                     
    assert isinstance(system, SecuritySystem), "Builder must return a SecuritySystem"
    
    # System should start disarmed
    assert system.trip_sensor() == "Ignored"
    
    # Arm and trigger
    system.arm()
    result = system.trip_sensor()
    
    # Verify State interacted with the Builder's Decorator chain
    expected = "Siren, Phone, Police"
    assert result == expected, f"End-to-end integration failed. Expected '{expected}', got '{result}'"

if __name__ == "__main__":
    tests = [
        test_decorator_chaining,
        test_state_machine_transitions,
        test_builder_validation,
        test_end_to_end_integration
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