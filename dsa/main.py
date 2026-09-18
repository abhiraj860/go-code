from concurrent.futures import ThreadPoolExecutor
import time

def fetchUser(userId):
    time.sleep(1)
    return f"User data for ID: {userId}"

userId = [101, 102, 103, 104, 105]

with ThreadPoolExecutor(max_workers=1) as executor:
    results = executor.map(fetchUser, userId) 
    for result in results:
        print(result)   