from typing import Dict, Optional

class HttpRequest:
    def __init__(self, method: str, url: str, headers: Dict[str, str], query_params: Dict[str, str], body: Optional[str]):
        self.method = method
        self.url = url
        self.headers = headers
        self.query_params = query_params
        self.body = body

class HttpRequestBuilder:
    def __init__(self):
        self.method = "GET"
        self.url = None
        self.headers = {}
        self.query_param = {}
        self.body = None

    def set_method(self, method: str) -> 'HttpRequestBuilder':
        self.method = method
        return self

    def set_url(self, url: str) -> 'HttpRequestBuilder':
        self.url = url
        return self

    def add_header(self, key: str, value: str) -> 'HttpRequestBuilder':
        self.headers[key] = value
        return self

    def add_query_param(self, key: str, value: str) -> 'HttpRequestBuilder':
        self.query_param[key] = value
        return self

    def set_body(self, body: str) -> 'HttpRequestBuilder':
        self.body = body
        return self

    def build(self) -> HttpRequest:
        if self.url is None:
            raise ValueError("Please input URL")
        http_request = HttpRequest(method = self.method, url = self.url, headers = self.headers, query_params = self.query_param, body = self.body)
        return http_request        