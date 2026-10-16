import sys
import traceback

try:
    from main import Projector, SoundSystem, StreamingDevice, HomeTheaterFacade
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_watch_movie_sequence():
    projector = Projector()
    sound = SoundSystem()
    streaming = StreamingDevice()
    facade = HomeTheaterFacade(projector, sound, streaming)

    logs = facade.watch_movie("Inception", 75)

    expected = [
        "Projector powered on",
        "Projector input set to HDMI 1",
        "Sound system powered on",
        "Sound system volume set to 75",
        "Streaming device powered on",
        "Playing 'Inception'"
    ]
    assert logs == expected, f"Sequence mismatch.\nGot: {logs}\nExpected: {expected}"

def test_watch_movie_volume_validation():
    projector = Projector()
    sound = SoundSystem()
    streaming = StreamingDevice()
    facade = HomeTheaterFacade(projector, sound, streaming)

    try:
        facade.watch_movie("Inception", 150)
        assert False, "Should have raised ValueError for volume > 100"
    except ValueError:
        pass

    try:
        facade.watch_movie("Inception", -10)
        assert False, "Should have raised ValueError for volume < 0"
    except ValueError:
        pass

def test_end_movie_sequence():
    projector = Projector()
    sound = SoundSystem()
    streaming = StreamingDevice()
    facade = HomeTheaterFacade(projector, sound, streaming)

    logs = facade.end_movie()

    expected = [
        "Streaming device powered off",
        "Sound system powered off",
        "Projector powered off"
    ]
    assert logs == expected, f"Sequence mismatch.\nGot: {logs}\nExpected: {expected}"

if __name__ == "__main__":
    tests = [
        test_watch_movie_sequence,
        test_watch_movie_volume_validation,
        test_end_movie_sequence
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