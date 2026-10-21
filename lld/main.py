import threading

class AtomicInteger:
    def __init__(self, initial_value: int = 0):
        self.value = initial_value
        self._locks = threading.Lock()

    def get(self) -> int:
        with self._locks:
            return self.value

    def compare_and_set(self, expected: int, new_value: int) -> bool:
        with self._locks:
            if self.value == expected:
                self.value = new_value
                return True
            return False

    def increment_and_get(self) -> int:
        with self._locks:
            self.value += 1
            return self.value

class EventBatcher:
    def __init__(self, batch_size: int):
        self.buffer = []
        self.condition = threading.Condition()
        self.total_submitted = AtomicInteger(0)
        self.batch = batch_size

    def submit(self, event: str) -> None:
        with self.condition:
            self.buffer.append(event)            
            self.total_submitted.increment_and_get()
            if len(self.buffer) >= self.batch:
                self.condition.notify_all()
            
            
    def extract_batch(self, timeout: float) -> list[str]:
        with self.condition:
            self.condition.wait_for(lambda: len(self.buffer) >= self.batch, timeout=timeout)
            if not self.buffer:
                return []
            batch = self.buffer[:]
            self.buffer.clear()
            return batch
            