# BookMyShow / Movie Booking: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a seat is one physical seat, but its status belongs to one show, so there is one `ShowSeat` per seat per show, and that object guards itself with a lock. A booking is a record that holds those show seats, and it does its own confirm, cancel and expire, so every path that frees a seat goes through one place.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The system should allow users to search movies by city and view the theaters and shows that run them.
2. Each theater should have multiple screens, and each screen should have a seat layout and show movies at different times.
3. Seats should have different types with different prices.
4. The system should allow users to pick seats for a show and book them.
5. Selected seats should be held temporarily while payment is in progress and released if payment fails or times out.
6. Two users must never be able to book the same seat for the same show.
7. The system should support multiple payment methods.
8. The system should allow users to cancel a booking and free the seats.

**Types and conventions**
- Money is `int`, in the smallest currency unit. Times are `datetime`. The caller passes the current time (`now`) in, so the design never reads a clock and is easy to test.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put`.
- Users are identified only by a `user_id: str`. No requirement asks for registration, so there is no User class.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Movie | Describes one movie | id, title |
| Seat | Describes one physical seat of a screen | seat number, seat type |
| Screen | Holds the seat layout of one screen | its list of seats |
| ShowSeat | Holds one seat's status and price for one show and guards it | status, price |
| Show | Runs one movie on one screen at one time and owns that show's seats | times, its show seats |
| Theater | Owns screens and shows and checks that shows on a screen don't overlap | screens, shows |
| MovieCatalog | Finds movies, theaters and shows, and sets them up | movies, theaters |
| PaymentMethod (interface) | Takes and returns a payment | none |
| CardPayment | Pays and refunds by card | amount paid |
| UpiPayment | Pays and refunds by UPI | amount paid |
| Booking | Holds chosen seats and runs its own confirm, cancel and expire | status, expiry time |
| BookingService | Holds seats all-or-nothing, stores bookings, finds a booking by id, sweeps timed-out holds | the bookings |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Movie, city, theater, show, screen, seat layout, seat, seat type, price, user, booking, payment method, hold.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Movie, Theater, Screen, Seat, Show, Booking are classes. City, price and seat number are attributes. "Hold" is a status, not a class.
2. **Find fixed sets of values.** Seat types, the states of a seat in a show, the stages of a booking.
   - *Why:* a fixed set is an enum, so a typo like "premum" cannot exist.
   - *Result:* `SeatType`, `ShowSeatStatus`, `BookingStatus`.
3. **Give every fact one owner.** Seat type belongs to the physical Seat. Status and price belong to ShowSeat. The booking stage belongs to Booking.
   - *Why:* two copies of one fact will eventually disagree.
   - *Key insight:* the same physical seat A1 is in many shows, so its status can't live on the Seat. It must live per show, in `ShowSeat`.
4. **Mark what can be computed and do not store it.** The booking total is the sum of its seat prices. The available seats of a show are those with status AVAILABLE. Whether a movie plays in a city is found from the shows.
   - *Why:* a stored total beside the seat prices would go wrong the moment one changes.
5. **Give every verb to the class whose data changes.** Hold, confirm and release a seat: ShowSeat. Confirm, cancel and expire a booking: Booking. Check overlapping shows: Theater.
6. **Find the action that touches several classes.** Holding several seats must be all-or-nothing and needs an undo. That is `BookingService.hold_seats`.
7. **Find the hint phrase for a pattern.** "Support multiple payment methods" is Strategy. Nothing else hints at a pattern.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirement 6 names it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** `Booking` or None for a hold, `bool` for confirm and cancel, lists and maps for search.

## 4. Pattern Guide

### Used: Strategy
- **Meaning:** a way of doing something is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrase:** "the system should support multiple payment methods".
- **Without it:** the booking would have an `if` on the payment type, and every new method would mean editing the booking.
- **When NOT to use it:** when there is one fixed way and nobody asked to add another.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A booking's stages only decide which method is allowed, and each method checks the status itself. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. All seats and shows are plain data with a type value. |
| Facade | "a single simple entry point" | Not added as a separate class. `MovieCatalog` and `BookingService` are the two entry points because they are the managers. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. A passed-in catalog and service are easier to test. |

## 5. Enums

### SeatType
- Values: `REGULAR`, `PREMIUM`, `VIP`
- *Why:* requirement 3 says seats have types with different prices. The price is data keyed by this type.
- *Extensibility:* a new seat type is one new value here, plus its price in each show's price table.

### ShowSeatStatus
- Values: `AVAILABLE`, `HELD`, `BOOKED`
- *Why:* requirement 5 needs a "held while paying" condition between free and sold. A seat has exactly one of these three at a time.

### BookingStatus
- Values: `PENDING`, `CONFIRMED`, `CANCELLED`, `EXPIRED`
- *Why:* PENDING is the hold while payment is in progress. CONFIRMED is paid. CANCELLED is the user's cancellation. EXPIRED is a payment that failed or timed out, which keeps it apart from a cancellation the user chose.

## 6. Classes (smallest dependency first)

### Movie
**Why this class exists:** a movie is a thing with its own identity that many shows refer to. It stays separate from Show so many shows share one movie record.
- Attributes:
  - `movie_id: str`. *Why:* the identity used in searches and when adding a show.
  - `title: str`. *Why:* what the user sees.
- Constructor: `Movie(movie_id: str, title: str)`. *Why:* both fields are set once. It refuses an empty id or title.
- Methods: `get_movie_id() -> str`, `get_title() -> str`. *Why:* read-only views. There are no setters.
- *Extensibility:* a new movie is a new Movie value, with no code change.

### Seat
**Why this class exists:** it is the fixed, physical seat of a screen. It has no status, because the same seat is used in many shows.
- Attributes:
  - `seat_number: str`. *Why:* the label on the layout, such as "A1".
  - `seat_type: SeatType`. *Why:* decides the price a show charges for this seat.
- Constructor: `Seat(seat_number: str, seat_type: SeatType)`. *Why:* both fields are set once. It refuses an empty number.
- Methods: `get_seat_number() -> str`, `get_seat_type() -> SeatType`. *Why:* read-only views.
- *Extensibility:* none needed.

### Screen
**Why this class exists:** requirement 2 gives each screen a seat layout. The layout belongs to the screen, so every show on that screen uses the same one.
- Attributes:
  - `screen_code: str`. *Why:* identifies the screen inside its theater.
  - `seats: list[Seat]`. *Why:* the layout. The seat numbers are the layout. Drawing them as a grid is a display concern.
- Constructor: `Screen(screen_code: str, seats: list[Seat])`. *Why:* a screen is meaningless without seats. It keeps its own copy of the list and refuses an empty list or a repeated seat number.
- Methods: `get_screen_code() -> str`, `get_seats() -> list[Seat]` (returns a copy). *Why a copy:* so a caller can't edit the layout.
- *Extensibility:* none needed.

### ShowSeat
**Why this class exists:** requirement 6 needs one object per seat per show that decides "only one user gets this seat". That object owns the status and the lock.
- Attributes:
  - `seat: Seat`. *Why:* which physical seat this is.
  - `price: int`. *Why:* the price for this seat in this show. It is the only copy of the price, set from the show's price table.
  - `status: ShowSeatStatus`. *Why:* the single fact about the seat in this show.
  - `lock: Lock`. *Why:* makes "check available, then hold" one step.
- Constructor: `ShowSeat(seat: Seat, price: int)`. *Why:* a show seat is never created without its seat and price. The status starts AVAILABLE and the lock is created. It refuses `price <= 0`.
- Methods:
  - `get_seat() -> Seat`, `get_price() -> int`, `get_status() -> ShowSeatStatus`. *Why:* read-only views.
  - `try_hold() -> bool`. *Why:* under the lock, AVAILABLE to HELD. False if it is not AVAILABLE. This is what stops two users getting the same seat.
  - `confirm() -> bool`. *Why:* under the lock, HELD to BOOKED. False otherwise.
  - `release() -> bool`. *Why:* under the lock, HELD or BOOKED to AVAILABLE. False if it is already AVAILABLE.
  - There is no setter, because a setter would let anyone skip the checks.
- *Extensibility:* none needed.

### Show
**Why this class exists:** requirement 2 says a screen shows movies at different times. A show ties one movie, one screen and one time together, and owns the show seats for that showing.
- Attributes:
  - `show_id: str`. *Why:* the identity users and bookings use.
  - `movie: Movie`. *Why:* what is playing.
  - `screen: Screen`. *Why:* where it plays and which seat layout it uses.
  - `start_time: datetime`, `end_time: datetime`. *Why:* the time slot, used to refuse overlapping shows and to stop booking a show that has started.
  - `show_seats: map{str: ShowSeat}` (seat number to the ShowSeat for that seat in this show). *Why:* gives a lookup by seat number and keeps every seat's status inside this show.
- Constructor: `Show(show_id: str, movie: Movie, screen: Screen, start_time: datetime, end_time: datetime, prices: map{SeatType: int})`. *Why:* the `prices` map (seat type to the price of one seat of that type) is read once here to create one ShowSeat per screen seat, and is not kept, so the price has one home. It refuses `end_time <= start_time`, or a price table that is missing a seat type used by the screen.
- Methods:
  - `get_show_id() -> str`, `get_movie() -> Movie`, `get_screen() -> Screen`, `get_start_time() -> datetime`, `get_end_time() -> datetime`. *Why:* read-only views.
  - `get_show_seat(seat_number: str) -> ShowSeat or None`. *Why:* the one lookup from a seat number to its show seat.
  - `get_available_seats() -> list[ShowSeat]`. *Why:* computed by filtering on AVAILABLE, never stored.
- *Extensibility:* different prices for different shows are just different price tables.

### Theater
**Why this class exists:** requirement 2 gives a theater multiple screens. The theater is the only one who can check that two shows don't overlap on one of its screens.
- Attributes:
  - `theater_id: str`, `name: str`, `city: str`. *Why:* the identity, the display name, and the city that search uses.
  - `screens: map{str: Screen}` (screen code to the Screen). *Why:* a screen is found by its code.
  - `shows: map{str: Show}` (show id to the Show). *Why:* the theater owns the shows that run on its screens.
  - `lock: Lock`. *Why:* guards both maps so check-then-insert is one step.
- Constructor: `Theater(theater_id: str, name: str, city: str)`. *Why:* a theater needs an id, a name and a city. It starts with no screens or shows and creates its lock. It refuses empty values.
- Methods:
  - `get_theater_id() -> str`, `get_name() -> str`, `get_city() -> str`. *Why:* read-only views.
  - `add_screen(screen_code: str, seats: list[Seat]) -> Screen or None`. *Why:* the theater builds the Screen itself. None if the code exists or the seats are invalid.
  - `add_show(show_id: str, movie: Movie, screen_code: str, start_time: datetime, end_time: datetime, prices: map{SeatType: int}) -> Show or None`. *Why:* under the lock it finds the screen once, refuses an existing id or an overlap with another show on that screen, builds the Show, and stores it. None on any refusal.
  - `get_show(show_id: str) -> Show or None`. *Why:* the lookup by id.
  - `get_shows(movie_id: str, now: datetime) -> list[Show]`. *Why:* computed on request. The shows of that movie that start after `now`, from a copy taken under the lock.
- *Extensibility:* none needed.

### MovieCatalog (browsing and setup)
**Why this class exists:** requirement 1 needs a place that answers "which movies play in this city". It is separate from the booking service because browsing changes nothing, while booking does.
- Attributes:
  - `movies: map{str: Movie}` (movie id to the Movie). *Why:* one record per movie.
  - `theaters: map{str: Theater}` (theater id to the Theater). *Why:* the catalog owns the theaters.
  - `lock: Lock`. *Why:* guards both maps.
- Constructor: `MovieCatalog()`. *Why:* it starts with empty maps and its own lock.
- Methods:
  - `add_movie(movie_id: str, title: str) -> Movie or None`. *Why:* administrator setup. None if the id exists.
  - `add_theater(theater_id: str, name: str, city: str) -> Theater or None`. *Why:* administrator setup. None if the id exists.
  - `add_screen(theater_id: str, screen_code: str, seats: list[Seat]) -> bool`. *Why:* it finds the theater once and asks it to add the screen. False if the theater is unknown or the theater refuses.
  - `add_show(theater_id: str, show_id: str, movie_id: str, screen_code: str, start_time: datetime, end_time: datetime, prices: map{SeatType: int}) -> Show or None`. *Why:* it turns the theater id and movie id into objects once. It refuses a show id that already exists anywhere in the catalog, so a show id is unique across theaters. Then it asks the theater to add the show.
  - `search_movies(city: str, now: datetime) -> list[Movie]`. *Why:* computed. The distinct movies with at least one show after `now` in a theater of that city.
  - `get_shows(city: str, movie_id: str, now: datetime) -> map{str: list[Show]}`. *Why:* answers "which theaters and shows run it". The key is the theater id and the value is that theater's shows of the movie after `now`. Theaters with none are left out.
  - `get_theater(theater_id: str) -> Theater or None`. *Why:* the lookup by id.
  - `get_show(show_id: str) -> Show or None`. *Why:* the booking service has only a show id. It asks each theater and returns the first match.
- *Extensibility:* new cities, theaters, screens, shows and movies are all data entries.

### PaymentMethod (interface)
**Why it is an interface:** requirement 7 asks for multiple payment methods, so the booking depends on these two methods and not on any specific way of paying.
- Methods:
  - `pay(amount: int) -> bool`. *Why a `bool`:* the booking only needs to know whether it was paid.
  - `refund(amount: int) -> bool`. *Why:* requirement 8 frees seats on cancel, and a confirmed booking must give the money back.
- *Extensibility:* a new method is a new class implementing both methods.

### CardPayment (implements PaymentMethod)
**Why this class exists:** the first way to pay, and a real implementation of the interface.
- Attributes: `card_number: str`, `paid_amount: int`, `lock: Lock`. *Why:* the card to charge, how much was paid (so a refund can only return what was paid), and a lock so concurrent calls can't pay or refund twice.
- Constructor: `CardPayment(card_number: str)`. *Why:* `paid_amount` starts at 0. It refuses an empty number. One object is used for one booking.
- Methods:
  - `pay(amount: int) -> bool`. *Why:* False if `amount <= 0` or it was already paid. This is where a real card gateway call would go, and a decline returns False. On success it sets `paid_amount`.
  - `refund(amount: int) -> bool`. *Why:* True only if `paid_amount >= amount` and `amount > 0`, then it reduces `paid_amount`.
- *Extensibility:* none needed.

### UpiPayment (implements PaymentMethod)
**Why this class exists:** the second implementation, which shows the interface really holds more than one way to pay.
- Attributes: `upi_id: str`, `paid_amount: int`, `lock: Lock`. *Why each:* the same reasons as for the card.
- Constructor: `UpiPayment(upi_id: str)`. *Why:* `paid_amount` starts at 0. It refuses an empty id.
- Methods: `pay(amount: int) -> bool` and `refund(amount: int) -> bool`, with the same rules as the card.
- *Extensibility:* none needed.

### Booking
**Why this class exists:** a booking is the record of a user's chosen seats and its stage. It runs its own confirm, cancel and expire because those are the only paths that free or sell its seats, so a direct call from anywhere is still safe.
- Attributes:
  - `booking_id: str`, `user_id: str`. *Why:* the identity and the owner.
  - `show: Show`. *Why:* which showing this booking is for.
  - `show_seats: list[ShowSeat]`. *Why:* the seats it holds. Holding the objects means no later lookup.
  - `status: BookingStatus`. *Why:* the booking's stage.
  - `created_at: datetime`, `expires_at: datetime`. *Why:* the hold time limit. The sweep and the late-payment check both read `expires_at`.
  - `payment: PaymentMethod or None`. *Why:* kept after a successful payment so a cancel can refund through the same object. None until paid.
  - `lock: Lock`. *Why:* makes confirm, cancel and expire run one at a time per booking.
- Constructor: `Booking(booking_id: str, user_id: str, show: Show, show_seats: list[ShowSeat], created_at: datetime, expires_at: datetime)`. *Why:* the seats must already be held by the caller. The status starts PENDING, `payment` starts None and the lock is created. It refuses an empty seat list or `expires_at <= created_at`.
- Methods:
  - `get_booking_id() -> str`, `get_user_id() -> str`, `get_show() -> Show`, `get_status() -> BookingStatus`, `get_expires_at() -> datetime`. *Why:* read-only views.
  - `get_show_seats() -> list[ShowSeat]` (returns a copy). *Why a copy:* so a caller can't change what the booking holds.
  - `get_total() -> int`. *Why:* computed as the sum of the seat prices, never stored.
  - `confirm(payment: PaymentMethod, now: datetime) -> bool`. *Why:* the whole sale in one place, under the lock. Steps in section 9. True only if the booking was PENDING, still inside its hold time, and paid.
  - `cancel() -> bool`. *Why:* PENDING or CONFIRMED only. If CONFIRMED it first calls `payment.refund(total)` and returns False if that fails, so the user's money is never lost. Then it releases every seat and sets CANCELLED.
  - `expire() -> bool`. *Why:* PENDING only. It releases every seat and sets EXPIRED. The sweep and a failed payment both end here.
- *Extensibility:* none needed.

### BookingService (the manager)
**Why this class exists:** holding several seats must be all-or-nothing with an undo, and bookings need one place to be stored and found by id. No single seat or booking can do that.
- Attributes:
  - `bookings: map{str: Booking}` (booking id to the Booking). *Why:* the single record of every booking. It is a thread-safe map.
  - `catalog: MovieCatalog`. *Why:* to find a show from its id.
  - `hold_minutes: int`. *Why:* how long a hold lasts. It is data, so changing it needs no code change.
  - `counter: AtomicInt`. *Why:* makes booking ids unique when users act at once.
  - `lock: Lock`. *Why:* guards only the `bookings` snapshot used by the sweep.
- Constructor: `BookingService(catalog: MovieCatalog, hold_minutes: int)`. *Why:* the catalog is passed in so tests can use a small one. It starts with no bookings. It refuses `hold_minutes <= 0`.
- Methods:
  - `hold_seats(show_id: str, seat_numbers: list[str], user_id: str, now: datetime) -> Booking or None`. *Why it takes ids:* it is an entry point. It looks up the show once and each seat once, then works with objects.
  - `confirm_booking(booking_id: str, payment: PaymentMethod, now: datetime) -> bool`. *Why:* turns the id into a Booking once and asks the booking to confirm itself. False if the id is unknown.
  - `cancel_booking(booking_id: str) -> bool`. *Why:* the same, for cancel.
  - `get_booking(booking_id: str) -> Booking or None`. *Why:* the lookup by id.
  - `expire_stale(now: datetime) -> int`. *Why:* the timeout release. It expires every PENDING booking whose `expires_at` has passed and returns how many.
- *Extensibility:* the hold time, the catalog and the payment method are all passed in.

## 7. Relationships
- MovieCatalog **has** Movie, Theater (created inside `add_movie` and `add_theater`).
- Theater **has** Screen, Show (created inside `add_screen` and `add_show`).
- Screen **has** Seat (the layout dies with the screen).
- Show **has** ShowSeat (show seats are created in the constructor and die with the show).
- BookingService **has** Booking (created inside `hold_seats`).
- Show **uses** Movie, Screen (both are created elsewhere and live on).
- ShowSeat **uses** Seat (the physical seat lives on across shows).
- Booking **uses** Show, ShowSeat, PaymentMethod.
- BookingService **uses** MovieCatalog (passed in).
- CardPayment, UpiPayment **implement** PaymentMethod.
- Enums are always **uses**: Seat to SeatType, ShowSeat to ShowSeatStatus, Booking to BookingStatus.

## 8. Interaction

### Layers
1. **Entry points and managers:** MovieCatalog (browsing and setup) and BookingService (holds and bookings).
2. **Domain classes:** Theater, Show, Booking.
3. **Data owners that guard themselves:** ShowSeat and the payment implementations.
4. **Plain data:** Movie, Seat, Screen.

### Call chains
- **Browse**
  1. `User -> MovieCatalog.search_movies(city, now)`: the movies that play in that city.
  2. `User -> MovieCatalog.get_shows(city, movie_id, now)`: the theaters and shows for one movie.
  3. `MovieCatalog -> Theater.get_shows(movie_id, now)`: each theater filters its own shows.
- **Hold seats**
  1. `User -> BookingService.hold_seats(show_id, seat_numbers, user_id, now)`.
  2. `BookingService -> MovieCatalog.get_show(show_id)`: the one show lookup.
  3. `BookingService -> Show.get_show_seat(seat_number)`: every seat is looked up before any is held, so a bad seat number changes nothing.
  4. `BookingService -> ShowSeat.try_hold()`: the seat decides, under its own lock, whether this user gets it.
  5. `BookingService -> Booking(...)`: created last, only once every seat is held.
- **Confirm**
  1. `User -> BookingService.confirm_booking(booking_id, payment, now)`.
  2. `BookingService -> Booking.confirm(payment, now)`: the booking runs the whole sale.
  3. `Booking -> PaymentMethod.pay(total)`, then `ShowSeat.confirm()`: money is taken, then the seats are sold.
- **Cancel**
  1. `User -> BookingService.cancel_booking(booking_id)`.
  2. `BookingService -> Booking.cancel()`: refund first, then free the seats.
  3. `Booking -> PaymentMethod.refund(total)`, then `ShowSeat.release()`.
- **Timeout sweep**
  1. `Scheduler -> BookingService.expire_stale(now)`.
  2. `BookingService -> Booking.expire()` for each timed-out PENDING booking.

### Who may call whom
- Users, administrators and the scheduler call `MovieCatalog` and `BookingService` only.
- `BookingService` calls `MovieCatalog`, `Show`, `ShowSeat` and `Booking`.
- `Booking` calls `ShowSeat` and `PaymentMethod`.
- `MovieCatalog` calls `Theater`, and `Theater` calls `Show`.
- `ShowSeat`, `Seat`, `Screen`, `Movie` and the payment classes call no other class.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Browse
1. `search_movies(city, now)` and `get_shows(city, movie_id, now)`. Both only read.
- **Failure:** an unknown city or movie gives an empty list or map. Nothing changes.

### Set up (administrator)
1. `add_movie`, `add_theater`, `add_screen`, `add_show`.
2. `add_show` looks up the theater and movie once. It refuses a show id that exists anywhere. Then, under the theater lock, it refuses an unknown screen, a bad time range, an overlap with another show on that screen, or a missing price. Otherwise it builds the Show.
- **Failure:** returns None or False and nothing is created.

### Hold seats
1. `hold_seats(show_id: str, seat_numbers: list[str], user_id: str, now: datetime)`. If the list is empty or repeats a seat number, return None.
2. `show: Show or None = catalog.get_show(show_id)`. If None, or `now >= show.start_time`, return None. *Why:* you can't book a show that is unknown or has started.
3. For each seat number, `show.get_show_seat(n)`. If any is None, return None. *Why before holding:* then a bad number leaves nothing to undo.
4. For each ShowSeat in a fixed order, `try_hold()`. *Why a fixed order:* it makes the undo easy to reason about.
5. If any `try_hold()` returns False, call `release()` on every seat this call already held, and return None. *Undone:* exactly the seats this call held, so the other user's seats are untouched.
6. Take the next number from `counter`, build the `Booking` with `expires_at = now + hold_minutes`, store it in `bookings`, and return it.

### Confirm
1. `confirm_booking(booking_id: str, payment: PaymentMethod, now: datetime)`. `booking: Booking or None = bookings.get(booking_id)`, looked up once. If None, return False.
2. `booking.confirm(payment, now)`, run under the booking lock:
   1. If the status is not PENDING, return False. *Why:* a booking can be confirmed only once.
   2. If `now >= expires_at`, release the seats, set EXPIRED, return False. *Why before paying:* a late payment is never taken.
   3. `payment.pay(total)`. If False, release the seats, set EXPIRED, return False. *Undone:* only the hold, which is exactly what requirement 5 asks for.
   4. `confirm()` on every seat. They are held by this booking, so this can't fail. If one does fail, refund the payment, release the seats, set EXPIRED and return False.
   5. Store the payment, set CONFIRMED, return True.

### Cancel
1. `cancel_booking(booking_id: str)`. Look up once. If None, return False.
2. `booking.cancel()`, under the booking lock:
   1. If the status is not PENDING or CONFIRMED, return False.
   2. If CONFIRMED: `payment.refund(total)`. If False, return False with nothing changed. *Why refund first:* if freeing the seats came first and the refund then failed, the user would lose the seats and the money.
   3. `release()` on every seat, set CANCELLED, return True.

### Timeout sweep
1. `expire_stale(now: datetime) -> int`. Copy the booking list under the lock, then release the lock.
2. For each booking that is PENDING with `now >= expires_at`, call `expire()` and count the True results.
- **Failure:** a booking confirmed meanwhile makes `expire()` return False and it is skipped.

### Concurrency (requirement 6)
- **Seat:** the lock sits inside `ShowSeat`, around `try_hold`, `confirm` and `release`. It makes "check available, then hold" one step, so exactly one user wins a seat.
- **Booking:** the lock sits inside `Booking`, around `confirm`, `cancel` and `expire`. A confirm and an expire can't run on the same booking at once.
- **Payment:** the lock sits inside each payment class, around `pay` and `refund`.
- **Maps:** the lock sits inside `Theater` (screens and shows), `MovieCatalog` (movies and theaters), and `BookingService` (the sweep snapshot). `bookings` is a thread-safe map. Ids come from an atomic counter.
- **Lock order:** a method holds at most two locks at a time, always `MovieCatalog` then `Theater`, or `Booking` then a seat or payment. Nothing takes them in the other order, so no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new payment method (wallet, net banking) | A new class implementing `PaymentMethod`, passed to `confirm_booking` | No. |
| A new seat type (recliner) | One value in `SeatType`, then a price for it in the show's price table | No. It is data. |
| Different prices for weekends or premieres | A different price table when calling `add_show` | No. It is data. |
| A longer or shorter hold time | A different `hold_minutes` when building the service | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **Status lives in `ShowSeat`, not in `Seat`.**
   - *Why:* the same physical seat is in many shows, so one status field would show a seat taken in tomorrow's show because it was booked today.
   - *Rejected:* a status on Seat and resetting it after each show.
2. **A hold is a seat status plus a booking expiry, and a sweep releases timed-out holds.**
   - *Why:* only the booking knows when its hold ends, so the time lives in one place.
   - *Rejected:* storing the expiry time on the seat and treating expired holds as free. The seat and the booking would then disagree about who owns it.
3. **Holding several seats is all-or-nothing, with seat lookups first and an undo of only this call's holds.**
   - *Why:* a user must not end up holding two of three seats.
   - *Rejected:* holding as you go and ignoring a later failure.
4. **The price lives once, on `ShowSeat`, copied from the show's price table at creation.**
   - *Why:* the booking total is then computed from one source.
   - *Rejected:* a price map kept on the Show and a price on each ShowSeat. They would disagree.
5. **`Booking` does its own confirm, cancel and expire.**
   - *Why:* every way to sell or free a seat goes through one place, so a direct call is still safe.
   - *Rejected:* the service freeing seats itself. A second caller could skip the refund or the status change.

### Edge cases handled
- One of three seats already taken: the other two holds are released and the user gets nothing.
- An unknown seat number or show id changes nothing.
- A show that has started can't be booked.
- Payment after the hold has timed out is refused before any money is taken.
- Payment fails: the hold is released and the booking is EXPIRED.
- Confirming twice is refused.
- Cancelling a PENDING booking frees the seats with no refund. Cancelling a CONFIRMED booking refunds first.
- Two shows overlapping on one screen are refused.
- The same seat number repeated in one request is refused.

### Likely follow-up questions
1. **"How do timed-out holds get released automatically?"** A scheduler calls `expire_stale(now)` every minute. `confirm` also checks `expires_at` itself, so a late payment can never succeed even if the sweep has not run yet.
2. **"What if two users tap the same seat at the same moment?"** Each calls `try_hold` on it. The seat's lock lets one succeed. The other call releases whatever else it held and returns None.
3. **"How would you add discount coupons?"** A `Discount` rule passed to `confirm_booking` and applied to the total before `pay`. Today the total is computed inside `Booking.get_total`, so that is the one place that would change.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| User accounts and login | Users are only a `user_id`. Accounts would be a new class. |
| Cancellation deadlines or partial refunds | A cancel always refunds in full. A refund policy would be a new rule inside `cancel`. |
| Showing a grid layout | The layout is the seat numbers. Drawing rows and columns is a display concern. |
| Ticket numbers or QR codes | A confirmed booking's id is the reference. A ticket would be a new record. |
| Real payment gateways | `pay` and `refund` are the one place a gateway call goes. |
| Running the sweep on a timer | The scheduler is outside the design. It only has to call `expire_stale(now)`. |
| Seat recommendations | Not required. It would be a read-only method on `Show`. |
| Notifications | No requirement asks for them. |

## 13. Requirements Met
1. Search movies by city, view theaters and shows: `MovieCatalog.search_movies` and `get_shows`.
2. Theaters with screens, seat layouts and shows at different times: `Theater`, `Screen`, `Show`, and the overlap check in `Theater.add_show`.
3. Seat types with different prices: `Seat` with `SeatType`, and the price table that creates each `ShowSeat.price`.
4. Pick seats and book them: `BookingService.hold_seats` and `confirm_booking`.
5. Hold during payment, release on failure or timeout: `ShowSeat.try_hold`, `Booking.confirm`, `Booking.expire` and `BookingService.expire_stale`.
6. Never the same seat twice for a show: the lock inside `ShowSeat.try_hold`.
7. Multiple payment methods: `PaymentMethod` with `CardPayment` and `UpiPayment`.
8. Cancel and free seats: `BookingService.cancel_booking` and `Booking.cancel`.

## 14. Requirement Fit Check

**Setup used in every example:** city Chennai. Theater `T1` has screen `S1` with seats `A1` (REGULAR), `A2` (REGULAR), `B1` (PREMIUM) and `C1` (VIP). Movie `M1` is "Leo". Show `SH1` runs on `S1` from 18:00 to 20:30 with prices REGULAR 150, PREMIUM 250, VIP 400. The hold time is 10 minutes. Users are `U1` and `U2`. The time now is 17:00.

1. **Search by city, view theaters and shows**
   - *Success:* `search_movies("Chennai", 17:00)` returns [Leo]. `get_shows("Chennai", "M1", 17:00)` returns {"T1": [SH1]}.
   - *Failure:* `search_movies("Delhi", 17:00)` returns an empty list. `get_shows("Chennai", "M9", 17:00)` returns an empty map.
2. **Screens, layouts and shows at different times**
   - *Success:* `add_screen("T1", "S1", seats)` returns True. `add_show` for SH1 returns the show. A second show `SH2` on `S1` from 21:00 to 23:30 is accepted, because it does not overlap SH1.
   - *Failure:* `SH3` on `S1` from 19:00 to 21:00 overlaps SH1, so `add_show` returns None and nothing is created.
3. **Seat types with prices**
   - *Success:* the show seats of SH1 are priced `A1` 150, `B1` 250, `C1` 400.
   - *Failure:* `add_show` with a price table that has no VIP price is refused, so a VIP seat can never exist without a price.
4. **Pick and book seats**
   - *Success:* `hold_seats("SH1", ["A1", "B1"], "U1", 17:00)` holds both, and Booking `BK1` is PENDING with total 150 + 250 = 400. `confirm_booking("BK1", UpiPayment("u1@upi"), 17:05)` pays 400, the seats become BOOKED, and the booking is CONFIRMED.
   - *Failure:* `hold_seats("SH1", ["A2", "Z9"], "U2", 17:00)`. `Z9` does not exist, so it fails at the lookup step and `A2` is never held.
5. **Hold, then release on failure or timeout**
   - *Success:* `hold_seats("SH1", ["A2"], "U2", 17:00)` makes `BK2` PENDING with `A2` HELD until 17:10. Its payment is declined, so `confirm` expires the booking and `A2` is AVAILABLE again.
   - *Failure (timeout):* `BK3` holds `C1` at 17:00. At 17:12, `confirm_booking("BK3", ..., 17:12)` is refused and the seat is released with no money taken. If instead `expire_stale(17:12)` runs first, it returns 1 and frees `C1`.
6. **Never the same seat twice**
   - *Success:* U1 and U2 both call `hold_seats` for `C1` at the same moment. Both call `try_hold` on it. The seat's lock lets one succeed, so that user gets a booking.
   - *Failure:* the other user's `try_hold` returns False. Any other seat that user had already held in the same call is released, and the result is None.
7. **Multiple payment methods**
   - *Success:* `confirm_booking("BK4", CardPayment("4111"), 17:03)` follows exactly the same steps as the UPI case and ends CONFIRMED.
   - *Failure:* if the card is declined, `pay` returns False, the hold is released and the booking is EXPIRED.
8. **Cancel and free seats**
   - *Success:* `cancel_booking("BK1")` on the CONFIRMED booking refunds 400, releases `A1` and `B1`, and the booking is CANCELLED. Cancelling a PENDING booking frees its seats with no refund.
   - *Failure:* `cancel_booking("BK1")` again returns False because the booking is already CANCELLED. `cancel_booking("BK99")` returns False because the id is unknown.
