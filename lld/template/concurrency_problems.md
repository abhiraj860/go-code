# Concurrency Practice: 20 Problems

Every concurrency scenario in the LLD cards (`#01`–`#29`, kept in [moves_problems.md](moves_problems.md)), plus every primitive in `concurrency_and_system_design/`, is turned into one interview-size problem. Names follow the deep_dive cards and your templates. This file is the companion to [design_pattern_problems.md](design_pattern_problems.md), where P33–P35, P37 and P39 practise the queue, pool and breaker *design*. Here the focus is **thread safety**.

### How to use

1. Read the problem, write your solution in a new `.py` file, paste the **driver code** under it, and run it.
2. Compare the output with the comments, using the same rules as the pattern file:
   - `print(x)   # True` means that line must print `True`.
   - A comment-only line under a `print` is the next expected output line.
   - Text after `->` is only an explanation. It is not printed.
3. Names under **Names the driver uses** are fixed. Everything else is your design.

### The `yield_point()` rule (important)

On CPython the GIL hides most races. An unlocked `if free: take()` almost always passes a test by luck. To make a **missing lock actually fail**, every driver defines:

```python
def yield_point():
    time.sleep(0)      # forces a thread switch right here
```

Each problem tells you **where to call `yield_point()`**. It always goes **inside your critical section, between the check and the change** (for example, between "seat is available" and "mark it held"). With your lock in place nothing changes. Without the lock, another thread slips in and the driver output goes wrong. Your solution can call it freely, because the driver defines it before running anything.

### How these drivers were verified

- Each driver was run against a reference solution, **several times in a row**, and printed exactly the expected output every time.
- **Mutation check:** the lock-based problems (C01–C13, C19, C20) were each run again with every `threading.Lock()` replaced by a lock that does nothing. Every one of those drivers then **failed**, so they really do catch a missing lock.
- For the problems built on conditions or semaphores, a specific bug was injected instead, and the driver caught each one:
  - C08: wrong lock order (deadlock)
  - C14: a missing `notify_all`
  - C15: missing backpressure
  - C17: no politeness slot
  - C18: no in-flight limit
- A few lines check timing, for example `elapsed < 0.6 → True`. On a very slow or heavily loaded machine those lines alone could print `False` even when your code is right.

### Index

| # | Problem | Primitive / move | Source |
|---|---|---|---|
| | **Part 1: Atomics** | | |
| C01 | Atomic Counter, Ticket IDs & Max Tracker | atomic value, compare-and-set loop | `11_atomics.py`, "ids from an atomic counter" |
| C02 | Snowflake ID Generator | one lock around read-clock + bump-sequence | #22 |
| | **Part 2: Lock the scarce thing (M1)** | | |
| C03 | Parking Lot with Many Gates | per-spot lock, `try_occupy`, atomic ticket claim | #01 |
| C04 | Seat Holds That Expire | per-seat lock, all-or-nothing undo, confirm vs expire (M6) | #02 |
| C05 | Meeting Room Calendars | atomic overlap check, reserve-before-release | #24 (also #25, #28 crew) |
| C06 | Ride Matching on Stale Snapshots | decide on copies, `try_reserve` may refuse (M10) | #08 (also #04, #25) |
| C07 | Amazon Locker | per-compartment lock, `put_if_absent` codes, collect vs expire | #18 |
| | **Part 3: Multiple locks (M8)** | | |
| C08 | Wallet Transfers | lock two in fixed order, idempotency key claimed first | #27, `12_locks_mutex.py` |
| | **Part 4: Locking shared structures** | | |
| C09 | Thread-safe LRU Cache | one lock, reads lock too | #07, `23_lru_cache.py` |
| C10 | Rate Limiter Under a Burst | per-key lock, safe lazy creation | #06, `22_rate_limiter.py` |
| C11 | Logging From Many Threads | one lock per destination | #09 |
| C12 | KV Store with TTL + Transactions | thread-confined layers, one-lock commit | #21 |
| | **Part 5: Semaphores** | | |
| C13 | Per-Endpoint Limit & Connection Pool | counting semaphore, timeout | #29, `13_semaphores.py` |
| | **Part 6: Condition variables** | | |
| C14 | Delay Queue & Countdown Latch | `while` wait, timed wait, notify | #17 follow-up, `14_condition_variables.py` |
| | **Part 7: Producer–consumer & worker pools (M7)** | | |
| C15 | Processing Pipeline | bounded blocking queue, backpressure, poison pill | `15_blocking_queue.py`, #19 |
| C16 | Worker Pool with Retries | N workers, backoff, graceful shutdown | `19_worker_pool.py`, #13 #17 |
| C17 | Polite Multi-threaded Crawler | per-site frontier, reserved slot, pending count | #23 |
| C18 | Webhook Lanes Under Load | atomic in-flight limit, fair blocking take | #29 |
| | **Part 8: Pub-Sub & Circuit Breaker** | | |
| C19 | Log-based Pub-Sub with Blocking Reads | condition-guarded log, get-or-create registry | #19, `21_pub_sub.py` |
| C20 | Circuit Breaker Under Concurrency | exactly one half-open trial, no lock during the call | #29, `20_circuit_breaker.py` |


---

## C01 · Atomic Counter, Ticket IDs & Max Tracker — Atomics

**Primitive:** atomic value (value + lock) and compare-and-set · **Difficulty:** Easy · **Time:** ~15 min · **Source:** template `11_atomics.py`; "ids come from an atomic counter" in `#01`, `#02`, `#08`, `#24`, and others

### Problem

Python has no atomic integer, so build one from a value plus a lock. Then use it for two things every LLD needs: **unique ids handed out to many threads**, and a **lock-free style max update** using compare-and-set in a retry loop.

1. `AtomicInteger(value=0)`:
   - `get()`, `increment()` (returns the new value), `add_and_get(delta)`.
   - `compare_and_set(expected, new)` sets the value only if it currently equals `expected`, and returns whether it did.
   - Every operation holds the lock.
2. `IdGenerator(prefix).next_id()` returns `<prefix>1`, `<prefix>2`, …. It is safe from any number of threads, so ids are never repeated or skipped.
3. `MaxTracker.offer(value)` raises the stored maximum if `value` is bigger, and returns `True` if it raised it.
   - Use a **compare-and-set loop**: read the current value; if `value` isn't bigger, return `False`; otherwise try `compare_and_set(current, value)`, and repeat if another thread changed it first.
   - Don't add a separate lock here.
4. **`yield_point()`:** call it inside `increment` between reading and writing the value, and inside `compare_and_set` between the check and the write.

### Names the driver uses

- `AtomicInteger(value=0)`: `get`, `increment`, `add_and_get`, `compare_and_set`
- `IdGenerator(prefix)`: `next_id`
- `MaxTracker()`: `offer(value) -> bool`, `get`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)                           # -> forces a thread switch inside your critical sections


def run_threads(n, target):
    barrier = threading.Barrier(n)
    def wrapped(i):
        barrier.wait()                      # -> start every thread at the same moment
        target(i)
    threads = [threading.Thread(target=wrapped, args=(i,)) for i in range(n)]
    for t in threads: t.start()
    for t in threads: t.join()


if __name__ == "__main__":
    c = AtomicInteger()
    print(c.increment(), c.add_and_get(5), c.get())          # 1 6 6
    print(c.compare_and_set(5, 100), c.compare_and_set(6, 100), c.get())
                                                             # False True 100

    counter = AtomicInteger()
    run_threads(8, lambda i: [counter.increment() for _ in range(5000)])
    print(counter.get())                                     # 40000  -> no lost updates

    gen = IdGenerator("T")
    ids, ids_lock = [], threading.Lock()
    def take_ids(i):
        mine = [gen.next_id() for _ in range(500)]
        with ids_lock: ids.extend(mine)
    run_threads(8, take_ids)
    print(len(ids), len(set(ids)))                           # 4000 4000  -> every ticket id unique
    print(sorted(ids, key=lambda s: int(s[1:]))[-1])         # T4000

    best = MaxTracker()
    run_threads(8, lambda i: [best.offer(i * 1000 + k) for k in range(1000)])
    print(best.get())                                        # 7999
    print(best.offer(5), best.offer(8000))                   # False True
```


---

## C02 · Snowflake ID Generator — Lock around read-time + bump-sequence

**Primitive:** one lock around a multi-step update · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#22 · Unique ID Generator (Snowflake)`

### Problem

Generate unique 64-bit ids on many machines without coordination. An id is **`timestamp_ms | machine_id | sequence`**, so it sorts roughly by time. Within one generator, many threads call `next_id()` at once. Reading the clock, bumping the sequence and saving the last timestamp must happen **as one step**.

1. **Layout:** `(timestamp << (10 + seq_bits)) | (machine_id << seq_bits) | sequence`.
   - The machine id has 10 bits, so it must be `0..1023`. Anything else raises `ValueError`.
   - `seq_bits` defaults to 12.
2. **Next id:**
   - In the same millisecond as the last id, the sequence goes up by 1.
   - In a new millisecond, the sequence restarts at 0.
   - **Overflow:** if the sequence would go past its maximum, wait for the next millisecond (`clock.wait_until(last + 1)`) and use sequence 0.
3. **Clock moving backwards:**
   - Up to 5 ms behind the last timestamp: wait until the last timestamp (`clock.wait_until(last)`), then continue as the same millisecond.
   - More than 5 ms behind: raise `ClockMovedBackwardsError`.
4. `decode(id)` returns `(timestamp, machine_id, sequence)`.
5. **Clocks:**
   - `SystemClock` uses real time; its `wait_until` sleeps or spins until that millisecond.
   - `FakeClock(start_ms)` has `set(ms)`, and its `wait_until(ms)` jumps time forward and counts the jump in `.waits`.
6. **`yield_point()`:** call it inside the lock, just before saving `last` and `sequence`.

### Names the driver uses

- `SnowflakeGenerator(machine_id, clock, sequence_bits=12)`: `next_id`, `decode`
- `SystemClock()`, `FakeClock(start_ms)` with `set`, `waits`
- `ClockMovedBackwardsError`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


if __name__ == "__main__":
    g0 = SnowflakeGenerator(3, FakeClock(5))
    i0 = g0.next_id()
    print(i0, i0 >> 22, (i0 >> 12) & 1023, i0 & 4095)   # 20983808 5 3 0   -> 41 bits time | 10 bits machine | 12 bits sequence

    clock = FakeClock(1000)
    g = SnowflakeGenerator(7, clock, sequence_bits=2)   # -> only 4 ids per millisecond, to test overflow
    ids = [g.next_id() for _ in range(5)]
    print([(ts, seq) for ts, _, seq in map(g.decode, ids)])
                                                         # [(1000, 0), (1000, 1), (1000, 2), (1000, 3), (1001, 0)]
                                                         # -> sequence ran out, so it waited for the next ms
    print(clock.waits, ids == sorted(ids), g.decode(ids[0])[1])
                                                         # 1 True 7

    clock.set(1003)
    print(g.decode(g.next_id()))                         # (1003, 7, 0)
    clock.set(1000)                                      # -> clock jumps back 3 ms (small)
    print(g.decode(g.next_id()))                         # (1003, 7, 1)   -> waited until 1003, so no duplicate
    clock.set(900)                                       # -> clock jumps back 103 ms (big)
    try:
        g.next_id()
    except ClockMovedBackwardsError:
        print("refused")                                 # refused
    try:
        SnowflakeGenerator(1024, clock)
    except ValueError:
        print("bad machine id")                          # bad machine id

    real = SnowflakeGenerator(1, SystemClock())
    results, lock, barrier = [], threading.Lock(), threading.Barrier(4)
    def worker():
        barrier.wait()
        mine = [real.next_id() for _ in range(2000)]
        with lock: results.append(mine)
    threads = [threading.Thread(target=worker) for _ in range(4)]
    for t in threads: t.start()
    for t in threads: t.join()
    every = [i for mine in results for i in mine]
    print(len(every), len(set(every)))                   # 8000 8000   -> unique across threads
    print(all(mine == sorted(mine) for mine in results)) # True   -> each thread sees increasing ids
```


---

## C03 · Parking Lot with Many Gates — Lock the scarce thing where it lives (M1)

**Primitive:** one lock per resource + `try_xxx() -> bool` · **Difficulty:** Easy–Medium · **Time:** ~25 min · **Source:** `#01 · Parking Lot` (requirement 8)

### Problem

Many entry and exit gates run at the same time. **Two gates must never give out the same spot**, and one ticket must never exit twice. Don't use one big lock on the lot. **Each spot guards itself**, so gates contend only when they want the same spot.

1. **Spot:**
   - `try_occupy(plate, vehicle_type)` does "check free and type matches, then occupy" **as one step under the spot's lock**. Exactly one caller wins.
   - `release(plate)` frees the spot only if that plate is the one parked there.
   - Call `yield_point()` between the check and the write.
2. **Floor:** copy its spot list under its own lock, **release that lock**, then try the spots one by one. Never hold two locks at once.
3. **Lot:**
   - `enter(plate, type)` tries the floors in the order they were added. It returns ticket `T1`, `T2`, … (ids from a counter under the lot lock), or `None` if no spot fits.
   - `exit(ticket_id)` **claims and removes the ticket in one step** under the lot lock (call `yield_point()` between finding it and deleting it). It then releases the spot and returns the spot code. A ticket that is unknown or already used gives `None`.
4. `free_count(type)` is computed from the spots.

### Names the driver uses

- `VehicleType`: `BIKE`, `CAR`
- `ParkingLot()`: `add_floor(n)`, `add_spot(floor, code, vehicle_type)`, `enter(plate, vehicle_type) -> ticket_id or None`, `exit(ticket_id) -> spot_code or None`, `free_count(vehicle_type)`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


def race(n, fn):
    barrier, out, lock = threading.Barrier(n), [], threading.Lock()
    def run(i):
        barrier.wait()
        r = fn(i)
        with lock: out.append(r)
    ts = [threading.Thread(target=run, args=(i,)) for i in range(n)]
    for t in ts: t.start()
    for t in ts: t.join()
    return out


if __name__ == "__main__":
    lot = ParkingLot()
    lot.add_floor(1)
    lot.add_floor(2)
    for i in range(3): lot.add_spot(1, f"F1-C{i}", VehicleType.CAR)
    for i in range(2): lot.add_spot(2, f"F2-C{i}", VehicleType.CAR)
    lot.add_spot(2, "F2-B0", VehicleType.BIKE)

    tickets = [t for t in race(20, lambda i: lot.enter(f"KA-{i}", VehicleType.CAR)) if t]
    print(len(tickets), len(set(tickets)))             # 5 5    -> 20 cars at 20 gates, 5 car spots, 5 winners
    print(lot.free_count(VehicleType.CAR), lot.free_count(VehicleType.BIKE))
                                                       # 0 1
    winners = race(10, lambda i: lot.exit(tickets[0]))
    print(sum(1 for w in winners if w is not None))    # 1      -> one ticket can exit only once
    print(lot.free_count(VehicleType.CAR))             # 1

    for t in tickets[1:]: lot.exit(t)
    def churn(i):
        ok = 0
        for k in range(200):
            t = lot.enter(f"G{i}-{k}", VehicleType.CAR)
            if t and lot.exit(t): ok += 1
        return ok
    done = race(8, churn)
    print(sum(done) > 0, lot.free_count(VehicleType.CAR))
                                                       # True 5  -> after the storm every spot is free again
```


---

## C04 · Seat Holds That Expire — M1 + Hold → confirm → expire (M6)

**Primitive:** per-seat lock, all-or-nothing undo, per-record lock for confirm vs expire · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#02 · BookMyShow` (requirements 5 and 6)

### Problem

Users hold several seats at once, then confirm before the hold expires. Three races must be safe:
- Two users tap the same seat.
- Overlapping multi-seat requests must never leave a **half-held** booking behind.
- A confirm racing the expiry sweep at the deadline: **exactly one** of them wins.

1. **`ShowSeat`:** `try_hold()` does "check `AVAILABLE`, then set `HELD`" as one step under the seat lock (`yield_point()` between them). It also has `confirm()` (set `BOOKED`) and `release()` (set `AVAILABLE`).
2. **`hold_seats(seat_numbers, now)`:**
   - Hold the seats one by one. If any is unknown or taken, **release only the seats this call held** and return `None`.
   - Otherwise create booking `B1`, `B2`, … with `expires_at = now + hold_ttl`.
3. **`Booking`** has its own lock around `confirm` and `expire`:
   - `confirm(now)` works only while `PENDING`. **If `now ≥ expires_at`, it expires the booking itself** (releases the seats, sets `EXPIRED`) and returns `False`. This keeps it safe even when the sweep is late. Otherwise it books the seats and sets `CONFIRMED`.
   - `expire(now)` works only while `PENDING` and `now ≥ expires_at`. It releases the seats and sets `EXPIRED`.
   - Call `yield_point()` in both, between the check and the change.
4. **`expire_stale(now)`** copies the booking list under the service lock, then expires each booking without that lock. It returns how many expired.
5. **Lock order:** booking lock, then seat lock. Never the other way round.

### Names the driver uses

- `ShowSeatStatus`: `AVAILABLE`, `HELD`, `BOOKED` · `BookingStatus`: `PENDING`, `CONFIRMED`, `EXPIRED`
- `BookingService(seat_numbers, hold_ttl)`: `hold_seats(seat_numbers, now) -> Booking or None`, `confirm_booking(booking_id, now)`, `expire_stale(now) -> int`, `seat_status(seat_number)`
- Booking: `booking_id`, `get_status()`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


def race(fns):
    barrier, out, lock = threading.Barrier(len(fns)), {}, threading.Lock()
    def run(i):
        barrier.wait()
        r = fns[i]()
        with lock: out[i] = r
    ts = [threading.Thread(target=run, args=(i,)) for i in range(len(fns))]
    for t in ts: t.start()
    for t in ts: t.join()
    return [out[i] for i in range(len(fns))]


if __name__ == "__main__":
    svc = BookingService(["A1", "A2", "A3"], hold_ttl=10)
    b = svc.hold_seats(["A1", "A2"], now=0)
    print(svc.hold_seats(["A3", "A2"], now=1))              # None   -> A2 taken
    print(svc.seat_status("A3").name)                       # AVAILABLE  -> this call's hold on A3 was undone
    print(svc.confirm_booking(b.booking_id, now=15))        # False  -> confirm checks expiry itself, the sweep has not run yet
    print(b.get_status().name, svc.seat_status("A1").name)  # EXPIRED AVAILABLE

    svc = BookingService(["A1", "A2"], hold_ttl=10)
    got = race([lambda: svc.hold_seats(["A1", "A2"], now=0) for _ in range(8)])
    print(sum(1 for g in got if g))                         # 1      -> 8 users tap the same seats, one wins

    seats = [f"S{i}" for i in range(11)]
    svc = BookingService(seats, hold_ttl=10)
    got = race([lambda i=i: svc.hold_seats([f"S{i}", f"S{i + 1}"], now=0) for i in range(10)])
    wins = [g for g in got if g]
    held = sum(1 for s in seats if svc.seat_status(s) == ShowSeatStatus.HELD)
    print(held == 2 * len(wins), len(wins) >= 1)            # True True  -> no half-held booking left behind

    consistent = True
    for _ in range(50):
        svc = BookingService(["X"], hold_ttl=10)
        bk = svc.hold_seats(["X"], now=0)
        confirmed, expired = race([lambda: svc.confirm_booking(bk.booking_id, now=9), lambda: svc.expire_stale(now=10)])
        seat = svc.seat_status("X")
        ok = (confirmed and expired == 0 and seat == ShowSeatStatus.BOOKED) or \
             (not confirmed and expired == 1 and seat == ShowSeatStatus.AVAILABLE)
        consistent = consistent and ok
    print(consistent)                                       # True   -> confirm and expire never both win
```


---

## C05 · Meeting Room Calendars — Atomic overlap check + reserve-before-release

**Primitive:** per-room lock around "check free + store" · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#24 · Meeting Room Booking` (the same idea is `CrewMember.try_assign` in `#28` and `Cab.try_reserve` in `#25`)

### Problem

Each room owns a calendar of reserved slots. Two people booking overlapping slots at the same moment must not both succeed. When a meeting moves to another room or slot, **reserve the new slot first and release the old one only after that**. That way a failed change never loses the original booking.

1. A slot `[start, end)` overlaps another when `a.start < b.end and b.start < a.end`. Times are minutes (`int`).
2. **Room** (one lock):
   - `try_reserve(booking_id, slot)` checks for overlaps and stores the slot **as one step** (`yield_point()` between them).
   - `replace(booking_id, new_slot)` does the same but ignores the booking's own current slot.
   - It also has `release(booking_id)` and `holds(booking_id)`.
3. **`book(room_id, slot)`** returns `M1`, `M2`, …, or `None` if the slot overlaps another booking.
4. **`change_booking(booking_id, new_room_id, new_slot)`:**
   - Hold that booking's own lock, so two changes to one meeting run one after the other.
   - Same room → `replace`.
   - Different room → `try_reserve` on the new room first. Only if that succeeds, release the old room and update the record.
   - **Never hold two room locks at once.**
5. **Lock order:** booking lock, then a room lock.

### Names the driver uses

- `TimeSlot(start, end)`
- `MeetingService()`: `add_room(room_id)`, `book(room_id, slot)`, `change_booking(booking_id, new_room_id, new_slot)`, `where(booking_id) -> room_id`, attribute `rooms` (`{room_id: Room}`)
- Room: `holds(booking_id)`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


def race(fns):
    barrier, out, lock = threading.Barrier(len(fns)), {}, threading.Lock()
    def run(i):
        barrier.wait()
        r = fns[i]()
        with lock: out[i] = r
    ts = [threading.Thread(target=run, args=(i,)) for i in range(len(fns))]
    for t in ts: t.start()
    for t in ts: t.join()
    return [out[i] for i in range(len(fns))]


if __name__ == "__main__":
    svc = MeetingService()
    svc.add_room("R1")
    svc.add_room("R2")
    got = race([lambda i=i: svc.book("R1", TimeSlot(540 + i, 600 + i)) for i in range(10)])
    print(sum(1 for g in got if g))           # 1      -> 10 overlapping requests, one wins
    got = race([lambda i=i: svc.book("R2", TimeSlot(60 * i, 60 * i + 60)) for i in range(10)])
    print(sum(1 for g in got if g))           # 10     -> back-to-back slots do not overlap, all win

    svc = MeetingService()
    svc.add_room("R1")
    m = svc.book("R1", TimeSlot(540, 600))
    print(svc.change_booking(m, "R1", TimeSlot(570, 630)))
                                              # True   -> a change may overlap its own old slot
    print(svc.book("R1", TimeSlot(600, 620))) # None   -> 600 to 620 is now inside the changed meeting

    consistent = True
    for _ in range(30):
        svc = MeetingService()
        svc.add_room("R1")
        svc.add_room("R2")
        x = svc.book("R1", TimeSlot(540, 600))
        changed, other = race([lambda: svc.change_booking(x, "R2", TimeSlot(660, 720)),
                               lambda: svc.book("R2", TimeSlot(660, 720))])
        exactly_one = changed != (other is not None)
        never_lost = (svc.where(x) == "R2" and svc.rooms["R2"].holds(x) and not svc.rooms["R1"].holds(x)) or \
                     (not changed and svc.where(x) == "R1" and svc.rooms["R1"].holds(x))
        consistent = consistent and exactly_one and never_lost
    print(consistent)                         # True   -> new slot is reserved before the old one is released
```


---

## C06 · Ride Matching on Stale Snapshots — Decide on copies, reservation may refuse (M10)

**Primitive:** lock-free decision on snapshots + `try_reserve` + retry · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#08 · Ride Sharing` (the same pattern is used for elevators in `#04` and cabs in `#25`)

### Problem

Many riders request at once. The matching rule looks at **read-only snapshots** of the free drivers, with no lock held, so a snapshot may already be **stale** by the time the service tries to reserve that driver. **The reservation is the real check:** if it refuses, exclude that driver and decide again. Nobody is double-booked, and nobody gives up while a free driver exists.

1. **Driver** (own lock):
   - `snapshot()` returns `DriverInfo(driver_id, location)`, or `None` if the driver is busy.
   - `try_reserve()` does "check free, then set busy" as one step (`yield_point()` between them).
   - `release()` frees the driver.
2. **The nearest rule:** `select(candidates, pickup)` picks the smallest `|location − pickup|`, with ties going to the smaller id. It returns `None` for an empty list. Locations are integers on a line.
3. **`request_ride(rider_id, pickup)`** loops:
   1. Snapshot every free driver that is not excluded.
   2. Let the rule choose. If nothing is chosen, return `None`.
   3. Call `try_reserve`. On success return the driver id; on refusal exclude that driver and loop again.
4. `complete_ride(driver_id)` releases the driver, and returns `False` for an unknown id.
5. `available_count()` is computed from snapshots.
6. The driver's slow rule subclasses **your** `NearestDriverStrategy` to widen the race window.

### Names the driver uses

- `DriverInfo(driver_id, location)`, `NearestDriverStrategy` with `select(candidates, pickup)` (the driver subclasses it)
- `RideService(strategy)`: `add_driver(driver_id, location)`, `request_ride(rider_id, pickup) -> driver_id or None`, `complete_ride(driver_id)`, `available_count()`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


class SlowNearest(NearestDriverStrategy):
    def select(self, candidates, pickup):
        yield_point()                                # -> widen the gap between snapshot and reserve
        return super().select(candidates, pickup)


def race(fns):
    barrier, out, lock = threading.Barrier(len(fns)), {}, threading.Lock()
    def run(i):
        barrier.wait()
        r = fns[i]()
        with lock: out[i] = r
    ts = [threading.Thread(target=run, args=(i,)) for i in range(len(fns))]
    for t in ts: t.start()
    for t in ts: t.join()
    return [out[i] for i in range(len(fns))]


if __name__ == "__main__":
    svc = RideService(SlowNearest())
    for i in range(1, 6): svc.add_driver(f"D{i}", i)
    print(svc.request_ride("solo", 4))               # D4
    svc.complete_ride("D4")

    got = race([lambda i=i: svc.request_ride(f"U{i}", 0) for i in range(12)])
    matched = [g for g in got if g]
    print(len(matched), len(set(matched)))           # 5 5    -> 12 riders, 5 drivers, nobody double-booked
    print(sorted(matched), got.count(None))          # ['D1', 'D2', 'D3', 'D4', 'D5'] 7
    print(svc.available_count())                     # 0
    for d in matched: svc.complete_ride(d)
    print(svc.available_count())                     # 5
    print(svc.request_ride("late", 10), svc.complete_ride("D9"))
                                                     # D5 False
```


---

## C07 · Amazon Locker — Per-compartment lock, put-if-absent codes, collect vs expire

**Primitive:** per-resource lock, atomic `put_if_absent` with retry, per-record lock deciding one outcome once · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#18 · Amazon Locker`

### Problem

Delivery agents deposit packages concurrently, and customers collect them with a code before it expires. Four things must hold under concurrency:
- Two packages never share a compartment.
- Every active code is unique, even when two deposits draw the same random code.
- A code hands out its package once.
- A collection racing the expiry sweep: **exactly one** of them wins.

1. **Compartment** (own lock): `try_occupy(package_id, size)` succeeds only if it is empty and big enough (`SMALL < MEDIUM < LARGE`). Check and occupy happen as one step, with `yield_point()` between them.
2. **`deposit(package_id, size, now)`:**
   - Copy the compartment list under the station lock, sort it smallest first (ties by id), and `try_occupy` each one until a compartment accepts. Return `None` if none does.
   - Then draw codes from the generator until `put_if_absent(code, assignment)` succeeds. That method checks and inserts under the station lock, with `yield_point()` between.
   - Return the code. The assignment expires at `now + ttl`.
3. **`Assignment`** (own lock) decides **once**:
   - `collect(now)` works only while `WAITING` and `now < expires_at`.
   - `expire(now)` works only while `WAITING` and `now ≥ expires_at`.
   - Both free the compartment. Use `yield_point()` between the check and the change.
4. **`collect(code, now)`** returns the package id, or `None`, and removes the code when it succeeds. **`expire_stale(now)`** returns the list of expired package ids.
5. **The test code generator:** `SequenceCodeGenerator(codes)` returns the codes in order, looping, and is thread-safe.

### Names the driver uses

- `Size`: `SMALL`, `MEDIUM`, `LARGE`
- `SequenceCodeGenerator(codes)`
- `LockerStation(code_generator, ttl)`: `add_compartment(cid, size)`, `deposit(package_id, size, now) -> code or None`, `collect(code, now) -> package_id or None`, `expire_stale(now) -> list`, `free_count()`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


def race(fns):
    barrier, out, lock = threading.Barrier(len(fns)), {}, threading.Lock()
    def run(i):
        barrier.wait()
        r = fns[i]()
        with lock: out[i] = r
    ts = [threading.Thread(target=run, args=(i,)) for i in range(len(fns))]
    for t in ts: t.start()
    for t in ts: t.join()
    return [out[i] for i in range(len(fns))]


def station(codes, sizes):
    s = LockerStation(SequenceCodeGenerator(codes), ttl=10)
    for i, size in enumerate(sizes): s.add_compartment(f"C{i}", size)
    return s


if __name__ == "__main__":
    s = station(["777", "777", "888"], [Size.SMALL, Size.LARGE])
    print(s.deposit("P1", Size.SMALL, now=0))        # 777
    print(s.deposit("P2", Size.SMALL, now=0))        # 888    -> 777 was taken, so it drew again
    print(s.deposit("P3", Size.LARGE, now=0))        # None   -> the large one already holds P2

    s = station([str(n) for n in range(100, 200)], [Size.SMALL, Size.SMALL, Size.LARGE])
    codes = race([lambda i=i: s.deposit(f"P{i}", Size.SMALL, now=0) for i in range(6)])
    ok = [c for c in codes if c]
    print(len(ok), len(set(ok)), s.free_count())     # 3 3 0  -> 6 agents, 3 compartments

    s = station(["555", "555", "555", "666"], [Size.SMALL, Size.SMALL])
    codes = race([lambda i=i: s.deposit(f"P{i}", Size.SMALL, now=0) for i in range(2)])
    print(sorted(codes))                             # ['555', '666']  -> same code drawn twice, still unique

    got = race([lambda: s.collect("555", now=5) for _ in range(10)])
    print(sum(1 for g in got if g), s.free_count())  # 1 1    -> one package handed out once

    consistent = True
    for _ in range(50):
        s = station(["900"], [Size.SMALL])
        s.deposit("PX", Size.SMALL, now=0)
        collected, expired = race([lambda: s.collect("900", now=9), lambda: s.expire_stale(now=10)])
        consistent = consistent and ((collected is None) != (expired == [])) and s.free_count() == 1
    print(consistent)                                # True   -> collected or expired, never both
```


---

## C08 · Wallet Transfers — Two locks in a fixed order + idempotency key (M8)

**Primitive:** ordered locking (deadlock avoidance), check-then-change under the lock, claim the key first · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#27 · Digital Wallet`, template `12_locks_mutex.py`

### Problem

Transfers move money between two wallets, and each wallet guards its balance with its own lock. A transfer needs **both** locks. Taking them in call order deadlocks when A→B and B→A run at the same time, so **always lock the wallet created first, first**. Every request also carries an **idempotency key**: a retried request is applied once.

1. `create_wallet(balance)` returns `W1`, `W2`, …. Each wallet also keeps its creation number, which is used for lock order.
2. **`transfer(key, from_id, to_id, amount)` returns a `Transaction`:**
   1. **Claim the key first.** Under the service lock: if the key is known, return the **same** transaction object. Otherwise store a new `PENDING` transaction (`X1`, `X2`, …) under that key.
   2. Outside the service lock, lock both wallets **in creation order**. If `from` has less than `amount`, mark the transaction `FAILED`. Otherwise move the money (`yield_point()` between reading the balance and writing it) and mark it `SUCCESS`.
   3. Unknown wallets, the same wallet twice, or `amount ≤ 0` give `FAILED`.
3. A balance never goes below 0. Money is never created or lost.
4. **Never hold the service lock while holding wallet locks.**

> If your lock order is wrong, the driver prints `False deadlocked` and stops instead of hanging.

### Names the driver uses

- `WalletService()`: `create_wallet(balance) -> wallet_id`, `transfer(key, from_id, to_id, amount) -> Transaction`, `get_balance(wallet_id)`
- Transaction: `txn_id`, `status` (`"PENDING"`, `"SUCCESS"`, `"FAILED"`)

### Driver code

```python
import threading
import time
import uuid


def yield_point():
    time.sleep(0)


def run_all(fns, timeout=20):
    barrier, out, lock = threading.Barrier(len(fns)), {}, threading.Lock()
    def run(i):
        barrier.wait()
        r = fns[i]()
        with lock: out[i] = r
    ts = [threading.Thread(target=run, args=(i,), daemon=True) for i in range(len(fns))]
    for t in ts: t.start()
    for t in ts: t.join(timeout)
    finished = not any(t.is_alive() for t in ts)
    return finished, [out.get(i) for i in range(len(fns))]


if __name__ == "__main__":
    svc = WalletService()
    a, b = svc.create_wallet(100), svc.create_wallet(100)
    print(svc.transfer("k1", a, b, 30).status, svc.get_balance(a), svc.get_balance(b))
                                                       # SUCCESS 70 130
    print(svc.transfer("k2", a, b, 500).status)        # FAILED  -> not enough money, nothing moved
    print(svc.transfer("k1", a, b, 30).status, svc.get_balance(a))
                                                       # SUCCESS 70   -> same key, not applied again

    def ping(src, dst):
        return lambda: [svc.transfer(str(uuid.uuid4()), src, dst, 1) for _ in range(1500)]
    finished, _ = run_all([ping(a, b), ping(b, a)])
    print(finished, svc.get_balance(a) + svc.get_balance(b) if finished else "deadlocked")
                                                       # True 200  -> opposite transfers, no deadlock, no money lost
    if not finished: raise SystemExit("deadlock: lock the two wallets in a fixed order")

    shop, buyer = svc.create_wallet(0), svc.create_wallet(100)
    finished, txns = run_all([lambda: svc.transfer(str(uuid.uuid4()), buyer, shop, 10) for _ in range(20)])
    print(sum(1 for t in txns if t.status == "SUCCESS"), svc.get_balance(buyer), svc.get_balance(shop))
                                                       # 10 0 100   -> never overdrawn

    payer, payee = svc.create_wallet(50), svc.create_wallet(0)
    finished, txns = run_all([lambda: svc.transfer("same-key", payer, payee, 50) for _ in range(10)])
    print(len({t.txn_id for t in txns}), svc.get_balance(payer), svc.get_balance(payee))
                                                       # 1 0 50     -> 10 retries of one request, charged once
```


---

## C09 · Thread-safe LRU Cache — One lock, and reads take it too

**Primitive:** a single lock around a multi-pointer data structure · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#07 · LRU / LFU Cache`, template `23_lru_cache.py`

### Problem

Build an LRU cache from a hashmap plus a doubly linked list with sentinel head and tail. Many threads call `get` and `put` at once. **A `get` moves the node to the front, so it changes the list and needs the lock just like `put`.** Without the lock, pointer updates from two threads interleave and the list breaks.

1. `get(key)` returns the value and moves the node to the front, or returns `None`.
2. `put(key, val)`:
   - Updating an existing key replaces its node and moves it to the front.
   - Inserting into a full cache evicts the tail's node (the least recently used) first.
3. **List helpers:**
   - `_remove(node)`: call `yield_point()` between the two pointer updates.
   - `_add_front(node)`: call `yield_point()` after setting the node's own pointers.
4. `size()` returns the number of entries.
5. **`check_consistent()`** walks the list from head to tail, under the lock, and returns `True` only if all of these hold:
   - Every `node.next.prev is node`.
   - The keys in the list equal the keys in the map, with the same count.
   - The size is ≤ capacity.

### Names the driver uses

- `LRUCache(capacity)`: `get`, `put`, `size`, `check_consistent`

### Driver code

```python
import random
import threading
import time


def yield_point():
    time.sleep(0)


def run_all(fns):
    barrier = threading.Barrier(len(fns))
    def run(f):
        barrier.wait()
        f()
    ts = [threading.Thread(target=run, args=(f,)) for f in fns]
    for t in ts: t.start()
    for t in ts: t.join()


if __name__ == "__main__":
    c = LRUCache(2)
    c.put("a", 1)
    c.put("b", 2)
    c.get("a")                                    # -> a is now the most recent
    c.put("c", 3)                                 # -> evicts b
    print(c.get("b"), c.get("a"), c.get("c"))     # None 1 3
    c.put("a", 10)
    print(c.get("a"), c.size(), c.check_consistent())
                                                  # 10 2 True

    big = LRUCache(100)
    run_all([lambda t=t: [big.put(f"t{t}-{k}", k) for k in range(25)] for t in range(4)])
    print(big.size(), all(big.get(f"t{t}-{k}") == k for t in range(4) for k in range(25)))
                                                  # 100 True   -> 4 writers, nothing lost

    small = LRUCache(20)
    def hammer(seed):
        rnd = random.Random(seed)
        for _ in range(3000):
            k = rnd.randrange(60)
            if rnd.random() < 0.5: small.put(k, k * 10)
            else:
                v = small.get(k)
                assert v is None or v == k * 10
    run_all([lambda s=s: hammer(s) for s in range(8)])
    print(small.size(), small.check_consistent()) # 20 True   -> reads move nodes too, so they need the lock
```


---

## C10 · Rate Limiter Under a Burst — Per-key lock + safe lazy creation

**Primitive:** lock per tracker, atomic get-or-create under the limiter lock · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#06 · Rate Limiter` (requirement 6), template `22_rate_limiter.py`

### Problem

Fifty requests from one client arrive in the same instant, and only `limit` of them may pass. Two simultaneous requests must never both take the last token. Different clients must not wait for each other, so each gets **its own tracker with its own lock**. The first requests for a new client race to create its tracker, and **exactly one tracker may be created and used**.

1. **Trackers** take an injected clock, so time only moves when the driver moves it. Each does "check, then change" as one step under its own lock, with `yield_point()` between them.
   - **`TokenBucket(capacity, rate, clock)`:** starts full and refills `elapsed × rate`, capped at capacity. It needs 1 whole token per request.
   - **`FixedWindow(limit, window, clock)`:** allows `limit` requests per window. The window restarts once `now − start ≥ window`.
2. **`RateLimiter(make_tracker)`:**
   - `allow(key)` gets the key's tracker, or creates it with `make_tracker()`. **The lookup and the insert happen as one step under the limiter lock** (`yield_point()` between them).
   - It then calls `try_acquire()` **after releasing** the limiter lock.
   - `get_tracker_count()` returns how many trackers exist.
3. `FakeClock(t=0.0)` has `now()` and `advance(seconds)`.

### Names the driver uses

- `FakeClock()`, `TokenBucket(capacity, rate, clock)`, `FixedWindow(limit, window, clock)`
- `RateLimiter(make_tracker)`: `allow(key) -> bool`, `get_tracker_count()`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


def burst(fns):
    barrier, out, lock = threading.Barrier(len(fns)), [], threading.Lock()
    def run(f):
        barrier.wait()
        r = f()
        with lock: out.append(r)
    ts = [threading.Thread(target=run, args=(f,)) for f in fns]
    for t in ts: t.start()
    for t in ts: t.join()
    return out


if __name__ == "__main__":
    clock = FakeClock()                                   # -> time is frozen unless the driver advances it
    rl = RateLimiter(lambda: TokenBucket(capacity=10, rate=1, clock=clock))

    got = burst([lambda: rl.allow("new-user") for _ in range(50)])
    print(sum(got), rl.get_tracker_count())               # 10 1   -> 50 first-time requests share one bucket

    got = burst([lambda k=k: rl.allow(k) for k in ["a", "b"] * 30])
    print(sum(got), rl.get_tracker_count())               # 20 3   -> 10 for a, 10 for b, separate locks

    clock.advance(3)                                      # -> 3 tokens refilled
    got = burst([lambda: rl.allow("a") for _ in range(8)])
    print(sum(got))                                       # 3

    fw = RateLimiter(lambda: FixedWindow(limit=5, window=10, clock=clock))
    got = burst([lambda: fw.allow("svc") for _ in range(20)])
    print(sum(got))                                       # 5
    clock.advance(10)
    print(sum(burst([lambda: fw.allow("svc") for _ in range(20)])))
                                                          # 5      -> new window
```


---

## C11 · Logging From Many Threads — One lock per destination

**Primitive:** fine-grained locks (one per destination), not one global lock · **Difficulty:** Easy–Medium · **Time:** ~20 min · **Source:** `#09 · Logging Framework` (requirement 8)

### Problem

Many threads log at once. A destination writes each line in **several pieces**, so without a lock, pieces from different threads interleave into garbage. Give **each destination its own lock** around the whole write. Two different destinations must never wait for each other, which is why you **don't** put one lock around the whole of `log`.

1. **`BufferDestination.write(level, message)`:**
   - Under its own lock, append three chunks to `self.chunks`: the level name, `": "`, and `message + "\n"`. Call `yield_point()` between the chunks.
   - The lock must be stored as `self.lock`, because the driver's slow destination subclasses yours and uses it.
2. `get_lines()` joins the chunks and splits them into lines.
3. **`Logger(min_level, destinations)`:**
   - `log` reads `min_level` under the logger's lock, **releases it**, drops entries below the minimum (`False`), and otherwise writes to each destination (`True`).
   - `set_min_level` takes the same lock.
4. No method holds two locks at once.

### Names the driver uses

- `LogLevel`: `DEBUG`, `INFO`, `ERROR`
- `BufferDestination()`: `write(level, message)`, `get_lines()`, attributes `chunks`, `lock`
- `Logger(min_level, destinations)`: `log`, `set_min_level`

### Driver code

```python
import re
import threading
import time


def yield_point():
    time.sleep(0)


class SlowDestination(BufferDestination):
    def write(self, level, message):
        with self.lock:
            time.sleep(0.3)                          # -> holds its own lock for a long time
            self.chunks.append(f"{level.name}: {message}\n")
        return True


def run_all(fns):
    barrier = threading.Barrier(len(fns))
    def run(f):
        barrier.wait()
        f()
    ts = [threading.Thread(target=run, args=(f,)) for f in fns]
    for t in ts: t.start()
    for t in ts: t.join()


if __name__ == "__main__":
    console, file_ = BufferDestination(), BufferDestination()
    logger = Logger(LogLevel.INFO, [console, file_])
    print(logger.log(LogLevel.DEBUG, "skip"), logger.log(LogLevel.INFO, "start"))
                                                     # False True
    def spam(t):
        for k in range(300):
            logger.log(LogLevel.ERROR if k % 2 else LogLevel.INFO, f"t{t}-{k}")
    run_all([lambda t=t: spam(t) for t in range(8)])
    lines = console.get_lines()
    good = all(re.fullmatch(r"(INFO|ERROR): (t\d+-\d+|start)", l) for l in lines)
    print(len(lines), good, len(file_.get_lines()))
                                                     # 2401 True 2401   -> no line was mixed with another

    slow, fast = SlowDestination(), BufferDestination()
    took = {}
    def write_fast():
        time.sleep(0.05)                             # -> start after slow has taken its lock
        t0 = time.monotonic()
        fast.write(LogLevel.INFO, "quick")
        took["fast"] = time.monotonic() - t0
    run_all([lambda: slow.write(LogLevel.ERROR, "db write"), write_fast])
    print(took["fast"] < 0.1, slow.get_lines(), fast.get_lines())
                                                     # True ['ERROR: db write'] ['INFO: quick']
                                                     # -> each destination has its own lock, so fast did not wait

    logger.set_min_level(LogLevel.ERROR)
    print(logger.log(LogLevel.INFO, "x"), logger.log(LogLevel.ERROR, "y"))
                                                     # False True
```


---

## C12 · KV Store with TTL + Transactions — Private layers, one-lock commit

**Primitive:** thread-confined state (no lock needed) + one lock for an atomic multi-key apply · **Difficulty:** Medium–Hard · **Time:** ~35 min · **Source:** `#21 · KV Store with TTL + Transactions`

### Problem

The shared store holds **only committed data**. Each session belongs to one thread and keeps a **stack of change layers**, one per open transaction. Those layers are private, which is why uncommitted changes stay hidden and **need no lock**. An outer commit applies all of its changes **in one hold of the store lock**, so no reader ever sees half of a commit.

1. **Store** (one lock):
   - `get(key)` and `get_many(keys)` (reads all the keys under one lock).
   - `apply(changes)` writes `{key: (value, ttl)}` under one lock, with `yield_point()` after each key.
   - `purge_expired()` returns how many keys it removed.
   - **Expiry is lazy:** a key whose `expires_at ≤ now` reads as `None` even before it is purged.
2. **Session:**
   - `begin()` pushes a layer.
   - `set(key, value, ttl=None)` and `delete(key)` write to the top layer. **A delete is a tombstone.** With no open layer, they apply straight to the store.
   - `get` looks through the session's own layers top-down, then the store.
   - `commit()`: an inner commit merges the top layer into the layer below. The outer commit calls `store.apply(...)` once.
   - `rollback()` pops the top layer.
   - `commit` and `rollback` with no open transaction return `False`.
3. **TTL:** `expires_at = now + ttl`, set when the change is applied.
4. `FakeClock(t=0)` has `now()` and `advance(seconds)`.

### Names the driver uses

- `FakeClock()`, `KeyValueStore(clock)`: `get`, `get_many(keys)`, `purge_expired()`
- `Session(store)`: `begin`, `set(key, value, ttl=None)`, `delete`, `get`, `commit`, `rollback`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


if __name__ == "__main__":
    clock = FakeClock()
    store = KeyValueStore(clock)
    s1, s2 = Session(store), Session(store)

    s1.set("x", 1)
    print(s2.get("x"))                           # 1      -> no transaction, written straight away
    s1.begin()
    s1.set("x", 2)
    print(s1.get("x"), s2.get("x"))              # 2 1    -> uncommitted change is hidden from others
    s1.begin()
    s1.delete("x")
    print(s1.get("x"))                           # None   -> tombstone in the inner layer
    s1.rollback()
    print(s1.get("x"))                           # 2
    s1.begin()
    s1.set("y", 5)
    s1.commit()                                  # -> inner commit merges into the outer layer only
    print(s2.get("y"))                           # None
    s1.commit()
    print(s2.get("x"), s2.get("y"))              # 2 5
    print(s1.commit(), s1.rollback())            # False False  -> no open transaction

    s1.set("otp", "4821", ttl=10)
    s1.set("session", "abc", ttl=30)
    clock.advance(9)
    print(s2.get("otp"))                         # 4821
    clock.advance(1)
    print(s2.get("otp"), store.purge_expired(), s2.get("session"))
                                                 # None 1 abc   -> expired keys hidden on read, then cleaned up

    done, torn = threading.Event(), []
    def writer():
        w = Session(store)
        for i in range(400):
            w.begin()
            w.set("a", i)
            w.set("b", i)
            w.commit()
        done.set()
    def reader():
        while not done.is_set():
            a, b = store.get_many(["a", "b"])
            if a != b: torn.append((a, b))
    threads = [threading.Thread(target=writer)] + [threading.Thread(target=reader) for _ in range(3)]
    for t in threads: t.start()
    for t in threads: t.join()
    print(len(torn), store.get_many(["a", "b"])) # 0 [399, 399]   -> readers never saw half a commit
```


---

## C13 · Per-Endpoint Limit & Connection Pool — Semaphores

**Primitive:** counting semaphore (at most N at once), acquire with timeout · **Difficulty:** Easy–Medium · **Time:** ~20 min · **Source:** `#29 · Webhook Delivery` ("at most `max_in_flight_per_endpoint` workers can be stuck on one endpoint"), template `13_semaphores.py`

### Problem

**Part A: endpoint limiter.** Allow at most `N` concurrent calls **per endpoint**, so one slow customer can tie up at most `N` workers. Every endpoint gets its own semaphore, created lazily. Two threads seeing a new endpoint at the same moment must end up sharing **one** semaphore.

1. `acquire(endpoint, timeout=None)` returns `True` once a permit is taken, or `False` if the timeout runs out. `release(endpoint)` gives a permit back.
2. **Get-or-create the semaphore** in one step under the limiter's lock, with `yield_point()` between the lookup and the insert.

**Part B: connection pool.** `size` connections, `C1..C<size>`. A semaphore counts the free connections, and a lock guards the free list.

1. `acquire(timeout=None)` waits for a permit, then takes a connection from the free list under the lock. Inside the lock, call `yield_point()` between reading the last connection and popping it. It returns `None` on timeout.
2. `release(conn)` puts the connection back, then releases the permit.
3. A connection is never handed to two threads at once. `available()` returns the number of free connections.

### Names the driver uses

- `EndpointLimiter(max_per_endpoint)`: `acquire(endpoint, timeout=None) -> bool`, `release(endpoint)`
- `ConnectionPool(size)`: `acquire(timeout=None) -> Connection or None`, `release(conn)`, `available()`
- Connection: `conn_id`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


def run_all(fns):
    barrier = threading.Barrier(len(fns))
    def run(f):
        barrier.wait()
        f()
    ts = [threading.Thread(target=run, args=(f,)) for f in fns]
    for t in ts: t.start()
    for t in ts: t.join()


if __name__ == "__main__":
    limiter = EndpointLimiter(max_per_endpoint=3)
    active, peak, lock = {"A": 0}, {"A": 0}, threading.Lock()
    def call_a():
        limiter.acquire("A")
        with lock:
            active["A"] += 1
            peak["A"] = max(peak["A"], active["A"])
        time.sleep(0.05)
        with lock: active["A"] -= 1
        limiter.release("A")
    run_all([call_a for _ in range(10)])
    print(peak["A"])                                  # 3      -> never more than 3 at once on one endpoint

    for _ in range(3): limiter.acquire("slow")        # -> a slow customer fills all 3 of its slots
    t0 = time.monotonic()
    print(limiter.acquire("fast", timeout=1), time.monotonic() - t0 < 0.05)
                                                      # True True   -> other endpoints are not blocked
    print(limiter.acquire("slow", timeout=0.05))      # False  -> slow is full, timed out
    limiter.release("slow")
    print(limiter.acquire("slow", timeout=0.05))      # True

    pool = ConnectionPool(3)
    in_use, clashes, ulock = set(), [0], threading.Lock()
    def borrow():
        for _ in range(50):
            c = pool.acquire()
            with ulock:
                if c.conn_id in in_use: clashes[0] += 1
                in_use.add(c.conn_id)
            time.sleep(0)
            with ulock: in_use.discard(c.conn_id)
            pool.release(c)
    run_all([borrow for _ in range(8)])
    print(clashes[0], pool.available())               # 0 3    -> a connection is never shared
    held = [pool.acquire() for _ in range(3)]
    print(pool.acquire(timeout=0.05))                 # None   -> pool exhausted, timed out
    print(sorted(c.conn_id for c in held))            # ['C1', 'C2', 'C3']
```


---

## C14 · Delay Queue & Countdown Latch — Condition variables

**Primitive:** `Condition` with `while`-loop waits, timed waits, notify on every change · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#17 · Task Scheduler` (follow-up: "use a condition variable signalled by `add`, and wait until the next `run_at`"), template `14_condition_variables.py`

### Problem

**Part A: delay queue.** Idle workers must not poll. A taker **sleeps on a condition variable until the earliest item is due**. If an item that is due **sooner** is added while it sleeps, `put` must **wake it**, so it doesn't oversleep until the old earliest item.

1. `put(item, delay)` stores the item as ready at `now + delay` (use `time.monotonic()`) and calls `notify_all()`.
2. `take(timeout=None)` returns the earliest **ready** item, in ready-time order with ties in insert order.
   - While nothing is ready, wait **only until the earliest ready time or the timeout**, whichever comes first, then re-check in a `while` loop.
   - Return `None` on timeout, or when the queue is closed and empty.
3. `close()` wakes every waiter.
4. Each item is taken by exactly one consumer.

**Part B: countdown latch.** Wait until `N` workers have finished.

1. `count_down()` lowers the count, never below 0, and wakes the waiters when it reaches 0.
2. `await_(timeout=None)` returns `True` once the count is 0, or `False` on timeout. `Condition.wait_for` is fine here.

> The driver also catches a **missing `notify_all`**: the first check prints `False`.

### Names the driver uses

- `DelayQueue()`: `put(item, delay)`, `take(timeout=None)`, `close()`
- `CountDownLatch(count)`: `count_down()`, `await_(timeout=None) -> bool`

### Driver code

```python
import threading
import time


if __name__ == "__main__":
    q = DelayQueue()
    q.put("late", 0.4)
    got = {}
    def taker():
        t0 = time.monotonic()
        got["first"] = q.take()
        got["waited"] = time.monotonic() - t0
    t = threading.Thread(target=taker)
    t.start()
    time.sleep(0.05)
    q.put("soon", 0.05)                         # -> must wake the sleeping taker early
    t.join()
    print(got["first"], got["waited"] < 0.3)    # soon True   -> not stuck sleeping until "late" was due
    print(q.take())                             # late

    t0 = time.monotonic()
    print(q.take(timeout=0.05), time.monotonic() - t0 >= 0.05)
                                                # None True   -> empty, waited for the timeout
    q.put("a", 0.06)
    q.put("b", 0.02)
    q.put("c", 0.04)
    print([q.take() for _ in range(3)])         # ['b', 'c', 'a']   -> ready time order, not insert order

    out, olock = [], threading.Lock()
    def consumer():
        while True:
            item = q.take()
            if item is None: return
            with olock: out.append(item)
    consumers = [threading.Thread(target=consumer) for _ in range(4)]
    for c in consumers: c.start()
    for i in range(20): q.put(i, 0)
    time.sleep(0.1)
    q.close()                                   # -> wakes every waiting consumer
    for c in consumers: c.join()
    print(sorted(out) == list(range(20)))       # True   -> each item taken exactly once

    latch = CountDownLatch(3)
    def worker(d):
        time.sleep(d)
        latch.count_down()
    for d in (0.02, 0.04, 0.06): threading.Thread(target=worker, args=(d,)).start()
    print(latch.await_(timeout=2))              # True   -> woke when the third worker finished
    print(latch.await_(timeout=0.01), CountDownLatch(1).await_(timeout=0.05))
                                                # True False
```


---

## C15 · Processing Pipeline — Blocking queue, backpressure, poison pill

**Primitive:** bounded blocking queue between stages, shutdown signal that flows downstream · **Difficulty:** Medium · **Time:** ~30 min · **Source:** templates `15_blocking_queue.py` and `19_worker_pool.py`; "bounded queue … shutdown" in recall Type E and `#19`

### Problem

Chain stages with **bounded blocking queues**: producer → stage 1 → stage 2 → … → collector, one thread per stage. A slow stage must **hold back** everything upstream (backpressure), never let a queue grow past its capacity. Shutting down is a **poison pill**: when a stage receives it, the stage passes it downstream and exits.

1. **`BlockingQueue(capacity)`:**
   - `put` waits while the queue is full, and `get` waits while it is empty. Use one condition, with `while` around every wait.
   - It tracks the **largest size it ever reached**, returned by `get_peak()`.
2. **`Pipeline(stages, capacity)`:**
   - Creates `len(stages) + 1` queues: the input of each stage, plus the final output.
   - `run(items)` starts one thread per stage and a collector thread, puts every item and then the pill, joins every thread, and returns the results **in input order**.
3. A stage function that raises **skips that item**, records `(stage_index, item)` in `get_errors()`, and keeps going.
4. After `run` returns, no pipeline thread is left alive.
5. `get_peaks()` returns each queue's peak size.

### Names the driver uses

- `Pipeline(stages, capacity)`: `run(items) -> list`, `get_errors() -> list[(stage_index, item)]`, `get_peaks() -> list[int]`

### Driver code

```python
import threading
import time


def parse(s): return int(s)
def square(n): return n * n
def slow_format(n):
    time.sleep(0.01)                                  # -> the last stage is the slowest
    return f"<{n}>"


if __name__ == "__main__":
    p = Pipeline([parse, square, slow_format], capacity=2)
    t0 = time.monotonic()
    print(p.run([str(i) for i in range(8)]))          # ['<0>', '<1>', '<4>', '<9>', '<16>', '<25>', '<36>', '<49>']
    print(all(peak <= 2 for peak in p.get_peaks()))   # True   -> backpressure: no queue ever grew past 2
    print(time.monotonic() - t0 >= 0.08)              # True   -> the producer was held back by the slow stage

    p2 = Pipeline([parse, square], capacity=3)
    print(p2.run(["1", "x", "3", "", "5"]))           # [1, 9, 25]
    print(p2.get_errors())                            # [(0, 'x'), (0, '')]   -> bad items skipped, pipeline kept going
    print(threading.active_count())                   # 1      -> every stage thread stopped after the poison pill
```


---

## C16 · Worker Pool with Retries — Fast half queues, slow half works (M7)

**Primitive:** N threads on one queue, retry with exponential backoff, graceful shutdown with one poison pill per worker · **Difficulty:** Medium · **Time:** ~25 min · **Source:** template `19_worker_pool.py`; recall Type E skeleton (`#13`, `#17`, `#26`, `#29`)

### Problem

`submit` only records the job and enqueues it, then returns. `N` worker threads take jobs, run the handler, and retry failures with backoff. `shutdown()` must **finish every job already submitted** (retries included) before the workers stop, and it must be safe to call twice.

1. A `Job(job_id, payload)` has `status` (`PENDING` → `DONE` or `FAILED`) and `attempts`.
2. **`RetryPolicy(max_attempts, base_delay)`:**
   - `should_retry(attempts)` is `attempts < max_attempts`.
   - `delay(attempts)` is `base × 2^(attempts−1)`.
3. **The worker:**
   - Each attempt adds 1 to `attempts` and calls `handler(job)`.
   - If the handler raises, the worker retries after the delay, or marks the job `FAILED` when retries run out.
   - On success the job is `DONE`.
4. **Submitting and shutting down:**
   - `submit(job)` returns `True`, or `False` once shutdown has begun. The closed check and the enqueue happen under one lock, so no job slips in after the pills.
   - `shutdown()` puts **one pill per worker** at the end of the queue and joins every thread.
5. Each attempt runs on exactly one worker.

### Names the driver uses

- `Job(job_id, payload)`: `status`, `attempts`
- `RetryPolicy(max_attempts, base_delay)`
- `WorkerPool(n, handler, retry)`: `submit(job) -> bool`, `shutdown()`

### Driver code

```python
import threading
import time


class FlakyHandler:
    """payload 'fail:k' fails the first k attempts of that job, anything else succeeds."""
    def __init__(self): self.calls, self.lock = {}, threading.Lock()
    def __call__(self, job):
        with self.lock:
            self.calls[job.job_id] = self.calls.get(job.job_id, 0) + 1
            n = self.calls[job.job_id]
        if job.payload.startswith("fail:") and n <= int(job.payload[5:]): raise IOError("network")


if __name__ == "__main__":
    handler = FlakyHandler()
    pool = WorkerPool(3, handler, RetryPolicy(max_attempts=3, base_delay=0.001))
    kinds = ["ok", "fail:1", "fail:2", "fail:5"]
    jobs = [Job(i, kinds[i % 4]) for i in range(12)]
    print(all(pool.submit(j) for j in jobs))          # True   -> submit returns at once
    pool.shutdown()                                   # -> finishes every queued job, retries included
    summary = sorted({(j.payload, j.status, j.attempts) for j in jobs})
    print(summary)                                    # [('fail:1', 'DONE', 2), ('fail:2', 'DONE', 3), ('fail:5', 'FAILED', 3), ('ok', 'DONE', 1)]
    print(all(handler.calls[j.job_id] == j.attempts for j in jobs))
                                                      # True   -> each attempt ran exactly once, on one worker
    print(pool.submit(Job(99, "ok")))                 # False  -> closed
    pool.shutdown()                                   # -> calling it twice is harmless

    slow = WorkerPool(4, lambda job: time.sleep(0.1), RetryPolicy(1, 0))
    t0 = time.monotonic()
    for i in range(8): slow.submit(Job(i, "sleep"))
    slow.shutdown()
    print(time.monotonic() - t0 < 0.35)               # True   -> 4 workers in parallel (one by one is 0.8s)
    print(threading.active_count())                   # 1      -> all workers stopped
```


---

## C17 · Polite Multi-threaded Crawler — Per-site frontier with reserved slots

**Primitive:** condition-variable frontier that reserves a site's next allowed time while handing out work; pending counter for termination · **Difficulty:** Medium–Hard · **Time:** ~35 min · **Source:** `#23 · Web Crawler` (requirement 6, and "`take` reserves the site's next allowed time while holding the lock")

### Problem

Several workers crawl several sites. **A site must never be hit twice within `delay` seconds**, no matter how many workers there are, while different sites are crawled **in parallel**. The frontier keeps one queue per site plus each site's next allowed time. Handing out a URL and **reserving that site's next slot happen in one step** under the lock.

1. **`Frontier(delay)`:**
   - `add(domain, url)` appends to that site's queue and adds 1 to `pending`.
   - `take()` finds a site that has URLs and whose `next_allowed ≤ now`. It sets `next_allowed = now + delay` and returns `(domain, url)`, all under the lock.
   - If no site is ready yet, `take()` waits on the condition **until the soonest `next_allowed`**, then looks again.
   - `done()` lowers `pending` when a worker finishes a URL, and wakes the waiters.
   - `take()` returns `None` only when `pending == 0`, meaning nothing is queued **and** nothing is in flight.
2. **`PoliteCrawler(fetcher, delay, workers)`:**
   - Claims each URL in `seen` (under a lock) before adding it. The domain is `url.split("/")[2]`.
   - Workers loop on take → fetch (with no lock held) → claim the links (words starting with `http://`) → `done()`. **Call `done()` in a `finally`.**
   - `crawl(seeds)` returns the fetched URLs sorted.

> The driver also catches a frontier that forgets to reserve the slot: the gap check prints `False`.

### Names the driver uses

- `PoliteCrawler(fetcher, delay, workers)`: `crawl(seeds) -> list[str]`
- The driver's `RecordingFetcher` provides `fetch(url) -> str or None`

### Driver code

```python
import threading
import time


class RecordingFetcher:
    """Two sites, 4 pages each, every page links to the next one and back to the first."""
    def __init__(self):
        self.times, self.lock = {}, threading.Lock()
        self.pages = {}
        for site in ("a.com", "b.com"):
            for i in range(4):
                self.pages[f"http://{site}/p{i}"] = f"page http://{site}/p{(i + 1) % 4} http://{site}/p0"
    def fetch(self, url):
        with self.lock: self.times.setdefault(url.split("/")[2], []).append(time.monotonic())
        time.sleep(0.01)
        return self.pages.get(url)


if __name__ == "__main__":
    fetcher = RecordingFetcher()
    crawler = PoliteCrawler(fetcher, delay=0.1, workers=4)
    t0 = time.monotonic()
    print(crawler.crawl(["http://a.com/p0", "http://b.com/p0"]))
                                         # ['http://a.com/p0', 'http://a.com/p1', 'http://a.com/p2', 'http://a.com/p3', 'http://b.com/p0', 'http://b.com/p1', 'http://b.com/p2', 'http://b.com/p3']
    elapsed = time.monotonic() - t0
    gaps = [b - a for ts in fetcher.times.values() for a, b in zip(ts, ts[1:])]
    print(len(gaps), min(gaps) >= 0.09)  # 6 True   -> one site is never hit twice within 0.1s, even by 4 threads
    print(elapsed < 0.6)                 # True   -> the two sites were crawled in parallel (one after the other is about 0.8s)
    print(threading.active_count())      # 1      -> workers exit when nothing is queued or in flight
```


---

## C18 · Webhook Lanes Under Load — Atomic in-flight limit + fair blocking take

**Primitive:** condition-variable queue where "check the limit and raise it" is one step; round-robin across lanes · **Difficulty:** Medium–Hard · **Time:** ~35 min · **Source:** `#29 · Webhook Delivery` (requirement 8: one customer's slow endpoint must not delay others)

### Problem

Real worker threads deliver webhooks. One customer's endpoint is slow and has a backlog. If workers took from one shared FIFO, they would all get stuck on that endpoint. Instead:
- Keep **one lane per endpoint**.
- Serve the lanes **round-robin**.
- Cap the **in-flight deliveries per endpoint**. The check against the cap and raising the count are **one step** under the lock, so two workers can't both take the last slot.

1. **`LaneQueue(max_in_flight)`:**
   - `add(endpoint, delivery)`: a new lane is created on its first delivery. Wake waiters after adding.
   - `take()` blocks until some lane has work **and** has `in_flight < max_in_flight`. Lanes are visited round-robin in creation order, starting after the lane served last. It raises `in_flight` and returns `(endpoint, delivery)`. Call `yield_point()` between the limit check and the increment.
   - `take()` returns `None` once the queue is closed and no lane qualifies.
   - `done(endpoint)` lowers `in_flight` and wakes waiters.
   - `is_idle()` / `wait_idle()`: true / block until no delivery is waiting and none is in flight.
   - `close()` wakes every waiter.
2. **`WebhookWorkerPool(queue, sender, n)`:**
   - `start()` launches `n` workers that loop: take → `sender.post(endpoint, delivery)` → `done` (in a `finally`).
   - `stop()` waits until the queue is idle, closes it and joins the workers.

> The driver also catches a missing in-flight limit (`4` instead of `2`, and the fast endpoint finishing late).

### Names the driver uses

- `LaneQueue(max_in_flight)`: `add`, `take`, `done`, `is_idle`, `wait_idle`, `close`
- `WebhookWorkerPool(queue, sender, n)`: `start()`, `stop()`
- The driver's `RecordingSender` provides `post(endpoint, delivery)`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


class RecordingSender:
    def __init__(self, start):
        self.start, self.lock = start, threading.Lock()
        self.active, self.peak, self.finished = {}, {}, {}
    def post(self, endpoint, delivery):
        with self.lock:
            self.active[endpoint] = self.active.get(endpoint, 0) + 1
            self.peak[endpoint] = max(self.peak.get(endpoint, 0), self.active[endpoint])
        time.sleep(0.1 if endpoint == "slow" else 0.005)
        with self.lock:
            self.active[endpoint] -= 1
            self.finished[endpoint] = time.monotonic() - self.start


if __name__ == "__main__":
    q = LaneQueue(max_in_flight=2)
    for i in range(10): q.add("slow", f"S{i}")          # -> one customer's endpoint is slow and has a backlog
    for i in range(10): q.add("fast", f"F{i}")
    sender = RecordingSender(time.monotonic())
    pool = WebhookWorkerPool(q, sender, 4)
    pool.start()
    pool.stop()                                         # -> waits until every delivery is done
    print(sender.peak["slow"], sender.peak["fast"] <= 2)
                                                        # 2 True   -> never more than 2 workers on one endpoint
    print(sender.finished["fast"] < 0.25, sender.finished["slow"] >= 0.45)
                                                        # True True  -> fast finished early, slow did not starve it
    print(q.is_idle(), threading.active_count())        # True 1
```


---

## C19 · Log-based Pub-Sub with Blocking Reads — Condition + get-or-create

**Primitive:** append-only log guarded by a condition, blocking read with timeout, thread-safe registry · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#19 · Bounded Queue + Pub-Sub`, template `21_pub_sub.py`

### Problem

Each topic is an **append-only log** that is never removed on read. Each subscriber owns only **its offset**, so it reads at its own pace and can replay. A subscriber that has caught up **blocks** until a new message arrives or its timeout expires. Many threads ask the broker for a topic by name, and **racing creators must all get the same topic object**.

1. **`Topic(name)`:**
   - `publish(msg)` appends, wakes every waiting reader, and returns the offset.
   - `read(offset, timeout=None)` waits while `offset ≥ len(log)` and returns `log[offset]`, or `None` on timeout.
2. **`Subscriber(name, topic)`:**
   - `offset` starts at 0.
   - `poll(timeout=None)` reads at its offset and advances it on success.
   - `seek(offset)` sets the offset, which is how a replay works.
3. **`Broker.topic(name)`** gets or creates the topic in one step under the broker lock, with `yield_point()` between the lookup and the insert.

### Names the driver uses

- `Broker()`: `topic(name) -> Topic`
- `Topic`: `publish(msg)`, `read(offset, timeout=None)`
- `Subscriber(name, topic)`: `poll(timeout=None)`, `seek(offset)`, attribute `offset`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


if __name__ == "__main__":
    broker = Broker()
    seen, slock, barrier = [], threading.Lock(), threading.Barrier(10)
    def grab():
        barrier.wait()
        t = broker.topic("orders")
        with slock: seen.append(id(t))
    ts = [threading.Thread(target=grab) for _ in range(10)]
    for t in ts: t.start()
    for t in ts: t.join()
    print(len(set(seen)))                         # 1      -> 10 racing creators, one topic

    orders = broker.topic("orders")
    results = {}
    def consume(name):
        sub, got = Subscriber(name, orders), []
        while len(got) < 100:
            m = sub.poll(timeout=2)
            if m is None: break
            got.append(m)
        results[name] = got
    consumers = [threading.Thread(target=consume, args=(f"S{i}",)) for i in range(3)]
    for c in consumers: c.start()                 # -> they block until messages arrive
    for i in range(100):
        orders.publish(f"order-{i}")
    for c in consumers: c.join()
    expected = [f"order-{i}" for i in range(100)]
    print(all(results[n] == expected for n in results), len(results))
                                                  # True 3   -> every subscriber got every message, in order

    late = Subscriber("late", orders)
    print(late.poll(), late.offset)               # order-0 1   -> the log is kept, so a new reader can read history
    late.seek(100)
    t0 = time.monotonic()
    print(late.poll(timeout=0.05), time.monotonic() - t0 >= 0.05)
                                                  # None True   -> caught up, timed out
    got = {}
    def waiter():
        t1 = time.monotonic()
        got["msg"], got["took"] = late.poll(timeout=2), time.monotonic() - t1
    w = threading.Thread(target=waiter)
    w.start()
    time.sleep(0.05)
    orders.publish("order-100")                   # -> wakes the blocked reader
    w.join()
    print(got["msg"], got["took"] < 0.5)          # order-100 True
    late.seek(0)
    print(late.poll())                            # order-0   -> replay
```


---

## C20 · Circuit Breaker Under Concurrency — Exactly one half-open trial

**Primitive:** lock-guarded state machine, never held while calling out; a "trial in flight" flag · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#29 · Webhook Delivery` (requirement 9), template `20_circuit_breaker.py`

### Problem

Many threads call one endpoint through a breaker. After the cooldown, the breaker goes half-open, and it must let **exactly one** trial call through. Every other thread arriving while that trial is running is **rejected**, never sent to the endpoint. **Don't hold the lock while the endpoint runs.**

1. **States** `CLOSED → OPEN → HALF_OPEN → CLOSED / OPEN`. The clock is injected (`FakeClock`).
2. **Before the call** (under the lock):
   - `OPEN` with `now − opened_at < cooldown` → raise `CircuitOpenError`.
   - `OPEN` once the cooldown has passed → switch to `HALF_OPEN`.
   - `HALF_OPEN` with a trial already running → raise `CircuitOpenError`.
   - `HALF_OPEN` with no trial running → mark the trial as running; this caller is the trial. Call `yield_point()` between the check and the mark.
3. **Call `fn()` with no lock held.**
4. **After the call** (under the lock):
   - A trial clears the running flag.
   - Success → `CLOSED`, failure count reset to 0.
   - Failure → add 1 to the count. A failed trial, or reaching `threshold`, → `OPEN` with `opened_at = now`.
   - Re-raise `fn`'s exception after recording it.

### Names the driver uses

- `CircuitState`: `CLOSED`, `OPEN`, `HALF_OPEN` · `CircuitOpenError` · `FakeClock()` with `advance`
- `CircuitBreaker(threshold, cooldown, clock)`: `call(fn)`, `get_state()`

### Driver code

```python
import threading
import time


def yield_point():
    time.sleep(0)


def storm(n, cb, fn):
    barrier, counts, lock = threading.Barrier(n), {"ok": 0, "failed": 0, "rejected": 0}, threading.Lock()
    def run():
        barrier.wait()
        try:
            cb.call(fn)
            key = "ok"
        except CircuitOpenError:
            key = "rejected"
        except IOError:
            key = "failed"
        with lock: counts[key] += 1
    ts = [threading.Thread(target=run) for _ in range(n)]
    for t in ts: t.start()
    for t in ts: t.join()
    return counts


class Endpoint:
    def __init__(self): self.calls, self.healthy, self.lock = 0, False, threading.Lock()
    def __call__(self):
        with self.lock: self.calls += 1
        time.sleep(0.05)                                   # -> slow enough that other threads arrive mid-call
        if not self.healthy: raise IOError("down")
        return "ok"


if __name__ == "__main__":
    clock, ep = FakeClock(), Endpoint()
    cb = CircuitBreaker(threshold=3, cooldown=10, clock=clock)

    for _ in range(3):
        try: cb.call(ep)
        except IOError: pass
    print(cb.get_state().name, ep.calls)                   # OPEN 3
    print(storm(10, cb, ep), ep.calls)                     # {'ok': 0, 'failed': 0, 'rejected': 10} 3   -> open: nothing reaches the endpoint

    clock.advance(10)
    print(storm(10, cb, ep), ep.calls)                     # {'ok': 0, 'failed': 1, 'rejected': 9} 4
                                                           # -> half-open lets exactly ONE trial through, and it failed
    print(cb.get_state().name)                             # OPEN

    clock.advance(10)
    ep.healthy = True
    print(storm(10, cb, ep), ep.calls)                     # {'ok': 1, 'failed': 0, 'rejected': 9} 5
    print(cb.get_state().name)                             # CLOSED
    print(storm(10, cb, ep), ep.calls)                     # {'ok': 10, 'failed': 0, 'rejected': 0} 15   -> closed again, all calls pass
```

