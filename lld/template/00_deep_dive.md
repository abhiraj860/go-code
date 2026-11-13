# LLD Deep Detail: 29 Problems

> The companion to [00-RECALL_CARDS.md](00-RECALL_CARDS.md). Revise the recall card first; use this file to explain and defend a design.
> Each card holds: Enums · Class details (job, what it owns, methods, state behaviour, ordered steps) · Relationships · Call flows · Failure and undo · Concurrency · Key decisions (and what was rejected) · Edge cases · Extensibility · Follow-up questions · Out of scope.
---

## Type A: Booking a Shared Resource

#### 01 · Parking Lot
- **Enums:**
  - **VehicleType**: `MOTORCYCLE`, `CAR`, `TRUCK`
- **Class details (job · what it owns · key methods):**
  - **Vehicle**: Describes one vehicle · *owns:* plate number, vehicle type
    - `get_plate_number() -> str`, `get_vehicle_type() -> VehicleType`
  - **Spot**: Holds at most one vehicle and guards that with a lock · *owns:* which vehicle is parked in it
    - `get_code() -> str`, `get_vehicle_type() -> VehicleType`, `is_free() -> bool`, `try_occupy(vehicle: Vehicle) -> bool`, `release(vehicle: Vehicle) -> bool`
  - **Floor**: Groups spots and finds a free fitting one on its floor · *owns:* the list of its spots
    - `get_floor_number() -> int`, `add_spot(number: str, vehicle_type: VehicleType) -> Spot or None`, `find_and_occupy(vehicle: Vehicle) -> Spot or None`, `get_availability() -> map{VehicleType: int}`
  - **Ticket**: Records one entry · *owns:* vehicle, spot, entry time
    - `get_ticket_id() -> str`, `get_vehicle() -> Vehicle`, `get_spot() -> Spot`, `get_entry_time() -> datetime`
  - **Receipt**: Reports one completed exit · *owns:* fee, exit time
    - `get_ticket_id() -> str`, `get_plate_number() -> str`, `get_spot_code() -> str`, `get_fee() -> int`, `get_exit_time() -> datetime`
  - **PricingStrategy**: Turns parked time into a fee
    - `calculate_fee(vehicle_type: VehicleType, entry_time: datetime, exit_time: datetime) -> int`
  - **HourlyPricing**: Charges a per-hour rate per vehicle type, rounded up · *owns:* the rate table
    - `calculate_fee(vehicle_type: VehicleType, entry_time: datetime, exit_time: datetime) -> int`
  - **PaymentMethod**: Takes a payment of an amount
    - `pay(amount: int) -> bool`
  - **CashPayment**: Pays with cash handed over · *owns:* cash given
    - `pay(amount: int) -> bool`
  - **CardPayment**: Pays with a card · *owns:* card number
    - `pay(amount: int) -> bool`
  - **ParkingLot**: Runs entry and exit in the right order and shows availability · *owns:* floors, active tickets
    - `add_floor(floor_number: int) -> bool`, `add_spot(floor_number: int, number: str, vehicle_type: VehicleType) -> bool`, `enter(vehicle: Vehicle, entry_time: datetime) -> Ticket or None`, `exit(ticket_id: str, payment: PaymentMethod, exit_time: datetime) -> Receipt or None`, `get_availability() -> map{VehicleType: int}`
- **Relationships:**
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
- **Call flows:**
  - **Enter**
    1. `Gate -> ParkingLot.enter(vehicle, entry_time)`
    2. `ParkingLot -> Floor.find_and_occupy(vehicle)`
    3. `Floor -> Spot.try_occupy(vehicle)`
    4. `ParkingLot -> Ticket(...)`, then `active_tickets.put`
  - **Exit**
    1. `Gate -> ParkingLot.exit(ticket_id, payment, exit_time)`
    2. `ParkingLot -> active_tickets.pop(ticket_id)`
    3. `ParkingLot -> PricingStrategy.calculate_fee(type, entry, exit)`
    4. `ParkingLot -> PaymentMethod.pay(fee)`
    5. `ParkingLot -> Spot.release(vehicle)`
  - **Availability**
    1. `Display -> ParkingLot.get_availability()`
    2. `ParkingLot -> Floor.get_availability()`
  - **Setup**
    1. `Admin -> ParkingLot.add_floor(n)`, then `add_spot(n, number, type)`
    2. `ParkingLot -> Floor.add_spot(number, type)`
- **Failure and undo (per flow):**
  - **Enter:** the only failure is step 3, which changed nothing. The ticket is created after the spot is secured, so there is never a ticket without a spot.
  - **Availability:** none. A type with no spots shows 0.
  - **Add floor and add spot:** returns False and nothing is created.
- **Concurrency:**
  - **Spot:** the lock sits inside `Spot`, around `try_occupy` and `release`. It makes "check free, then occupy" one step, so exactly one gate wins a spot.
  - **Floor:** the lock sits inside `Floor`, around the spot list only. `find_and_occupy` copies the list under it and releases it before touching any spot.
  - **Lot:** the lock sits inside `ParkingLot`, around the `floors` map only.
  - **Tickets:** `active_tickets` is a thread-safe map. Its atomic `pop` makes the exit claim safe.
  - **Ids:** the counter is atomic.
  - **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
  - **Why not one big lock on the lot:** it would make every gate wait for every other gate, which is slower and unnecessary because only a single spot is ever contested.
- **Key decisions:**
  1. **The spot guards its own occupancy with its own lock.** *Rejected:* one lock on the whole lot. Every gate would queue behind every other gate.
  2. **Occupancy is one field, `vehicle`, and "free" is computed.** *Rejected:* a `SpotStatus` enum beside the vehicle. They would disagree after one missed update.
  3. **Exit pops the ticket first, then pays, and puts it back if payment fails.** *Rejected:* look up, pay, then remove. Two gates could both pass the lookup and both charge.
  4. **Strategy for the fee and for payment, passed in.** *Rejected:* an `if` on vehicle type inside the lot.
  5. **A spot is built for one vehicle type, with an exact match.** *Rejected:* size enums plus a "fits" table. It adds a second fact to keep in step.
- **Edge cases:**
  - The lot is full for this vehicle type, so entry is refused with nothing changed.
  - A ticket used twice, or an unknown ticket, is refused.
  - A failed payment leaves the vehicle parked and the ticket valid.
  - An exit time before the entry time is refused.
  - Under an hour is charged as one hour.
  - A wrong vehicle can't release someone else's spot.
  - Two gates racing for the last spot: one wins and the other is refused.
- **Extensibility (change → what to add):**
  - A new pricing rule (flat fee, weekend rate, first hour free) → A new class implementing `PricingStrategy`, passed to the `ParkingLot` constructor *(edits existing code: No)*
  - A new payment method (UPI, wallet) → A new class implementing `PaymentMethod`, passed to `exit` *(edits existing code: No)*
  - A new vehicle type (electric car, bus) → One value in `VehicleType`, a rate in the `HourlyPricing` table, and spots added with `add_spot` *(edits existing code: No logic changes. It is data)*
  - More floors or spots → Calls to `add_floor` and `add_spot` *(edits existing code: No)*
- **Follow-up questions:**
  1. **"Can a small vehicle use a bigger spot?"** Yes. Change the one condition inside `Spot.try_occupy` from an equal type to "this spot is at least as big", and search the smallest spots first.
  2. **"How would you choose the spot nearest the exit?"** Add a `SpotFinder` interface that `ParkingLot.enter` calls. Today the floor order lives inside `enter`, so that is the one place that would change.
  3. **"What if two gates want the last spot?"** Each calls `try_occupy` on it. The spot's lock lets one succeed and the other get False, so the second gate scans on, finds nothing, and the vehicle is refused.
- **Out of scope (one-line answer if asked):**
  - A lost ticket → It needs a lookup by plate. That would be one more index to keep in step with the ticket map.
  - The same plate entering twice → Not checked. A plate check would be added inside the lot's `enter`.
  - Reserving a spot in advance → No requirement asks for it. It would add a reserved state to `Spot`.
  - A real card gateway → `CardPayment.pay` is the one place a gateway call would go.
  - Spot choice other than lowest floor first → See follow-up question 2.
  - Grace periods or night rates → A new `PricingStrategy` implementation.
  - Availability per floor → `Floor.get_availability` already exists, so only a lot-level view method would be added.
  - Reading the clock → The caller passes the time in, so tests need no clock.

#### 02 · BookMyShow
- **Enums:**
  - **SeatType**: `REGULAR`, `PREMIUM`, `VIP`
  - **ShowSeatStatus**: `AVAILABLE`, `HELD`, `BOOKED`
  - **BookingStatus**: `PENDING`, `CONFIRMED`, `CANCELLED`, `EXPIRED`
- **Class details (job · what it owns · key methods):**
  - **Movie**: Describes one movie · *owns:* id, title
    - `get_movie_id() -> str`, `get_title() -> str`
  - **Seat**: Describes one physical seat of a screen · *owns:* seat number, seat type
    - `get_seat_number() -> str`, `get_seat_type() -> SeatType`
  - **Screen**: Holds the seat layout of one screen · *owns:* its list of seats
    - `get_screen_code() -> str`, `get_seats() -> list[Seat]`
  - **ShowSeat**: Holds one seat's status and price for one show and guards it · *owns:* status, price
    - `get_seat() -> Seat`, `get_price() -> int`, `get_status() -> ShowSeatStatus`, `try_hold() -> bool`, `confirm() -> bool`, `release() -> bool`
  - **Show**: Runs one movie on one screen at one time and owns that show's seats · *owns:* times, its show seats
    - `get_show_id() -> str`, `get_movie() -> Movie`, `get_screen() -> Screen`, `get_start_time() -> datetime`, `get_end_time() -> datetime`, `get_show_seat(seat_number: str) -> ShowSeat or None`, `get_available_seats() -> list[ShowSeat]`
  - **Theater**: Owns screens and shows and checks that shows on a screen don't overlap · *owns:* screens, shows
    - `get_theater_id() -> str`, `get_name() -> str`, `get_city() -> str`, `add_screen(screen_code: str, seats: list[Seat]) -> Screen or None`, `add_show(show_id: str, movie: Movie, screen_code: str, start_time: datetime, end_time: datetime, prices: map{SeatType: int}) -> Show or None`, `get_show(show_id: str) -> Show or None`, `get_shows(movie_id: str, now: datetime) -> list[Show]`
  - **MovieCatalog**: Finds movies, theaters and shows, and sets them up · *owns:* movies, theaters
    - `add_movie(movie_id: str, title: str) -> Movie or None`, `add_theater(theater_id: str, name: str, city: str) -> Theater or None`, `add_screen(theater_id: str, screen_code: str, seats: list[Seat]) -> bool`, `add_show(theater_id: str, show_id: str, movie_id: str, screen_code: str, start_time: datetime, end_time: datetime, prices: map{SeatType: int}) -> Show or None`, `search_movies(city: str, now: datetime) -> list[Movie]`, `get_shows(city: str, movie_id: str, now: datetime) -> map{str: list[Show]}`, `get_theater(theater_id: str) -> Theater or None`, `get_show(show_id: str) -> Show or None`
  - **PaymentMethod**: Takes and returns a payment
    - `pay(amount: int) -> bool`, `refund(amount: int) -> bool`
  - **CardPayment**: Pays and refunds by card · *owns:* amount paid
    - `pay(amount: int) -> bool`, `refund(amount: int) -> bool`
  - **UpiPayment**: Pays and refunds by UPI · *owns:* amount paid
    - `pay(amount: int) -> bool`, `refund(amount: int) -> bool`
  - **Booking**: Holds chosen seats and runs its own confirm, cancel and expire · *owns:* status, expiry time
    - `get_booking_id() -> str`, `get_user_id() -> str`, `get_show() -> Show`, `get_status() -> BookingStatus`, `get_expires_at() -> datetime`, `get_show_seats() -> list[ShowSeat]`, `get_total() -> int`, `confirm(payment: PaymentMethod, now: datetime) -> bool`, `cancel() -> bool`, `expire() -> bool`
  - **BookingService**: Holds seats all-or-nothing, stores bookings, finds a booking by id, sweeps timed-out holds · *owns:* the bookings
    - `hold_seats(show_id: str, seat_numbers: list[str], user_id: str, now: datetime) -> Booking or None`, `confirm_booking(booking_id: str, payment: PaymentMethod, now: datetime) -> bool`, `cancel_booking(booking_id: str) -> bool`, `get_booking(booking_id: str) -> Booking or None`, `expire_stale(now: datetime) -> int`
- **Relationships:**
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
- **Call flows:**
  - **Browse**
    1. `User -> MovieCatalog.search_movies(city, now)`
    2. `User -> MovieCatalog.get_shows(city, movie_id, now)`
    3. `MovieCatalog -> Theater.get_shows(movie_id, now)`
  - **Hold seats**
    1. `User -> BookingService.hold_seats(show_id, seat_numbers, user_id, now)`
    2. `BookingService -> MovieCatalog.get_show(show_id)`
    3. `BookingService -> Show.get_show_seat(seat_number)`
    4. `BookingService -> ShowSeat.try_hold()`
    5. `BookingService -> Booking(...)`
  - **Confirm**
    1. `User -> BookingService.confirm_booking(booking_id, payment, now)`
    2. `BookingService -> Booking.confirm(payment, now)`
    3. `Booking -> PaymentMethod.pay(total)`, then `ShowSeat.confirm()`
  - **Cancel**
    1. `User -> BookingService.cancel_booking(booking_id)`
    2. `BookingService -> Booking.cancel()`
    3. `Booking -> PaymentMethod.refund(total)`, then `ShowSeat.release()`
  - **Timeout sweep**
    1. `Scheduler -> BookingService.expire_stale(now)`
    2. `BookingService -> Booking.expire()`
- **Failure and undo (per flow):**
  - **Browse:** an unknown city or movie gives an empty list or map. Nothing changes.
  - **Set up (administrator):** returns None or False and nothing is created.
  - **Timeout sweep:** a booking confirmed meanwhile makes `expire()` return False and it is skipped.
- **Concurrency:**
  - **Seat:** the lock sits inside `ShowSeat`, around `try_hold`, `confirm` and `release`. It makes "check available, then hold" one step, so exactly one user wins a seat.
  - **Booking:** the lock sits inside `Booking`, around `confirm`, `cancel` and `expire`. A confirm and an expire can't run on the same booking at once.
  - **Payment:** the lock sits inside each payment class, around `pay` and `refund`.
  - **Maps:** the lock sits inside `Theater` (screens and shows), `MovieCatalog` (movies and theaters), and `BookingService` (the sweep snapshot). `bookings` is a thread-safe map. Ids come from an atomic counter.
  - **Lock order:** a method holds at most two locks at a time, always `MovieCatalog` then `Theater`, or `Booking` then a seat or payment. Nothing takes them in the other order, so no deadlock is possible.
- **Key decisions:**
  1. **Status lives in `ShowSeat`, not in `Seat`.** *Rejected:* a status on Seat and resetting it after each show.
  2. **A hold is a seat status plus a booking expiry, and a sweep releases timed-out holds.** *Rejected:* storing the expiry time on the seat and treating expired holds as free. The seat and the booking would then disagree about who owns it.
  3. **Holding several seats is all-or-nothing, with seat lookups first and an undo of only this call's holds.** *Rejected:* holding as you go and ignoring a later failure.
  4. **The price lives once, on `ShowSeat`, copied from the show's price table at creation.** *Rejected:* a price map kept on the Show and a price on each ShowSeat. They would disagree.
  5. **`Booking` does its own confirm, cancel and expire.** *Rejected:* the service freeing seats itself. A second caller could skip the refund or the status change.
- **Edge cases:**
  - One of three seats already taken: the other two holds are released and the user gets nothing.
  - An unknown seat number or show id changes nothing.
  - A show that has started can't be booked.
  - Payment after the hold has timed out is refused before any money is taken.
  - Payment fails: the hold is released and the booking is EXPIRED.
  - Confirming twice is refused.
  - Cancelling a PENDING booking frees the seats with no refund. Cancelling a CONFIRMED booking refunds first.
  - Two shows overlapping on one screen are refused.
  - The same seat number repeated in one request is refused.
- **Extensibility (change → what to add):**
  - A new payment method (wallet, net banking) → A new class implementing `PaymentMethod`, passed to `confirm_booking` *(edits existing code: No)*
  - A new seat type (recliner) → One value in `SeatType`, then a price for it in the show's price table *(edits existing code: No. It is data)*
  - Different prices for weekends or premieres → A different price table when calling `add_show` *(edits existing code: No. It is data)*
  - A longer or shorter hold time → A different `hold_minutes` when building the service *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"How do timed-out holds get released automatically?"** A scheduler calls `expire_stale(now)` every minute. `confirm` also checks `expires_at` itself, so a late payment can never succeed even if the sweep has not run yet.
  2. **"What if two users tap the same seat at the same moment?"** Each calls `try_hold` on it. The seat's lock lets one succeed. The other call releases whatever else it held and returns None.
  3. **"How would you add discount coupons?"** A `Discount` rule passed to `confirm_booking` and applied to the total before `pay`. Today the total is computed inside `Booking.get_total`, so that is the one place that would change.
- **Out of scope (one-line answer if asked):**
  - User accounts and login → Users are only a `user_id`. Accounts would be a new class.
  - Cancellation deadlines or partial refunds → A cancel always refunds in full. A refund policy would be a new rule inside `cancel`.
  - Showing a grid layout → The layout is the seat numbers. Drawing rows and columns is a display concern.
  - Ticket numbers or QR codes → A confirmed booking's id is the reference. A ticket would be a new record.
  - Real payment gateways → `pay` and `refund` are the one place a gateway call goes.
  - Running the sweep on a timer → The scheduler is outside the design. It only has to call `expire_stale(now)`.
  - Seat recommendations → Not required. It would be a read-only method on `Show`.
  - Notifications → No requirement asks for them.

#### 15 · Library Management
- **Enums:**
  - **Role**: `LIBRARIAN`, `MEMBER`
  - **Action**: `MANAGE_BOOKS`, `BORROW`, `RETURN`, `RESERVE`, `SEARCH`
  - **CopyStatus**: `AVAILABLE`, `BORROWED`, `RESERVED`, `LOST`
- **Class details (job · what it owns · key methods):**
  - **BookCopy**: Is one physical copy and guards its own status · *owns:* status, who it is held for
    - `get_copy_id() -> str`, `get_status() -> CopyStatus`, `get_held_for() -> str or None`, `borrow(member_id: str) -> bool`, `return_copy() -> bool`, `hold_for(member_id: str) -> bool`, `release_hold() -> bool`, `mark_lost() -> bool`
  - **ReservationObserver**: Reacts when a reserved book is ready
    - `on_book_available(member_id: str, isbn: str, copy_id: str) -> None`
  - **MemberNotifier**: Keeps the messages sent to each member · *owns:* the messages
    - `on_book_available(member_id: str, isbn: str, copy_id: str) -> None`, `get_messages(member_id: str) -> list[str]`
  - **Book**: Owns its details, its copies and its waiting list · *owns:* details, copies, waiting list
    - `get_isbn() -> str`, `get_title() -> str`, `get_author() -> str`, `get_subject() -> str`, `update_details(title: str, author: str, subject: str) -> bool`, `add_copies(count: int) -> bool`, `get_copy(copy_id: str) -> BookCopy or None`, `count_available() -> int`, `has_active_copies() -> bool`, `find_borrowable_copy(member_id: str) -> BookCopy or None`, `add_reservation(member_id: str) -> bool`, `cancel_reservation(member_id: str) -> bool`, `return_copy(copy_id: str) -> bool`, `mark_lost(copy_id: str) -> bool`
  - **SearchCriterion**: Decides whether a book matches a search
    - `matches(book: Book) -> bool`
  - **TitleContains**: Matches on part of the title · *owns:* the text
    - `matches(book: Book) -> bool`
  - **Membership**: Holds when a membership ends and how many books it allows · *owns:* expiry date, loan limit
    - `get_expires_on() -> date`, `get_max_loans() -> int`, `is_active(today: date) -> bool`
  - **User**: Is a librarian or a member · *owns:* role, membership
    - `get_user_id() -> str`, `get_role() -> Role`, `get_membership() -> Membership or None`
  - **PermissionPolicy**: Says which role may do which action · *owns:* the permission table
    - `allows(role: Role, action: Action) -> bool`
  - **LateFeePolicy**: Works out a late fee · *owns:* fee per day, grace days
    - `calculate(due_on: date, returned_on: date) -> int`
  - **Loan**: Records one borrowing · *owns:* due date, return date, late fee
    - `get_loan_id() -> str`, `get_member_id() -> str`, `get_book() -> Book`, `get_copy_id() -> str`, `get_due_on() -> date`, `get_late_fee() -> int`, `is_open() -> bool`, `close(returned_on: date, policy: LateFeePolicy) -> int or None`
  - **LibraryService**: Runs each action in the right order and holds users, books and loans · *owns:* users, books, loans
    - `add_librarian(user_id: str, name: str) -> User or None`, `register_member(user_id: str, name: str, today: date) -> User or None`, `add_book(actor_id: str, isbn: str, title: str, author: str, subject: str, copy_count: int) -> Book or None`, `update_book(actor_id: str, isbn: str, title: str, author: str, subject: str) -> bool`, `add_copies(actor_id: str, isbn: str, count: int) -> bool`, `remove_book(actor_id: str, isbn: str) -> bool`, `mark_copy_lost(actor_id: str, isbn: str, copy_id: str) -> bool`, `search(actor_id: str, criterion: SearchCriterion) -> list[Book]`, `borrow_book(member_id: str, isbn: str, today: date) -> Loan or None`, `return_book(actor_id: str, loan_id: str, today: date) -> int or None`, `reserve_book(member_id: str, isbn: str, today: date) -> bool`, `cancel_reservation(member_id: str, isbn: str) -> bool`, `get_fees_due(member_id: str) -> int`
- **Relationships:**
  - LibraryService **has** User, Book, Loan (created inside the add, register and borrow methods).
  - Book **has** BookCopy (the copies die with the book).
  - User **has** Membership (a membership belongs to one member).
  - Book **uses** ReservationObserver (passed in).
  - Loan **uses** Book (the book lives on after the loan).
  - LibraryService **uses** PermissionPolicy, LateFeePolicy, ReservationObserver (passed in).
  - TitleContains, AuthorContains, SubjectContains **implement** SearchCriterion. MemberNotifier **implements** ReservationObserver.
  - Enums are always **uses**: BookCopy to CopyStatus, User to Role, PermissionPolicy to Role and Action.
  - `SearchCriterion` appears only as a method input, so no attribute holds it and it has no line.
- **Call flows:**
  - **Search**
    1. `User -> LibraryService.search(actor_id, criterion)`
    2. `LibraryService -> PermissionPolicy.allows(role, SEARCH)`, then `SearchCriterion.matches(book)`
  - **Borrow**
    1. `Member -> LibraryService.borrow_book(member_id, isbn, today)`
    2. `LibraryService -> PermissionPolicy.allows(role, BORROW)`
    3. `LibraryService -> Book.find_borrowable_copy(member_id)`, then `BookCopy.borrow(member_id)`
    4. `LibraryService -> Loan(...)`
  - **Return**
    1. `Member -> LibraryService.return_book(actor_id, loan_id, today)`
    2. `LibraryService -> Loan.close(today, fee_policy)`
    3. `LibraryService -> Book.return_copy(copy_id)`, then `BookCopy.return_copy()`
    4. `Book -> BookCopy.hold_for(next member)` and `ReservationObserver.on_book_available(...)`
  - **Reserve**
    1. `Member -> LibraryService.reserve_book(...)`, then `Book.add_reservation(member_id)`
- **Failure and undo (per flow):**
  - **Manage books:** a refusal changes nothing.
  - **Register a member:** None, and nothing is created.
  - **Search:** no match returns an empty list. Nothing changes.
  - **Borrow:** steps 1 to 5 change nothing. The copy is changed only in step 6, and nothing after it can fail, so there is nothing to undo.
  - **Return:** a closed loan or a wrong actor is refused with no change. If the book refuses step 4 (it should not), the loan is already closed, and the fee stays recorded.
  - **Reserve and cancel a reservation:** False, with nothing changed. Reserving a book that has an available copy is refused, because the member should just borrow it.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added.
  - If it were needed later, one lock inside `Book` around its copies and waiting list would protect the shared facts, and a lock inside `LibraryService` around the loan count and the borrow step would stop a member passing the limit.
- **Key decisions:**
  1. **A book and a copy are separate classes.** *Rejected:* one class with a copy count. It could not say which copy is lost or reserved.
  2. **The return goes through `Book`, which serves the waiting list.** *Rejected:* the service freeing the copy and then checking the queue. A direct call would skip the queue.
  3. **A member's loans are not stored on the member.** *Rejected:* a loan list on `User` kept in step with the loans.
  4. **A permission is a role and an action in a table.** *Rejected:* `if role == LIBRARIAN` checks scattered through the service.
  5. **The late fee is calculated by a policy and stored once on the loan.** *Rejected:* recomputing the fee each time it is read.
- **Edge cases:**
  - A member with an expired membership, or at the loan limit, can't borrow.
  - A reserved copy can be borrowed only by the member it is held for.
  - A new copy added while people are waiting goes to the first person waiting.
  - Reserving a book that has an available copy is refused.
  - A member can't reserve the same book twice.
  - A librarian can't borrow, and a member can't change the catalogue.
  - A book with a borrowed or reserved copy can't be removed.
  - A loan can't be returned twice.
  - A return on the due date costs nothing, and the grace days are respected.
  - Cancelling a reservation that holds a copy passes the copy to the next person.
- **Extensibility (change → what to add):**
  - A new search field (publisher, year) → A new class implementing `SearchCriterion` *(edits existing code: No)*
  - A new kind of notice (email, SMS) → A new class implementing `ReservationObserver`, passed to the service *(edits existing code: No)*
  - A new permission or a new role rule → A new entry in the `grants` table passed to `PermissionPolicy` *(edits existing code: No. It is data)*
  - Different fees, loan length or limits → Different numbers in `LateFeePolicy` and the service constructor *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"What happens if a reserved copy is never picked up?"** Add an expiry date to the hold and a sweep that releases it to the next person. It is not built here because no requirement asks for it.
  2. **"What if a borrowed copy is lost?"** Add a lost-while-borrowed path: close the loan with a replacement fee and set the copy to `LOST`. Today only an available copy can be marked lost.
  3. **"How would you stop a member borrowing the same book twice?"** Check the member's open loans for that ISBN in step 3 of the borrow. It is a new rule, because the requirements don't ask for it.
- **Out of scope (one-line answer if asked):**
  - A hold that expires → See follow-up question 1.
  - A borrowed copy reported lost → See follow-up question 2.
  - Renewing a loan or a membership → Not required. A renewal would change a loan's due date or a membership's expiry.
  - Paying a late fee → A fee is charged and recorded. Payment is not a requirement.
  - Selling books (the bookstore side) → The requirements describe lending. Selling would be a new set of classes.
  - Blocking a member who owes fees → Not required. It would be a check in step 3 of the borrow.
  - Concurrent users → The Concurrency list above gives the lock placement.
  - Real email or SMS notices → The notifier is the one place a real call goes.

#### 18 · Amazon Locker
- **Enums:**
  - **Size**: `SMALL`, `MEDIUM`, `LARGE`, with ranks 1, 2 and 3. Method: `get_rank() -> int`
  - **CompartmentStatus**: `AVAILABLE`, `OCCUPIED`, `OUT_OF_SERVICE`
  - **AssignmentStatus**: `WAITING`, `COLLECTED`, `EXPIRED`
- **Class details (job · what it owns · key methods):**
  - **Package**: Describes one package · *owns:* id, size, customer
    - `get_package_id() -> str`, `get_size() -> Size`, `get_customer_id() -> str`
  - **Compartment**: Holds at most one package and guards that with a lock · *owns:* the package inside, out-of-service flag
    - `get_compartment_id() -> str`, `get_size() -> Size`, `get_status() -> CompartmentStatus`, `fits(size: Size) -> bool`, `try_occupy(package: Package) -> bool`, `release(package_id: str) -> Package or None`, `take_out_of_service() -> bool`, `return_to_service() -> bool`
  - **CodeGenerator**: Makes a candidate access code
    - `next_code() -> str`
  - **CustomerNotifier**: Sends a code to a customer
    - `send_code(customer_id: str, code: str, compartment_id: str, expires_at: datetime) -> bool`
  - **Assignment**: Joins a code to a package and its compartment, and decides collected or expired once · *owns:* status, expiry time
    - `get_code() -> str`, `get_package() -> Package`, `get_compartment() -> Compartment`, `get_status() -> AssignmentStatus`, `get_expires_at() -> datetime`, `is_expired(now: datetime) -> bool`, `collect(now: datetime) -> bool`, `expire(now: datetime) -> bool`, `cancel() -> None`
  - **LockerStation**: Deposits, collects and expires packages, and manages compartments · *owns:* compartments, active codes
    - `add_compartment(compartment_id: str, size: Size) -> bool`, `deposit(package_id: str, size: Size, customer_id: str, now: datetime) -> str or None`, `collect(code: str, now: datetime) -> Package or None`, `expire_stale(now: datetime) -> list[Package]`, `take_out_of_service(compartment_id: str) -> bool`, `return_to_service(compartment_id: str) -> bool`, `get_compartment_status(compartment_id: str) -> CompartmentStatus or None`, `find_compartment(compartment_id: str) -> Compartment or None`
- **Relationships:**
  - LockerStation **has** Compartment (created inside `add_compartment`), Assignment (created inside `deposit`).
  - Compartment **uses** Package (the package lives on after it leaves).
  - Assignment **uses** Package, Compartment.
  - LockerStation **uses** CodeGenerator, CustomerNotifier (passed in).
  - RandomCodeGenerator **implements** CodeGenerator. InboxNotifier **implements** CustomerNotifier.
  - Enums are always **uses**: Package to Size, Compartment to Size, Assignment to AssignmentStatus.
  - `CompartmentStatus` appears only as a return type, so it has no line.
- **Call flows:**
  - **Deposit**
    1. `Agent -> LockerStation.deposit(package_id, size, customer_id, now)`
    2. `LockerStation -> Compartment.try_occupy(package)`
    3. `LockerStation -> CodeGenerator.next_code()`, then `active.put_if_absent(code, assignment)`
    4. `LockerStation -> CustomerNotifier.send_code(...)`
  - **Collect**
    1. `Customer -> LockerStation.collect(code, now)`
    2. `LockerStation -> active.get(code)`, then `Assignment.collect(now)`
    3. `LockerStation -> Compartment.release(package_id)`, then `active.remove(code)`
  - **Expire**
    1. `Scheduler -> LockerStation.expire_stale(now)`
    2. For each live code
- **Failure and undo (per flow):**
  - **Collect:** a wrong, used or expired code opens nothing.
  - **Expire stale packages:** an assignment collected meanwhile makes `expire` False, so it is skipped. A package can't be both collected and removed.
  - **Out of service:** False, with no change. Deposits skip an out-of-service compartment because `try_occupy` refuses it.
- **Concurrency:**
  - **Compartment:** the lock sits inside `Compartment`, around `try_occupy`, `release` and the service changes. "Check free, then occupy" is one step, so two agents can never be given the same compartment. The loser just tries the next one.
  - **Assignment:** the lock sits inside `Assignment`, around `collect` and `expire`, so a collection and an expiry at the deadline can't both succeed.
  - **Codes:** `active` is a thread-safe map, and `put_if_absent` makes a new code unique in one step, even if two deposits draw the same code.
  - **Station:** the lock sits inside `LockerStation`, around the compartment list only, and is released before any compartment is touched.
  - **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
- **Key decisions:**
  1. **Each compartment guards itself with a lock.** *Rejected:* one lock on the whole station, which would make every agent wait for every other.
  2. **A compartment's status is computed, not stored.** *Rejected:* a stored `OCCUPIED` status beside the package.
  3. **`Assignment` decides collected or expired, once.** *Rejected:* the station checking the time and then acting, which two callers could both pass.
  4. **The code is made unique with `put_if_absent`, and retried.** *Rejected:* checking the map and then adding, which two deposits could both pass with the same code.
  5. **The customer is told last, and a failed message undoes the deposit.** *Rejected:* telling the customer first, which could send a code for a compartment that was never secured.
- **Edge cases:**
  - A small package goes into a small compartment even when bigger ones are free.
  - No compartment fits, so the deposit is refused with nothing changed.
  - Two agents racing for the last compartment: one gets it, the other tries the next.
  - A generated code that is already live is retried.
  - A message that can't be sent undoes the whole deposit.
  - A wrong, used or expired code opens nothing.
  - A collection at the very moment of expiry: exactly one of the two wins.
  - An occupied compartment can't be taken out of service.
- **Extensibility (change → what to add):**
  - More compartments or new stations → Calls to `add_compartment`, or more `LockerStation` objects *(edits existing code: No. It is data)*
  - A different code lifetime → A different `code_ttl_minutes` *(edits existing code: No. It is data)*
  - Sending codes by a new channel → A new class implementing `CustomerNotifier` *(edits existing code: No)*
  - A different code format (letters, longer) → A new class implementing `CodeGenerator`, or a different length *(edits existing code: No)*
- **Follow-up questions:**
  1. **"What stops someone guessing codes?"** Lock out a station, or a customer, after a few wrong codes in a row. It would be a counter checked at the start of `collect`. It is not required here.
  2. **"How does expiry run on its own?"** A scheduler calls `expire_stale(now)` every few minutes. `collect` also checks the deadline itself, so an expired code never works even before the sweep runs.
  3. **"Why not put a package in a bigger compartment when the right size is full?"** It already does. "Smallest free that fits" moves on to the next size when every small one is taken.
- **Out of scope (one-line answer if asked):**
  - Locking out after wrong codes → See follow-up question 1.
  - Several packages for one customer under one code → One code per package, as requirement 3 says.
  - Returning a package to the sender → `expire_stale` returns the removed packages. Sending them back is outside the locker.
  - Running the expiry sweep on a timer → Infrastructure outside the design. It only needs to call `expire_stale(now)`.
  - Choosing between several stations → One station. A choice of station would be a layer above.
  - A real SMS or email service → `CustomerNotifier.send_code` is the one place it goes.

#### 24 · Meeting Room Booking
- **Enums:**
  - **Facility**: `PROJECTOR`, `WHITEBOARD`, `VIDEO_CONFERENCE`, `TV_SCREEN`
  - **BookingStatus**: `CONFIRMED`, `CANCELLED`
  - **NoticeKind**: `INVITED`, `UPDATED`, `CANCELLED`
- **Class details (job · what it owns · key methods):**
  - **TimeSlot**: Describes a start and end and checks overlap · *owns:* start, end
    - `get_start() -> datetime`, `get_end() -> datetime`, `overlaps(other: TimeSlot) -> bool`
  - **RoomInfo**: Gives a read-only picture of one room for search results · *owns:* a copy of key facts
  - **Room**: Holds a room's details and its calendar, and guards the calendar · *owns:* capacity, facilities, reserved slots
    - `get_room_id() -> str`, `get_capacity() -> int`, `get_info() -> RoomInfo`, `suits(min_capacity: int, needed: set[Facility]) -> bool`, `is_free(slot: TimeSlot) -> bool`, `try_reserve(booking_id: str, slot: TimeSlot) -> bool`, `replace(booking_id: str, new_slot: TimeSlot) -> bool`, `release(booking_id: str) -> bool`, `get_booking_ids(window: TimeSlot) -> list[str]`
  - **Booking**: Holds one meeting's details, and runs its own changes and cancellation · *owns:* organizer, title, room, slot, attendees, status
    - `get_attendee_ids() -> set[str]`, `involves(person_id: str) -> bool`, `cancel(requester_id: str, room: Room) -> bool`, `change(requester_id: str, old_room: Room, new_room: Room, new_slot: TimeSlot, new_attendees: set[str]) -> set[str] or None`
  - **MeetingNotifier**: Sends a message about a meeting to a person
    - `notify(person_id: str, kind: NoticeKind, booking: Booking) -> None`
  - **MeetingService**: Searches, books, changes, cancels and shows schedules · *owns:* the rooms, the bookings
    - `add_room(room_id: str, name: str, capacity: int, facilities: set[Facility]) -> bool`, `search_rooms(slot: TimeSlot, min_capacity: int, needed: set[Facility]) -> list[RoomInfo]`, `book(organizer_id: str, room_id: str, slot: TimeSlot, title: str, attendee_ids: set[str], now: datetime) -> Booking or None`, `change_booking(booking_id: str, requester_id: str, new_room_id: str or None, new_slot: TimeSlot or None, new_attendees: set[str] or None, now: datetime) -> bool`, `cancel_booking(booking_id: str, requester_id: str) -> bool`, `get_room_schedule(room_id: str, window: TimeSlot) -> list[Booking] or None`, `get_person_schedule(person_id: str, window: TimeSlot) -> list[Booking]`, `fits(room: Room, organizer_id: str, attendee_ids: set[str]) -> bool`
- **Relationships:**
  - MeetingService **has** Room, Booking (created inside `add_room` and `book`).
  - Room **uses** TimeSlot. Booking **uses** TimeSlot, Room (rooms are passed into `cancel` and `change`, and live on).
  - MeetingService **uses** MeetingNotifier (passed in).
  - InboxNotifier **implements** MeetingNotifier.
  - Enums are always **uses**: Room to Facility, Booking to BookingStatus, MeetingNotifier to NoticeKind.
  - `RoomInfo` appears only as a return type, so it has no line.
- **Call flows:**
  - **Search**
    1. `User -> MeetingService.search_rooms(slot, capacity, facilities)`
    2. `MeetingService -> Room.suits(...)` and `Room.is_free(slot)`
  - **Book**
    1. `User -> MeetingService.book(...)`
    2. `MeetingService -> rooms.get(room_id)`, then `fits(...)`
    3. `MeetingService -> Room.try_reserve(booking_id, slot)`
    4. `MeetingService -> Booking(...)`, then `MeetingNotifier.notify(person, INVITED, booking)`
  - **Change**
    1. `User -> MeetingService.change_booking(...)`
    2. `MeetingService -> Booking.change(requester, old_room, new_room, new_slot, new_attendees)`
    3. `Booking -> Room.replace`
    4. `MeetingService -> MeetingNotifier.notify(...)`
  - **Cancel**
    1. `User -> MeetingService.cancel_booking(...)`, then `Booking.cancel(requester, room)`, then `Room.release`
- **Failure and undo (per flow):**
  - **Book:** steps 1 to 3 change nothing if they refuse. A failed notice is skipped and does not undo the booking.
  - **Change:** inside `change`, the new slot is reserved before the old one is released. If the new reservation fails, nothing was released and nothing changed. *Why this order:* the meeting always holds at least one slot, and never loses its room in the middle of a move.
  - **Cancel:** someone other than the organizer, or a booking already cancelled, is refused with no change.
  - **Schedules:** an unknown room returns None. A person with no meetings gets an empty list.
- **Concurrency:**
  - **Room:** the lock sits inside `Room`, around every method. `try_reserve` checks for overlap and stores the slot as one step, so two people booking the same slot at the same moment can't both succeed.
  - **Booking:** the lock sits inside `Booking`, around `change` and `cancel`, so a change and a cancel on the same meeting run one after the other.
  - **Maps and ids:** `rooms` and `bookings` are thread-safe maps, and ids come from an atomic counter.
  - **Lock order:** a booking lock is always taken before a room lock, and a booking never holds two room locks at once (it reserves the new room, lets go, then releases the old one). `book` takes only a room lock. So no deadlock is possible.
- **Key decisions:**
  1. **Each room guards its own calendar with a lock.** *Rejected:* one lock for the whole system, which would make every booking wait for every other.
  2. **The room stores only booking ids and slots, and the booking stores the meeting details.** *Rejected:* rooms holding full booking objects, which makes a change touch two places.
  3. **A change reserves the new slot before releasing the old one.** *Rejected:* releasing first, which could leave the meeting with no room if someone takes it in between.
  4. **A person's schedule is computed, not stored.** *Rejected:* a per-person index kept in step with every booking change. It is faster, but it is a second copy.
  5. **Search returns the smallest suitable room first.** *Rejected:* any order, which wastes big rooms on small meetings.
- **Edge cases:**
  - A meeting ending at 11:00 and one starting at 11:00 do not overlap.
  - Booking a slot in the past is refused.
  - A meeting with more people than the room holds is refused.
  - Changing to a slot that is taken leaves the booking as it was.
  - Only the organizer can change or cancel.
  - A cancelled booking can't be changed or cancelled again.
  - Removed attendees are told the meeting is cancelled for them, and new ones are invited.
  - Two people booking the same slot at the same moment: one wins.
- **Extensibility (change → what to add):**
  - More rooms → Calls to `add_room` *(edits existing code: No. It is data)*
  - A new facility → One new `Facility` value *(edits existing code: No logic changes)*
  - Sending invitations by email or calendar → A new class implementing `MeetingNotifier` *(edits existing code: No)*
  - Searching by floor or building → A new field on `Room` and `RoomInfo`, and one more check in `suits` *(edits existing code: Yes, a small addition to `suits`)*
- **Follow-up questions:**
  1. **"Isn't checking every reservation slow for a busy room?"** Keep each room's slots sorted by start time (or in an interval tree), so the overlap check looks only at neighbours.
  2. **"How would you handle recurring meetings?"** Expand a recurring meeting into one booking per occurrence, reserved together, and refuse the series if any occurrence clashes. It is a new rule on top of `try_reserve`.
  3. **"How would you show a person's schedule quickly?"** Keep an index from person to booking ids, updated inside the same steps that book, change and cancel. It is a deliberate second copy kept for speed.
- **Out of scope (one-line answer if asked):**
  - People accepting or declining invitations → Not required. It would be a response per attendee on the booking.
  - Recurring meetings → See follow-up question 2.
  - Checking that people are free → Only rooms are checked for overlap. Person clashes would use the person schedule before booking.
  - Room approval or restricted rooms → Not required.
  - Real email or calendar invites → `MeetingNotifier.notify` is the one place they go.
  - Fast schedule lookups → See follow-up questions 1 and 3.

#### 28 · Airline Management
- **Enums:**
  - **SeatClass**: `ECONOMY`, `BUSINESS`, `FIRST`
  - **SeatStatus**: `AVAILABLE`, `HELD`, `BOOKED`
  - **BookingStatus**: `PENDING`, `CONFIRMED`, `CANCELLED`
  - **FlightStatus**: `SCHEDULED`, `DELAYED`, `DEPARTED`, `LANDED`, `CANCELLED`
  - **CrewRole**: `PILOT`, `CO_PILOT`, `CABIN_CREW`
- **Class details (job · what it owns · key methods):**
  - **TimeWindow**: Describes a start and end and checks overlap · *owns:* start, end
    - `get_start()`, `get_end()`, `overlaps(other: TimeWindow) -> bool`
  - **Seat**: Describes one physical seat · *owns:* number, class
  - **FlightSeat**: Holds one seat's status on one flight and guards it · *owns:* status
    - `get_seat() -> Seat`, `get_status() -> SeatStatus`, `try_hold() -> bool`, `confirm() -> bool`, `release() -> bool`
  - **FlightObserver**: Reacts to a flight changing stage
    - `on_flight_status_changed(flight_number: str, status: FlightStatus, departure: datetime, arrival: datetime) -> None`, `send(email: str, message: str) -> None`
  - **PassengerSubscription**: Watches one flight for one passenger and tells them of changes · *owns:* who to tell
    - `get_subscription_id() -> str`, `on_flight_status_changed(...)`
  - **Flight**: Holds a flight's route, times, stage and seats, and tells its watchers · *owns:* times, stage, seats, watchers
    - `get_window() -> TimeWindow`, `get_flight_seat(seat_number: str) -> FlightSeat or None`, `get_available_seats() -> list[FlightSeat]`, `is_bookable(now: datetime) -> bool`, `subscribe(subscription: FlightObserver, subscription_id: str) -> bool`, `unsubscribe(subscription_id: str) -> bool`, `delay(new_departure: datetime, new_arrival: datetime) -> bool`, `depart() -> bool`, `land() -> bool`, `cancel() -> bool`
  - **PricingStrategy**: Prices a seat class on a flight
    - `price_for(seat_class: SeatClass, base_fare: int) -> int`
  - **PaymentMethod**: Takes and returns a payment
    - `pay(amount: int) -> bool`, `refund(amount: int) -> bool`
  - **Passenger**: Holds a traveller's details · *owns:* name, document, email
  - **BoardingPass**: Records a check-in · *owns:* an unchangeable record
  - **BookingItem**: Joins one passenger to one seat at one price, and their boarding pass · *owns:* price, boarding pass
    - `set_boarding_pass(boarding_pass: BoardingPass) -> bool`
  - **Booking**: Holds a booking's items and runs its stages, payment, refund and check-in · *owns:* status, payment, hold expiry
    - `get_total() -> int`, `confirm(payment: PaymentMethod, now: datetime, notifier: PassengerNotifier) -> bool`, `cancel(requester_id: str) -> bool`, `cancel_by_airline() -> bool`, `cancel_internal()`, `payment.refund(total)`, `expire(now: datetime) -> bool`, `check_in(document_number: str, now: datetime, opens_hours: int, closes_minutes: int, pass_id: str) -> BoardingPass or None`
  - **CrewMember**: Holds a crew member's flight windows and guards them · *owns:* assignments
    - `is_assigned(flight_number: str) -> bool`, `try_assign(flight_number: str, window: TimeWindow) -> bool`, `reschedule(flight_number: str, new_window: TimeWindow) -> bool`, `unassign(flight_number: str) -> bool`
  - **FlightCatalog**: Stores aircraft and flights, and searches them · *owns:* aircraft, flights
    - `add_aircraft(aircraft: Aircraft) -> bool`, `add_flight(flight: Flight) -> bool`, `get_aircraft(tail_number: str) -> Aircraft or None`, `get_flight(flight_number: str) -> Flight or None`, `search(source: str, destination: str, day: date) -> list[Flight]`
  - **CrewRoster**: Stores crew members · *owns:* crew
    - `add(member: CrewMember) -> bool`, `get(crew_id: str) -> CrewMember or None`, `all() -> list[CrewMember]`
  - **AirlineOperations**: Runs admin actions: flights, stages, crew · *owns:* none of its own
    - `add_aircraft(tail_number: str, model: str, seats: list[Seat]) -> bool`, `add_flight(flight_number: str, source: str, destination: str, departure: datetime, arrival: datetime, tail_number: str, base_fare: int) -> Flight or None`, `add_crew(crew_id: str, name: str, role: CrewRole) -> bool`, `assign_crew(flight_number: str, crew_id: str) -> bool`, `get_crew(flight_number: str) -> list[CrewMember]`, `delay_flight(flight_number: str, new_departure: datetime, new_arrival: datetime) -> list[str] or None`, `depart_flight(flight_number: str) -> bool`, `land_flight(flight_number: str) -> bool`, `cancel_flight(flight_number: str) -> bool`
  - **BookingService**: Runs user actions: hold, pay, cancel, check in · *owns:* bookings
    - `search(source: str, destination: str, day: date) -> list[Flight]`, `quote(flight_number: str, seat_class: SeatClass) -> int or None`, `hold_seats(user_id: str, flight_number: str, requests: list[(Passenger, str)], now: datetime) -> Booking or None`, `confirm_booking(booking_id: str, user_id: str, payment: PaymentMethod, now: datetime) -> bool`, `cancel_booking(booking_id: str, user_id: str) -> bool`, `cancel_flight_bookings(flight_number: str) -> int`, `expire_stale(now: datetime) -> int`, `check_in(booking_id: str, user_id: str, document_number: str, now: datetime) -> BoardingPass or None`, `get_booking(booking_id: str, user_id: str) -> Booking or None`
- **Relationships:**
  - Aircraft **has** Seat. Flight **has** FlightSeat. Booking **has** BookingItem.
  - FlightCatalog **has** Aircraft, Flight. CrewRoster **has** CrewMember. BookingService **has** Booking.
  - Flight **uses** Aircraft, FlightObserver. FlightSeat **uses** Seat.
  - BookingItem **uses** Passenger, FlightSeat, BoardingPass. Booking **uses** Flight, PaymentMethod.
  - CrewMember **uses** TimeWindow. PassengerSubscription **uses** PassengerNotifier.
  - AirlineOperations **uses** FlightCatalog, CrewRoster, BookingService. BookingService **uses** FlightCatalog, PricingStrategy, PassengerNotifier.
  - PassengerSubscription **implements** FlightObserver. InboxNotifier **implements** PassengerNotifier. ClassMultiplierPricing **implements** PricingStrategy. CardPayment, UpiPayment **implement** PaymentMethod.
  - Enums are always **uses**: Seat to SeatClass, FlightSeat to SeatStatus, Booking to BookingStatus, Flight to FlightStatus, CrewMember to CrewRole, BoardingPass to SeatClass.
- **Call flows:**
  - **Book**
    1. `User -> BookingService.hold_seats(...)`, then `FlightCatalog.get_flight`, then `Flight.get_flight_seat`
    2. `BookingService -> FlightSeat.try_hold()`
    3. `User -> BookingService.confirm_booking(...)`, then `Booking.confirm(payment, now, notifier)`
    4. `Booking -> PaymentMethod.pay`
  - **Flight change**
    1. `Admin -> AirlineOperations.delay_flight(...)`, then `Flight.delay(...)`
    2. `Flight -> FlightObserver.on_flight_status_changed(...)`
    3. `AirlineOperations -> CrewMember.reschedule(...)`
  - **Cancel a flight**
    1. `Admin -> AirlineOperations.cancel_flight(...)`, then `Flight.cancel()`
    2. `AirlineOperations -> BookingService.cancel_flight_bookings(flight_number)`, then `Booking.cancel_by_airline()`
    3. `AirlineOperations -> CrewMember.unassign(flight_number)`
  - **Check in**
    1. `Passenger -> BookingService.check_in(...)`, then `Booking.check_in(...)`
- **Failure and undo (per flow):**
  - **Cancel a booking (requirement 8):** another user, a booking already cancelled, or a flight already departed is refused with no change.
  - **Flight stages (requirement 9):** an illegal stage change (landing a flight that never departed, delaying a landed flight) is refused with no change and no message.
  - **Assign crew (requirement 10):** an overlap returns False and nothing changes.
  - **Check in (requirement 11):** outside the window, an unknown passenger, or a booking not confirmed returns None. Checking in twice returns the same pass.
- **Concurrency:**
  - **Seat:** the lock sits inside `FlightSeat`, around `try_hold`, `confirm` and `release`. Two passengers can never hold the same seat on the same flight.
  - **Crew:** the lock sits inside `CrewMember`, around every change to `assignments`. Two simultaneous assignments can't overlap.
  - **Booking:** the lock sits inside `Booking`, around confirm, cancel, expire and check-in.
  - **Flight:** the lock sits inside `Flight`, around its times, stage and watchers, and is released before watchers are called.
  - **Payment:** each payment class has its own lock.
  - **Maps and ids:** thread-safe maps and an atomic counter.
  - **Lock order:** a booking's lock is taken before a seat, payment or flight lock, never the reverse. `AirlineOperations` holds no lock while calling `BookingService`. Each method holds at most two locks at once, so no deadlock is possible.
- **Key decisions:**
  1. **Seat status lives in `FlightSeat`, not in `Seat`.** *Rejected:* a status on Seat that is reset after each flight.
  2. **A crew member owns their calendar and guards it.** *Rejected:* a crew list on each flight, which would need its own overlap checks and could disagree with the crew member.
  3. **Passengers watch a flight through small subscription objects.** *Rejected:* the flight holding bookings, or the booking service watching every flight, which would create cycles.
  4. **The price is fixed when the seat is held.** *Rejected:* computing the price again at payment.
  5. **Admin and user actions are in two managers.** *Rejected:* one `AirlineSystem` class with everything, which grows into a god class.
- **Edge cases:**
  - One of several chosen seats is taken: the others are released and nothing is booked.
  - A hold that passes its deadline is cancelled and the seats freed, and a late payment is refused before money is taken.
  - A failed payment frees the seats.
  - A cancel refunds before it frees the seats.
  - A cancelled flight refunds every booking and frees its crew.
  - A delay that makes a crew member clash removes them and reports them.
  - A flight that has departed can't be booked, cancelled by a user, or checked into.
  - Checking in twice returns the same boarding pass.
  - Illegal flight stage changes are refused without notifying anyone.
- **Extensibility (change → what to add):**
  - A new pricing rule (demand-based, weekend) → A new class implementing `PricingStrategy`, passed to `BookingService` *(edits existing code: No)*
  - A new payment method → A new class implementing `PaymentMethod` *(edits existing code: No)*
  - A new way to notify passengers (SMS, app push) → A new class implementing `PassengerNotifier` *(edits existing code: No)*
  - A new seat class (premium economy) → One `SeatClass` value and its percentage in the pricing table *(edits existing code: No logic changes)*
  - Different hold, check-in or rest times → Different numbers in the managers' constructors *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"What if the notifier is slow?"** Watchers are called after the flight's lock is released, so the flight isn't blocked. For many passengers, the notifier would put messages on a queue, as in a notification service.
  2. **"How would you stop an aircraft being on two flights at once?"** Give `Aircraft` a calendar like `CrewMember`, and check it in `add_flight` and `delay_flight`. No requirement asks for it here.
  3. **"How would you handle overbooking?"** Allow more holds than seats for economy and resolve at check-in. It is a deliberate change to requirement 6, so it would be a separate, explicit rule.
- **Out of scope (one-line answer if asked):**
  - An aircraft on two flights at once → See follow-up question 2.
  - Changing a booking to another flight → Not required. It is a cancel and a new booking.
  - Partial refunds or cancellation fees → A cancel always refunds in full. A fee would be a rule in `cancel_internal`.
  - Baggage → Not in these requirements.
  - User accounts and admin login → Users and admins are only callers. Permissions would be checked in the managers.
  - Gates and boarding groups → Not required. They would be fields on `BoardingPass`.
  - Running the hold expiry on a timer → Infrastructure outside the design. It only needs to call `expire_stale(now)`.


---

## Type B: State Machines

#### 04 · Elevator
- **Enums:**
  - **Direction**: `UP`, `DOWN`
  - **DoorStatus**: `OPEN`, `CLOSED`
  - **ElevatorStateType**: `IDLE`, `MOVING_UP`, `MOVING_DOWN`, `OUT_OF_SERVICE`
- **Class details (job · what it owns · key methods):**
  - **ElevatorInfo**: Gives a read-only picture of one elevator · *owns:* a copy of its key facts
    - `get_elevator_id() -> str`, `get_current_floor() -> int`, `get_state_type() -> ElevatorStateType`, `get_stop_count() -> int`, `get_min_floor() -> int`, `get_max_floor() -> int`
  - **Car**: Owns one elevator's floor, door, load and pending stops, and guards moving it · *owns:* floor, door, load, stops
    - `get_car_id() -> str`, `get_current_floor() -> int`, `get_door() -> DoorStatus`, `get_load() -> int`, `can_serve(floor: int) -> bool`, `is_overloaded() -> bool`, `report_load(weight: int) -> bool`, `add_stop(floor: int) -> bool`, `has_stops() -> bool`, `has_stops_above() -> bool`, `has_stops_below() -> bool`, `get_stop_count() -> int`, `serve_stop() -> bool`, `handle_door() -> bool`, `move(direction: Direction) -> bool`
  - **ElevatorState**: Lists the actions a state must answer
    - `add_stop(car: Car, floor: int) -> ElevatorStateType or None`, `step(car: Car) -> ElevatorStateType`, `take_out_of_service(car: Car) -> ElevatorStateType or None`, `return_to_service(car: Car) -> ElevatorStateType or None`
  - **IdleState**: Behavior when waiting with no stops
    - `add_stop`: if `car.add_stop(floor)` is False, return None. Otherwise return `MOVING_UP` if the floor is above the car, `MOVING_DOWN` if below, and `IDLE` if it is the current floor (the next tick opens the door)
    - `step`: if `car.handle_door()` is True, return `IDLE`. If `car.serve_stop()` is True, return `IDLE`. If there are stops above, return `MOVING_UP`. If below, `MOVING_DOWN`. Otherwise `IDLE`
    - `take_out_of_service`: if the door is CLOSED and there are no stops, return `OUT_OF_SERVICE`. Otherwise None
    - `return_to_service`: None
  - **MovingUpState**: Behavior when heading up
    - `add_stop`: if `car.add_stop(floor)` is False, return None. Otherwise return `MOVING_UP`
    - `step`
    - `take_out_of_service`, `return_to_service`: None
  - **MovingDownState**: Behavior when heading down
    - the same as `MovingUpState`, with above and below swapped and `move(DOWN)` used
  - **OutOfServiceState**: Refuses calls and does not move
    - `add_stop`: None. `step`: return `OUT_OF_SERVICE` and do nothing. `take_out_of_service`: None
    - `return_to_service`: return `IDLE`
  - **Elevator**: Runs one elevator: asks the current state, applies the result, and holds the lock · *owns:* which state it is in
    - `get_elevator_id() -> str`, `add_stop(floor: int) -> bool`, `step() -> None`, `take_out_of_service() -> bool`, `return_to_service() -> bool`, `report_load(weight: int) -> bool`, `snapshot() -> ElevatorInfo`
  - **SchedulingStrategy**: Chooses which elevator takes a call
    - `select(candidates: list[ElevatorInfo], floor: int, direction: Direction) -> ElevatorInfo or None`
  - **NearestElevatorStrategy**: Picks the best elevator by position and direction
    - `select(candidates: list[ElevatorInfo], floor: int, direction: Direction) -> ElevatorInfo or None`
  - **ElevatorController**: Takes calls, picks an elevator, and advances all elevators · *owns:* the elevators
    - `add_elevator(elevator_id: str, capacity: int, min_floor: int, max_floor: int, start_floor: int) -> bool`, `request_elevator(floor: int, direction: Direction) -> str or None`, `select_destination(elevator_id: str, floor: int) -> bool`, `report_load(elevator_id: str, weight: int) -> bool`, `take_out_of_service(elevator_id: str) -> bool`, `return_to_service(elevator_id: str) -> bool`, `step_all() -> None`, `get_status() -> list[ElevatorInfo]`
- **Relationships:**
  - ElevatorController **has** Elevator (created inside `add_elevator`).
  - Elevator **has** Car (the car dies with its elevator).
  - ElevatorController **uses** SchedulingStrategy, ElevatorState (both passed in).
  - Elevator **uses** ElevatorState (the shared state objects, passed in).
  - IdleState, MovingUpState, MovingDownState, OutOfServiceState **implement** ElevatorState.
  - NearestElevatorStrategy **implements** SchedulingStrategy.
  - Enums are always **uses**: Car to DoorStatus, Elevator to ElevatorStateType, ElevatorController to ElevatorStateType, ElevatorInfo to ElevatorStateType.
  - `Direction` and `ElevatorInfo` appear only as method inputs or outputs, so no attribute holds them and they have no line.
- **Call flows:**
  - **Call an elevator**
    1. `User -> ElevatorController.request_elevator(floor, direction)`
    2. `ElevatorController -> Elevator.snapshot()`
    3. `ElevatorController -> SchedulingStrategy.select(candidates, floor, direction)`
    4. `ElevatorController -> Elevator.add_stop(floor)`
    5. If step 4 is refused, drop that candidate and go back to step 3
  - **Choose a destination**
    1. `User -> ElevatorController.select_destination(elevator_id, floor)`
    2. `ElevatorController -> Elevator.add_stop(floor)`
    3. `Elevator -> current state.add_stop(car, floor)`, then `Car.add_stop(floor)`
  - **Tick**
    1. `Timer -> ElevatorController.step_all()`
    2. `ElevatorController -> Elevator.step()`
    3. `Elevator -> current state.step(car)`
    4. `State -> Car.handle_door()`
  - **Weight sensor**
    1. `Sensor -> ElevatorController.report_load(elevator_id, weight)`
    2. `ElevatorController -> Elevator.report_load(weight)`, then `Car.report_load(weight)`
  - **Take out of service**
    1. `Operator -> ElevatorController.take_out_of_service(elevator_id)`
    2. `ElevatorController -> Elevator.take_out_of_service()`, then `current state.take_out_of_service(car)`
- **Failure and undo (per flow):**
  - **Call an elevator:** a refused `add_stop` changes nothing, so there is nothing to undo. When no candidate is left, the result is None.
  - **Choose a destination:** returns False and nothing changes.
  - **Tick (one elevator, one step):** none of these steps can leave a half-done state. Each is one change, made by one method.
  - **Weight limit:** a refused report leaves the old load.
  - **Out of service:** any other state refuses and stays as it was. A moving car must finish its stops first.
- **Concurrency:**
  - **Elevator:** the lock sits inside `Elevator`, around `add_stop`, `step`, `take_out_of_service`, `return_to_service`, `report_load` and `snapshot`. Every change to one elevator runs one at a time, so two requests can't corrupt its stop set or state.
  - **Car and states:** they have no lock. Only the Elevator, holding its lock, reaches them.
  - **Controller:** the lock sits inside `ElevatorController`, around the `elevators` map only. It is never held while calling an elevator.
  - **Assignment race:** the snapshot may be stale by the time `add_stop` runs. A refused `add_stop` makes the controller drop that candidate and choose again, so no request is lost and no elevator is assigned something it refuses.
  - **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
  - **Why not one big lock:** it would make every elevator wait for every other, which is slower and unnecessary because requests for different elevators don't conflict.
- **Key decisions:**
  1. **`Car` owns the physical facts and rules, and the states hold no data.** *Rejected:* states that hold the elevator. They would point at it while it points at them, in a circle.
  2. **One lock per elevator, and a retry if the chosen elevator refuses.** *Rejected:* one lock for the whole controller, which would serialize everything.
  3. **One tick does one thing.** *Rejected:* a single `step` that moves, opens and closes at once. The door rule would not be observable.
  4. **The scheduler sees `ElevatorInfo` copies, not live elevators.** *Rejected:* passing elevators to the rule. A rule could modify one, and reading several facts could see a half-changed elevator.
  5. **"Overloaded" is computed from the load.** *Rejected:* an `overloaded` flag next to the load. They would disagree after one missed update.
- **Edge cases:**
  - A call for a floor no elevator serves, or when every elevator is out of service, returns None.
  - An elevator going out of service between the snapshot and the assignment: the controller picks another.
  - The same floor requested twice becomes one stop.
  - A stop at the floor the car is already on opens the door on the next tick.
  - Stops behind a moving elevator wait until it reverses.
  - Overload with the door closed: the door opens and the car holds. Overload with the door open: it stays open.
  - A destination outside an elevator's range is refused.
  - Taking a moving elevator out of service is refused.
- **Extensibility (change → what to add):**
  - A new scheduling rule (least busy, zoned elevators, express) → A new class implementing `SchedulingStrategy`, passed to the controller *(edits existing code: No)*
  - More elevators, or elevators serving different floors → Calls to `add_elevator` with different numbers *(edits existing code: No. It is data)*
  - A different weight limit per elevator → A different `capacity` in `add_elevator` *(edits existing code: No. It is data)*
  - A new state, such as MAINTENANCE or FIRE_MODE → A new state class, one new enum value, one new map entry *(edits existing code: The one state that moves into it must return the new type, so that return value changes. This is the cost of keeping transitions inside states)*
- **Follow-up questions:**
  1. **"How would you pick elevators more cleverly?"** Write another `SchedulingStrategy`, such as a zoned or least-busy rule. Nothing else changes.
  2. **"Why is each elevator driven by ticks?"** It makes the behavior deterministic and testable. A timer or one thread per elevator can call `step`, and the elevator's lock keeps it safe either way.
  3. **"What happens when two calls arrive at the same instant?"** Both take snapshots and may pick the same elevator. Each `add_stop` runs under that elevator's lock, so both stops are added. If the elevator refuses one, the controller chooses again.
- **Out of scope (one-line answer if asked):**
  - Fire mode or emergency stop → A new state class. See the Extensibility list above.
  - Door obstruction sensors → The door closes when not overloaded. A sensor would add a reason `handle_door` keeps the door open.
  - Serving stops first-come-first-served → Stops are served by direction order. It is the standard behavior and meets requirement 3.
  - The direction of a hall call after assignment → It only helps choose an elevator. Once assigned, the floor is just a stop.
  - Cancelling a request → No requirement asks for it. It would be a `remove_stop` on `Car`.
  - Travel time and speed → One floor per tick. The caller sets how often ticks happen.
  - Button lights and floor displays → No requirement asks for them. `get_status` provides what a display needs.
  - Weight measured by the design → The sensor reports the weight through `report_load`.

#### 05 · Vending Machine
- **Enums:**
  - **MachineStateType**: `IDLE`, `HAS_MONEY`, `DISPENSING`, `OUT_OF_SERVICE`
- **Class details (job · what it owns · key methods):**
  - **Product**: Describes one sellable item · *owns:* name, price
    - `get_name() -> str`, `get_price() -> int`
  - **Slot**: Holds one product and counts what is left · *owns:* quantity
    - `get_code() -> str`, `get_product() -> Product`, `get_quantity() -> int`, `is_in_stock() -> bool`, `take_one() -> bool`, `restock(count: int) -> bool`
  - **Inventory**: Sets up slots and finds a slot by its code · *owns:* the map of slots
    - `add_slot(code: str, product: Product, quantity: int) -> bool`, `get_slot(code: str) -> Slot or None`, `restock(code: str, count: int) -> bool`
  - **CashBox**: Keeps money paid by customers until the operator collects it · *owns:* collected total
    - `add(amount: int) -> bool`, `collect() -> int`
  - **Session**: Tracks one customer's inserted money and chosen slot · *owns:* balance, selected slot
    - `add_money(amount: int) -> bool`, `select_slot(slot: Slot) -> bool`, `get_selected_slot() -> Slot or None`, `can_afford(price: int) -> bool`, `take_payment(price: int) -> int or None`, `refund_all() -> int`
  - **Response**: Reports the result of a user action · *owns:* an unchangeable record
    - `is_accepted() -> bool`, `get_state_after() -> MachineStateType`, `get_product_name() -> str or None`, `get_money_returned() -> int`
  - **MachineState**: Lists every action a state must answer
    - `insert_money(amount: int) -> Response`, `select_product(slot: Slot or None) -> Response`, `cancel() -> Response`, `dispense() -> Response`, `take_out_of_service() -> Response`, `return_to_service() -> Response`, `allows_operator_actions() -> bool`
  - **IdleState**: Behavior while waiting for money
    - `insert_money`: `session.add_money(amount)`. If True, accept and go to `HAS_MONEY`. If False (bad amount), refuse and stay `IDLE`
    - `select_product`: refuse, stay `IDLE`
    - `cancel`: refuse, stay `IDLE`
    - `dispense`: refuse, stay `IDLE`
    - `take_out_of_service`: accept, go to `OUT_OF_SERVICE`
    - `return_to_service`: refuse, stay `IDLE`
    - `allows_operator_actions`: True
  - **HasMoneyState**: Behavior while the customer has money
    - `insert_money`: `session.add_money(amount)`. If False, refuse and stay. Otherwise, if a slot is selected and `session.can_afford(price)`, accept and go to `DISPENSING`. Else accept and stay `HAS_MONEY`
    - `select_product(slot)`: refuse if `slot` is None (unknown code), or `slot.is_in_stock()` is False (requirement 6), or `session.select_slot(slot)` returns False. Otherwise accept. If `session.can_afford(price)`, go to `DISPENSING`, else stay `HAS_MONEY` and wait for more money
    - `cancel`: `session.refund_all()`, accept, go to `IDLE`, with `money_returned` set to the refund
    - `dispense`: refuse, stay `HAS_MONEY`
    - `take_out_of_service`, `return_to_service`: refuse, stay
    - `allows_operator_actions`: False
  - **DispensingState**: Runs the sale
    - `insert_money`, `select_product`, `cancel`, `take_out_of_service`, `return_to_service`: refuse, stay `DISPENSING`
    - `allows_operator_actions`: False
    - `dispense`: see the sale steps below
    - *Sale steps, in this order (the order is the whole point):*
      1. `slot = session.get_selected_slot()`. If None, or `session.can_afford(price)` is False, refuse and go to `HAS_MONEY`. Nothing has changed yet
      2. `slot.take_one()`. If False, refuse and go to `HAS_MONEY`. Nothing has changed
      3. `change = session.take_payment(price)`. If None (it should not happen, since step 1 checked), undo with `slot.restock(1)`, refuse and go to `HAS_MONEY`
      4. `cash_box.add(price)`
      5. Return `Response(True, IDLE, product name, change)`
  - **OutOfServiceState**: Refuses customers
    - `insert_money`, `select_product`, `cancel`, `dispense`, `take_out_of_service`: refuse, stay `OUT_OF_SERVICE`
    - `return_to_service`: accept, go to `IDLE`
    - `allows_operator_actions`: True
  - **VendingMachine**: Hands each action to the current state, applies the result, guards operator actions · *owns:* which state it is in
    - `insert_money(amount: int) -> Response`, `select_product(slot_code: str) -> Response`, `cancel() -> Response`, `take_out_of_service() -> Response`, `return_to_service() -> Response`, `add_slot(slot_code: str, product: Product, quantity: int) -> bool`, `restock(slot_code: str, count: int) -> bool`, `collect_money() -> int or None`, `get_state() -> MachineStateType`, `finish(response: Response) -> Response`, `response.get_state_after()`, `dispense()`
- **Relationships:**
  - Inventory **has** Slot (slots are created inside `add_slot`).
  - Slot **uses** Product (the product exists on its own and can be shared).
  - Session **uses** Slot (the slot lives on in the inventory).
  - VendingMachine **uses** Inventory, CashBox, MachineState (all created in setup and passed in).
  - IdleState **uses** Session.
  - HasMoneyState **uses** Session.
  - DispensingState **uses** Session, CashBox.
  - IdleState, HasMoneyState, DispensingState, OutOfServiceState **implement** MachineState.
  - Enums are always **uses**: Response to MachineStateType, VendingMachine to MachineStateType.
  - `Response` is only a return type, so it has no line from the states or the machine, because no attribute holds it.
- **Call flows:**
  - **Insert money**
    1. `User -> VendingMachine.insert_money(amount)`
    2. `VendingMachine -> current state.insert_money(amount)`
    3. `State -> Session.add_money(amount)`
    4. `VendingMachine.finish(response)`
  - **Select a product**
    1. `User -> VendingMachine.select_product(slot_code)`
    2. `VendingMachine -> Inventory.get_slot(slot_code)`
    3. `VendingMachine -> current state.select_product(slot)`
    4. `HasMoneyState -> Slot.is_in_stock()`
    5. `HasMoneyState -> Session.select_slot(slot)`, then `Session.can_afford(price)`
    6. `VendingMachine.finish(response)`
  - **Automatic dispense (after `finish` sees `DISPENSING`)**
    1. `VendingMachine -> DispensingState.dispense()`
    2. `DispensingState -> Session.can_afford(price)`
    3. `DispensingState -> Session.take_payment(price)`
    4. `DispensingState -> CashBox.add(price)`
  - **Cancel**
    1. `User -> VendingMachine.cancel()`
    2. `VendingMachine -> current state.cancel()`
    3. `HasMoneyState -> Session.refund_all()`
  - **Restock**
    1. `Operator -> VendingMachine.restock(slot_code, count)`
    2. `VendingMachine -> current state.allows_operator_actions()`
    3. `VendingMachine -> Inventory.restock(slot_code, count)`, then `Slot.restock(count)`
  - **Collect money**
    1. `Operator -> VendingMachine.collect_money()`
    2. `VendingMachine -> current state.allows_operator_actions()`
    3. `VendingMachine -> CashBox.collect()`
- **Failure and undo (per flow):**
  - **Insert money:** `amount <= 0`, or the state is not Idle or HasMoney. The response is refused, `state_after` is the same state, and nothing changes. There is nothing to undo.
  - **Select a product:** unknown code, out of stock, or wrong state. The response is refused and the money stays in the session, so the customer can choose again or cancel.
  - **Dispense (automatic):** 
  - **Cancel:** in Idle, Dispensing or OutOfService it is refused, and nothing changes. There is nothing to refund, or a sale is running.
  - **Restock:** wrong state, unknown code, or `count <= 0`. It returns False and the quantity is unchanged.
  - **Collect money:** wrong state returns None and the total stays.
  - **Take out of service and return to service:** any other state refuses and stays as it was. A customer's money is never stranded, because HasMoney refuses until the customer cancels.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added.
  - The `DISPENSING` state is not a lock. It is a state that refuses every action while the sale runs.
  - If concurrency were required later, one lock inside `VendingMachine` around every public method would be enough, because all state is reachable only through it.
- **Key decisions:**
  1. **"In stock" is computed, not stored.** *Rejected:* a `SlotStatus` enum next to quantity. The two would disagree the moment someone forgot to update one.
  2. **State pattern, with states returning a Response.** *Rejected:* an `if` on the state in every method, and states calling a public `set_state` on the machine (anyone could call it).
  3. **A separate Session for the customer's money and selection.** *Rejected:* balance and selection as machine fields. The machine would hold changing data and mutate it in many methods.
  4. **Money first, then selection.** *Rejected:* selecting in Idle. It would need another place to keep a pre-money selection.
  5. **One `take_payment` method that checks, deducts and clears.** *Rejected:* separate deduct and clear calls.
- **Edge cases:**
  - Exact payment gives change 0. Overpayment gives change.
  - Selecting a second product in HasMoney replaces the first.
  - Not enough money waits, and later money triggers the sale.
  - Out-of-stock selection is refused and the money stays.
  - Zero or negative money is refused.
  - Restock and collect are refused during a customer's session.
  - OutOfService refuses money.
- **Extensibility (change → what to add):**
  - More products and slots → Call `add_slot` with a new Product *(edits existing code: No. It is data)*
  - Pay by card → A new `CardReader` class that charges the card and calls `insert_money(amount)` *(edits existing code: No)*
  - Give change in coins → A new `ChangeMaker` class that turns `money_returned` into coin counts *(edits existing code: No, as long as coins never run out)*
  - A new state, such as MAINTENANCE → A new state class, one new enum value, one new map entry *(edits existing code: The one state that moves into it must return the new type, so that return value changes. This is the cost of keeping transitions inside states)*
- **Follow-up questions:**
  1. **"Why is Dispensing a state if the sale is instant?"** It models the physical dispensing time and refuses every action while the sale runs, so nothing can interrupt it.
  2. **"How do you give change in coins?"** A `ChangeMaker` class converts the change amount into coins. If coins can run out, the check must go before step 2 of the sale, which means a small edit in `DispensingState`.
  3. **"What if two people use it at once?"** One lock inside `VendingMachine` around every public method, since every state is reachable only through it.
- **Out of scope (one-line answer if asked):**
  - Coin and note denominations → Money is one whole-number amount, and change is assumed payable. `ChangeMaker` would add coins.
  - Operator roles and login → No requirement defines them. The four operator methods are only allowed while no customer is mid-purchase.
  - Showing the product list or balance → No requirement asks for a display, so it is a read-only view that can be added.
  - Timeouts for money left in the machine → Money stays until the customer cancels. A timeout would be a new rule in HasMoneyState.
  - Hardware faults such as a jammed item → The sale steps are all-or-nothing in software. A jam would need a hardware step added before step 4.
  - Going out of service when empty → The operator takes it out of service. An automatic rule would be a new transition.
  - Selecting before inserting money → It is refused in Idle by design.
  - Concurrent customers → The Concurrency list above gives the one-lock answer.

#### 12 · ATM
- **Enums:**
  - **OperationType**: `CHECK_BALANCE`, `WITHDRAW`, `DEPOSIT`
  - **ATMStateType**: `IDLE`, `HAS_CARD`, `AUTHENTICATED`, `DISPENSING`
  - **AuthResult**: `SUCCESS`, `WRONG_PIN`, `BLOCKED`
- **Class details (job · what it owns · key methods):**
  - **Card**: Checks a PIN and counts wrong attempts · *owns:* PIN hash, wrong-attempt count
    - `get_card_number() -> str`, `is_blocked() -> bool`, `try_pin(pin: str) -> AuthResult`
  - **Account**: Holds a balance and guards changes to it · *owns:* balance
    - `get_balance() -> int`, `deposit(amount: int) -> bool`, `withdraw(amount: int) -> bool`
  - **BankService**: Creates accounts and cards, links them, and authenticates · *owns:* the accounts, cards and links
    - `create_account(account_number: str, initial_balance: int) -> Account or None`, `create_card(card_number: str, pin: str, max_attempts: int, account_number: str) -> Card or None`, `is_card_usable(card_number: str) -> bool`, `authenticate(card_number: str, pin: str) -> AuthResult`, `get_balance(card_number: str) -> int or None`, `deposit(card_number: str, amount: int) -> bool`, `withdraw(card_number: str, amount: int) -> bool`
  - **NoteDispenser**: Is one link in the chain, for one note value · *owns:* its note count
    - `get_value() -> int`, `get_count() -> int`, `plan(amount: int) -> map{int: int} or None`, `remove(n: int) -> bool`, `add(n: int) -> bool`
  - **CashDispenser**: Builds the chain and plans and takes cash · *owns:* the chain, the dispensers
    - `has_denomination(value: int) -> bool`, `add_notes(value: int, count: int) -> bool`, `plan(amount: int) -> map{int: int} or None`, `take(plan: map{int: int}) -> bool`, `restore(plan: map{int: int}) -> bool`
  - **Transaction**: Records one operation and prints a receipt · *owns:* an unchangeable record
    - `get_transaction_id() -> int`, `get_operation() -> OperationType`, `get_amount() -> int`, `get_balance_after() -> int or None`, `is_success() -> bool`, `to_receipt() -> str`
  - **TransactionLog**: Keeps every transaction · *owns:* the history
    - `record(card_number: str, operation: OperationType, amount: int, balance_after: int or None, success: bool, timestamp: datetime) -> Transaction`, `get_all() -> list[Transaction]`
  - **Session**: Holds the card in the machine and the requested withdrawal · *owns:* card number, requested amount
    - `get_card_number() -> str or None`, `get_requested_amount() -> int`, `start(card_number: str) -> bool`, `request_withdrawal(amount: int) -> bool`, `clear_request() -> None`, `clear() -> None`
  - **Response**: Reports the result of a user action · *owns:* an unchangeable record
    - `is_accepted() -> bool`, `get_state_after() -> ATMStateType`, `get_transaction() -> Transaction or None`
  - **ATMState**: Lists every action a state must answer
    - `insert_card(card_number: str)`, `enter_pin(pin: str)`, `select_operation(operation: OperationType, amount: int, notes: map{int: int}, now: datetime)`, `dispense(now: datetime)`, `cancel()`
  - **IdleState**: Behavior with no card
    - `insert_card`: if `bank_service.is_card_usable(card_number)` is False (unknown or blocked), refuse and stay `IDLE`. Otherwise `session.start(card_number)`, accept, go to `HAS_CARD`
    - `enter_pin`, `select_operation`, `dispense`, `cancel`: refuse and stay `IDLE`
  - **HasCardState**: Behavior with a card, before the PIN
    - `enter_pin(pin)`: `result = bank_service.authenticate(session card number, pin)`
    - `cancel`: `session.clear()`, accept, go to `IDLE`. This is requirement 7
    - `insert_card`, `select_operation`, `dispense`: refuse and stay `HAS_CARD`
  - **AuthenticatedState**: Behavior after a correct PIN
    - `select_operation`
    - `cancel`: `session.clear()`, accept, go to `IDLE`
    - `insert_card`, `enter_pin`, `dispense`: refuse and stay `AUTHENTICATED`
  - **DispensingState**: Runs a withdrawal
    - `session.clear_request()`, `Response(False, AUTHENTICATED, transaction)`
    - `insert_card`, `enter_pin`, `select_operation`, `cancel`: refuse and stay `DISPENSING`
    - `dispense(now)`: see the withdrawal steps below
    - *Withdrawal steps, in this order (checks first, changes last):*
      1. `amount = session.get_requested_amount()`, `card = session.get_card_number()`
      2. `plan = cash_dispenser.plan(amount)`. If None (the machine can't pay it), go to failure
      3. `bank_service.withdraw(card, amount)`. If False (not enough balance), go to failure
      4. `cash_dispenser.take(plan)`. If False (it should not happen), undo with `bank_service.deposit(card, amount)` and go to failure
      5. Record a success with the balance after, `session.clear_request()`, and return `Response(True, AUTHENTICATED, transaction)`
  - **ATM**: Hands each action to the current state and applies the result · *owns:* which state it is in
    - `insert_card(card_number: str) -> Response`, `enter_pin(pin: str) -> Response`, `cancel() -> Response`, `select_operation(operation: OperationType, amount: int, notes: map{int: int}, now: datetime) -> Response`, `get_state() -> ATMStateType`, `finish(response: Response, now: datetime) -> Response`, `dispense(now)`
- **Relationships:**
  - BankService **has** Account, Card (created inside `create_account` and `create_card`).
  - CashDispenser **has** NoteDispenser (the chain is built inside the constructor).
  - NoteDispenser **uses** NoteDispenser (the next link).
  - TransactionLog **has** Transaction (created inside `record`).
  - ATM **uses** ATMState (the states passed in).
  - IdleState **uses** Session, BankService.
  - HasCardState **uses** Session, BankService.
  - AuthenticatedState **uses** Session, BankService, CashDispenser, TransactionLog.
  - DispensingState **uses** Session, BankService, CashDispenser, TransactionLog.
  - IdleState, HasCardState, AuthenticatedState, DispensingState **implement** ATMState.
  - Enums are always **uses**: Transaction to OperationType, Response to ATMStateType, ATM to ATMStateType.
  - `Response` and `AuthResult` appear only as return types, so no attribute holds them and they have no line.
- **Call flows:**
  - **Insert a card and enter the PIN**
    1. `User -> ATM.insert_card(card_number)`, then `ATM -> current state.insert_card(...)`
    2. `IdleState -> BankService.is_card_usable(card_number)`
    3. `User -> ATM.enter_pin(pin)`, then `HasCardState -> BankService.authenticate(card, pin)`
    4. `BankService -> Card.try_pin(pin)`
  - **Check the balance or deposit**
    1. `User -> ATM.select_operation(...)`, then `AuthenticatedState -> BankService.get_balance`
    2. `AuthenticatedState -> CashDispenser.add_notes`
  - **Withdraw**
    1. `User -> ATM.select_operation(WITHDRAW, amount, ...)`, then `AuthenticatedState -> Session.request_withdrawal(amount)`
    2. `ATM.finish -> DispensingState.dispense(now)`
    3. `DispensingState -> CashDispenser.plan(amount)`
    4. `DispensingState -> BankService.withdraw(card, amount)`, then `CashDispenser.take(plan)`, then `TransactionLog.record`
  - **Cancel**
    1. `User -> ATM.cancel()`, then `current state.cancel()`, then `Session.clear()`
- **Failure and undo (per flow):**
  - **Insert a card and authenticate:** a wrong PIN changes only the card's count, and a correct PIN resets it. A blocked card is ejected, and can't be inserted again.
  - **Check the balance:** an unknown account refuses. Nothing changes.
  - **Deposit:** the validation happens before any change. If the bank refuses the deposit, no notes are added, and a failed transaction is recorded.
  - **Withdraw:** 
  - **Cancel:** in `IDLE` there is no card, and in `DISPENSING` the cash is being handed out, so it is refused.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added.
  - If it were needed later, a lock inside `Account` around `deposit` and `withdraw`, and one around `CashDispenser.take`, would protect the two shared facts.
- **Key decisions:**
  1. **A withdrawal checks first and changes last.** *Rejected:* taking the notes first, which would need a restock when the account refuses.
  2. **The card owns the wrong-PIN count, and "blocked" is computed.** *Rejected:* a stored `blocked` flag beside the count.
  3. **States return a `Response`, and the machine applies it.** *Rejected:* states calling a `change_state` method on the machine, which anyone could call.
  4. **The link from a card to an account is stored once, in `BankService`.** *Rejected:* the card holding its account and the account holding its cards.
  5. **Failed transactions are recorded too.** *Rejected:* recording only successes, which hides refused withdrawals.
- **Edge cases:**
  - A wrong PIN is counted, and a correct PIN resets the count.
  - A blocked card is ejected, and a blocked or unknown card is refused at insertion.
  - Cancel works in the card and authenticated states, and is refused when idle or dispensing.
  - A withdrawal of 0 or less is refused.
  - The account has too little money: refused, with the machine's cash untouched.
  - The machine has too little cash, or can't make the amount exactly: refused, with the account untouched.
  - A deposit with an unknown note value is refused.
  - A failed `take` after the debit pays the money back.
- **Extensibility (change → what to add):**
  - A new note denomination (200) → One more number in the list passed to `CashDispenser`, then `add_notes` *(edits existing code: No. It is data)*
  - A different number of allowed wrong PINs → A different `max_attempts` in `create_card` *(edits existing code: No. It is data)*
  - More accounts and cards → Calls to `create_account` and `create_card` *(edits existing code: No)*
  - A new operation (transfer) → A new `OperationType` value and a new branch in `AuthenticatedState.select_operation` *(edits existing code: Yes, one branch. That is where operations live)*
  - A new state, such as OUT_OF_SERVICE → A new state class, one new enum value, one new map entry *(edits existing code: The one state that moves into it must return the new type, so that return value changes)*
- **Follow-up questions:**
  1. **"Doesn't largest-first fail for some amounts?"** Yes. With notes 100, 50 and 20, an amount of 60 is payable as 20+20+20, but largest-first takes one 50 and has 10 left, so the plan is None and the withdrawal is refused. The requirement asks for largest first. A fix is to search the combinations (backtracking or dynamic programming) inside `CashDispenser.plan`, which changes only that method.
  2. **"What if two ATMs use the same account?"** Put a lock inside `Account` around `deposit` and `withdraw`, so the check and the change are one step.
  3. **"Why is Dispensing a state if it is instant?"** It names the condition from requirement 2 and refuses every action, including cancel, while the cash is handed out.
- **Out of scope (one-line answer if asked):**
  - Amounts that largest-first can't make → They are refused. See follow-up question 1.
  - Keeping or swallowing a blocked card → The card comes out and can't be inserted again.
  - A daily withdrawal limit → No requirement asks for it. It would be a rule in `DispensingState` before step 3.
  - Changing a PIN → Not required. It would be a method on `Card`.
  - Loading cash into the machine → The machine's cash is loaded at setup with `add_notes`, and by deposits.
  - Printing the receipt → `to_receipt()` gives the text. Printing is a hardware concern.
  - Transfers between accounts → See the Extensibility list above.
  - Concurrent users of one account → See follow-up question 2.


---

## Type C: Matching / Dispatch

#### 08 · Ride Sharing
- **Enums:**
  - **VehicleType**: `AUTO`, `SEDAN`, `SUV`
  - **DriverStatus**: `OFFLINE`, `AVAILABLE`, `BUSY`
  - **RideStatus**: `REQUESTED`, `ACCEPTED`, `IN_PROGRESS`, `COMPLETED`, `CANCELLED`
- **Class details (job · what it owns · key methods):**
  - **Location**: Describes a point and measures distance to another · *owns:* x, y
    - `get_x() -> float`, `get_y() -> float`, `distance_to(other: Location) -> float`
  - **Vehicle**: Describes one registered vehicle · *owns:* plate, type
    - `get_plate_number() -> str`, `get_vehicle_type() -> VehicleType`
  - **ReputationScore**: Adds up ratings into an average · *owns:* total stars, count
    - `add(stars: int) -> bool`, `get_average() -> float or None`, `get_count() -> int`
  - **Rider**: Is a registered rider with a rating · *owns:* id, name, rating
    - `get_rider_id() -> str`, `get_name() -> str`, `add_rating(stars: int) -> bool`, `get_average_rating() -> float or None`
  - **Driver**: Is a registered driver and guards their own availability · *owns:* status, location, vehicles, rating
    - `get_driver_id() -> str`, `get_name() -> str`, `get_status() -> DriverStatus`, `get_average_rating() -> float or None`, `add_vehicle(vehicle: Vehicle) -> bool`, `go_online(plate_number: str, location: Location) -> bool`, `go_offline() -> bool`, `update_location(location: Location) -> bool`, `try_reserve() -> bool`, `release() -> bool`, `finish_ride(location: Location) -> bool`, `add_rating(stars: int) -> bool`, `snapshot() -> DriverInfo or None`
  - **DriverInfo**: Gives a read-only picture of one available driver · *owns:* a copy of key facts
    - `get_driver_id() -> str`, `get_location() -> Location`, `get_vehicle_type() -> VehicleType`, `get_average_rating() -> float or None`
  - **RateCard**: Holds the three prices for one vehicle type · *owns:* base, per km, per minute
    - `get_base() -> int`, `get_per_km() -> int`, `get_per_minute() -> int`
  - **PricingStrategy**: Turns distance, time and vehicle type into a fare
    - `calculate_fare(vehicle_type: VehicleType, distance_km: float, duration_minutes: float) -> int`
  - **DistanceTimePricing**: Charges base plus per-km plus per-minute · *owns:* the rate cards
    - `calculate_fare(vehicle_type: VehicleType, distance_km: float, duration_minutes: float) -> int`
  - **MatchingStrategy**: Chooses a driver for a pickup
    - `select(candidates: list[DriverInfo], pickup: Location) -> DriverInfo or None`
  - **NearestDriverStrategy**: Chooses the nearest driver within a radius · *owns:* the radius
    - `select(candidates: list[DriverInfo], pickup: Location) -> DriverInfo or None`
  - **PaymentMethod**: Takes a payment
    - `pay(amount: int) -> bool`
  - **CardPayment**: Pays by card · *owns:* amount paid
    - `pay(amount: int) -> bool`
  - **UpiPayment**: Pays by UPI · *owns:* amount paid
    - `pay(amount: int) -> bool`
  - **Ride**: Holds one trip and runs its own stages, payment and ratings · *owns:* status, driver, fare
    - `get_ride_id() -> str`, `get_status() -> RideStatus`, `get_driver() -> Driver or None`, `get_fare() -> int or None`, `get_vehicle_type() -> VehicleType`, `get_pickup() -> Location`, `get_rejected_driver_ids() -> set[str]`, `is_paid() -> bool`, `assign(driver: Driver) -> bool`, `accept(driver_id: str) -> bool`, `reject(driver_id: str) -> bool`, `cancel_unmatched() -> bool`, `cancel(rider_id: str) -> bool`, `start(driver_id: str, now: datetime) -> bool`, `complete(driver_id: str, now: datetime, pricing: PricingStrategy) -> bool`, `pay(rider_id: str, payment: PaymentMethod) -> bool`, `rate_driver(rider_id: str, stars: int) -> bool`, `rate_rider(driver_id: str, stars: int) -> bool`
  - **RideService**: Registers people, matches drivers, and runs each action in the right order · *owns:* the riders, drivers and rides
    - `register_rider(rider_id: str, name: str) -> Rider or None`, `register_driver(driver_id: str, name: str) -> Driver or None`, `add_vehicle(driver_id: str, plate_number: str, vehicle_type: VehicleType) -> bool`, `go_online(driver_id: str, plate_number: str, location: Location) -> bool`, `go_offline(driver_id: str) -> bool`, `update_location(driver_id: str, location: Location) -> bool`, `estimate_fare(pickup: Location, dropoff: Location, vehicle_type: VehicleType) -> int or None`, `request_ride(rider_id: str, pickup: Location, dropoff: Location, vehicle_type: VehicleType, now: datetime) -> Ride or None`, `respond_to_ride(ride_id: str, driver_id: str, accept: bool) -> bool`, `start_ride(ride_id: str, driver_id: str, now: datetime) -> bool`, `complete_ride(ride_id: str, driver_id: str, now: datetime) -> bool`, `cancel_ride(ride_id: str, rider_id: str) -> bool`, `pay_ride(ride_id: str, rider_id: str, payment: PaymentMethod) -> bool`, `rate_driver(ride_id: str, rider_id: str, stars: int) -> bool`, `rate_rider(ride_id: str, driver_id: str, stars: int) -> bool`, `get_ride(ride_id: str) -> Ride or None`, `reserve_driver(vehicle_type: VehicleType, pickup: Location, excluded: set[str]) -> Driver or None`
- **Relationships:**
  - RideService **has** Rider, Driver, Ride (created inside the register and request methods).
  - Rider **has** ReputationScore. Driver **has** ReputationScore.
  - Driver **has** Vehicle (vehicles die with the driver record).
  - Driver **uses** Location (a value that lives on). Ride **uses** Rider, Driver, Location, PaymentMethod.
  - DriverInfo **uses** Location. DistanceTimePricing **uses** RateCard.
  - RideService **uses** MatchingStrategy, PricingStrategy (passed in).
  - DistanceTimePricing **implements** PricingStrategy. NearestDriverStrategy **implements** MatchingStrategy. CardPayment, UpiPayment **implement** PaymentMethod.
  - Enums are always **uses**: Vehicle, Ride and DriverInfo to VehicleType; Driver to DriverStatus; Ride to RideStatus.
  - `PricingStrategy` also appears as a method input on `Ride`, but no attribute of `Ride` holds it, so there is no line there.
- **Call flows:**
  - **Request a ride**
    1. `Rider -> RideService.request_ride(...)`
    2. `RideService -> Driver.snapshot()`
    3. `RideService -> MatchingStrategy.select(candidates, pickup)`
    4. `RideService -> Driver.try_reserve()`
    5. `RideService -> Ride(...)`
  - **Driver responds**
    1. `Driver -> RideService.respond_to_ride(ride_id, driver_id, accept)`
    2. `RideService -> Ride.accept(driver_id)`
    3. On reject
  - **Complete and pay**
    1. `Driver -> RideService.complete_ride(...)`, then `RideService -> Ride.complete(driver_id, now, pricing)`
    2. `Ride -> PricingStrategy.calculate_fare(...)`, then `Ride -> Driver.finish_ride(dropoff)`
    3. `Rider -> RideService.pay_ride(...)`, then `Ride -> PaymentMethod.pay(fare)`
  - **Rate**
    1. `Rider -> RideService.rate_driver(...)`, then `Ride.rate_driver(...)`, then `Driver.add_rating(stars)`
- **Failure and undo (per flow):**
  - **Register and set availability:** returns None or False and nothing changes.
  - **Estimate the fare:** None for identical locations. Nothing is stored, because an estimate is only a calculation.
  - **Request a ride:** if the ride can't be built, call `driver.release()` so the driver is not left BUSY. No other step changes anything.
  - **Reserve a driver (private helper):** a refused `try_reserve` changes nothing, so there is nothing to undo.
  - **Driver accepts or rejects:** a wrong driver or a wrong stage is refused with no change. Each step either completes or is undone by releasing the one driver it reserved.
  - **Start, complete and cancel:** wrong person or wrong stage returns False. A trip in progress can't be cancelled, so a driver is never freed mid-trip by mistake.
  - **Pay and rate:** returns False and nothing changes. A bad star value does not use up the one rating.
- **Concurrency:**
  - **Driver:** the lock sits inside `Driver`, around every status change. `try_reserve` checks AVAILABLE and sets BUSY as one step, so two requests choosing the same driver can't both win.
  - **Ride:** the lock sits inside `Ride`, around every method.
  - **Rating and payment:** the locks sit inside `ReputationScore` and each payment class.
  - **Maps and ids:** the three maps are thread-safe maps, and ids come from an atomic counter.
  - **Lock order:** always `Ride` first, then a `Driver`, a payment or a score, one at a time. `try_reserve` is called with no ride lock held. No method takes them in the other order, so no deadlock is possible.
  - **Stale snapshots:** a snapshot can go out of date before `try_reserve`. A refused reservation makes the helper pick again, so nobody is double-booked.
- **Key decisions:**
  1. **A driver is reserved before the ride is created, using `try_reserve`.** *Rejected:* creating the ride first and assigning a driver afterwards. A second request could assign the same driver in between.
  2. **The estimate is computed on demand, and the final fare is stored once at completion.** *Rejected:* storing the estimate on the ride and later comparing it with the final fare.
  3. **Matching, pricing and payment are all passed-in strategies.** *Rejected:* one `if` chain per rule inside the service.
  4. **`Ride` runs its own stages and checks who is acting.** *Rejected:* the service changing the status itself. A second caller could skip a check.
  5. **A rating is a total and a count, not a list of ratings.** *Rejected:* a stored average, which would be a second copy.
- **Edge cases:**
  - No driver is nearby or free, so the request returns None and nothing is created.
  - Two riders race for the same driver: one wins, the other picks the next best.
  - A driver rejects and the next nearest is offered. If everyone rejects, the ride is cancelled.
  - A driver who is not the offered one can't accept, start or complete.
  - A trip in progress can't be cancelled.
  - A declined payment leaves the ride unpaid, and another method can be tried.
  - A rating before completion, a second rating, or stars outside 1 to 5 is refused.
  - A driver who is busy can't go offline.
- **Extensibility (change → what to add):**
  - A new matching rule (highest rated, balanced load) → A new class implementing `MatchingStrategy`, passed to `RideService` *(edits existing code: No)*
  - A new pricing rule or new rates → A new class implementing `PricingStrategy`, or new `RateCard` values *(edits existing code: No)*
  - A new payment method (wallet, cash) → A new class implementing `PaymentMethod`, passed to `pay_ride` *(edits existing code: No)*
  - A new vehicle type (bike, luxury) → One value in `VehicleType` and a `RateCard` for it *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"How would you add surge pricing?"** It needs a new input, the current demand, so the pricing method gains a parameter. That is one real change to `PricingStrategy`, and then a new class for the surge rule.
  2. **"What if a driver never responds?"** Add a time limit to an offer and a sweep that rejects stale offers, the same pattern as expiring holds. It is not built here because no requirement asks for it.
  3. **"What if no driver is within the radius?"** Pass a larger radius or a different `MatchingStrategy`. The service doesn't change.
- **Out of scope (one-line answer if asked):**
  - Driver response time limits → See follow-up question 2.
  - Live tracking and road distance → Distance is the straight line from pickup to drop-off, so a real route would change `Location.distance_to` and add tracking.
  - Cancellation fees or driver cancels → Only the rider can cancel, and without a fee. A fee would be a new rule inside `Ride.cancel`.
  - Pooled or shared rides → One rider per ride.
  - Booking for a later time → No requirement asks for it.
  - The rider choosing from several estimates → The rider picks one vehicle type and sees one estimate.
  - Ride history views → `get_ride` reads one ride. A history list would be a new view.
  - Real payment gateways → `pay` is the one place a gateway call goes.

#### 25 · Cab Booking (with scheduled trips)
- **Enums:**
  - **CabType**: `MINI`, `SEDAN`, `SUV`
  - **CabStatus**: `AVAILABLE`, `ON_TRIP`, `OFF_DUTY`
  - **TripStatus**: `BOOKED`, `STARTED`, `COMPLETED`, `CANCELLED`
- **Class details (job · what it owns · key methods):**
  - **Location**: Describes a point and measures distance · *owns:* x, y
    - `distance_to(other: Location) -> float`
  - **TimeWindow**: Describes a start and end and checks overlap · *owns:* start, end
    - `get_start() -> datetime`, `get_end() -> datetime`, `overlaps(other: TimeWindow) -> bool`
  - **RateCard**: Holds the prices for one cab type · *owns:* base, per km
    - `get_base() -> int`, `get_per_km() -> int`
  - **FareCalculator**: Works out a fare from distance and cab type · *owns:* the rate cards
    - `calculate(cab_type: CabType, distance_km: float) -> int`
  - **CabInfo**: Gives a read-only picture of one cab for the rule · *owns:* a copy of key facts
  - **Cab**: Holds a cab's type, status, location and calendar, and guards them · *owns:* status, location, reserved windows
    - `get_cab_id() -> str`, `get_cab_type() -> CabType`, `get_status() -> CabStatus`, `get_location() -> Location`, `snapshot_if_free(window: TimeWindow) -> CabInfo or None`, `try_reserve(trip_id: str, window: TimeWindow) -> bool`, `release(trip_id: str) -> bool`, `start_trip(trip_id: str) -> bool`, `finish_trip(trip_id: str, dropoff: Location) -> bool`, `go_off_duty() -> bool`, `go_on_duty() -> bool`, `update_location(location: Location) -> bool`
  - **AllocationStrategy**: Chooses a cab for a pickup
    - `select(candidates: list[CabInfo], pickup: Location) -> CabInfo or None`
  - **NearestCabStrategy**: Chooses the nearest cab
    - `select(...)`
  - **Trip**: Holds one trip and runs its own stages · *owns:* status, cab, fare
    - `start(cab_id: str, now: datetime) -> bool`, `complete(cab_id: str) -> bool`, `cancel(rider_id: str) -> bool`
  - **CabService**: Books, starts, completes and cancels trips, and manages cabs · *owns:* the cabs, the trips
    - `add_cab(cab_id: str, driver_name: str, cab_type: CabType, location: Location) -> bool`, `go_off_duty(cab_id: str) -> bool`, `go_on_duty(cab_id: str) -> bool`, `update_location(cab_id: str, location: Location) -> bool`, `estimate_fare(pickup: Location, dropoff: Location, cab_type: CabType) -> int`, `book(rider_id: str, pickup: Location, dropoff: Location, cab_type: CabType, pickup_time: datetime, now: datetime) -> Trip or None`, `start_trip(trip_id: str, cab_id: str, now: datetime) -> bool`, `complete_trip(trip_id: str, cab_id: str) -> bool`, `cancel_trip(trip_id: str, rider_id: str) -> bool`, `get_trip(trip_id: str) -> Trip or None`
- **Relationships:**
  - CabService **has** Cab, Trip (created inside `add_cab` and `book`).
  - Trip **uses** Cab, Location, TimeWindow. Cab **uses** Location, TimeWindow.
  - CabService **uses** AllocationStrategy, FareCalculator (passed in).
  - FareCalculator **uses** RateCard. CabInfo **uses** Location.
  - NearestCabStrategy **implements** AllocationStrategy.
  - Enums are always **uses**: Cab to CabType and CabStatus, Trip to TripStatus, FareCalculator and CabInfo to CabType.
- **Call flows:**
  - **Book**
    1. `Rider -> CabService.book(...)`
    2. `CabService -> Cab.snapshot_if_free(window)`
    3. `CabService -> AllocationStrategy.select(candidates, pickup)`
    4. `CabService -> Cab.try_reserve(trip_id, window)`
    5. `CabService -> FareCalculator.calculate(cab_type, distance)`, then `Trip(...)`
  - **Start, complete, cancel**
    1. `Driver or rider -> CabService.start_trip / complete_trip / cancel_trip`
    2. `CabService -> Trip.start / complete / cancel`
    3. `Trip -> Cab.start_trip / finish_trip / release`
- **Failure and undo (per flow):**
  - **Book:** steps 1 to 4 change nothing. If building the trip ever failed after step 5, `cab.release(trip_id)` would undo the reservation.
  - **Start and complete:** the wrong cab or the wrong stage is refused with no change.
  - **Cancel:** another rider, or a trip that has started, is refused.
- **Concurrency:**
  - **Cab:** the lock sits inside `Cab`, around every method. `try_reserve` checks for overlap and stores the window as one step, so two bookings can't give one cab overlapping trips.
  - **Trip:** the lock sits inside `Trip`, around `start`, `complete` and `cancel`.
  - **Stale copies:** a `CabInfo` may be out of date by the time `try_reserve` runs. A refused reservation makes the service choose again, so no rider is given a double-booked cab.
  - **Maps and ids:** both maps are thread-safe, and ids come from an atomic counter.
  - **Lock order:** always the trip first, then its cab. `book` takes only cab locks, one at a time. So no deadlock is possible.
- **Key decisions:**
  1. **A cab has a calendar of windows, not just a busy flag.** *Rejected:* a busy flag, which works only for "book now".
  2. **Each cab guards its calendar with a lock.** *Rejected:* one lock for the whole service.
  3. **The rule sees copies, and the reservation can refuse.** *Rejected:* passing live cabs to the rule.
  4. **The fare is fixed at booking.** *Rejected:* computing it at the end from time taken, which no requirement asks for.
  5. **`Trip` tells its cab to start, finish or release.** *Rejected:* the service changing the trip and the cab separately, where one could be forgotten.
- **Edge cases:**
  - Two bookings racing for one cab's window: one wins, the other picks the next cab.
  - A trip ending at 9:30 and one starting at 9:30 can use the same cab.
  - A cab that is off duty is never offered.
  - A cab with future bookings can't go off duty.
  - A pickup time in the past, or identical pickup and drop-off, is refused.
  - A trip can't start before its window or be started by another cab.
  - A started trip can't be cancelled.
  - A missing price for a cab type is refused at setup.
- **Extensibility (change → what to add):**
  - A new allocation rule (least busy today, highest rated) → A new class implementing `AllocationStrategy`, passed to the service *(edits existing code: No)*
  - New prices → A new `RateCard` in the `FareCalculator` table *(edits existing code: No. It is data)*
  - A new cab type → One new `CabType` value and its `RateCard` *(edits existing code: No logic changes)*
  - Longer buffers or a different average speed → Different numbers passed to the service *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"For a later booking, isn't the cab's current location a poor guess?"** Yes. A better rule uses the drop-off of the cab's previous trip before that time. That is a new `AllocationStrategy` with an extra field in `CabInfo`.
  2. **"What if a trip runs late and clashes with the next booking?"** The buffer reduces the risk. A complete answer moves the next booking to another cab when a trip overruns, which is a new rule in `Trip.complete`.
  3. **"Should we assign a cab for a booking a week away right now?"** Many systems assign close to the time instead, when locations are known. That would hold the booking unassigned and run allocation later from a scheduler.
- **Out of scope (one-line answer if asked):**
  - Live tracking and road distance → Distance is a straight line, and the cab's location is updated by the driver.
  - A final fare based on time taken → The fare is fixed at booking from distance and type.
  - Payment → Not required.
  - Cancellation fees → Not required. It would be a rule in `Trip.cancel`.
  - Assigning cabs close to the time → See follow-up question 3.
  - Trips that overrun → See follow-up question 2.

#### 14 · Food Delivery
- **Enums:**
  - **OrderStatus**: `PLACED`, `ACCEPTED`, `PREPARING`, `PICKED_UP`, `DELIVERED`, `CANCELLED`
  - **PartnerStatus**: `AVAILABLE`, `BUSY`
- **Class details (job · what it owns · key methods):**
  - **MenuItem**: Describes one dish · *owns:* id, name, price
    - `get_item_id() -> str`, `get_name() -> str`, `get_price() -> int`
  - **ReputationScore**: Adds ratings into an average · *owns:* total stars, count
    - `add(stars: int) -> bool`, `get_average() -> float or None`, `get_count() -> int`
  - **Restaurant**: Holds a menu and a rating, and says if it matches a search · *owns:* the menu, rating
    - `get_restaurant_id() -> str`, `get_name() -> str`, `add_item(item_id: str, name: str, price: int) -> bool`, `get_menu() -> list[MenuItem]`, `get_item(item_id: str) -> MenuItem or None`, `matches(text: str) -> bool`, `add_rating(stars: int) -> bool`, `get_average_rating() -> float or None`
  - **RestaurantCatalog**: Finds restaurants by id or by search text · *owns:* the restaurants
    - `add_restaurant(restaurant_id: str, name: str) -> Restaurant or None`, `get_restaurant(restaurant_id: str) -> Restaurant or None`, `list_all() -> list[Restaurant]`, `search(text: str) -> list[Restaurant]`
  - **Cart**: Holds one user's chosen items from one restaurant · *owns:* restaurant, item quantities
    - `get_restaurant() -> Restaurant or None`, `get_quantities() -> map{str: int}`, `is_empty() -> bool`, `add_item(restaurant: Restaurant, item_id: str, quantity: int) -> bool`, `remove_item(item_id: str) -> bool`, `clear() -> None`
  - **OrderLine**: Records one ordered dish at the price it was bought · *owns:* name, price, quantity
    - `get_item_name() -> str`, `get_unit_price() -> int`, `get_quantity() -> int`, `get_line_total() -> int`
  - **DeliveryPartner**: Is a delivery person who guards their own availability · *owns:* status, rating
    - `get_partner_id() -> str`, `get_name() -> str`, `get_status() -> PartnerStatus`, `try_reserve() -> bool`, `release() -> bool`, `add_rating(stars: int) -> bool`, `get_average_rating() -> float or None`
  - **DiscountStrategy**: Turns an order subtotal into a discount
    - `discount_for(subtotal: int) -> int`
  - **PercentDiscount**: Takes a percentage off, up to a cap · *owns:* percent, cap
    - `discount_for(subtotal: int) -> int`
  - **FlatDiscount**: Takes a fixed amount off · *owns:* amount
    - `discount_for(subtotal: int) -> int`
  - **Coupon**: Joins a code, a minimum order and a discount rule · *owns:* code, minimum, rule
    - `get_code() -> str`, `calculate_discount(subtotal: int) -> int or None`
  - **PaymentMethod**: Takes and returns a payment
    - `pay(amount: int) -> bool`, `refund(amount: int) -> bool`
  - **CardPayment**: Pays and refunds by card · *owns:* amount paid
    - `pay(amount: int) -> bool`, `refund(amount: int) -> bool`
  - **UpiPayment**: Pays and refunds by UPI · *owns:* amount paid
    - `pay(amount: int) -> bool`, `refund(amount: int) -> bool`
  - **OrderObserver**: Reacts to an order changing stage
    - `on_status_changed(order_id: str, user_id: str, status: OrderStatus) -> None`
  - **UserNotifier**: Keeps the messages sent to each user · *owns:* the messages
    - `on_status_changed(order_id: str, user_id: str, status: OrderStatus) -> None`, `get_messages(user_id: str) -> list[str]`
  - **OrderTracking**: Gives a read-only picture of one order · *owns:* a copy of key facts
    - `get_order_id() -> str`, `get_status() -> OrderStatus`, `get_restaurant_name() -> str`, `get_partner_name() -> str or None`, `get_total() -> int`
  - **Order**: Holds one order and runs its own stages, refunds and ratings · *owns:* status, partner, payment
    - `get_order_id() -> str`, `get_user_id() -> str`, `get_status() -> OrderStatus`, `get_partner() -> DeliveryPartner`, `get_restaurant() -> Restaurant`, `get_subtotal() -> int`, `get_total() -> int`, `track() -> OrderTracking`, `accept(restaurant_id: str) -> bool`, `reject(restaurant_id: str) -> bool`, `start_preparing(restaurant_id: str) -> bool`, `pick_up(partner_id: str) -> bool`, `deliver(partner_id: str) -> bool`, `cancel(user_id: str) -> bool`, `rate_restaurant(user_id: str, stars: int) -> bool`, `rate_partner(user_id: str, stars: int) -> bool`
  - **OrderService**: Runs the ordering flow and holds the carts, orders, partners and coupons · *owns:* carts, orders, partners, coupons
    - `add_partner(partner_id: str, name: str) -> DeliveryPartner or None`, `add_coupon(code: str, min_order_total: int, strategy: DiscountStrategy) -> bool`, `add_to_cart(user_id: str, restaurant_id: str, item_id: str, quantity: int) -> bool`, `remove_from_cart(user_id: str, item_id: str) -> bool`, `view_cart(user_id: str) -> list[OrderLine] or None`, `place_order(user_id: str, delivery_address: str, payment: PaymentMethod, coupon_code: str or None) -> Order or None`, `respond_to_order(order_id: str, restaurant_id: str, accept: bool) -> bool`, `start_preparing(order_id: str, restaurant_id: str) -> bool`, `pick_up(order_id: str, partner_id: str) -> bool`, `deliver(order_id: str, partner_id: str) -> bool`, `cancel_order(order_id: str, user_id: str) -> bool`, `track_order(order_id: str, user_id: str) -> OrderTracking or None`, `rate_restaurant(order_id: str, user_id: str, stars: int) -> bool`, `rate_partner(order_id: str, user_id: str, stars: int) -> bool`, `reserve_partner() -> DeliveryPartner or None`
- **Relationships:**
  - RestaurantCatalog **has** Restaurant (created inside `add_restaurant`).
  - Restaurant **has** MenuItem, ReputationScore. DeliveryPartner **has** ReputationScore.
  - Order **has** OrderLine (the lines die with the order).
  - OrderService **has** Cart, Order, DeliveryPartner, Coupon (created inside the add, cart and place methods).
  - Cart **uses** Restaurant (the restaurant lives on without the cart).
  - Order **uses** Restaurant, DeliveryPartner, PaymentMethod, OrderObserver.
  - Coupon **uses** DiscountStrategy (passed in).
  - OrderService **uses** RestaurantCatalog, OrderObserver (passed in).
  - PercentDiscount, FlatDiscount **implement** DiscountStrategy. CardPayment, UpiPayment **implement** PaymentMethod. UserNotifier **implements** OrderObserver.
  - Enums are always **uses**: Order to OrderStatus, DeliveryPartner to PartnerStatus.
  - `OrderTracking` appears only as a return type, so no attribute holds it and it has no line.
- **Call flows:**
  - **Browse and search**
    1. `User -> RestaurantCatalog.search(text)`
    2. `RestaurantCatalog -> Restaurant.matches(text)`, then `Restaurant.get_menu()`
  - **Fill the cart**
    1. `User -> OrderService.add_to_cart(user_id, restaurant_id, item_id, quantity)`
    2. `OrderService -> RestaurantCatalog.get_restaurant(restaurant_id)`
    3. `OrderService -> Cart.add_item(restaurant, item_id, quantity)`
  - **Place an order**
    1. `User -> OrderService.place_order(...)`
    2. `OrderService -> Cart`, then `Restaurant.get_item(...)`
    3. `OrderService -> Coupon.calculate_discount(subtotal)`, then `DiscountStrategy.discount_for(subtotal)`
    4. `OrderService -> DeliveryPartner.try_reserve()`
    5. `OrderService -> PaymentMethod.pay(total)`, then `Order(...)`
  - **Restaurant and partner steps**
    1. `Restaurant -> OrderService.respond_to_order(...)`, then `Order.accept(restaurant_id)`
    2. `Order -> PaymentMethod.refund(total)` and `DeliveryPartner.release()`
    3. `Order -> OrderObserver.on_status_changed(...)`
- **Failure and undo (per flow):**
  - **Browse and search:** no match returns an empty list. Nothing changes.
  - **Add to the cart:** the cart refuses a second restaurant, a dish not on the menu, or a quantity of 0 or less, and nothing changes.
  - **Place an order:** steps 1 to 3 change nothing. A declined payment undoes the one change made so far, the partner reservation.
  - **Restaurant steps:** a restaurant that does not own the order, or the wrong stage, is refused. If a refund fails, the rejection is refused and the order stays `PLACED`.
  - **Pick up, deliver and cancel:** the wrong person or the wrong stage is refused with no change. An order being cooked can't be cancelled by the user.
  - **Track and rate:** another user's order, a rating before delivery, a second rating, or stars outside 1 to 5 is refused. A bad star value does not use up the one rating.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added.
  - `try_reserve` is already a checked step. If concurrency were required, one lock inside `DeliveryPartner` around its status changes would make it safe, and the order stages would need a lock inside `Order`.
- **Key decisions:**
  1. **A partner is reserved before the payment is taken.** *Rejected:* paying first and assigning later. A paid order with no partner would need a refund.
  2. **`Order` runs its own stages and tells its observers.** *Rejected:* the service calling the notifier after each change. A direct call on the order would be silent.
  3. **The cart stores ids and counts, and prices are read from the menu at order time.** *Rejected:* a cart that copies prices.
  4. **The order stores the discount it got, once.** *Rejected:* recomputing the discount from the coupon each time.
  5. **The coupon owns its minimum-order rule, and a strategy owns the discount.** *Rejected:* an `if` on the coupon kind inside the order.
- **Edge cases:**
  - A cart can hold only one restaurant's items.
  - A dish removed from the menu after it was added refuses the order.
  - A coupon that is unknown, or whose minimum is not met, refuses the order.
  - A coupon never takes the total below zero. If the total is 0, no payment is taken.
  - No free partner means no order and no payment.
  - A declined payment releases the reserved partner.
  - A rejection or a cancel refunds the payment and releases the partner.
  - An order being cooked can't be cancelled.
  - Another user can't track or cancel someone's order.
  - A failing observer does not stop an order from changing.
- **Extensibility (change → what to add):**
  - A new kind of coupon (buy one get one) → A new class implementing `DiscountStrategy`, used in a new `Coupon` *(edits existing code: No)*
  - A new payment method (wallet) → A new class implementing `PaymentMethod`, passed to `place_order` *(edits existing code: No)*
  - Another party told about order changes (the restaurant, analytics) → A new class implementing `OrderObserver`, passed to the service *(edits existing code: No)*
  - More restaurants, dishes, partners and coupons → Calls to the add methods *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"What if the restaurant never responds?"** Add a time limit and a sweep that rejects stale `PLACED` orders, the same pattern as expiring a hold. It is not built here because no requirement asks for it.
  2. **"How would you pick the nearest partner?"** Replace `reserve_partner`'s id order with a matching rule that needs locations, passed in as a strategy. Today no requirement mentions distance.
  3. **"What if two orders race for the last partner?"** `try_reserve` is one checked step. With real concurrency, a lock inside `DeliveryPartner` makes it atomic, and the loser tries the next partner or gets None.
- **Out of scope (one-line answer if asked):**
  - User accounts → Users are only an id. Accounts would be a new class.
  - Choosing the nearest partner → See follow-up question 2.
  - A time limit for the restaurant → See follow-up question 1.
  - A history of stage changes with times → `track_order` shows the current stage. Observers received each change, so a history observer could record them.
  - Coupon limits (one use per user) → Not required. It would be a rule inside `Coupon`.
  - Delivery fees and taxes → Not required. They would be added to the total computation in `Order`.
  - Several restaurants in one cart → Requirement 2 says one.
  - Real payment and notification services → `pay`, `refund` and the notifier are the one place each outside call goes.


---

## Type D: Money / Ledger

#### 03 · Splitwise
- **Class details (job · what it owns · key methods):**
  - **User**: Describes one registered person · *owns:* id, name
    - `get_user_id() -> str`, `get_name() -> str`
  - **Expense**: Records one payment that was shared · *owns:* payer, amount, each person's share
    - `get_payer_id() -> str`, `get_amount() -> int`, `get_shares() -> map{str: int}`
  - **Transfer**: Records money moved from one person to another · *owns:* from, to, amount
    - `get_from_user_id() -> str`, `get_to_user_id() -> str`, `get_amount() -> int`
  - **SplitStrategy**: Turns a total into each person's share
    - `calculate_shares(total: int) -> map{str: int} or None`
  - **EqualSplit**: Shares a total equally among chosen people · *owns:* the chosen people
    - `calculate_shares(total: int) -> map{str: int} or None`
  - **ExactSplit**: Uses exact amounts per person · *owns:* the amounts
    - `calculate_shares(total: int) -> map{str: int} or None`
  - **PercentSplit**: Uses a percentage per person · *owns:* the percentages
    - `calculate_shares(total: int) -> map{str: int} or None`
  - **DebtSimplifier**: Turns net balances into the fewest payments
    - `simplify(net_balances: map{str: int}) -> list[Transfer]`
  - **Group**: Keeps the members and the expense and transfer history, and computes balances from it · *owns:* members, expenses, transfers
    - `get_group_id() -> str`, `get_name() -> str`, `get_member_ids() -> list[str]`, `is_member(user_id: str) -> bool`, `add_member(user_id: str) -> bool`, `add_expense(payer_id: str, amount: int, strategy: SplitStrategy) -> Expense or None`, `settle(from_user_id: str, to_user_id: str, amount: int) -> bool`, `get_balance_between(user_a: str, user_b: str) -> int or None`, `get_balances_of(user_id: str) -> map{str: int} or None`, `get_net_balances() -> map{str: int}`
  - **SplitwiseService**: Registers users, creates groups, and runs each action in the right order · *owns:* users, groups
    - `register_user(user_id: str, name: str) -> User or None`, `create_group(group_id: str, name: str, member_ids: list[str]) -> Group or None`, `add_member(group_id: str, user_id: str) -> bool`, `add_expense(group_id: str, payer_id: str, amount: int, strategy: SplitStrategy) -> Expense or None`, `settle(group_id: str, from_user_id: str, to_user_id: str, amount: int) -> bool`, `get_group_balances(group_id: str, user_id: str) -> map{str: int} or None`, `get_balances(user_id: str) -> map{str: int} or None`, `get_simplified_payments(group_id: str) -> list[Transfer] or None`
- **Relationships:**
  - SplitwiseService **has** User, Group (created inside `register_user` and `create_group`).
  - Group **has** Expense, Transfer (created inside `add_expense` and `settle`).
  - SplitwiseService **uses** DebtSimplifier (passed in).
  - EqualSplit, ExactSplit, PercentSplit **implement** SplitStrategy.
  - There are no enums, so there are no enum lines.
  - `SplitStrategy` and `Transfer` also appear as a method input or output on other classes, but no attribute holds them there, so they have no line from those classes.
- **Call flows:**
  - **Register and create a group**
    1. `Caller -> SplitwiseService.register_user(user_id, name)`
    2. `Caller -> SplitwiseService.create_group(group_id, name, member_ids)`
  - **Add an expense**
    1. `Caller -> SplitwiseService.add_expense(group_id, payer_id, amount, strategy)`
    2. `SplitwiseService -> groups.get(group_id)`
    3. `SplitwiseService -> Group.add_expense(payer_id, amount, strategy)`
    4. `Group -> SplitStrategy.calculate_shares(amount)`
    5. `Group -> Expense(...)`
  - **View balances**
    1. `Caller -> SplitwiseService.get_balances(user_id)`
    2. `SplitwiseService -> Group.get_balances_of(user_id)`
  - **Settle**
    1. `Caller -> SplitwiseService.settle(group_id, from, to, amount)`
    2. `SplitwiseService -> Group.settle(from, to, amount)`
  - **Fewest payments**
    1. `Caller -> SplitwiseService.get_simplified_payments(group_id)`
    2. `SplitwiseService -> Group.get_net_balances()`, then `DebtSimplifier.simplify(net_balances)`
- **Failure and undo (per flow):**
  - **Register a user:** nothing is stored.
  - **Create a group:** nothing is created.
  - **Add an expense:** every check happens before the expense is appended, so a failure changes nothing and there is nothing to undo.
  - **View balances:** an unknown user or group returns None. A user with nothing owed returns an empty map.
  - **Settle:** nothing is appended. The check and the append happen in one call, so a refused settle can't half-apply.
  - **Fewest payments:** an unknown group returns None. A fully settled group returns an empty list. Nothing is changed in either case, because the result is only a suggestion.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added.
  - All of a group's data is changed only inside `Group` methods, and balances are computed from that data.
  - If concurrency were required later, one lock inside each `Group` around `add_member`, `add_expense`, `settle` and the balance methods would be enough. The service map would need a thread-safe map.
- **Key decisions:**
  1. **Balances are computed from the history and never stored.** *Rejected:* a running "who owes whom" map updated on every expense and settle. It is a second copy of the history, and one missed update makes it wrong forever.
  2. **A split strategy returns shares, and `Expense` checks they add up.** *Rejected:* an `if` on the split type inside the group.
  3. **Money is a whole `int` and leftover units go to the first people in the list.** *Rejected:* decimals. 100 split three ways would give 33.33 three times and lose a cent.
  4. **Every expense belongs to a group, and two friends are a two-person group.** *Rejected:* a separate expense path outside groups, which would double all the balance logic.
  5. **A settlement can't exceed what is owed.** *Rejected:* accepting any amount and letting the balance flip silently.
- **Edge cases:**
  - Exact amounts that do not add up to the total, or percentages that do not add up to 100, are refused.
  - The payer is not a member, or a person in the split is not a member, so the expense is refused.
  - The payer owes nothing because they are not in the shares.
  - Remainder units in an equal or percentage split.
  - A settlement larger than the debt, a settlement with oneself, or a zero amount is refused.
  - A new member joins with a zero balance.
  - A fully settled group has no suggested payments.
- **Extensibility (change → what to add):**
  - A new way of splitting (by shares such as 2:1:1, or by adjustments) → A new class implementing `SplitStrategy`, passed to `add_expense` *(edits existing code: No)*
  - Equal split among only some people, or leaving the payer out → A different list passed to `EqualSplit` *(edits existing code: No. It is data)*
  - More users, groups and members → Calls to `register_user`, `create_group` and `add_member` *(edits existing code: No)*
  - More fields on a user (email, phone) → Add them to `User` *(edits existing code: No other class changes)*
- **Follow-up questions:**
  1. **"Is the fewest-payments result really the minimum?"** It gives at most (people − 1) payments, which is usually the minimum, but the true minimum is a hard problem (it means splitting people into the most groups that each add up to zero). The greedy approach is the practical answer.
  2. **"Balances are slow when there are thousands of expenses. What do you do?"** Keep a running balance inside the `Group`, updated by the same methods that add expenses and transfers, and drop it if it is ever in doubt. It is a deliberate second copy kept for speed, so it is a trade-off that should be said out loud.
  3. **"How would you swap the simplification method?"** Make `DebtSimplifier` an interface at that point, with the current greedy class as the first implementation. It is passed into the service already, so only the declaration changes.
- **Out of scope (one-line answer if asked):**
  - Several currencies → All amounts share one currency. Converting would be a new class applied before the split.
  - Editing or deleting an expense → No requirement asks for it. It would reverse an entry in the history.
  - Descriptions and dates on expenses → No requirement asks for them. They are extra fields on `Expense`.
  - Decimal percentages → Percentages are whole numbers. Finer ones would use a bigger scale such as hundredths of a percent.
  - Removing a member → Not required. A member with a non-zero balance would need to be settled first.
  - Names in balance results → Results use user ids. The caller looks names up.
  - Concurrent users → The Concurrency list above gives the one-lock-per-group answer.
  - Notifications about debts → No requirement asks for them.

#### 27 · Digital Wallet
- **Enums:**
  - **TransactionType**: `TOP_UP`, `TRANSFER`, `MERCHANT_PAYMENT`. *Why:* the three movements in requirements 2 and 3
  - **TransactionStatus**: `PENDING`, `SUCCESS`, `FAILED`, `REFUNDED`
  - **MethodType**: `CARD`, `BANK_ACCOUNT`. *Why:* requirement 2's two sources, and the key the router uses to pick a gateway
  - **GatewayOutcome**: `SUCCESS`, `DECLINED`, `ERROR`
- **Class details (job · what it owns · key methods):**
  - **Wallet**: Holds a balance and guards every change to it, including transfers · *owns:* balance
    - `get_wallet_id() -> str`, `get_owner_id() -> str`, `get_balance() -> int`, `credit(amount: int) -> bool`, `debit(amount: int) -> bool`, `transfer(from_wallet: Wallet, to_wallet: Wallet, amount: int) -> bool`
  - **PaymentMethod**: Describes a source or destination of outside money
    - `method_type() -> MethodType`, `describe() -> str`
  - **GatewayResult**: Records what a gateway answered · *owns:* outcome, reference
    - `get_outcome() -> GatewayOutcome`, `get_reference() -> str or None`
  - **PaymentGateway**: Charges or pays out through an outside provider
    - `charge(method: PaymentMethod, amount: int, idempotency_key: str) -> GatewayResult`, `payout(method: PaymentMethod, amount: int, idempotency_key: str) -> GatewayResult`
  - **GatewayRouter**: Picks the gateway for a payment method · *owns:* the method-to-gateway table
    - `gateway_for(method: PaymentMethod) -> PaymentGateway`
  - **Merchant**: Describes a merchant and where they are paid · *owns:* id, name, settlement account
  - **Transaction**: Records one money movement and its status · *owns:* type, parties, amount, status
    - `involves(wallet_id: str) -> bool`, `succeed(reference: str or None) -> bool`, `fail(note: str) -> bool`, `mark_refunded(note: str) -> bool`
  - **WalletService**: Runs every money movement in the right order, once per key · *owns:* wallets, merchants, transactions, keys
    - `create_wallet(owner_id: str) -> Wallet or None`, `add_merchant(merchant_id: str, name: str, settlement_method: PaymentMethod) -> bool`, `top_up(key: str, wallet_id: str, method: PaymentMethod, amount: int, now: datetime) -> Transaction or None`, `transfer(key: str, from_wallet_id: str, to_wallet_id: str, amount: int, now: datetime) -> Transaction or None`, `pay_merchant(key: str, wallet_id: str, merchant_id: str, amount: int, now: datetime) -> Transaction or None`, `get_balance(wallet_id: str) -> int or None`, `get_history(wallet_id: str) -> list[Transaction] or None`, `claim(key: str, build: a new Transaction) -> (Transaction, bool)`, `call_with_retries(call) -> GatewayResult`
- **Relationships:**
  - WalletService **has** Wallet, Transaction, Merchant (created inside its methods).
  - Merchant **uses** PaymentMethod (passed in).
  - GatewayRouter **uses** PaymentGateway (passed in).
  - WalletService **uses** GatewayRouter (passed in).
  - CardMethod, BankAccountMethod **implement** PaymentMethod. CardGateway, BankGateway **implement** PaymentGateway.
  - Enums are always **uses**: Transaction to TransactionType and TransactionStatus, PaymentMethod to MethodType, GatewayResult to GatewayOutcome, GatewayRouter to MethodType.
  - `GatewayResult` appears only as a return type, so it has no line from the service.
- **Call flows:**
  - **Top up**
    1. `User -> WalletService.top_up(key, wallet, method, amount, now)`
    2. `WalletService -> claim(key, ...)`
    3. `WalletService -> GatewayRouter.gateway_for(method)`, then `PaymentGateway.charge(...)`
    4. On `SUCCESS`
  - **Transfer**
    1. `User -> WalletService.transfer(key, from, to, amount, now)`, then `claim(key, ...)`
    2. `WalletService -> Wallet.transfer(from_wallet, to_wallet, amount)`
    3. `Transaction.succeed`
  - **Pay a merchant**
    1. `User -> WalletService.pay_merchant(...)`, then `claim(key, ...)`
    2. `WalletService -> Wallet.debit(amount)`
    3. `WalletService -> PaymentGateway.payout(merchant method, amount, key)`
    4. On failure
- **Failure and undo (per flow):**
  - **Transfer (requirements 3, 6, 7):** nothing to undo. Both balances change in one locked step or neither does.
- **Concurrency:**
  - **Wallet:** the lock sits inside each `Wallet`, around `credit`, `debit` and balance reads. A check and a change are one step, so two debits can never both pass a balance check meant for one.
  - **Transfer:** `Wallet.transfer` takes both wallets' locks, always in order of wallet id. Opposite transfers between the same two wallets can't deadlock, and no one ever sees money that has left one wallet but not reached the other.
  - **Transaction:** the lock sits inside each `Transaction`, around status changes.
  - **Keys and maps:** `keys` uses `put_if_absent`, so only one request per key goes on. All maps are thread-safe, and ids come from an atomic counter.
  - **No lock during a gateway call:** gateway calls happen with no wallet lock held, so a slow provider never blocks other users' balances. A merchant payment's money is already debited, so it can't be spent twice while the payout runs.
  - **Lock order:** at most two wallet locks, always in id order, and never together with a transaction lock. So no deadlock is possible.
- **Key decisions:**
  1. **Balances change only inside `Wallet`, under its lock.** *Rejected:* the service reading a balance and then setting a new one, which loses updates under concurrency.
  2. **A transfer locks both wallets in order of id.** *Rejected:* a debit then a separate credit, where a reader could see the money missing for a moment, and a failure in between needs fixing up.
  3. **The idempotency key is claimed before anything else.** *Rejected:* checking for a key after processing, which lets two copies both move money.
  4. **Money held first, payout second, and a credit back if the payout fails.** *Rejected:* paying out first and debiting after, which can pay a merchant from an empty wallet.
  5. **The gateway for each method type is a table.** *Rejected:* an `if` on the provider inside the service.
- **Edge cases:**
  - A transfer to the same wallet, or of 0 or less, is refused.
  - A transfer or payment larger than the balance fails and is recorded as failed.
  - A declined card is not retried. A gateway error is retried a set number of times.
  - A merchant payout that fails every attempt is paid back, and the transaction shows `REFUNDED`.
  - A repeated key returns the first transaction, even if it failed.
  - Opposite transfers between two wallets at the same moment don't deadlock.
  - A user can't open a second wallet.
  - A card or account is kept as a provider token, not as a real number.
- **Extensibility (change → what to add):**
  - A new payment provider for cards → A new class implementing `PaymentGateway`, set in the router's table *(edits existing code: No)*
  - A new payment method (UPI) → A new `MethodType` value, a new `PaymentMethod` class, and a router entry *(edits existing code: No logic changes)*
  - More retry attempts → A different `max_gateway_attempts` *(edits existing code: No. It is data)*
  - New merchants → Calls to `add_merchant` *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"What if the gateway charged the card but its answer was lost?"** The retry uses the same key, so the provider returns the original charge instead of charging again. A daily reconciliation against the provider's records catches anything left over.
  2. **"How would you scale this across many servers?"** Balances move to a database, the wallet lock becomes a row lock (or a version check), and the key map becomes a table with a unique key column.
  3. **"Why keep a balance at all instead of adding up the history?"** Adding up the history on every payment is slow. A real system keeps a double-entry ledger as the record and the balance as a checked summary of it.
- **Out of scope (one-line answer if asked):**
  - Waiting between retries → Retries happen straight away. A delay would be added in `call_with_retries`.
  - Reconciling with providers → See follow-up question 1.
  - Withdrawing money to a bank → Not required. It would be a payout from the user's own wallet.
  - A full double-entry ledger → See follow-up question 3.
  - Limits, fees and fraud checks → Not required. They would be checks before the debit.
  - Expiring old idempotency keys → Keys are kept forever here. A real system would drop them after a set time.


---

## Type E: Queue + Workers (async systems)

#### 13 · Notification Service
- **Enums:**
  - **NotificationType**: `TRANSACTIONAL`, `PROMOTIONAL`, `ALERT`
  - **Priority**: `LOW`, `MEDIUM`, `HIGH`, with ranks 1, 2 and 3
  - **ChannelType**: `EMAIL`, `SMS`, `PUSH`
  - **DeliveryStatus**: `PENDING`, `RETRYING`, `DELIVERED`, `FAILED`
- **Class details (job · what it owns · key methods):**
  - **RenderedContent**: Holds a finished subject and body · *owns:* subject, body
    - `get_subject() -> str`, `get_body() -> str`
  - **Template**: Turns values into a finished message · *owns:* the subject and body patterns
    - `get_name() -> str`, `render(values: map{str: str}) -> RenderedContent or None`
  - **RetryPolicy**: Says how long to wait before each retry, and when to give up · *owns:* max attempts, base delay
    - `next_delay(attempts_made: int) -> int or None`
  - **User**: Holds a person's contacts and choices · *owns:* contacts, enabled channels, enabled types
    - `get_user_id() -> str`, `set_contact(channel_type: ChannelType, address: str) -> bool`, `set_channel_enabled(channel_type: ChannelType, enabled: bool) -> None`, `set_type_enabled(notification_type: NotificationType, enabled: bool) -> None`, `wants(notification_type: NotificationType) -> bool`, `get_deliverable_channels() -> list[ChannelType]`, `get_address(channel_type: ChannelType) -> str or None`
  - **NotificationChannel**: Delivers one message through one kind of channel
    - `send(address: str, content: RenderedContent) -> bool`
  - **EmailChannel**: Sends email
    - `send(address: str, content: RenderedContent) -> bool`
  - **SmsChannel**: Sends SMS
    - `send(address: str, content: RenderedContent) -> bool`
  - **PushChannel**: Sends push notifications
    - `send(address: str, content: RenderedContent) -> bool`
  - **ChannelRegistry**: Finds the channel for a channel type · *owns:* the channels
    - `register(channel_type: ChannelType, channel: NotificationChannel) -> bool`, `supports(channel_type: ChannelType) -> bool`, `get(channel_type: ChannelType) -> NotificationChannel or None`
  - **Delivery**: Tracks one notification on one channel · *owns:* status, attempts, next attempt time
    - `get_delivery_id() -> str`, `get_notification_id() -> str`, `get_priority() -> Priority`, `get_channel_type() -> ChannelType`, `get_address() -> str`, `get_content() -> RenderedContent`, `get_status() -> DeliveryStatus`, `get_attempts() -> int`, `get_next_attempt_at() -> datetime or None`, `record_attempt(success: bool, now: datetime, policy: RetryPolicy) -> DeliveryStatus or None`
  - **Notification**: Holds one message for one user and its deliveries · *owns:* the message, its deliveries
    - `get_notification_id() -> str`, `get_user_id() -> str`, `get_priority() -> Priority`, `get_deliveries() -> list[Delivery]`, `get_statuses() -> map{ChannelType: DeliveryStatus}`
  - **DeliveryQueue**: Hands out ready deliveries, highest priority first · *owns:* the waiting jobs
    - `put(delivery: Delivery, now: datetime) -> None`, `take_ready(now: datetime) -> Delivery or None`, `size() -> int`
  - **DeliveryWorker**: Takes one job, delivers it and records the result
    - `process_next(now: datetime) -> bool`
  - **Event**: Describes something that happened and who to tell · *owns:* an unchangeable record
    - `get_event_name() -> str`, `get_notification_type() -> NotificationType`, `get_priority() -> Priority`, `get_template_name() -> str`, `get_values() -> map{str: str}`, `get_recipient_ids() -> list[str]`, `get_occurred_at() -> datetime`
  - **EventListener**: Reacts to an event
    - `on_event(event: Event) -> None`
  - **NotificationListener**: Turns an event into notifications
    - `on_event(event: Event) -> None`
  - **EventPublisher**: Tells every listener about an event · *owns:* the listeners
    - `subscribe(listener: EventListener) -> bool`, `unsubscribe(listener: EventListener) -> bool`, `publish(event: Event) -> int`
  - **NotificationService**: Builds and queues notifications, and holds users, templates and notifications · *owns:* the users, templates, notifications
    - `register_user(user_id: str, name: str) -> User or None`, `set_contact(user_id: str, channel_type: ChannelType, address: str) -> bool`, `set_channel_enabled(user_id: str, channel_type: ChannelType, enabled: bool) -> bool`, `set_type_enabled(user_id: str, notification_type: NotificationType, enabled: bool) -> bool`, `add_template(name: str, subject_pattern: str, body_pattern: str) -> bool`, `send(user_id: str, notification_type: NotificationType, priority: Priority, template_name: str, values: map{str: str}, now: datetime) -> Notification or None`, `get_statuses(notification_id: str) -> map{ChannelType: DeliveryStatus} or None`
- **Relationships:**
  - NotificationService **has** User, Template, Notification (created inside the register, add and send methods).
  - Notification **has** Delivery (deliveries die with the notification).
  - DeliveryQueue **uses** Delivery (the jobs it holds are created elsewhere and live on).
  - ChannelRegistry **uses** NotificationChannel (passed in).
  - NotificationService **uses** ChannelRegistry, DeliveryQueue (passed in).
  - DeliveryWorker **uses** DeliveryQueue, ChannelRegistry, RetryPolicy (passed in).
  - EventPublisher **uses** EventListener (subscribed in).
  - NotificationListener **uses** NotificationService (passed in).
  - Delivery **uses** RenderedContent. Notification **uses** RenderedContent.
  - EmailChannel, SmsChannel, PushChannel **implement** NotificationChannel. NotificationListener **implements** EventListener.
  - Enums are always **uses**: User to ChannelType and NotificationType, Delivery to ChannelType, DeliveryStatus and Priority, Notification to NotificationType and Priority, Event to NotificationType and Priority, ChannelRegistry to ChannelType.
  - `Event` appears only as a method input, so no attribute holds it and it has no line.
- **Call flows:**
  - **Send (fast half)**
    1. `Caller -> NotificationService.send(...)`
    2. `NotificationService -> User.wants(type)`, then `User.get_deliverable_channels()`
    3. `NotificationService -> Template.render(values)`
    4. `NotificationService -> ChannelRegistry.supports(channel_type)`
    5. `NotificationService -> DeliveryQueue.put(delivery, now)`
  - **Deliver (slow half)**
    1. `Worker thread -> DeliveryWorker.process_next(now)`
    2. `DeliveryWorker -> DeliveryQueue.take_ready(now)`
    3. `DeliveryWorker -> ChannelRegistry.get(channel_type)`, then `NotificationChannel.send(address, content)`
    4. `DeliveryWorker -> Delivery.record_attempt(success, now, policy)`
  - **Event**
    1. `Event source -> EventPublisher.publish(event)`
    2. `EventPublisher -> EventListener.on_event(event)`
    3. `NotificationListener -> NotificationService.send(...)`
- **Failure and undo (per flow):**
  - **Send (fast half):** every check happens before the first queue entry, so a refusal queues nothing and there is nothing to undo.
  - **Deliver (slow half):** a failed send changes only the delivery's own record. `FAILED` after the last attempt is a final state, and it stays visible through tracking.
  - **Track status:** an unknown id returns None.
  - **Publish an event:** a failing listener is skipped and the others still run.
- **Concurrency:**
  - **Queue:** the lock sits inside `DeliveryQueue`, around `put` and `take_ready`. Many senders and many workers can use it at once, and a job is handed to exactly one worker.
  - **Delivery:** the lock sits inside `Delivery`, around `record_attempt`, so a status changes one step at a time.
  - **User:** the lock sits inside `User`, around reading and changing choices, so a send never sees a half-changed set.
  - **Maps and ids:** the three maps are thread-safe maps, and ids come from an atomic counter.
  - **Publisher:** the lock sits inside `EventPublisher`, around its list only, and is released before any listener runs.
  - **Not blocking the caller:** `send` only does quick work and puts jobs on the queue. The calls to the real channels happen only in `process_next`, on worker threads.
  - **Lock order:** no method ever holds two locks at once. The worker releases the queue lock before it takes a delivery lock, so no deadlock is possible.
- **Key decisions:**
  1. **A `Delivery` per channel is the unit of retry and status.** *Rejected:* one status per notification. A retry would resend on channels that already worked.
  2. **A notification's overall status is computed from its deliveries.** *Rejected:* a stored status on the notification, which would disagree after one delivery changed.
  3. **`send` queues and returns, and a worker delivers.** *Rejected:* sending inside `send`, which would block the caller on the slowest gateway.
  4. **The queue orders by priority, and holds retries until their time.** *Rejected:* a plain first-in first-out list with a sleep before each retry.
  5. **A user's choices live in `User`, with usable channels computed.** *Rejected:* the service checking preferences held in several places.
- **Edge cases:**
  - A user who opted out of a type gets nothing, and `send` returns None.
  - A channel with no address, or no registered class, is skipped.
  - A template placeholder with no value refuses the send before anything is queued.
  - A retry waits, with the wait doubling, and the last failure ends in `FAILED`.
  - A high-priority job is taken before older low-priority ones.
  - A listener that fails does not stop the others.
  - A channel that throws an error counts as a failed attempt.
  - An unknown user, template or notification id returns None.
- **Extensibility (change → what to add):**
  - A new channel (WhatsApp) → One new `ChannelType` value, a new class implementing `NotificationChannel`, and one `register` call *(edits existing code: No logic changes. It is an enum value, a class and data)*
  - A new notification type → One new `NotificationType` value *(edits existing code: No)*
  - A new template or new wording → A new `add_template` call *(edits existing code: No. It is data)*
  - Different retry timing → A different `RetryPolicy` passed to the worker *(edits existing code: No. It is data)*
  - Another party interested in events (audit, analytics) → A new class implementing `EventListener`, then one `subscribe` call *(edits existing code: No)*
- **Follow-up questions:**
  1. **"Can a notification be delivered twice?"** Yes, if a channel accepted a message but the answer was lost, the worker retries it. The fix is an idempotency key sent with each delivery so the channel can ignore a repeat.
  2. **"What if the process crashes with jobs on the queue?"** The queue is in memory, so those jobs are lost. A durable queue (a database or message broker) would replace `DeliveryQueue`, and `DeliveryWorker` would not change.
  3. **"How would you stop a user being flooded?"** Add a limit per user and channel, checked in `send` before queuing, or a digest that groups promotional notifications. It is a new rule, because nothing in the requirements limits volume.
- **Out of scope (one-line answer if asked):**
  - Real email, SMS and push services → Only the three `send` methods touch the outside world.
  - A saved queue that survives a crash → See follow-up question 2.
  - Sending at a later time → Not required. It would be a `next_attempt_at` on the first delivery.
  - Alerts that ignore opt-out → A user's choice always wins. A rule such as "ALERT can't be turned off" would go into `User.wants`.
  - Duplicates after a lost answer → See follow-up question 1.
  - Limits on how many a user receives → See follow-up question 3.
  - Starting the worker threads → Infrastructure outside the design. They only need to call `process_next` in a loop.
  - Different wording per language → A template per language, chosen by the caller.

#### 17 · Task Scheduler
- **Enums:**
  - **JobStatus**: `SCHEDULED`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`
  - **Priority**: `LOW`, `MEDIUM`, `HIGH`, with ranks 1, 2 and 3. Method: `get_rank() -> int`
- **Class details (job · what it owns · key methods):**
  - **RetryPolicy**: Says how long to wait before each retry, and when to give up · *owns:* max attempts, base delay
    - `next_delay(attempts_made: int) -> int or None`
  - **Schedule**: Says when a job first runs and when it runs next
    - `first_run() -> datetime`, `next_run_after(planned: datetime) -> datetime or None`
  - **OneTimeSchedule**: Runs once at one time · *owns:* the time
    - `first_run()`, `next_run_after(...)`
  - **FixedIntervalSchedule**: Runs at a start time and then every interval · *owns:* start, interval
    - `first_run()`, `next_run_after(planned)`
  - **Task**: Does one piece of work
    - `execute() -> bool`, `get_name() -> str`
  - **PrintTask**: Prints a message · *owns:* the message
    - `execute()`, `get_name()`
  - **FlakyTask**: Fails a set number of times, then succeeds · *owns:* the failures left
    - `execute()`, `get_name()`
  - **Clock**: Says what time it is
    - `now() -> datetime`
  - **QueueEntry**: Is one ticket in the queue for one run of a job · *owns:* an unchangeable record
  - **JobQueue**: Hands out due entries, highest priority first · *owns:* the waiting entries
    - `add(job_id: str, version: int, run_at: datetime, priority: Priority) -> None`, `take_ready(now: datetime) -> QueueEntry or None`, `size() -> int`
  - **ScheduledJob**: Holds one job's timing and stage, and guards every change · *owns:* stage, next run, version
    - `get_job_id() -> str`, `get_status() -> JobStatus`, `get_priority() -> Priority`, `get_next_try_at() -> datetime`, `get_version() -> int`, `get_task() -> Task`, `start_run(version: int) -> bool`, `finish_run(success: bool, now: datetime, policy: RetryPolicy) -> bool`, `cancel() -> bool`, `update(schedule: Schedule or None, priority: Priority or None) -> bool`
  - **Scheduler**: Adds, updates and cancels jobs, and runs the next due one · *owns:* the jobs
    - `schedule(task: Task, schedule: Schedule, priority: Priority) -> str`, `cancel(job_id: str) -> bool`, `update(job_id: str, schedule: Schedule or None, priority: Priority or None) -> bool`, `get_status(job_id: str) -> JobStatus or None`, `run_next(now: datetime) -> bool`
  - **WorkerPool**: Runs several worker threads that keep asking the scheduler for work · *owns:* the threads
    - `start() -> bool`, `shutdown() -> None`
- **Relationships:**
  - Scheduler **has** ScheduledJob (created inside `schedule`).
  - JobQueue **has** QueueEntry (created inside `add`).
  - ScheduledJob **uses** Task, Schedule (passed in, and they live on).
  - Scheduler **uses** JobQueue, RetryPolicy (passed in).
  - WorkerPool **uses** Scheduler, Clock (passed in).
  - OneTimeSchedule, FixedIntervalSchedule **implement** Schedule. PrintTask, FlakyTask **implement** Task. SystemClock **implements** Clock.
  - Enums are always **uses**: ScheduledJob to JobStatus and Priority, QueueEntry to Priority.
- **Call flows:**
  - **Schedule a job**
    1. `Caller -> Scheduler.schedule(task, schedule, priority)`
    2. `Scheduler -> ScheduledJob(...)`, then `jobs.put`, then `JobQueue.add(job_id, version, first run, priority)`
  - **A worker runs the next job**
    1. `Worker thread -> Scheduler.run_next(clock.now())`
    2. `Scheduler -> JobQueue.take_ready(now)`
    3. `Scheduler -> ScheduledJob.start_run(entry version)`
    4. `Scheduler -> Task.execute()`
    5. `Scheduler -> ScheduledJob.finish_run(success, now, policy)`
  - **Cancel or update**
    1. `Caller -> Scheduler.cancel(job_id)`
    2. On an update, `Scheduler -> JobQueue.add(...)` with the new version
- **Failure and undo (per flow):**
  - **Schedule a job:** invalid inputs are refused when the schedule or the job is built, so nothing is stored or queued.
  - **Run the next job (one worker step):** a failed task changes only its own job record, and the retry or next run is queued with a new version. A stale entry is dropped without touching anything.
  - **Cancel:** a running, completed, failed or already cancelled job is refused.
  - **Update:** a job that is not `SCHEDULED` is refused, with no change.
- **Concurrency:**
  - **Queue:** the lock sits inside `JobQueue`, around `add` and `take_ready`. Callers can add jobs while workers take them, and each entry is handed to exactly one worker.
  - **Job:** the lock sits inside `ScheduledJob`, around every change. `start_run` checks the version and the stage and changes the stage as one step, so a job can never run twice at once.
  - **Map and ids:** `jobs` is a thread-safe map, and ids come from an atomic counter.
  - **No lock while working:** a task runs with no lock held.
  - **Lock order:** no method ever holds two locks at once. The scheduler finishes with the queue before it touches a job, so no deadlock is possible.
  - **Why stale entries are skipped instead of removed:** removing an entry from the middle of a priority queue is slow and needs extra locking. A version check is one comparison.
- **Key decisions:**
  1. **Version numbers instead of removing entries from the queue.** *Rejected:* searching the queue to remove the old entry on every cancel or update.
  2. **`start_run` checks the version and the stage as one step.** *Rejected:* checking the stage in the scheduler and then setting it, which two workers could both pass.
  3. **A repeating job's next run is counted from the planned time.** *Rejected:* `finish time + interval`.
  4. **A task runs with no lock held.** *Rejected:* holding the job lock during the whole run, which would make `cancel` and `get_status` wait.
  5. **Due jobs are ordered by priority first, then by time.** *Rejected:* time first, which would let a pile of overdue low-priority jobs delay an urgent one.
- **Edge cases:**
  - A cancelled or updated job's old entry is skipped.
  - A job can't be cancelled or updated while it runs.
  - A failed run retries with a doubling delay, and the last failure ends a one-time job as `FAILED`.
  - A repeating job whose run keeps failing moves on to its next planned run.
  - A task that throws an error counts as a failure and doesn't kill the worker.
  - A repeating job can't overlap itself, because it is `RUNNING` until its run ends.
  - A worker that falls behind runs each missed repeating run once and then catches up to the schedule.
  - Jobs added while workers are busy are queued and picked up when due.
- **Extensibility (change → what to add):**
  - A new kind of task (send a report, clean files) → A new class implementing `Task`, passed to `schedule` *(edits existing code: No)*
  - A new kind of timing (cron expression, weekdays only) → A new class implementing `Schedule` *(edits existing code: No)*
  - More or fewer workers → A different `size` for `WorkerPool` *(edits existing code: No. It is data)*
  - Different retry timing → A different `RetryPolicy` *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"Why do idle workers wait and check again instead of being woken?"** It is simple and correct. A better version uses a condition variable that `add` signals, and a wait until the next `run_at`, so no worker checks too often.
  2. **"What if the process crashes?"** Jobs are in memory and are lost. A durable store would keep jobs and entries, and `Scheduler` would reload them on start.
  3. **"How would you stop a slow task from running forever?"** Give each run a time limit and treat a timeout as a failure, so it goes through the same retry path.
- **Out of scope (one-line answer if asked):**
  - Cancelling a task while it runs → A running task finishes its current run. Stopping it mid-run needs the task to support it.
  - Surviving a crash → See follow-up question 2.
  - Time limits on a run → See follow-up question 3.
  - Cron expressions → A new `Schedule` class.
  - Jobs that depend on other jobs → No requirement asks for it.
  - A history of every run → `get_status` shows the current stage. A run log would be a list kept by the scheduler.
  - Waking workers instead of polling → See follow-up question 1.

#### 19 · Bounded Queue + Pub-Sub
- **Class details (job · what it owns · key methods):**
  - **BoundedBlockingQueue**: Holds up to a fixed number of items, in order, making producers and consumers wait when needed · *owns:* the items, closed flag
    - `put(item: T, timeout_ms: int or None) -> bool`, `take(timeout_ms: int or None) -> T or None`, `close() -> None`, `size() -> int`, `is_closed() -> bool`
  - **Message**: Records one published message · *owns:* an unchangeable record
    - `get_topic() -> str`, `get_offset() -> int`, `get_payload() -> str`
  - **Topic**: Keeps one topic's ordered log and each subscriber's position in it · *owns:* the log, the positions
    - `publish(payload: str) -> int or None`, `subscribe(subscriber_id: str) -> bool`, `unsubscribe(subscriber_id: str) -> bool`, `read(subscriber_id: str, timeout_ms: int or None) -> Message or None`, `close() -> None`, `trim() -> None`
  - **Broker**: Creates topics and routes publish, subscribe, poll and shutdown to them · *owns:* the topics
    - `create_topic(name: str) -> bool`, `publish(topic_name: str, payload: str) -> int or None`, `subscribe(topic_name: str, subscriber_id: str) -> bool`, `unsubscribe(topic_name: str, subscriber_id: str) -> bool`, `poll(topic_name: str, subscriber_id: str, timeout_ms: int or None) -> Message or None`, `shutdown() -> None`
- **Relationships:**
  - Broker **has** Topic (created inside `create_topic`).
  - Topic **has** Message (created inside `publish`, and dropped by trimming).
  - BoundedBlockingQueue stands alone. It holds items of type `T`, not other classes of this design.
  - There are no enums, so there are no enum lines.
- **Call flows:**
  - **Queue**
    1. `Producer -> BoundedBlockingQueue.put(item, timeout)`
    2. `Consumer -> BoundedBlockingQueue.take(timeout)`
  - **Publish**
    1. `Publisher -> Broker.publish(topic_name, payload)`
    2. `Broker -> Topic.publish(payload)`
  - **Subscribe and poll**
    1. `Subscriber -> Broker.subscribe(topic_name, subscriber_id)`, then `Topic.subscribe(...)`
    2. `Subscriber -> Broker.poll(topic_name, subscriber_id, timeout)`, then `Topic.read(...)`
  - **Shutdown**
    1. `Owner -> Broker.shutdown()`, then `Topic.close()`
- **Failure and undo (per flow):**
  - **Put (producer):** a timeout or a shutdown returns False and the item was never added, so nothing is half-done.
- **Concurrency:**
  - **Queue:** the lock sits inside `BoundedBlockingQueue`, around every method. Two conditions on that lock make producers and consumers wait separately.
  - **Topic:** the lock sits inside each `Topic`, around its log and positions. Different topics never wait for each other.
  - **Broker:** the lock sits inside `Broker`, around creating topics and shutdown only. It is released before a topic is touched. `topics` is a thread-safe map.
  - **Lock order:** no method ever holds two locks at once, apart from `shutdown`, which always takes the broker lock and then one topic lock at a time, and no method takes them in the other order. So no deadlock is possible.
  - **Spurious wake-ups:** every wait is inside a `while` that re-checks the reason, so a thread never acts on a change that is not there.
- **Key decisions:**
  1. **One lock and two conditions for the queue.** *Rejected:* busy-waiting in a loop, which burns the processor, and one condition for both sides.
  2. **A topic is one log, and each subscriber is one position in it.** *Rejected:* a bounded queue per subscriber. A full one would either block the publisher (breaking requirement 9) or drop messages (breaking requirement 8).
  3. **The log is trimmed to the slowest position.** *Rejected:* keeping every message forever.
  4. **Offsets are given out under the topic's lock.** *Rejected:* giving out offsets outside the lock, which could put messages in two different orders.
  5. **Shutdown closes and wakes, and lets readers drain.** *Rejected:* clearing everything at once, which would lose accepted items.
- **Edge cases:**
  - A full queue with a timeout returns False when the time runs out.
  - An empty queue with a timeout returns None when the time runs out.
  - A wake with nothing changed is re-checked.
  - A subscriber that joins late gets only messages published after it joined.
  - A slow subscriber keeps old messages in memory but never slows the publisher or other subscribers.
  - Unsubscribing frees messages only that subscriber still needed.
  - Publishing to a topic with no subscribers stores nothing.
  - Polling an unknown topic or as a non-subscriber returns None.
  - After shutdown, puts and publishes are refused, waiting threads are woken, and what was already accepted can still be read.
- **Extensibility (change → what to add):**
  - More topics → Calls to `create_topic` *(edits existing code: No. It is data)*
  - A bigger or smaller queue → A different `capacity` *(edits existing code: No. It is data)*
  - A subscriber that starts from the oldest message still kept → A `subscribe` variant that sets the position to `base_offset` *(edits existing code: One new method on `Topic`)*
  - Another kind of item in the queue → Use the queue with a different `T` *(edits existing code: No)*
- **Follow-up questions:**
  1. **"What if one subscriber never reads?"** The log grows from its position. The usual fix is a limit on how far behind a subscriber may fall, after which it is removed (or skipped forward) and told it lost messages. It is a trade-off between requirement 8 and memory.
  2. **"Why `notify_all` and not `notify`?"** It is the safe choice with timeouts and shutdown. It can wake more threads than needed, and each re-checks its reason. `notify` is faster but easy to get wrong.
  3. **"How would you deliver to subscribers by callback instead of polling?"** Run one thread per subscriber that polls and calls the callback. The topic stays the same, and a slow callback still affects only its own subscriber.
- **Out of scope (one-line answer if asked):**
  - A limit on how far behind a subscriber may fall → See follow-up question 1.
  - Saving messages to disk → Everything is in memory. A durable log would replace `Topic.log`.
  - Delivery by callback → See follow-up question 3.
  - Several readers sharing one subscription (a consumer group) → Each subscriber id has its own position. A shared position would need its own lock.
  - Deleting topics → Not required. It would close the topic and remove it from the map.
  - Message keys or filtering → Not required.

#### 23 · Web Crawler
- **Enums:**
  - **FetchStatus**: `OK`, `NOT_FOUND`, `ERROR`, `TIMEOUT`
- **Class details (job · what it owns · key methods):**
  - **UrlNormalizer**: Turns any URL into one standard form and finds its domain
    - `normalize(url: str, base: str or None) -> str or None`, `domain_of(url: str) -> str or None`, `path_of(url: str) -> str`
  - **CrawlTask**: Records one URL waiting to be fetched · *owns:* url, depth, attempts, earliest time
  - **FetchResult**: Records the outcome of one fetch · *owns:* status, content
    - `get_status() -> FetchStatus`, `get_content() -> str or None`
  - **Page**: Holds one fetched page for the processors · *owns:* url, depth, content
    - `get_url() -> str`, `get_depth() -> int`, `get_content() -> str`
  - **PageFetcher**: Fetches a URL with a time limit
    - `fetch(url: str, timeout_ms: int) -> FetchResult`
  - **LinkExtractor**: Finds the links in a page
    - `extract(page_url: str, content: str) -> list[str]`
  - **RobotsRules**: Holds one site's rules and checks a path · *owns:* disallowed paths, delay
    - `parse(text: str or None) -> RobotsRules`, `allows(path: str) -> bool`, `get_crawl_delay_ms() -> int or None`
  - **RobotsCache**: Fetches each site's rules once and keeps them · *owns:* the rules per domain
    - `get(domain: str) -> RobotsRules`
  - **CrawlPolicy**: Says how deep to go and which domains are allowed · *owns:* depth limit, allowed domains
    - `allows_domain(domain: str) -> bool`, `allows_following(depth: int) -> bool`
  - **RetryPolicy**: Says how long to wait before each retry, and when to give up · *owns:* max attempts, base delay
    - `next_delay(attempts_made: int) -> int or None`
  - **ContentProcessor**: Does something with a fetched page
    - `process(page: Page) -> None`
  - **ArchiveProcessor**: Keeps a copy of every fetched page · *owns:* the pages kept
  - **Frontier**: Hands out the next page to fetch, one site at a time, respecting each site's wait · *owns:* the waiting tasks, each site's next allowed time
    - `add(domain: str, task: CrawlTask) -> None`, `set_domain_delay(domain: str, delay_ms: int) -> None`, `take(now_ms: int) -> CrawlTask or None`, `done() -> None`, `is_finished() -> bool`
  - **Crawler**: Runs one crawl step and holds the seen set · *owns:* the seen URLs, failed URLs
    - `add_seeds(urls: list[str]) -> int`, `process_next(now_ms: int) -> bool`, `is_finished() -> bool`, `get_failed() -> list[str]`, `enqueue(url: str, depth: int) -> bool`
  - **CrawlerWorkerPool**: Runs several worker threads until the crawl is done · *owns:* the threads
    - `start() -> bool`, `stop() -> None`, `await_finished() -> None`
- **Relationships:**
  - Frontier **has** CrawlTask (tasks wait in its queues until taken; the crawler creates them).
  - RobotsCache **has** RobotsRules (created inside `get`).
  - Crawler **uses** Frontier, PageFetcher, LinkExtractor, UrlNormalizer, RobotsCache, CrawlPolicy, RetryPolicy, ContentProcessor (all passed in).
  - LinkExtractor **uses** UrlNormalizer. RobotsCache **uses** PageFetcher.
  - CrawlerWorkerPool **uses** Crawler, Clock.
  - HttpFetcher **implements** PageFetcher. ArchiveProcessor, WordCountProcessor **implement** ContentProcessor.
  - Enums are always **uses**: FetchResult to FetchStatus.
  - `Page` and `FetchResult` appear only as method inputs and outputs, so no attribute holds them and they have no line.
- **Call flows:**
  - **Seed**
    1. `Owner -> Crawler.add_seeds(urls)`
  - **One crawl step**
    1. `Worker thread -> Crawler.process_next(now)`
    2. `Crawler -> Frontier.take(now)`
    3. `Crawler -> RobotsCache.get(domain)`, then `RobotsRules.allows(path)`, and `Frontier.set_domain_delay`
    4. `Crawler -> PageFetcher.fetch(url, timeout)`
    5. On success
    6. On a failure worth retrying
    7. `Crawler -> Frontier.done()`
- **Failure and undo (per flow):**
  - **Add seeds:** an invalid URL, a domain not allowed, or a repeat of another seed is skipped.
  - **One crawl step:** nothing to undo. A failed fetch changes only the frontier (a retry) or the failed list.
- **Concurrency:**
  - **Seen set:** `seen` is a thread-safe set, and `add_if_absent` claims a URL in one step. Two workers finding the same link at once can't both queue it.
  - **Frontier:** the lock sits inside `Frontier`, around every method. `take` reserves the site's next allowed time while holding the lock, so two workers can't hit the same site too soon.
  - **Robots cache:** the map is thread-safe, and `put_if_absent` keeps one set of rules per site even if two workers fetch at once.
  - **Processors:** each processor guards its own data with its own lock, because pages arrive from many threads.
  - **Crawler:** the lock sits inside `Crawler`, around `failed` only.
  - **No lock while fetching:** the network call happens with no lock held, so a slow site never blocks other workers.
  - **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
- **Key decisions:**
  1. **A URL is cleaned and claimed in `seen` before it is queued, not when it is fetched.** *Rejected:* checking when fetching. The same URL could sit in the queue many times.
  2. **The frontier groups tasks by site and reserves each site's next time when it hands out a task.** *Rejected:* each worker sleeping before a request, which can still hit a site twice when two workers wake together.
  3. **A site's rules are fetched once and cached.** *Rejected:* fetching the rules before each page.
  4. **Processors are passed in as a list.** *Rejected:* indexing or saving code inside the crawler.
  5. **The end of the crawl counts tasks in flight.** *Rejected:* stopping as soon as the queue is empty.
- **Edge cases:**
  - Two spellings of one URL (`https://A.com/#top` and `https://a.com/`) count as one.
  - A link back to a page already seen is ignored.
  - A disallowed path is never fetched.
  - A domain not on the list is never queued.
  - A page at the depth limit is fetched but its links are not followed.
  - A slow page times out and is retried with a growing wait, and given up after the last try.
  - A missing page is not retried.
  - A processor that fails doesn't stop the other processors or the crawl.
  - A site's `robots.txt` that can't be fetched means "allow everything".
- **Extensibility (change → what to add):**
  - A new use of pages (search index, link graph, save to disk) → A new class implementing `ContentProcessor`, added to the list *(edits existing code: No)*
  - A different fetcher (a headless browser) → A new class implementing `PageFetcher` *(edits existing code: No)*
  - Different depth, domains, delays or retries → Different numbers and sets in the policies and the frontier *(edits existing code: No. It is data)*
  - More threads → A different `size` for the pool *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"How would you crawl billions of pages?"** Split the work across machines by domain, so each site is handled by one machine and politeness stays local. The seen set moves to a shared store or a Bloom filter, and the frontier to a durable queue.
  2. **"Why not one queue for all sites?"** A big site at the front would block every worker on that site's delay. Grouping by site lets workers move on to sites that are ready.
  3. **"How would you avoid crawler traps (endless calendar pages)?"** The depth limit already caps them. More defences are a limit on pages per site and on URL length, both new rules in `enqueue`.
- **Out of scope (one-line answer if asked):**
  - Crawling across many machines → See follow-up question 1.
  - Saving the crawl to resume after a crash → The frontier and the seen set are in memory. Durable versions would replace them.
  - Pages built by scripts → A browser-based `PageFetcher`.
  - Recrawling pages that change → Not required. It would need a time per URL in `seen`.
  - Robots rules for named crawlers, and rules that expire → The rules for all crawlers are read once.
  - Limits on pages per site → See follow-up question 3.

#### 26 · Online Code Judge
- **Enums:**
  - **Difficulty**: `EASY`, `MEDIUM`, `HARD`. *Why:* requirement 1
  - **Verdict**: `ACCEPTED`, `WRONG_ANSWER`, `TIME_LIMIT_EXCEEDED`, `MEMORY_LIMIT_EXCEEDED`, `RUNTIME_ERROR`, `COMPILE_ERROR`
  - **SubmissionStatus**: `QUEUED`, `RUNNING`, `JUDGED`
- **Class details (job · what it owns · key methods):**
  - **TestCase**: Holds one input and its expected output · *owns:* input, output, sample flag
  - **Problem**: Holds a problem and its limits · *owns:* statement, difficulty, tags, tests, limits
    - `get_sample_cases() -> list[TestCase]`, `has_tag(tag: str) -> bool`
  - **ProblemCatalog**: Stores problems and filters them for browsing · *owns:* the problems
    - `add_problem(problem: Problem) -> bool`, `get_problem(problem_id: str) -> Problem or None`, `list_problems(difficulty: Difficulty or None, tag: str or None) -> list[Problem]`
  - **LanguageSpec**: Describes how to build and run one language · *owns:* file name, commands
    - `needs_compile() -> bool`
  - **ExecResult**: Records what happened when a command ran · *owns:* exit code, output, time, memory, limit flags
  - **Sandbox**: Runs commands sealed off from everything else
    - `write_file(name: str, content: str) -> bool`, `execute(command: str, input_text: str, time_limit_ms: int, memory_limit_mb: int) -> ExecResult`, `destroy() -> None`, `create() -> Sandbox`
  - **JudgeResult**: Records the outcome of judging · *owns:* verdict, tests passed, time, memory, message
  - **Judge**: Compiles and runs one submission in its own sandbox
    - `evaluate(source: str, language: LanguageSpec, problem: Problem) -> JudgeResult`
  - **Submission**: Holds one submission and its progress · *owns:* status, result
    - `get_verdict() -> Verdict or None`, `start() -> bool`, `finish(result: JudgeResult) -> bool`
  - **SubmissionQueue**: Hands out waiting submissions, oldest first · *owns:* the waiting ids
    - `put(submission_id: str) -> bool`, `take(timeout_ms: int) -> str or None`, `close() -> None`
  - **LeaderboardRow**: Gives one line of a leaderboard · *owns:* an unchangeable record
  - **Contest**: Holds a contest's problems, points, times and participants, and ranks them · *owns:* times, points, participants
    - `register(user_id: str) -> bool`, `accepts(user_id: str, problem_id: str, at: datetime) -> bool`, `build_leaderboard(submissions: list[Submission]) -> list[LeaderboardRow]`
  - **JudgeService**: Takes submissions, runs a worker step, and answers history and leaderboard queries · *owns:* languages, submissions, contests
    - `add_language(spec: LanguageSpec) -> bool`, `submit(user_id: str, problem_id: str, language_id: str, source: str, contest_id: str or None, now: datetime) -> str or None`, `process_next(timeout_ms: int) -> bool`, `get_submission(submission_id: str, user_id: str) -> Submission or None`, `get_history(user_id: str) -> list[Submission]`, `create_contest(contest: Contest) -> bool`, `register_for_contest(contest_id: str, user_id: str) -> bool`, `get_leaderboard(contest_id: str) -> list[LeaderboardRow] or None`
  - **JudgeWorkerPool**: Runs several worker threads · *owns:* the threads
    - `start() -> bool`, `service.process_next(wait_ms)`, `stop() -> None`
- **Relationships:**
  - ProblemCatalog **has** Problem (added and kept there). Problem **has** TestCase.
  - JudgeService **has** Submission, Contest (created inside `submit` and `create_contest`).
  - JudgeService **uses** ProblemCatalog, Judge, SubmissionQueue, LanguageSpec.
  - Judge **uses** SandboxFactory. Submission **uses** JudgeResult.
  - JudgeWorkerPool **uses** JudgeService.
  - Enums are always **uses**: Problem to Difficulty, Submission to SubmissionStatus, JudgeResult to Verdict.
  - `Sandbox`, `ExecResult` and `LeaderboardRow` appear only as method inputs and outputs, so no attribute holds them and they have no line.
- **Call flows:**
  - **Submit**
    1. `User -> JudgeService.submit(...)`
    2. `JudgeService -> ProblemCatalog.get_problem`
    3. `JudgeService -> Submission(...)`
  - **Judge one submission (worker step)**
    1. `Worker -> JudgeService.process_next(wait)`
    2. `JudgeService -> SubmissionQueue.take(wait)`, then `Submission.start()`
    3. `JudgeService -> Judge.evaluate(source, language, problem)`
    4. `Judge -> SandboxFactory.create()`, then `Sandbox.write_file`
    5. `JudgeService -> Submission.finish(result)`
  - **Leaderboard**
    1. `User -> JudgeService.get_leaderboard(contest_id)`, then `Contest.build_leaderboard(contest submissions)`
- **Failure and undo (per flow):**
  - **Submit:** nothing is stored or queued.
  - **History and leaderboard:** an unknown contest returns None.
- **Concurrency:**
  - **Queue:** the lock sits inside `SubmissionQueue`, around `put`, `take` and `close`, with a condition so idle workers sleep. Each id is handed to exactly one worker.
  - **Submission:** the lock sits inside `Submission`, around `start` and `finish`, so it is judged at most once.
  - **Contest:** the lock sits inside `Contest`, around `participants`.
  - **Judge:** it holds no changing data, so all workers share one. Each submission gets its own sandbox.
  - **Maps and ids:** thread-safe maps and an atomic counter.
  - **No lock while judging:** compiling and running happen with no lock held, so a slow submission never blocks others.
  - **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
- **Key decisions:**
  1. **A language is data, not a class.** *Rejected:* one class per language, which would be the same code with different strings.
  2. **A fresh sandbox per submission, always destroyed.** *Rejected:* reusing sandboxes between submissions, which leaks files and state.
  3. **Submitting only queues.** *Rejected:* judging inside `submit`, which would make the user wait for compilation and every test.
  4. **The leaderboard is computed from judged submissions.** *Rejected:* a stored score table updated on every verdict.
  5. **Judging stops at the first failing test.** *Rejected:* running every test anyway, which only helps if partial scores are wanted.
- **Edge cases:**
  - An unknown problem or language, or empty code, is refused.
  - A submission for a contest the user didn't join, or outside its times, is refused.
  - Code that doesn't compile gets `COMPILE_ERROR` and is never run.
  - A run that hits the time or memory limit gets the matching verdict.
  - Spaces at the ends of lines don't cause a wrong answer.
  - A submission is judged at most once, even if its id is taken twice.
  - The sandbox is destroyed even when judging fails.
  - Wrong submissions after a problem is solved don't add penalty.
- **Extensibility (change → what to add):**
  - A new language → A new `LanguageSpec` passed to `add_language` *(edits existing code: No. It is data)*
  - More workers → A different `size` for `JudgeWorkerPool` *(edits existing code: No. It is data)*
  - A different isolation technology → A new class implementing `Sandbox` and `SandboxFactory` *(edits existing code: No)*
  - New problems or contests → Calls to `add_problem` and `create_contest` *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"How would you scale to thousands of submissions a minute?"** Move the queue to a message broker and run workers on many machines. `JudgeService.submit` and `process_next` stay the same around it.
  2. **"How would you rejudge after a test case is fixed?"** Reset the affected submissions to `QUEUED` and put them back on the queue. The leaderboard updates itself because it is computed.
  3. **"How would you support problems with many correct answers?"** Pass a checker program to the judge, run it in the sandbox, and use its result instead of a text comparison.
- **Out of scope (one-line answer if asked):**
  - The real isolation technology → It is behind `Sandbox`. Containers or jails would implement it.
  - Partial scores per test → Judging stops at the first failure. Partial scores need every test run and a weight per test.
  - Custom checkers → See follow-up question 3.
  - Rejudging → See follow-up question 2.
  - User accounts → Users are only an id.
  - Showing hidden test cases → Only sample tests are shown, by design.
  - Freezing the leaderboard near the end → Not required. It would hide submissions after a cut-off time.

#### 29 · Webhook Delivery
- **Enums:**
  - **DeliveryStatus**: `PENDING`, `RETRYING`, `SUCCEEDED`, `DEAD`
  - **EndpointStatus**: `ACTIVE`, `DISABLED`. *Why:* requirement 9
  - **ResponseClass**: `SUCCESS`, `RETRYABLE`, `PERMANENT`
- **Class details (job · what it owns · key methods):**
  - **Event**: Records one thing that happened · *owns:* id, type, payload, time
    - `to_body() -> str`
  - **SecretEntry**: Holds one signing secret and until when it is valid · *owns:* secret, expiry
    - `get_secret() -> str`, `is_valid(now_ms: int) -> bool`
  - **Endpoint**: Holds a customer's URL, event types, secrets and health, and switches itself off · *owns:* status, failure count, secrets
    - `is_active() -> bool`, `subscribes_to(event_type: str) -> bool`, `set_event_types(types: set[str]) -> bool`, `signing_secrets(now_ms: int) -> list[str]`, `rotate_secret(new_secret: str, now_ms: int, overlap_ms: int) -> bool`, `record_success() -> None`, `record_failure(threshold: int) -> bool`, `enable() -> bool`, `disable() -> bool`
  - **HttpResponse**: Records what an endpoint answered · *owns:* status code, time-out flag, timing
    - `post(url: str, headers: map{str: str}, body: str, timeout_ms: int) -> HttpResponse`
  - **PayloadSigner**: Builds the signature header
    - `sign(secrets: list[str], timestamp_s: int, body: str) -> str`
  - **ResponseClassifier**: Sorts an answer into success, retry or give up
    - `classify(response: HttpResponse) -> ResponseClass`
  - **RetryPolicy**: Says how long to wait before each retry, and when to give up · *owns:* attempts, delays
    - `next_delay(attempts_made: int, random_fraction: float) -> int or None`
  - **DeliveryAttempt**: Records one try · *owns:* an unchangeable record
  - **Delivery**: Tracks one event to one endpoint, its tries and its status · *owns:* status, tries, next time
    - `get_attempts() -> list[DeliveryAttempt]`, `record_attempt(response: HttpResponse, result: ResponseClass, now_ms: int, policy: RetryPolicy, random_fraction: float) -> DeliveryStatus or None`, `mark_dead(reason: str) -> bool`, `reset_for_replay(now_ms: int) -> bool`
  - **DeliveryQueue**: Hands out due deliveries fairly across endpoints, with a limit per endpoint · *owns:* the waiting deliveries, work in progress per endpoint
    - `add(endpoint_id: str, delivery_id: str, due_ms: int) -> None`, `take(now_ms: int) -> (str, str) or None`, `done(endpoint_id: str) -> None`, `size() -> int`
  - **WebhookService**: Registers endpoints, publishes events, runs a worker step, and answers queries · *owns:* endpoints, events, deliveries
    - `register_endpoint(customer_id: str, url: str, event_types: set[str], secret: str) -> Endpoint or None`, `update_event_types(customer_id: str, endpoint_id: str, event_types: set[str]) -> bool`, `rotate_secret(customer_id: str, endpoint_id: str, new_secret: str, now_ms: int) -> bool`, `enable_endpoint(customer_id: str, endpoint_id: str) -> bool`, `disable_endpoint(customer_id: str, endpoint_id: str) -> bool`, `publish(event_type: str, payload: str, now_ms: int) -> Event`, `process_next(now_ms: int, random_fraction: float) -> bool`, `get_attempts(customer_id: str, delivery_id: str) -> list[DeliveryAttempt] or None`, `list_deliveries(customer_id: str, endpoint_id: str, status: DeliveryStatus or None) -> list[Delivery] or None`, `list_dead_letters(customer_id: str) -> list[Delivery]`, `replay(customer_id: str, delivery_id: str, now_ms: int) -> bool`, `replay_all_dead(customer_id: str, endpoint_id: str, now_ms: int) -> int`, `owned_endpoint(customer_id: str, endpoint_id: str) -> Endpoint or None`
  - **WebhookWorkerPool**: Runs several worker threads · *owns:* the threads
    - `start() -> bool`, `service.process_next(clock.now_millis(), a fresh random number)`, `stop() -> None`
- **Relationships:**
  - WebhookService **has** Endpoint, Event, Delivery (created inside `register_endpoint` and `publish`).
  - Endpoint **has** SecretEntry. Delivery **has** DeliveryAttempt.
  - Delivery **uses** Event (shared by every delivery of that event).
  - WebhookService **uses** DeliveryQueue, HttpSender, PayloadSigner, ResponseClassifier, RetryPolicy (passed in).
  - WebhookWorkerPool **uses** WebhookService, Clock.
  - Enums are always **uses**: Endpoint to EndpointStatus, Delivery to DeliveryStatus, DeliveryAttempt to ResponseClass.
  - `HttpResponse` appears only as a method input and output, so it has no line.
- **Call flows:**
  - **Publish**
    1. `Platform code -> WebhookService.publish(event_type, payload, now)`
    2. `WebhookService -> Endpoint.is_active()` and `subscribes_to(event_type)`
    3. `WebhookService -> Delivery(...)`
  - **Send one delivery (worker step)**
    1. `Worker -> WebhookService.process_next(now, random)`
    2. `WebhookService -> DeliveryQueue.take(now)`
    3. `WebhookService -> Endpoint.signing_secrets(now)`, then `PayloadSigner.sign(...)`
    4. `WebhookService -> HttpSender.post(url, headers, body, timeout)`
    5. `WebhookService -> ResponseClassifier.classify(response)`, then `Delivery.record_attempt(...)`
    6. `WebhookService -> Endpoint.record_success()`
    7. If `RETRYING`
  - **Replay**
    1. `Customer -> WebhookService.replay(customer, delivery, now)`, then `Delivery.reset_for_replay(now)`, then `DeliveryQueue.add(...)`
- **Failure and undo (per flow):**
  - **Publish (requirements 2, 3, 7):** an endpoint that is disabled at publish time gets no delivery. An event with no subscribers is stored and simply goes nowhere.
  - **Send one delivery:** nothing to undo. A failure only adds an attempt, may schedule a retry, and may move the delivery to the dead-letter list.
  - **Replay (requirement 6):** returns False for another customer's delivery, a delivery that is not dead, or a disabled endpoint.
- **Concurrency:**
  - **Queue:** the lock sits inside `DeliveryQueue`, around every method. Each delivery is handed to exactly one worker, and the in-flight limit is checked and raised in one step.
  - **Endpoint:** the lock sits inside `Endpoint`, around its types, secrets, status and failure count.
  - **Delivery:** the lock sits inside `Delivery`, around `record_attempt`, `mark_dead` and `reset_for_replay`, so a replay can't overlap a worker's update.
  - **Maps and ids:** thread-safe maps and an atomic counter.
  - **No lock during the POST:** the HTTP call happens with no lock held, so a slow endpoint holds only its own worker.
  - **Isolation:** at most `max_in_flight_per_endpoint` workers can be stuck on one endpoint. The rest keep serving other customers.
  - **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
- **Key decisions:**
  1. **Publishing only queues.** *Rejected:* sending inside `publish`, which would make a payment wait on a stranger's slow server.
  2. **One line per endpoint, round-robin, with a limit per endpoint.** *Rejected:* one shared first-in first-out queue, where a backlog for one endpoint delays everyone.
  3. **Answers are sorted once into success, retry or give up.** *Rejected:* retrying every failure.
  4. **At least once, with an event id, instead of exactly once.** *Rejected:* promising exactly once, which can't be guaranteed across a network.
  5. **The dead-letter list is a status, not a separate store.** *Rejected:* copying failed deliveries into another table.
- **Edge cases:**
  - A non-HTTPS URL, no event types, or a short secret is refused.
  - An event with no subscribers goes nowhere.
  - A 400 goes straight to the dead-letter list without retrying.
  - A time-out or 503 is retried with a doubling delay (with jitter), up to the limit.
  - A response lost after the customer processed it is sent again with the same event id.
  - An endpoint switched off while deliveries are queued moves them to the dead-letter list, and they can be replayed later.
  - A replay to a disabled endpoint is refused.
  - During a secret rotation, both secrets sign every delivery.
  - A customer can't see or replay another customer's deliveries.
- **Extensibility (change → what to add):**
  - Different retry timing → A different `RetryPolicy` *(edits existing code: No. It is data)*
  - A different time limit, failure threshold or overlap → Different numbers for the service *(edits existing code: No. It is data)*
  - A real HTTP client, or one with a proxy → A new class implementing `HttpSender` *(edits existing code: No)*
  - More workers, or more per endpoint → A different pool size or queue limit *(edits existing code: No. It is data)*
  - A second signing scheme → Make `PayloadSigner` an interface at that point *(edits existing code: One change, made only when a second scheme is needed)*
- **Follow-up questions:**
  1. **"What if the process crashes with deliveries in the queue?"** Store deliveries in a database and write them in the same transaction as the business change that raised the event (the outbox pattern). Workers then read due deliveries from the database or a durable queue.
  2. **"Can you guarantee events arrive in order?"** Only by sending one delivery at a time per endpoint and making later ones wait for a retrying one, which lets one bad event block all the rest. Most platforms don't promise order, and give `created_at` so customers can sort.
  3. **"How does the customer verify a request?"** Take the timestamp from the header, refuse it if it is more than five minutes old, compute HMAC-SHA256 of `"<timestamp>.<raw body>"` with their secret, and compare with each `v1` value using a constant-time comparison.
- **Out of scope (one-line answer if asked):**
  - Surviving a crash → See follow-up question 1.
  - Ordered delivery → Not promised. See follow-up question 2.
  - Removing old attempts after a retention period → Not required. A sweep would delete attempts older than the limit.
  - Limits on how many requests per second an endpoint receives → The in-flight limit caps concurrency. A per-second limit would be a rate limiter checked in `take`.
  - Customers choosing payload versions → Not required. It would be a version on the endpoint used by `to_body`.
  - Checking that the URL belongs to the customer → Not required. A challenge request at registration would do it.


---

## Type F: Data Structures / Infra

#### 06 · Rate Limiter
- **Class details (job · what it owns · key methods):**
  - **RateLimitRule**: Describes one limit · *owns:* the limit and the window
    - `get_limit() -> int`, `get_window_seconds() -> float`
  - **RuleRegistry**: Finds the right rule for a client and endpoint, and lets rules be changed · *owns:* the rules
    - `get_rule(client_id: str, endpoint: str) -> RateLimitRule`, `set_rule(client_id: str or None, endpoint: str or None, rule: RateLimitRule) -> bool`, `remove_rule(client_id: str or None, endpoint: str or None) -> bool`, `set_default_rule(rule: RateLimitRule) -> None`
  - **UsageTracker**: Decides allow or reject for one client and endpoint
    - `try_acquire(rule: RateLimitRule, now: float) -> bool`
  - **FixedWindowTracker**: Counts requests in fixed windows · *owns:* window start, count
    - `try_acquire(rule: RateLimitRule, now: float) -> bool`
  - **SlidingWindowLogTracker**: Remembers request times inside the window · *owns:* the recent request times
    - `try_acquire(rule: RateLimitRule, now: float) -> bool`
  - **TokenBucketTracker**: Spends and refills tokens · *owns:* tokens, last refill time
    - `try_acquire(rule: RateLimitRule, now: float) -> bool`, `(now - last_refill) * limit / window_seconds`
  - **LimitAlgorithm**: Creates a fresh tracker of one kind
    - `create_tracker() -> UsageTracker`
  - **FixedWindowAlgorithm**: Creates fixed-window trackers
    - `create_tracker() -> UsageTracker`
  - **SlidingWindowLogAlgorithm**: Creates sliding-log trackers
    - `create_tracker() -> UsageTracker`
  - **TokenBucketAlgorithm**: Creates token-bucket trackers
    - `create_tracker() -> UsageTracker`
  - **RateLimiter**: Finds the rule and the tracker for each request and asks the tracker · *owns:* the trackers
    - `allow(client_id: str, endpoint: str, now: float) -> bool`
- **Relationships:**
  - RuleRegistry **uses** RateLimitRule (rules are unchangeable records that can be shared, and the registry just holds the ones it is given).
  - RateLimiter **has** UsageTracker (trackers are created inside `allow`, kept in the map, and die with the limiter).
  - RateLimiter **uses** RuleRegistry, LimitAlgorithm (both passed in).
  - FixedWindowTracker, SlidingWindowLogTracker, TokenBucketTracker **implement** UsageTracker.
  - FixedWindowAlgorithm, SlidingWindowLogAlgorithm, TokenBucketAlgorithm **implement** LimitAlgorithm.
  - There are no enums, so there are no enum lines.
  - Each algorithm creates its tracker and each tracker takes a rule as a method input, but no attribute holds those classes there, so they have no line.
- **Call flows:**
  - **Check a request**
    1. `Caller -> RateLimiter.allow(client_id, endpoint, now)`
    2. `RateLimiter -> RuleRegistry.get_rule(client_id, endpoint)`
    3. `RateLimiter -> trackers.get(key)`
    4. If none exists
    5. `RateLimiter -> UsageTracker.try_acquire(rule, now)`
  - **Change a limit**
    1. `Administrator -> RuleRegistry.set_rule(client_id, endpoint, rule)`
- **Failure and undo (per flow):**
  - **Check a request:** the only failure is the empty-id refusal at step 1, which changed nothing. A rejection at step 5 counts nothing, so there is nothing to undo.
  - **Change or remove a limit:** returns False and nothing changes. Existing trackers keep their state and use the new rule from their next call.
- **Concurrency:**
  - **Tracker:** the lock sits inside each tracker, around the whole of `try_acquire`. "Check the count, then add one" is one step, so two simultaneous requests can never both take the last slot.
  - **Different keys:** each client and endpoint has its own tracker and lock, so requests for different keys never wait for each other.
  - **Tracker creation:** `trackers` is a thread-safe map, and `put_if_absent` makes first-time creation safe. A tracker that loses the race is simply discarded.
  - **Registry:** the lock sits inside `RuleRegistry`, around its map and default. A rule change and a lookup can't overlap.
  - **Lock order:** no method ever holds two locks at once. `allow` finishes with the registry lock before it takes a tracker lock, so no deadlock is possible.
- **Key decisions:**
  1. **Two levels: the algorithm creates a tracker, and the limiter manages keys and rules.** *Rejected:* a separate limiter class per algorithm, each with its own map and its own rule lookup. That code would be copied three times.
  2. **A tracker holds only counting state, and the rule is passed in on every call.** *Rejected:* a tracker that stores its rule. It would keep using the old limit after a change.
  3. **One lock per tracker.** *Rejected:* one lock for the whole limiter. Every request would wait for every other.
  4. **A rule is an unchangeable record that is replaced whole.** *Rejected:* changing `limit` and `window_seconds` in place.
  5. **`put_if_absent` creates a tracker.** *Rejected:* check-then-insert, which could give each request its own tracker and let both through.
- **Edge cases:**
  - The first request for a pair creates its tracker.
  - A window that has ended starts a new one.
  - A request with an empty client id or endpoint is rejected.
  - A rule changed in the middle of a window takes effect at once.
  - A client with no specific rule gets the default.
  - A token bucket with a fraction of a token refuses until a whole token is available.
  - Many simultaneous requests for one pair: exactly the allowed number succeed.
  - A client on a different endpoint has a separate count.
- **Extensibility (change → what to add):**
  - A new algorithm (leaky bucket, sliding window counter) → A new tracker class and a new `LimitAlgorithm` class, passed to the `RateLimiter` constructor *(edits existing code: No)*
  - A different limit for a client, an endpoint, or a plan (free versus paid) → Calls to `set_rule` *(edits existing code: No. It is data)*
  - Changing a limit while the limiter runs → A new rule passed to `set_rule` or `set_default_rule` *(edits existing code: No. It is data)*
  - A different default limit → A different `RateLimitRule` passed to the `RuleRegistry` constructor *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"Doesn't a fixed window allow a double burst at the boundary?"** Yes. A client can use the whole limit at the end of one window and again at the start of the next. The sliding log and the token bucket avoid this. The sliding log costs memory in proportion to the limit, and the token bucket keeps just two numbers.
  2. **"What happens to trackers for clients that never come back?"** They stay in memory. A fix is to record each tracker's last-used time and sweep out idle ones with a periodic method. It is not built here because no requirement asks for it.
  3. **"How would this work across many servers?"** The in-memory trackers would move to a shared store, with the check-and-update done as one atomic operation there. `UsageTracker` is already the boundary where that change goes.
- **Out of scope (one-line answer if asked):**
  - A limit shared across all of a client's endpoints → Counting is per client and endpoint pair. A shared quota would need a tracker keyed by client alone.
  - Telling the caller when to retry or how many requests remain → `allow` returns only allow or reject. A richer answer would be a new return record.
  - Removing idle trackers → See follow-up question 2.
  - Changing the algorithm while running → The algorithm is chosen when the limiter is built. Switching would start fresh counts, so a new limiter replaces the old one.
  - Time going backwards → Time is assumed never to go backwards.
  - Fractional limits → A limit is a whole number of requests.
  - Many servers → See follow-up question 3.
  - Reading the clock → The caller passes the time in, so tests need no clock.

#### 07 · LRU / LFU Cache
- **Class details (job · what it owns · key methods):**
  - **UsageNode**: Is one link in an ordered chain of keys · *owns:* its key and its two neighbours
    - `get_key() -> K or None`, `get_previous() -> UsageNode or None`, `get_next() -> UsageNode or None`, `set_previous(node: UsageNode or None) -> None`, `set_next(node: UsageNode or None) -> None`
  - **UsageList**: Keeps keys in order, oldest first, with constant-time add, remove and take-oldest · *owns:* the order of its keys
    - `add_newest(key: K) -> bool`, `remove(key: K) -> bool`, `touch(key: K) -> bool`, `pop_oldest() -> K or None`, `contains(key: K) -> bool`, `size() -> int`
  - **EvictionPolicy**: Tracks key usage and names the key to evict
    - `on_insert(key: K) -> bool`, `on_access(key: K) -> bool`, `evict() -> K or None`
  - **LruPolicy**: Evicts the key used longest ago · *owns:* the recency order
    - `on_insert(key: K) -> bool`, `on_access(key: K) -> bool`, `evict() -> K or None`
  - **LfuPolicy**: Evicts the key with the fewest uses, oldest first among ties · *owns:* each key's use count
    - `on_insert(key: K) -> bool`, `on_access(key: K) -> bool`, `evict() -> K or None`
  - **Cache**: Stores the values, enforces the capacity, and keeps the policy informed · *owns:* the values, the capacity
    - `get(key: K) -> V or None`, `put(key: K, value: V) -> bool`, `size() -> int`, `get_capacity() -> int`
- **Relationships:**
  - Cache **uses** EvictionPolicy (passed in, and can be any rule).
  - LruPolicy **has** UsageList (created inside the constructor).
  - LfuPolicy **has** UsageList (one is created inside for each count).
  - UsageList **has** UsageNode (nodes are created inside `add_newest`).
  - LruPolicy, LfuPolicy **implement** EvictionPolicy.
  - There are no enums, so there are no enum lines.
- **Call flows:**
  - **Read**
    1. `Caller -> Cache.get(key)`
    2. `Cache -> store`
    3. `Cache -> EvictionPolicy.on_access(key)`
    4. `LruPolicy -> UsageList.touch(key)`
  - **Write a new key when full**
    1. `Caller -> Cache.put(key, value)`
    2. `Cache -> EvictionPolicy.evict()`
    3. `Cache -> store`
    4. `Cache -> EvictionPolicy.on_insert(key)`
- **Failure and undo (per flow):**
  - **Read (`get`):** the only failure is the miss, which changed nothing.
  - **Write (`put`):** steps 1 and 3 refuse before changing anything. In step 4 the undo removes exactly the one entry just stored.
- **Concurrency:**
  - **Cache:** the lock sits inside `Cache`, around the whole of `get` and `put`. Each operation checks and changes the map and the policy as one step, so two threads can't both see room for the last entry, and the policy's lists can't be corrupted.
  - **Why reads are locked too:** a read changes the order or the counts, so it is not a read-only action.
  - **Policies and lists:** they have no lock. Only `Cache`, holding its lock, reaches them.
  - **Lock order:** only one lock exists, so no deadlock is possible.
- **Key decisions:**
  1. **Values live only in `Cache`, and the policy tracks only keys.** *Rejected:* the classic design where each list node also holds the value. The policy would then own the data, and a second rule would need its own copy of the storage.
  2. **Three tiny policy methods that the cache calls in step.** *Rejected:* a policy that inspects the cache. It would depend on the cache while the cache depends on it.
  3. **One `UsageList` used by both policies.** *Rejected:* a separate structure for each policy.
  4. **LFU uses count buckets and a minimum count.** *Rejected:* a heap, which costs time proportional to the logarithm of the size, and a scan for the minimum, which is linear.
  5. **One lock on the whole cache, even for reads.** *Rejected:* a lock only around writes. Two reads could both reorder the list at once and corrupt it.
- **Edge cases:**
  - Updating an existing key never evicts.
  - A miss changes nothing, and does not count as a use.
  - A capacity of 0 or less is refused.
  - A value of None is refused, so None always means missing.
  - A capacity of 1 evicts on every new key.
  - LFU: keys with the same count are evicted oldest-use first.
  - A new key in LFU starts at a count of 1 and resets the minimum to 1.
  - The size never goes above the capacity.
- **Extensibility (change → what to add):**
  - A new eviction rule (first in first out, most recently used) → A new class implementing `EvictionPolicy`, passed to the `Cache` constructor *(edits existing code: No)*
  - A different capacity → A different number passed to the `Cache` constructor *(edits existing code: No. It is data)*
  - Both LRU and LFU in the same program → Two caches built with different policies *(edits existing code: No)*
  - Different key and value types → Use the cache with other types (`K` and `V` are generic) *(edits existing code: No)*
- **Follow-up questions:**
  1. **"Why does a read need the lock?"** Because a read moves the key in the order or raises its count. Two threads doing that at once could corrupt the linked list.
  2. **"How would you add a time to live?"** Store an expiry time with each value and treat an expired key as missing on read. That means a small change inside `Cache`, because the values and their times must be kept together. Eviction by rule and expiry by time are separate reasons for removal.
  3. **"Can reads run in parallel?"** Only by splitting the cache into several independent caches by key, each with its own lock. The trade-off is that the capacity is divided, so it is no longer one global limit.
- **Out of scope (one-line answer if asked):**
  - Deleting or invalidating a key → No requirement asks for it. It would be a `remove` on `Cache` and on the policy.
  - Time to live → See follow-up question 2.
  - Storing None as a value → Values are never None, so None can mean missing. A wrapper type would lift this.
  - Hit and miss statistics → No requirement asks for them. They would be counters in `Cache`.
  - Capacity measured in bytes → The capacity counts entries. A size would be a new field with each value.
  - Telling someone when an entry is evicted → No requirement asks for it. It would be a listener called after step 3 of `put`.
  - Reading in parallel → See follow-up question 3.

#### 09 · Logging Framework
- **Enums:**
  - **LogLevel**: `DEBUG` 10, `INFO` 20, `WARN` 30, `ERROR` 40, `FATAL` 50
- **Class details (job · what it owns · key methods):**
  - **LogEntry**: Records one log event · *owns:* timestamp, level, message
    - `get_timestamp() -> datetime`, `get_level() -> LogLevel`, `get_message() -> str`
  - **LogFormatter**: Turns an entry into text
    - `format(entry: LogEntry) -> str`
  - **PlainFormatter**: Makes a one-line readable text
    - `format(entry: LogEntry) -> str`
  - **JsonFormatter**: Makes a JSON text
    - `format(entry: LogEntry) -> str`
  - **LogDestination**: Writes one entry somewhere
    - `write(entry: LogEntry) -> bool`
  - **ConsoleDestination**: Writes to the console · *owns:* its formatter
    - `write(entry: LogEntry) -> bool`
  - **FileDestination**: Writes to a file · *owns:* file path, formatter
    - `write(entry: LogEntry) -> bool`
  - **DatabaseDestination**: Writes to a database table · *owns:* table name, formatter
    - `write(entry: LogEntry) -> bool`
  - **LogHandler**: Is one link in the chain, dealing with one level · *owns:* its level, its destinations
    - `handle(entry: LogEntry) -> bool`
  - **Clock**: Says what time it is
    - `now() -> datetime`
  - **SystemClock**: Reads the real time
    - `now() -> datetime`
  - **Logger**: Filters by minimum level, stamps the entry and starts the chain · *owns:* minimum level
    - `log(level: LogLevel, message: str) -> bool`, `set_min_level(level: LogLevel) -> None`, `get_min_level() -> LogLevel`, `create_default() -> Logger`
  - **LogManager**: Is the one shared place everyone gets the configured logger from · *owns:* the logger
    - `get_instance() -> LogManager`, `initialize(logger: Logger) -> bool`, `get_logger() -> Logger`
- **Relationships:**
  - LogManager **has** Logger (the default is created inside, and the set one is kept).
  - Logger **has** LogHandler (the head of the chain; the handlers live and die with the logger's configuration).
  - LogHandler **uses** LogDestination, LogHandler (the next link). Destinations are passed in.
  - Logger **uses** Clock (passed in).
  - ConsoleDestination, FileDestination, DatabaseDestination **use** LogFormatter (passed in).
  - ConsoleDestination, FileDestination, DatabaseDestination **implement** LogDestination.
  - PlainFormatter, JsonFormatter **implement** LogFormatter. SystemClock **implements** Clock.
  - Enums are always **uses**: LogEntry to LogLevel, LogHandler to LogLevel, Logger to LogLevel.
  - `LogEntry` appears only as a method input, so no attribute holds it and it has no line.
- **Call flows:**
  - **Get the shared logger**
    1. `Any code -> LogManager.get_instance().get_logger()`
  - **Log a message**
    1. `Any code -> Logger.log(level, message)`
    2. `Logger -> Logger.min_level`
    3. `Logger -> Clock.now()`
    4. `Logger -> LogHandler.handle(entry)`
    5. `LogHandler -> next LogHandler.handle(entry)`
    6. `LogHandler -> LogDestination.write(entry)`
    7. `LogDestination -> LogFormatter.format(entry)`
  - **Configure once**
    1. `Startup -> LogManager.get_instance().initialize(logger)`
- **Failure and undo (per flow):**
  - **Log a message:** there is nothing to undo. A message that is filtered out, has no handler, or hits a failing destination changes no program state. A failing destination returns False and the others still write.
  - **Pass along the chain:** a level with no handler is dropped and `log` returns False. A destination that fails does not stop the loop.
  - **Set up the shared logger:** a second `initialize` returns False and changes nothing. Before any setup, `get_logger()` still returns the default logger, so a log call never fails.
  - **Change the minimum level:** none. The next `log` call uses the new level.
- **Concurrency:**
  - **Destinations:** the lock sits inside each destination, around the whole of `write`. One line is formatted and written as one step, so lines from different threads are never mixed. Two different destinations don't wait for each other.
  - **Logger:** the lock sits inside `Logger`, around `min_level` only. It is released before the chain runs.
  - **Chain and formatters:** they have no lock, because a chain is never changed once built and the formatters hold no data.
  - **LogManager:** the lock sits inside `LogManager`, around creating the instance and the one-time setup.
  - **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
  - **Why not one lock around the whole `log`:** every thread would wait for the slowest destination, even when it only needed the console.
- **Key decisions:**
  1. **One `LogHandler` class with the level as data.** *Rejected:* a subclass per level, which copies the same code five times.
  2. **Each destination owns its formatter and its lock.** *Rejected:* one formatter for the whole logger, and one lock for the whole logger.
  3. **The minimum level is checked in the logger before the chain.** *Rejected:* every handler checking the minimum, which repeats the rule.
  4. **Only `LogManager` is a singleton.** *Rejected:* making `Logger` itself a singleton, which would make it hard to test.
  5. **A failing destination returns False instead of throwing.** *Rejected:* letting the error escape into the application.
- **Edge cases:**
  - A message below the minimum level is dropped without a timestamp being taken.
  - A level with no handler is dropped and `log` returns False.
  - One destination failing does not stop the others.
  - Two threads asking for the manager at the same moment get the same instance.
  - A second `initialize` is refused.
  - The default logger is used if no one configures one.
  - A None message is refused. An empty message is allowed.
  - The minimum level can change while other threads are logging.
- **Extensibility (change → what to add):**
  - A new destination (a message queue, a remote service) → A new class implementing `LogDestination`, added to a handler's list when the chain is built *(edits existing code: No)*
  - A new format (XML, a custom layout) → A new class implementing `LogFormatter`, passed to a destination *(edits existing code: No)*
  - A new level (TRACE) → One value in `LogLevel`, and one more `LogHandler` in the chain *(edits existing code: No)*
  - Sending ERROR to a different set of places → A different destination list for the ERROR handler *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"How would you make logging not slow the application?"** Put a queue in front of a destination, with a worker thread that writes. That is a new class implementing `LogDestination` that wraps another destination, so nothing else changes.
  2. **"How would you rotate a log file?"** Inside `FileDestination.write`: when the file is too big or a day has passed, start a new file, under the same lock.
  3. **"How do you test code that uses a singleton?"** Test with a `Logger` you build yourself, because `Logger` is an ordinary class. Only the shared access point is a singleton.
- **Out of scope (one-line answer if asked):**
  - Named loggers per class or package → One shared logger. Named loggers would be a map from a name to a `Logger` in the manager.
  - Changing the chain or destinations while running → A new configuration means a new `Logger`. The chain is built once.
  - Filtering by level per destination → The handler for each level chooses the destinations, which gives per-level routing.
  - Asynchronous logging → See follow-up question 1.
  - Rotating files → See follow-up question 2.
  - Real file and database access → Only the three destination `write` methods touch the outside world.
  - Logging exceptions with stack traces → The message is text. A stack trace would be added to it by the caller.
  - `info()` and `error()` shortcuts → They would only call `log`. They can be added without any other change.

#### 20 · In-Memory File System
- **Class details (job · what it owns · key methods):**
  - **Path**: Turns path text into clean, absolute steps from the root · *owns:* the list of names
    - `parse(text: str, current: Path) -> Path or None`, `is_root() -> bool`, `get_name() -> str or None`, `parent() -> Path or None`, `child(name: str) -> Path`, `starts_with(other: Path) -> bool`, `to_text() -> str`
  - **Node**: Is anything in the tree, and answers size and search
    - `is_directory() -> bool`, `get_size() -> int`, `find(path: Path, criterion: SearchCriterion) -> list[str]`
  - **File**: Holds text content · *owns:* the content
    - `is_directory() -> bool`, `read() -> str`, `write(content: str) -> bool`, `append(content: str) -> bool`, `get_size() -> int`, `find(path: Path, criterion: SearchCriterion) -> list[str]`
  - **Directory**: Holds named children of either kind · *owns:* its children and their names
    - `is_directory() -> bool`, `get_child(name: str) -> Node or None`, `add_child(name: str, node: Node) -> bool`, `remove_child(name: str) -> Node or None`, `rename_child(old_name: str, new_name: str) -> bool`, `list_names() -> list[str]`, `is_empty() -> bool`, `get_size() -> int`, `find(path: Path, criterion: SearchCriterion) -> list[str]`
  - **EntryInfo**: Gives a read-only picture of one listed item · *owns:* name, kind, size
    - `get_name() -> str`, `is_directory() -> bool`, `get_size() -> int`
  - **SearchCriterion**: Decides whether a file name matches
    - `matches(file_name: str) -> bool`
  - **NameEquals**: Matches an exact file name · *owns:* the name
  - **FileSystem**: Resolves paths and runs every user action · *owns:* the root, the current directory
    - `change_directory(path_text: str) -> bool`, `get_current_directory() -> str`, `create_file(path_text: str, content: str) -> bool`, `create_directory(path_text: str) -> bool`, `delete(path_text: str, recursive: bool) -> bool`, `rename(path_text: str, new_name: str) -> bool`, `move(source_text: str, destination_dir_text: str) -> bool`, `read_file(path_text: str) -> str or None`, `write_file(path_text: str, content: str) -> bool`, `append_file(path_text: str, content: str) -> bool`, `list_directory(path_text: str or None) -> list[EntryInfo] or None`, `search(path_text: str or None, criterion: SearchCriterion) -> list[str] or None`, `get_size(path_text: str) -> int or None`, `resolve(path: Path) -> Node or None`, `resolve_parent(path: Path) -> Directory or None`
- **Relationships:**
  - FileSystem **has** Directory (the root, created in the constructor).
  - Directory **has** Node (children live and die inside their directory).
  - FileSystem **uses** Path (the current directory is a value).
  - File, Directory **extend** Node.
  - NameEquals, ExtensionEquals **implement** SearchCriterion.
  - There are no enums, so there are no enum lines.
  - `EntryInfo` and `SearchCriterion` appear only as method outputs and inputs, so no attribute holds them and they have no line.
- **Call flows:**
  - **Any action with a path**
    1. `User -> FileSystem.<action>(path_text, ...)`
    2. `FileSystem -> Path.parse(path_text, current)`
    3. `FileSystem.resolve`
    4. `FileSystem -> File`
  - **Size and search**
    1. `User -> FileSystem.get_size(path_text)`, then `Node.get_size()`
    2. `User -> FileSystem.search(path_text, criterion)`, then `Node.find(path, criterion)`
  - **Move**
    1. `FileSystem -> Path.starts_with`
- **Failure and undo (per flow):**
  - **Create:** nothing is created. The same steps with `Directory()` create a directory.
  - **Delete:** nothing is removed.
  - **Rename:** a missing item, an invalid name or a used name changes nothing.
  - **Move:** steps 1 to 4 check everything first, so step 5's add can't fail. If it ever did, the node would be added back to `old_parent`.
  - **List, search and size:** a missing path, or the wrong kind, returns None.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added.
  - If it were needed later, one read-write lock in `FileSystem` would be the simple answer: reads (read, list, search, size) share it, and changes take it alone.
- **Key decisions:**
  1. **A name is stored only as its parent's key.** *Rejected:* a `name` field on each node as well as the parent's map.
  2. **No node points back to its parent.** *Rejected:* a parent pointer on every node, which must be kept in step on every move.
  3. **Paths are cleaned up before any walking.** *Rejected:* each method handling `..` itself while walking.
  4. **Size is computed by the tree (Composite).** *Rejected:* a stored size on each directory, updated on every write.
  5. **The current directory is a Path, not a node.** *Rejected:* holding the current directory as a node.
- **Edge cases:**
  - `..` at the root stays at the root.
  - A directory can't be moved into itself or its own subtree.
  - The root can't be deleted, renamed or moved.
  - A directory containing the current directory can't be deleted, renamed or moved.
  - A non-empty directory needs `recursive` to be deleted.
  - Creating inside a missing parent, or over an existing name, is refused.
  - Reading or writing a directory, or listing a file, is refused.
  - A search for an extension ignores case.
- **Extensibility (change → what to add):**
  - A new kind of search (a name pattern, a size limit) → A new class implementing `SearchCriterion` *(edits existing code: No)*
  - A new kind of node (a symbolic link) → A new subclass of `Node` *(edits existing code: No for the tree. Resolving a link would be a new step in `resolve`)*
  - Creating missing parent directories at once → A `create_directory` option that creates each missing step *(edits existing code: One new branch in `create_directory`)*
  - Extra details in listings (dates) → New fields in `EntryInfo` and on `Node` *(edits existing code: Additions only)*
- **Follow-up questions:**
  1. **"Isn't computing a directory's size every time slow?"** For a big tree, yes. A cached size per directory, updated on every change along the path to the root, makes it instant, at the cost of a second copy of the data that must be kept in step.
  2. **"How would you support permissions?"** Store an owner and permission bits on each `Node`, and check them in `FileSystem` before each action.
  3. **"How would you make it safe for many users?"** See the Concurrency list above: one read-write lock for the whole tree is simple. Finer locks per directory are faster but must be taken in a fixed order to avoid deadlock.
- **Out of scope (one-line answer if asked):**
  - Permissions and owners → See follow-up question 2.
  - Links → See the Extensibility list above.
  - Dates and times on files → Not required. They would be fields on `Node`.
  - Binary content → Content is text. Bytes would replace `str` in `File`.
  - Moving the current directory along with a move → Refused instead.
  - Concurrent users → See follow-up question 3.
  - Saving to disk → It is in memory, as the requirement says.

#### 21 · KV Store with TTL + Transactions
- **Enums:**
  - **ChangeType**: `SET`, `DELETE`
- **Class details (job · what it owns · key methods):**
  - **Clock**: Says what time it is
    - `now() -> datetime`
  - **Entry**: Holds one committed value and when it expires · *owns:* value, expiry time
    - `get_value() -> str`, `get_expires_at() -> datetime or None`, `is_expired(now: datetime) -> bool`
  - **Change**: Records one pending set or delete · *owns:* kind, value, expiry time
    - `get_change_type() -> ChangeType`, `get_value() -> str or None`, `get_expires_at() -> datetime or None`, `is_expired(now: datetime) -> bool`
  - **TransactionLayer**: Holds the pending changes of one open transaction · *owns:* its changes
    - `record(key: str, change: Change) -> None`, `lookup(key: str) -> Change or None`, `merge_into(lower: TransactionLayer) -> None`, `get_changes() -> map{str: Change}`
  - **KeyValueStore**: Holds committed data, applies commits in one step, and removes expired keys · *owns:* the data, the expiry queue
    - `get(key: str) -> str or None`, `set(key: str, value: str, ttl_seconds: int or None) -> bool`, `delete(key: str) -> bool`, `apply(changes: map{str: Change}) -> None`, `purge_expired() -> int`, `size() -> int`
  - **Session**: Gives one client its own view, and runs begin, commit and rollback · *owns:* its stack of layers
    - `get(key: str) -> str or None`, `set(key: str, value: str, ttl_seconds: int or None) -> bool`, `delete(key: str) -> bool`, `begin() -> None`, `commit() -> bool`, `rollback() -> bool`, `in_transaction() -> bool`, `depth() -> int`
  - **ExpiryCleaner**: Runs the cleanup in the background on a timer · *owns:* its thread
    - `start() -> bool`, `stop() -> None`
- **Relationships:**
  - KeyValueStore **has** Entry (entries are created inside `set` and `apply`).
  - TransactionLayer **has** Change (changes live and die with their layer).
  - Session **has** TransactionLayer (created by `begin`, dropped by `commit` and `rollback`).
  - Session **uses** KeyValueStore, Clock. KeyValueStore **uses** Clock. ExpiryCleaner **uses** KeyValueStore.
  - SystemClock **implements** Clock.
  - Enums are always **uses**: Change to ChangeType.
- **Call flows:**
  - **Read**
    1. `Client -> Session.get(key)`
    2. `Session -> TransactionLayer.lookup(key)`
    3. If none
  - **Write inside a transaction**
    1. `Client -> Session.set(...)`
  - **Commit**
    1. `Client -> Session.commit()`
    2. Inner
  - **Clean up**
    1. `ExpiryCleaner thread -> KeyValueStore.purge_expired()`
- **Failure and undo (per flow):**
  - **Set, get and delete without a transaction:** a None value or a time to live of 0 or less is refused with no change.
  - **Begin, commit and roll back:** commit or rollback with no open transaction returns False. A rollback needs no undo, because nothing was ever written outside the layer.
- **Concurrency:**
  - **Store:** the lock sits inside `KeyValueStore`, around `get`, `set`, `delete`, `apply` and `purge_expired`. A commit's changes are applied in one hold, so no reader sees half of a commit.
  - **Sessions:** no lock. Each session belongs to one thread, and its layers are private, which is what keeps uncommitted changes hidden.
  - **Cleaner:** it takes the same store lock, so cleanup never interleaves with a commit.
  - **Lock order:** there is only one lock, so no deadlock is possible.
  - **Two sessions changing the same key:** the later commit wins. Detecting the clash would be optimistic versioning (follow-up question 1).
- **Key decisions:**
  1. **Pending changes live in private layers, and only commits touch the store.** *Rejected:* writing to the store and keeping an undo log. Other sessions would see the changes before the commit.
  2. **Each nested transaction is one more layer.** *Rejected:* copying the whole data set at each `begin`, which costs memory in proportion to the store.
  3. **The outermost commit is applied under one hold of the store's lock.** *Rejected:* applying changes one by one with the lock taken each time.
  4. **Expiry is checked on every read and cleaned in the background.** *Rejected:* relying on the cleaner alone, which could return an expired key.
  5. **The cleaner uses a min-heap of expiry times and skips stale entries.** *Rejected:* scanning the whole map each time.
- **Edge cases:**
  - A key set again without a time to live is not deleted by its old expiry.
  - A delete inside a transaction hides the committed value from that session only.
  - A key set with a time to live inside a transaction that expires before commit is applied as a delete.
  - Commit or rollback with no open transaction returns False.
  - An inner commit keeps changes hidden until the outermost commit.
  - Rolling back an inner transaction leaves the outer one's changes in place.
  - Deleting a missing or expired key returns False.
  - A time to live of 0 or less is refused.
- **Extensibility (change → what to add):**
  - A different cleanup interval → A different `interval_ms` *(edits existing code: No. It is data)*
  - Values other than text → Use another value type in `Entry` and `Change` *(edits existing code: Type change only)*
  - A test clock → A new class implementing `Clock` *(edits existing code: No)*
  - Reading a key's remaining time to live → A new read-only method on `KeyValueStore` and `Session` *(edits existing code: Additions only)*
- **Follow-up questions:**
  1. **"How would you detect two sessions changing the same key?"** Give each entry a version number. Record the version each read saw, and at commit, refuse if any of those keys has a newer version. That is optimistic concurrency.
  2. **"Can reads run in parallel?"** Replace the store's lock with a read-write lock, so reads share it and only changes, commits and cleanup take it alone.
  3. **"What if one session is shared by several threads?"** Add a lock inside `Session` around every method. Today a session belongs to one client thread.
- **Out of scope (one-line answer if asked):**
  - Detecting conflicting commits → The later commit wins. See follow-up question 1.
  - Parallel reads → See follow-up question 2.
  - A session shared by threads → See follow-up question 3.
  - Saving to disk → It is in memory, as the requirement says.
  - A memory limit or eviction → Not required. It would be an eviction policy like an LRU cache.
  - Changing only the time to live of a key → Not required. It is a new store method.

#### 22 · Unique ID Generator (Snowflake)
- **Class details (job · what it owns · key methods):**
  - **IdLayout**: Knows how many bits each part takes, and packs and unpacks IDs · *owns:* the bit widths
    - `max_timestamp() -> int`, `max_machine_id() -> int`, `max_sequence() -> int`, `compose(timestamp: int, machine_id: int, sequence: int) -> int or None`, `decompose(id_value: int) -> IdParts or None`
  - **IdParts**: Holds the three parts of one ID · *owns:* time, machine, counter
    - `get_timestamp() -> int`, `get_machine_id() -> int`, `get_sequence() -> int`
  - **Clock**: Says the current time in milliseconds
    - `now_millis() -> int`
  - **SnowflakeGenerator**: Makes the next ID for one machine, safely across threads · *owns:* last time used, current counter
    - `next_id() -> int or None`, `decode(id_value: int) -> IdParts or None`, `get_machine_id() -> int`, `wait_until_after(timestamp: int) -> int`
- **Relationships:**
  - SnowflakeGenerator **uses** IdLayout, Clock (passed in).
  - SystemClock **implements** Clock.
  - There are no enums, so there are no enum lines.
  - `IdParts` appears only as a return type, so it has no line.
- **Call flows:**
  - **Make an ID**
    1. `Caller -> SnowflakeGenerator.next_id()`
    2. `SnowflakeGenerator -> Clock.now_millis()`
    3. Under the lock
    4. `SnowflakeGenerator -> IdLayout.compose(timestamp, machine_id, sequence)`
  - **Read an ID back**
    1. `Caller -> SnowflakeGenerator.decode(id)`, then `IdLayout.decompose(id)`
- **Concurrency:**
  - **Generator:** the lock sits inside `SnowflakeGenerator`, around the whole of steps 2 to 7. Reading the time, changing the counter and saving `last_timestamp` happen as one step, so two threads can never get the same counter in the same millisecond.
  - **Layout and parts:** they never change, so they need no lock.
  - **Lock order:** there is only one lock, so no deadlock is possible.
- **Key decisions:**
  1. **Time in the highest bits.** *Rejected:* the machine number first, which would order IDs by machine.
  2. **A machine number fixed at deployment.** *Rejected:* asking a central service for each ID, which makes every ID depend on a network call.
  3. **Wait on a small backwards jump, refuse on a large one.** *Rejected:* ignoring the clock and carrying on, which can make duplicate IDs.
  4. **One lock around the whole step.** *Rejected:* a lock-free compare-and-swap loop, which is faster under heavy contention but much harder to get right.
  5. **All bit arithmetic in `IdLayout`.** *Rejected:* shifts written inline in the generator.
- **Edge cases:**
  - Many IDs in one millisecond: the counter goes up.
  - The counter runs out: the generator waits for the next millisecond.
  - The clock goes back a little: the generator waits.
  - The clock goes back a lot: `next_id` returns None instead of risking a duplicate.
  - The time part runs out after about 69 years: `next_id` returns None.
  - A machine number too big for its bits, or a start date in the future, is refused at creation.
  - The highest bit is always 0, so IDs are never negative.
- **Extensibility (change → what to add):**
  - More machines, or more IDs per millisecond → A different `IdLayout`, such as 41 / 12 / 10 bits *(edits existing code: No. It is data)*
  - A different start date → A different `epoch_millis` *(edits existing code: No. It is data)*
  - More tolerance for clock jumps → A different `max_backward_wait_ms` *(edits existing code: No. It is data)*
  - Splitting the machine number into data centre and machine → A layout with four parts *(edits existing code: Yes, `IdLayout` gains a part)*
- **Follow-up questions:**
  1. **"How do machines get unique numbers?"** At deployment, from configuration or a coordination service such as ZooKeeper, once at start-up. The design only requires that two running processes never share one.
  2. **"Are IDs strictly ordered across machines?"** No, only roughly. Within one millisecond, IDs from different machines are ordered by machine number, and clocks on different machines are never exactly the same.
  3. **"What happens after a restart if the clock went backwards while it was down?"** `last_timestamp` starts at −1 after a restart, so the generator can't notice. Saving the last time used at shutdown and refusing to start before it would cover that.
- **Out of scope (one-line answer if asked):**
  - Handing out machine numbers → See follow-up question 1.
  - Strict order across machines → See follow-up question 2.
  - Clock jumps across a restart → See follow-up question 3.
  - IDs that hide when they were made → The time is readable from an ID by design. Scrambling it would lose the ordering.
  - More than 4096 IDs per millisecond per machine → A layout with more counter bits.


---

## Type G: Games

#### 10 · Snake and Ladder
- **Enums:**
  - **GameStatus**: `IN_PROGRESS`, `FINISHED`
- **Class details (job · what it owns · key methods):**
  - **Player**: Identifies one player · *owns:* id, name
    - `get_player_id() -> str`, `get_name() -> str`
  - **TurnResult**: Reports what happened in one turn · *owns:* an unchangeable record
    - `get_player_id() -> str`, `get_roll() -> int`, `get_start_cell() -> int`, `get_landed_cell() -> int`, `get_end_cell() -> int`, `has_won() -> bool`
  - **Board**: Knows the size and where snakes and ladders are · *owns:* size, snakes, ladders
    - `get_size() -> int`, `get_start_cell() -> int`, `get_final_cell() -> int`, `add_snake(head: int, tail: int) -> bool`, `add_ladder(bottom: int, top: int) -> bool`, `resolve(cell: int) -> int`
  - **Dice**: Rolls one die
    - `roll() -> int`
  - **StandardDice**: Rolls a random die with a chosen number of sides · *owns:* the number of sides
    - `roll() -> int`
  - **SequenceDice**: Rolls a fixed list of values in order · *owns:* the list and its position
    - `roll() -> int`
  - **DiceSet**: Rolls all dice and adds them up · *owns:* the list of dice
    - `roll_total() -> int`
  - **Game**: Runs turns, tracks positions and decides the winner · *owns:* positions, whose turn, status
    - `take_turn() -> TurnResult or None`, `get_status() -> GameStatus`, `get_winner() -> Player or None`, `get_current_player() -> Player or None`, `get_position(player_id: str) -> int or None`
- **Relationships:**
  - Game **has** Player (the game owns the list of players it was given).
  - Game **uses** Board, DiceSet (both passed in).
  - Board **uses** nothing else. DiceSet **uses** Dice (passed in).
  - StandardDice, SequenceDice **implement** Dice.
  - Enums are always **uses**: Game to GameStatus.
  - `TurnResult` appears only as a return type, so no attribute holds it and it has no line.
- **Call flows:**
  - **Take a turn**
    1. `Caller -> Game.take_turn()`
    2. `Game -> DiceSet.roll_total()`
    3. `DiceSet -> Dice.roll()`
    4. `Game -> Board.resolve(cell)`
    5. `Game -> positions`
- **Failure and undo (per flow):**
  - **Take a turn:** the only refusal is a finished game, which changes nothing. Every step before the position is updated only reads, so there is nothing to undo.
  - **Add a snake or a ladder:** returns False and nothing is added, for example a ladder pointing downward or two jumps on one cell.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added. All state changes happen inside `take_turn`.
  - If it were needed later, one lock inside `Game` around `take_turn` would be enough.
- **Key decisions:**
  1. **Positions live in `Game`, not on `Player`.** *Rejected:* a `position` field on Player that the game updates. Anyone could set it, and a player in two games would have two positions.
  2. **The board knows snakes and ladders, but not players.** *Rejected:* the board moving players. It would need to know about the game.
  3. **Snakes and ladders are two small maps, not classes.** *Rejected:* a `Snake` and `Ladder` class with identical fields.
  4. **`DiceSet` rolls and adds.** *Rejected:* the game looping over dice itself.
  5. **`take_turn` returns a record of what happened.** *Rejected:* returning only a bool.
- **Edge cases:**
  - A roll that would pass the final cell leaves the player where they are, and the turn passes.
  - Landing on a snake head or ladder bottom follows it once.
  - A ladder that goes to the final cell wins.
  - A turn after the game ends returns None.
  - A snake starting on the final cell is refused.
  - Two jumps starting on the same cell are refused.
  - One player, or two players with the same id, is refused.
  - A die that could return less than 1 can't be built.
- **Extensibility (change → what to add):**
  - A new kind of die (weighted, 8 sides) → A new class implementing `Dice`, or a `StandardDice` with a different number of sides *(edits existing code: No)*
  - More or fewer dice → A longer or shorter list passed to `DiceSet` *(edits existing code: No. It is data)*
  - A bigger board or different snakes and ladders → A different size and more `add_snake` and `add_ladder` calls *(edits existing code: No. It is data)*
  - More players → A longer list passed to `Game` *(edits existing code: No. It is data)*
- **Follow-up questions:**
  1. **"How would you give an extra turn for rolling a six?"** In `take_turn`, do not move `current_index` on when the roll is the maximum. Because it is a rule about turns, a `TurnRule` object passed in would keep it out of `Game`.
  2. **"What if the player must bounce back instead of staying?"** Change step 3: the overshoot is subtracted from the final cell. It is the one place the final-cell rule lives.
  3. **"How do you test a game with random dice?"** Use `SequenceDice` so every roll is known in advance.
- **Out of scope (one-line answer if asked):**
  - Needing an exact roll, or bouncing back, at the end → The player stays put. See follow-up question 2.
  - An extra turn for a six → See follow-up question 1.
  - Several players sharing a cell, or capturing → Players can share a cell.
  - Players joining during the game → The player list is fixed when the game is created.
  - Showing the board → `get_position` gives each player's cell. Drawing it is a display concern.
  - Concurrent turns → One lock inside `Game` around `take_turn` would be enough.

#### 11 · Tic-Tac-Toe
- **Enums:**
  - **Symbol**: `X`, `O`
  - **GameStatus**: `IN_PROGRESS`, `WON`, `DRAWN`
- **Class details (job · what it owns · key methods):**
  - **Player**: Identifies one player and their symbol · *owns:* name, symbol
    - `get_name() -> str`, `get_symbol() -> Symbol`
  - **Board**: Holds the cells and checks moves and lines · *owns:* the cells
    - `get_size() -> int`, `get_cell(row: int, col: int) -> Symbol or None`, `place(row: int, col: int, symbol: Symbol) -> bool`, `completes_line(row: int, col: int) -> bool`, `is_full() -> bool`
  - **Game**: Runs turns, decides the result and tracks the status · *owns:* whose turn, status, winner
    - `make_move(player_name: str, row: int, col: int) -> bool`, `get_status() -> GameStatus`, `get_winner() -> Player or None`, `get_current_player() -> Player or None`, `get_cell(row: int, col: int) -> Symbol or None`
- **Relationships:**
  - Game **has** Board (the board is created for the game and dies with it).
  - Game **has** Player (the game owns the two players it was given).
  - Board **uses** Symbol (the cells hold symbols, which are shared values).
  - Player **uses** Symbol. Game **uses** GameStatus.
  - Enums are always **uses**.
- **Call flows:**
  - **Make a move**
    1. `Caller -> Game.make_move(player_name, row, col)`
    2. `Game -> players[current_index]`
    3. `Game -> Board.place(row, col, symbol)`
    4. `Game -> Board.completes_line(row, col)`
    5. `Game -> Board.is_full()`
- **Failure and undo (per flow):**
  - **Make a move:** steps 1 to 3 refuse before anything changes. After step 3 succeeds nothing can fail, so there is nothing to undo.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added.
  - If it were needed later, one lock inside `Game` around `make_move` would be enough, because the board is reachable only through the game.
- **Key decisions:**
  1. **A win is found by checking only the lines through the last move.** *Rejected:* a counter for every row, column and diagonal. It is faster, but it is a second copy of the board that must be updated in step.
  2. **`is_full` is computed by looking at the cells.** *Rejected:* a stored count of filled cells.
  3. **The board refuses bad moves itself.** *Rejected:* the game checking and then writing to the cells.
  4. **`Game` gives read-only access to cells instead of the board.** *Rejected:* a `get_board` method.
  5. **The player is identified by name.** *Rejected:* passing a `Player` object into `make_move`, which a caller could fake.
- **Edge cases:**
  - A move outside the board, or on a filled cell, is refused.
  - A move out of turn is refused.
  - A move after a win or a draw is refused.
  - A win on the last empty cell is a win, not a draw.
  - Two players with the same symbol or name are refused.
  - A board smaller than 3x3 is refused.
- **Extensibility (change → what to add):**
  - A bigger board (4x4, 5x5) → A different `board_size` passed to `Game` *(edits existing code: No. It is data)*
  - A computer opponent → A new class that chooses a cell and calls `make_move` like any player *(edits existing code: No)*
  - Showing the board → A display class that reads `get_cell` *(edits existing code: No)*
- **Follow-up questions:**
  1. **"How would you make it faster for a big board?"** Keep a counter per row, column and diagonal, updated in `place`. It makes the win check constant time, at the price of a second copy of the data that must be kept in step.
  2. **"How would you support K in a row on a larger board?"** `completes_line` would count a run of the same symbol through the cell, up to a `win_length` passed to the board. That is the one place the rule lives.
  3. **"How would you add undo?"** Keep a list of moves in the game and clear the last cell. That is a new rule and a new `clear` method on the board, because the board has no way to empty a cell today.
- **Out of scope (one-line answer if asked):**
  - Telling the player why a move was refused → `make_move` returns only a bool. A reason would be a new return type.
  - K in a row → See follow-up question 2.
  - Undo or move history → See follow-up question 3.
  - More than two players → Requirement 2 says two.
  - Showing the board → `get_cell` gives what a display needs.
  - Concurrent moves → One lock inside `Game` around `make_move` would be enough.

#### 16 · Chess
- **Enums:**
  - **Color**: `WHITE`, `BLACK`. Method: `opposite() -> Color`. *Why:* the two sides of requirement 2, and the other side is needed everywhere
  - **PieceType**: `KING`, `QUEEN`, `ROOK`, `BISHOP`, `KNIGHT`, `PAWN`. *Why:* the six kinds of piece, and the choices for promotion
  - **MoveKind**: `NORMAL`, `CASTLE`, `EN_PASSANT`, `PROMOTION`. *Why:* requirement 7's special moves change more than one square or change the piece, so the board must know which kind it is making
  - **GameStatus**: `ACTIVE`, `CHECKMATE`, `STALEMATE`, `RESIGNED`. *Why:* requirement 8's endings. Checkmate and resignation have a winner, and stalemate is a draw
- **Class details (job · what it owns · key methods):**
  - **Position**: Names one square and finds its neighbours · *owns:* file, rank
    - `get_file() -> int`, `get_rank() -> int`, `offset(file_step: int, rank_step: int) -> Position or None`, `to_notation() -> str`, `from_notation(text: str) -> Position or None`
  - **Move**: Records one move, including what it captured and any special kind · *owns:* an unchangeable record
    - `to_notation() -> str`
  - **Piece**: Holds a piece's colour and whether it has moved, and lists its squares and moves · *owns:* colour, has-moved
    - `get_color() -> Color`, `has_moved() -> bool`, `get_type() -> PieceType`, `moved() -> Piece`, `attacked_squares(board: Board, from_pos: Position) -> list[Position]`, `candidate_moves(board: Board, from_pos: Position) -> list[Move]`
  - **SlidingPiece**: Walks in straight lines until blocked · *owns:* its directions
    - `directions() -> list[(int, int)]`, `attacked_squares(board: Board, from_pos: Position) -> list[Position]`
  - **Rook**
    - `directions()`
  - **Knight**: Jumps in an L shape
    - `attacked_squares(...)`
  - **King**: Steps one square, and castles
    - `attacked_squares(...)`, `candidate_moves(board: Board, from_pos: Position) -> list[Move]`
  - **Pawn**: Moves forward, captures diagonally, en passant and promotion
    - `attacked_squares(...)`, `candidate_moves(board: Board, from_pos: Position) -> list[Move]`
  - **Board**: Holds where every piece stands and finds legal moves · *owns:* the squares, the last move
    - `standard() -> Board`, `get_piece(pos: Position) -> Piece or None`, `get_last_move() -> Move or None`, `positions_of(color: Color) -> list[Position]`, `find_king(color: Color) -> Position`, `is_attacked(pos: Position, by_color: Color) -> bool`, `is_in_check(color: Color) -> bool`, `with_move(move: Move) -> Board`, `legal_moves(color: Color) -> list[Move]`
  - **HistoryEntry**: Records one played move and whether it gave check · *owns:* an unchangeable record
    - `get_move_number() -> int`, `get_move() -> Move`, `gave_check() -> bool`, `to_notation() -> str`
  - **Game**: Runs turns, checks legality, detects the end, and keeps the history · *owns:* the current board, turn, status, history
    - `make_move(player_name: str, from_square: str, to_square: str, promotion: PieceType or None) -> bool`, `resign(player_name: str) -> bool`, `is_in_check() -> bool`, `get_legal_moves(from_square: str) -> list[Move]`, `get_status() -> GameStatus`, `get_winner() -> Color or None`, `get_turn() -> Color`, `get_history() -> list[HistoryEntry]`
- **Relationships:**
  - Game **has** Board (the current board belongs to the game), HistoryEntry (created inside `make_move`).
  - Board **uses** Piece (pieces never change, so the same piece objects are shared by the old board and the new one).
  - Board **uses** Move (the last move is a shared record).
  - HistoryEntry **uses** Move.
  - Move **uses** Position. Board **uses** Position (the map keys).
  - SlidingPiece **extends** Piece. Rook, Bishop, Queen **extend** SlidingPiece. Knight, King, Pawn **extend** Piece.
  - Enums are always **uses**: Piece to Color; Move to Color, PieceType and MoveKind; Game to Color and GameStatus.
- **Call flows:**
  - **Make a move**
    1. `Player -> Game.make_move(name, "e2", "e4", None)`
    2. `Game -> Position.from_notation(...)`
    3. `Game -> Board.legal_moves(turn)`
    4. `Board -> Piece.candidate_moves(board, pos)`
    5. `Board.is_in_check -> Board.is_attacked -> Piece.attacked_squares(...)`
    6. `Game -> Board.with_move(chosen)`
    7. `Game -> Board.is_in_check(opponent)` and `Board.legal_moves(opponent)`
- **Failure and undo (per flow):**
  - **Make a move:** steps 1 to 5 change nothing. After step 6 nothing can fail. Because the board is never changed in place, testing a move never needs an undo.
- **Concurrency:**
  - No requirement mentions concurrent use, so no lock is added.
  - The board never changes in place, so it is safe to share and to read from several threads. Only `Game` changes, and one lock around `make_move` and `resign` would be enough if needed.
- **Key decisions:**
  1. **A move is legal only if your king is safe after it, tested on a copy.** *Rejected:* separate rules for each case, which are easy to get wrong.
  2. **The board never changes in place.** *Rejected:* make and unmake on one board. It is faster, but forgetting to restore one detail (a captured piece, a castling flag) breaks the game.
  3. **`attacked_squares` is separate from `candidate_moves`.** *Rejected:* using the move list to detect check.
  4. **Pieces never change. Moving makes a new piece with `has_moved` set.** *Rejected:* a flag set on the existing piece.
  5. **Sliding is written once in `SlidingPiece`.** *Rejected:* the same walk in the rook, bishop and queen.
- **Edge cases:**
  - A pinned piece can't move off the line to its king.
  - En passant is allowed only on the move right after the double step.
  - Castling is refused if the king or that rook has moved, a square between is occupied, or the king is in, passes through, or lands in check.
  - A pawn reaching the last rank must promote, to a queen if no choice is given. A king or pawn as the choice is refused.
  - Checkmate and stalemate both mean no legal moves. Check decides which.
  - Moving out of turn, moving the other side's piece, or moving after the game ends is refused.
- **Extensibility (change → what to add):**
  - A new kind of piece (a chess variant) → A new subclass of `Piece`, one new `PieceType` value, and one line in the board's private piece-building helper *(edits existing code: The helper gains one line. The rest is new code)*
  - A different start position (Chess960) → A new class method on `Board` beside `standard` *(edits existing code: No logic changes)*
  - Other draw rules (repetition, 50 moves) → A new check in step 7 of `make_move`, and a counter or a list of past positions in `Game` *(edits existing code: Yes, step 7. It is the one place endings are decided)*
  - A computer opponent → A new class that reads `get_legal_moves` and calls `make_move` like a player *(edits existing code: No)*
- **Follow-up questions:**
  1. **"Isn't copying the board for every test slow?"** It costs one small map copy per candidate move, which is fine for a game between people. An engine would switch to make and unmake, or to bitboards, behind the same `Board` methods.
  2. **"How would you add draw by repetition?"** Keep a count of each position in `Game`, using a key built from the board and whose turn it is, and add a check in step 7.
  3. **"How would you add undo?"** Keep the list of past boards in `Game`. Because boards never change, undo is just going back to the previous one.
- **Out of scope (one-line answer if asked):**
  - Draw by agreement, repetition, 50 moves or too little material → Stalemate is the draw covered. See follow-up question 2.
  - Clocks and time limits → No requirement asks for them.
  - Standard algebraic notation (Nf3) → The history uses coordinate notation such as `g1-f3`.
  - Undo → See follow-up question 3.
  - A computer opponent → See the Extensibility list above.
  - Concurrent games → Each `Game` is separate, and boards are safe to share.
