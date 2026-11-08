# Airline Management System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a physical seat belongs to an aircraft, but its status belongs to one flight, so every flight has its own `FlightSeat` objects that guard themselves with a lock. A crew member owns a calendar of flight windows and guards it the same way. A booking runs its own stages, payment and refund, and a flight tells its watching passengers about every change of stage. Admin actions and user actions go through two separate managers, so neither becomes one huge class.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
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

**Types and conventions**
- Money is `int`, in the smallest currency unit. Times are `datetime`, and the caller passes the current time (`now`) in. A `date` is a calendar day.
- A time window includes its start and excludes its end.
- A crew member's window for a flight runs from departure to arrival plus a rest period, which is data.
- A hold lasts a set number of minutes. A booking not paid by then is cancelled and its seats freed.
- Check-in opens a set number of hours before departure and closes a set number of minutes before it. Both are data.
- Users are identified only by a `user_id: str`. A passenger is identified within a booking by their document number.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put`.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| SeatClass, SeatStatus, BookingStatus, FlightStatus, CrewRole (enums) | Name the fixed values | none |
| TimeWindow | Describes a start and end and checks overlap | start, end |
| Seat | Describes one physical seat | number, class |
| Aircraft | Holds a seat layout | tail number, model, seats |
| FlightSeat | Holds one seat's status on one flight and guards it | status |
| FlightObserver (interface) | Reacts to a flight changing stage | none |
| PassengerNotifier (interface) | Sends a message to a passenger | none |
| InboxNotifier | Keeps the messages sent to each email address | the messages |
| PassengerSubscription | Watches one flight for one passenger and tells them of changes | who to tell |
| Flight | Holds a flight's route, times, stage and seats, and tells its watchers | times, stage, seats, watchers |
| PricingStrategy (interface) | Prices a seat class on a flight | none |
| ClassMultiplierPricing | Prices each class as a percentage of the base fare | the percentages |
| PaymentMethod (interface) | Takes and returns a payment | none |
| CardPayment, UpiPayment | Pay and refund by card or UPI | amount paid |
| Passenger | Holds a traveller's details | name, document, email |
| BoardingPass | Records a check-in | an unchangeable record |
| BookingItem | Joins one passenger to one seat at one price, and their boarding pass | price, boarding pass |
| Booking | Holds a booking's items and runs its stages, payment, refund and check-in | status, payment, hold expiry |
| CrewMember | Holds a crew member's flight windows and guards them | assignments |
| FlightCatalog | Stores aircraft and flights, and searches them | aircraft, flights |
| CrewRoster | Stores crew members | crew |
| AirlineOperations | Runs admin actions: flights, stages, crew | none of its own |
| BookingService | Runs user actions: hold, pay, cancel, check in | bookings |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Admin, flight, source, destination, time, aircraft, seat layout, seat, class, user, booking, passenger, price, pricing rule, payment method, stage, refund, crew member, check-in, boarding pass.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Flight, Aircraft, Seat, FlightSeat, Booking, Passenger, CrewMember and BoardingPass are classes. Source, destination and price are values.
2. **Find fixed sets of values.** Seat classes, seat statuses, booking stages, flight stages, crew roles.
   - *Result:* `SeatClass`, `SeatStatus`, `BookingStatus`, `FlightStatus`, `CrewRole`.
3. **Give every fact one owner.** A seat's status on a flight belongs to the FlightSeat. A crew member's assignments belong to the CrewMember. A booking's stage, payment and boarding passes belong to the Booking. A flight's stage belongs to the Flight.
   - *Why:* the same seat is used on every flight of that aircraft, so its status can't live on the Seat. A flight's crew list is not stored on the flight either, because it would be a second copy of the crew members' assignments.
4. **Mark what can be computed and do not store it.** A booking's total, a flight's available seats, a flight's crew list, and search results are computed.
5. **Give every verb to the class whose data changes.** Holding a seat changes the FlightSeat. Assigning crew changes the CrewMember. Delaying changes the Flight. Confirming, cancelling and checking in change the Booking.
6. **Find the actions that touch several classes.** Holding seats (all or nothing), cancelling a flight (stage, refunds, crew) and delaying a flight (times, crew windows) each need a manager. Admin actions go to `AirlineOperations` and user actions to `BookingService`.
7. **Find the hint phrases for patterns.** "The pricing rule should be easy to swap" and "multiple payment methods" are Strategy. "Passengers should be notified at each change" is Observer.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirements 6 and 10 need it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A `Booking` or None for a hold, a `bool` for stage changes, a `BoardingPass` or None for check-in.

## 4. Pattern Guide

### Used: Strategy (pricing and payment)
- **Meaning:** a rule is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrases:** "seat prices should differ by class, and the pricing rule should be easy to swap" and "the system should support multiple payment methods".
- **Without it:** the booking code would have an `if` for each pricing rule and each payment kind.

### Used: Observer (flight changes)
- **Meaning:** a flight keeps a list of watchers and tells each of them when its stage changes, without knowing who they are.
- **Hint phrase:** "passengers should be notified at each change".
- **How it avoids circular references:** a watcher is a small `PassengerSubscription` that holds only a passenger's contact details and the notifier, not the booking or the flight.
- **When NOT to use it:** when only one fixed party ever cares.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A flight's or booking's stage only decides which method is allowed, and each method checks it. |
| Factory | "create different kinds of objects by type" | Not used. |
| Facade | "a single simple entry point" | Not one class. Admin and user actions are kept in two managers, so neither becomes a god class. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

- `SeatClass`: `ECONOMY`, `BUSINESS`, `FIRST`. *Why:* requirement 2's classes, and what pricing depends on.
- `SeatStatus`: `AVAILABLE`, `HELD`, `BOOKED`. *Why:* requirement 6's "held while payment is in progress" sits between free and sold.
- `BookingStatus`: `PENDING`, `CONFIRMED`, `CANCELLED`. *Why:* exactly requirement 8's stages. An unpaid or expired hold ends as `CANCELLED`.
- `FlightStatus`: `SCHEDULED`, `DELAYED`, `DEPARTED`, `LANDED`, `CANCELLED`. *Why:* exactly requirement 9's stages.
- `CrewRole`: `PILOT`, `CO_PILOT`, `CABIN_CREW`. *Why:* requirement 10's crew members have roles.

## 6. Classes (smallest dependency first)

### TimeWindow
**Why this class exists:** crew overlap (requirement 10) is about intervals. A start and end, and the overlap rule, belong together.
- Attributes: `start: datetime`, `end: datetime`.
- Constructor: `TimeWindow(start: datetime, end: datetime)`. *Why:* refuses `end <= start`.
- Methods: `get_start()`, `get_end()`, `overlaps(other: TimeWindow) -> bool`. *Why `overlaps`:* `start < other.end` and `other.start < end`.

### Seat and Aircraft
**Why they exist:** requirement 2. A seat is a fixed place with a class, and an aircraft owns its layout.
- `Seat(seat_number: str, seat_class: SeatClass)`: both set once. It has no status, because the same seat is used on many flights. Methods: `get_seat_number() -> str`, `get_seat_class() -> SeatClass`.
- `Aircraft(tail_number: str, model: str, seats: list[Seat])`: it keeps its own copy of the layout and refuses an empty list or a repeated seat number. Methods: `get_tail_number() -> str`, `get_model() -> str`, `get_seats() -> list[Seat]` (a copy).
- *Extensibility:* a new layout is new data.

### FlightSeat
**Why this class exists:** requirement 6 needs one place per seat per flight where "only one passenger gets this seat" is decided.
- Attributes: `seat: Seat`, `status: SeatStatus`, `lock: Lock`.
- Constructor: `FlightSeat(seat: Seat)`. *Why:* starts `AVAILABLE`.
- Methods (each under the lock):
  - `get_seat() -> Seat`, `get_status() -> SeatStatus`.
  - `try_hold() -> bool`. *Why:* `AVAILABLE` to `HELD`. False otherwise. This is what stops two passengers getting one seat.
  - `confirm() -> bool`. *Why:* `HELD` to `BOOKED`.
  - `release() -> bool`. *Why:* `HELD` or `BOOKED` to `AVAILABLE`.
- *Why there is no setter:* each method checks the current status first.

### FlightObserver (interface), PassengerNotifier (interface) and InboxNotifier
**Why they exist:** requirement 9 tells passengers about flight changes. The flight depends only on `FlightObserver`, and sending messages is the outside world behind `PassengerNotifier`.
- `FlightObserver` method: `on_flight_status_changed(flight_number: str, status: FlightStatus, departure: datetime, arrival: datetime) -> None`.
- `PassengerNotifier` method: `send(email: str, message: str) -> None`.
- `InboxNotifier()`: attributes `messages: map{str: list[str]}` (an email address to its messages, oldest first) and `lock: Lock`. `send` adds the message. `get_messages(email: str) -> list[str]` returns a copy. A real email or SMS service would be called here.

### PassengerSubscription (implements FlightObserver)
**Why this class exists:** a watcher must not hold the booking or the flight, or the objects would point at each other. This small object holds only who to tell and how.
- Attributes: `subscription_id: str`, `passenger_name: str`, `email: str`, `notifier: PassengerNotifier`.
- Constructor: `PassengerSubscription(subscription_id: str, passenger_name: str, email: str, notifier: PassengerNotifier)`.
- Methods:
  - `get_subscription_id() -> str`.
  - `on_flight_status_changed(...)`. *Why:* sends a message such as `Hi Asha, flight AI101 is now DELAYED: departs 07:00, arrives 09:45`.

### Flight
**Why this class exists:** requirements 1 and 9. A flight owns its route, times and stage, owns its seats for this flight, and tells its watchers when the stage changes.
- Attributes:
  - `flight_number: str`, `source: str`, `destination: str`. *Why:* the identity and route.
  - `departure: datetime`, `arrival: datetime`. *Why:* the times, which change when delayed.
  - `aircraft: Aircraft`. *Why:* the plane and its layout.
  - `base_fare: int`. *Why:* what the pricing rule starts from.
  - `status: FlightStatus`. *Why:* the stage.
  - `seats: map{str: FlightSeat}` (a seat number to that seat on this flight). *Why:* the status of every seat for this flight.
  - `observers: map{str: FlightObserver}` (a subscription id to its watcher). *Why:* who to tell, found by id so they can be removed.
  - `lock: Lock`. *Why:* guards the times, the stage and the watchers.
- Constructor: `Flight(flight_number: str, source: str, destination: str, departure: datetime, arrival: datetime, aircraft: Aircraft, base_fare: int)`. *Why:* creates one `FlightSeat` per aircraft seat and starts `SCHEDULED`. It refuses `arrival <= departure`, the same source and destination, or a fare of 0 or less.
- Methods:
  - Getters for each field, and `get_window() -> TimeWindow`.
  - `get_flight_seat(seat_number: str) -> FlightSeat or None`.
  - `get_available_seats() -> list[FlightSeat]`. *Why:* computed.
  - `is_bookable(now: datetime) -> bool`. *Why:* `SCHEDULED` or `DELAYED`, and `now` before departure.
  - `subscribe(subscription: FlightObserver, subscription_id: str) -> bool`, `unsubscribe(subscription_id: str) -> bool`.
  - `delay(new_departure: datetime, new_arrival: datetime) -> bool`. *Why:* from `SCHEDULED` or `DELAYED` only, and only to a later departure. It sets the new times and `DELAYED`, then tells the watchers.
  - `depart() -> bool`. *Why:* from `SCHEDULED` or `DELAYED`. Sets `DEPARTED` and tells the watchers.
  - `land() -> bool`. *Why:* from `DEPARTED`. Sets `LANDED` and tells the watchers.
  - `cancel() -> bool`. *Why:* from `SCHEDULED` or `DELAYED`. Sets `CANCELLED` and tells the watchers.
- *How watchers are told:* the stage change happens under the lock. A copy of the watcher list is taken, the lock is released, and then each watcher is called, guarded so one failure doesn't stop the rest. A slow notifier never holds the flight's lock.

### PricingStrategy (interface) and ClassMultiplierPricing
**Why they exist:** requirement 5. Prices differ by class, and the rule must be swappable.
- `PricingStrategy` method: `price_for(seat_class: SeatClass, base_fare: int) -> int`.
- `ClassMultiplierPricing(percentages: map{SeatClass: int})`: the map is a seat class to its price as a percentage of the base fare. It refuses a missing class or a value of 0 or less. `price_for` returns `base_fare * percentage // 100`.
- *Extensibility:* a new rule (demand-based, weekend) is a new class.

### PaymentMethod (interface), CardPayment and UpiPayment
**Why they exist:** requirement 7, and requirement 8's refund on cancel.
- `PaymentMethod` methods: `pay(amount: int) -> bool`, `refund(amount: int) -> bool`.
- `CardPayment(card_number: str)` and `UpiPayment(upi_id: str)`: each keeps `paid_amount: int` and a `lock`. `pay` refuses `amount <= 0` or a second payment. `refund` succeeds only up to what was paid. A real provider call goes inside each.

### Passenger
**Why this class exists:** requirement 4 enters passenger details. They are a fixed record.
- Attributes: `full_name: str`, `document_number: str`, `email: str`.
- Constructor: `Passenger(full_name: str, document_number: str, email: str)`. *Why:* refuses empty values.
- Methods: getters for each field.

### BoardingPass
**Why this class exists:** requirement 11. A boarding pass is a fixed record of a check-in.
- Attributes: `pass_id: str`, `flight_number: str`, `passenger_name: str`, `seat_number: str`, `seat_class: SeatClass`, `departure: datetime`.
- Constructor: `BoardingPass(...)` with each field. *Why:* all set once.
- Methods: getters for each field.

### BookingItem
**Why this class exists:** one booking can hold several passengers. Each passenger has their own seat, price and boarding pass, so those belong together.
- Attributes: `passenger: Passenger`, `flight_seat: FlightSeat`, `price: int`, `boarding_pass: BoardingPass or None`. *Why:* who, which seat, what they paid (fixed when the seat was held), and their pass once checked in.
- Constructor: `BookingItem(passenger: Passenger, flight_seat: FlightSeat, price: int)`. *Why:* the pass starts None.
- Methods: getters, and `set_boarding_pass(boarding_pass: BoardingPass) -> bool`. *Why:* allowed only once. It is only called by its `Booking`, under the booking's lock.

### Booking
**Why this class exists:** requirement 8. A booking runs its own stages, payment and refund, and requirement 11's check-in, so every path that sells, frees or checks in a seat goes through one place.
- Attributes:
  - `booking_id: str`, `user_id: str`, `flight: Flight`. *Why:* the identity, the owner, and the flight.
  - `items: list[BookingItem]`. *Why:* each passenger and their seat.
  - `status: BookingStatus`, `hold_expires_at: datetime`. *Why:* the stage, and the deadline for paying.
  - `payment: PaymentMethod or None`. *Why:* kept after paying, so a cancel can refund through it.
  - `subscription_ids: list[str]`. *Why:* the watchers this booking added to the flight, so it can remove them.
  - `lock: Lock`. *Why:* confirm, cancel and check-in run one at a time.
- Constructor: `Booking(booking_id: str, user_id: str, flight: Flight, items: list[BookingItem], hold_expires_at: datetime)`. *Why:* the seats are already held by the caller. It starts `PENDING` and refuses an empty list.
- Methods (each under the booking lock):
  - Getters, and `get_total() -> int` (computed as the sum of item prices).
  - `confirm(payment: PaymentMethod, now: datetime, notifier: PassengerNotifier) -> bool`. *Why:* `PENDING` only. If `now >= hold_expires_at`, it releases the seats, sets `CANCELLED`, and returns False with no money taken. Then `payment.pay(total)`. If False, it releases the seats, sets `CANCELLED` and returns False. Then it confirms every seat, stores the payment, subscribes a `PassengerSubscription` per passenger to the flight, and sets `CONFIRMED`.
  - `cancel(requester_id: str) -> bool`. *Why:* only the booking's user, and only while the flight has not departed. See `cancel_internal`.
  - `cancel_by_airline() -> bool`. *Why:* used when the flight is cancelled. The same steps without the user check.
  - Private `cancel_internal()`: `PENDING` or `CONFIRMED` only. If `CONFIRMED`, `payment.refund(total)` first, and if that fails, nothing changes and it returns False. Then it releases every seat, unsubscribes its watchers, and sets `CANCELLED`.
  - `expire(now: datetime) -> bool`. *Why:* `PENDING` and past the deadline. Releases the seats and sets `CANCELLED`.
  - `check_in(document_number: str, now: datetime, opens_hours: int, closes_minutes: int, pass_id: str) -> BoardingPass or None`. *Why:* `CONFIRMED` only, the flight must not be cancelled or departed, and `now` must be inside the check-in window counted back from the flight's current departure. If that passenger already has a pass, it is returned again. Otherwise a new pass is made and stored on their item.
- *Lock order:* the booking's lock first, then a seat, a payment or the flight, one at a time.

### CrewMember
**Why this class exists:** requirement 10 needs one place per crew member where "no overlapping flights" is decided.
- Attributes: `crew_id: str`, `name: str`, `role: CrewRole`, `assignments: map{str: TimeWindow}` (a flight number to the window it occupies), `lock: Lock`.
- Constructor: `CrewMember(crew_id: str, name: str, role: CrewRole)`. *Why:* starts with no assignments.
- Methods (each under the lock):
  - Getters, and `is_assigned(flight_number: str) -> bool`.
  - `try_assign(flight_number: str, window: TimeWindow) -> bool`. *Why:* False if already on that flight or any assignment overlaps. Otherwise stores it. This is requirement 10.
  - `reschedule(flight_number: str, new_window: TimeWindow) -> bool`. *Why:* moves an assignment to new times, checking overlap against the others. False if it would clash.
  - `unassign(flight_number: str) -> bool`.

### FlightCatalog
**Why this class exists:** requirement 3's search, and requirement 1's storage of aircraft and flights.
- Attributes: `aircraft: map{str: Aircraft}` (a tail number to the Aircraft), `flights: map{str: Flight}` (a flight number to the Flight). Thread-safe maps.
- Constructor: `FlightCatalog()`.
- Methods:
  - `add_aircraft(aircraft: Aircraft) -> bool`, `add_flight(flight: Flight) -> bool`. *Why:* False if the id exists.
  - `get_aircraft(tail_number: str) -> Aircraft or None`, `get_flight(flight_number: str) -> Flight or None`.
  - `search(source: str, destination: str, day: date) -> list[Flight]`. *Why:* all three must match (the day is the departure date), and only flights that are `SCHEDULED` or `DELAYED`. Sorted by departure.

### CrewRoster
**Why this class exists:** crew members need one place to be found.
- Attributes: `crew: map{str: CrewMember}` (a crew id to the CrewMember).
- Methods: `add(member: CrewMember) -> bool`, `get(crew_id: str) -> CrewMember or None`, `all() -> list[CrewMember]`.

### AirlineOperations (the admin manager)
**Why this class exists:** admin actions such as delaying or cancelling a flight touch the flight, the crew and the bookings, in order. Keeping them apart from user actions avoids one huge class.
- Attributes: `catalog: FlightCatalog`, `roster: CrewRoster`, `bookings: BookingService`, `crew_rest_minutes: int`. *Why:* passed in. The rest period is data.
- Constructor: `AirlineOperations(catalog: FlightCatalog, roster: CrewRoster, bookings: BookingService, crew_rest_minutes: int)`.
- Methods:
  - `add_aircraft(tail_number: str, model: str, seats: list[Seat]) -> bool`.
  - `add_flight(flight_number: str, source: str, destination: str, departure: datetime, arrival: datetime, tail_number: str, base_fare: int) -> Flight or None`. *Why:* looks up the aircraft once, builds the Flight, and stores it.
  - `add_crew(crew_id: str, name: str, role: CrewRole) -> bool`.
  - `assign_crew(flight_number: str, crew_id: str) -> bool`. *Why:* the window is the flight's departure to arrival plus the rest period. `try_assign` decides.
  - `get_crew(flight_number: str) -> list[CrewMember]`. *Why:* computed from crew members who are assigned to it.
  - `delay_flight(flight_number: str, new_departure: datetime, new_arrival: datetime) -> list[str] or None`. *Why:* delays the flight (which tells passengers), then reschedules each assigned crew member. Anyone who would now clash is unassigned, and their ids are returned for the admin to replace. None if the delay is refused.
  - `depart_flight(flight_number: str) -> bool`, `land_flight(flight_number: str) -> bool`. *Why:* ask the flight, which tells passengers.
  - `cancel_flight(flight_number: str) -> bool`. *Why:* cancels the flight (passengers are told), cancels every booking on it with a refund through `BookingService`, and unassigns its crew.

### BookingService (the user manager)
**Why this class exists:** holding several seats must be all-or-nothing with an undo, and bookings need one place to be found.
- Attributes:
  - `catalog: FlightCatalog`, `pricing: PricingStrategy`, `notifier: PassengerNotifier`. *Why:* passed in.
  - `bookings: map{str: Booking}` (a booking id to the Booking). Thread-safe map.
  - `hold_minutes: int`, `check_in_opens_hours: int`, `check_in_closes_minutes: int`. *Why:* data.
  - `counter: AtomicInt`. *Why:* makes booking and pass ids unique.
- Constructor: `BookingService(catalog: FlightCatalog, pricing: PricingStrategy, notifier: PassengerNotifier, hold_minutes: int, check_in_opens_hours: int, check_in_closes_minutes: int)`. *Why:* refuses non-positive numbers, or a close time later than the open time.
- Methods:
  - `search(source: str, destination: str, day: date) -> list[Flight]`. *Why:* requirement 3, from the catalog.
  - `quote(flight_number: str, seat_class: SeatClass) -> int or None`. *Why:* the price a user would pay, from the pricing rule.
  - `hold_seats(user_id: str, flight_number: str, requests: list[(Passenger, str)], now: datetime) -> Booking or None`. *Why:* requirement 4. Each request is a passenger and a seat number.
  - `confirm_booking(booking_id: str, user_id: str, payment: PaymentMethod, now: datetime) -> bool`.
  - `cancel_booking(booking_id: str, user_id: str) -> bool`.
  - `cancel_flight_bookings(flight_number: str) -> int`. *Why:* used by `AirlineOperations` when a flight is cancelled. Returns how many were cancelled.
  - `expire_stale(now: datetime) -> int`. *Why:* frees seats held past their deadline.
  - `check_in(booking_id: str, user_id: str, document_number: str, now: datetime) -> BoardingPass or None`. *Why:* requirement 11.
  - `get_booking(booking_id: str, user_id: str) -> Booking or None`. *Why:* a user sees only their own.

## 7. Relationships
- Aircraft **has** Seat. Flight **has** FlightSeat. Booking **has** BookingItem.
- FlightCatalog **has** Aircraft, Flight. CrewRoster **has** CrewMember. BookingService **has** Booking.
- Flight **uses** Aircraft, FlightObserver. FlightSeat **uses** Seat.
- BookingItem **uses** Passenger, FlightSeat, BoardingPass. Booking **uses** Flight, PaymentMethod.
- CrewMember **uses** TimeWindow. PassengerSubscription **uses** PassengerNotifier.
- AirlineOperations **uses** FlightCatalog, CrewRoster, BookingService. BookingService **uses** FlightCatalog, PricingStrategy, PassengerNotifier.
- PassengerSubscription **implements** FlightObserver. InboxNotifier **implements** PassengerNotifier. ClassMultiplierPricing **implements** PricingStrategy. CardPayment, UpiPayment **implement** PaymentMethod.
- Enums are always **uses**: Seat to SeatClass, FlightSeat to SeatStatus, Booking to BookingStatus, Flight to FlightStatus, CrewMember to CrewRole, BoardingPass to SeatClass.

## 8. Interaction

### Layers
1. **Managers:** AirlineOperations (admins) and BookingService (users).
2. **Stores:** FlightCatalog and CrewRoster.
3. **Classes that guard themselves:** Flight, FlightSeat, Booking, CrewMember.
4. **Rules and outside world:** PricingStrategy, PaymentMethod, PassengerNotifier, and the FlightObserver subscriptions.
5. **Plain data:** Seat, Aircraft, TimeWindow, Passenger, BookingItem, BoardingPass.

### Call chains
- **Book**
  1. `User -> BookingService.hold_seats(...)`, then `FlightCatalog.get_flight`, then `Flight.get_flight_seat` for each seat.
  2. `BookingService -> FlightSeat.try_hold()` for each, then `PricingStrategy.price_for` for each, then `Booking(...)`.
  3. `User -> BookingService.confirm_booking(...)`, then `Booking.confirm(payment, now, notifier)`.
  4. `Booking -> PaymentMethod.pay`, `FlightSeat.confirm`, `Flight.subscribe(PassengerSubscription)`.
- **Flight change**
  1. `Admin -> AirlineOperations.delay_flight(...)`, then `Flight.delay(...)`.
  2. `Flight -> FlightObserver.on_flight_status_changed(...)` for each watcher, then `PassengerNotifier.send`.
  3. `AirlineOperations -> CrewMember.reschedule(...)` for each assigned crew member.
- **Cancel a flight**
  1. `Admin -> AirlineOperations.cancel_flight(...)`, then `Flight.cancel()` (passengers are told).
  2. `AirlineOperations -> BookingService.cancel_flight_bookings(flight_number)`, then `Booking.cancel_by_airline()` for each, which refunds and frees seats.
  3. `AirlineOperations -> CrewMember.unassign(flight_number)` for each assigned crew member.
- **Check in**
  1. `Passenger -> BookingService.check_in(...)`, then `Booking.check_in(...)`, which creates the `BoardingPass`.

### Who may call whom
- Admins call `AirlineOperations` only. Users call `BookingService` only.
- `AirlineOperations` calls the catalog, the roster, `Flight`, `CrewMember` and `BookingService`.
- `BookingService` calls the catalog, `Flight`, `FlightSeat`, `Booking` and the pricing rule.
- `Booking` calls `FlightSeat`, `PaymentMethod` and `Flight`. `Flight` calls its watchers. Watchers call the notifier.
- `BookingService` never calls `AirlineOperations`, and watchers hold no booking or flight, so there are no cycles.

## 9. Flows

### Hold seats (requirements 4 and 6)
1. `hold_seats(user_id, flight_number, requests, now)`. If the list is empty or repeats a seat or a document number, return None.
2. `flight = catalog.get_flight(flight_number)`. If None or `flight.is_bookable(now)` is False, return None.
3. Look up every `FlightSeat` first. If any is missing, return None. *Why first:* a bad seat number leaves nothing to undo.
4. `try_hold()` each seat in a fixed order. If any fails, `release()` every seat this call already held, and return None. *Undone:* exactly this call's holds.
5. Price each seat with `pricing.price_for(seat class, base_fare)`, build the items and the `Booking` with `hold_expires_at = now + hold_minutes`, store it, and return it.

### Confirm (requirements 7 and 8)
1. `confirm_booking(booking_id, user_id, payment, now)`. Look up once. The booking must belong to the user.
2. `booking.confirm(payment, now, notifier)`: the deadline is checked before paying, a failed payment frees the seats, and a success confirms seats and subscribes passengers.

### Cancel a booking (requirement 8)
1. `cancel_booking(booking_id, user_id)` asks `booking.cancel(user_id)`.
2. A confirmed booking is refunded first. Only then are seats freed and watchers removed. *Why refund first:* if the refund failed after the seats were freed, the user would lose both.
- **Failure:** another user, a booking already cancelled, or a flight already departed is refused with no change.

### Flight stages (requirement 9)
1. `delay_flight`, `depart_flight`, `land_flight` and `cancel_flight` each ask the flight to change its stage, which tells every watcher.
2. `delay_flight` then reschedules each crew member, unassigning anyone who would clash and returning their ids.
3. `cancel_flight` then cancels every booking on the flight with a refund, and unassigns its crew.
- **Failure:** an illegal stage change (landing a flight that never departed, delaying a landed flight) is refused with no change and no message.

### Assign crew (requirement 10)
1. `assign_crew(flight_number, crew_id)`. Look up both once.
2. `member.try_assign(flight_number, TimeWindow(departure, arrival + crew_rest_minutes))`.
- **Failure:** an overlap returns False and nothing changes.

### Check in (requirement 11)
1. `check_in(booking_id, user_id, document_number, now)`. Look up once. The booking must belong to the user.
2. `booking.check_in(document_number, now, opens, closes, new pass id)`.
- **Failure:** outside the window, an unknown passenger, or a booking not confirmed returns None. Checking in twice returns the same pass.

### Concurrency (requirements 6 and 10)
- **Seat:** the lock sits inside `FlightSeat`, around `try_hold`, `confirm` and `release`. Two passengers can never hold the same seat on the same flight.
- **Crew:** the lock sits inside `CrewMember`, around every change to `assignments`. Two simultaneous assignments can't overlap.
- **Booking:** the lock sits inside `Booking`, around confirm, cancel, expire and check-in.
- **Flight:** the lock sits inside `Flight`, around its times, stage and watchers, and is released before watchers are called.
- **Payment:** each payment class has its own lock.
- **Maps and ids:** thread-safe maps and an atomic counter.
- **Lock order:** a booking's lock is taken before a seat, payment or flight lock, never the reverse. `AirlineOperations` holds no lock while calling `BookingService`. Each method holds at most two locks at once, so no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new pricing rule (demand-based, weekend) | A new class implementing `PricingStrategy`, passed to `BookingService` | No. |
| A new payment method | A new class implementing `PaymentMethod` | No. |
| A new way to notify passengers (SMS, app push) | A new class implementing `PassengerNotifier` | No. |
| A new seat class (premium economy) | One `SeatClass` value and its percentage in the pricing table | No logic changes. |
| Different hold, check-in or rest times | Different numbers in the managers' constructors | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **Seat status lives in `FlightSeat`, not in `Seat`.**
   - *Why:* one physical seat is on every flight of that aircraft, so one status field would be wrong for all but one flight.
   - *Rejected:* a status on Seat that is reset after each flight.
2. **A crew member owns their calendar and guards it.**
   - *Why:* "no overlapping flights" is decided in one locked step, and a flight's crew list is computed from it, so there is no second copy.
   - *Rejected:* a crew list on each flight, which would need its own overlap checks and could disagree with the crew member.
3. **Passengers watch a flight through small subscription objects.**
   - *Why:* the flight tells everyone without knowing about bookings, and no object points back at its owner.
   - *Rejected:* the flight holding bookings, or the booking service watching every flight, which would create cycles.
4. **The price is fixed when the seat is held.**
   - *Why:* a pricing rule can change at any time, but a user pays the price they were shown.
   - *Rejected:* computing the price again at payment.
5. **Admin and user actions are in two managers.**
   - *Why:* each stays small and has one job.
   - *Rejected:* one `AirlineSystem` class with everything, which grows into a god class.

### Edge cases handled
- One of several chosen seats is taken: the others are released and nothing is booked.
- A hold that passes its deadline is cancelled and the seats freed, and a late payment is refused before money is taken.
- A failed payment frees the seats.
- A cancel refunds before it frees the seats.
- A cancelled flight refunds every booking and frees its crew.
- A delay that makes a crew member clash removes them and reports them.
- A flight that has departed can't be booked, cancelled by a user, or checked into.
- Checking in twice returns the same boarding pass.
- Illegal flight stage changes are refused without notifying anyone.

### Likely follow-up questions
1. **"What if the notifier is slow?"** Watchers are called after the flight's lock is released, so the flight isn't blocked. For many passengers, the notifier would put messages on a queue, as in a notification service.
2. **"How would you stop an aircraft being on two flights at once?"** Give `Aircraft` a calendar like `CrewMember`, and check it in `add_flight` and `delay_flight`. No requirement asks for it here.
3. **"How would you handle overbooking?"** Allow more holds than seats for economy and resolve at check-in. It is a deliberate change to requirement 6, so it would be a separate, explicit rule.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| An aircraft on two flights at once | See follow-up question 2. |
| Changing a booking to another flight | Not required. It is a cancel and a new booking. |
| Partial refunds or cancellation fees | A cancel always refunds in full. A fee would be a rule in `cancel_internal`. |
| Baggage | Not in these requirements. |
| User accounts and admin login | Users and admins are only callers. Permissions would be checked in the managers. |
| Gates and boarding groups | Not required. They would be fields on `BoardingPass`. |
| Running the hold expiry on a timer | Infrastructure outside the design. It only needs to call `expire_stale(now)`. |

## 13. Requirements Met
1. Add and manage flights: `AirlineOperations.add_flight`, `delay_flight`, `cancel_flight`, and `Flight`.
2. A seat layout with classes: `Aircraft`, `Seat` and `SeatClass`.
3. Search by source, destination and date: `FlightCatalog.search` through `BookingService.search`.
4. Book with seats and passenger details: `BookingService.hold_seats` and `confirm_booking`, with `Passenger` and `BookingItem`.
5. Prices by class, swappable rule: `PricingStrategy` with `ClassMultiplierPricing`.
6. No double booking, and a hold during payment: `FlightSeat.try_hold`, `Booking.confirm` and `expire`.
7. Multiple payment methods: `PaymentMethod` with `CardPayment` and `UpiPayment`.
8. Pending, confirmed, cancelled, with a refund: `BookingStatus` and `Booking.confirm` and `cancel_internal`.
9. Flight stages with notifications: `FlightStatus`, the stage methods of `Flight`, `FlightObserver` and `PassengerSubscription`.
10. Crew without overlaps: `CrewMember.try_assign` and `reschedule`, and `AirlineOperations.assign_crew`.
11. Check-in and boarding pass: `Booking.check_in` and `BoardingPass`.

## 14. Requirement Fit Check

**Setup used in every example:** aircraft `A1` (A320) with seats `1A` (FIRST), `2A` (BUSINESS), `10A` and `10B` (ECONOMY). Flight `AI101` from MAA to DEL on 2026-11-20, departing 06:00 and arriving 08:45, base fare 5000. Pricing is ECONOMY 100%, BUSINESS 250%, FIRST 400%. Holds last 10 minutes. Check-in opens 24 hours before and closes 60 minutes before departure. Crew rest is 60 minutes. Crew `P1` (pilot) and `C1` (cabin crew). User `U1` books passengers Asha and Ravi.

1. **Add and manage flights**
   - *Success:* `add_aircraft("A1", ...)` and `add_flight("AI101", "MAA", "DEL", 06:00, 08:45, "A1", 5000)` succeed.
   - *Failure:* a flight arriving before it departs is refused. `add_flight(..., "A9", ...)` returns None because the aircraft is unknown.
2. **Seat layout with classes**
   - *Success:* AI101 has four flight seats, one FIRST, one BUSINESS and two ECONOMY, all `AVAILABLE`.
   - *Failure:* an aircraft with two seats numbered `10A` is refused.
3. **Search**
   - *Success:* `search("MAA", "DEL", 2026-11-20)` returns [AI101].
   - *Failure:* the same search for 2026-11-21 returns an empty list. A cancelled AI101 would not be returned either.
4. **Book with seats and passenger details**
   - *Success:* at 2026-11-18 10:00, `hold_seats("U1", "AI101", [(Asha, "10A"), (Ravi, "10B")], now)` returns booking `B1`, `PENDING`, total 5000 + 5000 = 10000. `confirm_booking("B1", "U1", CardPayment("4111"), 10:05)` makes it `CONFIRMED`.
   - *Failure:* asking for seat `99Z` returns None, and nothing is held. A passenger with an empty name is refused.
5. **Prices by class, swappable rule**
   - *Success:* `quote("AI101", BUSINESS)` is 12500 and `quote("AI101", FIRST)` is 20000. Building `BookingService` with a different `PricingStrategy` changes every quote with no other edit.
   - *Failure:* a `ClassMultiplierPricing` without a FIRST percentage is refused at setup.
6. **No double booking, and a hold during payment**
   - *Success:* two users hold `1A` at the same moment. The seat's lock lets one succeed.
   - *Failure:* the other user's hold returns None. A hold made at 10:00 and paid at 10:12 is refused, its seats are freed, and no money is taken. A declined card frees the seats.
7. **Multiple payment methods**
   - *Success:* confirming with `UpiPayment("u1@upi")` follows exactly the same steps as the card.
   - *Failure:* a declined UPI payment cancels the booking and frees the seats.
8. **Pending, confirmed, cancelled, with a refund**
   - *Success:* B1 goes `PENDING`, then `CONFIRMED`. `cancel_booking("B1", "U1")` refunds 10000, frees `10A` and `10B`, removes the passengers' watchers, and B1 is `CANCELLED`.
   - *Failure:* `cancel_booking("B1", "U2")` returns False. Cancelling after the flight has departed returns False.
9. **Flight stages with notifications**
   - *Success:* `delay_flight("AI101", 07:00, 09:45)` makes it `DELAYED`, and Asha and Ravi each receive a message with the new times. `depart_flight` and `land_flight` send `DEPARTED` and `LANDED` messages. `cancel_flight` instead sends `CANCELLED`, refunds every booking and frees the crew.
   - *Failure:* `land_flight` on a flight that has not departed is refused, and no message is sent.
10. **Crew without overlaps**
    - *Success:* `assign_crew("AI101", "P1")` holds 06:00 to 09:45 (arrival plus rest).
    - *Failure:* `AI202`, from DEL to BOM, 07:30 to 09:30 the same day, overlaps, so `assign_crew("AI202", "P1")` returns False. After AI101 is delayed to arrive 09:45 (window to 10:45), a crew member also on a flight leaving at 10:00 would be unassigned from AI101 and returned to the admin.
11. **Check in and get a boarding pass**
    - *Success:* `check_in("B1", "U1", Asha's document, 2026-11-19 08:00)` returns a boarding pass for AI101, seat `10A`, ECONOMY, departing 06:00.
    - *Failure:* checking in at 2026-11-18 10:00 (more than 24 hours before) returns None. Checking in on a cancelled booking returns None. Asha checking in again returns the same pass.
