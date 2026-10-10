import pytest
from main import HttpRequest, HttpRequestBuilder

def test_builder_requires_url():
    with pytest.raises(TypeError):
        HttpRequestBuilder()

def test_default_state():
    request = HttpRequestBuilder("https://api.example.com").build()
    
    assert isinstance(request, HttpRequest)
    assert request.url == "https://api.example.com"
    assert request.method == "GET"
    assert request.headers == {}
    assert request.body is None

def test_fluent_chaining_and_population():
    request = (HttpRequestBuilder("https://api.example.com/users")
               .method("POST")
               .add_header("Authorization", "Bearer 12345")
               .add_header("Content-Type", "application/json")
               .body('{"name": "Alice"}')
               .build())
    
    assert request.url == "https://api.example.com/users"
    assert request.method == "POST"
    assert request.headers == {
        "Authorization": "Bearer 12345",
        "Content-Type": "application/json"
    }
    assert request.body == '{"name": "Alice"}'

def test_header_overwrites_existing_key():
    request = (HttpRequestBuilder("https://api.example.com")
               .add_header("Accept", "text/plain")
               .add_header("Accept", "application/json")
               .build())
               
    assert request.headers == {"Accept": "application/json"}

def test_validation_get_with_body_raises_error():
    builder = HttpRequestBuilder("https://api.example.com").body('{"query": "test"}')
    
    with pytest.raises(ValueError, match="GET requests cannot have a body"):
        builder.build()
        
def test_validation_explicit_get_with_body_raises_error():
    builder = (HttpRequestBuilder("https://api.example.com")
               .method("GET")
               .body('{"query": "test"}'))
               
    with pytest.raises(ValueError, match="GET requests cannot have a body"):
        builder.build()