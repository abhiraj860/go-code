import sys
import traceback

try:
    from main import DatabaseConnection
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def setup_function():
    DatabaseConnection.reset_instance()

def test_same_instance_returned():
    setup_function()
    db1 = DatabaseConnection("postgres://prod")
    db2 = DatabaseConnection("postgres://prod")
    assert db1 is db2, "DatabaseConnection must return the exact same instance"

def test_initialization_stickiness():
    setup_function()
    db1 = DatabaseConnection("postgres://prod")
    db2 = DatabaseConnection("mysql://staging")
    assert db2.connection_string == "postgres://prod", "Initial connection_string should not be overwritten"

def test_query_execution_and_counting():
    setup_function()
    db = DatabaseConnection("sqlite://mydb")
    assert db.get_query_count() == 0, "Initial query count should be 0"
    
    res1 = db.execute_query("SELECT * FROM users")
    assert res1 == "Executed: SELECT * FROM users on sqlite://mydb"
    assert db.get_query_count() == 1
    
    db2 = DatabaseConnection()
    res2 = db2.execute_query("UPDATE users SET active = 1")
    assert res2 == "Executed: UPDATE users SET active = 1 on sqlite://mydb"
    assert db.get_query_count() == 2
    assert db2.get_query_count() == 2

def test_reset_instance():
    setup_function()
    db1 = DatabaseConnection("db1")
    db1.execute_query("SELECT 1")
    
    DatabaseConnection.reset_instance()
    
    db2 = DatabaseConnection("db2")
    assert db1 is not db2, "After reset, a new instance should be created"
    assert db2.connection_string == "db2", "New instance should use new connection string"
    assert db2.get_query_count() == 0, "New instance query count should start at 0"

if __name__ == "__main__":
    tests = [
        test_same_instance_returned,
        test_initialization_stickiness,
        test_query_execution_and_counting,
        test_reset_instance
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