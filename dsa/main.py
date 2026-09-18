import asyncio

async def fetchData():
    print("Start fetching...")
    await asyncio.sleep(2)
    print("Data fetched!")
    return {"data": 100} 

async def main():
    result = await fetchData()
    print(f"Result {result}")   

print("Hello")
asyncio.run(main())
