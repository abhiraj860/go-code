# QUEUE + WORKER POOL: fast half enqueues and returns, slow half works + retries.
import queue
import random
import threading
import time

# 1. Job = record with its own status
class Job:
    def __init__(self, job_id, payload):
        self.id = job_id
        self.payload = payload
        self.status = "PENDING"
        self.attempts = 0

# 2. Retry policy: exponential backoff, capped attempts
class RetryPolicy:
    def __init__(self, max_attempts=3, base=0.05):
        self.max_attempts = max_attempts
        self.base = base

    def should_retry(self, attempts):
        return attempts < self.max_attempts

    def delay(self, attempts):
        return self.base * (2 ** (attempts - 1))

# 3. Pool: N threads take from one queue
class WorkerPool:
    STOP = object()

    def __init__(self, n, handler, retry):
        self.q = queue.Queue()
        self.handler = handler
        self.retry = retry
        self.threads = [threading.Thread(target=self._loop) for _ in range(n)]
        for t in self.threads:
            t.start()

    def submit(self, job):              # fast half: record + enqueue, return
        self.q.put(job)

    def _loop(self):                    # slow half
        while True:
            job = self.q.get()
            if job is self.STOP:
                return
            self._run(job)

    def _run(self, job):
        while True:
            job.attempts += 1
            try:
                self.handler(job.payload)
                job.status = "DONE"
                return
            except Exception:
                if not self.retry.should_retry(job.attempts):
                    job.status = "FAILED"
                    return
                time.sleep(self.retry.delay(job.attempts))

    def shutdown(self):
        for _ in self.threads:
            self.q.put(self.STOP)       # one poison pill per worker
        for t in self.threads:
            t.join()


if __name__ == "__main__":
    def flaky_send(msg):
        if random.random() < 0.5:
            raise IOError("network")

    pool = WorkerPool(3, flaky_send, RetryPolicy())
    jobs = [Job(i, f"msg-{i}") for i in range(6)]
    for j in jobs:
        pool.submit(j)
    pool.shutdown()
    for j in jobs:
        print(j.id, j.status, "attempts:", j.attempts)
