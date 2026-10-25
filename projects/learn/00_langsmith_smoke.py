from dotenv import load_dotenv

load_dotenv()  # finds projects/.env by walking up from this file

from langsmith import Client, traceable

# 1. Verify the key works
client = Client()
print(list(client.list_projects(limit=1)))  # errors here = bad key/endpoint

# 2. Send a test trace
@traceable
def ping(x: str) -> str:
    return x.upper()

ping("hello")
client.flush()  # make sure it's sent before the script exits
