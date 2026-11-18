# Design Pattern Practice: 45 Problems

Every place a design pattern is used in [00_a_recall.md](../template/00_a_recall.md) and [00_deep_dive.md](../template/00_deep_dive.md) is turned into one interview-size problem: **40 problems**. On top of that come **5 adapted problems** for the patterns in `template/design_patterns/` that the docs never name (Builder, Prototype, Decorator, Facade, Adapter). Class, enum and method names follow the deep_dive cards wherever they fit.

### How to use

1. Read the problem, design the classes, and write your solution in a new `.py` file.
2. Paste the **driver code** under your solution and run it.
3. Compare what it prints with the comments:
   - `print(x)   # True` means that line must print `True`.
   - A comment-only line under a `print` (lined up with the comments) is the next expected output line.
   - Text after `->` is only an explanation. It is not printed.
4. Names listed under **Names the driver uses** are fixed. Everything else (helper classes, private methods, internal state) is your design.

Every driver here was run against a reference solution, and all the expected output is real output.

### Index

| # | Problem | Pattern | Source card |
|---|---|---|---|
| | **Part 1: Strategy** | | |
| P01 | Parking Lot | Strategy ×2 (pricing, payment) | #01 |
| P02 | BookMyShow | Strategy (payment) | #02 |
| P03 | Splitwise | Strategy (split) | #03 |
| P04 | Elevator Dispatch | Strategy (scheduling) | #04 |
| P05 | Rate Limiter Algorithms | Strategy (algorithm) | #06 |
| P06 | LRU / LFU Cache | Strategy (eviction) | #07 |
| P07 | Ride Sharing | Strategy ×3 (matching, pricing, payment) | #08 |
| P08 | Logging Formats & Destinations | Strategy (formatter, destination) | #09 |
| P09 | Snake and Ladder | Strategy (dice) | #10 |
| P10 | Notification Channels | Strategy (channel) | #13 |
| P11 | Food Delivery Coupons | Strategy (discount, payment) | #14 |
| P12 | Library Search | Strategy (search criterion) | #15 |
| P13 | Task Scheduler Timing | Strategy (schedule) | #17 |
| P14 | File Search | Strategy (search criterion) | #20 |
| P15 | Web Crawler Processing | Strategy (content processor) | #23 |
| P16 | Cab Booking (scheduled) | Strategy (allocation) | #25 |
| P17 | Wallet Top-up Gateways | Strategy (gateway) | #27 |
| P18 | Airline Seat Pricing | Strategy (pricing, payment) | #28 |
| | **Part 2: State** | | |
| P19 | Vending Machine | State | #05 |
| P20 | ATM | State | #12 |
| P21 | Elevator Car | State | #04 |
| | **Part 3: Observer** | | |
| P22 | Event Fan-out for Notifications | Observer | #13 |
| P23 | Food Order Tracking | Observer | #14 |
| P24 | Library Reservations | Observer | #15 |
| P25 | Flight Status Alerts | Observer | #28 |
| | **Part 4: Chain of Responsibility** | | |
| P26 | Log Level Handlers | Chain of Responsibility | #09 |
| P27 | ATM Note Dispenser | Chain of Responsibility | #12 |
| | **Part 5: Composite** | | |
| P28 | In-Memory File System | Composite | #20 |
| | **Part 6: Command** | | |
| P29 | Task Scheduler Jobs | Command | #17 |
| | **Part 7: Singleton** | | |
| P30 | Log Manager | Singleton | #09 |
| | **Part 8: Factory** | | |
| P31 | Per-Client Rate Limiter | Factory | #06 |
| | **Part 9: Queue + Worker Pool** | | |
| P32 | Async Notification Delivery | Queue + Worker | #13 |
| P33 | Scheduler Worker Pool | Queue + Worker Pool (threads) | #17 |
| P34 | Multi-threaded Web Crawler | Queue + Worker Pool (threads) | #23 |
| P35 | Online Code Judge | Queue + Worker Pool (threads) | #26 |
| P36 | Webhook Retries & Dead Letters | Queue + Worker | #29 |
| | **Part 10: Producer–Consumer / Pub-Sub** | | |
| P37 | Bounded Queue + Pub-Sub Broker | Producer–Consumer, Pub-Sub | #19 |
| P38 | Webhook Fan-out with Fair Lanes | Pub-Sub fan-out | #29 |
| | **Part 11: Circuit Breaker** | | |
| P39 | Failing Endpoint Breaker | Circuit Breaker | #29 |
| | **Part 12: Polymorphism via inheritance** | | |
| P40 | Chess Move Generation | Polymorphism | #16 |
| | **Part 13: Adapted (not named in the docs)** | | |
| P41 | Flight Setup | Builder | from #28 |
| P42 | Show Seat Layouts | Prototype | from #02 |
| P43 | Stackable Log Destinations | Decorator | from #09 |
| P44 | One-Call Ticket Booking | Facade | from #02 |
| P45 | Third-party Payment SDKs | Adapter | from #27 |


---

## P01 · Parking Lot — Strategy (pricing + payment)

**Pattern:** Strategy ×2 · **Difficulty:** Easy · **Time:** ~25 min · **Source:** `#01 · Parking Lot`

### Problem

Design a parking lot with floors and spots. Each spot fits exactly one vehicle type. The **fee rule** and the **payment method** must both be swappable without changing `ParkingLot`.

1. The lot has floors, and each floor has spots of type `MOTORCYCLE`, `CAR` or `TRUCK`. Adding a floor that already exists, or a spot on a floor that does not exist, is refused.
2. A vehicle can park only in a free spot of its own type. Search floors in the order they were added, and spots in the order they were added.
3. Entry gives a ticket with id `T1`, `T2`, … that records the vehicle, the spot and the entry time (minutes, as an `int`).
4. Entry is refused when there is no free spot of that type, or when the same plate number is already parked.
5. Exit takes a ticket id, a payment method and an exit time. The fee comes from the current pricing rule. If payment fails, exit is refused and the vehicle stays parked. An unknown or already-used ticket is refused.
6. **Pricing rules:**
   - Hourly pricing: rate per vehicle type × hours parked, rounded up, minimum 1 hour.
   - Flat pricing: a fixed fee per vehicle type.
   - The lot's pricing rule can be replaced at any time.
7. **Payment methods:**
   - Cash succeeds only if the cash given ≥ the fee.
   - Card always succeeds.
8. Availability shows the number of free spots per vehicle type. It is computed, not stored.
9. **Refused actions** return `False` (for `bool` methods) or `None`.

### Names the driver uses

- `VehicleType` enum: `MOTORCYCLE`, `CAR`, `TRUCK`
- `Vehicle(plate_number, vehicle_type)`
- `HourlyPricing(rates_by_type)`, `FlatPricing(fees_by_type)`
- `CashPayment(cash_given)`, `CardPayment(card_number)`
- `ParkingLot(pricing)`: `add_floor`, `add_spot(floor_number, code, vehicle_type)`, `enter(vehicle, entry_time)`, `exit(ticket_id, payment, exit_time)`, `get_availability() -> {VehicleType: int}`, `set_pricing(pricing)`
- Ticket: `get_ticket_id()`, `get_spot().get_code()`
- Receipt: `get_fee()`, `get_spot_code()`

### Driver code

```python
if __name__ == "__main__":
    lot = ParkingLot(HourlyPricing({VehicleType.MOTORCYCLE: 10, VehicleType.CAR: 20, VehicleType.TRUCK: 50}))
    print(lot.add_floor(1))                                      # True
    print(lot.add_floor(1))                                      # False  -> floor exists
    lot.add_spot(1, "F1-C1", VehicleType.CAR)
    lot.add_spot(1, "F1-C2", VehicleType.CAR)
    lot.add_spot(1, "F1-M1", VehicleType.MOTORCYCLE)
    print(lot.add_spot(2, "F2-C1", VehicleType.CAR))             # False  -> no floor 2
    print(lot.get_availability()[VehicleType.CAR])               # 2

    t1 = lot.enter(Vehicle("KA-01", VehicleType.CAR), 0)
    t2 = lot.enter(Vehicle("KA-02", VehicleType.CAR), 30)
    print(t1.get_ticket_id(), t1.get_spot().get_code())          # T1 F1-C1
    print(t2.get_ticket_id(), t2.get_spot().get_code())          # T2 F1-C2
    print(lot.enter(Vehicle("KA-03", VehicleType.CAR), 40))      # None   -> car spots full
    print(lot.enter(Vehicle("TR-01", VehicleType.TRUCK), 40))    # None   -> no truck spots
    print(lot.get_availability()[VehicleType.CAR])               # 0

    print(lot.exit("T1", CashPayment(30), 130))                  # None   -> fee is 60 (3 hours), cash too low
    print(lot.get_availability()[VehicleType.CAR])               # 0      -> still parked
    r1 = lot.exit("T1", CashPayment(100), 130)
    print(r1.get_fee(), r1.get_spot_code())                      # 60 F1-C1
    print(lot.exit("T1", CardPayment("4111"), 140))              # None   -> ticket already used
    print(lot.get_availability()[VehicleType.CAR])               # 1

    t3 = lot.enter(Vehicle("MH-09", VehicleType.MOTORCYCLE), 0)
    print(lot.exit(t3.get_ticket_id(), CardPayment("4111"), 10).get_fee())   # 10     -> minimum 1 hour
    print(lot.enter(Vehicle("KA-02", VehicleType.CAR), 50))      # None   -> KA-02 already parked

    lot.set_pricing(FlatPricing({VehicleType.MOTORCYCLE: 5, VehicleType.CAR: 100, VehicleType.TRUCK: 300}))
    print(lot.exit("T2", CardPayment("4111"), 500).get_fee())    # 100    -> new pricing rule, no other change
```


---

## P02 · BookMyShow — Strategy (payment method)

**Pattern:** Strategy · **Difficulty:** Easy–Medium · **Time:** ~25 min · **Source:** `#02 · BookMyShow`

### Problem

Users hold seats for a show and then pay. The booking must work with **any payment method** that can pay and refund, and it must never know which one it got.

1. A show has seats. Each seat has a type (`REGULAR`, `PREMIUM`), and the show sets a price per seat type.
2. Each seat of a show is `AVAILABLE`, `HELD` or `BOOKED`.
3. Holding seats is **all-or-nothing**. If any requested seat is unknown or not available, release the seats this call already held and return `None`. Booking ids are `B1`, `B2`, ….
4. A new booking is `PENDING`. Its total is the sum of its seat prices.
5. Confirming a pending booking charges the payment method. If the payment fails, the booking stays `PENDING` and the seats stay held. On success the seats become `BOOKED` and the booking becomes `CONFIRMED`.
6. Cancelling a `PENDING` booking releases the seats. Cancelling a `CONFIRMED` booking also **refunds the total to the same payment method** it was paid with. Cancelling twice is refused.
7. **Payment methods:**
   - Card has a limit: a payment succeeds if the amount ≤ the limit. Refund always succeeds.
   - UPI has a balance: a payment deducts from it if there is enough. A refund adds the amount back.
8. **Refused actions** return `False` or `None`.

### Names the driver uses

- `SeatType`: `REGULAR`, `PREMIUM`
- `Show(show_id, prices_by_type)`: `add_seat(seat_number, seat_type)`, `get_show_seat(seat_number).get_status()`
- Status enum names printed: `AVAILABLE`, `HELD`, `BOOKED`, `PENDING`, `CONFIRMED`, `CANCELLED`
- `CardPayment(limit)`, `UpiPayment(balance)` with `get_balance()`
- `BookingService()`: `add_show`, `hold_seats(show_id, seat_numbers, user_id)`, `confirm_booking(booking_id, payment)`, `cancel_booking(booking_id)`
- Booking: `get_booking_id()`, `get_total()`, `get_status()`

### Driver code

```python
if __name__ == "__main__":
    show = Show("S1", {SeatType.REGULAR: 200, SeatType.PREMIUM: 350})
    show.add_seat("A1", SeatType.REGULAR)
    show.add_seat("A2", SeatType.REGULAR)
    show.add_seat("P1", SeatType.PREMIUM)
    svc = BookingService()
    svc.add_show(show)

    b1 = svc.hold_seats("S1", ["A1", "P1"], "u1")
    print(b1.get_booking_id(), b1.get_total())               # B1 550
    print(svc.hold_seats("S1", ["A2", "P1"], "u2"))          # None   -> P1 already held
    print(show.get_show_seat("A2").get_status().name)        # AVAILABLE  -> A2 hold was undone
    print(svc.hold_seats("S1", ["A2", "Z9"], "u2"))          # None   -> unknown seat

    upi = UpiPayment(500)
    print(svc.confirm_booking("B1", upi))                    # False  -> 500 < 550
    print(b1.get_status().name, upi.get_balance())           # PENDING 500
    card = CardPayment(1000)
    print(svc.confirm_booking("B1", card))                   # True
    print(show.get_show_seat("P1").get_status().name)        # BOOKED
    print(svc.confirm_booking("B1", card))                   # False  -> already confirmed

    b2 = svc.hold_seats("S1", ["A2"], "u2")
    upi2 = UpiPayment(300)
    print(svc.confirm_booking(b2.get_booking_id(), upi2))    # True
    print(upi2.get_balance())                                # 100
    print(svc.cancel_booking("B2"))                          # True
    print(upi2.get_balance())                                # 300    -> refunded to the same method
    print(show.get_show_seat("A2").get_status().name)        # AVAILABLE
    print(svc.cancel_booking("B2"))                          # False  -> already cancelled

    b3 = svc.hold_seats("S1", ["A2"], "u3")
    print(svc.cancel_booking(b3.get_booking_id()))           # True   -> pending, nothing to refund
    print(svc.cancel_booking("B9"))                          # False  -> unknown booking
```


---

## P03 · Splitwise — Strategy (split type)

**Pattern:** Strategy · **Difficulty:** Easy–Medium · **Time:** ~25 min · **Source:** `#03 · Splitwise`

### Problem

A group of friends records shared expenses. One person pays, and several people share the cost. **The way the cost is split is a swappable rule**, and adding a new split type must not change `Group`.

1. Money is an `int` (paise or cents). No floats.
2. **Split rules:**
   - **Equal** among a given list of users. Each gets `total // n`, and the leftover units go one each to the first users in the list.
   - **Exact** amounts per user. Refused if they don't add up to the total.
   - **Percent** per user. Refused if the percentages don't add up to 100. Each share is `total * pct // 100`, and the leftover units go one each to the first users (in the order given).
3. Adding an expense is refused if the amount ≤ 0, the payer or any person sharing is not a group member, or the split rule refuses.
4. **Store only what happened** (expenses and settlements). Every balance is computed from that history, never stored.
5. `get_balance_between(a, b)`: positive means **b owes a**, and negative means a owes b.
6. `get_net_balances()`: for each member, positive means they are owed money and negative means they owe.
7. `settle(from, to, amount)` records a payment from one member to another. It is refused for the same person, a non-member, or an amount ≤ 0.

### Names the driver uses

- `EqualSplit(user_ids)`, `ExactSplit({user: amount})`, `PercentSplit({user: percent})`
- `Group(group_id, member_ids)`: `add_expense(payer_id, amount, strategy) -> Expense or None`, `settle`, `get_balance_between`, `get_net_balances() -> {user: int}` (in member order)
- Expense: `get_shares() -> {user: int}`

### Driver code

```python
if __name__ == "__main__":
    g = Group("trip", ["A", "B", "C"])

    e1 = g.add_expense("A", 100, EqualSplit(["A", "B", "C"]))
    print(e1.get_shares())                                          # {'A': 34, 'B': 33, 'C': 33}
    print(g.get_balance_between("A", "B"))                          # 33     -> B owes A 33

    print(g.add_expense("B", 90, ExactSplit({"A": 50, "C": 30})))   # None   -> 80 != 90
    e2 = g.add_expense("B", 90, ExactSplit({"A": 60, "C": 30}))
    print(e2.get_shares())                                          # {'A': 60, 'C': 30}
    print(g.get_balance_between("A", "B"))                          # -27    -> now A owes B 27

    print(g.add_expense("C", 50, PercentSplit({"A": 50, "B": 40}))) # None   -> 90 percent
    e3 = g.add_expense("C", 99, PercentSplit({"A": 50, "B": 25, "C": 25}))
    print(e3.get_shares())                                          # {'A': 50, 'B': 25, 'C': 24}
    print(g.add_expense("D", 10, EqualSplit(["A", "B"])))           # None   -> D not a member

    print(g.get_net_balances())                                     # {'A': -44, 'B': 32, 'C': 12}
    print(g.settle("A", "B", 32))                                   # True
    print(g.get_net_balances())                                     # {'A': -12, 'B': 0, 'C': 12}
    print(g.get_balance_between("C", "A"))                          # 17     -> A owes C 17
    print(g.settle("A", "A", 5))                                    # False  -> same person
```


---

## P04 · Elevator Dispatch — Strategy (scheduling rule)

**Pattern:** Strategy (decides on read-only copies, M10) · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#04 · Elevator`

### Problem

A building has several elevators. When someone presses a call button (a floor plus a direction), the controller picks an elevator **using a scheduling rule that can be replaced**. The rule sees only **read-only snapshots** of the elevators, never the elevator objects themselves.

1. Each elevator has an id, a current floor and an ordered list of pending stops. Adding a floor that is already a stop does nothing (still accepted).
2. An elevator's state is **computed**, never stored:
   - `OUT_OF_SERVICE` if it was taken out of service.
   - Otherwise `IDLE` if it has no stops.
   - Otherwise `MOVING_UP` if its **oldest** pending stop is above its current floor, else `MOVING_DOWN`.
3. The controller builds snapshots of the in-service elevators only, asks the rule to choose, then adds the call floor as a stop on the chosen elevator and returns its id. If there are no candidates it returns `None`.
4. **Nearest rule:**
   - An elevator is *suitable* if it is idle, or moving up with the call floor ≥ its floor and the call direction `UP`, or moving down with the call floor ≤ its floor and the call direction `DOWN`.
   - Pick the suitable one with the smallest distance. Ties go to fewer stops, then the smaller id.
   - If none is suitable, pick the elevator with the fewest stops. Ties go to the smaller distance, then the smaller id.
5. **Least-busy rule:** pick the fewest stops. Ties go to the smaller distance, then the smaller id.
6. The rule can be swapped on a running controller.
7. Adding a destination to an out-of-service or unknown elevator is refused.

### Names the driver uses

- `Direction`: `UP`, `DOWN` · `ElevatorStateType`: `IDLE`, `MOVING_UP`, `MOVING_DOWN`, `OUT_OF_SERVICE`
- `NearestElevatorStrategy()`, `LeastBusyStrategy()`
- `ElevatorController(strategy)`: `add_elevator(elevator_id, start_floor)`, `request_elevator(floor, direction) -> id or None`, `select_destination(elevator_id, floor)`, `take_out_of_service(elevator_id)`, `set_strategy`, `get_status() -> list of ElevatorInfo` (in the order the elevators were added)
- `ElevatorInfo`: `get_elevator_id()`, `get_current_floor()`, `get_state_type()`, `get_stop_count()`

### Driver code

```python
if __name__ == "__main__":
    c = ElevatorController(NearestElevatorStrategy())
    c.add_elevator("E1", 0)
    c.add_elevator("E2", 10)
    c.add_elevator("E3", 5)

    print(c.request_elevator(4, Direction.UP))      # E3     -> all idle, E3 is nearest (now heading down to 4)
    print(c.request_elevator(6, Direction.UP))      # E2     -> E3 heads down, so not suitable for UP
    print(c.request_elevator(2, Direction.DOWN))    # E1     -> idle E1 at distance 2 beats E3 at distance 3
    print(c.request_elevator(3, Direction.DOWN))    # E3     -> E3 moving down and 3 is on its way
    print(c.request_elevator(9, Direction.UP))      # E1     -> only E1 is moving up
    for info in c.get_status():
        print(info.get_elevator_id(), info.get_current_floor(), info.get_state_type().name, info.get_stop_count())
                                                    # E1 0 MOVING_UP 2
                                                    # E2 10 MOVING_DOWN 1
                                                    # E3 5 MOVING_DOWN 2
    print(c.request_elevator(12, Direction.DOWN))   # E2     -> nobody suitable, fewest stops wins

    c.set_strategy(LeastBusyStrategy())
    print(c.take_out_of_service("E3"))             # True
    print(c.request_elevator(1, Direction.UP))      # E1     -> E1 and E2 both have 2 stops, E1 is nearer
    print(c.request_elevator(0, Direction.DOWN))    # E2     -> E2 has 2 stops, E1 now has 3
    print(c.select_destination("E3", 1))            # False  -> out of service
    print(c.take_out_of_service("E9"))             # False  -> unknown elevator

    empty = ElevatorController(NearestElevatorStrategy())
    print(empty.request_elevator(3, Direction.UP))  # None   -> no elevators
```


---

## P05 · Rate Limiter Algorithms — Strategy (limiting algorithm)

**Pattern:** Strategy · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#06 · Rate Limiter`

### Problem

A rate limiter decides whether each request is allowed. **The limiting algorithm must be swappable** without the caller changing. A rule says "at most `limit` requests per `window_seconds`". Time is passed in as a number, so tests are deterministic.

1. **Fixed window:**
   - Time is cut into windows `[0, w)`, `[w, 2w)`, ….
   - At most `limit` requests are allowed per window, and the count resets when a new window starts.
2. **Sliding window log:**
   - Remember the times of allowed requests.
   - A request at `now` is allowed if fewer than `limit` remembered times are `> now - window`.
   - Rejected requests are not remembered.
3. **Token bucket:**
   - The bucket starts full with `limit` tokens.
   - It refills at `limit / window_seconds` tokens per second (`elapsed × rate`), never above `limit`.
   - A request needs 1 whole token.
4. Each algorithm keeps its own state, so a fresh tracker starts fresh.
5. The limiter's algorithm can be replaced at runtime. The new one starts with its own fresh state.

> This problem uses one client only. Per-client and per-endpoint limits come in **P31 (Factory)**.

### Names the driver uses

- `RateLimitRule(limit, window_seconds)`
- `FixedWindowTracker()`, `SlidingWindowLogTracker()`, `TokenBucketTracker()`, each with `try_acquire(rule, now) -> bool`
- `RateLimiter(rule, tracker)`: `allow(now) -> bool`, `set_tracker(tracker)`

### Driver code

```python
if __name__ == "__main__":
    rule = RateLimitRule(3, 10)        # -> 3 requests per 10 seconds

    fixed = RateLimiter(rule, FixedWindowTracker())
    print([fixed.allow(t) for t in [0, 1, 2, 3, 9, 10, 11]])
                                       # [True, True, True, False, False, True, True]
    burst = RateLimiter(rule, FixedWindowTracker())
    print([burst.allow(t) for t in [9, 9, 9, 10, 10, 10]])
                                       # [True, True, True, True, True, True]
                                       # -> 6 requests in 1 second: the fixed-window boundary burst

    sliding = RateLimiter(rule, SlidingWindowLogTracker())
    print([sliding.allow(t) for t in [9, 9, 9, 10, 10, 10]])
                                       # [True, True, True, False, False, False]
    print([sliding.allow(t) for t in [19, 20]])
                                       # [True, True]
                                       # -> 19 drops the 9s, rejected calls were never logged

    bucket = RateLimiter(rule, TokenBucketTracker())
    print([bucket.allow(t) for t in [0, 0, 0, 0, 5, 10]])
                                       # [True, True, True, False, True, True]
                                       # -> refill 0.3 tokens per second
    print([bucket.allow(t) for t in [100, 100, 100, 100]])
                                       # [True, True, True, False]
                                       # -> tokens are capped at the limit

    bucket.set_tracker(FixedWindowTracker())
    print(bucket.allow(100))           # True   -> swapped algorithm starts fresh
```


---

## P06 · LRU / LFU Cache — Strategy (eviction policy)

**Pattern:** Strategy · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#07 · LRU / LFU Cache`

### Problem

Build a fixed-capacity key-value cache whose **eviction rule is replaceable**. The cache owns the values and the capacity. The policy owns only the key order or counts. The cache tells the policy about every insert and every use, and asks it which key to evict.

1. `get(key)` returns the value, or `None` if the key is missing. A hit counts as a **use**.
2. `put(key, value)` inserts or updates a key. Updating an existing key counts as a **use**.
3. Inserting a new key into a full cache first evicts the key that the policy names.
4. **LRU:** evict the key used longest ago.
5. **LFU:**
   - Evict the key with the fewest uses. A new key starts at 1 use.
   - Break ties by evicting the key whose last use is oldest.
6. Both `get` and `put` must be O(1).
7. `put` on a cache with capacity ≤ 0 is refused (`False`). Otherwise `put` returns `True`.

### Names the driver uses

- `LruPolicy()`, `LfuPolicy()`
- `Cache(capacity, policy)`: `get`, `put`, `size`

### Driver code

```python
if __name__ == "__main__":
    lru = Cache(2, LruPolicy())
    lru.put("a", 1)
    lru.put("b", 2)
    print(lru.get("a"))        # 1
    lru.put("c", 3)            # -> evicts b (least recently used)
    print(lru.get("b"))        # None
    print(lru.get("c"))        # 3
    lru.put("a", 10)           # -> update counts as a use
    lru.put("d", 4)            # -> evicts c
    print(lru.get("c"), lru.get("a"), lru.get("d"), lru.size())
                               # None 10 4 2

    lfu = Cache(2, LfuPolicy())
    lfu.put("a", 1)
    lfu.put("b", 2)
    lfu.get("a")
    lfu.get("a")               # -> a used 3 times
    lfu.get("b")               # -> b used 2 times
    lfu.put("c", 3)            # -> evicts b (fewest uses)
    lfu.put("d", 4)            # -> evicts c (1 use) even though c is the newest
    print(lfu.get("b"), lfu.get("c"), lfu.get("a"), lfu.get("d"))
                               # None None 1 4

    tie = Cache(2, LfuPolicy())
    tie.put("x", 1)
    tie.put("y", 2)
    tie.put("z", 3)            # -> x and y both 1 use, x is older, so x goes
    print(tie.get("x"), tie.get("y"), tie.get("z"))
                               # None 2 3

    print(Cache(0, LruPolicy()).put("a", 1))   # False  -> zero capacity
```


---

## P07 · Ride Sharing — Strategy ×3 (matching, pricing, payment)

**Pattern:** Strategy ×3 · **Difficulty:** Medium · **Time:** ~35 min · **Source:** `#08 · Ride Sharing`

### Problem

Riders request rides and the service matches them with drivers. **Three rules must be swappable on a running service:** how a driver is matched, how the fare is priced, and how the ride is paid.

1. A driver has an id, a vehicle type (`AUTO`, `SEDAN`) and a location `(x, y)`. Distance is straight-line (Euclidean).
2. The matching rule sees only **read-only snapshots of free drivers of the requested vehicle type**. The service then reserves the chosen driver. If the reservation fails, it excludes that driver and asks again.
3. **Nearest rule:** the nearest free driver within a radius. Ties go to the smaller id. If none is within the radius, there is no match.
4. A matched ride gets id `RIDE-1`, `RIDE-2`, … and status `ACCEPTED`. With no match, the request returns `None`.
5. **Pricing:**
   - Each vehicle type has a rate card (base, per km, per minute). The fare is `base + per_km × distance + per_minute × minutes`, as an `int`.
   - **Surge pricing** wraps another pricing rule and multiplies its fare, rounding down.
6. `estimate_fare` uses the current pricing rule with an expected number of minutes.
7. Completing a ride calculates and **stores** the fare once, sets the status to `COMPLETED`, and frees the driver at the drop-off location. Changing the pricing later never changes a stored fare. Completing twice is refused.
8. A ride can be paid only after completion, and only once. **Payment:** card always succeeds. UPI succeeds only if the balance ≥ the fare.
9. **Refused actions** return `False` or `None`.

### Names the driver uses

- `VehicleType`: `AUTO`, `SEDAN` · `RideStatus` names printed: `ACCEPTED`
- `Location(x, y)`, `RateCard(base, per_km, per_minute)`
- `NearestDriverStrategy(radius)`, `DistanceTimePricing({type: RateCard})`, `SurgePricing(base_pricing, multiplier)`
- `CardPayment()`, `UpiPayment(balance)`
- `RideService(matching, pricing)`: `add_driver(driver_id, vehicle_type, location)`, `estimate_fare(pickup, dropoff, vehicle_type, expected_minutes)`, `request_ride(rider_id, pickup, dropoff, vehicle_type)`, `complete_ride(ride_id, duration_minutes) -> fare or None`, `pay_ride(ride_id, payment)`, `set_matching`, `set_pricing`
- Ride: `get_ride_id()`, `get_driver_id()`, `get_status()`, `get_fare()`

### Driver code

```python
if __name__ == "__main__":
    cards = {VehicleType.AUTO: RateCard(20, 8, 1), VehicleType.SEDAN: RateCard(50, 12, 2)}
    svc = RideService(NearestDriverStrategy(5), DistanceTimePricing(cards))
    svc.add_driver("D1", VehicleType.SEDAN, Location(0, 0))
    svc.add_driver("D2", VehicleType.SEDAN, Location(6, 8))      # -> 10 km from the origin
    svc.add_driver("D3", VehicleType.AUTO, Location(1, 0))

    print(svc.estimate_fare(Location(0, 0), Location(3, 4), VehicleType.SEDAN, 10))   # 130    -> 50 + 12*5 + 2*10
    r1 = svc.request_ride("U1", Location(0, 0), Location(3, 4), VehicleType.SEDAN)
    print(r1.get_ride_id(), r1.get_driver_id(), r1.get_status().name)                # RIDE-1 D1 ACCEPTED
    print(svc.request_ride("U2", Location(0, 0), Location(0, 3), VehicleType.SEDAN))  # None   -> D1 busy, D2 outside radius 5

    svc.set_matching(NearestDriverStrategy(20))
    r2 = svc.request_ride("U2", Location(0, 0), Location(0, 3), VehicleType.SEDAN)
    print(r2.get_driver_id())                                                          # D2     -> wider radius rule
    print(svc.request_ride("U3", Location(0, 0), Location(0, 3), VehicleType.SEDAN))  # None   -> no free sedan

    print(svc.pay_ride(r1.get_ride_id(), CardPayment()))     # False  -> not completed yet
    print(svc.complete_ride(r1.get_ride_id(), 10))           # 130
    print(svc.complete_ride(r1.get_ride_id(), 10))           # None   -> already completed
    print(svc.pay_ride(r1.get_ride_id(), UpiPayment(100)))   # False  -> 100 < 130
    print(svc.pay_ride(r1.get_ride_id(), CardPayment()))     # True
    print(svc.pay_ride(r1.get_ride_id(), CardPayment()))     # False  -> already paid

    svc.set_pricing(SurgePricing(DistanceTimePricing(cards), 1.5))
    r3 = svc.request_ride("U3", Location(3, 0), Location(3, 8), VehicleType.SEDAN)
    print(r3.get_driver_id())                                # D1     -> D1 is free again, now at (3, 4)
    print(svc.complete_ride(r3.get_ride_id(), 4))            # 231    -> (50 + 12*8 + 2*4) * 1.5
    r4 = svc.request_ride("U4", Location(0, 0), Location(4, 3), VehicleType.AUTO)
    print(r4.get_driver_id(), svc.complete_ride(r4.get_ride_id(), 5))   # D3 97  -> 65 * 1.5 = 97.5, rounded down
    print(r1.get_fare())                                     # 130    -> stored fare does not change with new pricing
```


---

## P08 · Logging Formats & Destinations — Strategy (formatter, destination)

**Pattern:** Strategy ×2 · **Difficulty:** Easy · **Time:** ~20 min · **Source:** `#09 · Logging Framework`

### Problem

A logger writes each entry to **several destinations at once**. Each destination owns its own **formatter**. New destinations and new formats must be addable without touching `Logger`.

1. Levels in order: `DEBUG` (10) < `INFO` (20) < `WARN` (30) < `ERROR` (40).
2. The logger has a minimum level. A message below it is dropped, and `log` returns `False`. Otherwise every destination gets the entry, and `log` returns `True`.
3. Each entry has a timestamp (from a clock), a level and a message. The clock is injected so tests are deterministic. A fixed clock always returns the same text.
4. **Formats:**
   - Plain: `[<time>] <LEVEL>: <message>`
   - JSON: `json.dumps({"time": ..., "level": ..., "message": ...})`, with the keys in that order.
5. **Destinations:**
   - Console prints the formatted line.
   - Memory keeps the formatted lines in a list (it stands in for a file or DB in tests).
6. The minimum level can be changed at runtime.

> P26 adds a handler chain, P30 makes the logger a Singleton, and P43 wraps destinations with Decorators.

### Names the driver uses

- `LogLevel`: `DEBUG`, `INFO`, `WARN`, `ERROR`
- `FixedClock(text)` with `now()`
- `PlainFormatter()`, `JsonFormatter()`
- `ConsoleDestination(formatter)`, `MemoryDestination(formatter)` with `get_lines()`
- `Logger(min_level, clock)`: `add_destination`, `log(level, message) -> bool`, `set_min_level`

### Driver code

```python
if __name__ == "__main__":
    logger = Logger(LogLevel.INFO, FixedClock("2026-01-01 10:00"))
    logger.add_destination(ConsoleDestination(PlainFormatter()))
    memory = MemoryDestination(JsonFormatter())
    logger.add_destination(memory)

    print(logger.log(LogLevel.DEBUG, "cache miss"))   # False  -> below INFO, dropped
    print(logger.log(LogLevel.INFO, "server started"))
                                                      # [2026-01-01 10:00] INFO: server started
                                                      # True
    print(logger.log(LogLevel.ERROR, "db down"))      # [2026-01-01 10:00] ERROR: db down
                                                      # True
    for line in memory.get_lines():
        print(line)
                                                      # {"time": "2026-01-01 10:00", "level": "INFO", "message": "server started"}
                                                      # {"time": "2026-01-01 10:00", "level": "ERROR", "message": "db down"}

    logger.set_min_level(LogLevel.DEBUG)
    print(logger.log(LogLevel.DEBUG, "cache miss"))   # [2026-01-01 10:00] DEBUG: cache miss
                                                      # True
    print(len(memory.get_lines()))                    # 3
```


---

## P09 · Snake and Ladder — Strategy (dice)

**Pattern:** Strategy · **Difficulty:** Easy · **Time:** ~20 min · **Source:** `#10 · Snake and Ladder`

### Problem

Build a snake-and-ladder game where **the dice are swappable**. Real games use random dice, and tests use a fixed sequence. The number of dice is configurable.

1. The board has cells `1..size`. Players start at `0` (off the board), and `size` is the final cell.
2. A snake (`head → tail`) needs `0 < tail < head < size`. A ladder (`bottom → top`) needs `0 < bottom < top ≤ size`. A cell can hold only one snake head or ladder bottom. Invalid ones are refused (`False`).
3. Players take turns in the order given. A turn is: roll all dice and add them up → if the move would go past the final cell, stay put → otherwise move, then follow a snake or ladder (one jump) → check for a win.
4. The first player to land exactly on the final cell wins. The game then ends, and `take_turn` returns `None`.
5. Positions live in the game, not on the player.
6. **Dice:**
   - Standard dice rolls a random number in `1..sides`.
   - Sequence dice returns its list of values in order, looping.
   - A dice set adds up all of its dice.

### Names the driver uses

- `Board(size)`: `add_snake(head, tail)`, `add_ladder(bottom, top)`
- `StandardDice(sides)`, `SequenceDice(values)`, `DiceSet(dice_list)` with `roll_total()`
- `Player(player_id, name)` with `get_player_id()`
- `Game(board, players, dice_set)`: `take_turn() -> TurnResult or None`, `get_status()` (`IN_PROGRESS` / `FINISHED`), `get_winner()`, `get_position(player_id)`
- TurnResult: `get_player_id()`, `get_roll()`, `get_start_cell()`, `get_end_cell()`, `has_won()`

### Driver code

```python
if __name__ == "__main__":
    board = Board(20)
    print(board.add_ladder(3, 11), board.add_ladder(6, 15))   # True True
    print(board.add_snake(12, 2), board.add_snake(17, 4))     # True True
    print(board.add_snake(5, 9))                              # False  -> head must be above tail
    print(board.add_ladder(3, 7))                             # False  -> cell 3 already used

    dice = DiceSet([SequenceDice([3, 2, 4, 6, 6, 4, 5])])
    game = Game(board, [Player("A", "Asha"), Player("B", "Ben")], dice)
    for _ in range(7):
        r = game.take_turn()
        print(r.get_player_id(), r.get_roll(), r.get_start_cell(), r.get_end_cell(), r.has_won())
                                                              # A 3 0 11 False   -> ladder 3 to 11
                                                              # B 2 0 2 False
                                                              # A 4 11 15 False
                                                              # B 6 2 8 False
                                                              # A 6 15 15 False  -> 21 overshoots, stay
                                                              # B 4 8 2 False    -> snake 12 to 2
                                                              # A 5 15 20 True
    print(game.take_turn())                                   # None   -> game over
    print(game.get_status().name, game.get_winner().get_player_id(), game.get_position("B"))
                                                              # FINISHED A 2

    print(DiceSet([SequenceDice([1]), SequenceDice([2])]).roll_total())   # 3      -> two dice added
    print(all(1 <= StandardDice(6).roll() <= 6 for _ in range(1000)))      # True
```


---

## P10 · Notification Channels — Strategy (channel)

**Pattern:** Strategy (+ a channel registry) · **Difficulty:** Easy–Medium · **Time:** ~25 min · **Source:** `#13 · Notification Service`

### Problem

Send a notification to a user through every channel they can receive on. **Each channel is a swappable strategy.** A new channel is added by registering it, with no change to the service.

1. Channels: `EMAIL`, `SMS`, `PUSH`. A channel is used only if it is **registered**, the user has a **contact** for it, and the user has **not disabled** it.
2. A user can opt out of a whole notification type (`TRANSACTIONAL`, `PROMOTIONAL`). An opted-out send returns `{}` and sends nothing.
3. **Templates:**
   - A template has a subject pattern and a body pattern with `{placeholders}`.
   - Rendering fills every placeholder. If any value is missing, the send returns `None`.
4. Unknown user or template: `None`.
5. Channels are tried in enum order (`EMAIL`, `SMS`, `PUSH`). `send` returns `{ChannelType: bool}` for every channel tried.
6. **Channel behaviour:**
   - Email prints `EMAIL to <address>: <subject> | <body>`.
   - SMS prints `SMS to <address>: <body>`. It is **refused (`False`, no print)** if the body is longer than its max length.
   - Push prints `PUSH to <address>: <subject>`.

> The synchronous send here becomes a queue + worker in **P32**, and event fan-out in **P22**.

### Names the driver uses

- `ChannelType`: `EMAIL`, `SMS`, `PUSH` · `NotificationType`: `TRANSACTIONAL`, `PROMOTIONAL`
- `EmailChannel()`, `SmsChannel(max_length)`, `PushChannel()`
- `ChannelRegistry()`: `register(channel_type, channel)`
- `NotificationService(registry)`: `register_user(user_id) -> User`, `add_template(name, subject_pattern, body_pattern)`, `send(user_id, notification_type, template_name, values)`
- User: `set_contact(channel_type, address)`, `set_channel_enabled(channel_type, bool)`, `set_type_enabled(notification_type, bool)`

### Driver code

```python
def names(result):
    return None if result is None else {k.name: v for k, v in result.items()}


if __name__ == "__main__":
    registry = ChannelRegistry()
    registry.register(ChannelType.EMAIL, EmailChannel())
    registry.register(ChannelType.SMS, SmsChannel(40))
    svc = NotificationService(registry)

    u = svc.register_user("u1")
    u.set_contact(ChannelType.EMAIL, "asha@mail.com")
    u.set_contact(ChannelType.SMS, "+91-999")
    u.set_contact(ChannelType.PUSH, "device-7")
    svc.add_template("otp", "Your OTP", "Hi {name}, OTP is {code}")
    svc.add_template("sale", "Big sale", "Hello {name}, everything in the store is half price today")

    print(names(svc.send("u1", NotificationType.TRANSACTIONAL, "otp", {"name": "Asha", "code": "4821"})))
                                     # EMAIL to asha@mail.com: Your OTP | Hi Asha, OTP is 4821
                                     # SMS to +91-999: Hi Asha, OTP is 4821
                                     # {'EMAIL': True, 'SMS': True}
                                     # -> no PUSH channel registered yet

    registry.register(ChannelType.PUSH, PushChannel())  # -> new channel, no other code changes
    print(names(svc.send("u1", NotificationType.PROMOTIONAL, "sale", {"name": "Asha"})))
                                     # EMAIL to asha@mail.com: Big sale | Hello Asha, everything in the store is half price today
                                     # PUSH to device-7: Big sale
                                     # {'EMAIL': True, 'SMS': False, 'PUSH': True}
                                     # -> SMS refused, body longer than 40

    u.set_type_enabled(NotificationType.PROMOTIONAL, False)
    print(names(svc.send("u1", NotificationType.PROMOTIONAL, "sale", {"name": "Asha"})))
                                     # {}
    u.set_channel_enabled(ChannelType.EMAIL, False)
    print(names(svc.send("u1", NotificationType.TRANSACTIONAL, "otp", {"name": "Asha", "code": "77"})))
                                     # SMS to +91-999: Hi Asha, OTP is 77
                                     # PUSH to device-7: Your OTP
                                     # {'SMS': True, 'PUSH': True}
    print(names(svc.send("u1", NotificationType.TRANSACTIONAL, "otp", {"name": "Asha"})))
                                     # None   -> missing placeholder code
    print(names(svc.send("u9", NotificationType.TRANSACTIONAL, "otp", {})))
                                     # None   -> unknown user
```


---

## P11 · Food Delivery Coupons — Strategy (discount + payment)

**Pattern:** Strategy ×2 · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#14 · Food Delivery`

### Problem

Users add dishes to a cart and place an order with an optional coupon. **The discount rule and the payment method are both swappable strategies.**

1. A restaurant has a menu of items (id, name, price). Adding an existing item id again updates its name and price.
2. A user's cart holds items from **one restaurant only**. Adding from another restaurant, an unknown item, or a quantity ≤ 0 is refused.
3. The cart stores only item ids and counts. **Prices are read from the menu when the order is placed.**
4. **Coupons:**
   - A coupon has a code, a minimum subtotal and a discount rule.
   - **Percent:** `subtotal × percent // 100`, capped at a maximum.
   - **Flat:** a fixed amount, never more than the subtotal.
5. **Placing an order:**
   - The order is refused (`None`) if the cart is empty, the coupon is unknown, the subtotal is below the coupon minimum, or the payment fails.
   - **On a refusal the cart is kept.** On success the cart is cleared.
6. An order (id `O1`, `O2`, …) **stores** its subtotal and discount once. The total is `subtotal − discount`.
7. **Payment:** card always succeeds. UPI deducts from its balance and fails if the balance is too low. Cancelling an order refunds the total to the same payment method, and cancelling twice is refused.

### Names the driver uses

- `Restaurant(restaurant_id, name)`: `add_item(item_id, name, price)`
- `PercentDiscount(percent, cap)`, `FlatDiscount(amount)`
- `CardPayment()`, `UpiPayment(balance)` with `get_balance()`
- `OrderService()`: `add_restaurant`, `add_coupon(code, min_order_total, strategy)`, `add_to_cart(user_id, restaurant_id, item_id, quantity)`, `place_order(user_id, payment, coupon_code or None)`, `cancel_order(order_id)`
- Order: `get_order_id()`, `get_subtotal()`, `get_discount()`, `get_total()`

### Driver code

```python
if __name__ == "__main__":
    spice = Restaurant("R1", "Spice Hub")
    spice.add_item("I1", "Biryani", 250)
    spice.add_item("I2", "Lassi", 60)
    pizza = Restaurant("R2", "Pizza Point")
    pizza.add_item("P1", "Pizza", 300)
    svc = OrderService()
    svc.add_restaurant(spice)
    svc.add_restaurant(pizza)
    svc.add_coupon("SAVE10", 300, PercentDiscount(10, 50))
    svc.add_coupon("FLAT100", 500, FlatDiscount(100))

    print(svc.add_to_cart("u1", "R1", "I1", 2))     # True
    print(svc.add_to_cart("u1", "R2", "P1", 1))     # False  -> cart holds one restaurant only
    print(svc.add_to_cart("u1", "R1", "I9", 1))     # False  -> unknown item
    print(svc.add_to_cart("u1", "R1", "I2", 1))     # True   -> subtotal 560
    print(svc.place_order("u1", CardPayment(), "BOGUS"))   # None   -> unknown coupon
    o1 = svc.place_order("u1", CardPayment(), "SAVE10")
    print(o1.get_order_id(), o1.get_subtotal(), o1.get_discount(), o1.get_total())
                                                    # O1 560 50 510
                                                    # -> 10 percent is 56, capped at 50
    print(svc.place_order("u1", CardPayment(), None))      # None   -> cart was cleared

    svc.add_to_cart("u2", "R2", "P1", 1)
    print(svc.place_order("u2", CardPayment(), "FLAT100")) # None   -> 300 is below the 500 minimum
    print(svc.place_order("u2", UpiPayment(200), "SAVE10"))  # None   -> 270 due, balance 200, cart kept
    pizza.add_item("P1", "Pizza", 400)                     # -> price change before ordering
    upi = UpiPayment(1000)
    o2 = svc.place_order("u2", upi, "SAVE10")
    print(o2.get_order_id(), o2.get_subtotal(), o2.get_discount(), o2.get_total())
                                                    # O2 400 40 360
                                                    # -> price read from the menu at order time
    print(upi.get_balance())                        # 640
    print(svc.cancel_order("O2"), upi.get_balance())  # True 1000
    print(svc.cancel_order("O2"))                   # False  -> already cancelled
```


---

## P12 · Library Search — Strategy (search criterion)

**Pattern:** Strategy · **Difficulty:** Easy · **Time:** ~15 min · **Source:** `#15 · Library Management`

### Problem

Members search the catalogue by title, author or subject. **Each way of matching is a criterion object**, and the service just keeps the books that match. A new kind of search must be addable without editing `LibraryService`.

1. A book has an isbn, title, author and subject. Adding a duplicate isbn is refused.
2. **Criteria:**
   - Title, author and subject criteria match on **case-insensitive "contains"**.
   - A combined criterion matches only if **all** of its inner criteria match.
3. `search(criterion)` returns the matching books in the order they were added. It returns an empty list if none match.
4. Removing a book returns `True` once and `False` after that.

### Names the driver uses

- `TitleContains(text)`, `AuthorContains(text)`, `SubjectContains(text)`, `AllOf(criteria_list)`
- `LibraryService()`: `add_book(isbn, title, author, subject)`, `remove_book(isbn)`, `search(criterion) -> list of Book`
- Book: `get_title()`

### Driver code

```python
def titles(books):
    return [b.get_title() for b in books]


if __name__ == "__main__":
    lib = LibraryService()
    lib.add_book("1", "Clean Code", "Robert Martin", "Software")
    lib.add_book("2", "Clean Architecture", "Robert Martin", "Software")
    lib.add_book("3", "The Pragmatic Programmer", "Andy Hunt", "Software")
    lib.add_book("4", "Sapiens", "Yuval Harari", "History")
    print(lib.add_book("4", "Duplicate", "X", "Y"))                 # False  -> isbn exists

    print(titles(lib.search(TitleContains("clean"))))              # ['Clean Code', 'Clean Architecture']
    print(titles(lib.search(AuthorContains("HUNT"))))              # ['The Pragmatic Programmer']
    print(titles(lib.search(SubjectContains("history"))))          # ['Sapiens']
    print(titles(lib.search(AllOf([SubjectContains("software"), TitleContains("arch")]))))
                                                                   # ['Clean Architecture']
    print(titles(lib.search(TitleContains("rust"))))               # []
    print(lib.remove_book("1"), lib.remove_book("1"))              # True False
    print(titles(lib.search(AuthorContains("martin"))))            # ['Clean Architecture']
```


---

## P13 · Task Scheduler Timing — Strategy (schedule)

**Pattern:** Strategy · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#17 · Task Scheduler`

### Problem

A scheduler runs named jobs when they are due. **When a job runs is decided by a schedule object**, so a new kind of schedule must not require any change to `Scheduler`. Time is an `int` passed in.

1. **Schedules:**
   - A schedule answers two questions: when is the first run, and given a planned run time, when is the next one (`None` = no more).
   - **One-time:** runs once at a given time.
   - **Fixed interval:** starts at `start`, then every `interval`.
   - **List:** runs at each time in a list. The list may arrive unsorted.
2. `schedule(name, schedule)` returns job ids `J1`, `J2`, ….
3. `run_due(now)` runs **every occurrence that is due (planned ≤ now)**, earliest first. Ties at the same time go to the earlier-scheduled job. It returns a list of `"<name>@<planned_time>"`.
4. **No drift:** the next run is computed from the **planned** time, not from `now`. A late `run_due` catches up on every missed occurrence.
5. **Status:**
   - A job is `SCHEDULED` until its schedule has no next run (`COMPLETED`) or it is cancelled (`CANCELLED`).
   - Cancelling a job that is not `SCHEDULED` is refused.
   - An unknown job id returns status `None`.

> A min-heap ordered by `(run_at, seq)` makes this O(log n) per run. The same scheduler returns with Command tasks (P29) and a worker pool (P33).

### Names the driver uses

- `OneTimeSchedule(at)`, `FixedIntervalSchedule(start, interval)`, `ListSchedule(times)`
- `JobStatus` names printed: `COMPLETED`, `CANCELLED`
- `Scheduler()`: `schedule(name, schedule) -> job_id`, `run_due(now) -> list[str]`, `cancel(job_id)`, `get_status(job_id)`

### Driver code

```python
if __name__ == "__main__":
    s = Scheduler()
    j1 = s.schedule("backup", OneTimeSchedule(10))
    j2 = s.schedule("ping", FixedIntervalSchedule(0, 15))
    j3 = s.schedule("report", ListSchedule([12, 5]))
    print(j1, j2, j3)                  # J1 J2 J3

    print(s.run_due(0))                # ['ping@0']
    print(s.run_due(12))               # ['report@5', 'backup@10', 'report@12']
    print(s.run_due(14))               # []
    print(s.run_due(31))               # ['ping@15', 'ping@30']
                                       # -> late run catches up, and next times come from the planned time, not now
    print(s.get_status(j1).name, s.get_status(j3).name)
                                       # COMPLETED COMPLETED

    j4 = s.schedule("cleanup", OneTimeSchedule(50))
    print(s.cancel(j2), s.cancel(j2))  # True False
    print(s.run_due(100))              # ['cleanup@50']
    print(s.get_status(j2).name)       # CANCELLED

    s.schedule("a", OneTimeSchedule(200))
    s.schedule("b", FixedIntervalSchedule(200, 100))
    print(s.run_due(200))              # ['a@200', 'b@200']
                                       # -> same time, earlier job first
    print(s.cancel(j1), s.get_status("J99"))
                                       # False None
```


---

## P14 · File Search — Strategy (search criterion)

**Pattern:** Strategy · **Difficulty:** Easy · **Time:** ~20 min · **Source:** `#20 · In-Memory File System`

### Problem

Search a directory tree for files. **How a file name is matched is a criterion object**, so a new kind of search needs no change to `FileSystem`.

1. Paths are absolute, like `/docs/readme.md`. Creating a file or directory is refused if the parent directory does not exist or the name is already taken.
2. **Criteria** (match on the file name only, never on directories):
   - Exact name.
   - Extension, case-insensitive (`notes.TXT` matches `txt`).
   - Name starts with a prefix.
3. `search(path, criterion)` searches the whole subtree under `path` and returns full file paths, visiting each directory's children in **sorted name order** (depth-first).
4. Searching from a path that is not a directory returns `None`. No matches returns `[]`.

> A plain nested dict is fine here. The full Composite tree (size, move, rename, delete) is **P28**.

### Names the driver uses

- `NameEquals(name)`, `ExtensionEquals(ext)`, `NameStartsWith(prefix)`
- `FileSystem()`: `create_directory(path)`, `create_file(path, content)`, `search(path, criterion) -> list[str] or None`

### Driver code

```python
if __name__ == "__main__":
    fs = FileSystem()
    fs.create_directory("/docs")
    fs.create_directory("/docs/old")
    fs.create_directory("/src")
    fs.create_file("/docs/readme.md", "hi")
    fs.create_file("/docs/notes.TXT", "a")
    fs.create_file("/docs/old/readme.md", "old")
    fs.create_file("/src/main.py", "print()")
    fs.create_file("/src/test_main.py", "assert")
    print(fs.create_file("/nope/a.txt", "x"))                     # False  -> parent missing
    print(fs.create_file("/docs/readme.md", "x"))                 # False  -> already exists

    print(fs.search("/", NameEquals("readme.md")))                # ['/docs/old/readme.md', '/docs/readme.md']
    print(fs.search("/", ExtensionEquals("txt")))                 # ['/docs/notes.TXT']
    print(fs.search("/src", ExtensionEquals("py")))               # ['/src/main.py', '/src/test_main.py']
    print(fs.search("/", NameStartsWith("test_")))                # ['/src/test_main.py']
    print(fs.search("/docs/old", ExtensionEquals("py")))          # []
    print(fs.search("/src/main.py", NameEquals("main.py")))       # None   -> not a directory
```


---

## P15 · Web Crawler Processing — Strategy (content processor)

**Pattern:** Strategy · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#23 · Web Crawler`

### Problem

Crawl a (fake, in-memory) web breadth-first from seed URLs. **What happens to each fetched page is a list of pluggable processors.** A new processor needs no change to `Crawler`.

1. **Normalize every URL** before using it: lower-case it and drop one trailing `/`. The domain is the part between `://` and the next `/`.
2. **Claim a URL in `seen` before queueing it**, so each URL is fetched at most once.
3. **The policy:**
   - Skips links to domains that are not allowed.
   - Skips links deeper than `max_depth`. Seeds are depth 0, and links found on a depth-`d` page are depth `d+1`.
4. A fetch returns the page text, or `None` if the page doesn't exist. Failed URLs are recorded in order and are **not** passed to processors.
5. Links are the words in the page text that start with `http://`.
6. `crawl(seeds)` returns the successfully fetched URLs in BFS order.
7. **Processors:**
   - Archive keeps the fetched URLs in order.
   - Word count stores the number of whitespace-separated words per URL (`None` if the URL was never processed).

> The multi-threaded version of this crawler is **P34**.

### Names the driver uses

- `InMemoryFetcher(pages_dict)`, `CrawlPolicy(max_depth, allowed_domains)`
- `ArchiveProcessor()` with `get_urls()`, `WordCountProcessor()` with `get_count(url)`
- `Crawler(fetcher, policy, processors)`: `crawl(seeds) -> list[str]`, `get_failed() -> list[str]`

### Driver code

```python
if __name__ == "__main__":
    web = {
        "http://a.com": "home http://a.com/about http://b.com http://A.com/about/ http://evil.com",
        "http://a.com/about": "about us http://a.com/team",
        "http://b.com": "partner site http://a.com http://b.com/broken",
        "http://a.com/team": "team page http://a.com/deeper",
        "http://a.com/deeper": "too deep",
    }
    archive, words = ArchiveProcessor(), WordCountProcessor()
    crawler = Crawler(InMemoryFetcher(web), CrawlPolicy(2, ["a.com", "b.com"]), [archive, words])

    print(crawler.crawl(["http://a.com/"]))
                                       # ['http://a.com', 'http://a.com/about', 'http://b.com', 'http://a.com/team']
                                       # -> A.com/about/ is the same URL after normalizing, evil.com blocked, deeper is depth 3
    print(crawler.get_failed())        # ['http://b.com/broken']
    print(archive.get_urls() == ['http://a.com', 'http://a.com/about', 'http://b.com', 'http://a.com/team'])
                                       # True   -> every processor saw every fetched page
    print(words.get_count("http://a.com/about"), words.get_count("http://b.com"))
                                       # 3 4
    print(words.get_count("http://b.com/broken"))
                                       # None   -> failed pages are not processed
```


---

## P16 · Cab Booking (scheduled trips) — Strategy (allocation rule)

**Pattern:** Strategy (decides on read-only copies, M10) · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#25 · Cab Booking`

### Problem

Riders book cabs for a time window, now or later. A cab can't be on two overlapping trips, so **each cab owns a calendar of reserved windows**, not a busy flag. **Which free cab gets the trip is a swappable rule.**

1. Cabs have a type (`MINI`, `SEDAN`) and a fixed location.
2. Windows `[start, end)` overlap if `a.start < b.end and b.start < a.end`. Back-to-back windows don't overlap.
3. **Booking:**
   - Build read-only snapshots of the cabs of the requested type that are free for the window. Each snapshot holds the id, type, location and number of booked trips.
   - Ask the rule to choose, then try to reserve the window on the real cab.
   - If the reservation fails, exclude that cab and ask again. With no candidates, return `None`.
4. **Rules:**
   - **Nearest:** smallest distance to the pickup, ties go to the smaller id.
   - **Least trips:** fewest booked trips, then nearest, then the smaller id.
5. Trip ids are `T1`, `T2`, …. **A failed booking does not use up an id.** The fare is fixed at booking: `base + per_km × distance(pickup, dropoff)`, as an `int`.
6. Cancelling a booked trip frees the window on its cab. Cancelling twice is refused.

### Names the driver uses

- `CabType`: `MINI`, `SEDAN` · `Location(x, y)`, `RateCard(base, per_km)`, `FareCalculator({type: RateCard})`
- `NearestCabStrategy()`, `LeastTripsStrategy()`
- `CabService(strategy, fare_calculator)`: `add_cab(cab_id, cab_type, location)`, `book(rider_id, pickup, dropoff, cab_type, start, end) -> Trip or None`, `cancel_trip(trip_id)`, `set_strategy`
- Trip: `get_trip_id()`, `get_cab_id()`, `get_fare()`, `get_status()` (`BOOKED` / `CANCELLED`)

### Driver code

```python
if __name__ == "__main__":
    fares = FareCalculator({CabType.MINI: RateCard(30, 10), CabType.SEDAN: RateCard(50, 15)})
    svc = CabService(NearestCabStrategy(), fares)
    svc.add_cab("C1", CabType.MINI, Location(0, 0))
    svc.add_cab("C2", CabType.MINI, Location(10, 0))
    svc.add_cab("C3", CabType.SEDAN, Location(1, 0))

    t1 = svc.book("U1", Location(1, 0), Location(4, 4), CabType.MINI, 100, 160)
    print(t1.get_trip_id(), t1.get_cab_id(), t1.get_fare())        # T1 C1 80    -> 30 + 10*5
    t2 = svc.book("U2", Location(0, 0), Location(3, 4), CabType.MINI, 130, 200)
    print(t2.get_cab_id())                                          # C2     -> C1 busy 100-160
    print(svc.book("U3", Location(0, 0), Location(1, 1), CabType.MINI, 150, 170))
                                                                    # None   -> both minis busy then
    t4 = svc.book("U4", Location(0, 0), Location(0, 2), CabType.MINI, 160, 220)
    print(t4.get_cab_id())                                          # C1     -> back-to-back window is allowed

    print(svc.cancel_trip("T2"), svc.cancel_trip("T2"))             # True False
    t5 = svc.book("U5", Location(0, 0), Location(1, 0), CabType.MINI, 150, 170)
    print(t5.get_trip_id(), t5.get_cab_id())                        # T4 C2  -> C2 freed by the cancel

    svc.set_strategy(LeastTripsStrategy())
    t6 = svc.book("U6", Location(0, 0), Location(1, 0), CabType.MINI, 300, 310)
    print(t6.get_cab_id())                                          # C2     -> 1 trip vs C1's 2, even though C1 is nearer

    t7 = svc.book("U7", Location(1, 0), Location(1, 3), CabType.SEDAN, 100, 110)
    print(t7.get_cab_id(), t7.get_fare(), t7.get_status().name)     # C3 95 BOOKED
```


---

## P17 · Wallet Top-up Gateways — Strategy (gateway by method type)

**Pattern:** Strategy (+ a router that picks the strategy) · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#27 · Digital Wallet`

### Problem

Users top up a wallet from a card or a bank account. **Each payment method type goes through its own gateway**, chosen by a router, so gateways can be swapped without touching `WalletService`. Every request carries an **idempotency key**.

1. Wallet ids are `W1`, `W2`, …, and a new wallet's balance is 0.
2. Every gateway answers `SUCCESS` (with a reference), `DECLINED` or `ERROR`.
3. **Retry rule:**
   - `ERROR` is retried, up to `max_attempts` calls in total.
   - `DECLINED` is final and never retried.
4. On `SUCCESS` the wallet is credited and the transaction is `SUCCESS` with the gateway reference. Otherwise it is `FAILED` with reference `None`. The transaction records how many gateway calls were made.
5. **Idempotency:** a repeated key returns **the same transaction object** and never calls the gateway again, even if the first attempt failed.
6. Return `None` (no transaction recorded) for an unknown wallet, an amount ≤ 0, or a method type with no gateway.
7. **Gateways:**
   - The card gateway declines amounts above a limit.
   - The bank gateway returns `ERROR` for its first `n` calls, then `SUCCESS`.
   - References are `card-<key>` and `bank-<key>`. Both count their calls.

> P45 (Adapter) wraps third-party SDKs that have different APIs into this same gateway interface.

### Names the driver uses

- `MethodType`: `CARD`, `BANK_ACCOUNT` · status names printed: `SUCCESS`, `FAILED`
- `Card(card_number)`, `BankAccount(account_number)`
- `CardGateway(decline_above)`, `BankGateway(errors_before_success)`, both with `get_calls()`
- `GatewayRouter({MethodType: gateway})`
- `WalletService(router, max_attempts)`: `create_wallet(owner_id) -> wallet_id`, `top_up(key, wallet_id, method, amount) -> Transaction or None`, `get_balance(wallet_id)`
- Transaction: `get_status()`, `get_attempts()`, `get_reference()`

### Driver code

```python
def show(t):
    print(t.get_status().name, t.get_attempts(), t.get_reference())


if __name__ == "__main__":
    card_gw, bank_gw = CardGateway(5000), BankGateway(2)
    svc = WalletService(GatewayRouter({MethodType.CARD: card_gw, MethodType.BANK_ACCOUNT: bank_gw}), 3)
    w = svc.create_wallet("asha")
    print(w)                                                # W1

    t1 = svc.top_up("k1", w, Card("4111"), 1000)
    show(t1)                                                # SUCCESS 1 card-k1
    print(svc.get_balance(w))                               # 1000
    again = svc.top_up("k1", w, Card("4111"), 1000)
    print(again is t1, svc.get_balance(w))                  # True 1000  -> same key, not charged twice

    show(svc.top_up("k2", w, Card("4111"), 9000))           # FAILED 1 None   -> DECLINED is not retried
    show(svc.top_up("k3", w, BankAccount("ACC-1"), 500))    # SUCCESS 3 bank-k3  -> 2 ERRORs retried
    print(svc.get_balance(w), card_gw.get_calls(), bank_gw.get_calls())
                                                            # 1500 2 3

    svc2 = WalletService(GatewayRouter({MethodType.BANK_ACCOUNT: BankGateway(5)}), 3)
    w2 = svc2.create_wallet("ben")
    show(svc2.top_up("k1", w2, BankAccount("ACC-2"), 100)) # FAILED 3 None   -> gave up after 3 tries
    print(svc2.top_up("k2", w2, Card("4111"), 100))         # None   -> no gateway for cards here
    print(svc2.top_up("k3", "W9", BankAccount("ACC-2"), 100))
                                                            # None   -> unknown wallet
    print(svc2.top_up("k4", w2, BankAccount("ACC-2"), 0))   # None   -> bad amount
    print(svc2.get_balance(w2))                             # 0
```


---

## P18 · Airline Seat Pricing — Strategy (pricing + payment)

**Pattern:** Strategy ×2 · **Difficulty:** Easy–Medium · **Time:** ~20 min · **Source:** `#28 · Airline Management`

### Problem

A flight sells seats in different classes. **The pricing rule is swappable at any time**, but a passenger's price is **fixed at the moment the seat is held**. A later pricing change must not affect a seat that is already held.

1. A flight has a base fare and seats, each of class `ECONOMY`, `BUSINESS` or `FIRST`.
2. **Pricing rules:**
   - **Class multiplier:** `base_fare × multiplier[class]`.
   - **Surge:** wraps another rule and adds `percent`: `p + p × percent // 100`.
3. `quote(seat_class)` uses the current rule.
4. **Seat lifecycle** (`AVAILABLE` → `HELD` → `BOOKED`):
   - `hold_seat` works only on an `AVAILABLE` seat. It moves the seat to `HELD`, stores the current price and returns it.
   - `confirm` charges the stored price. If the payment fails, the seat stays `HELD`.
   - `release` returns a `HELD` seat to `AVAILABLE`.
5. **Payment:** a card succeeds if the amount ≤ its limit.
6. Unknown seats and wrong-state actions are refused (`None` / `False`).

### Names the driver uses

- `SeatClass`: `ECONOMY`, `BUSINESS`, `FIRST` · seat status names printed: `AVAILABLE`, `HELD`, `BOOKED`
- `ClassMultiplierPricing({class: multiplier})`, `SurgePricing(base_pricing, percent)`, `CardPayment(limit)`
- `Flight(flight_number, base_fare, pricing)`: `add_seat(seat_number, seat_class)`, `quote`, `hold_seat(seat_number, passenger_name) -> price or None`, `confirm(seat_number, payment)`, `release(seat_number)`, `set_pricing`, `get_seat_status(seat_number)`

### Driver code

```python
if __name__ == "__main__":
    by_class = ClassMultiplierPricing({SeatClass.ECONOMY: 1, SeatClass.BUSINESS: 3, SeatClass.FIRST: 5})
    f = Flight("AI-101", 5000, by_class)
    f.add_seat("1A", SeatClass.FIRST)
    f.add_seat("2A", SeatClass.BUSINESS)
    f.add_seat("10A", SeatClass.ECONOMY)
    f.add_seat("10B", SeatClass.ECONOMY)

    print(f.quote(SeatClass.ECONOMY), f.quote(SeatClass.BUSINESS), f.quote(SeatClass.FIRST))
                                                    # 5000 15000 25000
    print(f.hold_seat("10A", "Asha"))               # 5000
    print(f.hold_seat("10A", "Ben"))                # None   -> already held

    f.set_pricing(SurgePricing(by_class, 20))       # -> holiday surge, 20 percent on top
    print(f.quote(SeatClass.ECONOMY))               # 6000
    print(f.hold_seat("10B", "Ben"))                # 6000

    print(f.confirm("10A", CardPayment(5500)))      # True   -> Asha pays the 5000 fixed at hold time
    print(f.get_seat_status("10A").name)            # BOOKED
    print(f.confirm("10B", CardPayment(5500)))      # False  -> 6000 is over the card limit
    print(f.get_seat_status("10B").name)            # HELD
    print(f.release("10B"), f.get_seat_status("10B").name)
                                                    # True AVAILABLE
    print(f.confirm("2A", CardPayment(99999)))      # False  -> 2A was never held
    print(f.hold_seat("99Z", "Dev"))                # None   -> unknown seat
    print(f.hold_seat("1A", "Dev"))                 # 30000
```


---

## P19 · Vending Machine — State

**Pattern:** State · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#05 · Vending Machine`

### Problem

Design a vending machine that sells products from slots. Each slot holds one product (with a name and a price) and a quantity.

The machine is always in one of four states: **idle**, **has money**, **dispensing** or **out of service**. The same action behaves differently depending on the state.

1. A customer must insert money before selecting a product. Selecting a product while idle is refused.
2. Inserting zero or a negative amount is refused.
3. After money is in, the customer may select a product. Selecting an unknown or out-of-stock product is refused, and the money stays in the machine.
4. Selecting another product before the sale replaces the earlier selection.
5. As soon as a product is selected **and** enough money has been inserted, whichever happens last, the machine sells it automatically. It goes through the dispensing state, prints `dispensed <name>, change <amount>` and returns to idle. Exact payment gives change `0`.
6. If the money is not enough yet, the machine waits for more.
7. The customer may cancel while money is in and get all of it back.
8. The operator may restock a slot (count must be greater than 0) and collect the money earned from sales, but only when no customer is mid-purchase (idle or out of service).
9. The operator may take the machine out of service only when it is idle. While out of service, every customer action is refused until the operator returns it to service.
10. **Refused actions:** methods that return `bool` return `False`. `cancel()` and `collect_money()` return `0`. A refused action changes nothing.

**Constraint:** the main machine class must not branch on its current state. Behaviour must change by switching which state is active.

### Names the driver uses

Your other class names and internals are up to you. Only the driver below fixes these names.

- `MachineStateType` enum with `IDLE`, `HAS_MONEY`, `DISPENSING`, `OUT_OF_SERVICE`
- `Product(name, price)`
- `VendingMachine()` with `add_slot`, `insert_money`, `select_product`, `cancel`, `restock`, `collect_money`, `take_out_of_service`, `return_to_service`, `get_state`

Return values are shown in the driver's comments.

### Driver code

```python
if __name__ == "__main__":
    vm = VendingMachine()
    vm.add_slot("A1", Product("Coke", 25), 1)
    vm.add_slot("B1", Product("Chips", 40), 0)
    vm.add_slot("C1", Product("Candy", 50), 2)

    print(vm.select_product("A1"))    # False  -> idle: money first
    print(vm.insert_money(0))         # False  -> bad amount
    print(vm.insert_money(20))        # True
    print(vm.select_product("Z9"))    # False  -> unknown slot
    print(vm.select_product("B1"))    # False  -> out of stock, money stays
    print(vm.select_product("C1"))    # True   -> needs 30 more, waits
    print(vm.select_product("A1"))    # True   -> replaces C1, needs 5 more, waits
    print(vm.restock("A1", 2))        # False  -> customer mid-purchase
    print(vm.collect_money())         # 0      -> customer mid-purchase
    print(vm.take_out_of_service())   # False  -> customer mid-purchase
    print(vm.get_state().name)        # HAS_MONEY
    print(vm.insert_money(5))         # dispensed Coke, change 0
                                      # True
    print(vm.get_state().name)        # IDLE
    print(vm.cancel())                # 0      -> nothing to refund
    print(vm.insert_money(60))        # True
    print(vm.select_product("C1"))    # dispensed Candy, change 10
                                      # True
    print(vm.insert_money(50))        # True
    print(vm.select_product("A1"))    # False  -> A1 sold out now
    print(vm.cancel())                # 50
    print(vm.collect_money())         # 75
    print(vm.take_out_of_service())   # True
    print(vm.take_out_of_service())   # False  -> already out of service
    print(vm.insert_money(10))        # False  -> out of service
    print(vm.restock("A1", 0))        # False  -> bad count
    print(vm.restock("A1", 2))        # True
    print(vm.return_to_service())     # True
    print(vm.get_state().name)        # IDLE
```


---

## P20 · ATM — State

**Pattern:** State · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#12 · ATM`

### Problem

An ATM behaves differently depending on its state: **idle**, **has card**, **authenticated** or **dispensing**. The bank owns balances and PIN attempts. The ATM owns its cash.

1. A card can be inserted only while idle, and only if the bank knows it and it is not blocked.
2. With a card inside, the user enters a PIN. The result is `SUCCESS` (go to authenticated), `WRONG_PIN` (stay), or `BLOCKED` on the **3rd consecutive wrong PIN**. A blocked card is kept by the machine, and the ATM goes back to idle. A correct PIN resets the wrong-attempt count.
3. Only when authenticated can the user check the balance, withdraw or deposit. These stay authenticated afterwards, so several operations are allowed.
4. **Withdrawal:**
   - Refused if the amount is not a positive multiple of 100, exceeds the machine's cash, or exceeds the account balance.
   - Otherwise the ATM goes through the **dispensing** state, debits the account, removes the cash, prints `dispensed <amount>` and returns to authenticated.
5. A deposit (amount > 0) credits the account and adds the cash to the machine.
6. `cancel` returns the card and goes to idle, from has card or authenticated.
7. **Refused actions:** `bool` methods return `False`. `enter_pin` and `check_balance` return `None`.

**Constraint:** `ATM` must not branch on its state. It forwards each call to the current state object.

### Names the driver uses

- `ATMStateType`: `IDLE`, `HAS_CARD`, `AUTHENTICATED`, `DISPENSING` · `AuthResult`: `SUCCESS`, `WRONG_PIN`, `BLOCKED`
- `BankService()`: `add_account(card_number, pin, balance)`
- `ATM(bank, cash)`: `insert_card`, `enter_pin`, `check_balance`, `withdraw`, `deposit`, `cancel`, `get_state`, `get_cash`

### Driver code

```python
if __name__ == "__main__":
    bank = BankService()
    bank.add_account("C1", "1234", 1000)
    bank.add_account("C2", "0000", 500)
    bank.add_account("C3", "1111", 100)
    atm = ATM(bank, 800)

    print(atm.enter_pin("1234"))               # None   -> no card yet
    print(atm.insert_card("C9"))               # False  -> unknown card
    print(atm.insert_card("C1"))               # True
    print(atm.insert_card("C2"))               # False  -> a card is already inside
    print(atm.check_balance())                 # None   -> PIN first
    print(atm.enter_pin("9999").name)          # WRONG_PIN
    print(atm.enter_pin("1234").name)          # SUCCESS
    print(atm.get_state().name)                # AUTHENTICATED
    print(atm.check_balance())                 # 1000
    print(atm.withdraw(250))                   # False  -> not a multiple of 100
    print(atm.withdraw(900))                   # False  -> machine holds only 800
    print(atm.withdraw(300))                   # dispensed 300
                                               # True
    print(atm.get_state().name)                # AUTHENTICATED
    print(atm.deposit(200))                    # True
    print(atm.check_balance(), atm.get_cash()) # 900 700
    print(atm.cancel(), atm.get_state().name)  # True IDLE
    print(atm.cancel())                        # False  -> no card

    atm.insert_card("C3")
    atm.enter_pin("1111")
    print(atm.withdraw(200))                   # False  -> balance only 100
    atm.cancel()

    atm.insert_card("C2")
    print(atm.enter_pin("1").name, atm.enter_pin("2").name)
                                               # WRONG_PIN WRONG_PIN
    print(atm.enter_pin("3").name)             # BLOCKED
    print(atm.get_state().name)                # IDLE   -> card kept by the machine
    print(atm.insert_card("C2"))               # False  -> blocked card
```


---

## P21 · Elevator Car — State

**Pattern:** State · **Difficulty:** Medium–Hard · **Time:** ~35 min · **Source:** `#04 · Elevator`

### Problem

Model **one** elevator that runs on ticks. Its behaviour depends on its state: `IDLE`, `MOVING_UP`, `MOVING_DOWN` or `OUT_OF_SERVICE`. **Each tick does exactly one action:** open the door, close the door, move one floor, or change direction.

1. The elevator serves floors `min..max`. A stop outside that range is refused. Stops are a set, so adding one twice is fine.
2. **Adding a stop:**
   - When **idle**, a stop above switches to `MOVING_UP` straight away, and a stop below switches to `MOVING_DOWN`. A stop at the current floor stays `IDLE`, and the next tick opens the door.
   - When **moving**, the stop is just added.
   - When **out of service**, it is refused.
3. **Tick while moving**, checked in this order:
   1. Door open → close it. If no stops are left, become `IDLE`.
   2. The current floor is a stop → remove it and open the door.
   3. Load > capacity → do nothing (don't move).
   4. A stop is ahead in the current direction → move one floor.
   5. A stop is behind → reverse direction (no move this tick).
   6. Otherwise → `IDLE`.
4. **Tick while idle:** if the door is open, close it. Otherwise, if the current floor is a stop, remove it and open the door.
5. **Tick while out of service:** nothing happens.
6. **Service:**
   - `take_out_of_service` works only when idle, with the door closed and no stops.
   - `return_to_service` works only when out of service, and goes to `IDLE`.
7. `report_load(weight)` sets the current load.

**Constraint:** `Elevator` must not branch on its state. Use one class per state.

### Names the driver uses

- `DoorStatus`: `OPEN`, `CLOSED` · `ElevatorStateType`: `IDLE`, `MOVING_UP`, `MOVING_DOWN`, `OUT_OF_SERVICE`
- `Elevator(min_floor, max_floor, capacity, start_floor)`: `add_stop`, `step`, `report_load`, `take_out_of_service`, `return_to_service`, `get_floor`, `get_door`, `get_state`

### Driver code

```python
def tick(e):
    e.step()
    print(e.get_floor(), e.get_door().name, e.get_state().name)


if __name__ == "__main__":
    e = Elevator(0, 10, 500, 0)
    print(e.add_stop(11))          # False  -> above top floor
    print(e.add_stop(2), e.add_stop(1), e.get_state().name)
                                   # True True MOVING_UP
    tick(e)                        # 1 CLOSED MOVING_UP
    tick(e)                        # 1 OPEN MOVING_UP      -> stops on the way
    tick(e)                        # 1 CLOSED MOVING_UP
    e.report_load(600)
    tick(e)                        # 1 CLOSED MOVING_UP    -> overloaded, does not move
    e.report_load(200)
    tick(e)                        # 2 CLOSED MOVING_UP
    tick(e)                        # 2 OPEN MOVING_UP
    e.add_stop(0)                  # -> new call below while the door is open
    tick(e)                        # 2 CLOSED MOVING_UP
    tick(e)                        # 2 CLOSED MOVING_DOWN  -> nothing above, reverse
    tick(e)                        # 1 CLOSED MOVING_DOWN
    tick(e)                        # 0 CLOSED MOVING_DOWN
    tick(e)                        # 0 OPEN MOVING_DOWN
    tick(e)                        # 0 CLOSED IDLE         -> no stops left

    print(e.take_out_of_service()) # True
    print(e.add_stop(5))           # False  -> out of service
    tick(e)                        # 0 CLOSED OUT_OF_SERVICE
    print(e.take_out_of_service()) # False  -> already out
    print(e.return_to_service(), e.get_state().name)
                                   # True IDLE
    e.add_stop(4)
    print(e.take_out_of_service()) # False  -> has stops

    e2 = Elevator(0, 10, 500, 5)
    print(e2.add_stop(5), e2.get_state().name)
                                   # True IDLE
    tick(e2)                       # 5 OPEN IDLE           -> call at its own floor
    tick(e2)                       # 5 CLOSED IDLE
```


---

## P22 · Event Fan-out for Notifications — Observer

**Pattern:** Observer · **Difficulty:** Easy · **Time:** ~15 min · **Source:** `#13 · Notification Service` (requirement 6)

### Problem

When something happens in the platform, **several interested parties must react automatically**. The code that raises the event must not know who is listening.

1. An event has a name and a list of recipient ids.
2. Listeners subscribe to the publisher. Subscribing the same listener twice is refused. Unsubscribing a listener that isn't subscribed is refused.
3. `publish(event)` calls every subscribed listener **in subscription order** and returns how many handled it without raising.
4. **A crashing listener must not stop the others.**
5. **Listeners:**
   - The notification listener prints `notify <recipient>: <event_name>` for each recipient.
   - The audit listener keeps a list of the event names it has seen.

### Names the driver uses

- `Event(event_name, recipient_ids)`
- `EventListener` (abstract, with `on_event(event)`): the driver subclasses it
- `NotificationListener()`, `AuditListener()` with `get_log()`
- `EventPublisher()`: `subscribe`, `unsubscribe`, `publish(event) -> int`

### Driver code

```python
class FailingListener(EventListener):
    def on_event(self, event):
        raise RuntimeError("listener crashed")


if __name__ == "__main__":
    pub = EventPublisher()
    notifier, audit, broken = NotificationListener(), AuditListener(), FailingListener()
    print(pub.subscribe(notifier), pub.subscribe(audit))     # True True
    print(pub.subscribe(audit))                              # False  -> already subscribed

    print(pub.publish(Event("ORDER_SHIPPED", ["u1", "u2"])))
                                                             # notify u1: ORDER_SHIPPED
                                                             # notify u2: ORDER_SHIPPED
                                                             # 2
    pub.subscribe(broken)
    print(pub.publish(Event("PRICE_DROP", ["u3"])))
                                                             # notify u3: PRICE_DROP
                                                             # 2      -> the crash did not stop the others
    print(pub.unsubscribe(notifier), pub.unsubscribe(notifier))
                                                             # True False
    print(pub.publish(Event("NEW_LOGIN", ["u1"])))           # 1      -> only audit handled it
    print(audit.get_log())                                   # ['ORDER_SHIPPED', 'PRICE_DROP', 'NEW_LOGIN']
```


---

## P23 · Food Order Tracking — Observer

**Pattern:** Observer (+ a record that runs its own lifecycle, M3) · **Difficulty:** Easy–Medium · **Time:** ~20 min · **Source:** `#14 · Food Delivery`

### Problem

An order moves through stages, and **every watcher is told on every change**. The order checks **who** is acting and refuses illegal moves. A refused move notifies no one.

1. **Stages:**
   - `PLACED → ACCEPTED → PREPARING → PICKED_UP → DELIVERED`.
   - `CANCELLED` is allowed only from `PLACED` or `ACCEPTED`.
2. **Who may act:**
   - Only the order's restaurant may `accept` and `start_preparing`.
   - Only its delivery partner may `pick_up` and `deliver`.
   - Only its user may `cancel`.
3. After each successful change, call `on_status_changed(order_id, user_id, new_status)` on every observer.
4. Observers can be added and removed. Removing one that isn't attached is refused.
5. **Observers:**
   - The user notifier keeps messages per user in the form `"<order_id> is <STATUS>"`.
   - The status counter counts how many times each status was reached.

### Names the driver uses

- `OrderStatus`: `PLACED`, `ACCEPTED`, `PREPARING`, `PICKED_UP`, `DELIVERED`, `CANCELLED`
- `UserNotifier()` with `get_messages(user_id)`, `StatusCounter()` with `get_count(status)`
- `Order(order_id, user_id, restaurant_id, partner_id)`: `add_observer`, `remove_observer`, `accept(restaurant_id)`, `start_preparing(restaurant_id)`, `pick_up(partner_id)`, `deliver(partner_id)`, `cancel(user_id)`, `get_status`

### Driver code

```python
if __name__ == "__main__":
    users, counter = UserNotifier(), StatusCounter()
    o1 = Order("O1", "u1", "R1", "P1")
    o1.add_observer(users)
    o1.add_observer(counter)

    print(o1.start_preparing("R1"))        # False  -> must be accepted first
    print(o1.accept("R2"))                 # False  -> wrong restaurant
    print(o1.accept("R1"))                 # True
    print(o1.start_preparing("R1"))        # True
    print(o1.cancel("u1"))                 # False  -> too late to cancel once preparing
    print(o1.pick_up("P9"))                # False  -> wrong partner
    print(o1.pick_up("P1"), o1.deliver("P1"))
                                           # True True
    print(o1.get_status().name)            # DELIVERED
    print(users.get_messages("u1"))        # ['O1 is ACCEPTED', 'O1 is PREPARING', 'O1 is PICKED_UP', 'O1 is DELIVERED']
                                           # -> refused moves sent nothing

    o2 = Order("O2", "u2", "R1", "P2")
    o2.add_observer(users)
    o2.add_observer(counter)
    print(o2.remove_observer(users), o2.remove_observer(users))
                                           # True False
    print(o2.cancel("u1"), o2.cancel("u2"))  # False True  -> only the owner can cancel
    print(users.get_messages("u2"))        # []
    print(counter.get_count(OrderStatus.CANCELLED), counter.get_count(OrderStatus.ACCEPTED))
                                           # 1 1
```


---

## P24 · Library Reservations — Observer

**Pattern:** Observer (+ thing vs. status, M9) · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#15 · Library Management`

### Problem

A member can reserve a book whose copies are all borrowed. When a copy comes back, it goes to the **first member in the waiting list**, and that member is **notified automatically**.

1. A book (the title) owns its physical copies (`<isbn>-1`, `<isbn>-2`, …) and its waiting list. Each copy is `AVAILABLE`, `BORROWED` or `RESERVED`.
2. **Borrowing:** a member gets the copy reserved **for them** first, otherwise any `AVAILABLE` copy, otherwise `None`. A copy reserved for someone else can't be borrowed.
3. **Reserving:**
   - Refused if a copy is available (just borrow it), or if the member is already waiting.
   - The waiting list is first-come, first-served.
   - A member can cancel while they are still waiting.
4. **Returns go through the book:**
   - If someone is waiting, the copy becomes `RESERVED` for the first in line and the observer is told `on_book_available(member_id, title, copy_id)`.
   - Otherwise the copy becomes `AVAILABLE`.
   - Returning a copy that isn't borrowed is refused.
5. The member notifier keeps messages per member in the form `"<title> (<copy_id>) is ready for you"`.

### Names the driver uses

- `CopyStatus`: `AVAILABLE`, `BORROWED`, `RESERVED`
- `MemberNotifier()` with `get_messages(member_id)`
- `LibraryService(observer)`: `add_book(isbn, title, copy_count)`, `borrow(member_id, isbn) -> copy_id or None`, `reserve`, `cancel_reservation`, `return_copy(isbn, copy_id)`, `get_copy_status(isbn, copy_id)`

### Driver code

```python
if __name__ == "__main__":
    notifier = MemberNotifier()
    lib = LibraryService(notifier)
    lib.add_book("B1", "Dune", 1)

    print(lib.reserve("m1", "B1"))            # False  -> a copy is available, just borrow it
    print(lib.borrow("m1", "B1"))             # B1-1
    print(lib.borrow("m2", "B1"))             # None   -> no copy free
    print(lib.reserve("m2", "B1"), lib.reserve("m3", "B1"), lib.reserve("m2", "B1"))
                                              # True True False
    print(lib.return_copy("B1", "B1-1"))      # True   -> handed to m2, the first in line
    print(lib.get_copy_status("B1", "B1-1").name)
                                              # RESERVED
    print(notifier.get_messages("m2"))        # ['Dune (B1-1) is ready for you']
    print(lib.borrow("m3", "B1"))             # None   -> held for m2
    print(lib.borrow("m2", "B1"))             # B1-1

    print(lib.reserve("m4", "B1"), lib.cancel_reservation("m4", "B1"))
                                              # True True
    print(lib.return_copy("B1", "B1-1"))      # True   -> m3 is next, m4 cancelled
    print(notifier.get_messages("m3"), notifier.get_messages("m4"))
                                              # ['Dune (B1-1) is ready for you'] []
    print(lib.borrow("m3", "B1"))             # B1-1
    print(lib.return_copy("B1", "B1-1"), lib.get_copy_status("B1", "B1-1").name)
                                              # True AVAILABLE
    print(lib.return_copy("B1", "B1-1"))      # False  -> not borrowed
```


---

## P25 · Flight Status Alerts — Observer

**Pattern:** Observer · **Difficulty:** Easy–Medium · **Time:** ~20 min · **Source:** `#28 · Airline Management`

### Problem

Passengers and the departure board watch a flight. **Every legal stage change notifies all current watchers.** An illegal change is refused and notifies nobody.

1. **Stages:**
   - `SCHEDULED → DELAYED` (a delay can repeat) `→ DEPARTED → LANDED`.
   - `CANCELLED` is allowed only from `SCHEDULED` or `DELAYED`.
   - Departing is allowed from `SCHEDULED` or `DELAYED`.
2. A delay sets new departure and arrival times (strings like `"10:30"`).
3. Watchers subscribe with a **subscription id**. A duplicate id is refused. Unsubscribing an unknown id is refused.
4. After a change, call `on_flight_status_changed(flight_number, status, departure, arrival)` on every watcher, in subscription order.
5. **Call watchers after releasing the flight's lock**, so a slow or re-entrant watcher can't block the flight. Copy the watcher list and the data inside the lock, then notify outside it.
6. **Watchers:**
   - A passenger subscription prints `to <email>: <flight> <STATUS> <dep>-<arr>`.
   - The departure board remembers the latest status per flight.

### Names the driver uses

- `FlightStatus`: `SCHEDULED`, `DELAYED`, `DEPARTED`, `LANDED`, `CANCELLED`
- `PassengerSubscription(email)`, `DepartureBoard()` with `get_status(flight_number)`
- `Flight(flight_number, departure, arrival)`: `subscribe(observer, subscription_id)`, `unsubscribe(subscription_id)`, `delay(new_departure, new_arrival)`, `depart`, `land`, `cancel`

### Driver code

```python
if __name__ == "__main__":
    board = DepartureBoard()
    f = Flight("AI-101", "10:00", "12:00")
    print(f.subscribe(PassengerSubscription("asha@mail.com"), "s1"))   # True
    print(f.subscribe(PassengerSubscription("ben@mail.com"), "s2"))    # True
    print(f.subscribe(board, "board"))                                  # True
    print(f.subscribe(board, "board"))                                  # False  -> id already used

    print(f.land())                    # False  -> has not departed, nobody notified
    print(f.delay("10:30", "12:30"))   # to asha@mail.com: AI-101 DELAYED 10:30-12:30
                                       # to ben@mail.com: AI-101 DELAYED 10:30-12:30
                                       # True
    print(f.unsubscribe("s2"), f.unsubscribe("s2"))
                                       # True False
    print(f.depart())                  # to asha@mail.com: AI-101 DEPARTED 10:30-12:30
                                       # True
    print(f.cancel())                  # False  -> already departed
    print(f.land())                    # to asha@mail.com: AI-101 LANDED 10:30-12:30
                                       # True
    print(board.get_status("AI-101").name)
                                       # LANDED

    g = Flight("AI-202", "18:00", "20:00")
    g.subscribe(board, "board")
    print(g.cancel(), board.get_status("AI-202").name)
                                       # True CANCELLED
```


---

## P26 · Log Level Handlers — Chain of Responsibility

**Pattern:** Chain of Responsibility · **Difficulty:** Easy · **Time:** ~20 min · **Source:** `#09 · Logging Framework`

### Problem

A log entry is passed **along a chain of handlers**. Each handler deals with **its own level** and passes everything else to the next one. Every level has a different set of destinations: for example, errors also go to an error file and an alert feed.

1. **Use one `LogHandler` class with the level as data**, not one class per level.
2. `set_next(handler)` links the chain and **returns the handler it was given**, so calls can be chained: `a.set_next(b).set_next(c)`.
3. `handle(entry)`: if the entry's level equals this handler's level, write it to all of this handler's destinations and return `True`. Otherwise pass it to the next handler. At the end of the chain, return `False`.
4. **The logger:**
   - Drops entries below its minimum level (`False`) before the chain sees them.
   - Its minimum level can be changed.
   - The head of its chain can be replaced, which is how a new handler is added at the front.
5. **Destinations:**
   - Console prints `console <LEVEL>: <message>`.
   - Memory stores `"<LEVEL>: <message>"`.

### Names the driver uses

- `LogLevel`: `DEBUG`, `INFO`, `WARN`, `ERROR`
- `ConsoleDestination()`, `MemoryDestination()` with `get_lines()`
- `LogHandler(level, destinations)`: `set_next(handler) -> handler`, `handle(entry)`
- `Logger(min_level, head)`: `log(level, message) -> bool`, `set_min_level`, `set_head`

### Driver code

```python
if __name__ == "__main__":
    console, error_file, alerts = ConsoleDestination(), MemoryDestination(), MemoryDestination()
    info = LogHandler(LogLevel.INFO, [console])
    warn = LogHandler(LogLevel.WARN, [console])
    error = LogHandler(LogLevel.ERROR, [console, error_file, alerts])
    info.set_next(warn).set_next(error)
    logger = Logger(LogLevel.INFO, info)

    print(logger.log(LogLevel.DEBUG, "cache miss"))  # False  -> below the logger's minimum
    print(logger.log(LogLevel.INFO, "started"))      # console INFO: started
                                                     # True
    print(logger.log(LogLevel.ERROR, "db down"))     # console ERROR: db down
                                                     # True   -> passed info, passed warn, handled by error
    print(error_file.get_lines(), alerts.get_lines())
                                                     # ['ERROR: db down'] ['ERROR: db down']
    print(logger.log(LogLevel.WARN, "slow query"))   # console WARN: slow query
                                                     # True

    logger.set_min_level(LogLevel.DEBUG)
    print(logger.log(LogLevel.DEBUG, "cache miss"))  # False  -> reached the end of the chain, nobody handled it

    debug_file = MemoryDestination()
    debug = LogHandler(LogLevel.DEBUG, [debug_file])
    debug.set_next(info)
    logger.set_head(debug)
    print(logger.log(LogLevel.DEBUG, "cache miss"))  # True
    print(debug_file.get_lines(), error_file.get_lines())
                                                     # ['DEBUG: cache miss'] ['ERROR: db down']
```


---

## P27 · ATM Note Dispenser — Chain of Responsibility

**Pattern:** Chain of Responsibility · **Difficulty:** Easy–Medium · **Time:** ~20 min · **Source:** `#12 · ATM`

### Problem

An ATM pays out an amount by **passing it through note dispensers, largest note first**. Each dispenser takes as many of its notes as it can and passes the rest along. Nothing may change unless the whole amount can be paid. **Plan first, then take.**

1. Each dispenser holds one note value and a count.
2. `plan(amount)`: this dispenser uses `min(count, amount // value)` notes and passes the remainder to the next one.
   - If the remainder reaches 0, return the plan as `{value: notes}`, including **only values that were used**, ordered largest first.
   - If a remainder is left at the end of the chain, return `None`.
   - Planning **never changes the counts**.
3. `dispense(amount)` plans, and only if the plan succeeds removes the notes and returns the plan. Otherwise it returns `None` and changes nothing. An amount ≤ 0 returns `None`.
4. The chain is built from the given denominations, sorted largest first.
5. `add_notes(value, count)` restocks a known denomination. An unknown value or a count ≤ 0 is refused.
6. **This is greedy on purpose:** 600 with one 500 and only 200s fails, even though 3 × 200 would work. Be ready to say why.

### Names the driver uses

- `CashDispenser({value: count})`: `dispense(amount) -> dict or None`, `get_count(value)`, `add_notes(value, count)`

### Driver code

```python
if __name__ == "__main__":
    cash = CashDispenser({500: 2, 200: 3, 100: 5})

    print(cash.dispense(1800))       # {500: 2, 200: 3, 100: 2}
    print(cash.get_count(500), cash.get_count(200), cash.get_count(100))
                                     # 0 0 3
    print(cash.dispense(450))        # None   -> 50 left that no note can pay
    print(cash.dispense(400))        # None   -> only 300 left
    print(cash.get_count(100))       # 3      -> a failed plan changed nothing
    print(cash.dispense(300))        # {100: 3}

    print(cash.add_notes(500, 1), cash.add_notes(200, 3), cash.add_notes(50, 2))
                                     # True True False
    print(cash.dispense(600))        # None   -> greedy takes 500, then 100 is left with no 100s
    print(cash.get_count(500))       # 1
    print(cash.dispense(400))        # {200: 2}  -> 500 is skipped because it is bigger than 400
    print(cash.dispense(0))          # None
```


---

## P28 · In-Memory File System — Composite

**Pattern:** Composite · **Difficulty:** Medium · **Time:** ~35 min · **Source:** `#20 · In-Memory File System`

### Problem

Files and directories form a tree under one root. **Both are nodes**, so size, delete and move treat them the same way. A directory's size is **computed** from its children, never stored.

1. A file's size is the length of its content. A directory's size is the sum of its children's sizes.
2. **A name is stored only as the key in its parent's map**, not inside the node. Renaming is one key change.
3. **Paths:**
   - Paths are absolute.
   - Normalize `.` (stay) and `..` (go up one level, and stay put at the root) before walking from the root.
   - Don't use parent pointers.
4. **Creating:** creating a file or directory is refused if the parent is missing or not a directory, or the name is taken.
5. **Reading and writing:** `read_file`, `write_file` and `append_file` work only on files. Otherwise they return `None` / `False`.
6. **Listing:** `list_directory` returns child names sorted, with `/` added after directory names. On a non-directory it returns `None`.
7. **Deleting:**
   - Deleting a non-empty directory needs `recursive=True`.
   - The root can't be deleted.
8. **Renaming and moving:**
   - `rename(path, new_name)` is refused if the new name is taken.
   - `move(source, destination_dir)` is refused if the destination is not a directory, already has that name, or **is the source itself or inside it**.
9. `get_size` of an unknown path returns `None`.

### Names the driver uses

- `FileSystem()`: `create_directory`, `create_file(path, content)`, `read_file`, `write_file`, `append_file`, `list_directory`, `get_size`, `delete(path, recursive)`, `rename(path, new_name)`, `move(source, destination_dir)`

### Driver code

```python
if __name__ == "__main__":
    fs = FileSystem()
    fs.create_directory("/a")
    fs.create_directory("/a/b")
    fs.create_file("/a/b/x.txt", "hello")
    fs.create_file("/a/y.txt", "hi!")
    print(fs.create_file("/a/q/z.txt", "x"))      # False  -> parent missing

    print(fs.get_size("/a"), fs.get_size("/"))    # 8 8    -> a directory's size is the sum of its children
    print(fs.list_directory("/a"))                # ['b/', 'y.txt']
    print(fs.append_file("/a/y.txt", "!!"))       # True
    print(fs.read_file("/a/b/../y.txt"))          # hi!!!  -> .. goes up one level
    print(fs.read_file("/a/b"))                   # None   -> b is a directory
    print(fs.write_file("/a/b/x.txt", "hey"), fs.get_size("/a"))
                                                  # True 8

    print(fs.delete("/a/b", False))               # False  -> not empty
    print(fs.rename("/a/y.txt", "z.txt"))         # True
    print(fs.rename("/a/z.txt", "b"))             # False  -> name taken
    print(fs.list_directory("/a"))                # ['b/', 'z.txt']

    fs.create_directory("/c")
    print(fs.move("/a", "/a/b"))                  # False  -> cannot move a directory into itself
    print(fs.move("/a/b", "/c"))                  # True
    print(fs.list_directory("/c"), fs.get_size("/c"), fs.get_size("/"))
                                                  # ['b/'] 3 8

    print(fs.delete("/c", True))                  # True
    print(fs.list_directory("/"), fs.get_size("/"))
                                                  # ['a/'] 5
    print(fs.get_size("/nope"), fs.delete("/", True))
                                                  # None False
```


---

## P29 · Task Scheduler Jobs — Command

**Pattern:** Command (+ version numbers instead of deleting from the heap) · **Difficulty:** Medium–Hard · **Time:** ~35 min · **Source:** `#17 · Task Scheduler`

### Problem

Users queue **actions to run later**. Each action is a command object with one `execute()` method, so the scheduler runs any kind of task without knowing what it does. Jobs can be updated or cancelled after they are queued, and failed jobs are retried.

1. A task's `execute()` returns `True` on success.
   - **Print task:** prints its message and succeeds.
   - **Flaky task:** fails its first `n` runs (prints `<name> failed`), then prints `<name> ok` and succeeds.
2. `schedule(task, run_at, priority)` returns `J1`, `J2`, …. Priorities are `LOW < MEDIUM < HIGH`.
3. `run_due(now)` runs every job whose `run_at ≤ now`, ordered by **run_at, then higher priority, then the earlier job**. It returns how many tasks it executed.
4. **Retry:**
   - On failure, if `attempts < max_attempts`, the job is rescheduled at `now + retry_delay` and stays `SCHEDULED`.
   - Otherwise it becomes `FAILED`.
   - On success it becomes `COMPLETED`.
5. **Update and cancel without searching the heap:**
   - Every change to a job **bumps its version**, and a fresh queue entry is pushed.
   - A popped entry whose version doesn't match the job's current version is stale and is skipped.
6. `update(job_id, run_at)` and `cancel(job_id)` work only on `SCHEDULED` jobs.

### Names the driver uses

- `Priority`: `LOW`, `MEDIUM`, `HIGH` · `JobStatus`: `SCHEDULED`, `COMPLETED`, `FAILED`, `CANCELLED`
- `PrintTask(message)`, `FlakyTask(name, failures)`
- `Scheduler(max_attempts, retry_delay)`: `schedule(task, run_at, priority)`, `update(job_id, run_at)`, `cancel(job_id)`, `run_due(now) -> int`, `get_status(job_id)`, `get_attempts(job_id)`

### Driver code

```python
if __name__ == "__main__":
    s = Scheduler(max_attempts=2, retry_delay=5)
    j1 = s.schedule(PrintTask("send report"), 10, Priority.LOW)
    j2 = s.schedule(PrintTask("charge card"), 10, Priority.HIGH)
    j3 = s.schedule(FlakyTask("sync", 1), 12, Priority.MEDIUM)
    j4 = s.schedule(FlakyTask("upload", 5), 12, Priority.LOW)
    j5 = s.schedule(PrintTask("old reminder"), 11, Priority.LOW)
    print(s.update(j5, 30))            # True   -> moved later, the old queue entry goes stale

    print(s.run_due(12))               # charge card
                                       # send report
                                       # sync failed
                                       # upload failed
                                       # 4
                                       # -> same time: HIGH first, the stale entry for J5 at 11 was skipped
    print(s.get_status(j3).name, s.get_attempts(j3))
                                       # SCHEDULED 1   -> retry planned at 17
    print(s.run_due(16))               # 0
    print(s.run_due(17))               # sync ok
                                       # upload failed
                                       # 2
    print(s.get_status(j3).name, s.get_status(j4).name, s.get_attempts(j4))
                                       # COMPLETED FAILED 2

    print(s.cancel(j5), s.cancel(j5))  # True False
    print(s.run_due(100))              # 0      -> cancelled job never runs
    print(s.update(j1, 50))            # False  -> already completed
    print(s.get_status(j5).name)       # CANCELLED
```


---

## P30 · Log Manager — Singleton

**Pattern:** Singleton (thread-safe) · **Difficulty:** Easy · **Time:** ~15 min · **Source:** `#09 · Logging Framework`

### Problem

The configured logger must be reachable from **everywhere** in the app without passing it around. Make **only the manager a singleton**, not the logger. Many threads may ask for it at the same moment.

1. `LogManager.get_instance()` always returns the same object, even when 20 threads call it at once. Use **double-checked locking**.
2. Calling `LogManager()` directly must also return that same object.
3. `initialize(logger)` succeeds only the **first** time (`True`). Later calls return `False` and keep the first logger. Make the check-and-set safe across threads.
4. `get_logger()` returns `None` before initialization.
5. The logger prints `[<name>] <message>`.

### Names the driver uses

- `Logger(name)` with `log(message)`
- `LogManager`: `get_instance()` (class method), `initialize(logger)`, `get_logger()`

### Driver code

```python
if __name__ == "__main__":
    print(LogManager.get_instance().get_logger())             # None   -> not configured yet

    barrier, seen = threading.Barrier(20), []
    def grab():
        barrier.wait()                                        # -> all 20 threads ask at the same moment
        seen.append(id(LogManager.get_instance()))
    threads = [threading.Thread(target=grab) for _ in range(20)]
    for t in threads: t.start()
    for t in threads: t.join()
    print(len(seen), len(set(seen)))                          # 20 1

    m = LogManager.get_instance()
    print(m.initialize(Logger("app")))                        # True
    print(m.initialize(Logger("other")))                      # False  -> configured once only
    print(LogManager() is m)                                  # True   -> even direct construction gives the same one

    def somewhere_else_in_the_app():
        LogManager.get_instance().get_logger().log("payment done")
    somewhere_else_in_the_app()                               # [app] payment done
```


---

## P31 · Per-Client Rate Limiter — Factory

**Pattern:** Factory (simple factory by type + factory method `create_tracker`) · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#06 · Rate Limiter`

### Problem

Every **(client, endpoint)** pair needs its own fresh usage tracker, and the kind of tracker depends on the algorithm that was configured. The limiter must not know the concrete tracker classes. **It asks the algorithm to create one.** The algorithm itself is created **by type name** from one place.

1. **The type factory:** `AlgorithmFactory.create(kind)` returns an algorithm for `"fixed_window"` or `"token_bucket"`, and `None` for any other kind.
2. **The factory method:** each algorithm's `create_tracker()` returns a **new** tracker of its own kind.
3. **Trackers:** reuse your P05 fixed-window and token-bucket trackers, which use `try_acquire(rule, now)`.
4. **The limiter:**
   - Creates a tracker lazily, the first time it sees a (client, endpoint) pair.
   - Keeps one tracker per pair, so each pair is limited independently.
5. **Rule lookup** (most specific first):
   1. `(client, endpoint)`
   2. `(client, any endpoint)`
   3. `(any client, endpoint)`
   4. The default rule.
   - `set_rule(None, ...)` or `set_rule(..., None)` means "any".

### Names the driver uses

- `RateLimitRule(limit, window_seconds)`
- `AlgorithmFactory.create(kind)` (static method)
- `RuleRegistry()`: `set_default_rule(rule)`, `set_rule(client_id or None, endpoint or None, rule)`
- `RateLimiter(algorithm, registry)`: `allow(client_id, endpoint, now)`, `get_tracker_count()`, `get_tracker(client_id, endpoint)`
- Tracker class names printed: `FixedWindowTracker`, `TokenBucketTracker`

### Driver code

```python
if __name__ == "__main__":
    reg = RuleRegistry()
    reg.set_default_rule(RateLimitRule(2, 10))
    reg.set_rule("vip", None, RateLimitRule(5, 10))
    reg.set_rule(None, "/login", RateLimitRule(1, 10))
    reg.set_rule("vip", "/login", RateLimitRule(3, 10))

    print(AlgorithmFactory.create("leaky_bucket"))                    # None   -> unknown type
    limiter = RateLimiter(AlgorithmFactory.create("fixed_window"), reg)
    print([limiter.allow("bob", "/search", 0) for _ in range(3)])     # [True, True, False]   -> default rule
    print([limiter.allow("alice", "/search", 0) for _ in range(3)])   # [True, True, False]   -> own tracker
    print([limiter.allow("bob", "/login", 0) for _ in range(2)])      # [True, False]   -> endpoint rule
    print([limiter.allow("vip", "/search", 0) for _ in range(6)])
                                                                      # [True, True, True, True, True, False]
    print([limiter.allow("vip", "/login", 0) for _ in range(4)])      # [True, True, True, False]
                                                                      # -> client plus endpoint rule wins
    print(limiter.get_tracker_count())                                # 5
    print(type(limiter.get_tracker("bob", "/login")).__name__)        # FixedWindowTracker

    tb = RateLimiter(AlgorithmFactory.create("token_bucket"), reg)
    print([tb.allow("bob", "/search", t) for t in [0, 0, 0, 5]])      # [True, True, False, True]
    print(type(tb.get_tracker("bob", "/search")).__name__)            # TokenBucketTracker
```


---

## P32 · Async Notification Delivery — Queue + Worker (with retry)

**Pattern:** Queue + Worker (fast half queues, slow half works, M7) · **Difficulty:** Medium · **Time:** ~35 min · **Source:** `#13 · Notification Service`

### Problem

Sending must **not block the caller**. The fast half records the notification, creates **one delivery per channel**, queues them and returns at once. The slow half is a worker that takes one ready delivery at a time, sends it, records the result and retries with backoff. Time is passed in, so tests are deterministic.

1. `send(user_id, priority, message, channel_types)` returns `N1`, `N2`, … **without sending anything**. It skips channel types that have no channel.
2. Each delivery has its own status: `PENDING → DELIVERED`, or `RETRYING → … → DELIVERED / FAILED`.
3. **The queue:**
   - `take_ready(now)` returns a delivery whose `next_attempt_at ≤ now`.
   - Pick the **highest priority** first, then the earliest `next_attempt_at`, then the earliest put into the queue.
   - Return `None` if nothing is ready.
4. **The worker:**
   - `process_next(now)` takes one ready delivery and sends it.
   - On failure, the retry policy gives the next delay: `base × 2^(attempts−1)`, and `None` once `attempts ≥ max_attempts`.
   - With a delay, the delivery becomes `RETRYING` and is re-queued at `now + delay`. With `None`, it becomes `FAILED`.
   - It returns `False` if nothing was ready.
5. **A notification's overall status is computed** from its deliveries:
   - `DELIVERED` if all are delivered.
   - `PENDING` if any is still pending or retrying.
   - Otherwise `FAILED`.
6. **Test channel:** a scripted channel returns its list of results in order (then `True`), and prints `<NAME> to <user>: <message> [ok|fail]`.

### Names the driver uses

- `ChannelType`: `EMAIL`, `SMS`, `PUSH` · `Priority`: `LOW`, `MEDIUM`, `HIGH` · `DeliveryStatus`: `PENDING`, `RETRYING`, `DELIVERED`, `FAILED`
- `ScriptedChannel(name, results)`, `RetryPolicy(max_attempts, base_delay)`, `DeliveryQueue()` with `size()`
- `NotificationService(channels_dict, queue)`: `send`, `get_statuses(nid) -> {ChannelType: DeliveryStatus}`, `get_overall_status(nid)`
- `DeliveryWorker(queue, channels_dict, policy)`: `process_next(now) -> bool`

### Driver code

```python
def names(statuses):
    return {k.name: v.name for k, v in statuses.items()}


if __name__ == "__main__":
    channels = {
        ChannelType.EMAIL: ScriptedChannel("EMAIL", [True, True]),
        ChannelType.SMS: ScriptedChannel("SMS", [False, False, True]),
        ChannelType.PUSH: ScriptedChannel("PUSH", [False, False, False]),
    }
    q = DeliveryQueue()
    svc = NotificationService(channels, q)
    worker = DeliveryWorker(q, channels, RetryPolicy(3, 10))

    n1 = svc.send("u1", Priority.LOW, "weekly digest", [ChannelType.EMAIL])
    n2 = svc.send("u2", Priority.HIGH, "OTP 4821", [ChannelType.EMAIL, ChannelType.SMS])
    print(n1, n2, q.size())                   # N1 N2 3   -> returned at once, nothing sent yet
    print(names(svc.get_statuses(n2)))        # {'EMAIL': 'PENDING', 'SMS': 'PENDING'}

    worker.process_next(0)                    # EMAIL to u2: OTP 4821 [ok]      -> HIGH goes first
    worker.process_next(0)                    # SMS to u2: OTP 4821 [fail]      -> retry at 0 + 10
    worker.process_next(0)                    # EMAIL to u1: weekly digest [ok]
    print(worker.process_next(5))             # False  -> the retry is not due yet
    worker.process_next(10)                   # SMS to u2: OTP 4821 [fail]      -> retry at 10 + 20
    print(names(svc.get_statuses(n2)), svc.get_overall_status(n2).name)
                                              # {'EMAIL': 'DELIVERED', 'SMS': 'RETRYING'} PENDING
    worker.process_next(30)                   # SMS to u2: OTP 4821 [ok]
    print(svc.get_overall_status(n2).name)    # DELIVERED

    n3 = svc.send("u3", Priority.MEDIUM, "flash sale", [ChannelType.PUSH])
    for t in [30, 40, 60]:
        worker.process_next(t)
                                              # PUSH to u3: flash sale [fail]
                                              # PUSH to u3: flash sale [fail]
                                              # PUSH to u3: flash sale [fail]
    print(names(svc.get_statuses(n3)), svc.get_overall_status(n3).name)
                                              # {'PUSH': 'FAILED'} FAILED
    print(worker.process_next(1000), q.size())  # False 0
```


---

## P33 · Scheduler Worker Pool — Queue + Worker Pool (threads)

**Pattern:** Queue + Worker Pool · **Difficulty:** Medium–Hard · **Time:** ~35 min · **Source:** `#17 · Task Scheduler`

### Problem

Run due jobs **in parallel** on a pool of worker threads. Many workers pull from one scheduler, so **each due job must run exactly once**. A crashing task must not kill its worker. The time comes from an injected clock.

1. `schedule(task, run_at)` returns `J1`, `J2`, … and can be called while workers are running.
2. `take_due(now)` **atomically** pops the earliest due job that is still `SCHEDULED`, marks it `RUNNING` and returns its id. It returns `None` if nothing is due.
3. A worker loops:
   - Take a due job.
   - **Run it with no lock held.** An exception counts as a failure.
   - Mark it `COMPLETED` or `FAILED`.
   - If nothing is due, wait briefly on a condition variable (don't busy-spin), then check again.
4. `WorkerPool.start()` starts `n` threads once. A second call returns `False`.
5. `shutdown()` waits until **no due job is waiting or running**, then stops every worker and joins them. Jobs that are not due yet stay `SCHEDULED`.
6. **Task types:**
   - The count task increments a thread-safe counter.
   - The sleep task sleeps.
   - The failing task raises.
   - The fixed clock always returns the same time.

### Names the driver uses

- `JobStatus`: `SCHEDULED`, `RUNNING`, `COMPLETED`, `FAILED`
- `Counter()` with `.value`, `CountTask(counter)`, `SleepTask(seconds)`, `FailingTask()`, `FixedClock(t)`
- `Scheduler()`: `schedule(task, run_at)`, `get_status(job_id)`
- `WorkerPool(scheduler, n, clock)`: `start() -> bool`, `shutdown()`

### Driver code

```python
if __name__ == "__main__":
    counter = Counter()
    s = Scheduler()
    for _ in range(200):
        s.schedule(CountTask(counter), 0)
    bad = s.schedule(FailingTask(), 0)
    later = s.schedule(CountTask(counter), 1_000_000)

    pool = WorkerPool(s, 4, FixedClock(5))
    print(pool.start(), pool.start())          # True False  -> already started
    pool.shutdown()                            # -> waits for every due job, then stops the workers
    print(counter.value)                       # 200    -> each due job ran exactly once
    print(s.get_status(bad).name)              # FAILED -> a crashing task does not kill its worker
    print(s.get_status(later).name)            # SCHEDULED  -> not due yet
    print(s.get_status("J1").name)             # COMPLETED

    s2 = Scheduler()
    for _ in range(8):
        s2.schedule(SleepTask(0.1), 0)
    pool2 = WorkerPool(s2, 4, FixedClock(0))
    start = time.monotonic()
    pool2.start()
    pool2.shutdown()
    print(time.monotonic() - start < 0.6)      # True   -> 4 workers ran them in parallel (one by one is 0.8s)
```


---

## P34 · Multi-threaded Web Crawler — Queue + Worker Pool (threads)

**Pattern:** Queue + Worker Pool · **Difficulty:** Medium–Hard · **Time:** ~30 min · **Source:** `#23 · Web Crawler`

### Problem

Crawl with **several worker threads** sharing one frontier queue. Two threads can find the same link at the same moment, and **each URL must still be fetched at most once**. The crawl must also know when it is finished, which is when the queue is empty **and** no worker is mid-fetch.

1. **Claim a URL in `seen` (under a lock) before queueing it.** Never check-then-add without the lock.
2. Seeds are depth 0. Links found on a depth-`d` page are depth `d+1`. Skip anything deeper than `max_depth`.
3. A worker loops:
   - Take `(url, depth)` from the queue and fetch it **with no lock held**.
   - Record the URL as fetched (or as failed if the fetch returned `None`).
   - Claim and queue every link on the page.
4. **Termination:**
   - `crawl(seeds)` starts the workers, queues the seeds, and **waits until all queued work is done**. A worker must queue the new links *before* it marks its own item done.
   - Then stop every worker (one poison pill each) and join them.
5. `crawl` returns the fetched URLs **sorted**. `get_failed()` returns the failed URLs sorted.
6. Links are the words in the page text that start with `http://`. The driver provides the fake fetcher.

### Names the driver uses

- `CrawlerWorkerPool(fetcher, max_depth, workers)`: `crawl(seeds) -> list[str]`, `get_failed() -> list[str]`
- The driver's `CountingFetcher` provides `fetch(url) -> str or None`

### Driver code

```python
import time


class CountingFetcher:
    """Fake web: page i links to pages 2i+1 and 2i+2, and back to p0. p2 also links to a missing page."""
    def __init__(self, delay):
        self.delay, self.calls, self.lock = delay, {}, threading.Lock()
        self.pages = {}
        for i in range(100):
            links = [f"http://site.com/p{2 * i + 1}", f"http://site.com/p{2 * i + 2}", "http://site.com/p0"]
            if i == 2: links.append("http://site.com/p999")
            self.pages[f"http://site.com/p{i}"] = "page " + " ".join(links)
    def fetch(self, url):
        with self.lock: self.calls[url] = self.calls.get(url, 0) + 1
        time.sleep(self.delay)
        return self.pages.get(url)


if __name__ == "__main__":
    fetcher = CountingFetcher(delay=0.05)
    pool = CrawlerWorkerPool(fetcher, max_depth=3, workers=4)
    start = time.monotonic()
    urls = pool.crawl(["http://site.com/p0"])
    elapsed = time.monotonic() - start

    print(len(urls))                           # 15     -> depth 0..3 of the tree: 1 + 2 + 4 + 8
    print(urls[:3])                            # ['http://site.com/p0', 'http://site.com/p1', 'http://site.com/p10']
    print(pool.get_failed())                   # ['http://site.com/p999']
    print(max(fetcher.calls.values()))         # 1      -> no URL fetched twice, even with 4 threads
    print(sum(fetcher.calls.values()))         # 16     -> 15 pages + the missing one
    print(elapsed < 0.6)                       # True   -> one at a time would take about 0.8s
```


---

## P35 · Online Code Judge — Queue + Worker Pool (threads)

**Pattern:** Queue + Worker Pool · **Difficulty:** Medium · **Time:** ~35 min · **Source:** `#26 · Online Code Judge`

### Problem

Users submit solutions. **Submitting only records and queues the submission, then returns an id at once.** Worker threads judge the submissions in the background. **Every submission runs in its own fresh sandbox**, which is always destroyed, even when the code crashes.

1. `submit(user_id, problem_id, source)` returns `S1`, `S2`, …, or `None` for an unknown problem. The status starts as `QUEUED`.
2. A problem has test cases `(input, expected_output)`. A submission's source must define `solve(input) -> str`.
3. **Judging one submission** (status `RUNNING`, then `JUDGED`):
   1. Create a fresh sandbox, which here is a fresh namespace `dict`.
   2. Compile the source. A `SyntaxError` gives `COMPILE_ERROR`.
   3. Run it and call `solve` on each test, in order. Any exception gives `RUNTIME_ERROR`, and the first mismatch gives `WRONG_ANSWER`. Otherwise the verdict is `ACCEPTED`.
   4. Record how many tests passed before stopping.
   5. **Destroy the sandbox in a `finally`.**
4. **Isolation:** a global set by one submission must not be visible to another.
5. The sandbox factory counts sandboxes that are live and sandboxes created. `get_sandbox_stats()` returns `(live, created)`.
6. **Workers:** the pool's workers take ids from the queue and judge them. `stop()` waits until the queue is drained, then stops and joins every worker.
7. `get_history(user_id)` returns that user's submission ids in order.

### Names the driver uses

- `Verdict`: `ACCEPTED`, `WRONG_ANSWER`, `RUNTIME_ERROR`, `COMPILE_ERROR` · `SubmissionStatus`: `QUEUED`, `RUNNING`, `JUDGED`
- `Problem(problem_id, tests)`
- `JudgeService(problems)`: `submit`, `get_submission(sid)`, `get_history(user_id)`, `get_sandbox_stats()`
- Submission: `get_status()`, `get_verdict()`, `get_passed()`
- `JudgeWorkerPool(service, n)`: `start()`, `stop()`

### Driver code

```python
GOOD = "def solve(s):\n    a, b = map(int, s.split())\n    return str(a + b)\n"
WRONG = "def solve(s):\n    return '3'\n"
CRASH = "def solve(s):\n    return str(1 // 0)\n"
BROKEN = "def solve(s) return 1\n"
SETS_GLOBAL = "SECRET = 42\n" + GOOD
READS_GLOBAL = "def solve(s):\n    return str(SECRET)\n"

if __name__ == "__main__":
    svc = JudgeService([Problem("sum", [("1 2", "3"), ("5 7", "12")])])
    ids = [svc.submit("u1", "sum", GOOD), svc.submit("u1", "sum", WRONG), svc.submit("u2", "sum", CRASH),
           svc.submit("u1", "sum", BROKEN), svc.submit("u3", "sum", SETS_GLOBAL), svc.submit("u2", "sum", READS_GLOBAL)]
    print(ids)                                          # ['S1', 'S2', 'S3', 'S4', 'S5', 'S6']
    print(svc.submit("u1", "nope", GOOD))               # None   -> unknown problem
    print(svc.get_submission("S1").get_status().name)   # QUEUED -> submit returns at once, nothing judged yet

    pool = JudgeWorkerPool(svc, 3)
    pool.start()
    pool.stop()                                         # -> waits until the queue is drained
    for sid in ids:
        sub = svc.get_submission(sid)
        print(sid, sub.get_status().name, sub.get_verdict().name, sub.get_passed())
                                                        # S1 JUDGED ACCEPTED 2
                                                        # S2 JUDGED WRONG_ANSWER 1
                                                        # S3 JUDGED RUNTIME_ERROR 0
                                                        # S4 JUDGED COMPILE_ERROR 0
                                                        # S5 JUDGED ACCEPTED 2
                                                        # S6 JUDGED RUNTIME_ERROR 0
                                                        # -> S6 cannot see S5's SECRET, every run has its own sandbox
    print(svc.get_history("u2"))                        # ['S3', 'S6']
    print(svc.get_sandbox_stats())                      # (0, 6) -> 6 sandboxes created, all destroyed
```


---

## P36 · Webhook Retries & Dead Letters — Queue + Worker

**Pattern:** Queue + Worker (classify, retry with backoff, dead-letter list) · **Difficulty:** Medium · **Time:** ~35 min · **Source:** `#29 · Webhook Delivery`

### Problem

Customers register webhook endpoints for event types. **Publishing an event only creates and queues one delivery per subscribed endpoint.** A worker step sends one due delivery, **sorts the answer into success, retry or give up**, and records every attempt. Deliveries that give up go to a **dead-letter list**, and they can be replayed from there. Time is in ms and is passed in.

1. `register_endpoint(url, event_types)` returns `E1`, `E2`, …. `publish(event_type, payload, now_ms)` returns an event id (`evt_1`, …) and creates deliveries `D1`, `D2`, … that are due at `now_ms`. It creates none if nobody subscribes.
2. **Classifying the response code:**
   - `2xx` → success.
   - **Timeout (`None`), `429` or `5xx`** → retryable.
   - Anything else → permanent.
3. `process_next(now_ms)` takes the due delivery with the earliest due time (ties go to the lower delivery number), posts it and records the code.
   - Success gives `SUCCEEDED`.
   - Permanent gives `DEAD`.
   - Retryable: if the retry policy returns a delay (`base × 2^(attempts−1)`, and `None` once `attempts ≥ max`), the status is `RETRYING` and the delivery is due again at `now + delay`. Otherwise it is `DEAD`.
   - It returns `False` if nothing is due.
4. `list_dead_letters()` returns dead delivery ids in creation order.
5. `replay(delivery_id, now_ms)` works only on `DEAD` deliveries. It sets them back to `PENDING`, due now, and **keeps the attempt history**.
6. Every delivery of one event carries the **same event id**, so receivers can drop duplicates (at-least-once delivery).
7. **Test sender:** a scripted sender returns each URL's list of codes in order, then `200`.

> P38 adds fair per-endpoint lanes, and P39 adds the circuit breaker.

### Names the driver uses

- `DeliveryStatus`: `PENDING`, `RETRYING`, `SUCCEEDED`, `DEAD`
- `ScriptedSender({url: [codes]})`, `RetryPolicy(max_attempts, base_delay_ms)`
- `WebhookService(sender, policy)`: `register_endpoint`, `publish`, `queue_size`, `process_next(now_ms)`, `get_delivery(id)`, `list_dead_letters()`, `replay(id, now_ms)`
- Delivery: `get_status()`, `get_attempt_codes()`, `get_event_id()`

### Driver code

```python
if __name__ == "__main__":
    sender = ScriptedSender({
        "https://a.example/hook": [500, 429, 200],
        "https://b.example/hook": [400],
        "https://c.example/hook": [None, None, None],     # -> None means the request timed out
    })
    svc = WebhookService(sender, RetryPolicy(3, 1000))
    svc.register_endpoint("https://a.example/hook", ["order.paid"])
    svc.register_endpoint("https://b.example/hook", ["order.paid", "order.refunded"])
    svc.register_endpoint("https://c.example/hook", ["order.paid"])

    print(svc.publish("order.paid", '{"id": 7}', 0))    # evt_1
    print(svc.publish("user.created", '{"id": 9}', 0))  # evt_2
    print(svc.queue_size())                             # 3      -> one delivery per subscribed endpoint, nobody wants user.created

    print([svc.process_next(0) for _ in range(4)])      # [True, True, True, False]
                                                        # -> a gets 500 (retry at 1000), b gets 400 (dead), c times out (retry at 1000)
    print(svc.process_next(999))                        # False  -> retries not due yet
    print([svc.process_next(1000) for _ in range(2)])   # [True, True]   -> next retries at 3000
    print([svc.process_next(3000) for _ in range(2)])   # [True, True]
    for did in ["D1", "D2", "D3"]:
        d = svc.get_delivery(did)
        print(did, d.get_status().name, d.get_attempt_codes())
                                                        # D1 SUCCEEDED [500, 429, 200]
                                                        # D2 DEAD [400]
                                                        # D3 DEAD [None, None, None]
    print(svc.list_dead_letters())                      # ['D2', 'D3']

    print(svc.replay("D1", 5000))                       # False  -> only dead deliveries can be replayed
    print(svc.replay("D3", 5000), svc.process_next(5000))
                                                        # True True
    print(svc.get_delivery("D3").get_status().name, svc.get_delivery("D3").get_attempt_codes())
                                                        # SUCCEEDED [None, None, None, 200]
    print(svc.list_dead_letters())                      # ['D2']
    print(svc.get_delivery("D1").get_event_id() == svc.get_delivery("D3").get_event_id())
                                                        # True   -> same event id, so receivers can drop duplicates
```


---

## P37 · Bounded Queue + Pub-Sub Broker — Producer–Consumer / Pub-Sub

**Pattern:** Producer–Consumer (monitor: lock + condition variables) and log-based Pub-Sub · **Difficulty:** Medium–Hard · **Time:** ~40 min · **Source:** `#19 · Bounded Queue + Pub-Sub`

### Problem

**Part A: bounded blocking queue.** It holds at most `capacity` items, first in first out. **Each item goes to exactly one consumer.**

1. `put(item, timeout=None)` waits while the queue is full. It returns `False` if the timeout runs out or the queue is closed.
2. `take(timeout=None)` waits while the queue is empty. It returns `None` on timeout, or when the queue is closed **and** empty. Items still in the queue are drained first, even after `close`.
3. `close()` wakes every waiting thread. A `put` after `close` returns `False`.
4. Use **`while` (not `if`) around `wait()`**, and two conditions (`not_full`, `not_empty`) that share one lock.

**Part B: pub-sub broker (Kafka-style).** Each topic keeps **one ordered log**, and each subscriber keeps only **its own offset**. **Every subscriber gets every message** published after it subscribed, at its own pace.

1. `create_topic` refuses a duplicate name. `publish` returns the message's offset (0, 1, 2, …), or `None` for an unknown topic.
2. `subscribe` starts the subscriber at the **next** offset, so it doesn't see old messages. A duplicate subscriber id is refused, and so is unsubscribing an id that isn't subscribed.
3. `poll(topic, subscriber)` returns the subscriber's next message and advances its offset. It returns `None` if the subscriber is caught up or unknown.
4. **Trim the log:** drop every message that every current subscriber has already read. With no subscribers, nothing is kept.
5. Hand out offsets under the topic's lock.

### Names the driver uses

- `BoundedBlockingQueue(capacity)`: `put(item, timeout=None)`, `take(timeout=None)`, `close()`
- `Broker()`: `create_topic`, `publish(topic, payload)`, `subscribe(topic, subscriber_id)`, `unsubscribe`, `poll(topic, subscriber_id) -> Message or None`, `log_size(topic)`
- Message: `get_offset()`, `get_payload()`

### Driver code

```python
def show(m):
    print(None if m is None else f"{m.get_offset()} {m.get_payload()}")


if __name__ == "__main__":
    # -- Part A: bounded blocking queue --
    q = BoundedBlockingQueue(1)
    print(q.put("a"), q.put("b", timeout=0.05))      # True False  -> full, timed out
    print(q.take(), q.take(timeout=0.05))            # a None      -> empty, timed out

    q = BoundedBlockingQueue(5)
    got, got_lock = [], threading.Lock()
    def producer(p):
        for i in range(100): q.put((p, i))
    def consumer():
        while True:
            item = q.take()
            if item is None: return
            with got_lock: got.append(item)
    producers = [threading.Thread(target=producer, args=(p,)) for p in range(3)]
    consumers = [threading.Thread(target=consumer) for _ in range(3)]
    for t in producers + consumers: t.start()
    for t in producers: t.join()
    q.close()                                        # -> consumers drain what is left, then get None
    for t in consumers: t.join()
    print(len(got), len(set(got)))                   # 300 300  -> nothing lost, nothing duplicated
    print(q.put("late"))                             # False    -> closed

    # -- Part B: pub-sub with a log and one offset per subscriber --
    b = Broker()
    print(b.create_topic("orders"), b.create_topic("orders"))
                                                     # True False
    print(b.publish("orders", "o1"))                 # 0
    b.subscribe("orders", "s1")                      # -> s1 starts after o1
    print(b.publish("orders", "o2"), b.publish("orders", "o3"))
                                                     # 1 2
    b.subscribe("orders", "s2")
    print(b.publish("orders", "o4"))                 # 3
    show(b.poll("orders", "s1"))                     # 1 o2
    show(b.poll("orders", "s1"))                     # 2 o3
    show(b.poll("orders", "s2"))                     # 3 o4
    show(b.poll("orders", "s2"))                     # None   -> s2 is caught up
    print(b.log_size("orders"))                      # 1      -> only o4 is still needed (by s1)
    show(b.poll("orders", "s1"))                     # 3 o4
    print(b.log_size("orders"))                      # 0
    print(b.unsubscribe("orders", "s1"), b.unsubscribe("orders", "s1"))
                                                     # True False
    print(b.publish("nope", "x"), b.poll("orders", "ghost"))
                                                     # None None
```


---

## P38 · Webhook Fan-out with Fair Lanes — Pub-Sub fan-out

**Pattern:** Pub-Sub (fan-out) + fair queue · **Difficulty:** Medium · **Time:** ~30 min · **Source:** `#29 · Webhook Delivery`

### Problem

One published event **fans out** to every active endpoint subscribed to its type. One customer's slow endpoint with a big backlog **must not delay other customers**, so the queue keeps **one lane per endpoint**, serves the lanes **round-robin**, and caps the deliveries **in flight per endpoint**.

1. `register_endpoint(customer_id, event_types)` returns `E1`, `E2`, …. `disable_endpoint` refuses an endpoint that is unknown or already disabled.
2. `publish(event_type)` creates one delivery (`D1`, `D2`, …, numbered across all events) for each **active** endpoint subscribed to that type, in endpoint order. It returns the list of new ids.
3. **The fair queue:**
   - `add(endpoint_id, delivery_id)` appends to that endpoint's lane. A new lane is created on its first delivery.
   - `take()` visits lanes **round-robin in lane-creation order**, starting **after the lane served last**. It returns `(endpoint_id, delivery_id)` from the first lane that has work **and** is under its in-flight limit.
   - `take()` returns `None` if no lane qualifies.
   - `done(endpoint_id)` marks one delivery of that endpoint finished, which frees a slot.
   - `size()` is the number of waiting deliveries.

### Names the driver uses

- `WebhookHub(max_in_flight_per_endpoint)`: `register_endpoint(customer_id, event_types)`, `disable_endpoint(endpoint_id)`, `publish(event_type) -> list[str]`, attribute `queue`
- The queue: `take() -> (endpoint_id, delivery_id) or None`, `done(endpoint_id)`, `size()`

### Driver code

```python
if __name__ == "__main__":
    hub = WebhookHub(max_in_flight_per_endpoint=1)
    hub.register_endpoint("acme", ["order.paid"])                          # -> E1
    hub.register_endpoint("beta", ["order.paid", "order.refunded"])        # -> E2
    hub.register_endpoint("gamma", ["order.refunded"])                     # -> E3

    print(hub.publish("order.paid"))         # ['D1', 'D2']
    print(hub.publish("order.paid"))         # ['D3', 'D4']
    print(hub.publish("order.paid"))         # ['D5', 'D6']
    print(hub.publish("order.refunded"))     # ['D7', 'D8']
    print(hub.disable_endpoint("E3"), hub.disable_endpoint("E3"))
                                             # True False
    print(hub.publish("order.refunded"))     # ['D9']   -> disabled endpoint gets nothing
    print(hub.publish("user.created"))       # []
    print(hub.queue.size())                  # 9

    q = hub.queue
    print(q.take())                          # ('E1', 'D1')
    print(q.take())                          # ('E2', 'D2')
    print(q.take())                          # ('E3', 'D8')
    print(q.take())                          # None     -> every endpoint has 1 in flight
    q.done("E2")
    print(q.take())                          # ('E2', 'D4')   -> E1 is still busy, so E2 goes
    q.done("E1")
    q.done("E3")
    print(q.take())                          # ('E1', 'D3')   -> round-robin moves on past E2
    print(q.take())                          # None     -> E3 empty, E1 and E2 busy
    q.done("E2")
    q.done("E1")
    print([q.take(), q.take(), q.take()])    # [('E2', 'D6'), ('E1', 'D5'), None]
                                             # -> beta's backlog never starves acme
```


---

## P39 · Failing Endpoint Breaker — Circuit Breaker

**Pattern:** Circuit Breaker · **Difficulty:** Medium · **Time:** ~25 min · **Source:** `#29 · Webhook Delivery` (requirement 9)

### Problem

An endpoint that **keeps failing** should stop being called for a while, so it gets time to recover and doesn't waste workers. After a cooldown, **one trial call** decides whether to resume. Time (ms) is passed in.

1. **States:** `CLOSED` (normal) → `OPEN` (refuse calls) → `HALF_OPEN` (one trial) → `CLOSED` or back to `OPEN`.
2. `CLOSED`: call the function. A success resets the failure count to 0. A failure adds 1, and when the count reaches `threshold` the breaker goes `OPEN` and remembers when.
3. `OPEN`: if `now − opened_at < cooldown`, **don't call the function**, and raise `CircuitOpenError`. Otherwise move to `HALF_OPEN` and allow the call.
4. `HALF_OPEN`: a success goes to `CLOSED` (count reset). A failure goes straight back to `OPEN` with `opened_at = now`.
5. `call(fn, now_ms)` returns `fn()`'s result, or re-raises `fn`'s exception after recording it.
6. Guard state changes with a lock, but **don't hold the lock while `fn` runs**.

### Names the driver uses

- `CircuitState`: `CLOSED`, `OPEN`, `HALF_OPEN` · `CircuitOpenError` (an `Exception` subclass)
- `CircuitBreaker(threshold, cooldown_ms)`: `call(fn, now_ms)`, `get_state()`, `get_failure_count()`

### Driver code

```python
if __name__ == "__main__":
    healthy, calls = False, [0]
    def endpoint():
        calls[0] += 1
        if not healthy: raise IOError("endpoint down")
        return "ok"

    cb = CircuitBreaker(threshold=3, cooldown_ms=1000)
    def attempt(now):
        try:
            return cb.call(endpoint, now)
        except CircuitOpenError:
            return "rejected"
        except IOError:
            return "failed"

    print([attempt(t) for t in [0, 10, 20]])   # ['failed', 'failed', 'failed']
    print(cb.get_state().name)                 # OPEN   -> 3 failures in a row
    print(attempt(500), calls[0])              # rejected 3   -> endpoint not even called
    print(attempt(1020), cb.get_state().name)  # failed OPEN  -> cooldown over, the one trial failed, open again
    print(attempt(1500))                       # rejected     -> new cooldown counts from 1020
    healthy = True
    print(attempt(2020), cb.get_state().name)  # ok CLOSED    -> trial succeeded
    print(calls[0])                            # 5

    healthy = False
    attempt(3000)
    attempt(3010)
    healthy = True
    attempt(3020)                              # -> a success resets the count
    healthy = False
    print(attempt(3030), cb.get_state().name, cb.get_failure_count())
                                               # failed CLOSED 1
```


---

## P40 · Chess Move Generation — Polymorphism via inheritance

**Pattern:** Polymorphism (Strategy built into the class hierarchy) · **Difficulty:** Hard · **Time:** ~45 min · **Source:** `#16 · Chess`

### Problem

**Each piece type knows its own moves.** The board never switches on the piece type. Rook, bishop and queen share one **sliding** base class and differ only in their directions. A move is legal only if **your own king is not attacked after making it on a copy of the board**. That single test covers pins and moving into check.

1. Squares are named `a1`…`h8`. White pawns move up (rank +1) and black pawns move down.
2. **Piece kinds:**
   - **Sliding pieces** walk each direction until the edge or the first piece. That square is included, as a capture or as a block.
   - Knight and king use fixed offsets.
3. Every piece lists its **attacked squares**, and separately its **candidate moves**: the attacked squares that are not occupied by its own colour.
4. **The pawn is the exception:**
   - It attacks only diagonally forward.
   - It moves forward 1 onto an empty square, or 2 from its start rank (rank 2 for white, 7 for black) if both squares are empty.
   - It captures diagonally only when an enemy piece is there.
5. **The board is immutable for moves.** `with_move(from, to)` returns a **new** board, which makes the "try it on a copy" check easy. (`place` exists only for setting up.)
6. `legal_moves(square)` returns the legal target squares, sorted, or `[]` for an empty square. `try_move(from, to)` returns the new board, or `None` if the move is illegal.
7. `symbol()` returns `K Q R B N P` for white and the lower-case letter for black.
8. **Out of scope:** castling, en passant, promotion, checkmate and stalemate.

### Names the driver uses

- `Color`: `WHITE`, `BLACK`
- `King`, `Queen`, `Rook`, `Bishop`, `Knight`, `Pawn`, each built as `Piece(color)`, with `symbol()`
- `Board()`: `place(square, piece)`, `get_piece`, `legal_moves(square)`, `try_move(from, to)`, `is_in_check(color)`

### Driver code

```python
if __name__ == "__main__":
    W, B = Color.WHITE, Color.BLACK
    board = Board()
    for sq, piece in [("e1", King(W)), ("e2", Rook(W)), ("b1", Knight(W)), ("c1", Bishop(W)), ("a2", Pawn(W)),
                      ("h8", King(B)), ("e8", Rook(B)), ("b3", Pawn(B))]:
        board.place(sq, piece)

    print(board.legal_moves("b1"))    # ['a3', 'c3', 'd2']
    print(board.legal_moves("a2"))    # ['a3', 'a4', 'b3']   -> forward 1 or 2 from the start, capture diagonally
    print(board.legal_moves("c1"))    # ['a3', 'b2', 'd2', 'e3', 'f4', 'g5', 'h6']
    print(board.legal_moves("e2"))    # ['e3', 'e4', 'e5', 'e6', 'e7', 'e8']
                                      # -> pinned: leaving the e-file would expose the king
    print(board.legal_moves("e1"))    # ['d1', 'd2', 'f1', 'f2']
    print(board.legal_moves("d5"))    # []
    print(board.is_in_check(W))       # False

    print(board.try_move("e2", "d2")) # None   -> illegal, the rook is pinned
    after = board.try_move("e2", "e8")
    print(after.get_piece("e8").symbol(), board.get_piece("e8").symbol())
                                      # R r    -> a move returns a new board, the old one is unchanged
    print(after.is_in_check(B))       # True
    print(after.legal_moves("h8"))    # ['g7', 'h7']   -> g8 is attacked by the rook
```


---

## P41 · Flight Setup — Builder *(adapted)*

**Pattern:** Builder · **Difficulty:** Easy · **Time:** ~15 min · **Source:** adapted from `#28 · Airline Management` (Builder is not named in recall or deep_dive)

### Problem

An airline admin creates flights that have many fields, some required and some optional. A constructor with 9 parameters is error-prone, so **set the fields step by step on a builder, then call `build()`**, which validates everything at once.

1. Every setter returns the builder itself, so calls can be chained.
2. **Required:** route, times, aircraft, and at least one seat. **Optional, with defaults:** base fare `1000`, meal service `False`.
3. `add_seat_row(row, seat_class, letters)` adds seats `<row><letter>` for each letter, in order. It can be called many times.
4. **`build()`:**
   - Checks all of these rules and collects **every** error message, in this exact order and wording:
     1. `route is required`, or `source and destination must differ`
     2. `times are required`, or `arrival must be after departure`
     3. `aircraft is required`
     4. `at least one seat is required`
   - If there are any errors it returns `None`, and `get_errors()` returns the list.
   - Otherwise it returns a new `Flight`. The error list is empty until `build()` runs.
5. A built flight must **copy** the builder's seat list, so changing the builder later never changes a flight that was already built.
6. `describe()` returns `<number> <src>-<dst> <dep>-<arr> <aircraft> fare=<fare> seats=<n> meals=<bool>`.

### Names the driver uses

- `FlightBuilder(flight_number)`: `set_route(source, destination)`, `set_times(departure, arrival)`, `set_aircraft(tail_number)`, `set_base_fare`, `set_meal_service(bool)`, `add_seat_row(row, seat_class, letters)`, `build() -> Flight or None`, `get_errors()`
- Flight: `describe()`, `get_seats()`

### Driver code

```python
if __name__ == "__main__":
    f = (FlightBuilder("AI-101")
         .set_route("DEL", "BOM")
         .set_times(600, 780)
         .set_aircraft("VT-ANA")
         .set_base_fare(4500)
         .add_seat_row(1, "BUSINESS", "AB")
         .add_seat_row(10, "ECONOMY", "ABCD")
         .build())
    print(f.describe())       # AI-101 DEL-BOM 600-780 VT-ANA fare=4500 seats=6 meals=False
    print(f.get_seats())      # ['1A', '1B', '10A', '10B', '10C', '10D']

    bad = FlightBuilder("AI-102").set_route("DEL", "DEL").set_times(900, 800)
    print(bad.build())        # None
    print(bad.get_errors())   # ['source and destination must differ', 'arrival must be after departure', 'aircraft is required', 'at least one seat is required']
    empty = FlightBuilder("AI-000")
    print(empty.get_errors())  # []       -> nothing is checked until build()
    print(empty.build(), empty.get_errors())
                              # None ['route is required', 'times are required', 'aircraft is required', 'at least one seat is required']

    g = (FlightBuilder("AI-103").set_route("BOM", "GOI").set_times(100, 170)
         .set_aircraft("VT-XYZ").add_seat_row(5, "ECONOMY", "A").set_meal_service(True).build())
    print(g.describe())       # AI-103 BOM-GOI 100-170 VT-XYZ fare=1000 seats=1 meals=True
                              # -> optional fields fall back to defaults

    b = FlightBuilder("AI-104").set_route("GOI", "DEL").set_times(0, 150).set_aircraft("VT-Q").add_seat_row(1, "ECONOMY", "A")
    first = b.build()
    print(b.add_seat_row(2, "ECONOMY", "A") is b)
                              # True     -> every setter returns the builder for chaining
    second = b.build()
    print(len(first.get_seats()), len(second.get_seats()), first is second)
                              # 1 2 False  -> later builder changes do not leak into a built flight
```


---

## P42 · Show Seat Layouts — Prototype *(adapted)*

**Pattern:** Prototype (+ a prototype registry) · **Difficulty:** Easy · **Time:** ~15 min · **Source:** adapted from `#02 · BookMyShow` (M9: one seat status per show). Prototype is not named in recall or deep_dive.

### Problem

A screen has a fixed seat layout, but **each show needs its own copy** with its own seat statuses. Building the layout from scratch for every show is slow, so **register one prototype layout per screen and clone it for each show**.

1. A layout has a name and seats (seat number → seat type). Each seat starts `AVAILABLE`.
2. `book(seat_number)` changes an `AVAILABLE` seat to `BOOKED`. An unknown or already-booked seat is refused.
3. `clone()` returns a **deep copy**. Booking or adding seats on one clone must never affect another clone or the prototype. A shallow copy would share the seat objects, and the driver catches that.
4. The registry stores prototypes by key. `get_clone(key)` returns a fresh clone, or `None` for an unknown key.
5. Changing a prototype later affects only clones made **after** the change.
6. `available_seats()` returns seat numbers in insertion order.

### Names the driver uses

- `SeatLayout(name)`: `add_seat(seat_number, seat_type)`, `book`, `status_of` (returns `"AVAILABLE"` / `"BOOKED"`), `count`, `available_seats`, `clone`
- `LayoutRegistry()`: `register(key, layout)`, `get_clone(key)`

### Driver code

```python
if __name__ == "__main__":
    imax = SeatLayout("IMAX")
    imax.add_seat("A1", "REGULAR")
    imax.add_seat("A2", "REGULAR")
    imax.add_seat("P1", "PREMIUM")
    registry = LayoutRegistry()
    registry.register("imax", imax)

    show_9am = registry.get_clone("imax")
    show_6pm = registry.get_clone("imax")
    print(show_9am is show_6pm, show_9am is imax)      # False False
    print(show_9am.book("A1"), show_9am.book("A1"))    # True False
    print(show_9am.status_of("A1"), show_6pm.status_of("A1"), imax.status_of("A1"))
                                                       # BOOKED AVAILABLE AVAILABLE
                                                       # -> seats were deep-copied, not shared

    show_9am.add_seat("X1", "REGULAR")                 # -> extra folding seat for one show only
    print(show_9am.count(), show_6pm.count(), imax.count())
                                                       # 4 3 3
    print(show_9am.available_seats())                  # ['A2', 'P1', 'X1']

    imax.add_seat("P2", "PREMIUM")                     # -> the prototype changes later
    show_late = registry.get_clone("imax")
    print(show_late.count(), show_6pm.count())         # 4 3   -> old clones are unaffected
    print(registry.get_clone("4dx"))                   # None  -> unknown prototype
```


---

## P43 · Stackable Log Destinations — Decorator *(adapted)*

**Pattern:** Decorator · **Difficulty:** Medium · **Time:** ~25 min · **Source:** adapted from `#09 · Logging Framework` (follow-up: "Don't slow the app? An async destination wrapper; nothing else changes"). Decorator is not named in recall or deep_dive.

### Problem

Add behaviour to a log destination **by wrapping it**, without changing the destination or the logger. **A wrapper is itself a destination**, so wrappers can be stacked in any order.

1. A destination has `write(level, message) -> bool`. The memory destination stores `"<LEVEL>: <message>"`.
2. **The decorator base:**
   - It takes the destination to wrap and stores it as **`self.inner`**. The driver's own decorator relies on that name.
   - By default it forwards `write` to `self.inner`.
3. **Masking:** replaces every listed secret in the message with `***`, then forwards.
4. **Level filter:** returns `False` without forwarding when the level is below its minimum.
5. **Async:**
   - `write` only queues the entry and returns `True` at once. One background worker thread forwards entries **in order**.
   - `close()` flushes everything queued, then stops and joins the worker.
   - `write` after `close` returns `False`.

### Names the driver uses

- `LogLevel`: `DEBUG`, `INFO`, `WARN`, `ERROR`
- `LogDestination` (abstract), `MemoryDestination()` with `get_lines()`
- `DestinationDecorator(inner)` (the base; the driver subclasses it)
- `MaskingDestination(inner, secrets)`, `LevelFilterDestination(inner, min_level)`, `AsyncDestination(inner)` with `close()`

### Driver code

```python
import time


class SlowDestination(DestinationDecorator):
    def write(self, level, message):
        time.sleep(0.05)                      # -> pretend this is a slow network call
        return self.inner.write(level, message)


if __name__ == "__main__":
    mem = MemoryDestination()
    masked = MaskingDestination(mem, ["4111-1111", "hunter2"])
    print(masked.write(LogLevel.INFO, "card 4111-1111 charged"))   # True
    print(mem.get_lines())                                          # ['INFO: card *** charged']

    mem2 = MemoryDestination()
    stack = LevelFilterDestination(MaskingDestination(mem2, ["hunter2"]), LogLevel.WARN)
    print(stack.write(LogLevel.INFO, "password is hunter2"))        # False  -> filtered out before masking
    print(stack.write(LogLevel.ERROR, "login failed for hunter2"))  # True
    print(mem2.get_lines())                                         # ['ERROR: login failed for ***']
    print(isinstance(stack, LogDestination))                        # True   -> a decorator is still a destination

    mem3 = MemoryDestination()
    async_dest = AsyncDestination(SlowDestination(mem3))
    start = time.monotonic()
    results = [async_dest.write(LogLevel.INFO, f"event {i}") for i in range(5)]
    print(results, time.monotonic() - start < 0.05)
                                                                    # [True, True, True, True, True] True
                                                                    # -> the caller never waited for the slow write
    async_dest.close()                                              # -> flushes everything still queued
    print(mem3.get_lines())                                         # ['INFO: event 0', 'INFO: event 1', 'INFO: event 2', 'INFO: event 3', 'INFO: event 4']
    print(async_dest.write(LogLevel.INFO, "late"))                  # False  -> closed
```


---

## P44 · One-Call Ticket Booking — Facade *(adapted)*

**Pattern:** Facade · **Difficulty:** Easy–Medium · **Time:** ~20 min · **Source:** adapted from `#02 · BookMyShow` (`BookingService` as the single entry point). Facade is not named in recall or deep_dive.

### Problem

Booking tickets touches three subsystems: seat inventory, payments and notifications. Callers should make **one call** and never deal with the order of steps or the undo themselves. The facade runs the steps **in the right order** and **undoes earlier steps when a later one fails**.

1. **Seat inventory:**
   - A show has seats with prices, and each seat is `AVAILABLE`, `HELD` or `BOOKED`.
   - `hold` is all-or-nothing: if any seat is unknown or not available, nothing is held.
   - It also has `total`, `confirm` and `release`.
2. **Payments:**
   - Each user has a balance.
   - `charge(user, amount)` returns a payment id (`PAY1`, …), or `None` if the balance is too low.
   - `refund(payment_id)` gives the money back.
3. **Notifier:** prints `to <user>: <text>`.
4. **`book_tickets(user_id, show_id, seats)`:**
   1. Hold the seats. If that fails, return `None` (nothing else happens).
   2. Charge the total. If that fails, **release the hold** and return `None`.
   3. Confirm the seats, record booking `BK1`, `BK2`, …, notify `booked <s1,s2> for <show>, paid <amount>`, and return the booking id.
   - An empty seat list returns `None`.
5. **`cancel_booking(booking_id)`:** refund, release the seats, notify `cancelled <id>, refund issued`, and return `True`. An unknown or already-cancelled booking returns `False`.

### Names the driver uses

- `SeatInventory()`: `add_show(show_id, {seat: price})`, `status(show_id, seat)`
- `PaymentService({user: balance})`: `get_balance(user_id)`
- `Notifier()`
- `BookingFacade(inventory, payments, notifier)`: `book_tickets`, `cancel_booking`

### Driver code

```python
if __name__ == "__main__":
    inventory = SeatInventory()
    inventory.add_show("S1", {"A1": 200, "A2": 200, "P1": 350})
    payments = PaymentService({"u1": 1000, "u2": 300})
    facade = BookingFacade(inventory, payments, Notifier())

    print(facade.book_tickets("u1", "S1", ["A1", "P1"]))
                                                    # to u1: booked A1,P1 for S1, paid 550
                                                    # BK1
    print(payments.get_balance("u1"), inventory.status("S1", "P1"))
                                                    # 450 BOOKED
    print(facade.book_tickets("u2", "S1", ["A2", "P1"]))
                                                    # None   -> P1 taken, so nothing was held or charged
    print(inventory.status("S1", "A2"), payments.get_balance("u2"))
                                                    # AVAILABLE 300
    print(facade.book_tickets("u2", "S1", ["A2", "A1"]))
                                                    # None   -> A1 already booked
    inventory.add_show("S2", {"B1": 400})
    print(facade.book_tickets("u2", "S2", ["B1"]))  # None   -> 400 is more than u2's 300
    print(inventory.status("S2", "B1"))             # AVAILABLE  -> the hold was undone after the failed payment

    print(facade.cancel_booking("BK1"))             # to u1: cancelled BK1, refund issued
                                                    # True
    print(payments.get_balance("u1"), inventory.status("S1", "A1"))
                                                    # 1000 AVAILABLE
    print(facade.cancel_booking("BK1"))             # False
    print(facade.book_tickets("u2", "S1", []))      # None   -> no seats asked for
```


---

## P45 · Third-party Payment SDKs — Adapter *(adapted)*

**Pattern:** Adapter · **Difficulty:** Easy–Medium · **Time:** ~20 min · **Source:** adapted from `#27 · Digital Wallet` ("gateways behind one interface"). Adapter is not named in recall or deep_dive.

### Problem

Your checkout code only knows **one interface**: `PaymentGateway.charge(amount, idempotency_key) -> GatewayResult`. Two third-party libraries (given in the driver, **do not modify them**) have completely different APIs. **Write one adapter per library** that makes it fit your interface.

1. `GatewayResult` has an outcome (`SUCCESS`, `DECLINED`, `ERROR`) and a reference (or `None`).
2. **Card SDK adapter** (built with the SDK and a card token):
   - Your amounts are whole rupees, and the SDK wants **cents** (× 100).
   - Pass your idempotency key as the SDK's `request_id`.
   - `{"status": "approved", "txn": id}` → `SUCCESS` with that txn.
   - Any other status → `DECLINED`.
   - **A `TimeoutError` raised by the SDK → `ERROR`.** Never let it escape.
3. **Bank API adapter** (built with the API and an account):
   - The API wants a **float** amount and returns `(code, message)`.
   - Code `0` → `SUCCESS` with reference `bank-<key>`.
   - Code `51` → `DECLINED`.
   - Anything else → `ERROR`.
4. `Checkout(gateway).pay(amount, key)` returns `"<OUTCOME> <reference>"`. It works unchanged with either adapter.

### Names the driver uses

- `GatewayOutcome`: `SUCCESS`, `DECLINED`, `ERROR` · `PaymentGateway` (abstract)
- `CardSdkAdapter(sdk, card_token)`, `BankApiAdapter(api, account)`
- `Checkout(gateway)`: `pay(amount, key) -> str`
- Given third-party classes, defined in the driver: `LegacyCardSdk`, `BankTransferApi`

### Driver code

```python
# --- Third-party code: given, do not modify ---
class LegacyCardSdk:
    def __init__(self, limit_cents):
        self.limit_cents, self.seen, self.last_cents, self.n = limit_cents, {}, None, 0
    def make_payment(self, cents, token, request_id):
        self.last_cents = cents
        if request_id in self.seen: return self.seen[request_id]
        if request_id.startswith("slow"): raise TimeoutError("gateway timed out")
        self.n += 1
        resp = {"status": "approved", "txn": f"ch_{self.n}"} if cents <= self.limit_cents else {"status": "declined"}
        self.seen[request_id] = resp
        return resp


class BankTransferApi:
    def __init__(self, balances): self.balances, self.last_amount = balances, None
    def transfer(self, account, rupees, ref):
        self.last_amount = rupees
        if account not in self.balances: return 91, "system error"
        if self.balances[account] < rupees: return 51, "insufficient funds"
        self.balances[account] -= rupees
        return 0, "ok"
# --- end of third-party code ---


if __name__ == "__main__":
    sdk = LegacyCardSdk(limit_cents=500000)
    card = Checkout(CardSdkAdapter(sdk, "tok_visa"))
    print(card.pay(1500, "k1"))          # SUCCESS ch_1
    print(sdk.last_cents)                # 150000   -> the adapter converted rupees to cents
    print(card.pay(1500, "k1"))          # SUCCESS ch_1   -> same key, the SDK returned the first result
    print(card.pay(9000, "k2"))          # DECLINED None
    print(card.pay(100, "slow-1"))       # ERROR None     -> the SDK's TimeoutError became ERROR

    api = BankTransferApi({"ACC-1": 2000.0})
    bank = Checkout(BankApiAdapter(api, "ACC-1"))
    print(bank.pay(1200, "k3"))          # SUCCESS bank-k3
    print(api.last_amount)               # 1200.0   -> the bank API wants a float
    print(bank.pay(1200, "k4"))          # DECLINED None  -> code 51, only 800 left
    print(Checkout(BankApiAdapter(api, "ACC-404")).pay(10, "k5"))
                                         # ERROR None     -> code 91
    print(isinstance(CardSdkAdapter(sdk, "t"), PaymentGateway), isinstance(sdk, PaymentGateway))
                                         # True False     -> only the adapter fits our interface
```

