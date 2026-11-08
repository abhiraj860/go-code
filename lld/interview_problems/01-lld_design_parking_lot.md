# Parking Lot System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a parking spot guards its own occupancy with its own lock, so any number of gates can work at once and exactly one wins each spot. The two rules that may change, pricing and payment, are passed in as swappable objects, so the lot never changes when they do.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The parking lot should have multiple floors, each with spots of different sizes for different vehicle types.
2. The system should allow a vehicle to enter and be assigned an available spot that fits it.
3. The system should issue a ticket on entry that records the vehicle, the spot and the entry time.
4. The system should allow a vehicle to exit after paying a fee based on how long it was parked.
5. The fee should vary by vehicle type, and the pricing rule should be easy to swap.
6. The system should support multiple payment methods.
7. The system should show spot availability and refuse entry when the lot is full.
8. Multiple entry and exit gates should work at the same time without giving the same spot to two vehicles.

**Types and conventions**
- Money is `int`, in the smallest currency unit. Times are `datetime`. The caller passes the time in, so the design never reads a clock and is easy to test.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put`, and an atomic `pop` that removes and returns the value, or None.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Vehicle | Describes one vehicle | plate number, vehicle type |
| Spot | Holds at most one vehicle and guards that with a lock | which vehicle is parked in it |
| Floor | Groups spots and finds a free fitting one on its floor | the list of its spots |
| Ticket | Records one entry | vehicle, spot, entry time |
| Receipt | Reports one completed exit | fee, exit time |
| PricingStrategy (interface) | Turns parked time into a fee | none |
| HourlyPricing | Charges a per-hour rate per vehicle type, rounded up | the rate table |
| PaymentMethod (interface) | Takes a payment of an amount | none |
| CashPayment | Pays with cash handed over | cash given |
| CardPayment | Pays with a card | card number |
| ParkingLot | Runs entry and exit in the right order and shows availability | floors, active tickets |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Parking lot, floor, spot, size, vehicle, vehicle type, ticket, entry time, fee, pricing rule, payment method, gate.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Vehicle, Spot, Floor, Ticket, ParkingLot are classes. Entry time and fee are attributes. A gate has no data of its own, so it is not a class.
2. **Find fixed sets of values.** The kinds of vehicle.
   - *Why:* a fixed set is an enum, so a typo like "cra" cannot exist.
   - *Result:* `VehicleType`. A spot's "size" is the vehicle type it is built for, so one enum covers requirement 1.
3. **Give every fact one owner.** Occupancy belongs to Spot, the spot list to Floor, entry details to Ticket, which tickets are active to ParkingLot.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** "Is the spot free" is `vehicle is None`. "Is the lot full" is "no free fitting spot". The fee is computed at exit.
   - *Why:* a stored "free" flag next to the vehicle would go wrong the moment one is updated and the other is not.
5. **Give every verb to the class whose data changes.** Occupy and release: Spot. Find a spot on a floor: Floor. Compute a fee: the pricing object.
   - *Why:* the owner checks the rule before changing its data.
6. **Find the action that touches several classes.** Exit touches the ticket map, the pricing object, the payment and the spot, and the order matters.
   - *Why:* a multi-step action needs one place that knows the order and the undo.
   - *Result:* `ParkingLot` is the only manager.
7. **Find the hint phrases.** "Pricing rule should be easy to swap" and "multiple payment methods" both point to Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirement 8 names it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** `Ticket` on entry, `Receipt` on exit, and None for any refusal.

## 4. Pattern Guide

### Used: Strategy (twice)
- **Meaning:** a rule is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrases:** "the pricing rule should be easy to swap" and "the system should support multiple payment methods".
- **Without it:** the fee calculation and the payment step would each have an `if` on the type, and every new rule or method would mean editing the lot.
- **When NOT to use it:** when there is one fixed rule that nobody asked to change. A plain method is simpler.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A spot is free or taken, and that is computed from one field, so there is no behavior to swap. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. Vehicles and spots are plain data with a type value. |
| Facade | "a single simple entry point" | Not added as a separate class. `ParkingLot` is already the single entry point because it is the manager. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. No requirement says one lot only, and a passed-in lot is easier to test. |

## 5. Enums

### VehicleType
- Values: `MOTORCYCLE`, `CAR`, `TRUCK`
- *Why:* requirement 1 names different vehicle types. A spot is built for exactly one type, so this one enum also describes spot "sizes".
- *Why no separate size enum:* a second enum plus a table saying which size fits which type would be one more fact to keep in step.
- *Extensibility:* a new vehicle type is one new value here, plus a rate in the pricing table and spots added with `add_spot`.

## 6. Classes (smallest dependency first)

### Vehicle
**Why this class exists:** a vehicle is a thing with its own identity (the plate) and a type. It is separate from Ticket so the same vehicle record can be used on every visit.
- Attributes:
  - `plate_number: str`. *Why:* the identity used on the ticket and the receipt.
  - `vehicle_type: VehicleType`. *Why:* decides which spots fit it and which rate applies.
- Constructor: `Vehicle(plate_number: str, vehicle_type: VehicleType)`. *Why:* both fields are set once, so a vehicle can't change type after it is parked. It refuses an empty plate.
- Methods:
  - `get_plate_number() -> str`. *Why public:* the ticket and receipt show it.
  - `get_vehicle_type() -> VehicleType`. *Why public:* spots and pricing read it.
  - There are no setters.
- *Extensibility:* none needed.

### Spot
**Why this class exists:** requirement 8 needs one place where "only one vehicle gets this spot" is decided. The spot owns that decision, so it has its own lock.
- Attributes:
  - `code: str`. *Why:* a lot-wide unique label such as "1-A1", built from the floor number and the spot number, so the ticket needs no separate floor field.
  - `vehicle_type: VehicleType`. *Why:* the one type this spot is built for.
  - `vehicle: Vehicle or None`. *Why:* the single fact about occupancy. There is no separate "free" flag, because that would be a second copy.
  - `lock: Lock`. *Why:* makes "check free, then occupy" one step.
- Constructor: `Spot(code: str, vehicle_type: VehicleType)`. *Why:* a spot is never created without a type. It starts empty and creates its own lock. It refuses an empty code.
- Methods:
  - `get_code() -> str`, `get_vehicle_type() -> VehicleType`. *Why:* read-only views.
  - `is_free() -> bool`. *Why:* computed from `vehicle is None`, so it can't disagree with the vehicle.
  - `try_occupy(vehicle: Vehicle) -> bool`. *Why:* under the lock it checks the spot is free and the type matches, then stores the vehicle. False means "not this spot", which is not an error. There is no setter, because a setter would let anyone skip the check.
  - `release(vehicle: Vehicle) -> bool`. *Why:* under the lock it succeeds only if that exact vehicle is parked here, then empties the spot. This stops a wrong exit from freeing someone else's spot.
- *Extensibility:* a looser fit rule (a small vehicle in a bigger spot) is one changed condition inside `try_occupy`, and nothing else.

### Floor
**Why this class exists:** requirement 1 has multiple floors. A floor answers "do I have a free spot for this vehicle", so the lot doesn't need to know every spot.
- Attributes:
  - `floor_number: int`. *Why:* identifies the floor and orders the search.
  - `spots: list[Spot]`. *Why:* the spots on this floor in the order they were added. The order gives a predictable first-fit search.
  - `lock: Lock`. *Why:* guards only the list when a spot is added while others are searching.
- Constructor: `Floor(floor_number: int)`. *Why:* a floor is useful only with a number. It starts with no spots and creates its lock. It refuses a number below 0.
- Methods:
  - `get_floor_number() -> int`. *Why:* read-only view.
  - `add_spot(number: str, vehicle_type: VehicleType) -> Spot or None`. *Why:* the floor builds the Spot itself with code `floor_number + "-" + number`, so codes are unique across the lot. None if the number is empty or already used on this floor.
  - `find_and_occupy(vehicle: Vehicle) -> Spot or None`. *Why:* it copies the spot list under the floor lock, releases that lock, then asks each matching spot to `try_occupy`. It returns the first spot that accepted, or None. It never holds the floor lock while a spot lock is taken.
  - `get_availability() -> map{VehicleType: int}`. *Why:* counts free spots per type, computed on request and never stored. Every vehicle type appears, with 0 if none.
- *Extensibility:* none needed.

### Ticket
**Why this class exists:** requirement 3 asks for a record of the vehicle, spot and entry time. Keeping it as its own record means entry data is fixed at entry and never edited.
- Attributes: `ticket_id: str`, `vehicle: Vehicle`, `spot: Spot`, `entry_time: datetime`. *Why each:* exactly the four things requirement 3 names, plus the id the driver presents at exit.
- Constructor: `Ticket(ticket_id: str, vehicle: Vehicle, spot: Spot, entry_time: datetime)`. *Why:* all fields set once, so the entry time can't be altered later to cut the fee.
- Methods: `get_ticket_id() -> str`, `get_vehicle() -> Vehicle`, `get_spot() -> Spot`, `get_entry_time() -> datetime`. *Why:* read-only views.
- *Why there is no "used" flag:* a ticket is active exactly while it is in the lot's active-ticket map, so a flag would be a second copy of that fact.
- *Extensibility:* none needed.

### Receipt
**Why this class exists:** exit needs one consistent answer that says what was charged and when.
- Attributes: `ticket_id: str`, `plate_number: str`, `spot_code: str`, `fee: int`, `exit_time: datetime`. *Why:* the facts a driver needs as proof of payment. They are copied as plain values because the ticket and spot are no longer active.
- Constructor: `Receipt(ticket_id: str, plate_number: str, spot_code: str, fee: int, exit_time: datetime)`. *Why:* all fields set once, so a receipt can't be edited.
- Methods: `get_ticket_id() -> str`, `get_plate_number() -> str`, `get_spot_code() -> str`, `get_fee() -> int`, `get_exit_time() -> datetime`. *Why:* read-only views.
- *Extensibility:* a new receipt field is added here without changing callers.

### PricingStrategy (interface)
**Why it is an interface:** requirement 5 says the rule must be easy to swap, so the lot depends on this one method and not on a specific rule.
- Methods: `calculate_fee(vehicle_type: VehicleType, entry_time: datetime, exit_time: datetime) -> int`. *Why these inputs:* these three facts are all any time-based rule needs. *Why it returns an `int`:* the lot only needs the amount to charge.
- *Extensibility:* a new rule (flat fee, weekend rates, first hour free) is a new class implementing this method.

### HourlyPricing (implements PricingStrategy)
**Why this class exists:** it is the first rule: a per-hour rate for each vehicle type.
- Attributes: `rates: map{VehicleType: int}` (vehicle type to the price per hour). *Why:* requirement 5's "vary by vehicle type" is a table, so a new type is a data entry.
- Constructor: `HourlyPricing(rates: map{VehicleType: int})`. *Why:* it refuses a table that is missing any vehicle type or has a rate of 0 or less, so a fee can never be missing at exit.
- Methods: `calculate_fee(vehicle_type: VehicleType, entry_time: datetime, exit_time: datetime) -> int`. *Why:* it rounds the parked time up to whole hours, with a minimum of 1 hour, then multiplies by the rate. Rounding up is the common parking rule.
- *Extensibility:* rate changes are data. A different rule is a different class.

### PaymentMethod (interface)
**Why it is an interface:** requirement 6 asks for multiple payment methods, so the lot depends on one method and not on any specific way of paying.
- Methods: `pay(amount: int) -> bool`. *Why a `bool`:* the lot only needs to know whether it was paid.
- *Extensibility:* a new method (UPI, wallet) is a new class implementing `pay`.

### CashPayment (implements PaymentMethod)
**Why this class exists:** the simplest way to pay, and the first implementation.
- Attributes: `cash_given: int`. *Why:* the cash handed over at the gate.
- Constructor: `CashPayment(cash_given: int)`. *Why:* set once. It refuses a negative amount.
- Methods: `pay(amount: int) -> bool`. *Why:* True if `cash_given >= amount`, otherwise False. It never changes `cash_given`, because the gate returns the change.
- *Extensibility:* none needed.

### CardPayment (implements PaymentMethod)
**Why this class exists:** the second implementation, which shows that the interface really holds more than one way to pay.
- Attributes: `card_number: str`. *Why:* identifies the card to charge.
- Constructor: `CardPayment(card_number: str)`. *Why:* set once. It refuses an empty number.
- Methods: `pay(amount: int) -> bool`. *Why:* this is where a real card gateway call would go. A decline returns False.
- *Extensibility:* none needed.

### ParkingLot (the manager and only entry point)
**Why this class exists:** entry and exit touch several classes in a fixed order, so one class must run them. It holds the one fact the others can't: which tickets are active.
- Attributes:
  - `floors: map{int: Floor}` (floor number to the Floor). *Why:* gives one lookup by number and an ordered search.
  - `active_tickets: map{str: Ticket}` (ticket id to the Ticket for a vehicle that is inside). *Why:* the single record of who is inside. It is a thread-safe map.
  - `pricing: PricingStrategy`. *Why:* the swappable fee rule.
  - `counter: AtomicInt`. *Why:* makes every ticket id unique even when gates run at once.
  - `lock: Lock`. *Why:* guards only the `floors` map while a floor is added.
- Constructor: `ParkingLot(pricing: PricingStrategy)`. *Why:* the pricing rule is passed in, so it can be swapped and faked in tests. It starts with no floors and no tickets.
- Methods:
  - `add_floor(floor_number: int) -> bool`. *Why:* administrator setup. False if the floor exists.
  - `add_spot(floor_number: int, number: str, vehicle_type: VehicleType) -> bool`. *Why:* administrator setup. It looks up the floor once and asks it to add the spot. False if the floor is unknown or the floor refuses.
  - `enter(vehicle: Vehicle, entry_time: datetime) -> Ticket or None`. *Why public:* a gate calls it. None means the lot is full for this vehicle type.
  - `exit(ticket_id: str, payment: PaymentMethod, exit_time: datetime) -> Receipt or None`. *Why it takes an id:* a driver presents only a ticket id. None means the exit was refused and nothing changed.
  - `get_availability() -> map{VehicleType: int}`. *Why:* adds up each floor's free counts, computed on request.
- *Extensibility:* the pricing rule and the payment method are both passed in, so neither requires a change here.

## 7. Relationships
- ParkingLot **has** Floor (floors are created inside `add_floor`).
- ParkingLot **has** Ticket (tickets are created inside `enter` and removed on exit).
- Floor **has** Spot (spots are created inside `add_spot`).
- Spot **uses** Vehicle (the vehicle exists before and after it parks).
- Ticket **uses** Vehicle, Spot (both live on after the ticket is gone).
- ParkingLot **uses** PricingStrategy (passed in).
- HourlyPricing **implements** PricingStrategy.
- CashPayment, CardPayment **implement** PaymentMethod.
- Enums are always **uses**: Vehicle to VehicleType, Spot to VehicleType, HourlyPricing to VehicleType.
- `Receipt` and `PaymentMethod` appear only as a return type and a method input on the lot, so no attribute holds them and they have no line from `ParkingLot`.

## 8. Interaction

### Layers
1. **Entry point and manager:** ParkingLot. The only class gates and administrators call.
2. **Domain classes:** Floor and Spot. They find and guard spots.
3. **Rules passed in:** PricingStrategy with HourlyPricing, and PaymentMethod with its implementations.
4. **Plain data:** Vehicle, Ticket, Receipt.

### Call chains
- **Enter**
  1. `Gate -> ParkingLot.enter(vehicle, entry_time)`: the single entry point.
  2. `ParkingLot -> Floor.find_and_occupy(vehicle)`: floors are tried in ascending number, so the first free fit is used.
  3. `Floor -> Spot.try_occupy(vehicle)`: the spot decides, under its own lock, whether it takes the vehicle.
  4. `ParkingLot -> Ticket(...)`, then `active_tickets.put`: the ticket is created last, only after a spot is secured.
- **Exit**
  1. `Gate -> ParkingLot.exit(ticket_id, payment, exit_time)`.
  2. `ParkingLot -> active_tickets.pop(ticket_id)`: the one lookup, and it claims the ticket so no other gate can use it.
  3. `ParkingLot -> PricingStrategy.calculate_fee(type, entry, exit)`: the rule decides the amount.
  4. `ParkingLot -> PaymentMethod.pay(fee)`: money is taken before the spot is freed.
  5. `ParkingLot -> Spot.release(vehicle)`: the spot is freed only after payment succeeded.
- **Availability**
  1. `Display -> ParkingLot.get_availability()`.
  2. `ParkingLot -> Floor.get_availability()` on each floor: each floor counts its own free spots.
- **Setup**
  1. `Admin -> ParkingLot.add_floor(n)`, then `add_spot(n, number, type)`.
  2. `ParkingLot -> Floor.add_spot(number, type)`: the floor builds its own spot.

### Who may call whom
- Gates and administrators call `ParkingLot` only, and never `Floor`, `Spot` or the ticket map.
- `ParkingLot` calls `Floor`, `PricingStrategy`, `PaymentMethod` and `Spot.release`.
- `Floor` calls `Spot` only.
- `Spot`, `Ticket`, `Vehicle` and the strategies call no other class.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Enter
1. `enter(vehicle: Vehicle, entry_time: datetime)`.
2. For each floor in ascending floor number: `spot: Spot or None = floor.find_and_occupy(vehicle)`. Stop at the first spot returned. *Why:* the spot itself decides, so two gates can never both get it.
3. If no floor returned a spot, return None. *Why:* this is the "lot is full for this type" refusal, and nothing was changed.
4. `ticket_id` comes from `counter`. Create `Ticket(ticket_id, vehicle, spot, entry_time)` and `active_tickets.put(ticket_id, ticket)`.
5. Return the Ticket.
- **Failure and undo:** the only failure is step 3, which changed nothing. The ticket is created after the spot is secured, so there is never a ticket without a spot.

### Exit
1. `exit(ticket_id: str, payment: PaymentMethod, exit_time: datetime)`. `ticket: Ticket or None = active_tickets.pop(ticket_id)`. If None, return None. *Why pop first:* removing the ticket atomically claims it, so two gates presenting the same ticket can't both exit.
2. If `exit_time` is before the entry time, put the ticket back and return None. *Why:* a negative duration is bad input, and the ticket is still valid.
3. `fee: int = pricing.calculate_fee(vehicle type, entry_time, exit_time)`.
4. `ok: bool = payment.pay(fee)`. If False, put the ticket back and return None. *Undone:* only the claim. *Why this is enough:* nothing else has changed yet, so the vehicle stays parked and can try again.
5. `spot.release(vehicle)`. *Why last:* the spot is freed only once the money is taken. It can't fail here, because this ticket holds exactly this vehicle and only exit releases it.
6. Return `Receipt(ticket_id, plate, spot code, fee, exit_time)`.

### Availability
1. `get_availability()` asks each floor for its counts and adds them per vehicle type.
- **Failure:** none. A type with no spots shows 0.

### Add floor and add spot
1. `add_floor(floor_number)` under the lot lock. False if it exists.
2. `add_spot(floor_number, number, vehicle_type)` finds the floor once and asks it to add the spot. False if the floor is unknown or the spot number is empty or repeated.
- **Failure:** returns False and nothing is created.

### Concurrency (requirement 8)
- **Spot:** the lock sits inside `Spot`, around `try_occupy` and `release`. It makes "check free, then occupy" one step, so exactly one gate wins a spot.
- **Floor:** the lock sits inside `Floor`, around the spot list only. `find_and_occupy` copies the list under it and releases it before touching any spot.
- **Lot:** the lock sits inside `ParkingLot`, around the `floors` map only.
- **Tickets:** `active_tickets` is a thread-safe map. Its atomic `pop` makes the exit claim safe.
- **Ids:** the counter is atomic.
- **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
- **Why not one big lock on the lot:** it would make every gate wait for every other gate, which is slower and unnecessary because only a single spot is ever contested.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new pricing rule (flat fee, weekend rate, first hour free) | A new class implementing `PricingStrategy`, passed to the `ParkingLot` constructor | No. |
| A new payment method (UPI, wallet) | A new class implementing `PaymentMethod`, passed to `exit` | No. |
| A new vehicle type (electric car, bus) | One value in `VehicleType`, a rate in the `HourlyPricing` table, and spots added with `add_spot` | No logic changes. It is data. |
| More floors or spots | Calls to `add_floor` and `add_spot` | No. |

## 11. Interview Notes

### Key decisions
1. **The spot guards its own occupancy with its own lock.**
   - *Why:* only one spot is ever contested, so only that spot should wait.
   - *Rejected:* one lock on the whole lot. Every gate would queue behind every other gate.
2. **Occupancy is one field, `vehicle`, and "free" is computed.**
   - *Why:* one fact in one place.
   - *Rejected:* a `SpotStatus` enum beside the vehicle. They would disagree after one missed update.
3. **Exit pops the ticket first, then pays, and puts it back if payment fails.**
   - *Why:* the pop is atomic, so the same ticket can't be used twice.
   - *Rejected:* look up, pay, then remove. Two gates could both pass the lookup and both charge.
4. **Strategy for the fee and for payment, passed in.**
   - *Why:* the lot never changes when a rule or a method is added.
   - *Rejected:* an `if` on vehicle type inside the lot.
5. **A spot is built for one vehicle type, with an exact match.**
   - *Why:* it needs no size enum and no compatibility table.
   - *Rejected:* size enums plus a "fits" table. It adds a second fact to keep in step.

### Edge cases handled
- The lot is full for this vehicle type, so entry is refused with nothing changed.
- A ticket used twice, or an unknown ticket, is refused.
- A failed payment leaves the vehicle parked and the ticket valid.
- An exit time before the entry time is refused.
- Under an hour is charged as one hour.
- A wrong vehicle can't release someone else's spot.
- Two gates racing for the last spot: one wins and the other is refused.

### Likely follow-up questions
1. **"Can a small vehicle use a bigger spot?"** Yes. Change the one condition inside `Spot.try_occupy` from an equal type to "this spot is at least as big", and search the smallest spots first.
2. **"How would you choose the spot nearest the exit?"** Add a `SpotFinder` interface that `ParkingLot.enter` calls. Today the floor order lives inside `enter`, so that is the one place that would change.
3. **"What if two gates want the last spot?"** Each calls `try_occupy` on it. The spot's lock lets one succeed and the other get False, so the second gate scans on, finds nothing, and the vehicle is refused.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| A lost ticket | It needs a lookup by plate. That would be one more index to keep in step with the ticket map. |
| The same plate entering twice | Not checked. A plate check would be added inside the lot's `enter`. |
| Reserving a spot in advance | No requirement asks for it. It would add a reserved state to `Spot`. |
| A real card gateway | `CardPayment.pay` is the one place a gateway call would go. |
| Spot choice other than lowest floor first | See follow-up question 2. |
| Grace periods or night rates | A new `PricingStrategy` implementation. |
| Availability per floor | `Floor.get_availability` already exists, so only a lot-level view method would be added. |
| Reading the clock | The caller passes the time in, so tests need no clock. |

## 13. Requirements Met
1. Multiple floors with spots for different vehicle types: `ParkingLot.add_floor`, `Floor.add_spot`, `Spot` with `VehicleType`.
2. Assign an available spot that fits: `ParkingLot.enter`, `Floor.find_and_occupy`, `Spot.try_occupy`.
3. Ticket with vehicle, spot and entry time: `Ticket`.
4. Exit after paying a fee based on time: `ParkingLot.exit` with `PricingStrategy.calculate_fee`.
5. Fee by vehicle type, swappable rule: `HourlyPricing.rates` and the `PricingStrategy` interface.
6. Multiple payment methods: `PaymentMethod` with `CashPayment` and `CardPayment`.
7. Show availability and refuse when full: `ParkingLot.get_availability`, and `enter` returns None.
8. Concurrent gates without the same spot twice: the lock inside `Spot`, the atomic `pop` of the ticket, and the atomic counter.

## 14. Requirement Fit Check

**Setup used in every example:** the rates are MOTORCYCLE 10, CAR 20, TRUCK 50 per hour. Floor 1 has spots `1-A1` (CAR) and `1-M1` (MOTORCYCLE). Floor 2 has spot `2-A1` (CAR). Vehicles: KA01, KA02 and KA03 are CARs, MH09 is a MOTORCYCLE, TN01 is a TRUCK.

1. **Floors and spots**
   - *Success:* `add_floor(1)` returns True, `add_spot(1, "A1", CAR)` returns True and creates spot `1-A1`. `add_floor(2)` and `add_spot(2, "A1", CAR)` create `2-A1`.
   - *Failure:* `add_floor(1)` again returns False. `add_spot(5, "A1", CAR)` returns False because floor 5 is unknown. Nothing is created.
2. **Assign a fitting spot**
   - *Success:* `enter(KA01, 09:00)`. Floor 1 is tried first, and `1-A1` is a free CAR spot, so `try_occupy` returns True and the ticket is issued.
   - *Failure:* `enter(TN01, 09:00)`. No floor has a TRUCK spot, so every `try_occupy` is skipped and the result is None. No spot is changed.
3. **Ticket records vehicle, spot and time**
   - *Success:* the ticket for KA01 reads vehicle KA01, spot `1-A1`, entry 09:00.
   - *Failure:* the ticket exists only for a vehicle that got a spot. The refused TN01 has no ticket, so `active_tickets` has no entry for it.
4. **Exit and pay**
   - *Success:* `exit(T1, CashPayment(100), 11:10)`. The parked time is 2h10m, which rounds up to 3 hours, so the CAR fee is 3 × 20 = 60. `pay(60)` is True because 100 ≥ 60, the spot `1-A1` is freed, and the receipt shows fee 60.
   - *Failure:* `exit(T1, CashPayment(30), 11:10)`. `pay(60)` is False, so the ticket is put back, the spot stays occupied, and the result is None. Using the already-used T1 again after a successful exit also returns None.
5. **Fee by type, swappable rule**
   - *Success:* MH09 parked 30 minutes pays 1 hour × 10 = 10, and a CAR parked the same time pays 20. A different `PricingStrategy` passed to the constructor changes every fee with no other change.
   - *Failure:* `HourlyPricing` built without a TRUCK rate is refused at construction, so a missing fee can never appear at an exit.
6. **Multiple payment methods**
   - *Success:* `exit(T2, CardPayment("4111"), 12:00)` charges the fee through the card path, and the receipt is issued the same way as for cash.
   - *Failure:* if the card gateway declines, `pay` returns False, the ticket is put back, and the vehicle stays parked.
7. **Availability and full**
   - *Success:* at the start `get_availability()` is CAR 2, MOTORCYCLE 1, TRUCK 0. After KA01 enters it is CAR 1.
   - *Failure (the refusal):* KA02 enters and takes `2-A1`, so CAR is 0. `enter(KA03, 09:30)` returns None, which is "the lot is full for cars".
8. **Concurrent gates**
   - *Success:* gate A and gate B call `enter(KA02)` and `enter(KA03)` at the same moment, with only `2-A1` free. Both call `try_occupy` on it. The spot's lock lets one succeed, and that gate gets the ticket.
   - *Failure:* the other gate's `try_occupy` returns False. There is no other CAR spot, so its result is None, and the spot is never given to two vehicles.
