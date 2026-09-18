import asyncio

async def slowApi():
    await asyncio.sleep(1)
    return "API Success"

async def main():
    try:
        result = await asyncio.wait_for(slowApi(), timeout = 2.0)
        print(result)
    except asyncio.TimeoutError:
        print("Reqiestion time out")
        
asyncio.run(main())