import threading

balance = 0
lock = threading.Lock()

def deposit(amount):
    global balance
    for _ in range(100000):
        with lock:
            balance += amount

t1 = threading.Thread(target=deposit, args=(1,))
t2 = threading.Thread(target=deposit, args=(1,))

t1.start()
t2.start()
t1.join()
t2.join()

print(f"final balance {balance}")