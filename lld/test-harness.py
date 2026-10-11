import sys
import traceback

try:
    from main import JobBoard, JobSeeker
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_single_observer_notification():
    board = JobBoard()
    alice = JobSeeker("Alice")
    
    board.attach(alice)
    board.add_job("Software Engineer")
    
    assert len(alice.notifications) == 1, "Alice should have 1 notification"
    assert alice.notifications[0] == "Software Engineer", "Notification content mismatch"

def test_multiple_observers():
    board = JobBoard()
    bob = JobSeeker("Bob")
    charlie = JobSeeker("Charlie")
    
    board.attach(bob)
    board.attach(charlie)
    board.add_job("Data Scientist")
    
    assert bob.notifications == ["Data Scientist"], "Bob missed the notification"
    assert charlie.notifications == ["Data Scientist"], "Charlie missed the notification"

def test_detach_stops_notifications():
    board = JobBoard()
    diana = JobSeeker("Diana")
    
    board.attach(diana)
    board.add_job("Backend Dev")
    board.detach(diana)
    board.add_job("Frontend Dev")
    
    assert len(diana.notifications) == 1, "Diana should have stopped receiving updates"
    assert diana.notifications[0] == "Backend Dev", "Diana received the wrong job"

def test_attach_duplicate_raises_value_error():
    board = JobBoard()
    eve = JobSeeker("Eve")
    
    board.attach(eve)
    try:
        board.attach(eve)
        assert False, "Should have raised ValueError on duplicate attach"
    except ValueError:
        pass

def test_detach_nonexistent_raises_key_error():
    board = JobBoard()
    frank = JobSeeker("Frank")
    
    try:
        board.detach(frank)
        assert False, "Should have raised KeyError when detaching unknown observer"
    except KeyError:
        pass

if __name__ == "__main__":
    tests = [
        test_single_observer_notification,
        test_multiple_observers,
        test_detach_stops_notifications,
        test_attach_duplicate_raises_value_error,
        test_detach_nonexistent_raises_key_error
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