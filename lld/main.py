class DatabaseConnection:
    _instance = None
    _initialized = False
    _counter = 0
    
    def __new__(cls, connection_string: str = "sqlite://localhost"):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self, connection_string: str = "sqlite://localhost"):
        if DatabaseConnection._initialized:
            return
        self.connection_string = connection_string
        DatabaseConnection._initialized = True
        return 
            

    def execute_query(self, query: str) -> str:
       DatabaseConnection._counter += 1
       return f"Executed: {query} on {self.connection_string}" 

    def get_query_count(self) -> int:
        return DatabaseConnection._counter

    @classmethod
    def reset_instance(cls) -> None:
        DatabaseConnection._instance = None
        DatabaseConnection._initialized = False
        DatabaseConnection._counter = 0