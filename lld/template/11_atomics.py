# ATOMIC: one variable, changed safely by many threads.
# Python has no atomic int, so wrap value + lock.
import threading

class AtomicInteger:
    # 1. Value + lock
    def __init__(self, value=0):
        self.value = value
        self.lock = threading.Lock()

    # 2. Every op goes through the lock
    def get(self):
        with self.lock:
            return self.value

    def increment(self):
        with self.lock:
            self.value += 1
            return self.value

    # 3. Compare-and-set: update only if value is still what we expect
    def compare_and_set(self, expected, new):
        with self.lock:
            if self.value == expected:
                self.value = new
                return True
            return False


if __name__ == "__main__":
    c = AtomicInteger()
    def work():
        for _ in range(1000):
            c.increment()
    threads = [threading.Thread(target=work) for _ in range(4)]
    for t in threads: t.start()
    for t in threads: t.join()
    print(c.get())   # 4000
