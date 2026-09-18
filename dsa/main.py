import asyncio

async def sendEmail(userId):
    await asyncio.sleep(4)
    print(f"Email sent to user {userId}")
    
async def main():
    task = asyncio.create_task(sendEmail(100))
    print("Processing HTTP request...")
    await asyncio.sleep(1)
    print("HTTP response returned to user!")
    await task

asyncio.run(main())