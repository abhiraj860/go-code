# PUB-SUB (log / Kafka-style): one append-only log per topic,
# each subscriber keeps only its OFFSET. Messages are never removed on read.
import threading

# 1. Topic: append-only log + condition so readers can wait
class Topic:
    def __init__(self):
        self.log = []
        self.cond = threading.Condition()

    def publish(self, msg):
        with self.cond:
            self.log.append(msg)
            self.cond.notify_all()

    def read(self, offset, timeout=None):
        with self.cond:
            while offset >= len(self.log):
                if not self.cond.wait(timeout):
                    return None              # timed out
            return self.log[offset]

# 2. Subscriber: owns its offset, so each one reads at its own pace
class Subscriber:
    def __init__(self, name, topic):
        self.name = name
        self.topic = topic
        self.offset = 0

    def poll(self, timeout=None):
        msg = self.topic.read(self.offset, timeout)
        if msg is not None:
            self.offset += 1
        return msg

    def seek(self, offset):                  # replay
        self.offset = offset

# 3. Broker: topic registry
class Broker:
    def __init__(self):
        self.topics = {}
        self.lock = threading.Lock()

    def topic(self, name):
        with self.lock:
            return self.topics.setdefault(name, Topic())


if __name__ == "__main__":
    b = Broker()
    orders = b.topic("orders")
    a = Subscriber("A", orders)
    c = Subscriber("C", orders)

    for i in range(3):
        orders.publish(f"order-{i}")

    print("A:", [a.poll() for _ in range(3)])
    print("C:", c.poll())          # C reads independently from offset 0
    a.seek(0)
    print("A replay:", a.poll())
