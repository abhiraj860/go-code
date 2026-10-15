import sys
import traceback

try:
    from main import HttpRequestBuilder, HttpRequest
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def test_fluent_interface_and_build():
    request = (HttpRequestBuilder()
               .set_method("POST")
               .set_url("https://api.example.com/users")
               .add_header("Content-Type", "application/json")
               .add_header("Authorization", "Bearer token123")
               .add_query_param("timeout", "30")
               .set_body('{"name": "Alice"}')
               .build())
    
    assert isinstance(request, HttpRequest), "build() must return an HttpRequest instance"
    assert request.method == "POST"
    assert request.url == "https://api.example.com/users"
    assert request.headers == {"Content-Type": "application/json", "Authorization": "Bearer token123"}
    assert request.query_params == {"timeout": "30"}
    assert request.body == '{"name": "Alice"}'

def test_default_method_and_empty_collections():
    request = HttpRequestBuilder().set_url("https://api.example.com/health").build()
    
    assert request.method == "GET", "Method should default to GET"
    assert request.url == "https://api.example.com/health"
    assert request.headers == {}, "Headers should default to empty dict"
    assert request.query_params == {}, "Query params should default to empty dict"
    assert request.body is None, "Body should default to None"

def test_missing_url_raises_error():
    builder = HttpRequestBuilder().set_method("PUT").set_body("data")
    
    try:
        builder.build()
        assert False, "Should have raised ValueError because URL was not set"
    except ValueError:
        pass

def test_accumulation():
    builder = HttpRequestBuilder()
    builder.set_url("http://test.com")
    builder.add_query_param("page", "1")
    builder.add_query_param("sort", "desc")
    
    request = builder.build()
    assert request.query_params == {"page": "1", "sort": "desc"}, "Query params did not accumulate correctly"

if __name__ == "__main__":
    tests = [
        test_fluent_interface_and_build,
        test_default_method_and_empty_collections,
        test_missing_url_raises_error,
        test_accumulation
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