import sys
import traceback

try:
    from main import NPC, Prototype
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_clone_creates_new_instance():
    goblin_archetype = NPC("Goblin", 100, ["Dagger", "Health Potion"])
    spawned_goblin = goblin_archetype.clone()
    
    assert isinstance(spawned_goblin, NPC), "Clone must be an instance of NPC"
    assert spawned_goblin is not goblin_archetype, "Clone must be a entirely new object in memory"

def test_clone_deep_copy_inventory():
    goblin_archetype = NPC("Goblin", 100, ["Dagger"])
    spawned_goblin = goblin_archetype.clone()
    
    spawned_goblin.add_item("Stolen Gold")
    
    assert "Stolen Gold" in spawned_goblin.inventory, "Clone should have the new item"
    assert "Stolen Gold" not in goblin_archetype.inventory, "Original archetype's inventory MUST NOT be affected by clone modifications"

def test_clone_primitive_independence():
    goblin_archetype = NPC("Goblin", 100, [])
    spawned_goblin = goblin_archetype.clone()
    
    spawned_goblin.health -= 50
    spawned_goblin.name = "Wounded Goblin"
    
    assert goblin_archetype.health == 100, "Original archetype health should remain unchanged"
    assert goblin_archetype.name == "Goblin", "Original archetype name should remain unchanged"
    assert spawned_goblin.health == 50, "Clone health should be updated"

def test_health_validation():
    try:
        NPC("Zombie", -10, [])
        assert False, "Should raise ValueError for negative health"
    except ValueError:
        pass

if __name__ == "__main__":
    tests = [
        test_clone_creates_new_instance,
        test_clone_deep_copy_inventory,
        test_clone_primitive_independence,
        test_health_validation
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