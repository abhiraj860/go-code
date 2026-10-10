from typing import Dict, Optional

class HttpRequest:
    url: str
    method: str
    headers: Dict[str, str]
    body: Optional[str]

class HttpRequestBuilder:
    def __init__(self, url: str):
        self.url = url
        self._method = "GET"
        self.header = {}
        self._body = None

    def method(self, http_method: str) -> 'HttpRequestBuilder':
        self._method = http_method
        return self

    def add_header(self, key: str, value: str) -> 'HttpRequestBuilder':
        self.header[key] = value
        return self

    def body(self, payload: str) -> 'HttpRequestBuilder':
        self._body = payload
        return self
    
    def build(self) -> HttpRequest:
        if self.url is None:
            raise ValueError("URL not provided")
        http_request = HttpRequest()
        http_request.url = self.url
        http_request.method = self._method
        http_request.headers = self.header
        http_request.body = self._body
        if self._method == "GET" and self._body is not None:
            raise ValueError("GET requests cannot have a body")
        return http_request
        

    
