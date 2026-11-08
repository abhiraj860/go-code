# Meeting Room Booking System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a room owns its own calendar of reserved time slots and guards it with a lock, so "check the slot is free, then reserve it" is one step and two bookings can never overlap. A booking is a separate record of the meeting (organizer, title, attendees). It runs its own changes and cancellation, reaching into the rooms in a fixed order, so a change and a cancel on the same meeting can't tangle.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The building should have multiple meeting rooms, each with a capacity and a set of facilities.
2. Users should be able to search for rooms available for a time slot, capacity and facilities.
3. Users should be able to book a room for a time interval.
4. Two bookings for the same room must never overlap.
5. Users should be able to cancel or change a booking.
6. Attendees should be invited and notified about the meeting.
7. Users should be able to view the schedule of a room and of a person.
8. If two users try to book the same slot at the same time, only one should succeed.

**Types and conventions**
- Times are `datetime`, and the caller passes the current time (`now`) in. A time slot includes its start and excludes its end, so a meeting ending at 11:00 and one starting at 11:00 do not overlap.
- People are identified only by a `person_id: str`. No requirement asks for registration, so there is no Person class.
- The capacity of a room must hold the organizer and every attendee.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put`.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Facility (enum) | Names what a room can have | none |
| BookingStatus (enum) | Names whether a booking stands | none |
| NoticeKind (enum) | Names the kinds of message sent to people | none |
| TimeSlot | Describes a start and end and checks overlap | start, end |
| RoomInfo | Gives a read-only picture of one room for search results | a copy of key facts |
| Room | Holds a room's details and its calendar, and guards the calendar | capacity, facilities, reserved slots |
| Booking | Holds one meeting's details, and runs its own changes and cancellation | organizer, title, room, slot, attendees, status |
| MeetingNotifier (interface) | Sends a message about a meeting to a person | none |
| InboxNotifier | Keeps the messages sent to each person | the messages |
| MeetingService | Searches, books, changes, cancels and shows schedules | the rooms, the bookings |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Building, meeting room, capacity, facility, user, time slot, booking, attendee, schedule.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Room, TimeSlot and Booking are classes. Capacity is a number, and facilities are an enum. A schedule is a view, computed from bookings.
2. **Find fixed sets of values.** Facilities, booking status, and the kinds of notice.
   - *Why:* a fixed set is an enum, so a typo like "projecter" cannot exist.
   - *Result:* `Facility`, `BookingStatus`, `NoticeKind`.
3. **Give every fact one owner.** Which slots a room has taken belongs to the Room. The meeting's title, organizer and attendees belong to the Booking.
   - *Why:* if both the room and the booking stored the slot as their own fact, they could disagree. The room keeps only the slot per booking id, and the booking only records which room and slot it holds.
4. **Mark what can be computed and do not store it.** "Is the room free for this slot" is computed from its calendar. A person's schedule is computed from the bookings.
   - *Why:* a stored "free" flag can't be right for every slot at once.
5. **Give every verb to the class whose data changes.** Reserving and releasing a slot change the Room. Changing and cancelling a meeting change the Booking.
6. **Find the action that touches several classes.** A booking needs the room, a capacity check, a new Booking and notices to everyone, in that order. That lives in `MeetingService.book`.
7. **Find the hint phrase for a pattern.** None names a pattern. The notifier is an interface because sending messages is the outside world.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirement 8 names it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A `Booking` or None for booking, a `bool` for cancel and change, lists for search and schedules.

## 4. Pattern Guide

### Used: none of the named patterns
- **Why:** no requirement asks for a swappable rule, for behavior that changes by state, or for unknown parties to subscribe. The attendees are known and listed on the booking.
- **The notifier interface:** sending messages is the outside world, so it sits behind one method that a test can replace.

### Common patterns (not used here)
| Pattern | Signal phrase | Why not used here |
|---|---|---|
| Observer | "several interested parties notified automatically" | Not used as a pattern. The people to tell are exactly the attendees on the booking, so the service tells them directly. |
| Strategy | "a rule that can be replaced" | Not used. Search has one fixed rule. |
| State | "behaves differently depending on its state" | Not used. A booking's status only decides whether a change is allowed. |
| Facade | "a single simple entry point" | Not added as a separate class. `MeetingService` is already the single entry point. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### Facility
- Values: `PROJECTOR`, `WHITEBOARD`, `VIDEO_CONFERENCE`, `TV_SCREEN`
- *Why:* requirement 1 gives rooms a set of facilities, and search matches against them.
- *Extensibility:* a new facility is one new value.

### BookingStatus
- Values: `CONFIRMED`, `CANCELLED`
- *Why:* a cancelled booking stays on record but no longer holds a room.

### NoticeKind
- Values: `INVITED`, `UPDATED`, `CANCELLED`
- *Why:* requirement 6 tells people about a meeting. A new attendee is invited, a staying attendee hears of a change, and a removed attendee or a cancelled meeting gets a cancellation.

## 6. Classes (smallest dependency first)

### TimeSlot
**Why this class exists:** requirement 4 is about overlapping intervals. A start and end, and the overlap rule, belong together in one place.
- Attributes: `start: datetime`, `end: datetime`. *Why:* the interval.
- Constructor: `TimeSlot(start: datetime, end: datetime)`. *Why:* both set once. It refuses `end <= start`.
- Methods:
  - `get_start() -> datetime`, `get_end() -> datetime`. *Why:* read-only views.
  - `overlaps(other: TimeSlot) -> bool`. *Why:* `start < other.end` and `other.start < end`. Touching ends do not overlap.
- *Extensibility:* none needed.

### RoomInfo
**Why this class exists:** search results should show a room without handing out the room itself, which could be changed.
- Attributes: `room_id: str`, `name: str`, `capacity: int`, `facilities: set[Facility]`. *Why:* what a person picking a room needs.
- Constructor: `RoomInfo(room_id: str, name: str, capacity: int, facilities: set[Facility])`. *Why:* all set once.
- Methods: getters for each field.
- *Extensibility:* none needed.

### Room
**Why this class exists:** requirements 4 and 8 need one place where "is this slot free, and if so take it" is decided. The room owns its calendar and the lock around it.
- Attributes:
  - `room_id: str`, `name: str`. *Why:* the identity and display name.
  - `capacity: int`, `facilities: set[Facility]`. *Why:* requirement 1.
  - `reservations: map{str: TimeSlot}` (a booking id to the slot it holds in this room). *Why:* the room's calendar. It holds only ids and slots, because the meeting details belong to the booking.
  - `lock: Lock`. *Why:* makes the overlap check and the change one step.
- Constructor: `Room(room_id: str, name: str, capacity: int, facilities: set[Facility])`. *Why:* refuses empty text or `capacity < 1`. It starts with an empty calendar.
- Methods (each under the lock):
  - `get_room_id() -> str`, `get_capacity() -> int`, `get_info() -> RoomInfo`. *Why:* read-only views.
  - `suits(min_capacity: int, needed: set[Facility]) -> bool`. *Why:* capacity is enough and every needed facility is present.
  - `is_free(slot: TimeSlot) -> bool`. *Why:* no reservation overlaps the slot.
  - `try_reserve(booking_id: str, slot: TimeSlot) -> bool`. *Why:* if no reservation overlaps, store it and return True. Otherwise False. This is requirement 4 and 8 in one step.
  - `replace(booking_id: str, new_slot: TimeSlot) -> bool`. *Why:* moves an existing reservation to a new slot, checking overlap against every other reservation but not its own. False if the booking has no reservation here or the new slot clashes.
  - `release(booking_id: str) -> bool`. *Why:* removes the reservation. False if there was none.
  - `get_booking_ids(window: TimeSlot) -> list[str]`. *Why:* the ids whose slots overlap the window, sorted by start time, for the room's schedule.
- *Extensibility:* none needed.

### Booking
**Why this class exists:** a meeting is its own record: who organized it, what it is, who is invited, and which room and slot it holds. It runs its own changes and cancellation so they can't overlap with each other.
- Attributes:
  - `booking_id: str`, `organizer_id: str`, `title: str`. *Why:* the identity, the owner, and the name of the meeting.
  - `room_id: str`, `slot: TimeSlot`. *Why:* where and when. These record what the room has reserved.
  - `attendee_ids: set[str]`. *Why:* who is invited.
  - `status: BookingStatus`. *Why:* whether it still holds a room.
  - `lock: Lock`. *Why:* a change and a cancel on the same booking run one at a time.
- Constructor: `Booking(booking_id: str, organizer_id: str, title: str, room_id: str, slot: TimeSlot, attendee_ids: set[str])`. *Why:* the room has already reserved the slot. It keeps its own copy of the set, and the status starts `CONFIRMED`. It refuses empty ids or title.
- Methods:
  - Getters for each field, and `get_attendee_ids() -> set[str]` returns a copy.
  - `involves(person_id: str) -> bool`. *Why:* True for the organizer or an attendee.
  - `cancel(requester_id: str, room: Room) -> bool`. *Why:* only the organizer, only while `CONFIRMED`. Under the booking lock it releases the slot in the room and sets `CANCELLED`.
  - `change(requester_id: str, old_room: Room, new_room: Room, new_slot: TimeSlot, new_attendees: set[str]) -> set[str] or None`. *Why:* only the organizer, only while `CONFIRMED`. Under the booking lock: if the room is the same, `old_room.replace(booking_id, new_slot)`. Otherwise `new_room.try_reserve(booking_id, new_slot)`, and if that works, `old_room.release(booking_id)`. Then it updates its room, slot and attendees. It returns the attendees from before the change, so the right people can be told. None if refused, with nothing changed.
- *Why a booking reaches into rooms and not the other way:* the room only knows ids and slots, so dependencies point one way. The lock order is always the booking first, then one room at a time.
- *Extensibility:* none needed.

### MeetingNotifier (interface) and InboxNotifier
**Why they exist:** requirement 6 tells people about meetings. Sending is the outside world, so it sits behind one method.
- `MeetingNotifier` method: `notify(person_id: str, kind: NoticeKind, booking: Booking) -> None`. A failure to send must not undo a booking, so it returns nothing and the service guards the call.
- `InboxNotifier()`: attributes `messages: map{str: list[str]}` (a person id to their messages, oldest first) and `lock: Lock`. `notify` adds a message such as `INVITED: Sync in Nova, 10:00 to 11:00`. `get_messages(person_id: str) -> list[str]` returns a copy. A real email or calendar service would be called here.
- *Extensibility:* a new channel is a new class.

### MeetingService (the manager and only entry point)
**Why this class exists:** booking, changing and cancelling each need a room, a capacity check, a booking and notices to people, in a fixed order, and rooms and bookings need one place to be found.
- Attributes:
  - `rooms: map{str: Room}` (a room id to the Room), `bookings: map{str: Booking}` (a booking id to the Booking). *Why:* the one record of each. Both are thread-safe maps.
  - `notifier: MeetingNotifier`. *Why:* passed in.
  - `counter: AtomicInt`. *Why:* makes booking ids unique when people book at once.
- Constructor: `MeetingService(notifier: MeetingNotifier)`. *Why:* passed in so tests can read the messages. It starts with no rooms or bookings.
- Methods (outer ones take ids and look up once):
  - `add_room(room_id: str, name: str, capacity: int, facilities: set[Facility]) -> bool`. *Why:* setup. False if the id exists or the values are invalid.
  - `search_rooms(slot: TimeSlot, min_capacity: int, needed: set[Facility]) -> list[RoomInfo]`. *Why:* requirement 2. Rooms that suit and are free, smallest capacity first, so big rooms are kept for big meetings.
  - `book(organizer_id: str, room_id: str, slot: TimeSlot, title: str, attendee_ids: set[str], now: datetime) -> Booking or None`. *Why:* requirement 3. Steps in section 9.
  - `change_booking(booking_id: str, requester_id: str, new_room_id: str or None, new_slot: TimeSlot or None, new_attendees: set[str] or None, now: datetime) -> bool`. *Why:* requirement 5. None for a field means keep it.
  - `cancel_booking(booking_id: str, requester_id: str) -> bool`. *Why:* requirement 5.
  - `get_room_schedule(room_id: str, window: TimeSlot) -> list[Booking] or None`. *Why:* requirement 7. None if the room is unknown.
  - `get_person_schedule(person_id: str, window: TimeSlot) -> list[Booking]`. *Why:* requirement 7. Computed from every confirmed booking that involves the person and overlaps the window, sorted by start.
- Private helper: `fits(room: Room, organizer_id: str, attendee_ids: set[str]) -> bool`. *Why:* the organizer plus attendees must not exceed the capacity.
- *Extensibility:* the notifier is passed in, and rooms are data.

## 7. Relationships
- MeetingService **has** Room, Booking (created inside `add_room` and `book`).
- Room **uses** TimeSlot. Booking **uses** TimeSlot, Room (rooms are passed into `cancel` and `change`, and live on).
- MeetingService **uses** MeetingNotifier (passed in).
- InboxNotifier **implements** MeetingNotifier.
- Enums are always **uses**: Room to Facility, Booking to BookingStatus, MeetingNotifier to NoticeKind.
- `RoomInfo` appears only as a return type, so it has no line.

## 8. Interaction

### Layers
1. **Entry point and manager:** MeetingService.
2. **Classes that guard themselves:** Room (its calendar) and Booking (its lifecycle).
3. **Outside world:** MeetingNotifier.
4. **Plain data:** TimeSlot, RoomInfo.

### Call chains
- **Search**
  1. `User -> MeetingService.search_rooms(slot, capacity, facilities)`.
  2. `MeetingService -> Room.suits(...)` and `Room.is_free(slot)` for each room.
- **Book**
  1. `User -> MeetingService.book(...)`.
  2. `MeetingService -> rooms.get(room_id)`, then `fits(...)`: capacity is checked before anything changes.
  3. `MeetingService -> Room.try_reserve(booking_id, slot)`: the room decides.
  4. `MeetingService -> Booking(...)`, then `MeetingNotifier.notify(person, INVITED, booking)` for each attendee.
- **Change**
  1. `User -> MeetingService.change_booking(...)`.
  2. `MeetingService -> Booking.change(requester, old_room, new_room, new_slot, new_attendees)`.
  3. `Booking -> Room.replace` or `Room.try_reserve` then `Room.release`.
  4. `MeetingService -> MeetingNotifier.notify(...)` with `INVITED`, `UPDATED` or `CANCELLED` as fits each person.
- **Cancel**
  1. `User -> MeetingService.cancel_booking(...)`, then `Booking.cancel(requester, room)`, then `Room.release`, then a `CANCELLED` notice to each attendee.

### Who may call whom
- Users call `MeetingService` only.
- `MeetingService` calls `Room`, `Booking` and `MeetingNotifier`.
- `Booking` calls `Room`. `Room` calls no other class.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Book
1. `book(organizer_id, room_id, slot, title, attendee_ids, now)`. If `slot` starts before `now`, return None.
2. `room = rooms.get(room_id)`. If None, or `fits(room, organizer_id, attendee_ids)` is False, return None. *Why before reserving:* a meeting that can't fit must never take a slot.
3. Take the next id from `counter`. `room.try_reserve(booking_id, slot)`. If False (the slot is taken), return None.
4. Build the `Booking`, store it, tell each attendee `INVITED`, and return the booking.
- **Failure and undo:** steps 1 to 3 change nothing if they refuse. A failed notice is skipped and does not undo the booking.

### Change
1. `change_booking(booking_id, requester_id, new_room_id, new_slot, new_attendees, now)`. Look up the booking once. If None, return False.
2. Fill in any None field from the booking's current values. If the new slot starts before `now`, return False. Look up the new room once. If it is unknown, or `fits` is False with the new attendees, return False.
3. `old_attendees = booking.change(requester_id, old_room, new_room, new_slot, new_attendees)`. If None, return False.
4. Tell people. Those who stayed get `UPDATED`, those added get `INVITED`, and those removed get `CANCELLED`. Return True.
- **Failure and undo:** inside `change`, the new slot is reserved before the old one is released. If the new reservation fails, nothing was released and nothing changed. *Why this order:* the meeting always holds at least one slot, and never loses its room in the middle of a move.

### Cancel
1. `cancel_booking(booking_id, requester_id)`. Look up the booking and its room once. If either is None, return False.
2. `booking.cancel(requester_id, room)`. If False, return False.
3. Tell each attendee `CANCELLED`. Return True.
- **Failure:** someone other than the organizer, or a booking already cancelled, is refused with no change.

### Schedules
1. `get_room_schedule` asks the room for the booking ids in the window and returns those bookings in order.
2. `get_person_schedule` filters every confirmed booking that involves the person and overlaps the window.
- **Failure:** an unknown room returns None. A person with no meetings gets an empty list.

### Concurrency (requirement 8)
- **Room:** the lock sits inside `Room`, around every method. `try_reserve` checks for overlap and stores the slot as one step, so two people booking the same slot at the same moment can't both succeed.
- **Booking:** the lock sits inside `Booking`, around `change` and `cancel`, so a change and a cancel on the same meeting run one after the other.
- **Maps and ids:** `rooms` and `bookings` are thread-safe maps, and ids come from an atomic counter.
- **Lock order:** a booking lock is always taken before a room lock, and a booking never holds two room locks at once (it reserves the new room, lets go, then releases the old one). `book` takes only a room lock. So no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| More rooms | Calls to `add_room` | No. It is data. |
| A new facility | One new `Facility` value | No logic changes. |
| Sending invitations by email or calendar | A new class implementing `MeetingNotifier` | No. |
| Searching by floor or building | A new field on `Room` and `RoomInfo`, and one more check in `suits` | Yes, a small addition to `suits`. |

## 11. Interview Notes

### Key decisions
1. **Each room guards its own calendar with a lock.**
   - *Why:* only bookings for the same room compete, and "check and reserve" is one step.
   - *Rejected:* one lock for the whole system, which would make every booking wait for every other.
2. **The room stores only booking ids and slots, and the booking stores the meeting details.**
   - *Why:* the room's job (no overlaps) needs nothing else, and the meeting details change without touching the room.
   - *Rejected:* rooms holding full booking objects, which makes a change touch two places.
3. **A change reserves the new slot before releasing the old one.**
   - *Why:* if the new slot is taken, the meeting keeps its old one.
   - *Rejected:* releasing first, which could leave the meeting with no room if someone takes it in between.
4. **A person's schedule is computed, not stored.**
   - *Why:* one fact in one place.
   - *Rejected:* a per-person index kept in step with every booking change. It is faster, but it is a second copy.
5. **Search returns the smallest suitable room first.**
   - *Why:* big rooms stay free for big meetings.
   - *Rejected:* any order, which wastes big rooms on small meetings.

### Edge cases handled
- A meeting ending at 11:00 and one starting at 11:00 do not overlap.
- Booking a slot in the past is refused.
- A meeting with more people than the room holds is refused.
- Changing to a slot that is taken leaves the booking as it was.
- Only the organizer can change or cancel.
- A cancelled booking can't be changed or cancelled again.
- Removed attendees are told the meeting is cancelled for them, and new ones are invited.
- Two people booking the same slot at the same moment: one wins.

### Likely follow-up questions
1. **"Isn't checking every reservation slow for a busy room?"** Keep each room's slots sorted by start time (or in an interval tree), so the overlap check looks only at neighbours.
2. **"How would you handle recurring meetings?"** Expand a recurring meeting into one booking per occurrence, reserved together, and refuse the series if any occurrence clashes. It is a new rule on top of `try_reserve`.
3. **"How would you show a person's schedule quickly?"** Keep an index from person to booking ids, updated inside the same steps that book, change and cancel. It is a deliberate second copy kept for speed.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| People accepting or declining invitations | Not required. It would be a response per attendee on the booking. |
| Recurring meetings | See follow-up question 2. |
| Checking that people are free | Only rooms are checked for overlap. Person clashes would use the person schedule before booking. |
| Room approval or restricted rooms | Not required. |
| Real email or calendar invites | `MeetingNotifier.notify` is the one place they go. |
| Fast schedule lookups | See follow-up questions 1 and 3. |

## 13. Requirements Met
1. Rooms with capacity and facilities: `Room`, `Facility`, `MeetingService.add_room`.
2. Search by slot, capacity and facilities: `MeetingService.search_rooms` with `Room.suits` and `is_free`.
3. Book a room for an interval: `MeetingService.book` and `Room.try_reserve`.
4. No overlaps: `TimeSlot.overlaps` and the check inside `Room.try_reserve` and `replace`.
5. Cancel or change: `Booking.cancel` and `change`, through `MeetingService`.
6. Invite and notify attendees: `MeetingNotifier` with `InboxNotifier`, and `NoticeKind`.
7. Room and person schedules: `MeetingService.get_room_schedule` and `get_person_schedule`.
8. Only one of two simultaneous bookings succeeds: the lock inside `Room`.

## 14. Requirement Fit Check

**Setup used in every example:** room `R1` "Orion" (capacity 4, whiteboard), `R2` "Atlas" (capacity 10, projector and video conference), `R3` "Nova" (capacity 6, projector). Now is 2026-11-09 08:00. The slot `10–11` means 10:00 to 11:00 that day.

1. **Rooms with capacity and facilities**
   - *Success:* `add_room` for R1, R2 and R3 all return True.
   - *Failure:* `add_room("R1", ...)` again returns False. `add_room("R4", "Tiny", 0, ...)` returns False.
2. **Search by slot, capacity and facilities**
   - *Success:* `search_rooms(10–11, 5, {PROJECTOR})` returns [Nova, Atlas], smallest first. After Nova is booked for 10–11, the same search returns [Atlas].
   - *Failure:* `search_rooms(10–11, 20, {})` returns an empty list.
3. **Book a room**
   - *Success:* `book("alice", "R3", 10–11, "Sync", {"bob", "carol"}, now)` returns booking `B1`.
   - *Failure:* booking Nova for alice with 7 attendees returns None (8 people in a room for 6). Booking a slot that started at 07:00 returns None.
4. **No overlaps**
   - *Success:* `book("dave", "R3", 11–12, ...)` works, because it only touches B1's end.
   - *Failure:* `book("dave", "R3", 10:30–11:30, ...)` returns None because it overlaps B1.
5. **Cancel or change**
   - *Success:* `change_booking("B1", "alice", None, 14–15, None, now)` moves it within Nova. `change_booking("B1", "alice", "R2", None, None, now)` moves it to Atlas, and Nova's slot is free again. `cancel_booking("B1", "alice")` returns True.
   - *Failure:* `cancel_booking("B1", "bob")` returns False because bob is not the organizer. Changing B1 to a Nova slot that dave holds returns False, and B1 keeps its old room and slot.
6. **Invite and notify**
   - *Success:* after booking, bob and carol each have an `INVITED` message. `change_booking("B1", "alice", None, None, {"bob", "erin"}, now)` sends bob `UPDATED`, erin `INVITED`, and carol `CANCELLED`.
   - *Failure:* if the notifier fails for one person, the booking still stands and the others are still told.
7. **Room and person schedules**
   - *Success:* `get_room_schedule("R3", the whole day)` returns [B1 at 10–11]. `get_person_schedule("bob", the whole day)` returns [B1].
   - *Failure:* `get_room_schedule("R9", ...)` returns None. After B1 is cancelled, it no longer appears in either schedule.
8. **Only one of two simultaneous bookings succeeds**
   - *Success:* alice and dave both call `book` for Nova at 10–11 at the same moment. Both reach `try_reserve`, and the room's lock lets one store the slot.
   - *Failure to avoid:* the other gets False from `try_reserve`, so `book` returns None. Nova never holds two meetings at 10–11.
