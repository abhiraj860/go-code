# BLOCKING QUEUE: put() waits if full, get() waits if empty.
# (In real code just use queue.Queue(maxsize=N).)
import threading
from collections import deque

class BlockingQueue:
    # 1. deque + capacity + one condition
    def __init__(self, capacity):
        self.items = deque()
        self.capacity = capacity
        self.cond = threading.Condition()

    # 2. Producer: wait while full, add, notify
    def put(self, item):
        with self.cond:
            while len(self.items) >= self.capacity:
                self.cond.wait()
            self.items.append(item)
            self.cond.notify_all()

    # 3. Consumer: wait while empty, remove, notify
    def get(self):
        with self.cond:
            while not self.items:
                self.cond.wait()
            item = self.items.popleft()
            self.cond.notify_all()
            return item


if __name__ == "__main__":
    q = BlockingQueue(2)

    def producer():
        for i in range(5):
            q.put(i)

    def consumer():
        for _ in range(5):
            print("got", q.get())

    t1 = threading.Thread(target=producer)
    t2 = threading.Thread(target=consumer)
    t1.start(); t2.start(); t1.join(); t2.join()
