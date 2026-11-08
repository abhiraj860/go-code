# Ride-Sharing System (Uber/Ola): Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a driver is the scarce resource, so the driver object guards its own availability with a lock, and a driver is reserved before a ride is built around them. A ride is a record that runs its own lifecycle. The three rules most likely to change, matching, pricing and payment, are passed in as swappable objects.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
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

**Types and conventions**
- Money is `int`, in the smallest currency unit. Distance is `float` kilometers. Duration is `float` minutes. Times are `datetime`, passed in by the caller, so the design never reads a clock.
- A `Location` is a point on a flat city map measured in kilometers, and distance is the straight line between two points.
- The rider chooses the vehicle type when requesting a ride, because the fare depends on it.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put_if_absent`, which stores a value only if the key is empty.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Location | Describes a point and measures distance to another | x, y |
| Vehicle | Describes one registered vehicle | plate, type |
| ReputationScore | Adds up ratings into an average | total stars, count |
| Rider | Is a registered rider with a rating | id, name, rating |
| Driver | Is a registered driver and guards their own availability | status, location, vehicles, rating |
| DriverInfo | Gives a read-only picture of one available driver | a copy of key facts |
| RateCard | Holds the three prices for one vehicle type | base, per km, per minute |
| PricingStrategy (interface) | Turns distance, time and vehicle type into a fare | none |
| DistanceTimePricing | Charges base plus per-km plus per-minute | the rate cards |
| MatchingStrategy (interface) | Chooses a driver for a pickup | none |
| NearestDriverStrategy | Chooses the nearest driver within a radius | the radius |
| PaymentMethod (interface) | Takes a payment | none |
| CardPayment | Pays by card | amount paid |
| UpiPayment | Pays by UPI | amount paid |
| Ride | Holds one trip and runs its own stages, payment and ratings | status, driver, fare |
| RideService | Registers people, matches drivers, and runs each action in the right order | the riders, drivers and rides |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Rider, driver, vehicle, availability, location, ride, fare estimate, matching rule, stage, pricing rule, payment method, rating.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Rider, Driver, Vehicle, Location, Ride are classes. Fare and distance are values. A "rating" is a number added to a score.
2. **Find fixed sets of values.** Vehicle types, driver availability, ride stages.
   - *Why:* a fixed set is an enum, so a typo like "compleet" cannot exist.
   - *Result:* `VehicleType`, `DriverStatus`, `RideStatus`.
3. **Give every fact one owner.** Availability and location belong to the Driver. The stage, the assigned driver and the fare belong to the Ride. A rating average belongs to a ReputationScore.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** The fare estimate, the distance, "is the ride paid" and a rating average are all computed.
   - *Why:* a stored estimate would go stale the moment the pricing rule changes.
5. **Give every verb to the class whose data changes.** Reserving a driver changes the Driver. Accepting, starting and completing change the Ride.
6. **Find the action that touches several classes.** Requesting a ride needs the drivers, a matching rule, a reservation that can fail, and a new Ride. That order lives in `RideService`.
7. **Find the hint phrases for patterns.** "A matching rule that can be replaced", "the pricing rule should be easy to swap" and "multiple payment methods" are all Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** Requirement 10 names it, so decide exactly where the locks sit (section 8).
10. **Fix the outputs.** A `Ride` or None for a request, a `bool` for every command, and an `int` for the estimate.

## 4. Pattern Guide

### Used: Strategy (three times)
- **Meaning:** a rule is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrases:** "a matching rule that can be replaced", "the pricing rule should be easy to swap", and "the system should support multiple payment methods".
- **Without it:** the service would have an `if` for the matching rule, another for pricing and another for payment, and every new rule or method would mean editing it.
- **When NOT to use it:** when there is one fixed rule that nobody asked to change.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. The ride stages only decide which method is allowed, and each method checks the status itself. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. Vehicles differ by a type value, not by class. |
| Facade | "a single simple entry point" | Not added as a separate class. `RideService` is already the single entry point because it is the manager. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. A passed-in service is easier to test. |

## 5. Enums

### VehicleType
- Values: `AUTO`, `SEDAN`, `SUV`
- *Why:* requirement 7 prices by vehicle type, and a driver registers vehicles of some type.
- *Extensibility:* a new type is one new value here, plus a rate card in the pricing table.

### DriverStatus
- Values: `OFFLINE`, `AVAILABLE`, `BUSY`
- *Why:* requirement 1 has drivers set their availability. `BUSY` means offered a ride or on a ride, so the system can mark a driver unavailable without the driver doing anything.

### RideStatus
- Values: `REQUESTED`, `ACCEPTED`, `IN_PROGRESS`, `COMPLETED`, `CANCELLED`
- *Why:* exactly the five stages named in requirement 6.

## 6. Classes (smallest dependency first)

### Location
**Why this class exists:** a point and the distance between two points belong together, and many classes use them.
- Attributes: `x: float`, `y: float`. *Why:* the position in kilometers on the city map.
- Constructor: `Location(x: float, y: float)`. *Why:* both fields are set once, so a stored location can't change under a ride.
- Methods: `get_x() -> float`, `get_y() -> float`, `distance_to(other: Location) -> float`. *Why `distance_to`:* the straight-line distance, computed and never stored.
- *Extensibility:* real latitude and longitude would change the inside of `distance_to` only.

### Vehicle
**Why this class exists:** requirement 1 has drivers register vehicles. A vehicle is a thing with its own identity (the plate) and a type.
- Attributes: `plate_number: str`, `vehicle_type: VehicleType`. *Why:* the identity and what the fare depends on.
- Constructor: `Vehicle(plate_number: str, vehicle_type: VehicleType)`. *Why:* both fields are set once. It refuses an empty plate.
- Methods: `get_plate_number() -> str`, `get_vehicle_type() -> VehicleType`. *Why:* read-only views.
- *Extensibility:* none needed.

### ReputationScore
**Why this class exists:** riders and drivers are both rated, and the same averaging logic is needed for both. One class means one place for the rule.
- Attributes: `total_stars: int`, `count: int`, `lock: Lock`. *Why:* a sum and a count are enough to compute the average, and they use less memory than keeping every rating. The lock is there because one driver can be rated through two rides at once.
- Constructor: `ReputationScore()`. *Why:* starts at 0 and 0, and creates its lock.
- Methods:
  - `add(stars: int) -> bool`. *Why:* refuses stars outside 1 to 5, so no meaningless rating is counted.
  - `get_average() -> float or None`. *Why:* computed as the total divided by the count. None if there are no ratings yet.
  - `get_count() -> int`. *Why:* read-only view.
- *Extensibility:* none needed.

### Rider
**Why this class exists:** requirement 1 has riders register, and requirement 9 has them rated.
- Attributes: `rider_id: str`, `name: str`, `reputation: ReputationScore`. *Why each:* the identity, the display name, and the rating.
- Constructor: `Rider(rider_id: str, name: str)`. *Why:* the id and name are set once, and the reputation starts empty. It refuses empty values.
- Methods: `get_rider_id() -> str`, `get_name() -> str`, `add_rating(stars: int) -> bool`, `get_average_rating() -> float or None`. *Why:* the reputation is reached only through these two checked methods.
- *Extensibility:* none needed.

### Driver
**Why this class exists:** requirement 10 needs one place where "this driver can take only one ride" is decided. The driver owns that decision, so it owns the status and the lock.
- Attributes:
  - `driver_id: str`, `name: str`. *Why:* identity and display name.
  - `vehicles: map{str: Vehicle}` (a plate number to the Vehicle). *Why:* requirement 1 lets a driver register vehicles.
  - `active_vehicle: Vehicle or None`. *Why:* the vehicle the driver is working with right now, so the vehicle type of a match is known. None when offline.
  - `location: Location or None`. *Why:* where the driver is, so "nearby" can be judged. None until first online.
  - `status: DriverStatus`. *Why:* the single fact about availability.
  - `reputation: ReputationScore`. *Why:* the driver's rating.
  - `lock: Lock`. *Why:* makes "check available, then mark busy" one step.
- Constructor: `Driver(driver_id: str, name: str)`. *Why:* a new driver starts OFFLINE with no vehicles, location or ratings, and a lock. It refuses empty values.
- Methods:
  - `get_driver_id() -> str`, `get_name() -> str`, `get_status() -> DriverStatus`, `get_average_rating() -> float or None`. *Why:* read-only views.
  - `add_vehicle(vehicle: Vehicle) -> bool`. *Why:* False if that plate is already registered.
  - `go_online(plate_number: str, location: Location) -> bool`. *Why:* the driver sets availability. Only from OFFLINE, and only with a registered vehicle. It sets the active vehicle and location and becomes AVAILABLE.
  - `go_offline() -> bool`. *Why:* only from AVAILABLE. A busy driver can't just vanish from a ride. It clears the active vehicle.
  - `update_location(location: Location) -> bool`. *Why:* False if the driver is OFFLINE.
  - `try_reserve() -> bool`. *Why:* under the lock, AVAILABLE to BUSY. False otherwise. This is what stops a driver being given two rides.
  - `release() -> bool`. *Why:* under the lock, BUSY to AVAILABLE, with the location unchanged. Used when a ride is rejected or cancelled.
  - `finish_ride(location: Location) -> bool`. *Why:* under the lock, BUSY to AVAILABLE, and the driver is now at the drop-off.
  - `add_rating(stars: int) -> bool`. *Why:* the only way the reputation changes.
  - `snapshot() -> DriverInfo or None`. *Why:* a consistent copy taken under the lock. None unless the driver is AVAILABLE with a location and an active vehicle.
- *Extensibility:* none needed.

### DriverInfo
**Why this class exists:** a matching rule needs to see the available drivers without being able to change them, and a copy taken at one moment is consistent.
- Attributes: `driver_id: str`, `location: Location`, `vehicle_type: VehicleType`, `average_rating: float or None`. *Why each:* what a rule needs to judge how near and how good a driver is.
- Constructor: `DriverInfo(driver_id: str, location: Location, vehicle_type: VehicleType, average_rating: float or None)`. *Why:* all fields set once.
- Methods: `get_driver_id() -> str`, `get_location() -> Location`, `get_vehicle_type() -> VehicleType`, `get_average_rating() -> float or None`. *Why:* read-only views.
- *Extensibility:* a new field for a new rule is added here.

### RateCard
**Why this class exists:** the three prices for one vehicle type belong together.
- Attributes: `base: int`, `per_km: int`, `per_minute: int`. *Why:* the starting fee, the price of each kilometer and the price of each minute.
- Constructor: `RateCard(base: int, per_km: int, per_minute: int)`. *Why:* all set once. It refuses negative values.
- Methods: `get_base() -> int`, `get_per_km() -> int`, `get_per_minute() -> int`. *Why:* read-only views.
- *Extensibility:* none needed.

### PricingStrategy (interface)
**Why it is an interface:** requirement 7 says the pricing rule should be easy to swap, so the system depends on one method and not on a specific rule.
- Methods: `calculate_fare(vehicle_type: VehicleType, distance_km: float, duration_minutes: float) -> int`. *Why these inputs:* they are the three facts requirement 7 names.
- *Extensibility:* a new rule is a new class implementing this method.

### DistanceTimePricing (implements PricingStrategy)
**Why this class exists:** the first rule: a base fee plus a price per kilometer and per minute.
- Attributes: `rates: map{VehicleType: RateCard}` (a vehicle type to its rate card). *Why:* "vary by vehicle type" is a table, so a new type is a data entry.
- Constructor: `DistanceTimePricing(rates: map{VehicleType: RateCard})`. *Why:* it refuses a table missing any vehicle type, so a fare can never be missing.
- Methods: `calculate_fare(vehicle_type: VehicleType, distance_km: float, duration_minutes: float) -> int`. *Why:* `base + ceil(per_km * distance_km + per_minute * duration_minutes)`. Rounding up keeps the result a whole currency unit.
- *Extensibility:* rate changes are data. A different formula is a different class.

### MatchingStrategy (interface)
**Why it is an interface:** requirement 4 says the matching rule can be replaced.
- Methods: `select(candidates: list[DriverInfo], pickup: Location) -> DriverInfo or None`. *Why it takes copies:* a rule can judge drivers but can't change them. None means no suitable driver.
- *Extensibility:* a new rule is a new class implementing this method.

### NearestDriverStrategy (implements MatchingStrategy)
**Why this class exists:** the first rule: the nearest driver, as long as they are nearby.
- Attributes: `max_distance_km: float`. *Why:* "nearby" in requirement 4 is a distance limit.
- Constructor: `NearestDriverStrategy(max_distance_km: float)`. *Why:* set once. It refuses a value of 0 or less.
- Methods: `select(candidates: list[DriverInfo], pickup: Location) -> DriverInfo or None`. *Why:* it ignores drivers farther than the limit, then picks the smallest distance. Ties go to the higher average rating (no rating counts as lowest), then the lowest driver id, so the result is the same every time.
- *Extensibility:* none needed.

### PaymentMethod (interface)
**Why it is an interface:** requirement 8 asks for multiple payment methods.
- Methods: `pay(amount: int) -> bool`. *Why a `bool`:* the ride only needs to know whether it was paid.
- *Extensibility:* a new method is a new class implementing `pay`.

### CardPayment (implements PaymentMethod)
**Why this class exists:** the first way to pay.
- Attributes: `card_number: str`, `paid_amount: int`, `lock: Lock`. *Why:* the card, how much was charged, and a lock so concurrent calls can't charge twice.
- Constructor: `CardPayment(card_number: str)`. *Why:* `paid_amount` starts at 0. It refuses an empty number. One object is used for one payment.
- Methods: `pay(amount: int) -> bool`. *Why:* False if `amount <= 0` or it was already paid. This is where a real card gateway call goes, and a decline returns False.
- *Extensibility:* none needed.

### UpiPayment (implements PaymentMethod)
**Why this class exists:** the second implementation, which shows the interface holds more than one way to pay.
- Attributes: `upi_id: str`, `paid_amount: int`, `lock: Lock`. *Why each:* the same as the card.
- Constructor: `UpiPayment(upi_id: str)`. *Why:* `paid_amount` starts at 0. It refuses an empty id.
- Methods: `pay(amount: int) -> bool`, with the same rules as the card.
- *Extensibility:* none needed.

### Ride
**Why this class exists:** a ride is the record of one trip and its stage. It runs its own transitions, so a direct call from anywhere is still safe.
- Attributes:
  - `ride_id: str`. *Why:* the identity.
  - `rider: Rider`. *Why:* who asked for the trip.
  - `pickup: Location`, `dropoff: Location`, `vehicle_type: VehicleType`. *Why:* what was asked for.
  - `requested_at: datetime`. *Why:* when it was asked for.
  - `driver: Driver or None`. *Why:* the driver currently offered or assigned. None between a rejection and the next match.
  - `rejected_driver_ids: set[str]`. *Why:* drivers who said no, so the next match skips them. It lives here because it is a fact about this ride.
  - `status: RideStatus`. *Why:* the stage.
  - `started_at: datetime or None`. *Why:* when the trip began, so the duration for the fare is known.
  - `fare: int or None`. *Why:* the final fare, set once at completion. None before that.
  - `payment: PaymentMethod or None`. *Why:* the payment that settled it. None while unpaid, so "is it paid" is computed from this.
  - `driver_rated: bool`, `rider_rated: bool`. *Why:* so each side can rate only once. `driver_rated` means the rider has rated the driver.
  - `lock: Lock`. *Why:* makes the ride's transitions run one at a time.
- Constructor: `Ride(ride_id: str, rider: Rider, pickup: Location, dropoff: Location, vehicle_type: VehicleType, requested_at: datetime, driver: Driver)`. *Why:* the driver is already reserved by the caller, so a ride never exists without one on offer. The status starts REQUESTED. It refuses identical pickup and drop-off.
- Methods:
  - `get_ride_id() -> str`, `get_status() -> RideStatus`, `get_driver() -> Driver or None`, `get_fare() -> int or None`, `get_vehicle_type() -> VehicleType`, `get_pickup() -> Location`. *Why:* read-only views.
  - `get_rejected_driver_ids() -> set[str]` (returns a copy). *Why:* the service skips these when matching again.
  - `is_paid() -> bool`. *Why:* computed from `payment`.
  - `assign(driver: Driver) -> bool`. *Why:* the next driver offered after a rejection. REQUESTED with no driver only. The caller has already reserved the driver.
  - `accept(driver_id: str) -> bool`. *Why:* REQUESTED only, and only the offered driver. It sets ACCEPTED.
  - `reject(driver_id: str) -> bool`. *Why:* REQUESTED only, and only the offered driver. It releases the driver, remembers the id, and clears the driver.
  - `cancel_unmatched() -> bool`. *Why:* REQUESTED with no driver only. Ends the ride when nobody is left to ask.
  - `cancel(rider_id: str) -> bool`. *Why:* only that rider, and only in REQUESTED or ACCEPTED. It releases any driver and sets CANCELLED. An in-progress trip can't be cancelled.
  - `start(driver_id: str, now: datetime) -> bool`. *Why:* ACCEPTED only, and only the assigned driver. It sets IN_PROGRESS and `started_at`.
  - `complete(driver_id: str, now: datetime, pricing: PricingStrategy) -> bool`. *Why:* IN_PROGRESS only, only the assigned driver, and `now` not before `started_at`. It computes the fare from the straight-line distance and the real minutes taken, frees the driver at the drop-off, and sets COMPLETED.
  - `pay(rider_id: str, payment: PaymentMethod) -> bool`. *Why:* only that rider, COMPLETED only, and unpaid. It calls `payment.pay(fare)`. If that returns False, nothing is stored and another method can be tried.
  - `rate_driver(rider_id: str, stars: int) -> bool`, `rate_rider(driver_id: str, stars: int) -> bool`. *Why:* COMPLETED only, each at most once, and only by the right person. The flag is set only if the rating was accepted.
- *Why every method runs under the ride lock:* two actions on the same ride (a cancel and an accept) can't overlap.
- *Extensibility:* none needed.

### RideService (the manager and only entry point)
**Why this class exists:** requesting a ride needs the drivers, a rule, a reservation that can fail and a new ride, in that order. No single driver or ride can do that.
- Attributes:
  - `riders: map{str: Rider}` (rider id to the Rider), `drivers: map{str: Driver}` (driver id to the Driver), `rides: map{str: Ride}` (ride id to the Ride). *Why:* the one record of each. All are thread-safe maps.
  - `matching: MatchingStrategy`, `pricing: PricingStrategy`. *Why:* the swappable rules, passed in.
  - `average_speed_kmph: float`. *Why:* turns a distance into an estimated duration for the fare estimate. It is data, so it can be changed without code.
  - `counter: AtomicInt`. *Why:* makes ride ids unique when requests arrive at once.
- Constructor: `RideService(matching: MatchingStrategy, pricing: PricingStrategy, average_speed_kmph: float)`. *Why:* the rules are passed in so tests can use fakes. It refuses a speed of 0 or less, and starts with no people or rides.
- Methods (outer ones take ids and look up once):
  - `register_rider(rider_id: str, name: str) -> Rider or None`, `register_driver(driver_id: str, name: str) -> Driver or None`. *Why:* None if the id exists or the values are invalid. `put_if_absent` makes this safe at once.
  - `add_vehicle(driver_id: str, plate_number: str, vehicle_type: VehicleType) -> bool`. *Why:* builds the Vehicle and asks the driver. False if the driver is unknown or the plate exists.
  - `go_online(driver_id: str, plate_number: str, location: Location) -> bool`, `go_offline(driver_id: str) -> bool`, `update_location(driver_id: str, location: Location) -> bool`. *Why:* each looks up the driver once and asks them.
  - `estimate_fare(pickup: Location, dropoff: Location, vehicle_type: VehicleType) -> int or None`. *Why:* requirement 3. Computed from the pricing rule using the distance and the estimated duration. None if pickup equals drop-off.
  - `request_ride(rider_id: str, pickup: Location, dropoff: Location, vehicle_type: VehicleType, now: datetime) -> Ride or None`. *Why:* requirement 2. None if the rider is unknown, the locations match, or no driver can take it.
  - `respond_to_ride(ride_id: str, driver_id: str, accept: bool) -> bool`. *Why:* requirement 5. On a rejection it also matches the next driver.
  - `start_ride(ride_id: str, driver_id: str, now: datetime) -> bool`, `complete_ride(ride_id: str, driver_id: str, now: datetime) -> bool`, `cancel_ride(ride_id: str, rider_id: str) -> bool`, `pay_ride(ride_id: str, rider_id: str, payment: PaymentMethod) -> bool`, `rate_driver(ride_id: str, rider_id: str, stars: int) -> bool`, `rate_rider(ride_id: str, driver_id: str, stars: int) -> bool`. *Why:* each looks up the ride once and asks it. The ride checks that the right person is acting.
  - `get_ride(ride_id: str) -> Ride or None`. *Why:* the lookup by id.
- Private helper: `reserve_driver(vehicle_type: VehicleType, pickup: Location, excluded: set[str]) -> Driver or None`. *Why hidden:* no outside caller should be able to reserve a driver without making a ride. It is shared by `request_ride` and a re-match after a rejection.
- *Extensibility:* the three rules are all passed in.

## 7. Relationships
- RideService **has** Rider, Driver, Ride (created inside the register and request methods).
- Rider **has** ReputationScore. Driver **has** ReputationScore.
- Driver **has** Vehicle (vehicles die with the driver record).
- Driver **uses** Location (a value that lives on). Ride **uses** Rider, Driver, Location, PaymentMethod.
- DriverInfo **uses** Location. DistanceTimePricing **uses** RateCard.
- RideService **uses** MatchingStrategy, PricingStrategy (passed in).
- DistanceTimePricing **implements** PricingStrategy. NearestDriverStrategy **implements** MatchingStrategy. CardPayment, UpiPayment **implement** PaymentMethod.
- Enums are always **uses**: Vehicle, Ride and DriverInfo to VehicleType; Driver to DriverStatus; Ride to RideStatus.
- `PricingStrategy` also appears as a method input on `Ride`, but no attribute of `Ride` holds it, so there is no line there.

## 8. Interaction

### Layers
1. **Entry point and manager:** RideService. The only class callers use.
2. **Domain classes that guard themselves:** Driver and Ride.
3. **Rules passed in:** MatchingStrategy, PricingStrategy, PaymentMethod and their implementations.
4. **Plain data:** Location, Vehicle, DriverInfo, RateCard, ReputationScore.

### Call chains
- **Request a ride**
  1. `Rider -> RideService.request_ride(...)`.
  2. `RideService -> Driver.snapshot()` for each driver: consistent copies.
  3. `RideService -> MatchingStrategy.select(candidates, pickup)`: the rule chooses.
  4. `RideService -> Driver.try_reserve()`: the driver decides if they are still free. If not, choose again.
  5. `RideService -> Ride(...)`: created last, only once a driver is reserved.
- **Driver responds**
  1. `Driver -> RideService.respond_to_ride(ride_id, driver_id, accept)`.
  2. `RideService -> Ride.accept(driver_id)` or `Ride.reject(driver_id)`.
  3. On reject: `Ride -> Driver.release()`, then the service reserves and assigns the next driver.
- **Complete and pay**
  1. `Driver -> RideService.complete_ride(...)`, then `RideService -> Ride.complete(driver_id, now, pricing)`.
  2. `Ride -> PricingStrategy.calculate_fare(...)`, then `Ride -> Driver.finish_ride(dropoff)`.
  3. `Rider -> RideService.pay_ride(...)`, then `Ride -> PaymentMethod.pay(fare)`.
- **Rate**
  1. `Rider -> RideService.rate_driver(...)`, then `Ride.rate_driver(...)`, then `Driver.add_rating(stars)`.

### Who may call whom
- Riders and drivers call `RideService` only, and never `Ride`, `Driver` methods that change status, or the strategies.
- `RideService` calls `Driver`, `Ride`, `MatchingStrategy`.
- `Ride` calls `Driver`, `Rider`, `PricingStrategy` and `PaymentMethod`.
- `Driver` and `Rider` call only `ReputationScore`. Strategies call no other class.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Register and set availability
1. `register_rider` and `register_driver` use `put_if_absent`, so a repeated id returns None.
2. `add_vehicle`, then `go_online(driver_id, plate_number, location)`, which needs a registered vehicle and the OFFLINE status.
- **Failure:** returns None or False and nothing changes.

### Estimate the fare
1. `estimate_fare(pickup: Location, dropoff: Location, vehicle_type: VehicleType)`. If the two locations are the same, return None.
2. `distance = pickup.distance_to(dropoff)` and `minutes = distance / average_speed_kmph * 60`.
3. Return `pricing.calculate_fare(vehicle_type, distance, minutes)`.
- **Failure:** None for identical locations. Nothing is stored, because an estimate is only a calculation.

### Request a ride
1. `request_ride(rider_id: str, pickup: Location, dropoff: Location, vehicle_type: VehicleType, now: datetime)`. `rider = riders.get(rider_id)`. If None, or the two locations match, return None.
2. `driver: Driver or None = reserve_driver(vehicle_type, pickup, empty set)`. If None, return None. *Why before the ride is built:* a ride must never exist without a driver on offer.
3. `Ride(ride_id, rider, pickup, dropoff, vehicle_type, now, driver)` with the id from `counter`, stored in `rides`. Return it.
- **Failure and undo:** if the ride can't be built, call `driver.release()` so the driver is not left BUSY. No other step changes anything.

### Reserve a driver (private helper)
1. Take `snapshot()` of every driver. Keep the ones that are not None, match `vehicle_type`, and are not in `excluded`.
2. `chosen = matching.select(candidates, pickup)`. If None, return None.
3. `driver = drivers.get(chosen.get_driver_id())`, looked up once. `driver.try_reserve()`. If True, return the driver.
4. If False (another request took them first), remove that candidate and go back to step 2.
- **Failure and undo:** a refused `try_reserve` changes nothing, so there is nothing to undo.

### Driver accepts or rejects
1. `respond_to_ride(ride_id: str, driver_id: str, accept: bool)`. `ride = rides.get(ride_id)`, looked up once. If None, return False.
2. If `accept`: return `ride.accept(driver_id)`.
3. Otherwise `ride.reject(driver_id)`. If False, return False. The driver is released and recorded as rejected.
4. `next = reserve_driver(vehicle type, pickup, ride.get_rejected_driver_ids())`. If a driver is found, `ride.assign(next)`. If `assign` is refused, `next.release()`.
5. If no driver is found, `ride.cancel_unmatched()`. Return True.
- **Failure and undo:** a wrong driver or a wrong stage is refused with no change. Each step either completes or is undone by releasing the one driver it reserved.

### Start, complete and cancel
1. `start_ride` moves ACCEPTED to IN_PROGRESS. `complete_ride` moves IN_PROGRESS to COMPLETED, computing the fare and freeing the driver. `cancel_ride` moves REQUESTED or ACCEPTED to CANCELLED and releases any driver.
- **Failure:** wrong person or wrong stage returns False. A trip in progress can't be cancelled, so a driver is never freed mid-trip by mistake.

### Pay and rate
1. `pay_ride` calls `ride.pay(rider_id, payment)`. If the payment is declined, the ride stays unpaid and another method can be tried.
2. `rate_driver` and `rate_rider` need COMPLETED, are allowed once each, and add the stars to the other person's score.
- **Failure:** returns False and nothing changes. A bad star value does not use up the one rating.

### Concurrency (requirement 10)
- **Driver:** the lock sits inside `Driver`, around every status change. `try_reserve` checks AVAILABLE and sets BUSY as one step, so two requests choosing the same driver can't both win.
- **Ride:** the lock sits inside `Ride`, around every method.
- **Rating and payment:** the locks sit inside `ReputationScore` and each payment class.
- **Maps and ids:** the three maps are thread-safe maps, and ids come from an atomic counter.
- **Lock order:** always `Ride` first, then a `Driver`, a payment or a score, one at a time. `try_reserve` is called with no ride lock held. No method takes them in the other order, so no deadlock is possible.
- **Stale snapshots:** a snapshot can go out of date before `try_reserve`. A refused reservation makes the helper pick again, so nobody is double-booked.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new matching rule (highest rated, balanced load) | A new class implementing `MatchingStrategy`, passed to `RideService` | No. |
| A new pricing rule or new rates | A new class implementing `PricingStrategy`, or new `RateCard` values | No. |
| A new payment method (wallet, cash) | A new class implementing `PaymentMethod`, passed to `pay_ride` | No. |
| A new vehicle type (bike, luxury) | One value in `VehicleType` and a `RateCard` for it | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **A driver is reserved before the ride is created, using `try_reserve`.**
   - *Why:* it is one locked step, so no driver can be given two rides.
   - *Rejected:* creating the ride first and assigning a driver afterwards. A second request could assign the same driver in between.
2. **The estimate is computed on demand, and the final fare is stored once at completion.**
   - *Why:* the estimate and the final fare use the same pricing rule, so there is one rule and no stale stored estimate.
   - *Rejected:* storing the estimate on the ride and later comparing it with the final fare.
3. **Matching, pricing and payment are all passed-in strategies.**
   - *Why:* each of the three can change without editing the service.
   - *Rejected:* one `if` chain per rule inside the service.
4. **`Ride` runs its own stages and checks who is acting.**
   - *Why:* every transition is in one place, so a direct call is still safe.
   - *Rejected:* the service changing the status itself. A second caller could skip a check.
5. **A rating is a total and a count, not a list of ratings.**
   - *Why:* the average is computed, and the rule exists once for riders and drivers.
   - *Rejected:* a stored average, which would be a second copy.

### Edge cases handled
- No driver is nearby or free, so the request returns None and nothing is created.
- Two riders race for the same driver: one wins, the other picks the next best.
- A driver rejects and the next nearest is offered. If everyone rejects, the ride is cancelled.
- A driver who is not the offered one can't accept, start or complete.
- A trip in progress can't be cancelled.
- A declined payment leaves the ride unpaid, and another method can be tried.
- A rating before completion, a second rating, or stars outside 1 to 5 is refused.
- A driver who is busy can't go offline.

### Likely follow-up questions
1. **"How would you add surge pricing?"** It needs a new input, the current demand, so the pricing method gains a parameter. That is one real change to `PricingStrategy`, and then a new class for the surge rule.
2. **"What if a driver never responds?"** Add a time limit to an offer and a sweep that rejects stale offers, the same pattern as expiring holds. It is not built here because no requirement asks for it.
3. **"What if no driver is within the radius?"** Pass a larger radius or a different `MatchingStrategy`. The service doesn't change.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Driver response time limits | See follow-up question 2. |
| Live tracking and road distance | Distance is the straight line from pickup to drop-off, so a real route would change `Location.distance_to` and add tracking. |
| Cancellation fees or driver cancels | Only the rider can cancel, and without a fee. A fee would be a new rule inside `Ride.cancel`. |
| Pooled or shared rides | One rider per ride. |
| Booking for a later time | No requirement asks for it. |
| The rider choosing from several estimates | The rider picks one vehicle type and sees one estimate. |
| Ride history views | `get_ride` reads one ride. A history list would be a new view. |
| Real payment gateways | `pay` is the one place a gateway call goes. |

## 13. Requirements Met
1. Register and set availability: `RideService.register_rider`, `register_driver`, `add_vehicle`, `go_online`, `go_offline`.
2. Request a ride: `RideService.request_ride`.
3. Fare estimate: `RideService.estimate_fare`.
4. Replaceable matching rule: the `MatchingStrategy` interface with `NearestDriverStrategy`.
5. Accept or reject: `RideService.respond_to_ride`, `Ride.accept`, `Ride.reject`.
6. The five stages: `RideStatus` and the guarded methods of `Ride`.
7. Fare from distance, time and vehicle type, swappable: `PricingStrategy` with `DistanceTimePricing`, and `Ride.complete`.
8. Multiple payment methods: `PaymentMethod` with `CardPayment` and `UpiPayment`, and `Ride.pay`.
9. Mutual ratings: `Ride.rate_driver`, `Ride.rate_rider` and `ReputationScore`.
10. Never two rides for one driver: the lock inside `Driver.try_reserve`.

## 14. Requirement Fit Check

**Setup used in every example:** pickup `P` is (0, 0) and drop-off `D` is (6, 8), so the distance is 10 km. The speed for estimates is 30 km/h. SEDAN rates are base 50, 12 per km, 2 per minute. AUTO rates are base 30, 9 per km, 1 per minute. Matching is `NearestDriverStrategy(8)`. Drivers online: `D1` SEDAN at (1, 0), `D2` SEDAN at (3, 4), `D3` AUTO at (0, 2).

1. **Register and set availability**
   - *Success:* `register_driver("D1", "Dev")`, `add_vehicle("D1", "TN01AB1234", SEDAN)` and `go_online("D1", "TN01AB1234", (1, 0))` all succeed, and D1 is AVAILABLE.
   - *Failure:* `register_driver("D1", "Other")` returns None. `go_online("D1", "XX99", ...)` returns False because the plate is not registered.
2. **Request a ride**
   - *Success:* `request_ride("R1", P, D, SEDAN, 09:55)`. D1 and D2 are candidates, D1 is nearest, so D1 is reserved and the Ride is REQUESTED with D1 on offer.
   - *Failure:* an unknown rider returns None. A request for a vehicle type with no driver online returns None, and nothing is created.
3. **Fare estimate**
   - *Success:* `estimate_fare(P, D, SEDAN)`. The distance is 10 km, the estimate is 20 minutes, so the fare is 50 + 12 × 10 + 2 × 20 = 210.
   - *Failure:* `estimate_fare(P, P, SEDAN)` returns None because the locations are the same.
4. **Replaceable matching rule**
   - *Success:* D1 is 1 km from P and D2 is 5 km, so D1 is chosen. A driver 12 km away is outside the 8 km limit and is ignored. Passing a different `MatchingStrategy` to the service changes the rule with no other change.
   - *Failure:* if every driver is outside the radius, `select` returns None and the request returns None.
5. **Accept or reject**
   - *Success:* D1 rejects. D1 becomes AVAILABLE, is recorded as rejected, and D2 is matched and offered the ride. D2 accepts and the ride is ACCEPTED.
   - *Failure:* D3, who was never offered the ride, calls accept and gets False. If every driver rejects, the ride becomes CANCELLED.
6. **The five stages**
   - *Success:* REQUESTED, then ACCEPTED (D2 accepts), then IN_PROGRESS (start at 10:00), then COMPLETED (finish at 10:25).
   - *Failure:* completing before starting returns False. Cancelling after IN_PROGRESS returns False. Cancelling while ACCEPTED is allowed: the ride is CANCELLED and the driver is released.
7. **Fare by distance, time and vehicle type**
   - *Success:* the ride took 25 minutes over 10 km, so the SEDAN fare is 50 + 120 + 50 = 220. An AUTO ride of 10 km and 25 minutes costs 30 + 90 + 25 = 145. Passing a different `PricingStrategy` swaps the rule.
   - *Failure:* `DistanceTimePricing` built without a rate card for SUV is refused at construction, so a fare can never be missing.
8. **Multiple payment methods**
   - *Success:* `pay_ride(..., CardPayment("4111"))` charges 220 and the ride is paid. The same call works with `UpiPayment`.
   - *Failure:* a declined card returns False and the ride stays unpaid, so `UpiPayment` can be tried. Paying again after success returns False.
9. **Ratings**
   - *Success:* after completion, `rate_driver(ride, "R1", 5)` makes the driver's average 5.0, and `rate_rider(ride, "D2", 4)` rates the rider.
   - *Failure:* rating before completion returns False, a second rating returns False, and 6 stars returns False without using up the rating.
10. **Never two rides for one driver**
    - *Success:* two riders request at the same moment with only D1 able to take them. Both choose D1 and both call `try_reserve`. The driver's lock lets one succeed.
    - *Failure:* the other request's `try_reserve` returns False, so it picks again and finds nobody, and returns None. D1 is never on two rides.
