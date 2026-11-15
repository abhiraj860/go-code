# SEMAPHORE: allow at most N threads at once.
import threading
import time

class Pool:
    def __init__(self, n):
        self.sem = threading.Semaphore(n)   # 1. N permits

    def use(self, name):
        with self.sem:                      # 2. take a permit (wait if none left)
            print(name, "in")
            time.sleep(0.1)
            print(name, "out")
        # 3. permit given back automatically


if __name__ == "__main__":
    pool = Pool(2)
    threads = [threading.Thread(target=pool.use, args=(f"T{i}",)) for i in range(4)]
    for t in threads: t.start()
    for t in threads: t.join()
