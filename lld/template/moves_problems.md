# Moves Practice: 29 Problems

One small problem per LLD card (29 cards). Each problem covers **only that card's moves and hard part**, with no design patterns. Read the problem, code it, paste the driver under your code, and run it.

This file is also the **home of the reference material**: the 10 moves, the 10-step method, the hint-phrase table, the problem-type skeletons (all below), and each card's **full design** (requirements, idea, classes, pattern, hard part, follow-up) in a collapsible block inside its problem. Pattern practice is in [design_pattern_problems.md](design_pattern_problems.md) and thread-safety practice is in [concurrency_problems.md](concurrency_problems.md).

### How to revise a card

Cover the card, read only the title, and write on a blank page: core idea → classes → pattern → hard part → one follow-up. Then open the **Full design card** block and compare. Repeat each card on day 1, 3, 7 and 21.

## Reference 1: The 10 Moves (memorize these first)

Every design is mostly made of these moves (they are 80% of every design). Learn all 10 until you can say them from memory. Each problem below lists the moves it uses, like `M1 M3`.

| # | Move | One-line rule | Seen in |
|---|---|---|---|
| **M1** | **Lock the scarce thing where it lives** | The object that can be double-taken (spot, seat, driver, room) holds its own lock and has a `try_xxx()` that returns bool. Exactly one caller wins. | Parking, BookMyShow, Ride, Locker, Meeting, Cab, Airline, Wallet, Rate limiter |
| **M2** | **Compute, don't store** | If a fact can be derived (free, in stock, balance, size, status), compute it. Two copies of one fact will eventually disagree. | Splitwise, Vending, ATM, File system, Judge leaderboard, Notification status |
| **M3** | **A record runs its own lifecycle** | Booking / Ride / Order / Trip / Loan / Job / Delivery owns its stages and refuses illegal moves (`confirm`, `cancel`, `complete`). | BookMyShow, Ride, Food, Cab, Library, Scheduler, Webhook |
| **M4** | **Swappable rule = Strategy** | "can be replaced", "different ways to", "supports X and Y" means an interface plus one class per rule, passed into the constructor. | Almost all |
| **M5** | **Behaviour by condition = State** | "behaves differently when idle / has money / moving" means one class per state, and the context just asks the current state. | Elevator, Vending, ATM |
| **M6** | **Hold → confirm → expire** | A temporary hold gets an `expires_at`. A sweep `expire_stale(now)` frees it, **and** `confirm` checks expiry itself, so it's safe even if the sweep is late. | BookMyShow, Locker, Library reservation |
| **M7** | **Fast half queues, slow half works** | The API only records and enqueues, then returns. A worker pool takes jobs, does the slow call, retries with backoff and records the result. | Notification, Scheduler, Crawler, Judge, Webhook |
| **M8** | **Money safety** | Check first, change last, with an undo if a later step fails. An idempotency key is claimed first. Lock two accounts in a fixed (id) order. | ATM, Wallet, Food delivery, Parking exit |
| **M9** | **Thing vs. its per-instance status** | The physical item is separate from its status in one context: `Seat` vs `ShowSeat`, `Seat` vs `FlightSeat`, `Book` vs `BookCopy`. | BookMyShow, Airline, Library |
| **M10** | **Decision-maker sees copies; reservation may refuse** | The strategy picks from read-only snapshots (`ElevatorInfo`, `CabInfo`, `DriverInfo`), then `try_reserve` on the real object. If it fails, retry with the next one. | Elevator, Ride, Cab |

**Memory hook:** *"Lock it, compute it, let records live, swap rules, switch states. Hold it, queue it, keep money safe, split thing from status, decide on copies."*


---

## Reference 2: The 10-Step Method (works for any new problem)

1. **Nouns:** underline them. A noun with its own data is a class, and a single value is an attribute.
2. **Fixed sets:** these become enums (status, type, direction).
3. **One owner per fact:** who holds each piece of data?
4. **Computed facts:** don't store them (M2).
5. **Verbs:** give each verb to the class whose data changes.
6. **Cross-class action:** the order of steps lives in the Service/Manager (the entry point).
7. **Hint phrases:** match them to patterns (table below).
8. **Likely changes:** each one should be a new class, not an edit.
9. **Concurrency:** what can two threads fight over? Put the lock there (M1).
10. **Outputs:** bool for success/refusal, `X or None`, read-only info objects.

#### Hint phrase → pattern

| Phrase in the requirement | Pattern |
|---|---|
| "can be replaced", "different pricing/payment/strategy" | **Strategy** |
| "behaves differently when …", lots of status-based `if`s | **State** |
| "notify", "alert users when it changes", "subscribe" | **Observer** |
| "pass along until someone handles it", log levels, approval levels, note denominations | **Chain of Responsibility** |
| "files and folders", "tree", "part-whole" | **Composite** |
| "run this later / undo / queue an action" | **Command** |
| "only one instance", global config/log manager | **Singleton** (use sparingly) |
| "create object by type" | **Factory** |
| "async", "don't make caller wait", "retry" | **Queue + worker pool** |
| "endpoint keeps failing" | **Circuit breaker** |


---

## Reference 3: Problem Types and Their Skeletons

Each problem below belongs to one type. Learn the skeleton once and it fits every problem of that type.

| Type | Problems | Skeleton |
|---|---|---|
| Booking a Shared Resource | 01, 02, 15, 18, 24, 28 | Resource → Slot/Seat/Compartment → Booking/Ticket, with a Service as the single entry point. The hard part is always no double booking (M1). |
| State Machines | 04, 05, 12 | A Context class + a `State` interface + one class per state. A state returns a result, and the context applies it. Draw the state diagram first. |
| Matching / Dispatch | 08, 25, 14 | Request → Matcher (Strategy over snapshots) → `try_reserve` on the resource → a Record that runs its lifecycle (+ Observer). Reserve the scarce resource **before** taking money. |
| Money / Ledger | 03, 27 | — |
| Queue + Workers (async systems) | 13, 17, 19, 23, 26, 29 | `submit()` records the job and enqueues it, then returns → `BlockingQueue` → `WorkerPool` → handler (Strategy) → record the outcome, retry with backoff. Know: bounded queue, condition variables, shutdown, at-least-once + idempotency. |
| Data Structures / Infra | 06, 07, 09, 20, 21, 22 | One core data structure behind an interface. Interviewers test whether you know the data structure, so practise coding these. |
| Games | 10, 11, 16 | `Game` (turn, status, players) + `Board` (cells, move legality) + `Player`. A turn is a fixed sequence of steps. |

---

### How to use

1. Write your solution in a new `.py` file.
2. Paste the **driver code** under it and run it.
3. Compare the output with the comments: `print(x)  # True` means that line must print `True`.
4. Names under **Names the driver uses** are fixed. Everything else is up to you.
5. When the driver passes, do **Extend it**: add the new parts, then run **both** drivers. The first driver must still pass without any changes.

### Moves vs Extend it

- **The driver** checks the **moves**, meaning the code is *correct*: no double booking, no lost money, nothing left half done.
- **Extend it** checks the **extension point**, meaning the code is *easy to change*. One rule moves behind a small interface (or a function) that's passed in through the constructor, with a default that keeps the old behaviour.
- This is how the interview goes: get it correct first, then the interviewer asks "what if pricing changes?"
- Add only the one extension point they ask for. Don't build them everywhere.

### Move coverage (M4 Strategy and M5 State are left out: they're patterns)

| Move | Problems that code it |
|---|---|
| **M1** Lock the scarce thing | 01, 02, 04, 06, 08, 14, 15, 18, 24, 25, 27, 28 |
| **M2** Compute, don't store | 01, 02, 03, 04, 05, 07, 09, 10, 11, 12, 13, 16, 18, 20, 21, 24, 26 |
| **M3** A record runs its own lifecycle | 02, 08, 14, 15, 17, 18, 24, 25, 28 |
| **M6** Hold → confirm → expire | 02, 18 |
| **M7** Fast half queues, slow half works | 13, 17, 19, 23, 26, 29 |
| **M8** Money safety | 01, 12, 14, 27 |
| **M9** Thing vs. its per-instance status | 02, 15, 28 |
| **M10** Decide on copies, reservation may refuse | 04, 08, 25 |

Every card's own move list (minus M4/M5) is covered by its problem.

Time is always passed in as a number (`now`), so the outputs are always the same. Every driver was run against a reference solution, and every comment is real output.

### Index

| # | Problem | Moves | Hard part you code |
|---|---|---|---|
| 01 | Parking Lot | M1 M2 M8 | `try_occupy`, free is computed, put the ticket back if payment fails |
| 02 | BookMyShow | M1 M3 M6 M9 | all-or-nothing hold, expiry checked inside `confirm` |
| 03 | Splitwise | M2 | integer split, computed balances, simplify debts |
| 04 | Elevator | M1 M2 M10 | reserve from snapshots, retry on refusal, overload is computed |
| 05 | Vending Machine | M2 | money first, then selection; in stock is computed |
| 06 | Rate Limiter | M1 | fixed window, sliding log, token bucket |
| 07 | LRU / LFU Cache | M2 | hashmap + linked list; LFU `min_count` |
| 08 | Ride Sharing | M1 M3 M10 | `Driver.try_reserve`, ride checks *who* is acting |
| 09 | Logging Framework | M2 | the level is data, not one class per level |
| 10 | Snake and Ladder | M2 | positions live in `Game`; jumps are maps |
| 11 | Tic-Tac-Toe | M2 | the board refuses bad moves; full and winner are computed |
| 12 | ATM | M2 M8 | plan → debit → dispense → credit back |
| 13 | Notification Service | M2 M7 | retry queue by ready time; status is computed |
| 14 | Food Delivery | M1 M3 M8 | the cart stores ids, price fixed at order time, `try_assign` partner |
| 15 | Library Management | M1 M3 M9 | returns go through `Book`, so the waitlist is served |
| 16 | Chess | M2 | attacked squares vs candidate moves, immutable pieces |
| 17 | Task Scheduler | M3 M7 | version numbers in the heap, no drift |
| 18 | Amazon Locker | M1 M2 M3 M6 | unique code with retry; collect vs expire, decided once |
| 19 | Bounded Queue + Pub-Sub | M7 | offsets per group, trim to the slowest |
| 20 | In-Memory File System | M2 | size is computed, a move is one key change, no move into itself |
| 21 | KV Store with TTL + Transactions | M2 | layers, tombstones, lazy expiry |
| 22 | Unique ID Generator (Snowflake) | — | bit layout, sequence overflow, clock moving backwards |
| 23 | Web Crawler | M7 | per-site queues served round-robin; robots rules cached |
| 24 | Meeting Room Booking | M1 M2 M3 | overlap check, reserve the new slot before releasing the old one |
| 25 | Cab Booking (scheduled) | M1 M3 M10 | time window + buffer, snapshots, trip lifecycle |
| 26 | Online Code Judge | M2 M7 | submit only enqueues, a worker judges; sandbox always destroyed |
| 27 | Digital Wallet | M1 M8 | lock order, idempotency key, credit back if payout fails |
| 28 | Airline Management | M1 M3 M9 | `FlightSeat` per flight, price fixed at hold, notify after releasing the lock |
| 29 | Webhook Delivery | M7 | round-robin lanes, retries, dead letters, receiver dedupe |

---

## 01 · Parking Lot

**Moves:** M1 M2 M8 · **Difficulty:** Easy · **Time:** ~15 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The parking lot should have multiple floors, each with spots of different sizes for different vehicle types.
  2. The system should allow a vehicle to enter and be assigned an available spot that fits it.
  3. The system should issue a ticket on entry that records the vehicle, the spot and the entry time.
  4. The system should allow a vehicle to exit after paying a fee based on how long it was parked.
  5. The fee should vary by vehicle type, and the pricing rule should be easy to swap.
  6. The system should support multiple payment methods.
  7. The system should show spot availability and refuse entry when the lot is full.
  8. Multiple entry and exit gates should work at the same time without giving the same spot to two vehicles.
- **Idea:** Each spot guards its own occupancy with its own lock, so many gates work at once and one wins each spot. Pricing and payment are swappable.
- **Classes:** `Spot`, `Floor`, `Ticket`, `Receipt`, `PricingStrategy`→`HourlyPricing`, `PaymentMethod`→`Cash/Card`, `ParkingLot`
- **Pattern:** Strategy ×2 (fee, payment) · **Moves:** M1 M2 M4 M8
- **Hard part:** `Spot.try_occupy()` under lock. "Free" = `vehicle is None` (computed). On exit, pop the ticket, pay, and put it back if payment fails.
- **Follow-up:** *Small car in a big spot?* Change one condition in `try_occupy` to "spot ≥ vehicle", and search the smallest spots first.

</details>

### Problem

Spots have sizes `1` (small), `2` (medium) and `3` (large). A vehicle fits a spot of its size or bigger.

1. `Spot(spot_id, size)`
   - `try_occupy(plate, size) -> bool` takes the spot only if it's free and the vehicle fits. Do the check and the change under the spot's own lock.
   - `is_free()` is **computed** (`vehicle is None`). Don't store a separate flag.
   - `release()` frees the spot.
2. `ParkingLot(spots)`
   - `park(plate, size, now) -> ticket_id | None` tries spots **smallest first** and returns `"T1"`, `"T2"`, … or `None` if nothing fits.
   - `spot_of(ticket_id)` returns the spot id, or `None` if the ticket isn't active.
   - `free_count()` is computed.
3. `exit(ticket_id, now, pay) -> fee | None`
   - `now` is in minutes. The fee is `10` per started hour, minimum 1 hour.
   - **Pop the ticket first**, then call `pay(fee)`.
   - If payment fails, **put the ticket back** and return `None`. The car stays parked.
   - On success, free the spot and return the fee. An unknown ticket returns `None`.

### Names the driver uses

`Spot(spot_id, size)`: `try_occupy`, `is_free` · `ParkingLot(spots)`: `park`, `spot_of`, `free_count`, `exit`

### Driver code

```python
if __name__ == "__main__":
    s = Spot("X", 2)
    print(s.try_occupy("CAR1", 2), s.try_occupy("CAR2", 2), s.is_free())  # True False False
    print(Spot("Y", 1).try_occupy("TRUCK", 3))      # False

    lot = ParkingLot([Spot("S1", 1), Spot("L1", 3), Spot("M1", 2)])
    car = lot.park("CAR1", 2, now=0)
    bike = lot.park("BIKE", 1, now=0)
    print(car, lot.spot_of(car), bike, lot.spot_of(bike))  # T1 M1 T2 S1
    print(lot.free_count())                         # 1
    truck = lot.park("TRUCK", 3, now=0)
    print(lot.park("CAR2", 2, now=0), lot.free_count())  # None 0
    print(lot.exit(car, now=125, pay=lambda fee: False))  # None
    print(lot.spot_of(car), lot.free_count())       # M1 0
    print(lot.exit(car, now=125, pay=lambda fee: True))  # 30
    print(lot.exit(car, now=125, pay=lambda fee: True), lot.free_count())  # None 1
    print(lot.exit(bike, now=10, pay=lambda fee: True))  # 10
```

### Extend it: Pricing differs by vehicle size, and may change again.

1. `ParkingLot(spots, pricing=None)`. The default is `HourlyPricing({1: 10, 2: 10, 3: 10})`, so the old driver still passes.
2. `HourlyPricing(rates).fee(size, minutes)` charges `rates[size]` per started hour, with a minimum of 1 hour.
3. `FlatPricing(amount).fee(size, minutes)` always returns `amount`.
4. The ticket must remember the vehicle size. `exit` calls `pricing.fee(...)`, and the lock / pop / put-back steps don't change.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    lot = ParkingLot([Spot("S1", 1), Spot("L1", 3)], pricing=HourlyPricing({1: 5, 2: 10, 3: 20}))
    bike, truck = lot.park("BIKE", 1, now=0), lot.park("TRUCK", 3, now=0)
    print(lot.exit(bike, now=61, pay=lambda fee: True), lot.exit(truck, now=125, pay=lambda fee: True))  # 10 60
    flat = ParkingLot([Spot("M1", 2)], pricing=FlatPricing(50))
    car = flat.park("CAR", 2, now=0)
    print(flat.exit(car, now=500, pay=lambda fee: True))  # 50
    old = ParkingLot([Spot("M1", 2)])
    print(old.exit(old.park("CAR", 2, now=0), now=125, pay=lambda fee: True))  # 30
```

---

## 02 · BookMyShow

**Moves:** M1 M3 M6 M9 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should allow users to search movies by city and view the theaters and shows that run them.
  2. Each theater should have multiple screens, and each screen should have a seat layout and show movies at different times.
  3. Seats should have different types with different prices.
  4. The system should allow users to pick seats for a show and book them.
  5. Selected seats should be held temporarily while payment is in progress and released if payment fails or times out.
  6. Two users must never be able to book the same seat for the same show.
  7. The system should support multiple payment methods.
  8. The system should allow users to cancel a booking and free the seats.
- **Idea:** A seat is physical, but its status belongs to one show, so there is one `ShowSeat` per seat per show, each with its own lock. The Booking does its own confirm, cancel and expire.
- **Classes:** `Seat`, `Screen`, `ShowSeat`, `Show`, `Theater`, `MovieCatalog`, `PaymentMethod`, `Booking`, `BookingService`
- **Pattern:** Strategy (payment) · **Moves:** M1 M3 M6 M9
- **Hard part:** Holding several seats is all-or-nothing. If one fails, undo only this call's holds. A hold has `expires_at`, plus a sweep **and** a check inside `confirm`.
- **Follow-up:** *Two users tap the same seat?* Both call `try_hold`. The lock lets one win. The loser releases its other holds and gets None.

</details>

### Problem

One list of seat ids is shared by two shows, but each show keeps its **own status per seat** (M9).

1. `Show(show_id, seat_ids, hold_ttl=300)` creates one `ShowSeat` per seat id.
2. `seat_status(seat_id, now)` is **computed**:
   - `"BOOKED"` if the seat is booked
   - `"HELD"` if it's held and `now < expires_at`
   - `"AVAILABLE"` otherwise, so an expired hold reads as available
3. `hold(seat_ids, now) -> booking_id | None`
   - Return `None` straight away for an empty list or an unknown seat.
   - Booking ids are `"B1"`, `"B2"`, …, one per call that gets this far.
   - Hold each seat with `expires_at = now + hold_ttl` (lock per seat). If one is taken, **release only the seats this call held** and return `None`.
4. `confirm(booking_id, now) -> bool` works only if the booking is `PENDING` **and `now < expires_at`**. The check is inside `confirm`, so it's safe even if the sweep is late. The seats become `BOOKED` and the booking `CONFIRMED`.
5. `expire_stale(now) -> int` moves `PENDING` bookings with `now >= expires_at` to `EXPIRED` and releases their seats.
   - Release frees a seat **only if this booking still holds it**. Someone else may have taken the expired seat already.
   - It returns how many bookings expired.
6. `booking_status(booking_id)` returns `PENDING`, `CONFIRMED` or `EXPIRED`.

### Names the driver uses

`Show(show_id, seat_ids, hold_ttl=300)`: `hold`, `confirm`, `expire_stale`, `seat_status`, `booking_status`

### Driver code

```python
if __name__ == "__main__":
    seats = ["A1", "A2", "A3"]
    fri, sat = Show("FRI", seats), Show("SAT", seats)
    b1 = fri.hold(["A1", "A2"], now=0)
    print(b1, fri.seat_status("A1", 0), sat.seat_status("A1", 0))  # B1 HELD AVAILABLE
    print(fri.hold(["A3", "A2"], now=10))           # None
    print(fri.seat_status("A3", 10))                # AVAILABLE
    print(fri.hold(["A9"], now=10), fri.hold([], now=10))  # None None
    print(fri.confirm(b1, now=299), fri.booking_status(b1), fri.seat_status("A1", 10**9))  # True CONFIRMED BOOKED
    print(fri.confirm(b1, now=299))                 # False
    b2 = fri.hold(["A3"], now=0)
    print(fri.confirm(b2, now=300), fri.booking_status(b2))  # False PENDING
    print(fri.seat_status("A3", 300))               # AVAILABLE
    b3 = fri.hold(["A3"], now=301)
    print(b3 is not None)                           # True
    print(fri.expire_stale(301), fri.booking_status(b2))  # 1 EXPIRED
    print(fri.seat_status("A3", 301))               # HELD
    print(fri.expire_stale(301), fri.booking_status(b1))  # 0 CONFIRMED
```

### Extend it: Different seats cost different amounts.

1. `Show(..., pricing=None)`. The default is `FlatPrice(0)`.
2. `FlatPrice(p).price(seat_id)` and `RowPrice({"A": 500, ...}).price(seat_id)`. The row is the first character of the seat id.
3. The booking's amount is the sum of its seat prices, **fixed at hold time**. `booking_amount(booking_id)` returns it.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    s = Show("S", ["A1", "A2", "B1"], pricing=RowPrice({"A": 500, "B": 200}))
    b = s.hold(["A1", "B1"], now=0)
    print(s.booking_amount(b))                      # 700
    f = Show("F", ["X1", "X2"], pricing=FlatPrice(150))
    print(f.booking_amount(f.hold(["X1", "X2"], now=0)))  # 300
    print(Show("Z", ["A1"]).pricing.price("A1"))    # 0
```

---

## 03 · Splitwise

**Moves:** M2 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should allow users to register and create groups with members.
  2. The system should allow a user to add an expense that one person paid and several people share.
  3. An expense should be splittable equally, by exact amounts or by percentages.
  4. It should be easy to add new ways of splitting an expense.
  5. The system should track how much each user owes or is owed by every other user.
  6. Users should be able to view their balances with each person and within each group.
  7. Users should be able to settle their dues.
  8. The system should reduce the number of payments needed to clear all debts in a group.
- **Idea:** Store only what happened (expenses, settlements). Every balance, "who owes whom" and "fewest payments" is **computed** from history. The split type is a swappable object.
- **Classes:** `Expense`, `Transfer`, `SplitStrategy`→`Equal/Exact/Percent`, `DebtSimplifier`, `Group`, `SplitwiseService`
- **Pattern:** Strategy (split) · **Moves:** M2 M4
- **Hard part:** Money as `int` (paise/cents). Leftover units go to the first people. `Expense` checks that the shares add up. Simplify debts by greedily matching the biggest debtor with the biggest creditor.
- **Follow-up:** *Slow with many expenses?* Keep a running balance updated by the same methods (a deliberate cached copy).

</details>

### Problem

Money is an `int` (paise).

1. `split_equal(amount, users) -> dict` splits evenly. Leftover units go **one each to the first users**.
2. `Expense(paid_by, amount, shares)` raises `ValueError` unless the shares add up to `amount`.
3. `Splitwise.add_expense(paid_by, amount, shares)` stores the expense.
4. `balances()` is **computed from the expense list** every time (paid − owed), as a dict sorted by user. Don't keep a running total.
5. `simplify()` returns a list of `(debtor, creditor, amount)`.
   - Repeat: take the biggest creditor and the biggest debtor (ties go to the alphabetically first name) and settle `min(credit, debt)` between them.
   - Stop when everyone is at 0.

### Names the driver uses

`split_equal`, `Expense`, `Splitwise`: `add_expense`, `balances`, `simplify`

### Driver code

```python
def err(f):
    try:
        f()
        return "ok"
    except ValueError:
        return "ValueError"

if __name__ == "__main__":
    print(split_equal(1000, ["a", "b", "c"]))       # {'a': 334, 'b': 333, 'c': 333}
    print(split_equal(10, ["a", "b", "c", "d"]))    # {'a': 3, 'b': 3, 'c': 2, 'd': 2}
    print(err(lambda: Expense("a", 100, {"a": 50, "b": 40})))  # ValueError
    print(err(lambda: Expense("a", 100, {"a": 50, "b": 50})))  # ok

    sw = Splitwise()
    sw.add_expense("a", 900, split_equal(900, ["a", "b", "c"]))
    sw.add_expense("b", 300, {"b": 100, "c": 200})
    print(sw.balances())                            # {'a': 600, 'b': -100, 'c': -500}
    print(sum(sw.balances().values()))              # 0
    print(sw.simplify())                            # [('c', 'a', 500), ('b', 'a', 100)]
    sw.add_expense("c", 700, {"a": 700})
    print(sw.balances())                            # {'a': -100, 'b': -100, 'c': 200}
    print(sw.simplify())                            # [('a', 'c', 100), ('b', 'c', 100)]
```

### Extend it: Support equal, exact and percentage splits.

1. Each split type has `shares(amount, users) -> dict`:
   - `EqualSplit()`
   - `ExactSplit(amounts)`
   - `PercentSplit(pcts)`: raises `ValueError` unless the percentages add up to 100. Each share is rounded down, and the leftover goes one unit each to the first users.
2. `Splitwise.add_split(paid_by, amount, users, split)` turns it into a normal `add_expense`. `Expense` still checks the sum.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    print(EqualSplit().shares(100, ["a", "b", "c"]))  # {'a': 34, 'b': 33, 'c': 33}
    print(PercentSplit({"a": 50, "b": 30, "c": 20}).shares(999, ["a", "b", "c"]))  # {'a': 500, 'b': 300, 'c': 199}
    try:
        PercentSplit({"a": 50, "b": 40})
    except ValueError:
        print("ValueError")                         # ValueError
    sw = Splitwise()
    sw.add_split("a", 1000, ["a", "b"], EqualSplit())
    sw.add_split("b", 300, ["a", "c"], ExactSplit({"a": 100, "c": 200}))
    sw.add_split("c", 200, ["a", "b", "c"], PercentSplit({"a": 50, "b": 25, "c": 25}))
    print(sw.balances())                            # {'a': 300, 'b': -250, 'c': -50}
    try:
        sw.add_split("a", 100, ["a", "b"], ExactSplit({"a": 10, "b": 10}))
    except ValueError:
        print("ValueError")                         # ValueError
```

---

## 04 · Elevator

**Moves:** M1 M2 M10 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The building should have multiple elevators serving multiple floors.
  2. Users should be able to call an elevator from a floor by choosing a direction, and choose a destination floor once inside.
  3. Each elevator should move up or down, stop at requested floors along its direction, and then reverse.
  4. The system should assign each request to an elevator using a scheduling rule that can be replaced.
  5. An elevator should behave differently depending on whether it is idle, moving up, moving down or out of service.
  6. Doors should open and close at every stop.
  7. An elevator should not move when it is over its weight limit.
  8. Multiple requests should be handled at the same time without conflicts.
- **Idea:** `Car` owns the physical facts (floor, door, load, stops). A stateless `ElevatorState` only decides where to head next. The controller picks an elevator with a swappable strategy, and there is one lock per elevator.
- **Classes:** `ElevatorInfo`, `Car`, `ElevatorState`→`Idle/MovingUp/MovingDown/OutOfService`, `Elevator`, `SchedulingStrategy`→`NearestElevatorStrategy`, `ElevatorController`
- **Pattern:** State + Strategy · **Moves:** M1 M2 M5 M10
- **Hard part:** One tick = one action (door, or move one floor). Don't move if `load > capacity` (computed). The scheduler sees `ElevatorInfo` copies, and if the chosen elevator refuses, retry.
- **States:** Idle → MovingUp/MovingDown (stop on the way, reverse when no stops are ahead) → Idle; any → OutOfService.
- **Follow-up:** *Smarter scheduling?* A new `SchedulingStrategy` (zoned or least-busy). Nothing else changes.

</details>

### Problem

1. `Elevator(eid, floor=0, capacity=8)` has `load` and a `target` floor (`None` means idle).
   - `is_overloaded()` is computed: `load > capacity`.
   - `info()` returns a **frozen** `ElevatorInfo(eid, floor, idle)` copy.
   - `try_reserve(floor) -> bool` under a lock. It refuses if the elevator is already busy or overloaded, and otherwise sets the target.
   - `step()` is one tick: move one floor toward the target. On arrival the target becomes `None`. An overloaded elevator doesn't move.
2. `Dispatcher(elevators)`
   - `request(floor, infos=None) -> eid | None` uses the given snapshots, or takes fresh ones.
   - It sorts the **idle** ones by `(distance, eid)` and calls `try_reserve` on the real elevator.
   - **If it refuses, try the next one.** If none accepts, return `None`.

### Names the driver uses

`ElevatorInfo`, `Elevator`: `info`, `try_reserve`, `step`, `is_overloaded`, `floor`, `load` · `Dispatcher`: `request`

### Driver code

```python
if __name__ == "__main__":
    e1, e2 = Elevator("E1", floor=0), Elevator("E2", floor=10)
    d = Dispatcher([e1, e2])
    print(d.request(3), d.request(8), d.request(5))  # E1 E2 None
    e1.step(); e1.step()
    print(e1.floor, e1.info().idle)                 # 2 False
    e1.step()
    print(e1.floor, e1.info())                      # 3 ElevatorInfo(eid='E1', floor=3, idle=True)
    e2.step(); e2.step()
    print(e2.floor, e2.info().idle)                 # 8 True

    stale = [e1.info(), e2.info()]
    print(e1.try_reserve(0))                        # True
    print(d.request(4, infos=stale))                # E2
    print(e1.info().idle, e2.info().idle)           # False False

    e3 = Elevator("E3", floor=0, capacity=2)
    e3.load = 3
    print(e3.is_overloaded(), e3.try_reserve(5))    # True False
    e3.load = 2
    print(e3.is_overloaded(), e3.try_reserve(5))    # False True
    e3.load = 5
    e3.step()
    print(e3.floor)                                 # 0
    info = e3.info()
    try:
        info.floor = 99
    except Exception as ex:
        print(type(ex).__name__)                    # FrozenInstanceError
```

### Extend it: Each elevator serves a zone. Prefer the zone owner.

1. `Dispatcher(elevators, strategy=None)`. `strategy.rank(idle_infos, floor)` returns the snapshots in the order to try them.
2. `NearestFirst()` is the default: by `(distance, eid)`.
3. `ZonedStrategy({eid: (lo, hi)})` puts elevators whose zone contains the floor first, then sorts by distance, then by eid.
4. `request` keeps the M10 loop (try → refused → next). Only the ranking moves out.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    def build(strategy=None):
        e1, e2 = Elevator("E1", floor=7), Elevator("E2", floor=0)
        return e1, e2, Dispatcher([e1, e2], strategy)
    e1, e2, near = build()
    print(near.request(4))                          # E1
    zones = {"E1": (6, 10), "E2": (0, 5)}
    e1, e2, zoned = build(ZonedStrategy(zones))
    print(zoned.request(4))                         # E2
    print(zoned.request(3))                         # E1
    print(zoned.request(9))                         # None
```

---

## 05 · Vending Machine

**Moves:** M2 · **Difficulty:** Easy · **Time:** ~15 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The machine should hold multiple products in slots, each with a price and a quantity.
  2. The machine should allow a user to select a product and insert money.
  3. The machine should behave differently depending on whether it is idle, has money inserted, is dispensing or is out of service.
  4. The machine should dispense the product once enough money is inserted and return the change.
  5. The machine should allow the user to cancel and get a refund.
  6. The machine should refuse a selection when the product is out of stock.
  7. An operator should be able to restock products and collect the money.
- **Idea:** Each fact has one owner. Behaviour changes by swapping a small state object, not by `if`s.
- **Classes:** `Product`, `Slot`, `Inventory`, `CashBox`, `Session`, `Response`, `MachineState`→`Idle/HasMoney/Dispensing/OutOfService`, `VendingMachine`
- **Pattern:** State · **Moves:** M2 M5
- **Hard part:** Money first, then selection. `Session` holds the customer's money and choice. "In stock" is computed. States return a `Response`.
- **States:** Idle →(insert)→ HasMoney →(select)→ Dispensing → Idle; cancel refunds; OutOfService refuses everything.
- **Follow-up:** *Why is Dispensing a state?* It models dispensing time and blocks every action while a sale runs.

</details>

### Problem

1. `VendingMachine(products)` takes `products` as `{code: (price, count)}`. `balance` is the money inserted so far.
2. `insert(amount)` adds money and returns the balance.
3. `in_stock(code)` is computed: the code exists and `count > 0`.
4. `select(code)` checks in this order (money first, then selection):
   1. no money → `"INSERT_MONEY"`
   2. unknown code → `"UNKNOWN"`
   3. not in stock → `"OUT_OF_STOCK"`
   4. not enough money → `"NEED <missing>"`
   5. otherwise reduce the count, reset the balance to 0, and return `"DISPENSE <code> CHANGE <change>"`
5. `cancel()` returns the balance and resets it to 0.

### Names the driver uses

`VendingMachine(products)`: `insert`, `select`, `cancel`, `in_stock`, `balance`

### Driver code

```python
if __name__ == "__main__":
    vm = VendingMachine({"A1": (25, 1), "B1": (40, 0)})
    print(vm.select("A1"))                          # INSERT_MONEY
    print(vm.insert(10), vm.select("A1"))           # 10 NEED 15
    print(vm.select("Z9"), vm.select("B1"))         # UNKNOWN OUT_OF_STOCK
    print(vm.insert(20), vm.select("A1"))           # 30 DISPENSE A1 CHANGE 5
    print(vm.balance, vm.in_stock("A1"))            # 0 False
    print(vm.insert(50), vm.select("A1"))           # 50 OUT_OF_STOCK
    print(vm.cancel(), vm.balance, vm.cancel())     # 50 0 0
```

### Extend it: Happy-hour prices.

1. `VendingMachine(products, pricing=None)`. `pricing(code, base_price) -> price`. The default returns `base_price`.
2. `HappyHour(pct)` is a callable class: `base × (100 − pct) // 100`.
3. `select` uses the price from the pricing rule. Any plain function works too.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    vm = VendingMachine({"A1": (40, 2)}, pricing=HappyHour(25))
    print(vm.insert(20), vm.select("A1"))           # 20 NEED 10
    print(vm.insert(10), vm.select("A1"))           # 30 DISPENSE A1 CHANGE 0
    special = VendingMachine({"A1": (40, 2), "B1": (40, 2)}, pricing=lambda code, base: 1 if code == "B1" else base)
    print(special.insert(5), special.select("B1"))  # 5 DISPENSE B1 CHANGE 4
```

---

## 06 · Rate Limiter

**Moves:** M1 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should limit how many requests a client can make within a time window.
  2. The system should decide for each request whether to allow or reject it based on the client's recent usage.
  3. The system should support multiple limiting algorithms that can be swapped without changing the callers.
  4. Different clients or endpoints should be able to have different limits.
  5. The limits should be configurable.
  6. The limiter should stay correct when many requests arrive at the same time.
- **Idea:** Look up the right rule, plus a tiny counter (tracker) per client+endpoint that checks and updates itself under its own lock. The algorithm is swappable.
- **Classes:** `RuleRegistry`, `LimitAlgorithm`→`FixedWindow/SlidingWindowLog/TokenBucket`, `UsageTracker`→(one per algorithm), `RateLimiter`
- **Pattern:** Strategy (the algorithm creates a tracker, a factory-ish step) · **Moves:** M1 M4
- **Hard part:** Know all 3: fixed window (burst at the boundary), sliding log (exact, uses memory), token bucket (refill = elapsed × rate, capped). Rules are immutable and replaced whole.
- **Follow-up:** *Idle trackers leak memory?* Record last-used and sweep.

</details>

### Problem

Three algorithms, each with `allow(key, now) -> bool`. Every key has its own state, and each `allow` runs under a lock.

1. `FixedWindow(limit, window)`: the window is `now // window`, and you allow up to `limit` calls per window. This lets through a burst at the boundary.
2. `SlidingLog(limit, window)`: keep each key's timestamps. Drop those `<= now - window`, then allow if fewer than `limit` remain (and record `now`).
3. `TokenBucket(capacity, refill_per_sec)`: a new key starts full. Refill `elapsed × rate`, **capped at capacity**. Allow if at least 1 token is there, and take it.

### Names the driver uses

`FixedWindow`, `SlidingLog`, `TokenBucket`: `allow(key, now)`

### Driver code

```python
if __name__ == "__main__":
    fw = FixedWindow(limit=2, window=10)
    print([fw.allow("u1", t) for t in [9, 9, 9, 10, 10, 10]])  # [True, True, False, True, True, False]
    print(fw.allow("u2", 9))                        # True

    sl = SlidingLog(limit=2, window=10)
    print([sl.allow("u1", t) for t in [9, 9, 10, 18, 19, 19, 20]])  # [True, True, False, False, True, True, False]

    tb = TokenBucket(capacity=2, refill_per_sec=1)
    print([tb.allow("u1", t) for t in [0, 0, 0, 0.5, 1, 1, 10, 10, 10]])  # [True, True, False, False, True, False, True, True, False]
    print(tb.allow("u2", 0))                        # True
```

### Extend it: Free users get a fixed window; pro users get a token bucket.

1. `ClientLimiter(make_limiter)`: `make_limiter(client)` returns an algorithm object (any of the three).
2. `allow(client, now)` creates the client's limiter **once**, on its first call, under a lock, and keeps it in `limiters`. Then it calls `limiter.allow(client, now)`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    plans = {"free": lambda: FixedWindow(limit=1, window=10), "pro": lambda: TokenBucket(capacity=3, refill_per_sec=1)}
    plan_of = {"u1": "free", "u2": "pro"}
    cl = ClientLimiter(lambda client: plans[plan_of[client]]())
    print([cl.allow("u1", 0) for _ in range(2)])    # [True, False]
    print([cl.allow("u2", 0) for _ in range(4)])    # [True, True, True, False]
    print(cl.allow("u1", 10), cl.allow("u2", 1))    # True True
    print(sorted(cl.limiters), type(cl.limiters["u1"]).__name__, type(cl.limiters["u2"]).__name__)  # ['u1', 'u2'] FixedWindow TokenBucket
```

---

## 07 · LRU / LFU Cache

**Moves:** M2 · **Difficulty:** Medium · **Time:** ~30 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The cache should store key-value pairs up to a fixed capacity.
  2. The cache should return the value for a key, or indicate that the key is missing.
  3. The cache should allow inserting or updating a key, and evict an entry when it is full.
  4. The least recently used cache should evict the entry that was used longest ago.
  5. The least frequently used cache should evict the entry with the fewest uses, breaking ties by oldest use.
  6. Both reads and writes should run in constant time.
  7. The eviction rule should be replaceable.
  8. The cache should stay correct when accessed by multiple threads.
- **Idea:** The cache owns values + capacity. The eviction policy owns only key order/counts. The cache tells the policy about every insert/use and asks it who to evict. Everything is O(1).
- **Classes:** `UsageList` (doubly linked list), `EvictionPolicy`→`LruPolicy/LfuPolicy`, `Cache`
- **Pattern:** Strategy (eviction) · **Moves:** M4
- **Hard part:** LRU = hashmap + doubly linked list (move to front on use, evict the tail). LFU = map count→list + `min_count`. A read needs the lock (it mutates the order).
- **Follow-up:** *TTL?* Store expiry with the value; treat it as missing when expired on read.

</details>

### Problem

1. `LRUCache(capacity)`: `get(key)` returns `-1` if missing, and `put(key, val)`. Build it from a **hashmap + your own doubly linked list** (no `OrderedDict`).
   - On use, move the entry to the front. When full, evict the tail.
   - `keys()` lists keys from most to least recently used.
   - A `get` changes the order, so it needs the lock too.
2. `LFUCache(capacity)`: evict the key with the lowest use count. On a tie, evict the least recently used one.
   - Use `key → count` plus `count → keys in insertion order` plus `min_count`.
   - A new key resets `min_count` to 1.

### Names the driver uses

`LRUCache`: `get`, `put`, `keys` · `LFUCache`: `get`, `put`

### Driver code

```python
if __name__ == "__main__":
    lru = LRUCache(2)
    lru.put(1, "a"); lru.put(2, "b")
    print(lru.get(1), lru.keys())                   # a [1, 2]
    lru.put(3, "c")
    print(lru.get(2), lru.keys())                   # -1 [3, 1]
    lru.put(1, "A")
    print(lru.get(1), lru.get(3), lru.keys())       # A c [3, 1]
    lru.put(4, "d")
    print(lru.get(1), lru.keys())                   # -1 [4, 3]

    lfu = LFUCache(2)
    lfu.put(1, "a"); lfu.put(2, "b")
    print(lfu.get(1))                               # a
    lfu.put(3, "c")
    print(lfu.get(2), lfu.get(3), lfu.get(1))       # -1 c a
    lfu.put(4, "d")
    print(lfu.get(3), lfu.get(1), lfu.get(4))       # -1 a d
    lfu.put(5, "e")
    print(lfu.get(4), lfu.get(1), lfu.get(5))       # -1 a e
```

### Extend it: Make the eviction rule swappable.

1. `Cache(capacity, policy)`. The policy has `add(key)`, `touch(key)` and `evict() -> key`.
2. `LRUPolicy` (an `OrderedDict` is fine here) and `FIFOPolicy` (a `touch` does nothing).
3. `Cache` stores the values and asks the policy *which* key to evict.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    def run(policy):
        c = Cache(2, policy)
        c.put(1, "a"); c.put(2, "b")
        c.get(1)
        c.put(3, "c")
        return c.get(1), c.get(2), c.get(3)
    print(run(LRUPolicy()))                         # ('a', -1, 'c')
    print(run(FIFOPolicy()))                        # (-1, 'b', 'c')
```

---

## 08 · Ride Sharing

**Moves:** M1 M3 M10 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should allow riders and drivers to register, and drivers to register their vehicles and set their availability.
  2. A rider should be able to request a ride by giving a pickup and a drop-off location.
  3. The system should show a fare estimate before the ride is confirmed.
  4. The system should match the rider with a nearby available driver using a matching rule that can be replaced.
  5. A driver should be able to accept or reject a ride request.
  6. A ride should move through the stages of requested, accepted, in progress, completed and cancelled.
  7. The fare should be calculated from distance, time and vehicle type, and the pricing rule should be easy to swap.
  8. The system should support multiple payment methods after the ride.
  9. Riders and drivers should be able to rate each other.
  10. A driver must never be assigned to two rides at the same time.
- **Idea:** The driver is the scarce resource and guards its availability with a lock. Reserve the driver **before** building the ride. The Ride runs its own lifecycle. Matching, pricing and payment are all swappable.
- **Classes:** `Rider`, `Driver`, `DriverInfo`, `Vehicle`, `ReputationScore`, `RateCard`, `PricingStrategy`, `MatchingStrategy`→`NearestDriverStrategy`, `PaymentMethod`, `Ride`, `RideService`
- **Pattern:** Strategy ×3 · **Moves:** M1 M3 M4 M10
- **Hard part:** `Driver.try_reserve()`. The estimate is computed on demand, and the final fare is stored once at completion. `Ride` checks *who* is acting.
- **Follow-up:** *Surge pricing?* Add a demand parameter to `PricingStrategy` plus a new surge class.

</details>

### Problem

Locations are integers on a line.

1. `Driver(driver_id, location)`
   - `info()` returns a frozen `DriverInfo(driver_id, location, available)`.
   - `try_reserve(ride_id) -> bool` under a lock. `release()` makes the driver available again.
2. `estimate(distance)` is `50 + 10 × distance`, computed on demand.
3. `Ride(ride_id, rider, driver_id)` starts as `ACCEPTED`. It checks **who** is acting:
   - `start(by)`: only the driver, from `ACCEPTED`, moves it to `STARTED`
   - `complete(by, distance)`: only the driver, from `STARTED`, moves it to `COMPLETED`. The fare is **stored once** at this point.
   - `cancel(by)`: only the rider, from `ACCEPTED`, moves it to `CANCELLED`
4. `RideService(drivers)`
   - `request(rider, location, infos=None) -> Ride | None` gets ride id `"R1"`, `"R2"`, … first (one per call). Then it tries available drivers sorted by `(distance, driver_id)`, and moves to the next one on refusal.
   - `start`, `complete` and `cancel` take `(ride_id, by, …)`. A successful complete or cancel releases the driver.

### Names the driver uses

`Driver`, `DriverInfo`, `estimate`, `Ride`: `status`, `fare`, `driver_id`, `ride_id` · `RideService`: `request`, `start`, `complete`, `cancel`

### Driver code

```python
if __name__ == "__main__":
    d1, d2, d3 = Driver("D1", 0), Driver("D2", 5), Driver("D3", 9)
    svc = RideService([d1, d2, d3])
    r1 = svc.request("alice", location=4)
    print(r1.ride_id, r1.driver_id, r1.status)      # R1 D2 ACCEPTED
    print(estimate(10))                             # 150

    stale = [d1.info(), d2.info(), d3.info()]
    print(stale[1].available)                       # False
    print(d3.try_reserve("someone-else"))           # True
    r2 = svc.request("bob", location=8, infos=stale)
    print(r2.driver_id)                             # D1
    print(svc.request("carol", location=1))         # None

    print(svc.start(r1.ride_id, by="alice"), svc.start(r1.ride_id, by="D2"))  # False True
    print(svc.cancel(r1.ride_id, by="alice"))       # False
    print(svc.complete(r1.ride_id, by="alice", distance=7), svc.complete(r1.ride_id, by="D2", distance=7))  # False True
    print(r1.status, r1.fare)                       # COMPLETED 120
    print(svc.complete(r1.ride_id, by="D2", distance=99), r1.fare)  # False 120
    print(d2.info().available)                      # True

    print(svc.cancel(r2.ride_id, by="D1"), svc.cancel(r2.ride_id, by="bob"), r2.status)  # False True CANCELLED
    print(d1.info().available)                      # True
    r3 = svc.request("carol", location=1)
    print(r3.driver_id)                             # D1
```

### Extend it: Surge pricing.

1. `RideService(drivers, pricing=None)`, where `pricing.fare(distance)`.
   - `StandardPricing()` is the default (= `estimate`).
   - `SurgePricing(multiplier)` returns `int(estimate × multiplier)`.
2. The ride keeps the pricing that was active **when it was requested**. Changing `svc.pricing` later doesn't affect it.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    svc = RideService([Driver("D1", 0), Driver("D2", 9)], pricing=SurgePricing(1.5))
    r = svc.request("ann", location=0)
    svc.start(r.ride_id, by="D1")
    svc.pricing = StandardPricing()
    r2 = svc.request("bob", location=9)
    svc.start(r2.ride_id, by="D2")
    print(svc.complete(r.ride_id, "D1", 10), r.fare)  # True 225
    print(svc.complete(r2.ride_id, "D2", 10), r2.fare)  # True 150
```

---

## 09 · Logging Framework

**Moves:** M2 · **Difficulty:** Easy · **Time:** ~10 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should log messages at different severity levels.
  2. A logger should have a minimum level and ignore messages below it.
  3. The system should write logs to multiple destinations such as console, file and database at the same time.
  4. It should be easy to add new destinations.
  5. Each log entry should include a timestamp, a level and a message, and the format should be replaceable.
  6. A message should pass along a chain of handlers where each handler deals with its own level.
  7. The same logger configuration should be accessible from everywhere in the application.
  8. The logger should be safe to use from multiple threads.
- **Idea:** A log call becomes a `LogEntry`. The logger drops entries below its min level. The rest go down a chain of handlers, and the handler for that level writes to its destinations. Each destination owns its formatter **and its lock**.
- **Classes:** `LogEntry`, `LogLevel`, `LogFormatter`→`Plain/Json`, `LogDestination`→`Console/File/Database`, `LogHandler`, `Clock`, `Logger`, `LogManager` (singleton)
- **Pattern:** Chain of Responsibility, Strategy (formatter/destination), Singleton · **Moves:** M4
- **Hard part:** One `LogHandler` class with the level as data (not one class per level). Only `LogManager` is a singleton.
- **Follow-up:** *Don't slow the app?* An async destination wrapper (queue + worker); nothing else changes.

</details>

### Problem

Levels are data: `DEBUG=10, INFO=20, WARN=30, ERROR=40`. Write **one** `Destination` class with a `min_level`, not one class per level.

1. `Logger.add_destination(name, min_level)`.
2. `log(level, msg) -> list` writes `"[LEVEL] msg"` to every destination whose `min_level <= level`, and returns their names in the order they were added. An unknown level raises `ValueError(level)`.
3. `lines(name)` returns what that destination has written.
4. `LogManager.get()` always returns the **same** `Logger`.

### Names the driver uses

`Logger`: `add_destination`, `log`, `lines` · `LogManager.get`

### Driver code

```python
if __name__ == "__main__":
    log = Logger()
    log.add_destination("console", "DEBUG")
    log.add_destination("file", "INFO")
    log.add_destination("pager", "ERROR")
    print(log.log("DEBUG", "cache miss"))           # ['console']
    print(log.log("WARN", "slow query"))            # ['console', 'file']
    print(log.log("ERROR", "db down"))              # ['console', 'file', 'pager']
    print(log.lines("file"))                        # ['[WARN] slow query', '[ERROR] db down']
    print(log.lines("pager"))                       # ['[ERROR] db down']
    try:
        log.log("FATAL", "x")
    except ValueError as e:
        print("ValueError", e)                      # ValueError FATAL
    print(LogManager.get() is LogManager.get())     # True
    LogManager.get().add_destination("console", "WARN")
    print(LogManager.get().log("INFO", "hi"), LogManager.get().log("WARN", "hey"))  # [] ['console']
```

### Extend it: Some destinations want JSON.

1. `add_destination(name, min_level, formatter=None)`, where `formatter.format(level, msg) -> str`.
2. `PlainFormatter()` is the default (`"[LEVEL] msg"`). `JsonFormatter()` returns `json.dumps({"level": level, "msg": msg})`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    log = Logger()
    log.add_destination("console", "DEBUG")
    log.add_destination("elk", "WARN", formatter=JsonFormatter())
    print(log.log("WARN", "disk 90%"))              # ['console', 'elk']
    print(log.lines("console"))                     # ['[WARN] disk 90%']
    print(log.lines("elk"))                         # ['{"level": "WARN", "msg": "disk 90%"}']
```

---

## 10 · Snake and Ladder

**Moves:** M2 · **Difficulty:** Easy · **Time:** ~15 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The game should have a numbered board of fixed size with a start and a final cell.
  2. The board should have snakes that move a player down and ladders that move a player up.
  3. Two or more players should take turns.
  4. On each turn a player should roll the dice and move forward by the rolled number.
  5. A player should not move past the final cell.
  6. The first player to reach the final cell should win.
  7. The number and type of dice should be configurable.
- **Idea:** The board knows only snakes/ladders, the dice only roll, and Game owns turns + positions. A turn: roll → try move → follow snake/ladder → check win.
- **Classes:** `Player`, `TurnResult`, `Board` (two maps: snakes, ladders), `Dice`→`StandardDice/SequenceDice`, `DiceSet`, `Game`
- **Pattern:** Strategy (dice) · **Moves:** M4
- **Hard part:** Positions live in `Game`, not on `Player`. Snakes/ladders are maps, not classes. `SequenceDice` makes tests deterministic.
- **Follow-up:** *Extra turn on a 6?* Don't advance the turn index; better, a `TurnRule` object.

</details>

### Problem

1. `SequenceDice(rolls).roll()` returns the rolls in order, cycling round.
2. `Game(size, snakes, ladders, players, dice)`
   - Snakes and ladders are plain dicts `{from: to}`.
   - **Positions live in `Game.positions`** (everyone starts at 0), not on a player object.
3. `play_turn() -> (player, roll, position)`. Players take turns in order.
   - If the move would go past `size`, stay put.
   - Otherwise move, then take one snake or ladder jump.
   - Reaching exactly `size` sets `winner`. After that, `play_turn()` returns `None`.

### Names the driver uses

`SequenceDice`, `Game`: `play_turn`, `positions`, `winner`

### Driver code

```python
if __name__ == "__main__":
    g = Game(size=20, snakes={17: 4}, ladders={3: 11}, players=["ann", "bob"],
             dice=SequenceDice([3, 6, 6, 5, 2, 6, 4, 3]))
    print(g.play_turn())                            # ('ann', 3, 11)
    print(g.play_turn())                            # ('bob', 6, 6)
    print(g.play_turn())                            # ('ann', 6, 4)
    print(g.play_turn())                            # ('bob', 5, 11)
    print(g.play_turn())                            # ('ann', 2, 6)
    print(g.play_turn())                            # ('bob', 6, 4)
    print(g.positions)                              # {'ann': 6, 'bob': 4}
    print(g.play_turn())                            # ('ann', 4, 10)
    print(g.play_turn())                            # ('bob', 3, 7)
    print(g.winner)                                 # None
    g2 = Game(size=10, snakes={}, ladders={}, players=["x"], dice=SequenceDice([6, 6, 4]))
    print(g2.play_turn(), g2.play_turn(), g2.play_turn())  # ('x', 6, 6) ('x', 6, 6) ('x', 4, 10)
    print(g2.winner, g2.play_turn())                # x None
```

### Extend it: Roll two dice.

`MultiDice(dice).roll()` returns the sum of every die's `roll()`. `Game` doesn't change: it only calls `dice.roll()`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    two = MultiDice([SequenceDice([1, 2]), SequenceDice([6])])
    print(two.roll(), two.roll(), two.roll())       # 7 8 7
    g = Game(size=20, snakes={}, ladders={}, players=["ann"], dice=MultiDice([SequenceDice([6]), SequenceDice([4])]))
    print(g.play_turn(), g.play_turn(), g.winner)   # ('ann', 10, 10) ('ann', 10, 20) ann
```

---

## 11 · Tic-Tac-Toe

**Moves:** M2 · **Difficulty:** Easy · **Time:** ~15 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The game should have a square board, 3x3 by default.
  2. Two players with different symbols should play alternate turns.
  3. A player should place a symbol on an empty cell, and invalid moves should be rejected.
  4. A player should win by filling a full row, column or diagonal with their symbol.
  5. The game should end in a draw when the board is full and nobody has won.
  6. The game should track whether it is in progress, won or drawn.
- **Idea:** The Board knows the cells + legality, the Game knows the turn + result. Detect a win by checking **only the lines through the last move**.
- **Classes:** `GameStatus`, `Player`, `Board`, `Game`
- **Pattern:** none (keep it simple!) · **Moves:** M2
- **Hard part:** `is_full` is computed. The Board refuses bad moves itself. Game exposes read-only cells.
- **Follow-up:** *Big N×N board?* Counters per row/col/diagonal (+1 / −1) give an O(1) win check.

</details>

### Problem

1. `Board`
   - `place(r, c, mark) -> bool` **refuses** a cell that's out of range or already taken.
   - `is_full()` and `winner()` are computed.
   - `cells()` returns a **tuple of tuples**, a read-only copy.
2. `Game`: X starts, and `current` is the player to move.
3. `play(r, c)` returns:
   - `"INVALID"` for a refused move, and the turn doesn't change
   - `"X_WINS"` / `"O_WINS"` or `"DRAW"` when the game ends
   - `"OK"` otherwise, and then it's the other player's turn
   - `"GAME_OVER"` for any move after the end

### Names the driver uses

`Game`: `play`, `current`, `board` · `Board`: `cells`, `is_full`, `winner`

### Driver code

```python
if __name__ == "__main__":
    g = Game()
    print(g.play(0, 0), g.play(0, 0), g.play(3, 3), g.current)  # OK INVALID INVALID O
    print(g.play(1, 1), g.play(0, 1), g.play(2, 2), g.play(0, 2))  # OK OK OK X_WINS
    print(g.board.cells())                          # (('X', 'X', 'X'), (None, 'O', None), (None, None, 'O'))
    print(g.play(1, 0), g.play(2, 0), g.play(2, 2))  # GAME_OVER GAME_OVER GAME_OVER
    cells = g.board.cells()
    try:
        cells[0][0] = "O"
    except TypeError:
        print("read-only")                          # read-only

    d = Game()
    moves = [(0, 0), (0, 1), (0, 2), (1, 1), (1, 0), (1, 2), (2, 1), (2, 0), (2, 2)]
    print([d.play(r, c) for r, c in moves])         # ['OK', 'OK', 'OK', 'OK', 'OK', 'OK', 'OK', 'OK', 'DRAW']
    print(d.board.is_full(), d.board.winner())      # True None
```

### Extend it: Play on an n×n board.

`Game(n=3)` passes `n` to `Board(n)`. The win check must work for any `n`, so loop over rows, columns and both diagonals. Don't hard-code 3.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    g = Game(n=4)
    moves = [(0, 0), (1, 0), (0, 1), (1, 1), (0, 2), (1, 2)]
    print([g.play(r, c) for r, c in moves])         # ['OK', 'OK', 'OK', 'OK', 'OK', 'OK']
    print(g.play(0, 3), g.play(3, 3), g.board.n)    # X_WINS GAME_OVER 4
    print(Game().board.n)                           # 3
```

---

## 12 · ATM

**Moves:** M2 M8 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The user should insert a card and enter a PIN to be authenticated.
  2. The machine should behave differently depending on whether it is idle, has a card inserted, has an authenticated user or is dispensing cash.
  3. The user should be able to check the balance, withdraw cash and deposit cash.
  4. A withdrawal should succeed only if the account has enough balance and the machine has enough cash.
  5. Cash should be dispensed by passing the amount through note denominations, largest first.
  6. The card should be blocked after repeated wrong PIN entries.
  7. The user should be able to cancel at any step and get the card back.
  8. Every transaction should be recorded and a receipt provided.
- **Idea:** The bank owns money and PIN attempts, the dispenser owns the notes, and the ATM asks its state what each button does. A withdrawal **checks first, changes last**, with an undo.
- **Classes:** `Account`, `BankService`, `NoteDispenser`, `CashDispenser`, `Transaction`, `TransactionLog`, `Session`, `Response`, `ATMState`→`Idle/HasCard/Authenticated/Dispensing`, `ATM`
- **Pattern:** State + Chain of Responsibility (the amount passes through note denominations, largest first) · **Moves:** M2 M5 M8
- **Hard part:** Order: plan the notes (no change) → debit the account → take the notes → credit back if that fails. "Blocked card" = wrong attempts ≥ limit (computed).
- **Follow-up:** *Two ATMs, same account?* A lock inside `Account` around withdraw/deposit.

</details>

### Problem

1. `Account(pin, balance)` has a `wrong` counter. `is_blocked()` is computed: `wrong >= 3`.
2. `ATM(notes)` takes `notes` as `{denomination: count}`.
3. `plan(amount) -> dict | None`, greedy, largest note first, limited by the counts. It returns `None` if the amount can't be made exactly. **Planning changes nothing.**
4. `check_pin(account, pin) -> bool`. A blocked card always returns `False`. A wrong pin adds 1 to `wrong`. A right pin resets it to 0.
5. `withdraw(account, amount, dispenser_ok=True)` goes in this order:
   1. blocked → `"BLOCKED"`
   2. no plan → `"CANNOT_DISPENSE"`
   3. balance too low → `"INSUFFICIENT_FUNDS"`
   4. **debit** the account
   5. if the dispenser fails → **credit back** and return `"DISPENSE_FAILED"` (notes untouched)
   6. otherwise take the notes and return the plan

### Names the driver uses

`Account`: `balance`, `wrong`, `is_blocked` · `ATM(notes)`: `plan`, `check_pin`, `withdraw`, `notes`

### Driver code

```python
if __name__ == "__main__":
    atm = ATM({500: 2, 200: 3, 100: 1})
    print(atm.plan(1400), atm.plan(1800), atm.plan(250), atm.plan(5000))  # {500: 2, 200: 2} None None None
    print(atm.notes)                                # {500: 2, 200: 3, 100: 1}
    acc = Account(pin="1234", balance=3000)
    print(atm.check_pin(acc, "0000"), atm.check_pin(acc, "1234"), acc.wrong)  # False True 0
    print(atm.withdraw(acc, 1400))                  # {500: 2, 200: 2}
    print(acc.balance, atm.notes)                   # 1600 {500: 0, 200: 1, 100: 1}
    print(atm.withdraw(acc, 250), atm.withdraw(acc, 5000))  # CANNOT_DISPENSE CANNOT_DISPENSE
    print(atm.withdraw(acc, 300, dispenser_ok=False))  # DISPENSE_FAILED
    print(acc.balance, atm.notes)                   # 1600 {500: 0, 200: 1, 100: 1}
    poor = Account(pin="1", balance=100)
    print(atm.withdraw(poor, 200), poor.balance)    # INSUFFICIENT_FUNDS 100
    print([atm.check_pin(acc, "x") for _ in range(3)], acc.is_blocked())  # [False, False, False] True
    print(atm.check_pin(acc, "1234"), atm.withdraw(acc, 100))  # False BLOCKED
```

### Extend it: Let the bank choose how to make up the notes.

1. `ATM(notes, planner=None)`, where `planner(amount, notes) -> dict | None`.
2. `largest_first` is the default and `smallest_first` is the other one. Both are greedy, each in its own order.
3. `plan` calls the planner, and `withdraw` doesn't change.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    notes = {500: 2, 100: 10}
    print(ATM(notes).plan(800), ATM(notes, planner=smallest_first).plan(800))  # {500: 1, 100: 3} {100: 8}
    atm = ATM(notes, planner=smallest_first)
    acc = Account(pin="1", balance=5000)
    print(atm.withdraw(acc, 1200), ATM(notes).withdraw(acc, 1200))  # CANNOT_DISPENSE {500: 2, 100: 2}
```

---

## 13 · Notification Service

**Moves:** M2 M7 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should send notifications to users through multiple channels such as email, SMS and push.
  2. It should be easy to add new channels.
  3. Users should be able to choose which channels and types of notifications they want.
  4. Notification content should be created from templates with placeholders.
  5. Notifications should have types and priorities.
  6. Several interested parties should be notified automatically when an event happens.
  7. Failed deliveries should be retried.
  8. The system should track the delivery status of each notification.
  9. Notifications should be sent without blocking the caller.
- **Idea:** The fast half checks preferences, renders the template, enqueues **one Delivery per channel** and returns. The slow half's worker sends, records and retries. Each delivery has its own status.
- **Classes:** `Template`, `User`, `NotificationChannel`→`Email/Sms/Push`, `ChannelRegistry`, `Delivery`, `Notification`, `RetryPolicy`, `DeliveryQueue`, `DeliveryWorker`, `Event`, `EventListener`, `EventPublisher`, `NotificationService`
- **Pattern:** Strategy (channels), Observer (events), queue + worker · **Moves:** M2 M7
- **Hard part:** A priority queue that also holds retries until their time. Notification status is computed from its deliveries.
- **Follow-up:** *Duplicate sends?* An idempotency key per delivery. *Crash?* Swap in a durable queue; the worker is unchanged.

</details>

### Problem

No threads here. `process(now)` plays the worker.

1. `NotificationService(send, max_attempts=3, backoff=10)`. `send(channel, user, msg)` returns `True` or `False`.
2. `notify(user, msg, channels, now) -> "N1", "N2", …` creates **one delivery per channel** and puts each in a min-heap keyed by ready time `now`. Use a sequence number to break ties, so order follows insertion.
3. `process(now) -> int` handles every delivery whose ready time is `<= now` (including retries that become ready) and returns how many it attempted.
   - On success the delivery is `SENT`.
   - On failure: if `attempts == max_attempts` it's `FAILED`, otherwise push it back with ready time `now + backoff × 2^(attempts−1)`.
4. `status(nid)` is **computed from the deliveries**: `PENDING` if any is pending, else `SENT` if all were sent, else `FAILED`.
5. `delivery_states(nid)` returns a list of `(channel, state, attempts)`.

### Names the driver uses

`NotificationService`: `notify`, `process`, `status`, `delivery_states`

### Driver code

```python
if __name__ == "__main__":
    sms_fails = {"count": 0}
    def send(channel, user, msg):
        if channel == "push":
            return False
        if channel == "sms":
            sms_fails["count"] += 1
            return sms_fails["count"] >= 2
        return True

    svc = NotificationService(send, max_attempts=3, backoff=10)
    n1 = svc.notify("alice", "Order shipped", ["email", "sms"], now=0)
    n2 = svc.notify("bob", "Sale!", ["push"], now=0)
    print(n1, n2, svc.status(n1))                   # N1 N2 PENDING
    print(svc.process(now=0))                       # 3
    print(svc.delivery_states(n1), svc.status(n1))  # [('email', 'SENT', 1), ('sms', 'PENDING', 1)] PENDING
    print(svc.process(now=9))                       # 0
    print(svc.process(now=10))                      # 2
    print(svc.delivery_states(n1), svc.status(n1))  # [('email', 'SENT', 1), ('sms', 'SENT', 2)] SENT
    print(svc.delivery_states(n2), svc.status(n2))  # [('push', 'PENDING', 2)] PENDING
    print(svc.process(now=29), svc.process(now=30))  # 0 1
    print(svc.delivery_states(n2), svc.status(n2))  # [('push', 'FAILED', 3)] FAILED
    n3 = svc.notify("carol", "Hi", ["email", "push"], now=100)
    print(svc.process(now=1000), svc.process(now=1000), svc.process(now=1000))  # 2 0 0
    print(svc.status(n3), svc.delivery_states(n3))  # PENDING [('email', 'SENT', 1), ('push', 'PENDING', 1)]
```

### Extend it: Some channels want a fixed retry delay.

1. `NotificationService(send, max_attempts=3, backoff=10, retry=None)`, where `retry.delay(attempts) -> seconds`.
2. `ExponentialBackoff(base)` is the default (`base × 2^(attempts−1)`). `FixedDelay(seconds)` always returns the same delay.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    svc = NotificationService(lambda ch, u, m: False, max_attempts=3, retry=FixedDelay(5))
    n = svc.notify("ann", "hi", ["sms"], now=0)
    print(svc.process(0), svc.process(4), svc.process(5), svc.process(9), svc.process(10))  # 1 0 1 0 1
    print(svc.status(n), svc.delivery_states(n))    # FAILED [('sms', 'FAILED', 3)]
```

---

## 14 · Food Delivery

**Moves:** M1 M3 M8 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. Users should be able to browse and search restaurants and their menus.
  2. Users should be able to add items from one restaurant to a cart.
  3. Users should be able to place an order with a delivery address and pay for it.
  4. A restaurant should be able to accept or reject an order and mark it as being prepared.
  5. The system should assign a delivery partner to each order.
  6. An order should move through the stages of placed, accepted, preparing, picked up, delivered and cancelled, and the user should be notified at each change.
  7. Users should be able to track their order.
  8. The system should support coupons and discounts on the order total.
  9. The system should support multiple payment methods.
  10. Users should be able to rate the restaurant and the delivery partner.
- **Idea:** An Order runs its own stages and notifies its watchers on every change. Reserve the delivery partner **before** payment, so a paid order always has a rider. Every later failure has an undo.
- **Classes:** `Restaurant`, `RestaurantCatalog`, `Cart`, `OrderLine`, `DeliveryPartner`, `DiscountStrategy`→`Percent/Flat`, `Coupon`, `PaymentMethod`, `OrderObserver`→`UserNotifier`, `OrderTracking`, `Order`, `OrderService`
- **Pattern:** Observer (order changes), Strategy (discount, payment) · **Moves:** M1 M3 M8
- **Hard part:** The cart stores ids + counts, and prices are read from the menu at order time. The discount is stored once on the order.
- **Follow-up:** *Restaurant never accepts?* Timeout + sweep rejects stale PLACED orders (M6).

</details>

### Problem

1. `Cart` stores **only item ids and quantities** in `items`. `remove` takes the quantity down, and deletes the item at 0.
2. `FoodService(menu)` takes `menu` as `{item_id: price}`. `menu` is a public dict.
3. `place_order(cart, coupon_pct=0, pay=...) -> Order | None`
   - Return `None` if the cart is empty or has an unknown item.
   - Read the prices from the menu **now**. `discount = total × pct // 100`, stored once.
   - Call `pay(total − discount)`. If it fails, return `None` and leave the cart as it was.
   - On success, return order `"O1"`, `"O2"`, … and clear the cart.
4. `Order` has `items` (a copy), `total`, `discount`, `payable` (computed) and `partner`.
   - `advance()` moves `PLACED → ACCEPTED → PICKED_UP → DELIVERED`. It refuses (`False`) to leave `ACCEPTED` without a partner, and returns `False` at the end. Reaching `DELIVERED` releases the partner.
   - `cancel()` works only from `PLACED` or `ACCEPTED`, and releases the partner if there is one.
5. `DeliveryPartner(pid)`
   - `try_assign(order_id) -> bool` under its own lock.
   - `release(order_id)` frees the partner only if they're on that order.
   - `is_free()` is computed.
6. `assign_partner(order, partners) -> pid | None`
   - The order must be `ACCEPTED` and have no partner yet.
   - Try the partners in order and take the first one that accepts.

### Names the driver uses

`Cart`: `add`, `remove`, `items` · `FoodService(menu)`: `place_order`, `assign_partner`, `menu` · `Order`: `order_id`, `items`, `total`, `discount`, `payable`, `status`, `advance`, `cancel` · `DeliveryPartner`: `try_assign`, `is_free`

### Driver code

```python
if __name__ == "__main__":
    svc = FoodService({"pizza": 300, "coke": 50})
    cart = Cart()
    cart.add("pizza"); cart.add("coke", 3); cart.remove("coke")
    print(cart.items)                               # {'pizza': 1, 'coke': 2}
    print(svc.place_order(Cart()))                  # None
    print(svc.place_order(cart, pay=lambda amt: False), cart.items)  # None {'pizza': 1, 'coke': 2}
    o = svc.place_order(cart, coupon_pct=10)
    print(o.order_id, o.items, o.total, o.discount, o.payable, cart.items)  # O1 {'pizza': 1, 'coke': 2} 400 40 360 {}
    svc.menu["pizza"] = 999
    print(o.total, o.payable)                       # 400 360
    bad = Cart(); bad.add("sushi")
    print(svc.place_order(bad))                     # None

    p1, p2 = DeliveryPartner("P1"), DeliveryPartner("P2")
    print(svc.assign_partner(o, [p1, p2]))          # None
    print(o.advance(), o.status, o.advance())       # True ACCEPTED False
    print(svc.assign_partner(o, [p1, p2]), p1.is_free())  # P1 False
    print(o.cancel(), o.status, o.advance(), p1.is_free())  # True CANCELLED False True

    c2 = Cart(); c2.add("coke")
    o2 = svc.place_order(c2)
    o2.advance()
    print(p2.try_assign("someone-else"), svc.assign_partner(o2, [p2, p1]))  # True P1
    print(o2.advance(), o2.advance(), o2.status, p1.is_free())  # True True DELIVERED True
    print(o2.cancel(), o2.advance(), o2.status)     # False False DELIVERED
```

### Extend it: New coupon types.

1. `place_order(..., discount=None)`. If `discount` is given, it replaces `coupon_pct`. It has `amount(total)`.
2. `PercentOff(pct)` returns `total × pct // 100`.
3. `FlatOff(amount, min_total=0)` returns `min(amount, total)` if `total >= min_total`, and 0 otherwise.
4. The discount is still stored once on the order.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    svc = FoodService({"pizza": 300, "coke": 50})
    def cart(**items):
        c = Cart()
        for k, v in items.items():
            c.add(k, v)
        return c
    o1 = svc.place_order(cart(pizza=2), discount=FlatOff(100, min_total=500))
    o2 = svc.place_order(cart(coke=2), discount=FlatOff(100, min_total=500))
    o3 = svc.place_order(cart(pizza=1), discount=PercentOff(20))
    print(o1.payable, o2.payable, o3.payable)       # 500 100 240
```

---

## 15 · Library Management

**Moves:** M1 M3 M9 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should allow adding, updating and removing books, with multiple copies of each book.
  2. Users should be able to search books by title, author or subject.
  3. Members should be able to register and hold a membership.
  4. A member should be able to borrow books up to a limit, with a due date for each.
  5. A member should be able to return books, and a late fee should be charged after the due date.
  6. A member should be able to reserve a book that is currently borrowed and be notified when it is available.
  7. Each copy of a book should track whether it is available, borrowed, reserved or lost.
  8. Librarians and members should have different permissions.
- **Idea:** A Book (the title) and a BookCopy (the physical object) are different. The title owns its copies and the waiting list. A return goes through the title, so the copy is freed or handed to the next person waiting, who is notified.
- **Classes:** `Book`, `BookCopy`, `CopyStatus`, `Loan`, `User`, `Membership`, `SearchCriterion`→`Title/Author/SubjectContains`, `PermissionPolicy`, `LateFeePolicy`, `ReservationObserver`→`MemberNotifier`, `LibraryService`
- **Pattern:** Observer (reservations), Strategy (search) · **Moves:** M1 M3 M9
- **Hard part:** Returns are routed through `Book`, so the waiting list is always served. Permissions are a (role, action) data table, checked in one place.
- **Follow-up:** *Reserved copy never picked up?* Give the hold an expiry plus a sweep that passes it to the next person (M6).

</details>

### Problem

A `Book` (the title) owns its copies (`"Dune#1"`, …) and a waitlist. **Every return goes through the `Book`**, so the waitlist is always served.

1. Permissions are a data table: `can(role, action)` is true only for `(LIBRARIAN, add_book)`, `(LIBRARIAN, borrow)` and `(MEMBER, borrow)`.
2. `Library.add_book(role, title, n_copies) -> bool`.
3. `borrow(role, member, title, now) -> copy_id | None`
   - Return `None` if the role isn't allowed or the title is unknown.
   - Take the first `AVAILABLE` copy (now `BORROWED`), with due date `now + 14`.
   - If there's none, add the member to the waitlist (once) and return `None`.
4. `return_copy(copy_id, now) -> (fee, notified_member)`
   - The fee is `10` per day after the due date.
   - If someone is waiting, the copy becomes `RESERVED` for the first person waiting, and you return that member. Otherwise it's `AVAILABLE` and the member is `None`.
5. `pick_up(member, title, now) -> copy_id | None` takes the copy reserved for that member. It's then `BORROWED` with a new due date.
6. `copy_status(copy_id)`.

### Names the driver uses

`can`, `Library`: `add_book`, `borrow`, `return_copy`, `pick_up`, `copy_status`

### Driver code

```python
if __name__ == "__main__":
    lib = Library()
    print(lib.add_book("MEMBER", "Dune", 1), lib.add_book("LIBRARIAN", "Dune", 1))  # False True
    print(can("MEMBER", "borrow"), can("MEMBER", "add_book"), can("GUEST", "borrow"))  # True False False
    c = lib.borrow("MEMBER", "ann", "Dune", now=0)
    print(c, lib.copy_status(c))                    # Dune#1 BORROWED
    print(lib.borrow("MEMBER", "bob", "Dune", now=1), lib.borrow("MEMBER", "cat", "Dune", now=2))  # None None
    print(lib.borrow("GUEST", "dan", "Dune", now=2), lib.borrow("MEMBER", "dan", "Nope", now=2))  # None None
    print(lib.return_copy(c, now=17))               # (30, 'bob')
    print(lib.copy_status(c))                       # RESERVED
    print(lib.borrow("MEMBER", "eve", "Dune", now=18))  # None
    print(lib.pick_up("cat", "Dune", now=18), lib.pick_up("bob", "Dune", now=18))  # None Dune#1
    print(lib.copy_status(c))                       # BORROWED
    print(lib.return_copy(c, now=20))               # (0, 'cat')
    print(lib.pick_up("cat", "Dune", now=21))       # Dune#1
    print(lib.return_copy(c, now=30), lib.copy_status(c))  # (0, 'eve') RESERVED
```

### Extend it: Cap the late fee.

1. `Library(fee_policy=None)`, where `fee_policy.fee(days_late)`.
2. `PerDayFee(10)` is the default. `CappedFee(per_day, cap)` returns `min(cap, days × per_day)`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    lib = Library(fee_policy=CappedFee(per_day=10, cap=50))
    lib.add_book("LIBRARIAN", "Dune", 2)
    a = lib.borrow("MEMBER", "ann", "Dune", now=0)
    b = lib.borrow("MEMBER", "bob", "Dune", now=0)
    print(lib.return_copy(a, now=17), lib.return_copy(b, now=100))  # (30, None) (50, None)
    print(Library().fee_policy.fee(9))              # 90
```

---

## 16 · Chess

**Moves:** M2 · **Difficulty:** Hard · **Time:** ~35 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The game should have an 8x8 board with 32 pieces in the standard starting position.
  2. Two players should play alternate turns, with white moving first.
  3. Each type of piece should move according to its own rules.
  4. A player should be able to capture opponent pieces.
  5. The game should reject illegal moves such as moving through other pieces, capturing own pieces or leaving the king in check.
  6. The game should detect check, checkmate and stalemate.
  7. The game should support castling, en passant and pawn promotion.
  8. The game should end in a win, a draw or a resignation.
  9. The game should record the history of moves.
- **Idea:** Each piece knows its own moves. The board is **immutable** (a move returns a new board). A move is legal only if **your king isn't attacked after making it on a copy**. That one test covers pins, discovered checks and moving into check.
- **Classes:** `Move`, `Piece`→`SlidingPiece`→`Rook/Bishop/Queen`, `Knight`, `King`, `Pawn`, `Board`, `HistoryEntry`, `Game`
- **Pattern:** polymorphism via inheritance (Strategy built into the classes) · **Moves:** —
- **Hard part:** `attacked_squares` is separate from `candidate_moves`. Pieces are immutable (moving creates a new piece with `has_moved`). Special moves: castling, en passant, promotion.
- **Follow-up:** *Copying is slow?* Make/unmake or bitboards behind the same `Board` API.

</details>

### Problem

No pawns and no castling. Only K, Q, R, B, N.

1. `Piece = namedtuple("Piece", "kind color has_moved", defaults=(False,))` is immutable. A board is a dict `{"e1": Piece(...)}`.
2. `candidate_moves(board, square) -> sorted list` of squares the piece can move to. Sliders stop at the first piece and may capture an enemy there. They can't land on their own piece.
3. `attacked_squares(board, color) -> set` of every square that side attacks, **including squares occupied by its own pieces** (they're defended). This is separate from candidate moves.
4. `in_check(board, color)`: that side's king is on a square the enemy attacks.
5. `move(board, frm, to) -> new board`. The old board is untouched, and the moved piece is replaced with `has_moved=True`.
6. `legal_moves(board, square)`: the candidate moves that don't leave your own king in check.

### Names the driver uses

`Piece`, `candidate_moves`, `attacked_squares`, `in_check`, `move`, `legal_moves`

### Driver code

```python
if __name__ == "__main__":
    board = {
        "e1": Piece("K", "white"), "e2": Piece("R", "white"), "b1": Piece("N", "white"),
        "e8": Piece("K", "black"), "e7": Piece("Q", "black"), "h4": Piece("B", "black"),
    }
    print(candidate_moves(board, "b1"))             # ['a3', 'c3', 'd2']
    print(candidate_moves(board, "e2"))             # ['a2', 'b2', 'c2', 'd2', 'e3', 'e4', 'e5', 'e6', 'e7', 'f2', 'g2', 'h2']
    print(legal_moves(board, "e2"))                 # []
    print("e1" in attacked_squares(board, "black"), in_check(board, "white"))  # True True
    print(sorted(attacked_squares({"a1": Piece("R", "white"), "a3": Piece("N", "white")}, "white")))  # ['a2', 'a3', 'b1', 'b5', 'c1', 'c2', 'c4', 'd1', 'e1', 'f1', 'g1', 'h1']
    print(legal_moves(board, "e1"))                 # ['d1', 'd2', 'f1']

    after = move(board, "b1", "c3")
    print("b1" in board, board["b1"].has_moved, after["c3"])  # True False Piece(kind='N', color='white', has_moved=True)
    try:
        after["c3"].has_moved = False
    except AttributeError:
        print("immutable")                          # immutable
    check = {"e1": Piece("K", "white"), "a2": Piece("R", "white"), "e8": Piece("R", "black"), "h8": Piece("K", "black")}
    print(in_check(check, "white"))                 # True
    print(legal_moves(check, "a2"))                 # ['e2']
    print(legal_moves(check, "e1"))                 # ['d1', 'd2', 'f1', 'f2']
```

### Extend it: Add a new piece without editing the move code.

`register_piece(kind, steps=None, dirs=None)` adds the piece's jumps or slide directions to the movement tables. `candidate_moves`, `attacked_squares` and `in_check` then work for it, with no `if kind == ...` anywhere.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    register_piece("C", steps=[(1, 3), (3, 1), (-1, 3), (-3, 1), (1, -3), (3, -1), (-1, -3), (-3, -1)])
    print(candidate_moves({"a1": Piece("C", "white")}, "a1"))  # ['b4', 'd2']
    register_piece("W", dirs=[(1, 0)])
    print(candidate_moves({"a1": Piece("W", "white"), "d1": Piece("K", "black")}, "a1"))  # ['b1', 'c1', 'd1']
    board = {"e1": Piece("K", "white"), "d4": Piece("C", "black"), "h8": Piece("K", "black")}
    print(in_check(board, "white"))                 # True
```

---

## 17 · Task Scheduler

**Moves:** M3 M7 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should allow scheduling a task to run once at a given time or repeatedly at a fixed interval.
  2. Users should be able to cancel or update a scheduled task.
  3. The system should run tasks when they become due, earliest first.
  4. Multiple tasks should run at the same time using a pool of workers.
  5. Tasks should be able to have priorities.
  6. A task should move through the stages of scheduled, running, completed and failed.
  7. Failed tasks should be retried.
  8. It should be easy to add new kinds of tasks.
  9. The scheduler should stay correct when tasks are added while others are running.
- **Idea:** A job = what + when. A queue hands out due jobs in time order, and workers pull them. **Every change to a job bumps its version**, so stale queue entries are simply skipped.
- **Classes:** `Schedule`→`OneTime/FixedInterval`, `Task`→`PrintTask/FlakyTask`, `Clock`, `QueueEntry`, `JobQueue` (min-heap by run_at), `ScheduledJob`, `Scheduler`, `WorkerPool`
- **Pattern:** Command (task), Strategy (schedule), queue + workers · **Moves:** M3 M7
- **Hard part:** Version numbers instead of deleting from the heap. `start_run` checks version + stage atomically. The next run is counted from the **planned** time (no drift). Run the task with **no lock held**.
- **Follow-up:** *Idle workers polling?* Use a condition variable signalled by `add`, and wait until the next `run_at`.

</details>

### Problem

No threads here. `run_due(now)` plays the worker.

1. `schedule(task_id, run_at, every=None)` pushes `(run_at, task_id, version)` onto a heap. Scheduling the same id again **bumps the version**, so the old heap entry is ignored. Don't delete it from the heap.
2. `cancel(task_id) -> bool` works only while `SCHEDULED`. It bumps the version and sets `CANCELLED`.
3. `run_due(now) -> list of (task_id, planned_time)` pops every entry with time `<= now`:
   - skip entries whose version is stale
   - a one-off task becomes `DONE`
   - a recurring task is pushed again at **`planned + every`** (counted from the planned time, so there's no drift), and the same call catches up on any runs it missed
4. `stage(task_id)`.

### Names the driver uses

`Scheduler`: `schedule`, `cancel`, `run_due`, `stage`

### Driver code

```python
if __name__ == "__main__":
    s = Scheduler()
    s.schedule("backup", run_at=5)
    s.schedule("report", run_at=10)
    s.schedule("ping", run_at=0, every=10)
    print(s.run_due(4))                             # [('ping', 0)]
    print(s.run_due(5), s.stage("backup"))          # [('backup', 5)] DONE
    s.schedule("report", run_at=20)
    print(s.run_due(15))                            # [('ping', 10)]
    print(s.run_due(35))                            # [('ping', 20), ('report', 20), ('ping', 30)]
    print(s.cancel("ping"), s.cancel("ping"), s.stage("ping"))  # True False CANCELLED
    print(s.run_due(100))                           # []
    print(s.stage("report"), s.cancel("report"))    # DONE False
    s.schedule("ping", run_at=101, every=50)
    print(s.run_due(101), s.stage("ping"))          # [('ping', 101)] SCHEDULED
```

### Extend it: Run a task only N times.

1. `schedule(..., rule=None)`, where `rule.next(planned) -> next time | None`. `None` means the task is `DONE`.
2. `every=x` still works. It means `Every(x)`.
3. `Times(n, period)` runs exactly `n` times, each `period` apart.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    s = Scheduler()
    s.schedule("trial-reminder", run_at=0, rule=Times(3, period=10))
    s.schedule("tick", run_at=0, rule=Every(25))
    print(s.run_due(100))                           # [('tick', 0), ('trial-reminder', 0), ('trial-reminder', 10), ('trial-reminder', 20), ('tick', 25), ('tick', 50), ('tick', 75), ('tick', 100)]
    print(s.stage("trial-reminder"), s.stage("tick"))  # DONE SCHEDULED
```

---

## 18 · Amazon Locker

**Moves:** M1 M2 M3 M6 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The locker station should have compartments of different sizes.
  2. A delivery agent should be able to deposit a package, and the system should assign the smallest free compartment that fits it.
  3. The system should generate a unique access code for each package and send it to the customer.
  4. The customer should be able to open the compartment by entering the code and collect the package.
  5. The code should expire after a set time, and the package should then be removed and the compartment freed.
  6. A compartment should track whether it is available, occupied or out of service.
  7. Two packages must never be assigned to the same compartment at the same time.
- **Idea:** Each compartment guards itself with a lock. One `Assignment` record joins code + package + compartment, and decides **once** whether the package was collected or expired.
- **Classes:** `Package`, `Compartment`, `CodeGenerator`→`RandomCodeGenerator`, `CustomerNotifier`→`InboxNotifier`, `Assignment`, `LockerStation`
- **Pattern:** none named (just M1 + M6) · **Moves:** M1 M2 M3 M6
- **Hard part:** The code is made unique with `put_if_absent` and a retry. Compartment status is computed from whether it holds a package.
- **Follow-up:** *Brute-forcing codes?* A wrong-attempt counter checked at the start of `collect`, then lock out.

</details>

### Problem

1. `Compartment(cid, size)`: `status()` is **computed** (`AVAILABLE` if it holds no package, else `OCCUPIED`). `try_put(package_id, size)` takes it under a lock.
2. `LockerStation(compartments, code_gen, ttl=72)`
3. `deposit(package_id, size, now) -> code | None`
   - Pick the **smallest** free compartment that fits.
   - Then get a code from `code_gen()` and **claim it only if it isn't in use** (put-if-absent). If it is in use, call `code_gen()` again.
   - Return `None` without using a code if nothing fits.
4. One assignment per code decides **once** whether the package was `COLLECTED` or `EXPIRED`.
   - `collect(code, now) -> package_id | None` works only if the assignment is active and `now < expires_at` (`expires_at = now + ttl`).
   - `expire_stale(now) -> [package_id]` expires the rest.
   - Both free the compartment.
5. `status(cid)`.

### Names the driver uses

`Compartment(cid, size)`, `LockerStation`: `deposit`, `collect`, `expire_stale`, `status`

### Driver code

```python
if __name__ == "__main__":
    codes = iter(["111", "111", "222", "111", "222", "333", "444"])
    st = LockerStation([Compartment("L1", 3), Compartment("S1", 1), Compartment("M1", 2)],
                       code_gen=lambda: next(codes), ttl=72)
    c1 = st.deposit("PKG-A", 1, now=0)
    c2 = st.deposit("PKG-B", 1, now=0)
    print(c1, c2, st.status("S1"), st.status("M1"))  # 111 222 OCCUPIED OCCUPIED
    c3 = st.deposit("PKG-C", 3, now=0)
    print(c3, st.deposit("PKG-D", 1, now=0))        # 333 None
    print(st.collect("999", now=1), st.collect(c1, now=1), st.status("S1"))  # None PKG-A AVAILABLE
    print(st.collect(c1, now=2))                    # None
    print(st.collect(c2, now=72), st.status("M1"))  # None OCCUPIED
    print(st.expire_stale(72), st.status("M1"), st.status("L1"))  # ['PKG-B', 'PKG-C'] AVAILABLE AVAILABLE
    print(st.collect(c3, now=10), st.expire_stale(100))  # None []
    c4 = st.deposit("PKG-E", 2, now=100)
    print(c4, st.status("S1"))                      # 444 AVAILABLE
```

### Extend it: Tell the customer what happened.

1. `LockerStation(..., notify=None)`, where `notify(package_id, message)` is called:
   - after a deposit, with `"CODE <code>"`
   - after a collect, with `"COLLECTED"`
   - after a sweep, with `"EXPIRED"`
2. Call it only after the assignment has decided.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    inbox = []
    codes = iter(["111", "222"])
    st = LockerStation([Compartment("S1", 1), Compartment("S2", 1)], code_gen=lambda: next(codes), ttl=10,
                       notify=lambda pkg, msg: inbox.append((pkg, msg)))
    c1, c2 = st.deposit("A", 1, now=0), st.deposit("B", 1, now=0)
    st.collect(c1, now=5)
    st.expire_stale(10)
    print(inbox)                                    # [('A', 'CODE 111'), ('B', 'CODE 222'), ('A', 'COLLECTED'), ('B', 'EXPIRED')]
```

---

## 19 · Bounded Queue + Pub-Sub

**Moves:** M7 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The queue should hold a fixed number of items.
  2. Producers should add items, and should wait when the queue is full.
  3. Consumers should remove items in the order they were added, and should wait when the queue is empty.
  4. Multiple producers and consumers should work at the same time without losing or duplicating items.
  5. Adding and removing should support an optional timeout.
  6. Publishers should be able to publish messages to named topics.
  7. Subscribers should be able to subscribe to and unsubscribe from topics.
  8. Every subscriber of a topic should receive each message published after it subscribed.
  9. Each subscriber should consume at its own pace without affecting others.
  10. Messages within a topic should be delivered in the order they were published.
  11. The system should shut down cleanly.
- **Idea:** Both are built on a **monitor** (lock + condition variables). The queue gives each item to exactly one consumer. Pub-sub keeps one log per topic, and each subscriber keeps only its **offset**.
- **Classes:** `BoundedQueue` (lock, `not_full`, `not_empty`), `Message`, `Topic` (log + offsets), `Broker`
- **Pattern:** Producer–Consumer, Pub-Sub (log/Kafka-style) · **Moves:** M7
- **Hard part:** `while` (not `if`) around `wait()`. Trim the log up to the slowest offset. Hand out offsets under the topic lock.
- **Follow-up:** *Why `notify_all`?* It's safe with timeouts and shutdown; every woken thread re-checks its condition.

</details>

### Problem

1. `BoundedQueue(capacity)`: `offer(x) -> bool` returns `False` when full. `poll()` returns `None` when empty. `size()`.
2. `Topic(name)` is a log with absolute offsets.
   - `publish(msg)` returns its offset (0, 1, …).
   - `subscribe(group)` starts the group at the **oldest kept** offset. Subscribing again changes nothing.
   - `poll(group, max_n=10)` returns up to `max_n` messages from the group's offset and moves the offset forward.
   - `trim() -> int` drops every message below the **slowest** group's offset and returns how many it dropped.
   - `retained()` returns how many messages are kept.
   - Do every step under the topic lock.

### Names the driver uses

`BoundedQueue`: `offer`, `poll`, `size` · `Topic`: `publish`, `subscribe`, `poll`, `trim`, `retained`

### Driver code

```python
if __name__ == "__main__":
    q = BoundedQueue(2)
    print(q.offer("a"), q.offer("b"), q.offer("c"), q.size())  # True True False 2
    print(q.poll(), q.offer("c"), q.poll(), q.poll(), q.poll())  # a True b c None

    t = Topic("orders")
    t.subscribe("billing")
    print([t.publish(m) for m in ["o1", "o2", "o3", "o4"]])  # [0, 1, 2, 3]
    t.subscribe("email")
    print(t.poll("billing", max_n=3), t.poll("email", max_n=1))  # ['o1', 'o2', 'o3'] ['o1']
    print(t.trim(), t.retained())                   # 1 3
    print(t.poll("email"), t.trim(), t.retained())  # ['o2', 'o3', 'o4'] 2 1
    print(t.poll("billing"), t.trim(), t.retained())  # ['o4'] 1 0
    print(t.publish("o5"), t.poll("billing"), t.poll("email"))  # 4 ['o5'] ['o5']
    t.subscribe("audit")
    print(t.poll("audit"), t.trim(), t.retained())  # ['o5'] 1 0
```

### Extend it: A consumer group only wants some messages.

1. `subscribe(group, filter=None)`, where `filter(msg) -> bool`.
2. `poll` returns up to `max_n` **matching** messages. The offset moves past every message it scanned, matching or not, so `trim` still works.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    t = Topic("orders")
    t.subscribe("big-orders", filter=lambda m: m["amount"] >= 100)
    t.subscribe("all")
    for amt in [50, 150, 20, 300, 10]:
        t.publish({"amount": amt})
    print([m["amount"] for m in t.poll("big-orders", max_n=1)])  # [150]
    print([m["amount"] for m in t.poll("big-orders")], t.offsets["big-orders"])  # [300] 5
    print(len(t.poll("all")), t.trim(), t.retained())  # 5 5 0
```

---

## 20 · In-Memory File System

**Moves:** M2 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The file system should organize files and directories in a tree under a single root directory.
  2. Users should be able to create, delete, rename and move files and directories.
  3. A directory should be able to contain both files and other directories, and both should be treated the same way where possible.
  4. Users should be able to read, write and append to a file.
  5. Users should be able to list the contents of a directory.
  6. Users should be able to refer to items by absolute and relative paths.
  7. Users should be able to search for files by name or extension.
  8. The system should report the size of a file or of a whole directory.
- **Idea:** File and Directory are both a `Node` (Composite), so size/search/delete/move treat them the same. A name is stored **only as the parent's map key**. Paths are normalized (`.`/`..`) and walked from the root, with no parent pointers.
- **Classes:** `Node`→`File/Directory`, `EntryInfo`, `SearchCriterion`→`NameEquals/ExtensionEquals`, `FileSystem`
- **Pattern:** Composite, Strategy (search) · **Moves:** M2
- **Hard part:** Rename = one map key change. Directory size = sum of its children (computed). Prevent moving a directory into itself.
- **Follow-up:** *Slow size?* A cached size updated along the path to the root.

</details>

### Problem

Paths look like `"/a/b"`. Use a `Dir` (a dict of children) and a `File` (content).

1. `mkdir(path)` creates any missing parents. It returns `False` if a file is in the way.
2. `write(path, content) -> bool` returns `False` if the parent doesn't exist or the path is a directory.
3. `read(path)` returns the content, or `None`. `ls(path)` returns sorted child names, or `None`.
4. `size(path)` is **computed**: a file's size is `len(content)`, and a directory's is the sum of its children. Don't store it.
5. `mv(src, dst) -> bool`, where `dst` is the full new path. A move or rename is **one key move between parent dicts**. Refuse when:
   - `dst` is `src` or inside `src` (moving a directory into itself)
   - `src` is missing
   - the destination parent is missing
   - `dst` already exists

### Names the driver uses

`FileSystem`: `mkdir`, `write`, `read`, `ls`, `size`, `mv`

### Driver code

```python
if __name__ == "__main__":
    fs = FileSystem()
    print(fs.mkdir("/a/b/c"), fs.ls("/"), fs.ls("/a/b"))  # True ['a'] ['c']
    print(fs.write("/a/b/notes.txt", "hello"), fs.write("/a/x.txt", "abc"), fs.write("/nope/f.txt", "z"))  # True True False
    print(fs.size("/a/b"), fs.size("/a"), fs.size("/"), fs.size("/a/x.txt"))  # 5 8 8 3
    print(fs.mv("/a/b", "/a/b/c/b"), fs.mv("/a", "/a"))  # False False
    print(fs.mv("/a/b/notes.txt", "/a/b/c/readme.txt"), fs.read("/a/b/c/readme.txt"), fs.ls("/a/b"))  # True hello ['c']
    print(fs.mv("/a/b", "/top"), fs.ls("/"), fs.ls("/a"), fs.size("/top"))  # True ['a', 'top'] ['x.txt'] 5
    print(fs.mv("/a/x.txt", "/top"), fs.mv("/ghost", "/g"), fs.mv("/a/x.txt", "/missing/x.txt"))  # False False False
    print(fs.write("/top", "oops"), fs.mkdir("/a/x.txt/y"))  # False False
```

### Extend it: Search for files.

1. `find(path, criterion) -> sorted full paths` of the files anywhere under `path` that match. An unknown path returns `[]`.
2. `criterion.matches(name, size)`: `NameEndsWith(suffix)` and `SizeAtLeast(n)`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    fs = FileSystem()
    fs.mkdir("/docs/old")
    fs.write("/docs/a.txt", "hello world")
    fs.write("/docs/b.md", "hi")
    fs.write("/docs/old/c.txt", "x")
    print(fs.find("/", NameEndsWith(".txt")))       # ['/docs/a.txt', '/docs/old/c.txt']
    print(fs.find("/docs", SizeAtLeast(2)))         # ['/docs/a.txt', '/docs/b.md']
    print(fs.find("/docs/old", NameEndsWith(".md")), fs.find("/nope", SizeAtLeast(0)))  # [] []
```

---

## 21 · KV Store with TTL + Transactions

**Moves:** M2 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The store should allow setting, getting and deleting keys.
  2. A key should optionally have a time to live, and an expired key should not be returned.
  3. Expired keys should be cleaned up automatically.
  4. The store should support beginning, committing and rolling back a transaction.
  5. Transactions should be allowed to nest.
  6. Changes made inside a transaction should stay hidden from others until it is committed and be discarded on rollback.
  7. The store should stay correct when accessed by multiple threads.
- **Idea:** The shared store holds only committed data. Each session has a **stack of change layers**, one per open transaction. A read checks its own layers top-down, then the store. Inner commit = merge into the layer below. Outer commit = apply to the store under one lock.
- **Classes:** `Clock`, `Entry` (value + expires_at), `Change`, `TransactionLayer`, `KeyValueStore`, `Session`, `ExpiryCleaner`
- **Pattern:** none named (layer stack) · **Moves:** M2
- **Hard part:** Delete inside a txn = a tombstone in the layer. Rollback = pop a layer. Expiry is checked lazily on read, plus a background cleaner.
- **Follow-up:** *Conflicting sessions?* Optimistic concurrency: version per key, refuse the commit if a read key changed.

</details>

### Problem

1. `base` is a public dict `{key: (value, expires_at or None)}`. Open transactions are a stack of layers on top of it.
2. `set(key, value, now, ttl=None)` writes to the top layer, or to `base` if no transaction is open.
3. `delete(key)` writes a **tombstone** inside a transaction, and removes the key directly with no transaction open.
4. `get(key, now)` looks through the layers from top to bottom, then `base`. A tombstone means `None`. **Expiry is checked lazily**: an expired key in `base` is deleted on read and returns `None`.
5. `begin()`. `rollback() -> bool` pops the top layer. `commit() -> bool` merges the top layer into the one below it (or `base`), and a tombstone deletes the key there. Both return `False` with no transaction open.
6. `cleanup(now) -> int` is the background cleaner. It removes the expired keys from `base`.

### Names the driver uses

`KVStore`: `set`, `get`, `delete`, `begin`, `commit`, `rollback`, `cleanup`, `base`

### Driver code

```python
if __name__ == "__main__":
    kv = KVStore()
    kv.set("a", 1, now=0)
    kv.set("tmp", "x", now=0, ttl=10)
    print(kv.get("a", 5), kv.get("tmp", 9), kv.get("tmp", 10), "tmp" in kv.base)  # 1 x None False

    kv.begin()
    kv.set("a", 2, now=20)
    kv.delete("b")
    kv.set("b", 3, now=20)
    kv.begin()
    kv.delete("a")
    print(kv.get("a", 20), kv.get("b", 20))         # None 3
    print(kv.rollback(), kv.get("a", 20))           # True 2
    print(kv.commit(), kv.get("a", 20), kv.get("b", 20), kv.commit(), kv.rollback())  # True 2 3 False False

    kv.begin()
    kv.delete("a")
    print(kv.get("a", 21), kv.base["a"])            # None (2, None)
    print(kv.commit(), kv.get("a", 21), "a" in kv.base)  # True None False

    kv.set("s1", 1, now=0, ttl=5)
    kv.set("s2", 2, now=0, ttl=50)
    kv.set("s3", 3, now=0)
    print(kv.cleanup(10), sorted(kv.base))          # 1 ['b', 's2', 's3']
```

### Extend it: Replicate changes to a follower.

1. `on_change(fn)`: call `fn(key, value)` whenever `base` really changes. For a delete, `value` is `None`.
2. Writes inside a transaction fire **only when they're committed** into `base`.
3. A rollback, or deleting a missing key, fires nothing.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    kv = KVStore()
    events = []
    kv.on_change(lambda k, v: events.append((k, v)))
    kv.set("a", 1, now=0)
    kv.begin()
    kv.set("b", 2, now=0)
    kv.delete("a")
    print(events)                                   # [('a', 1)]
    kv.commit()
    print(events)                                   # [('a', 1), ('b', 2), ('a', None)]
    kv.begin(); kv.set("c", 3, now=0); kv.rollback()
    kv.delete("zzz")
    print(len(events))                              # 3
```

---

## 22 · Unique ID Generator (Snowflake)

**Moves:** — · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The system should generate unique IDs across multiple machines without them talking to each other.
  2. The IDs should be roughly ordered by the time they were created.
  3. Each ID should combine a timestamp, a machine identifier and a sequence number.
  4. A machine should be able to generate many IDs within the same millisecond, and wait for the next millisecond when its sequence runs out.
  5. The generator should be safe to use from multiple threads.
  6. The generator should handle the clock moving backwards.
- **Idea:** 64 bits = **time (high) | machine id | sequence**. Different machine ids mean no clash between machines. Within a millisecond the sequence differs. A lock makes threads take turns. Time in the high bits means roughly sortable.
- **Classes:** `IdParts`, `Clock`, `SystemClock`, `SnowflakeGenerator`
- **Pattern:** none named · **Moves:** —
- **Hard part:** Layout 41 + 10 + 12 bits. Sequence overflow → wait for the next ms. Clock goes backwards: small jump → wait, big jump → refuse.
- **Follow-up:** *Strictly ordered across machines?* No, only roughly (clocks differ).

</details>

### Problem

1. `FakeClock(ms)` has `now()` and `set(ms)`. `wait_until(ms)` jumps forward if the clock is behind `ms`, and always adds 1 to `waits`.
2. `Snowflake(machine_id, clock, seq_bits=12)` raises `ValueError` unless `0 <= machine_id <= 1023`. The id layout is `(t << (10 + seq_bits)) | (machine_id << seq_bits) | seq`.
3. `next_id()`
   - Same millisecond as the last id: `seq + 1`. If that overflows, call `wait_until(last + 1)` and use that millisecond with seq 0.
   - New millisecond: seq 0.
   - The clock is behind the last id by `<= 5 ms`: call `wait_until(last)` and continue as that millisecond.
   - It's behind by more: raise `ClockMovedBackwardsError(f"{diff} ms")`.
4. `decode(id) -> (t, machine_id, seq)`.

### Names the driver uses

`FakeClock`: `now`, `set`, `waits` · `Snowflake`: `next_id`, `decode` · `ClockMovedBackwardsError`

### Driver code

```python
if __name__ == "__main__":
    try:
        Snowflake(1024, FakeClock(0))
    except ValueError:
        print("ValueError")                         # ValueError
    clock = FakeClock(1000)
    g = Snowflake(machine_id=7, clock=clock, seq_bits=2)
    ids = [g.next_id() for _ in range(5)]
    print([g.decode(i) for i in ids])               # [(1000, 7, 0), (1000, 7, 1), (1000, 7, 2), (1000, 7, 3), (1001, 7, 0)]
    print(clock.waits, clock.now())                 # 1 1001
    print(ids == sorted(ids), len(set(ids)))        # True 5
    clock.set(1005)
    print(g.decode(g.next_id()))                    # (1005, 7, 0)
    clock.set(1002)
    print(g.decode(g.next_id()), clock.waits)       # (1005, 7, 1) 2
    clock.set(990)
    try:
        g.next_id()
    except ClockMovedBackwardsError as e:
        print("ClockMovedBackwardsError", e)        # ClockMovedBackwardsError 15 ms
    big = Snowflake(1, FakeClock(1_700_000_000_000))
    print(big.decode(big.next_id()), big.next_id().bit_length() <= 63)  # (1700000000000, 1, 0) True
```

### Extend it: Fewer machines, more ids per millisecond.

`Snowflake(..., seq_bits=12, machine_bits=10)`. The machine id must fit in `machine_bits`, otherwise raise `ValueError`. The layout and `decode` use `machine_bits` instead of a hard-coded 10.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    try:
        Snowflake(40, FakeClock(0), seq_bits=4, machine_bits=5)
    except ValueError:
        print("ValueError")                         # ValueError
    g = Snowflake(31, FakeClock(1000), seq_bits=4, machine_bits=5)
    i = g.next_id()
    print(g.decode(i), i == (1000 << 9) | (31 << 4))  # (1000, 31, 0) True
    print(Snowflake(1023, FakeClock(5)).decode(Snowflake(1023, FakeClock(5)).next_id()))  # (5, 1023, 0)
```

---

## 23 · Web Crawler

**Moves:** M7 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The crawler should start from a set of seed URLs and fetch their pages.
  2. The crawler should extract links from each page and add them to the list of pages to visit.
  3. The crawler should never visit the same URL twice.
  4. The crawler should be able to limit how deep it goes and which domains it visits.
  5. Multiple threads should crawl at the same time.
  6. The crawler should limit how often it hits the same domain and follow the site's crawling rules.
  7. It should be easy to add different ways of processing the fetched content.
  8. Failed fetches should be retried and should time out.
- **Idea:** Normalize the URL and **claim it in `seen` before queueing**, so each URL is fetched at most once. The frontier is grouped **by site** with a next-allowed time (politeness). Workers loop: take → robots check → fetch → processors → queue links.
- **Classes:** `UrlNormalizer`, `CrawlTask`, `Frontier`, `PageFetcher`→`HttpFetcher`, `LinkExtractor`, `RobotsRules`, `RobotsCache`, `CrawlPolicy`, `RetryPolicy`, `ContentProcessor`→`Archive/WordCount`, `Crawler`, `CrawlerWorkerPool`
- **Pattern:** Strategy (processors), queue + workers · **Moves:** M7
- **Hard part:** Per-site queues, so a slow site doesn't block the others. Robots rules are cached per site.
- **Follow-up:** *Billions of pages?* Shard by domain across machines. `seen` becomes a shared store or Bloom filter, and the frontier a durable queue.

</details>

### Problem

No threads here. The URL format is `"site/path"`, and the site is the part before the first `/`.

1. `Crawler(fetch, robots, max_pages=100)`
   - `fetch(url)` returns the list of links on that page.
   - `robots(site)` returns the list of disallowed path prefixes. **Cache it per site**, and count the calls in `robots_calls`.
2. `crawl(seeds) -> visited urls, in order`
   - Keep one queue per site.
   - Serve the sites **round-robin** in the order they were first seen, so a big site can't starve a small one.
   - Skip URLs already seen in this crawl, and URLs whose path starts with a disallowed prefix.
   - Stop at `max_pages`.

### Names the driver uses

`Crawler`: `crawl`, `robots_calls`

### Driver code

```python
if __name__ == "__main__":
    web = {
        "big.com/": ["big.com/1", "big.com/2", "big.com/3", "small.org/", "big.com/admin/x"],
        "big.com/1": ["big.com/2", "big.com/"],
        "big.com/2": [], "big.com/3": [],
        "small.org/": ["small.org/a"], "small.org/a": [],
    }
    robots = {"big.com": ["/admin"], "small.org": []}
    c = Crawler(fetch=lambda u: web.get(u, []), robots=lambda site: robots[site])
    print(c.crawl(["big.com/"]))                    # ['big.com/', 'small.org/', 'big.com/1', 'small.org/a', 'big.com/2', 'big.com/3']
    print(c.robots_calls)                           # 2
    print(Crawler(lambda u: web.get(u, []), lambda s: robots[s], max_pages=2).crawl(["big.com/"]))  # ['big.com/', 'small.org/']
    print(c.crawl(["big.com/admin/x", "small.org/a"]), c.robots_calls)  # ['small.org/a'] 2
```

### Extend it: Do something with each page.

1. `Crawler(..., processors=())`. Each processor's `process(url, links)` is called once per visited page.
2. `LinkCounter` keeps a `total` of links seen. `PagesPerSite` keeps `counts` per site.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    web = {"a.com/": ["a.com/1", "b.com/"], "a.com/1": ["a.com/"], "b.com/": ["b.com/x", "b.com/y"]}
    links, pages = LinkCounter(), PagesPerSite()
    c = Crawler(lambda u: web.get(u, []), lambda s: [], processors=[links, pages])
    print(c.crawl(["a.com/"]))                      # ['a.com/', 'b.com/', 'a.com/1', 'b.com/x', 'b.com/y']
    print(links.total, pages.counts)                # 5 {'a.com': 2, 'b.com': 3}
```

---

## 24 · Meeting Room Booking

**Moves:** M1 M2 M3 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. The building should have multiple meeting rooms, each with a capacity and a set of facilities.
  2. Users should be able to search for rooms available for a time slot, capacity and facilities.
  3. Users should be able to book a room for a time interval.
  4. Two bookings for the same room must never overlap.
  5. Users should be able to cancel or change a booking.
  6. Attendees should be invited and notified about the meeting.
  7. Users should be able to view the schedule of a room and of a person.
  8. If two users try to book the same slot at the same time, only one should succeed.
- **Idea:** A room owns a calendar of reserved time slots, guarded by a lock, so "check free, then reserve" is one step. A Booking holds the meeting details and touches rooms in a fixed order.
- **Classes:** `TimeSlot`, `Room`, `RoomInfo`, `Booking`, `BookingStatus`, `MeetingNotifier`, `MeetingService`
- **Pattern:** none named · **Moves:** M1 M2 M3
- **Hard part:** Overlap check: `a.start < b.end and b.start < a.end`. A change **reserves the new slot before releasing the old one**. A person's schedule is computed.
- **Follow-up:** *Slow overlap check?* Keep the slots sorted (or use an interval tree) and check only the neighbours.

</details>

### Problem

1. `overlaps(a_start, a_end, b_start, b_end)` is `a_start < b_end and b_start < a_end`, so back-to-back meetings don't overlap.
2. `Room(room_id)`
   - `try_book(mid, start, end, people) -> bool` and `try_move(mid, start, end) -> bool` each check for overlaps with the room's other meetings and then change them, all under one lock.
   - `cancel(mid)`.
3. `MeetingScheduler(rooms)`
   - `person_schedule(person)` is **computed** from every room's bookings: a sorted list of `(start, end, mid)`.
   - `book(people, start, end) -> mid | None` returns `None` if `start >= end` or any of the people is busy. Otherwise it takes id `"M1"`, `"M2"`, … and books the first room that accepts. If none accepts, the id is still used up.
4. `reschedule(mid, start, end) -> bool`
   - The people must be free, ignoring this meeting itself.
   - First try `try_move` in the same room.
   - Otherwise **book the new room first, and only then cancel the old one**.
5. `room_of(mid)`.

### Names the driver uses

`overlaps`, `Room`, `MeetingScheduler`: `book`, `reschedule`, `person_schedule`, `room_of`

### Driver code

```python
if __name__ == "__main__":
    print(overlaps(9, 10, 10, 11), overlaps(9, 11, 10, 12), overlaps(10, 12, 9, 13))  # False True True
    s = MeetingScheduler([Room("R1"), Room("R2")])
    m1 = s.book(["ann", "bob"], 9, 10)
    m2 = s.book(["cat"], 9, 10)
    print(m1, s.room_of(m1), m2, s.room_of(m2))     # M1 R1 M2 R2
    print(s.book(["dan"], 9, 10))                   # None
    print(s.book(["bob"], 9, 11), s.book(["dan"], 11, 10))  # None None
    m3 = s.book(["bob"], 10, 11)
    print(m3 is not None, s.room_of(m3))            # True R1
    print(s.person_schedule("bob"))                 # [(9, 10, 'M1'), (10, 11, 'M4')]
    print(s.reschedule(m1, 9, 10), s.reschedule(m1, 9.5, 10.5))  # True False
    print(s.reschedule(m1, 8.5, 9.5), s.person_schedule("ann"))  # True [(8.5, 9.5, 'M1')]
    print(s.book(["eve"], 12, 13) is not None, s.book(["fay"], 12, 14) is not None)  # True True
    print(s.reschedule(m2, 13, 14), s.room_of(m2), s.person_schedule("cat"))  # True R1 [(13, 14, 'M2')]
    print(s.reschedule(m2, 12.5, 13.5))             # False
```

### Extend it: Pick the smallest room that fits.

1. `Room(room_id, capacity=10)`. `MeetingScheduler(rooms, choose=None)`, where `choose(rooms, people) -> rooms to try, in order`.
2. `in_order` is the default: the given order, only rooms with `capacity >= len(people)`.
3. `smallest_fit` takes the fitting rooms and sorts them by `(capacity, room_id)`.
4. `book` and the cross-room part of `reschedule` both use `choose`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    rooms = lambda: [Room("Big", 10), Room("Small", 3), Room("Tiny", 2)]
    default = MeetingScheduler(rooms())
    m = default.book(["a", "b"], 9, 10)
    print(default.room_of(m))                       # Big
    smart = MeetingScheduler(rooms(), choose=smallest_fit)
    m1 = smart.book(["a", "b"], 9, 10)
    m2 = smart.book(["c", "d", "e"], 9, 10)
    m3 = smart.book(["f", "g"], 9, 10)
    print(smart.room_of(m1), smart.room_of(m2), smart.room_of(m3))  # Tiny Small Big
    print(smart.book(list("hijklmnopqr"), 11, 12))  # None
```

---

## 25 · Cab Booking (scheduled)

**Moves:** M1 M3 M10 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. Users should be able to book a cab with pickup and drop-off locations, now or for a later time.
  2. Cabs should have a type and an availability status.
  3. The system should allocate the nearest available cab using a rule that can be replaced.
  4. A cab must never be booked for overlapping trips.
  5. The fare should be calculated from distance and cab type.
  6. Users should be able to cancel a booking.
  7. A trip should move through the stages of booked, started, completed and cancelled.
- **Idea:** Trips can be booked for later, so a cab has a **calendar of time windows**, not a busy flag. Each cab locks its own calendar. Allocation is swappable, and the Trip runs its own stages.
- **Classes:** `Location`, `TimeWindow`, `RateCard`, `FareCalculator`, `Cab`, `CabInfo`, `AllocationStrategy`→`NearestCabStrategy`, `Trip`, `CabService`
- **Pattern:** Strategy (allocation) · **Moves:** M1 M3 M10
- **Hard part:** Window overlap + buffer. The strategy sees copies, and the reservation may refuse → try the next cab. The fare is fixed at booking.
- **Follow-up:** *Location for a future trip?* A new strategy using the previous trip's drop-off.

</details>

### Problem

1. `BUFFER = 10`. `clash(a_start, a_end, b_start, b_end)` is `a_start < b_end + BUFFER and b_start < a_end + BUFFER`.
2. `Cab(cab_id, location)` has a `trips` list of `(start, end)`.
   - `try_reserve(start, end) -> bool` checks for clashes and appends the trip, under a lock.
   - `release(start, end)` removes it.
   - `info()` returns a frozen `CabInfo(cab_id, location, trips tuple)` with `free(start, end)`.
3. `CabService(cabs, rate=10)`. `rate` is public.
   - `book(pickup, start, end, distance, infos=None) -> Trip | None` uses the snapshots (given, or fresh).
   - It sorts the cabs that are free in the snapshot by `(abs(location − pickup), cab_id)` and calls `try_reserve` on the real cab, moving to the next one on refusal.
   - Trip ids are `"T1"`, `"T2"`, …
4. A `Trip` has `trip_id`, `cab_id`, `start`, `end` and `fare = distance × rate`, **fixed at booking**. It runs its own lifecycle (M3):
   - `begin()`: `BOOKED → STARTED`
   - `complete()`: `STARTED → COMPLETED`
   - `cancel()`: only from `BOOKED`, sets `CANCELLED` **and gives the time window back to the cab**
   - Any other move returns `False`.

### Names the driver uses

`clash`, `Cab`: `try_reserve`, `info` · `CabService`: `book`, `rate` · `Trip`: `trip_id`, `cab_id`, `fare`, `status`, `begin`, `complete`, `cancel`

### Driver code

```python
if __name__ == "__main__":
    print(clash(0, 60, 70, 100), clash(0, 60, 69, 100), clash(80, 90, 0, 60))  # False True False
    c1, c2 = Cab("C1", 0), Cab("C2", 50)
    svc = CabService([c1, c2], rate=10)
    t1 = svc.book(pickup=5, start=100, end=160, distance=12)
    print(t1.trip_id, t1.cab_id, t1.fare, t1.status)  # T1 C1 120 BOOKED
    t2 = svc.book(pickup=5, start=165, end=200, distance=3)
    print(t2.cab_id)                                # C2
    t3 = svc.book(pickup=5, start=170, end=200, distance=3)
    print(t3.cab_id)                                # C1
    print(svc.book(pickup=5, start=150, end=180, distance=1))  # None
    stale = [c1.info(), c2.info()]
    print(c1.try_reserve(300, 400))                 # True
    t4 = svc.book(pickup=0, start=300, end=350, distance=5, infos=stale)
    print(t4.cab_id)                                # C2
    svc.rate = 99
    print(t1.fare, svc.book(pickup=0, start=1000, end=1010, distance=1).fare)  # 120 99
    print(c1.info().trips)                          # ((100, 160), (170, 200), (300, 400), (1000, 1010))

    print(t1.complete(), t1.begin(), t1.begin(), t1.cancel(), t1.complete(), t1.status)  # False True False False True COMPLETED
    print(t3.cancel(), t3.status, t3.cancel(), t3.begin())  # True CANCELLED False False
    print(c1.info().trips)                          # ((100, 160), (300, 400), (1000, 1010))
    print(svc.book(pickup=0, start=170, end=200, distance=1).cab_id)  # C1
```

### Extend it: Spread trips evenly across cabs.

1. `CabService(cabs, rate=10, strategy=None)`. `strategy.rank(free_infos, pickup)` returns the snapshots in the order to try them.
2. `Nearest()` is the default. `LeastBusy()` sorts by `(number of trips, cab_id)`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    c1, c2 = Cab("C1", 0), Cab("C2", 100)
    near = CabService([c1, c2])
    print([near.book(pickup=0, start=s, end=s + 10, distance=1).cab_id for s in (0, 50, 100)])  # ['C1', 'C1', 'C1']
    d1, d2 = Cab("C1", 0), Cab("C2", 100)
    fair = CabService([d1, d2], strategy=LeastBusy())
    print([fair.book(pickup=0, start=s, end=s + 10, distance=1).cab_id for s in (0, 50, 100)])  # ['C1', 'C2', 'C1']
```

---

## 26 · Online Code Judge

**Moves:** M2 M7 · **Difficulty:** Medium · **Time:** ~25 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. Users should be able to browse problems with a statement, difficulty, tags and test cases.
  2. Users should be able to submit a solution in a language of their choice.
  3. The system should compile and run each submission against the test cases within time and memory limits.
  4. Each submission should receive a verdict such as accepted, wrong answer, time limit exceeded, runtime error or compile error.
  5. It should be easy to add support for new languages.
  6. Submissions should be queued and processed in the background by multiple workers.
  7. Each submission should run in isolation from the others.
  8. Users should be able to view their submission history.
  9. Contests should rank participants on a leaderboard by score and time.
- **Idea:** Submit = record + enqueue, and answer at once. A worker creates a **fresh sandbox per submission**: compile → run every test → destroy. A language is **data** (file name, compile command, run command). The leaderboard is computed.
- **Classes:** `Problem`, `TestCase`, `ProblemCatalog`, `LanguageSpec`, `Sandbox`/`SandboxFactory`, `ExecResult`, `Judge`, `JudgeResult`, `Submission`, `SubmissionQueue`, `Contest`, `LeaderboardRow`, `JudgeService`, `JudgeWorkerPool`
- **Pattern:** queue + workers · **Moves:** M2 M7
- **Hard part:** The sandbox is always destroyed (try/finally). Verdicts: AC / WA / TLE / MLE / RE / CE.
- **Follow-up:** *Rejudge?* Reset to QUEUED and re-enqueue; the leaderboard updates itself (computed).

</details>

### Problem

1. `Sandbox()`: the class counter `Sandbox.live` counts sandboxes created and not yet destroyed. `destroy()`.
2. `judge(code, tests, run, time_limit=2, memory_limit=256) -> verdict`
   - Create one sandbox and **always destroy it** (`try/finally`).
   - Run each test with `run(code, input, box)`, which returns `(output, seconds, mb)`.
   - Stop at the first bad test: a `SyntaxError` is `CE`, any other exception is `RE`, then too slow is `TLE`, then too much memory is `MLE`, then a wrong output is `WA`.
   - If every test passes, return `AC`.
3. `Judge(tests_by_problem, run)` is the **fast half / slow half** (M7).
   - `submit(user, problem, code, at) -> "S1", "S2", …` only records the submission as `PENDING` and puts it on a FIFO queue. It does **not** judge.
   - `process(max_n=None) -> int` is the worker. It judges up to `max_n` queued submissions in order (all of them if `None`) and returns how many it judged.
   - `verdict(sid)` returns the verdict, or `PENDING`.
4. `leaderboard()` is **computed from the submissions**. It's a list of `(user, problems_solved)`, sorted by most solved, then the lowest sum of first-AC times, then the name. Pending submissions count for nothing yet, and users with no AC still appear.

### Names the driver uses

`Sandbox.live`, `judge`, `Judge`: `submit`, `process`, `verdict`, `leaderboard`

### Driver code

```python
def fake_run(code, inp, box):
    if code == "syntax error":
        raise SyntaxError("bad")
    if code == "crash":
        raise ZeroDivisionError
    if code == "slow":
        return inp * 2, 5, 10
    if code == "hungry":
        return inp * 2, 0.1, 999
    if code == "double":
        return inp * 2, 0.1, 10
    return inp + 1, 0.1, 10

if __name__ == "__main__":
    tests = [(1, 2), (5, 10)]
    print([judge(c, tests, fake_run) for c in ["double", "plus1", "slow", "hungry", "crash", "syntax error"]])  # ['AC', 'WA', 'TLE', 'MLE', 'RE', 'CE']
    print(Sandbox.live)                             # 0

    j = Judge({"P1": tests, "P2": [(3, 4)]}, fake_run)
    s1 = j.submit("ann", "P1", "plus1", at=5)
    s2 = j.submit("ann", "P1", "double", at=10)
    s3 = j.submit("ann", "P2", "plus1", at=20)
    print(s1, s2, s3, j.verdict(s1), j.leaderboard())  # S1 S2 S3 PENDING [('ann', 0)]
    print(j.process(max_n=2), j.verdict(s1), j.verdict(s2), j.verdict(s3))  # 2 WA AC PENDING
    s4 = j.submit("bob", "P1", "double", at=8)
    s5 = j.submit("bob", "P2", "plus1", at=9)
    s6 = j.submit("cat", "P1", "crash", at=1)
    print(j.leaderboard())                          # [('ann', 1), ('bob', 0), ('cat', 0)]
    print(j.process(), j.process(), [j.verdict(s) for s in [s3, s4, s5, s6]])  # 4 0 ['AC', 'AC', 'AC', 'RE']
    print(j.leaderboard())                          # [('bob', 2), ('ann', 2), ('cat', 0)]
```

### Extend it: Hard problems are worth more points.

1. `Judge(tests, run, scoring=None)`, where `scoring.score(solved_problems) -> number`.
2. `SolvedCount()` is the default (`len`). `Points({"easy": 100, ...})` returns the sum of the points.
3. `leaderboard` returns `(user, score)`, sorted the same way as before.

### Extend-it driver (the driver above must still pass too)

```python
def fake_run(code, inp, box):
    return (inp * 2 if code == "double" else inp + 1), 0.1, 10

if __name__ == "__main__":
    tests = {"easy": [(1, 2)], "hard": [(2, 4)]}
    j = Judge(tests, fake_run, scoring=Points({"easy": 100, "hard": 500}))
    j.submit("ann", "easy", "double", at=1)
    j.submit("ann", "hard", "plus1", at=2)
    j.submit("bob", "hard", "double", at=3)
    j.process()
    print(j.leaderboard())                          # [('bob', 500), ('ann', 100)]
    plain = Judge(tests, fake_run)
    plain.submit("ann", "easy", "double", at=1)
    plain.process()
    print(plain.leaderboard())                      # [('ann', 1)]
```

---

## 27 · Digital Wallet

**Moves:** M1 M8 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. Users should have a wallet with a balance.
  2. Users should be able to add money from a bank account or card.
  3. Users should be able to transfer money to other wallets and pay merchants.
  4. Users should be able to view their transaction history with the status of each transaction.
  5. The system should support multiple payment methods and gateways that can be swapped without changing the rest of the system.
  6. A transfer should either debit and credit both wallets or change neither.
  7. The system should reject a payment when the balance is not enough.
  8. Balances should stay correct when many transactions happen at the same time.
  9. Failed payments should be refunded or retried.
  10. A repeated request should not be processed twice.
- **Idea:** Only the wallet changes its balance, under its own lock. A transfer locks **both wallets in id order**. Every request has an **idempotency key claimed first**. Gateways sit behind one interface. If a payout fails after a debit, retry or refund, and record it either way.
- **Classes:** `Wallet`, `PaymentMethod`→`Card/BankAccount`, `PaymentGateway`→`Card/BankGateway`, `GatewayRouter`, `GatewayResult`, `Merchant`, `Transaction`, `WalletService`
- **Pattern:** Strategy (gateways, methods) · **Moves:** M1 M8
- **Hard part:** Deadlock avoided by fixed lock order. A repeated key returns the first result. Hold the money first, pay out second, credit back on failure.
- **Follow-up:** *Charged but the response was lost?* Retry with the same key, so the provider dedupes it. Plus daily reconciliation.

</details>

### Problem

1. `WalletService.open(wallet_id, balance)`, `balance(wallet_id)`.
2. **Every operation takes a `key`.** Claim the key first (one lock per key). If the key already has a result, return it without doing anything.
3. `transfer(src, dst, amount, key)` returns `"OK"`, `"INSUFFICIENT_FUNDS"` or `"INVALID"` (unknown wallet, same wallet, or `amount <= 0`).
   - Lock the two wallets **in `wallet_id` order**, whatever the direction, so it can't deadlock.
   - Check, then change.
4. `withdraw_to_bank(wallet_id, amount, key, payout) -> "OK" | "INSUFFICIENT_FUNDS" | "PAYOUT_FAILED"`
   - **Hold** (debit) the money first.
   - Then call `payout(amount)`.
   - If it fails, **credit it back**.

### Names the driver uses

`WalletService`: `open`, `balance`, `transfer`, `withdraw_to_bank`

### Driver code

```python
import threading

if __name__ == "__main__":
    ws = WalletService()
    ws.open("A", 1000); ws.open("B", 500)
    print(ws.transfer("A", "B", 300, key="k1"), ws.balance("A"), ws.balance("B"))  # OK 700 800
    print(ws.transfer("A", "B", 300, key="k1"), ws.balance("A"), ws.balance("B"))  # OK 700 800
    print(ws.transfer("A", "B", 5000, key="k2"), ws.transfer("A", "A", 1, key="k3"), ws.transfer("A", "Z", 1, key="k4"))  # INSUFFICIENT_FUNDS INVALID INVALID
    print(ws.transfer("A", "B", 0, key="k5"), ws.balance("A"))  # INVALID 700

    def move(src, dst):
        for i in range(500):
            ws.transfer(src, dst, 1, key=f"{src}{dst}{i}")
    threads = [threading.Thread(target=move, args=p) for p in [("A", "B"), ("B", "A")] * 4]
    for t in threads: t.start()
    for t in threads: t.join()
    print(ws.balance("A") + ws.balance("B"))        # 1500

    ws.open("C", 400)
    print(ws.withdraw_to_bank("C", 300, key="w1", payout=lambda amt: False), ws.balance("C"))  # PAYOUT_FAILED 400
    print(ws.withdraw_to_bank("C", 300, key="w2", payout=lambda amt: True), ws.balance("C"))  # OK 100
    print(ws.withdraw_to_bank("C", 300, key="w2", payout=lambda amt: True), ws.balance("C"))  # OK 100
    print(ws.withdraw_to_bank("C", 300, key="w3", payout=lambda amt: True), ws.balance("C"))  # INSUFFICIENT_FUNDS 100
```

### Extend it: If one bank is down, try the next.

1. `FallbackPayout([(name, pay), ...])` is a callable, so it plugs into `payout=` with no change to `WalletService`.
2. It tries each gateway in order and returns `True` at the first success. It records every name it tried in `tried`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    ws = WalletService()
    ws.open("A", 1000)
    payout = FallbackPayout([("bank-1", lambda amt: False), ("bank-2", lambda amt: True), ("bank-3", lambda amt: True)])
    print(ws.withdraw_to_bank("A", 400, key="w1", payout=payout), ws.balance("A"), payout.tried)  # OK 600 ['bank-1', 'bank-2']
    dead = FallbackPayout([("bank-1", lambda amt: False), ("bank-2", lambda amt: False)])
    print(ws.withdraw_to_bank("A", 400, key="w2", payout=dead), ws.balance("A"), dead.tried)  # PAYOUT_FAILED 600 ['bank-1', 'bank-2']
```

---

## 28 · Airline Management

**Moves:** M1 M3 M9 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. Admins should be able to add and manage flights with a source, a destination, departure and arrival times, and an aircraft.
  2. Each aircraft should have a seat layout with seats in different classes such as economy, business and first.
  3. Users should be able to search for flights by source, destination and date.
  4. Users should be able to book a flight by choosing seats and entering passenger details.
  5. Seat prices should differ by class, and the pricing rule should be easy to swap.
  6. A seat must never be booked by two passengers on the same flight, and a chosen seat should be held while payment is in progress.
  7. The system should support multiple payment methods.
  8. A booking should move through the stages of pending, confirmed and cancelled, and cancelling should trigger a refund.
  9. A flight should move through the stages of scheduled, delayed, departed, landed and cancelled, and passengers should be notified at each change.
  10. Crew members should be assigned to flights, and a crew member must never be assigned to overlapping flights.
  11. Passengers should be able to check in and receive a boarding pass.
- **Idea:** Same as BookMyShow. `FlightSeat` per flight per seat, with a lock. Crew members own a calendar of flight windows (like meeting rooms). The flight notifies watching passengers. There are two managers: admin (`AirlineOperations`) and user (`BookingService`).
- **Classes:** `Aircraft`, `Seat`, `FlightSeat`, `Flight`, `FlightObserver`, `PassengerSubscription`, `PricingStrategy`, `PaymentMethod`, `Booking`, `BookingItem`, `BoardingPass`, `CrewMember`, `CrewRoster`, `FlightCatalog`, `AirlineOperations`, `BookingService`
- **Pattern:** Strategy (pricing, payment), Observer (flight changes) · **Moves:** M1 M3 M9
- **Hard part:** The price is fixed when the seat is held. Observers are called **after** releasing the flight lock.
- **Follow-up:** *Aircraft on two flights at once?* Give `Aircraft` a calendar, the same as `CrewMember`.

</details>

### Problem

1. `Aircraft(model, seat_map)` takes `seat_map` as `{seat_id: fare_class}`. Two flights can share one aircraft.
2. `Flight(flight_id, aircraft, prices)` creates **its own `FlightSeat` per seat** (M9).
3. `hold(seat_id, passenger) -> price | None`
   - Return `None` if the seat is unknown or not `AVAILABLE`.
   - The **price is fixed at hold time** and stored on the seat. `set_price(fare_class, price)` changes only later holds.
4. `confirm(seat_id, passenger) -> price | None` works only for the passenger holding the seat. The seat becomes `BOOKED`.
5. `seat_status(seat_id)`.
6. `on_change(fn)` registers `fn(passenger, flight_id, state)`.
7. `set_state(new_state) -> sorted booked passengers`
   - Under the flight lock, set the state and copy the list of booked passengers.
   - **After releasing the lock**, call every listener for every passenger on that list.

### Names the driver uses

`Aircraft`, `Flight`: `hold`, `confirm`, `set_price`, `seat_status`, `on_change`, `set_state`

### Driver code

```python
if __name__ == "__main__":
    a320 = Aircraft("A320", {"1A": "BUSINESS", "1B": "BUSINESS", "20C": "ECONOMY"})
    am, pm = Flight("AI-101", a320, {"BUSINESS": 900, "ECONOMY": 200}), Flight("AI-202", a320, {"BUSINESS": 700, "ECONOMY": 150})
    print(am.hold("1A", "ann"), am.seat_status("1A"), pm.seat_status("1A"))  # 900 HELD AVAILABLE
    print(am.hold("1A", "bob"), am.hold("9Z", "bob"))  # None None
    am.set_price("BUSINESS", 1500)
    print(am.confirm("1A", "bob"), am.confirm("1A", "ann"), am.seat_status("1A"))  # None 900 BOOKED
    print(am.hold("1B", "bob"), am.confirm("1B", "bob"))  # 1500 1500
    print(am.hold("20C", "cat"))                    # 200

    inbox = []
    am.on_change(lambda p, f, st: inbox.append((p, f, st)))
    print(am.set_state("DELAYED"))                  # ['ann', 'bob']
    print(inbox)                                    # [('ann', 'AI-101', 'DELAYED'), ('bob', 'AI-101', 'DELAYED')]
```

### Extend it: Seats get more expensive as the flight fills.

1. `Flight(..., pricing=None)`, where `pricing(fare_class, base, taken) -> price`. `taken` is the number of seats not `AVAILABLE` at hold time. The default returns `base`.
2. `DemandPricing(pct_per_seat)` returns `base × (100 + pct × taken) // 100`.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    jet = Aircraft("A320", {"1A": "ECO", "1B": "ECO", "1C": "ECO"})
    f = Flight("AI-9", jet, {"ECO": 1000}, pricing=DemandPricing(pct_per_seat=10))
    print(f.hold("1A", "ann"), f.hold("1B", "bob"), f.hold("1C", "cat"))  # 1000 1100 1200
    print(Flight("AI-10", jet, {"ECO": 1000}).hold("1A", "dan"))  # 1000
```

---

## 29 · Webhook Delivery

**Moves:** M7 · **Difficulty:** Medium · **Time:** ~20 min

<details><summary><b>Full design card</b> (requirements, idea, classes, pattern, hard part, follow-up)</summary>

- **Requirements:**
  1. Customers should be able to register webhook endpoints (an HTTPS URL) and choose which event types each endpoint receives.
  2. When an event happens in the platform, it should be delivered to every endpoint subscribed to that event type.
  3. Each delivery should be sent as an HTTP POST with the event payload, without blocking the code that raised the event.
  4. Every delivery should be signed so the customer can check that it came from the platform and has not been replayed.
  5. A delivery should count as successful only on a 2xx response within a time limit. Time-outs, 429 and 5xx responses should be retried with growing delays, and other failures should not be retried.
  6. After the last retry fails, a delivery should go to a dead-letter list from which it can be replayed.
  7. Delivery should be at least once, and each event should carry a unique id so customers can ignore duplicates.
  8. One customer's slow or failing endpoint should not delay deliveries to other customers.
  9. An endpoint that keeps failing should be switched off automatically, and the customer should be able to switch it back on.
  10. The system should record every delivery attempt with its status, response code and timing, and let customers view them.
  11. Customers should be able to rotate an endpoint's signing secret without missing any deliveries.
- **Idea:** Publish = create one `Delivery` per subscribed endpoint and enqueue it. Workers sign with the endpoint's secret, sort the answer into success / retry / give up, and log every attempt. Endpoints get separate lanes with a per-endpoint worker limit. An endpoint that keeps failing is auto-disabled.
- **Classes:** `Event`, `Endpoint`, `SecretEntry`, `PayloadSigner` (HMAC), `HttpSender`, `ResponseClassifier`, `RetryPolicy`, `Delivery`, `DeliveryAttempt`, `DeliveryQueue`, `WebhookService`, `WebhookWorkerPool`
- **Pattern:** Pub-Sub (fan-out), queue + workers, Circuit Breaker · **Moves:** M7
- **Hard part:** Round-robin across endpoints so one bad customer can't starve the others. **At-least-once** delivery + event id for receiver dedupe (not exactly-once).
- **Follow-up:** *Crash?* Outbox pattern: write the delivery in the same DB transaction as the business change.

</details>

### Problem

No threads here. `deliver_next()` plays the worker.

1. `WebhookDispatcher(send, max_attempts=3)`. `send(endpoint, event_id, payload)` returns `True` or `False`.
2. `publish(event_id, payload, endpoints)` adds one job to **each endpoint's own lane**. Lanes are kept in the order endpoints were first seen.
3. `deliver_next() -> (endpoint, event_id, ok) | None`
   - Take the next **non-empty lane, round-robin** from where you left off, so one slow endpoint can't starve the others.
   - Send the job at the front of that lane.
   - If it fails with attempts left, put it at the **back of its lane**. Otherwise add `(endpoint, event_id)` to `dead_letters`.
   - Return `None` when every lane is empty.
4. `run_all()` calls `deliver_next` until it returns `None`, and returns the list of results.
5. Delivery is **at least once**, so a receiver can get the same event twice. `Receiver.receive(event_id, payload)` processes each event id only once and records it in `processed`.

### Names the driver uses

`WebhookDispatcher`: `publish`, `deliver_next`, `run_all`, `dead_letters` · `Receiver`: `receive`, `processed`

### Driver code

```python
if __name__ == "__main__":
    shop = Receiver()
    calls = {"shop": 0}
    def send(ep, event_id, payload):
        if ep == "down.io":
            return False
        if ep == "shop.com":
            calls["shop"] += 1
            shop.receive(event_id, payload)
            return calls["shop"] != 2
        return True

    d = WebhookDispatcher(send, max_attempts=3)
    for i in range(1, 4):
        d.publish(f"e{i}", {"n": i}, ["shop.com", "down.io"])
    d.publish("e4", {"n": 4}, ["tiny.dev"])
    print(d.deliver_next(), d.deliver_next(), d.deliver_next())  # ('shop.com', 'e1', True) ('down.io', 'e1', False) ('tiny.dev', 'e4', True)
    print(d.run_all())                              # [('shop.com', 'e2', False), ('down.io', 'e2', False), ('shop.com', 'e3', True), ('down.io', 'e3', False), ('shop.com', 'e2', True), ('down.io', 'e1', False), ('down.io', 'e2', False), ('down.io', 'e3', False), ('down.io', 'e1', False), ('down.io', 'e2', False), ('down.io', 'e3', False)]
    print(d.dead_letters)                           # [('down.io', 'e1'), ('down.io', 'e2'), ('down.io', 'e3')]
    print(shop.processed, calls["shop"])            # ['e1', 'e2', 'e3'] 4
    print(d.deliver_next())                         # None
```

### Extend it: Endpoints subscribe to event types.

1. `subscribe(endpoint, event_types)`.
2. `publish_event(event_id, event_type, payload)` publishes to every endpoint subscribed to that type, in the order they subscribed, and returns that list.
3. `publish` still works for direct sends.

### Extend-it driver (the driver above must still pass too)

```python
if __name__ == "__main__":
    d = WebhookDispatcher(lambda ep, eid, p: True)
    d.subscribe("billing.io", ["order.paid", "order.refunded"])
    d.subscribe("crm.io", ["order.created", "order.paid"])
    print(d.publish_event("e1", "order.created", {}), d.publish_event("e2", "order.paid", {}), d.publish_event("e3", "user.deleted", {}))  # ['crm.io'] ['billing.io', 'crm.io'] []
    print(d.run_all())                              # [('crm.io', 'e1', True), ('billing.io', 'e2', True), ('crm.io', 'e2', True)]
```
