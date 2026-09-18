import threading
import time

def downloadFile(filename, duration):
    print(f"[{filename}] Start download...")
    time.sleep(duration)
    print(f"[{filename}] Download finished...")
    
thread1 = threading.Thread(target=downloadFile, args=("file1.zip", 3))
thread2 = threading.Thread(target=downloadFile, args=("file2.zip", 1))

thread1.start()
thread2.start()

thread1.join()
thread2.join()
print("All download complete!!!")