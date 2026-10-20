import sys
import traceback

try:
    from main import (
        UIComponent, ButtonComponent, 
        PaddingDecorator, ShadowDecorator, ComponentRegistry
    )
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def setup_function():
    ComponentRegistry.reset_instance()

def test_singleton_registry():
    setup_function()
    reg1 = ComponentRegistry()
    reg2 = ComponentRegistry()
    assert reg1 is reg2, "ComponentRegistry must be a singleton"

def test_prototype_deep_copy():
    btn = ButtonComponent("Save", {"color": "blue"})
    cloned_btn = btn.clone()
    
    assert btn is not cloned_btn, "Clone must return a new instance"
    assert btn.render() == "[Button: Save]"
    
    # Mutate clone's dict
    cloned_btn.metadata["color"] = "red"
    assert btn.metadata["color"] == "blue", "Original metadata must not be affected by clone mutation"

def test_decorator_rendering():
    btn = ButtonComponent("Cancel", {})
    padded = PaddingDecorator(btn)
    shadowed_padded = ShadowDecorator(padded)
    
    assert shadowed_padded.render() == "shadow(padding([Button: Cancel]))"

def test_registry_integration():
    setup_function()
    registry = ComponentRegistry()
    
    # Register a plain button
    btn = ButtonComponent("Submit", {"theme": "dark"})
    registry.register("btn_primary", btn)
    
    # Fetch clone
    instance1 = registry.create_instance("btn_primary")
    assert instance1 is not btn, "Registry must return a clone, not the original reference"
    
    # Modify clone, ensure registry template is safe
    instance1.metadata["theme"] = "light"
    template = registry.create_instance("btn_primary")
    assert template.metadata["theme"] == "dark", "Registry template was corrupted"

def test_decorated_prototype_cloning():
    setup_function()
    registry = ComponentRegistry()
    
    # Create a complex, decorated base component
    btn = ButtonComponent("Alert", {"priority": "high"})
    complex_component = ShadowDecorator(PaddingDecorator(btn))
    
    # Register the entire decorated stack
    registry.register("alert_widget", complex_component)
    
    # Fetch a clone of the entire stack
    widget = registry.create_instance("alert_widget")
    
    assert isinstance(widget, ShadowDecorator), "Clone should maintain decorator structure"
    assert widget.render() == "shadow(padding([Button: Alert]))"
    
    # Because of deep copy, we should be able to dig into the clone's wrapper and mutate safely
    # widget -> Shadow -> Padding -> Button
    cloned_inner_btn = widget.wrapped.wrapped
    cloned_inner_btn.metadata["priority"] = "low"
    
    # Original template should remain 'high'
    master_template = registry.create_instance("alert_widget")
    master_inner_btn = master_template.wrapped.wrapped
    assert master_inner_btn.metadata["priority"] == "high", "Decorated clone mutation corrupted master template"

if __name__ == "__main__":
    tests = [
        test_singleton_registry,
        test_prototype_deep_copy,
        test_decorator_rendering,
        test_registry_integration,
        test_decorated_prototype_cloning
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