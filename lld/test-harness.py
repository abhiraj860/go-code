import sys
import traceback
import json

try:
    from main import AnalyticsLogger, CloudWatchService, CloudWatchAdapter
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_adapter_inheritance():
    service = CloudWatchService()
    adapter = CloudWatchAdapter(service)
    assert isinstance(adapter, AnalyticsLogger), "CloudWatchAdapter must implement AnalyticsLogger"

def test_log_event_translation():
    service = CloudWatchService()
    adapter = CloudWatchAdapter(service)
    
    adapter.log_event("api_timeout", {"endpoint": "/users", "latency_ms": 5000})
    
    assert len(service.metrics_store) == 1, "Should have pushed 1 metric to the service"
    
    metric_name, dimensions = service.metrics_store[0]
    assert metric_name == "API_TIMEOUT", f"Event name should be uppercase, got '{metric_name}'"
    
    expected_json = json.dumps({"endpoint": "/users", "latency_ms": 5000})
    assert dimensions == expected_json, f"Metadata should be JSON string, got '{dimensions}'"

def test_empty_metadata():
    service = CloudWatchService()
    adapter = CloudWatchAdapter(service)
    
    adapter.log_event("system_reboot", {})
    
    assert len(service.metrics_store) == 1, "Should have pushed 1 metric to the service"
    metric_name, dimensions = service.metrics_store[0]
    
    assert metric_name == "SYSTEM_REBOOT", f"Expected 'SYSTEM_REBOOT', got '{metric_name}'"
    assert dimensions == "{}", f"Empty dict should serialize to '{{}}', got '{dimensions}'"

if __name__ == "__main__":
    tests = [
        test_adapter_inheritance,
        test_log_event_translation,
        test_empty_metadata
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