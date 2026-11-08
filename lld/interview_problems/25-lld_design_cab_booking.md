# Cab Booking System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** because trips can be booked for later, a cab is not just "free" or "busy". It has a calendar of reserved time windows. Each cab guards its own calendar with a lock, so "check the window is free, then reserve it" is one step and a cab is never booked for overlapping trips. Choosing which cab to send is a replaceable rule, and a trip runs its own stages.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. Users should be able to book a cab with pickup and drop-off locations, now or for a later time.
2. Cabs should have a type and an availability status.
3. The system should allocate the nearest available cab using a rule that can be replaced.
4. A cab must never be booked for overlapping trips.
5. The fare should be calculated from distance and cab type.
6. Users should be able to cancel a booking.
7. A trip should move through the stages of booked, started, completed and cancelled.

**Types and conventions**
- Money is `int`, in the smallest currency unit. Distance is `float` kilometers. Times are `datetime`, and the caller passes the current time (`now`) in.
- A `Location` is a point on a flat city map measured in kilometers, and distance is the straight line between two points.
- A trip's time window starts at the pickup time and lasts the estimated travel time plus a buffer. The estimate uses an average speed, which is data.
- A time window includes its start and excludes its end, so a trip ending at 9:30 and one starting at 9:30 do not overlap.
- For a later booking, "nearest" uses each cab's current location, because where it will be later is unknown.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put`.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| CabType (enum) | Names the kinds of cab | none |
| CabStatus (enum) | Names a cab's availability | none |
| TripStatus (enum) | Names the stages of a trip | none |
| Location | Describes a point and measures distance | x, y |
| TimeWindow | Describes a start and end and checks overlap | start, end |
| RateCard | Holds the prices for one cab type | base, per km |
| FareCalculator | Works out a fare from distance and cab type | the rate cards |
| CabInfo | Gives a read-only picture of one cab for the rule | a copy of key facts |
| Cab | Holds a cab's type, status, location and calendar, and guards them | status, location, reserved windows |
| AllocationStrategy (interface) | Chooses a cab for a pickup | none |
| NearestCabStrategy | Chooses the nearest cab | none |
| Trip | Holds one trip and runs its own stages | status, cab, fare |
| CabService | Books, starts, completes and cancels trips, and manages cabs | the cabs, the trips |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** User, cab, pickup, drop-off, time, type, availability status, trip, fare, distance, booking.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Cab, Trip, Location and TimeWindow are classes. A booking is a trip in its booked stage, so it is not a separate class.
2. **Find fixed sets of values.** Cab types, cab availability, trip stages.
   - *Result:* `CabType`, `CabStatus`, `TripStatus`.
3. **Give every fact one owner.** A cab's status, location and reserved windows belong to the Cab. A trip's stage, cab and fare belong to the Trip.
   - *Why:* if the trip and the cab both stored "this window is taken" as their own fact, they could disagree. The cab owns the window, and the trip only holds its id in the cab's calendar.
4. **Mark what can be computed and do not store it.** "Is this cab free for this window" is computed from its calendar. Distance and the time window are computed from the locations.
5. **Give every verb to the class whose data changes.** Reserving and releasing a window change the Cab. Starting, completing and cancelling change the Trip.
6. **Find the action that touches several classes.** A booking needs every cab, the rule, a reservation that can fail and a new trip, in order. That lives in `CabService.book`.
7. **Find the hint phrase for a pattern.** "Allocate the nearest available cab using a rule that can be replaced" is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirement 4 needs it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A `Trip` or None for booking, a `bool` for every stage change.

## 4. Pattern Guide

### Used: Strategy (allocation)
- **Meaning:** a rule is a small object behind a shared interface, and the service uses whichever one it is given.
- **Hint phrase:** "allocate the nearest available cab using a rule that can be replaced".
- **Without it:** the service would hold the nearest-cab rule itself, and a new rule would mean editing it.
- **When NOT to use it:** when nobody asked to change the rule.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A trip's stage only decides which method is allowed, and each method checks it. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. Cab types differ by a value, not by class. |
| Facade | "a single simple entry point" | Not added as a separate class. `CabService` is already the single entry point. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### CabType
- Values: `MINI`, `SEDAN`, `SUV`
- *Why:* requirement 2 gives cabs a type, and requirement 5 prices by it.

### CabStatus
- Values: `AVAILABLE`, `ON_TRIP`, `OFF_DUTY`
- *Why:* requirement 2's availability. `OFF_DUTY` is set by the driver and means no new bookings. `ON_TRIP` means a trip is running right now. Future bookings are tracked separately, in the calendar.

### TripStatus
- Values: `BOOKED`, `STARTED`, `COMPLETED`, `CANCELLED`
- *Why:* exactly the four stages of requirement 7.

## 6. Classes (smallest dependency first)

### Location
**Why this class exists:** a point and the distance between two points belong together.
- Attributes: `x: float`, `y: float`. *Why:* the position in kilometers.
- Constructor: `Location(x: float, y: float)`. *Why:* both set once.
- Methods: `distance_to(other: Location) -> float`. *Why:* the straight-line distance, computed.
- *Extensibility:* road distance would change only the inside of `distance_to`.

### TimeWindow
**Why this class exists:** requirement 4 is about overlapping trips. A start and end, and the overlap rule, belong together.
- Attributes: `start: datetime`, `end: datetime`.
- Constructor: `TimeWindow(start: datetime, end: datetime)`. *Why:* refuses `end <= start`.
- Methods: `get_start() -> datetime`, `get_end() -> datetime`, `overlaps(other: TimeWindow) -> bool`. *Why `overlaps`:* `start < other.end` and `other.start < end`.
- *Extensibility:* none needed.

### RateCard
**Why this class exists:** requirement 5 prices by type. The two prices of one type belong together.
- Attributes: `base: int`, `per_km: int`. *Why:* the starting fee and the price of each kilometer.
- Constructor: `RateCard(base: int, per_km: int)`. *Why:* refuses negative values.
- Methods: `get_base() -> int`, `get_per_km() -> int`.
- *Extensibility:* none needed.

### FareCalculator
**Why this class exists:** requirement 5's rule (distance and cab type) belongs in one place, so the estimate and every trip use the same formula.
- Attributes: `rates: map{CabType: RateCard}` (a cab type to its prices). *Why:* the price per type is data.
- Constructor: `FareCalculator(rates: map{CabType: RateCard})`. *Why:* refuses a table missing any cab type, so a fare can never be missing.
- Methods: `calculate(cab_type: CabType, distance_km: float) -> int`. *Why:* `base + ceil(per_km * distance_km)`. Rounding up keeps it a whole currency unit.
- *Extensibility:* new prices are data.

### CabInfo
**Why this class exists:** the allocation rule needs to see cabs without being able to change them, and a copy taken at one moment is consistent.
- Attributes: `cab_id: str`, `cab_type: CabType`, `location: Location`. *Why:* what a rule needs to judge which cab is best.
- Constructor: `CabInfo(cab_id: str, cab_type: CabType, location: Location)`. *Why:* all set once.
- Methods: getters for each field.
- *Extensibility:* a new field for a new rule (rating, trips today) is added here.

### Cab
**Why this class exists:** requirement 4 needs one place where "this window is free, so take it" is decided. The cab owns its calendar and the lock.
- Attributes:
  - `cab_id: str`, `driver_name: str`, `cab_type: CabType`. *Why:* the identity, the driver, and requirement 2's type.
  - `status: CabStatus`. *Why:* requirement 2's availability.
  - `location: Location`. *Why:* where the cab is now, for "nearest".
  - `reservations: map{str: TimeWindow}` (a trip id to the window it holds). *Why:* the cab's calendar. It holds only trip ids and windows.
  - `lock: Lock`. *Why:* makes the overlap check and the change one step.
- Constructor: `Cab(cab_id: str, driver_name: str, cab_type: CabType, location: Location)`. *Why:* a new cab is `AVAILABLE` with an empty calendar. It refuses empty text.
- Methods (each under the lock):
  - `get_cab_id() -> str`, `get_cab_type() -> CabType`, `get_status() -> CabStatus`, `get_location() -> Location`. *Why:* read-only views.
  - `snapshot_if_free(window: TimeWindow) -> CabInfo or None`. *Why:* a consistent copy for the rule, but only if the cab is not `OFF_DUTY` and no reservation overlaps the window. None otherwise.
  - `try_reserve(trip_id: str, window: TimeWindow) -> bool`. *Why:* False if `OFF_DUTY` or any reservation overlaps. Otherwise it stores the window and returns True. This is requirement 4.
  - `release(trip_id: str) -> bool`. *Why:* removes a reservation. False if there was none.
  - `start_trip(trip_id: str) -> bool`. *Why:* `AVAILABLE` to `ON_TRIP`, only if that trip has a reservation here.
  - `finish_trip(trip_id: str, dropoff: Location) -> bool`. *Why:* `ON_TRIP` to `AVAILABLE`. It removes the reservation and moves the cab to the drop-off.
  - `go_off_duty() -> bool`. *Why:* only from `AVAILABLE` with no reservations, so no booked rider is left without a cab.
  - `go_on_duty() -> bool`. *Why:* `OFF_DUTY` to `AVAILABLE`.
  - `update_location(location: Location) -> bool`. *Why:* False while `OFF_DUTY`.
- *Extensibility:* none needed.

### AllocationStrategy (interface)
**Why it is an interface:** requirement 3 says the rule can be replaced.
- Methods: `select(candidates: list[CabInfo], pickup: Location) -> CabInfo or None`. *Why it takes copies:* a rule judges cabs but can't change them. None means no cab is suitable.
- *Extensibility:* a new rule is a new class implementing this method.

### NearestCabStrategy (implements AllocationStrategy)
**Why this class exists:** the first rule: the nearest cab.
- Attributes: none. Constructor: `NearestCabStrategy()`.
- Methods: `select(...)`. *Why:* the smallest distance to the pickup. Ties go to the lowest cab id, so the result is the same every time. None for an empty list.
- *Extensibility:* none needed.

### Trip
**Why this class exists:** a trip is the record of one booking and its stage. It runs its own transitions, and it is the only thing that tells its cab to start, finish or release.
- Attributes:
  - `trip_id: str`, `rider_id: str`. *Why:* the identity and the rider.
  - `cab: Cab`. *Why:* the cab that holds this trip's window.
  - `pickup: Location`, `dropoff: Location`, `window: TimeWindow`. *Why:* where and when.
  - `fare: int`. *Why:* fixed at booking, because requirement 5 depends only on distance and type, which are known then.
  - `status: TripStatus`. *Why:* the stage.
  - `lock: Lock`. *Why:* stage changes on one trip run one at a time.
- Constructor: `Trip(trip_id: str, rider_id: str, cab: Cab, pickup: Location, dropoff: Location, window: TimeWindow, fare: int)`. *Why:* the cab has already reserved the window. The status starts `BOOKED`.
- Methods (each under the trip lock):
  - Getters for each field.
  - `start(cab_id: str, now: datetime) -> bool`. *Why:* `BOOKED` only, only that trip's cab, and not before the window starts. It calls `cab.start_trip(trip_id)`, which must succeed, then sets `STARTED`.
  - `complete(cab_id: str) -> bool`. *Why:* `STARTED` only, and only that trip's cab. It calls `cab.finish_trip(trip_id, dropoff)`, then sets `COMPLETED`.
  - `cancel(rider_id: str) -> bool`. *Why:* `BOOKED` only, and only that rider. A started trip can't be cancelled. It calls `cab.release(trip_id)`, then sets `CANCELLED`.
- *Why the trip reaches into the cab and not the other way:* the cab only knows trip ids and windows, so dependencies point one way. The lock order is always trip first, then cab.
- *Extensibility:* none needed.

### CabService (the manager and only entry point)
**Why this class exists:** booking needs every cab, the rule, a reservation that can fail and a new trip, in that order. Cabs and trips need one place to be found.
- Attributes:
  - `cabs: map{str: Cab}` (a cab id to the Cab), `trips: map{str: Trip}` (a trip id to the Trip). *Why:* the one record of each. Both are thread-safe maps.
  - `strategy: AllocationStrategy`, `fares: FareCalculator`. *Why:* the rule and the prices, passed in.
  - `average_speed_kmph: float`, `buffer_minutes: int`. *Why:* how long a trip is expected to take, and the extra time added so back-to-back trips don't collide. Both are data.
  - `counter: AtomicInt`. *Why:* makes trip ids unique.
- Constructor: `CabService(strategy: AllocationStrategy, fares: FareCalculator, average_speed_kmph: float, buffer_minutes: int)`. *Why:* passed in so tests can use fakes. It refuses a speed of 0 or less, or a negative buffer.
- Methods (outer ones take ids and look up once):
  - `add_cab(cab_id: str, driver_name: str, cab_type: CabType, location: Location) -> bool`. *Why:* setup. False if the id exists.
  - `go_off_duty(cab_id: str) -> bool`, `go_on_duty(cab_id: str) -> bool`, `update_location(cab_id: str, location: Location) -> bool`. *Why:* look up once and ask the cab.
  - `estimate_fare(pickup: Location, dropoff: Location, cab_type: CabType) -> int`. *Why:* the same formula used for booking.
  - `book(rider_id: str, pickup: Location, dropoff: Location, cab_type: CabType, pickup_time: datetime, now: datetime) -> Trip or None`. *Why:* requirement 1. `pickup_time` equal to `now` is "book now", and a later time is a scheduled booking.
  - `start_trip(trip_id: str, cab_id: str, now: datetime) -> bool`, `complete_trip(trip_id: str, cab_id: str) -> bool`, `cancel_trip(trip_id: str, rider_id: str) -> bool`. *Why:* look up the trip once and ask it.
  - `get_trip(trip_id: str) -> Trip or None`. *Why:* the lookup by id.
- *Extensibility:* the rule and the prices are passed in.

## 7. Relationships
- CabService **has** Cab, Trip (created inside `add_cab` and `book`).
- Trip **uses** Cab, Location, TimeWindow. Cab **uses** Location, TimeWindow.
- CabService **uses** AllocationStrategy, FareCalculator (passed in).
- FareCalculator **uses** RateCard. CabInfo **uses** Location.
- NearestCabStrategy **implements** AllocationStrategy.
- Enums are always **uses**: Cab to CabType and CabStatus, Trip to TripStatus, FareCalculator and CabInfo to CabType.

## 8. Interaction

### Layers
1. **Entry point and manager:** CabService.
2. **Classes that guard themselves:** Cab (its calendar) and Trip (its stages).
3. **Rules passed in:** AllocationStrategy and FareCalculator.
4. **Plain data:** Location, TimeWindow, RateCard, CabInfo.

### Call chains
- **Book**
  1. `Rider -> CabService.book(...)`.
  2. `CabService -> Cab.snapshot_if_free(window)` for each cab of that type: copies of the cabs that could take it.
  3. `CabService -> AllocationStrategy.select(candidates, pickup)`.
  4. `CabService -> Cab.try_reserve(trip_id, window)`. If refused, drop that cab and choose again.
  5. `CabService -> FareCalculator.calculate(cab_type, distance)`, then `Trip(...)`.
- **Start, complete, cancel**
  1. `Driver or rider -> CabService.start_trip / complete_trip / cancel_trip`.
  2. `CabService -> Trip.start / complete / cancel`.
  3. `Trip -> Cab.start_trip / finish_trip / release`.

### Who may call whom
- Riders and drivers call `CabService` only.
- `CabService` calls `Cab`, `Trip`, `AllocationStrategy` and `FareCalculator`.
- `Trip` calls `Cab`. `Cab` and the rules call no other class.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Book
1. `book(rider_id, pickup, dropoff, cab_type, pickup_time, now)`. If `pickup_time` is before `now`, or the two locations are the same, return None.
2. `distance = pickup.distance_to(dropoff)`. `minutes = distance / average_speed_kmph * 60`. `window = TimeWindow(pickup_time, pickup_time + minutes + buffer_minutes)`.
3. Take the next trip id from `counter`. `candidates` is every non-None `snapshot_if_free(window)` for cabs of `cab_type`.
4. `chosen = strategy.select(candidates, pickup)`. If None, return None.
5. `cab = cabs.get(chosen.get_cab_id())`, looked up once. `cab.try_reserve(trip_id, window)`. If False (another booking took it first), remove that candidate and go back to step 4.
6. `fare = fares.calculate(cab_type, distance)`. Build the `Trip`, store it, and return it.
- **Failure and undo:** steps 1 to 4 change nothing. If building the trip ever failed after step 5, `cab.release(trip_id)` would undo the reservation.

### Start and complete
1. `start_trip(trip_id, cab_id, now)`: only the trip's cab, only `BOOKED`, and not before the window starts. The cab becomes `ON_TRIP`.
2. `complete_trip(trip_id, cab_id)`: only `STARTED`. The cab becomes `AVAILABLE` at the drop-off, and its reservation is removed.
- **Failure:** the wrong cab or the wrong stage is refused with no change.

### Cancel
1. `cancel_trip(trip_id, rider_id)`: only that rider and only `BOOKED`. The cab's window is released, and the trip is `CANCELLED`.
- **Failure:** another rider, or a trip that has started, is refused.

### Concurrency (requirement 4)
- **Cab:** the lock sits inside `Cab`, around every method. `try_reserve` checks for overlap and stores the window as one step, so two bookings can't give one cab overlapping trips.
- **Trip:** the lock sits inside `Trip`, around `start`, `complete` and `cancel`.
- **Stale copies:** a `CabInfo` may be out of date by the time `try_reserve` runs. A refused reservation makes the service choose again, so no rider is given a double-booked cab.
- **Maps and ids:** both maps are thread-safe, and ids come from an atomic counter.
- **Lock order:** always the trip first, then its cab. `book` takes only cab locks, one at a time. So no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new allocation rule (least busy today, highest rated) | A new class implementing `AllocationStrategy`, passed to the service | No. |
| New prices | A new `RateCard` in the `FareCalculator` table | No. It is data. |
| A new cab type | One new `CabType` value and its `RateCard` | No logic changes. |
| Longer buffers or a different average speed | Different numbers passed to the service | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **A cab has a calendar of windows, not just a busy flag.**
   - *Why:* requirement 1 allows booking for later, and a single flag can't say "free now but booked at 3 pm".
   - *Rejected:* a busy flag, which works only for "book now".
2. **Each cab guards its calendar with a lock.**
   - *Why:* only bookings for the same cab compete, and "check and reserve" is one step.
   - *Rejected:* one lock for the whole service.
3. **The rule sees copies, and the reservation can refuse.**
   - *Why:* the rule stays simple, and a stale copy is caught by `try_reserve`.
   - *Rejected:* passing live cabs to the rule.
4. **The fare is fixed at booking.**
   - *Why:* requirement 5 uses distance and type, which are known then, and the rider sees the price before the trip.
   - *Rejected:* computing it at the end from time taken, which no requirement asks for.
5. **`Trip` tells its cab to start, finish or release.**
   - *Why:* every stage change and its effect on the cab happen together, in one place.
   - *Rejected:* the service changing the trip and the cab separately, where one could be forgotten.

### Edge cases handled
- Two bookings racing for one cab's window: one wins, the other picks the next cab.
- A trip ending at 9:30 and one starting at 9:30 can use the same cab.
- A cab that is off duty is never offered.
- A cab with future bookings can't go off duty.
- A pickup time in the past, or identical pickup and drop-off, is refused.
- A trip can't start before its window or be started by another cab.
- A started trip can't be cancelled.
- A missing price for a cab type is refused at setup.

### Likely follow-up questions
1. **"For a later booking, isn't the cab's current location a poor guess?"** Yes. A better rule uses the drop-off of the cab's previous trip before that time. That is a new `AllocationStrategy` with an extra field in `CabInfo`.
2. **"What if a trip runs late and clashes with the next booking?"** The buffer reduces the risk. A complete answer moves the next booking to another cab when a trip overruns, which is a new rule in `Trip.complete`.
3. **"Should we assign a cab for a booking a week away right now?"** Many systems assign close to the time instead, when locations are known. That would hold the booking unassigned and run allocation later from a scheduler.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Live tracking and road distance | Distance is a straight line, and the cab's location is updated by the driver. |
| A final fare based on time taken | The fare is fixed at booking from distance and type. |
| Payment | Not required. |
| Cancellation fees | Not required. It would be a rule in `Trip.cancel`. |
| Assigning cabs close to the time | See follow-up question 3. |
| Trips that overrun | See follow-up question 2. |

## 13. Requirements Met
1. Book now or later: `CabService.book` with `pickup_time`.
2. Type and availability: `CabType`, `CabStatus`, and `Cab.go_off_duty`, `go_on_duty`, `start_trip`, `finish_trip`.
3. Nearest cab by a replaceable rule: `AllocationStrategy` with `NearestCabStrategy`.
4. No overlapping trips: `TimeWindow.overlaps` and the lock inside `Cab.try_reserve`.
5. Fare from distance and type: `FareCalculator` with `RateCard`.
6. Cancel: `CabService.cancel_trip` and `Trip.cancel`.
7. The four stages: `TripStatus` and the guarded methods of `Trip`.

## 14. Requirement Fit Check

**Setup used in every example:** the average speed is 30 km/h and the buffer is 10 minutes. MINI costs 40 plus 10 per km, SEDAN 60 plus 14 per km. Cabs: `C1` MINI at (0, 1), `C2` MINI at (0, 5), `C3` SEDAN at (2, 0). Pickup `P` is (0, 0) and drop-off `D` is (6, 8), so the trip is 10 km, an estimated 20 minutes, and a window of 30 minutes. Now is 09:00.

1. **Book now or later**
   - *Success:* `book("U1", P, D, MINI, 09:00, 09:00)` picks C1 (1 km away) for the window 09:00 to 09:30, and returns trip `T1`. A booking for 15:00 the same day also picks C1, because its 15:00 to 15:30 window is free.
   - *Failure:* `book("U1", P, D, MINI, 08:00, 09:00)` returns None because the pickup time has passed.
2. **Type and availability**
   - *Success:* only MINI cabs are considered for a MINI booking. `go_off_duty("C2")` makes C2 `OFF_DUTY`, and it is no longer offered.
   - *Failure:* `go_off_duty("C1")` returns False because C1 has bookings.
3. **Nearest cab by a replaceable rule**
   - *Success:* C1 at 1 km is chosen over C2 at 5 km. Passing a different `AllocationStrategy` to the service changes the rule with no other change.
   - *Failure:* with every MINI cab busy or off duty, `select` gets an empty list and `book` returns None.
4. **No overlapping trips**
   - *Success:* with T1 holding C1 from 09:00 to 09:30, `book("U2", P, D, MINI, 09:10, 09:00)` can't use C1, so it gets C2. A booking for 09:30 can use C1 again.
   - *Failure:* if C2 is off duty, that 09:10 booking returns None. Two riders booking the last free MINI for the same window at the same moment: the cab's lock lets one reserve it, and the other is refused.
5. **Fare from distance and type**
   - *Success:* 10 km in a MINI costs 40 + 100 = 140. In a SEDAN it costs 60 + 140 = 200.
   - *Failure:* a `FareCalculator` with no SUV price is refused at setup.
6. **Cancel**
   - *Success:* `cancel_trip("T1", "U1")` returns True, T1 is `CANCELLED`, and C1's 09:00 to 09:30 window is free again.
   - *Failure:* `cancel_trip("T1", "U2")` returns False because U2 did not book it. Cancelling after the trip has started returns False.
7. **The four stages**
   - *Success:* T1 goes `BOOKED`, then `STARTED` (C1 starts it at 09:02 and becomes `ON_TRIP`), then `COMPLETED` (C1 is `AVAILABLE` again at D).
   - *Failure:* `complete_trip` before `start_trip` returns False. `start_trip("T1", "C2", ...)` returns False because C2 is not T1's cab.
