# CONDITION: a thread sleeps until another thread says "ready".
import threading

class Gate:
    def __init__(self):
        self.cond = threading.Condition()   # 1. condition (has a lock inside)
        self.ready = False

    # 2. Waiter: lock, WHILE not ready -> wait
    def wait(self, name):
        with self.cond:
            while not self.ready:
                self.cond.wait()
            print(name, "proceeds")

    # 3. Signaler: lock, change state, notify
    def open(self):
        with self.cond:
            self.ready = True
            self.cond.notify_all()


if __name__ == "__main__":
    g = Gate()
    threads = [threading.Thread(target=g.wait, args=(f"T{i}",)) for i in range(3)]
    for t in threads: t.start()
    g.open()
    for t in threads: t.join()
