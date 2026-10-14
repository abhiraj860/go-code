import sys
import traceback

try:
    from main import MediaPlayer, StoppedState, PlayingState, PausedState
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_initial_state_is_stopped():
    player = MediaPlayer()
    assert player.stop() == "Already stopped", "Initial state stop() failed"
    assert player.pause() == "Can't pause, player is stopped", "Initial state pause() failed"

def test_stopped_to_playing_transition():
    player = MediaPlayer()
    assert player.play() == "Starting playback", "Should transition to playing"
    # Verify we are now in PlayingState by checking playing behavior
    assert player.play() == "Already playing", "Should be in playing state now"

def test_playing_to_paused_transition():
    player = MediaPlayer()
    player.play() # Now Playing
    
    assert player.pause() == "Playback paused", "Should transition to paused"
    # Verify we are now in PausedState
    assert player.pause() == "Already paused", "Should be in paused state now"

def test_paused_to_playing_transition():
    player = MediaPlayer()
    player.play()  # Playing
    player.pause() # Paused
    
    assert player.play() == "Resuming playback", "Should transition back to playing"
    assert player.play() == "Already playing", "Should be in playing state again"

def test_playing_to_stopped_transition():
    player = MediaPlayer()
    player.play() # Playing
    
    assert player.stop() == "Playback stopped", "Should transition to stopped"
    assert player.stop() == "Already stopped", "Should be in stopped state now"

def test_paused_to_stopped_transition():
    player = MediaPlayer()
    player.play()  # Playing
    player.pause() # Paused
    
    assert player.stop() == "Playback stopped", "Should transition to stopped from paused"
    assert player.pause() == "Can't pause, player is stopped", "Should be in stopped state now"

if __name__ == "__main__":
    tests = [
        test_initial_state_is_stopped,
        test_stopped_to_playing_transition,
        test_playing_to_paused_transition,
        test_paused_to_playing_transition,
        test_playing_to_stopped_transition,
        test_paused_to_stopped_transition
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