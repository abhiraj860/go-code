# LLD Recall Cards: 29 Problems on a Few Pages

> **How to use this file**
> 1. Read **Part 1 (the 10 moves)** until you can say all 10 from memory. These moves are 80% of every design.
> 2. To practise a problem, cover its card, read only the title, and on a blank page write: core idea → classes → pattern → hard part → one follow-up. Then compare with the card.
> 3. Repeat each problem on day 1, 3, 7 and 21.
> 4. For classes, methods, flows, concurrency, edge cases and every follow-up, open the card's **Deep detail** link, which goes to [00-DEEP_DETAIL.md](00-DEEP_DETAIL.md).
---

## Part 1: The 10 Moves (memorize these first)

Every design below is mostly made of these moves. Each card lists the moves it uses, like `M1 M3`.

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

## Part 2: The 10-Step Method (works for any new problem)

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

### Hint phrase → pattern

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

## Part 3: The Cards (grouped by type)

Card format: **Idea** (1 sentence) · **Classes** · **Pattern** · **Moves** · **Hard part** · **Follow-up** (Q → A).

---

### Type A: Booking a Shared Resource

*Skeleton: Resource → Slot/Seat/Compartment → Booking/Ticket, with a Service as the single entry point. The hard part is always no double booking (M1).*

#### 01 · Parking Lot
- **Deep detail:** [01 · Parking Lot →](00-DEEP_DETAIL.md#01--parking-lot)
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

#### 02 · BookMyShow
- **Deep detail:** [02 · BookMyShow →](00-DEEP_DETAIL.md#02--bookmyshow)
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

#### 15 · Library Management
- **Deep detail:** [15 · Library Management →](00-DEEP_DETAIL.md#15--library-management)
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

#### 18 · Amazon Locker
- **Deep detail:** [18 · Amazon Locker →](00-DEEP_DETAIL.md#18--amazon-locker)
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

#### 24 · Meeting Room Booking
- **Deep detail:** [24 · Meeting Room Booking →](00-DEEP_DETAIL.md#24--meeting-room-booking)
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

#### 28 · Airline Management
- **Deep detail:** [28 · Airline Management →](00-DEEP_DETAIL.md#28--airline-management)
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

---

### Type B: State Machines

*Skeleton: a Context class + a `State` interface + one class per state. A state returns a result, and the context applies it. Draw the state diagram first.*

#### 04 · Elevator
- **Deep detail:** [04 · Elevator →](00-DEEP_DETAIL.md#04--elevator)
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

#### 05 · Vending Machine
- **Deep detail:** [05 · Vending Machine →](00-DEEP_DETAIL.md#05--vending-machine)
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

#### 12 · ATM
- **Deep detail:** [12 · ATM →](00-DEEP_DETAIL.md#12--atm)
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

---

### Type C: Matching / Dispatch

*Skeleton: Request → Matcher (Strategy over snapshots) → `try_reserve` on the resource → a Record that runs its lifecycle (+ Observer). Reserve the scarce resource **before** taking money.*

#### 08 · Ride Sharing
- **Deep detail:** [08 · Ride Sharing →](00-DEEP_DETAIL.md#08--ride-sharing)
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

#### 25 · Cab Booking (with scheduled trips)
- **Deep detail:** [25 · Cab Booking (with scheduled trips) →](00-DEEP_DETAIL.md#25--cab-booking-with-scheduled-trips)
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

#### 14 · Food Delivery
- **Deep detail:** [14 · Food Delivery →](00-DEEP_DETAIL.md#14--food-delivery)
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

---

### Type D: Money / Ledger

#### 03 · Splitwise
- **Deep detail:** [03 · Splitwise →](00-DEEP_DETAIL.md#03--splitwise)
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

#### 27 · Digital Wallet
- **Deep detail:** [27 · Digital Wallet →](00-DEEP_DETAIL.md#27--digital-wallet)
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

---

### Type E: Queue + Workers (async systems)

*Skeleton: `submit()` records the job and enqueues it, then returns → `BlockingQueue` → `WorkerPool` → handler (Strategy) → record the outcome, retry with backoff. Know: bounded queue, condition variables, shutdown, at-least-once + idempotency.*

#### 13 · Notification Service
- **Deep detail:** [13 · Notification Service →](00-DEEP_DETAIL.md#13--notification-service)
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

#### 17 · Task Scheduler
- **Deep detail:** [17 · Task Scheduler →](00-DEEP_DETAIL.md#17--task-scheduler)
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

#### 19 · Bounded Queue + Pub-Sub
- **Deep detail:** [19 · Bounded Queue + Pub-Sub →](00-DEEP_DETAIL.md#19--bounded-queue--pub-sub)
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

#### 23 · Web Crawler
- **Deep detail:** [23 · Web Crawler →](00-DEEP_DETAIL.md#23--web-crawler)
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

#### 26 · Online Code Judge
- **Deep detail:** [26 · Online Code Judge →](00-DEEP_DETAIL.md#26--online-code-judge)
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

#### 29 · Webhook Delivery
- **Deep detail:** [29 · Webhook Delivery →](00-DEEP_DETAIL.md#29--webhook-delivery)
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

---

### Type F: Data Structures / Infra

*Skeleton: one core data structure behind an interface. Interviewers test whether you know the data structure, so practise coding these.*

#### 06 · Rate Limiter
- **Deep detail:** [06 · Rate Limiter →](00-DEEP_DETAIL.md#06--rate-limiter)
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

#### 07 · LRU / LFU Cache
- **Deep detail:** [07 · LRU / LFU Cache →](00-DEEP_DETAIL.md#07--lru--lfu-cache)
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

#### 09 · Logging Framework
- **Deep detail:** [09 · Logging Framework →](00-DEEP_DETAIL.md#09--logging-framework)
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

#### 20 · In-Memory File System
- **Deep detail:** [20 · In-Memory File System →](00-DEEP_DETAIL.md#20--in-memory-file-system)
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

#### 21 · KV Store with TTL + Transactions
- **Deep detail:** [21 · KV Store with TTL + Transactions →](00-DEEP_DETAIL.md#21--kv-store-with-ttl--transactions)
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

#### 22 · Unique ID Generator (Snowflake)
- **Deep detail:** [22 · Unique ID Generator (Snowflake) →](00-DEEP_DETAIL.md#22--unique-id-generator-snowflake)
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

---

### Type G: Games

*Skeleton: `Game` (turn, status, players) + `Board` (cells, move legality) + `Player`. A turn is a fixed sequence of steps.*

#### 10 · Snake and Ladder
- **Deep detail:** [10 · Snake and Ladder →](00-DEEP_DETAIL.md#10--snake-and-ladder)
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

#### 11 · Tic-Tac-Toe
- **Deep detail:** [11 · Tic-Tac-Toe →](00-DEEP_DETAIL.md#11--tic-tac-toe)
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

#### 16 · Chess
- **Deep detail:** [16 · Chess →](00-DEEP_DETAIL.md#16--chess)
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
