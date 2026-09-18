import asyncio
import time

async def downloadFile(fileId, duration):
    print(f"File {fileId}: Starting Download...")
    await asyncio.sleep(duration)
    print(f"File {fileId}: Finished in {duration}s!")
    return f"content_{fileId}"

async def main():
    start = time.time()
    result = await asyncio.gather(
        downloadFile(1, 3),
        downloadFile(2, 1),
        downloadFile(3, 2),
    )
    
    print(f"Download: {result}")
    print(f"Total time: {time.time() - start:.2f} seconds")
    
asyncio.run(main())
