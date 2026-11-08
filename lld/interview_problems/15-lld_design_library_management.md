# Library / Bookstore Management: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a book (a title) and a copy (one physical object) are two different things. The title owns the copies and the waiting list, each copy guards its own status, and a return always goes through the title, so the copy is either freed or handed to the next person waiting, and that person is told. Permissions are a table of data, checked in one place.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The system should allow adding, updating and removing books, with multiple copies of each book.
2. Users should be able to search books by title, author or subject.
3. Members should be able to register and hold a membership.
4. A member should be able to borrow books up to a limit, with a due date for each.
5. A member should be able to return books, and a late fee should be charged after the due date.
6. A member should be able to reserve a book that is currently borrowed and be notified when it is available.
7. Each copy of a book should track whether it is available, borrowed, reserved or lost.
8. Librarians and members should have different permissions.

**Types and conventions**
- Dates are `date`, passed in by the caller as `today`, so the design never reads a clock. Money is `int`, in the smallest currency unit.
- A "book" is a title identified by an ISBN. A "copy" is one physical object of that book.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- No locks are used, because no requirement mentions concurrent use.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Role (enum) | Names the two kinds of user | none |
| Action (enum) | Names the things a user may be allowed to do | none |
| CopyStatus (enum) | Names the four conditions of a copy | none |
| BookCopy | Is one physical copy and guards its own status | status, who it is held for |
| ReservationObserver (interface) | Reacts when a reserved book is ready | none |
| MemberNotifier | Keeps the messages sent to each member | the messages |
| Book | Owns its details, its copies and its waiting list | details, copies, waiting list |
| SearchCriterion (interface) | Decides whether a book matches a search | none |
| TitleContains | Matches on part of the title | the text |
| AuthorContains | Matches on part of the author | the text |
| SubjectContains | Matches on part of the subject | the text |
| Membership | Holds when a membership ends and how many books it allows | expiry date, loan limit |
| User | Is a librarian or a member | role, membership |
| PermissionPolicy | Says which role may do which action | the permission table |
| LateFeePolicy | Works out a late fee | fee per day, grace days |
| Loan | Records one borrowing | due date, return date, late fee |
| LibraryService | Runs each action in the right order and holds users, books and loans | users, books, loans |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Book, copy, member, librarian, membership, loan, due date, late fee, reservation, status, permission.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Book, BookCopy, User, Membership, Loan are classes. Due date and fee are attributes. A reservation is an entry in a book's waiting list, not a class.
2. **Find fixed sets of values.** The two roles, the allowed actions, and the four copy conditions.
   - *Why:* a fixed set is an enum, so a typo like "borowed" cannot exist.
   - *Result:* `Role`, `Action`, `CopyStatus`.
3. **Give every fact one owner.** A copy's status belongs to the copy. The copies and the waiting list belong to the Book. The expiry and limit belong to the Membership. The return date and fee belong to the Loan.
   - *Why:* two copies of one fact will eventually disagree. A member's list of loans is not stored on the member, because the loans already say who holds what.
4. **Mark what can be computed and do not store it.** "Is the membership active", "how many books does this member hold", "how many copies are available", and a late fee are all computed.
   - *Why:* a stored loan count beside the loans would go wrong the moment one is returned and the count is forgotten.
5. **Give every verb to the class whose data changes.** Borrowing or holding a copy changes the BookCopy. Waiting lists change the Book. Closing a loan changes the Loan.
6. **Find the action that touches several classes.** Borrowing needs the user, the permission, the book and a new loan. Returning needs the loan, the fee, and the book (to serve the waiting list). That order lives in `LibraryService`.
7. **Find the hint phrases for patterns.** "Be notified when it is available" is Observer. "Search by title, author or subject" is a swappable criterion, which is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Fix the outputs.** A `Loan` or None for borrowing, an `int` fee or None for returning, a `bool` for other commands.

## 4. Pattern Guide

### Used: Observer (reservations)
- **Meaning:** an object keeps a list of watchers and tells every one of them when something happens, without knowing what they do.
- **Hint phrase:** "be notified when it is available".
- **Without it:** the return code would have to call the message code directly, and sending a different kind of notice would mean editing it.
- **When NOT to use it:** when only one party ever cares and how never changes.

### Used: Strategy (search)
- **Meaning:** a way of doing something is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrase:** "search books by title, author or subject".
- **Without it:** one search method would have an `if` on the field, and a new field would mean editing it.
- **When NOT to use it:** when there is one fixed way to search.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A copy's status only decides which method is allowed, and each method checks it. |
| Factory | "create different kinds of objects by type" | Not used. |
| Facade | "a single simple entry point" | Not added as a separate class. `LibraryService` is already the single entry point. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### Role
- Values: `LIBRARIAN`, `MEMBER`
- *Why:* requirement 8 gives two kinds of user different permissions.

### Action
- Values: `MANAGE_BOOKS`, `BORROW`, `RETURN`, `RESERVE`, `SEARCH`
- *Why:* the things a user might be allowed to do. A permission is a role and an action, so adding a permission is data and not code.

### CopyStatus
- Values: `AVAILABLE`, `BORROWED`, `RESERVED`, `LOST`
- *Why:* exactly the four conditions named in requirement 7. `RESERVED` means held for a member who is waiting, and ready to be picked up.

## 6. Classes (smallest dependency first)

### BookCopy
**Why this class exists:** requirement 7 tracks each copy separately. A copy owns its status, so no other class can put it in a bad state.
- Attributes:
  - `copy_id: str`. *Why:* the identity of this one physical copy.
  - `status: CopyStatus`. *Why:* the single fact about the copy.
  - `held_for: str or None`. *Why:* the member a `RESERVED` copy is waiting for. None otherwise.
- Constructor: `BookCopy(copy_id: str)`. *Why:* a new copy starts `AVAILABLE` and held for nobody. It refuses an empty id.
- Methods:
  - `get_copy_id() -> str`, `get_status() -> CopyStatus`, `get_held_for() -> str or None`. *Why:* read-only views.
  - `borrow(member_id: str) -> bool`. *Why:* `AVAILABLE` to `BORROWED`, or `RESERVED` to `BORROWED` only if it is held for that same member. False otherwise. This is what stops another member taking a reserved copy.
  - `return_copy() -> bool`. *Why:* `BORROWED` to `AVAILABLE`. False otherwise.
  - `hold_for(member_id: str) -> bool`. *Why:* `AVAILABLE` to `RESERVED`, held for that member. False otherwise.
  - `release_hold() -> bool`. *Why:* `RESERVED` to `AVAILABLE`, held for nobody. False otherwise.
  - `mark_lost() -> bool`. *Why:* `AVAILABLE` to `LOST`. False otherwise.
- *Why there is no setter:* each method checks the current status first, so illegal jumps are refused.
- *Extensibility:* none needed.

### ReservationObserver (interface)
**Why it is an interface:** requirement 6 notifies a member when a book is ready, and the way of telling may change, so the book depends on one method and not on any specific notifier.
- Methods: `on_book_available(member_id: str, isbn: str, copy_id: str) -> None`. *Why these inputs:* plain ids, so a watcher never holds the book. A watcher that fails must not stop a return, so the book guards each call.
- *Extensibility:* a new notifier is a new class implementing this method.

### MemberNotifier (implements ReservationObserver)
**Why this class exists:** the first notifier: it keeps the messages sent to each member.
- Attributes: `messages: map{str: list[str]}` (a member id to the messages sent to that member, oldest first). *Why:* the one record of what each member was told. A real email or SMS service would be called here.
- Constructor: `MemberNotifier()`. *Why:* it starts with an empty map.
- Methods:
  - `on_book_available(member_id: str, isbn: str, copy_id: str) -> None`. *Why:* adds a message such as `Book 111 is ready for you`.
  - `get_messages(member_id: str) -> list[str]`. *Why:* read-only view. A copy. Empty if none.
- *Extensibility:* none needed.

### Book
**Why this class exists:** requirement 1 manages books with several copies, and requirement 6 keeps a waiting list per book. The book owns both, so a returned copy can be given to the next person in line.
- Attributes:
  - `isbn: str`. *Why:* the identity of the title.
  - `title: str`, `author: str`, `subject: str`. *Why:* the details that searches match against.
  - `copies: map{str: BookCopy}` (a copy id to the BookCopy). *Why:* the one record of the copies, found by id.
  - `waiting: list[str]` (the member ids waiting for a copy, first in line first). *Why:* the reservation queue. It lives with the book because it is about the title and not one copy.
  - `observers: list[ReservationObserver]`. *Why:* who to tell when a copy is held for someone. They are passed in.
  - `next_copy_number: int`. *Why:* gives each new copy a unique id such as `isbn-1`.
- Constructor: `Book(isbn: str, title: str, author: str, subject: str, copy_count: int, observers: list[ReservationObserver])`. *Why:* a book starts with its first copies, all `AVAILABLE`, and an empty waiting list. It keeps its own copy of the observer list. It refuses empty text or `copy_count < 1`.
- Methods:
  - `get_isbn() -> str`, `get_title() -> str`, `get_author() -> str`, `get_subject() -> str`. *Why:* read-only views.
  - `update_details(title: str, author: str, subject: str) -> bool`. *Why:* False if any is empty. The ISBN never changes.
  - `add_copies(count: int) -> bool`. *Why:* False if `count < 1`. Each new copy is first offered to the waiting list: if someone is waiting, the copy is held for them and they are told. Otherwise it is `AVAILABLE`.
  - `get_copy(copy_id: str) -> BookCopy or None`. *Why:* the lookup by id.
  - `count_available() -> int`. *Why:* computed by counting `AVAILABLE` copies.
  - `has_active_copies() -> bool`. *Why:* computed. True if any copy is `BORROWED` or `RESERVED`, so a book in use can't be removed.
  - `find_borrowable_copy(member_id: str) -> BookCopy or None`. *Why:* a copy held for this member first, otherwise an `AVAILABLE` copy. None if neither exists.
  - `add_reservation(member_id: str) -> bool`. *Why:* False if the member is already waiting, or if a copy is `AVAILABLE` (they should just borrow it), or if the book has no copies that could ever come back. Otherwise they join the end of the waiting list.
  - `cancel_reservation(member_id: str) -> bool`. *Why:* removes them from the waiting list. If a copy is already held for them, it is released and offered to the next person waiting. False if they had neither.
  - `return_copy(copy_id: str) -> bool`. *Why:* `copy.return_copy()`, then the copy is offered to the first person waiting. If someone is waiting, the copy is held for them and they are told. Otherwise it stays `AVAILABLE`. False if the copy is unknown or was not borrowed.
  - `mark_lost(copy_id: str) -> bool`. *Why:* asks the copy. False if unknown or not available.
- *Extensibility:* none needed.

### SearchCriterion (interface)
**Why it is an interface:** requirement 2 searches by three different fields, so the search depends on one method and not on any specific field.
- Methods: `matches(book: Book) -> bool`. *Why a `bool`:* the search only needs yes or no per book.
- *Extensibility:* a new field (publisher) is a new class implementing this method.

### TitleContains, AuthorContains, SubjectContains (implement SearchCriterion)
**Why these classes exist:** one class per field named in requirement 2. They are three classes because each reads a different field.
- Attributes: `text: str`. *Why:* the words to look for.
- Constructor: `TitleContains(text: str)`, `AuthorContains(text: str)`, `SubjectContains(text: str)`. *Why:* set once. Each refuses an empty text.
- Methods: `matches(book: Book) -> bool`. *Why:* True if the text appears (ignoring case) in the book's title, author or subject respectively.
- *Extensibility:* none needed.

### Membership
**Why this class exists:** requirement 3 gives a member a membership, and requirement 4 limits how many books it allows. The two belong to the membership, not to the person.
- Attributes: `expires_on: date`, `max_loans: int`. *Why:* when it ends, and how many books may be out at once.
- Constructor: `Membership(expires_on: date, max_loans: int)`. *Why:* both set once. It refuses `max_loans < 1`.
- Methods:
  - `get_expires_on() -> date`, `get_max_loans() -> int`. *Why:* read-only views.
  - `is_active(today: date) -> bool`. *Why:* computed as `today <= expires_on`, so there is no stored flag to forget.
- *Extensibility:* different limits are data.

### User
**Why this class exists:** requirements 3 and 8 need people with a role. A librarian has no membership, and a member does.
- Attributes: `user_id: str`, `name: str`, `role: Role`, `membership: Membership or None`. *Why:* the identity, the display name, what they may do, and the membership (None for a librarian).
- Constructor: `User(user_id: str, name: str, role: Role, membership: Membership or None)`. *Why:* all set once. It refuses empty values, a `MEMBER` with no membership, and a `LIBRARIAN` with one.
- Methods: `get_user_id() -> str`, `get_role() -> Role`, `get_membership() -> Membership or None`. *Why:* read-only views.
- *Extensibility:* none needed.

### PermissionPolicy
**Why this class exists:** requirement 8 says the two roles may do different things. A table of data in one class means one place decides, and a change is not a code change.
- Attributes: `grants: map{Role: set[Action]}` (a role to the actions it is allowed). *Why:* the single permission table.
- Constructor: `PermissionPolicy(grants: map{Role: set[Action]})`. *Why:* it keeps its own copy and refuses a map with no entry for a role.
- Methods: `allows(role: Role, action: Action) -> bool`. *Why:* True if the role's set includes the action.
- *Extensibility:* a new permission is a new entry in the table.

### LateFeePolicy
**Why this class exists:** the late fee rule is two numbers and a formula, and it belongs in one place.
- Attributes: `fee_per_day: int`, `grace_days: int`. *Why:* the price of each late day, and the days allowed before it starts.
- Constructor: `LateFeePolicy(fee_per_day: int, grace_days: int)`. *Why:* both set once. It refuses a negative value.
- Methods: `calculate(due_on: date, returned_on: date) -> int`. *Why:* `max(0, days between due_on and returned_on - grace_days) * fee_per_day`. A return on or before the due date costs 0.
- *Extensibility:* different numbers are data.

### Loan
**Why this class exists:** requirements 4 and 5 are about one borrowing: a due date, a return and a fee. A loan owns those, and who holds what is read from the loans.
- Attributes:
  - `loan_id: str`, `member_id: str`. *Why:* the identity and the borrower.
  - `book: Book`, `copy_id: str`. *Why:* which title and which copy. Holding the book lets a return reach it without another lookup.
  - `borrowed_on: date`, `due_on: date`. *Why:* the dates that fix the fee.
  - `returned_on: date or None`. *Why:* None while the book is out. "Is it still out" is computed from this.
  - `late_fee: int`. *Why:* the fee charged on return. 0 until then.
- Constructor: `Loan(loan_id: str, member_id: str, book: Book, copy_id: str, borrowed_on: date, due_on: date)`. *Why:* all set at creation, with `returned_on` None and `late_fee` 0. It refuses `due_on` before `borrowed_on`.
- Methods:
  - `get_loan_id() -> str`, `get_member_id() -> str`, `get_book() -> Book`, `get_copy_id() -> str`, `get_due_on() -> date`, `get_late_fee() -> int`. *Why:* read-only views.
  - `is_open() -> bool`. *Why:* computed as `returned_on is None`.
  - `close(returned_on: date, policy: LateFeePolicy) -> int or None`. *Why:* None if the loan is already closed or `returned_on` is before `borrowed_on`. Otherwise it records the date, charges `policy.calculate(due_on, returned_on)`, and returns that fee.
- *Extensibility:* none needed.

### LibraryService (the manager and only entry point)
**Why this class exists:** borrowing and returning touch the user, the permission, the book and a loan, in a fixed order. No single class can do that, and users, books and loans need one place to be found.
- Attributes:
  - `users: map{str: User}` (a user id to the User), `books: map{str: Book}` (an ISBN to the Book), `loans: map{str: Loan}` (a loan id to the Loan). *Why:* the one record of each.
  - `permissions: PermissionPolicy`, `fee_policy: LateFeePolicy`. *Why:* the permission table and the fee rule, both passed in.
  - `observers: list[ReservationObserver]`. *Why:* handed to each new Book.
  - `loan_days: int`, `membership_days: int`, `default_max_loans: int`. *Why:* how long a loan lasts, how long a new membership lasts, and the limit given to a new member. They are data, so they can change without code.
  - `next_loan_number: int`. *Why:* makes loan ids unique.
- Constructor: `LibraryService(permissions: PermissionPolicy, fee_policy: LateFeePolicy, observers: list[ReservationObserver], loan_days: int, membership_days: int, default_max_loans: int)`. *Why:* the rules are passed in so tests can use fakes. It refuses non-positive numbers and starts empty.
- Methods (outer ones take ids and look up once):
  - `add_librarian(user_id: str, name: str) -> User or None`. *Why:* setup. None if the id exists.
  - `register_member(user_id: str, name: str, today: date) -> User or None`. *Why:* requirement 3. It builds a `Membership` that ends `membership_days` after `today`, with `default_max_loans`. None if the id exists.
  - `add_book(actor_id: str, isbn: str, title: str, author: str, subject: str, copy_count: int) -> Book or None`, `update_book(actor_id: str, isbn: str, title: str, author: str, subject: str) -> bool`, `add_copies(actor_id: str, isbn: str, count: int) -> bool`, `remove_book(actor_id: str, isbn: str) -> bool`, `mark_copy_lost(actor_id: str, isbn: str, copy_id: str) -> bool`. *Why:* requirement 1 and 7. Each needs the actor to hold `MANAGE_BOOKS`. `remove_book` is refused while any copy is borrowed or reserved.
  - `search(actor_id: str, criterion: SearchCriterion) -> list[Book]`. *Why:* requirement 2. The actor needs `SEARCH`. Empty list if nothing matches or the actor is not allowed.
  - `borrow_book(member_id: str, isbn: str, today: date) -> Loan or None`. *Why:* requirement 4. Steps in section 9.
  - `return_book(actor_id: str, loan_id: str, today: date) -> int or None`. *Why:* requirement 5. It returns the late fee, or None if refused.
  - `reserve_book(member_id: str, isbn: str, today: date) -> bool`, `cancel_reservation(member_id: str, isbn: str) -> bool`. *Why:* requirement 6.
  - `get_fees_due(member_id: str) -> int`. *Why:* computed as the sum of `late_fee` over that member's loans.
- *Extensibility:* the policies and the observers are passed in. New permissions are table entries.

## 7. Relationships
- LibraryService **has** User, Book, Loan (created inside the add, register and borrow methods).
- Book **has** BookCopy (the copies die with the book).
- User **has** Membership (a membership belongs to one member).
- Book **uses** ReservationObserver (passed in).
- Loan **uses** Book (the book lives on after the loan).
- LibraryService **uses** PermissionPolicy, LateFeePolicy, ReservationObserver (passed in).
- TitleContains, AuthorContains, SubjectContains **implement** SearchCriterion. MemberNotifier **implements** ReservationObserver.
- Enums are always **uses**: BookCopy to CopyStatus, User to Role, PermissionPolicy to Role and Action.
- `SearchCriterion` appears only as a method input, so no attribute holds it and it has no line.

## 8. Interaction

### Layers
1. **Entry point and manager:** LibraryService. The only class callers use.
2. **Domain classes that guard themselves:** Book, BookCopy, Loan.
3. **Rules passed in:** PermissionPolicy, LateFeePolicy, SearchCriterion, ReservationObserver.
4. **Plain data:** User, Membership.

### Call chains
- **Search**
  1. `User -> LibraryService.search(actor_id, criterion)`.
  2. `LibraryService -> PermissionPolicy.allows(role, SEARCH)`, then `SearchCriterion.matches(book)` for each book.
- **Borrow**
  1. `Member -> LibraryService.borrow_book(member_id, isbn, today)`.
  2. `LibraryService -> PermissionPolicy.allows(role, BORROW)`, `Membership.is_active(today)`, and a count of the member's open loans against `max_loans`.
  3. `LibraryService -> Book.find_borrowable_copy(member_id)`, then `BookCopy.borrow(member_id)`: the copy decides.
  4. `LibraryService -> Loan(...)`: created last, only once a copy is taken.
- **Return**
  1. `Member -> LibraryService.return_book(actor_id, loan_id, today)`.
  2. `LibraryService -> Loan.close(today, fee_policy)`: the fee is worked out.
  3. `LibraryService -> Book.return_copy(copy_id)`, then `BookCopy.return_copy()`.
  4. `Book -> BookCopy.hold_for(next member)` and `ReservationObserver.on_book_available(...)`: the next person in line is served and told.
- **Reserve**
  1. `Member -> LibraryService.reserve_book(...)`, then `Book.add_reservation(member_id)`.

### Who may call whom
- Users call `LibraryService` only.
- `LibraryService` calls `PermissionPolicy`, `Book`, `Loan`, `LateFeePolicy`.
- `Book` calls `BookCopy` and the observers. `Loan` calls `LateFeePolicy`.
- Observers must never call back into `Book`. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Manage books
1. `add_book(actor_id, isbn, title, author, subject, copy_count)`. Look up the actor once. If the actor is unknown or `permissions.allows(role, MANAGE_BOOKS)` is False, return None. If the ISBN exists, return None. Build the `Book`, store it, return it.
2. `update_book`, `add_copies`, `mark_copy_lost` look up the book once and ask it. `remove_book` is refused if `book.has_active_copies()`.
- **Failure:** a refusal changes nothing.

### Register a member
1. `register_member(user_id, name, today)`. If the id exists, return None. Build a `Membership(today + membership_days, default_max_loans)` and the `User` with role `MEMBER`.
- **Failure:** None, and nothing is created.

### Search
1. `search(actor_id, criterion)`. If the actor may not `SEARCH`, return an empty list.
2. Return the books for which `criterion.matches(book)` is True, sorted by title.
- **Failure:** no match returns an empty list. Nothing changes.

### Borrow
1. `borrow_book(member_id: str, isbn: str, today: date)`. `member = users.get(member_id)`. If None, or the role may not `BORROW`, return None.
2. If the membership is not active on `today`, return None.
3. Count the member's open loans. If the count is at `max_loans`, return None.
4. `book = books.get(isbn)`. If None, return None.
5. `copy = book.find_borrowable_copy(member_id)`. If None, return None. *Why this finds a reserved copy first:* a copy held for this member is theirs.
6. `copy.borrow(member_id)`. If False, return None.
7. Create the `Loan` with `due_on = today + loan_days`, store it, and return it.
- **Failure and undo:** steps 1 to 5 change nothing. The copy is changed only in step 6, and nothing after it can fail, so there is nothing to undo.

### Return
1. `return_book(actor_id: str, loan_id: str, today: date)`. `loan = loans.get(loan_id)`. If None, return None.
2. The actor must be allowed `RETURN`. A member may only return their own loan, so the actor must be the loan's member unless they are a librarian.
3. `fee: int or None = loan.close(today, fee_policy)`. If None, return None. *Why first:* closing is the step that can refuse, so it comes before anything is freed.
4. `loan.get_book().return_copy(loan.get_copy_id())`. The copy is freed, or held for the next person waiting, who is told.
5. Return the fee.
- **Failure and undo:** a closed loan or a wrong actor is refused with no change. If the book refuses step 4 (it should not), the loan is already closed, and the fee stays recorded.

### Reserve and cancel a reservation
1. `reserve_book(member_id, isbn, today)`. The member must be allowed `RESERVE` and have an active membership. `book.add_reservation(member_id)` then enforces that no copy is available.
2. `cancel_reservation(member_id, isbn)` asks `book.cancel_reservation(member_id)`, which also frees a held copy for the next person.
- **Failure:** False, with nothing changed. Reserving a book that has an available copy is refused, because the member should just borrow it.

### Concurrency
- No requirement mentions concurrent use, so no lock is added.
- If it were needed later, one lock inside `Book` around its copies and waiting list would protect the shared facts, and a lock inside `LibraryService` around the loan count and the borrow step would stop a member passing the limit.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new search field (publisher, year) | A new class implementing `SearchCriterion` | No. |
| A new kind of notice (email, SMS) | A new class implementing `ReservationObserver`, passed to the service | No. |
| A new permission or a new role rule | A new entry in the `grants` table passed to `PermissionPolicy` | No. It is data. |
| Different fees, loan length or limits | Different numbers in `LateFeePolicy` and the service constructor | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **A book and a copy are separate classes.**
   - *Why:* a title has many physical copies, and each has its own status.
   - *Rejected:* one class with a copy count. It could not say which copy is lost or reserved.
2. **The return goes through `Book`, which serves the waiting list.**
   - *Why:* one place frees a copy or holds it for the next person, so a reserved copy can never be taken by someone else.
   - *Rejected:* the service freeing the copy and then checking the queue. A direct call would skip the queue.
3. **A member's loans are not stored on the member.**
   - *Why:* the loans already say who holds what, so the count is computed and cannot disagree.
   - *Rejected:* a loan list on `User` kept in step with the loans.
4. **A permission is a role and an action in a table.**
   - *Why:* one place decides, and a change is data.
   - *Rejected:* `if role == LIBRARIAN` checks scattered through the service.
5. **The late fee is calculated by a policy and stored once on the loan.**
   - *Why:* the fee a member was charged stays the same even if the rate changes later.
   - *Rejected:* recomputing the fee each time it is read.

### Edge cases handled
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

### Likely follow-up questions
1. **"What happens if a reserved copy is never picked up?"** Add an expiry date to the hold and a sweep that releases it to the next person. It is not built here because no requirement asks for it.
2. **"What if a borrowed copy is lost?"** Add a lost-while-borrowed path: close the loan with a replacement fee and set the copy to `LOST`. Today only an available copy can be marked lost.
3. **"How would you stop a member borrowing the same book twice?"** Check the member's open loans for that ISBN in step 3 of the borrow. It is a new rule, because the requirements don't ask for it.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| A hold that expires | See follow-up question 1. |
| A borrowed copy reported lost | See follow-up question 2. |
| Renewing a loan or a membership | Not required. A renewal would change a loan's due date or a membership's expiry. |
| Paying a late fee | A fee is charged and recorded. Payment is not a requirement. |
| Selling books (the bookstore side) | The requirements describe lending. Selling would be a new set of classes. |
| Blocking a member who owes fees | Not required. It would be a check in step 3 of the borrow. |
| Concurrent users | Section 9 gives the lock placement. |
| Real email or SMS notices | The notifier is the one place a real call goes. |

## 13. Requirements Met
1. Add, update and remove books with copies: `LibraryService.add_book`, `update_book`, `add_copies`, `remove_book`, with `Book` and `BookCopy`.
2. Search by title, author or subject: `SearchCriterion` with `TitleContains`, `AuthorContains`, `SubjectContains`, and `LibraryService.search`.
3. Register and hold a membership: `LibraryService.register_member`, `User` and `Membership`.
4. Borrow up to a limit with a due date: `LibraryService.borrow_book`, `Membership.max_loans` and `Loan.due_on`.
5. Return with a late fee: `LibraryService.return_book`, `Loan.close` and `LateFeePolicy`.
6. Reserve and be notified: `Book.add_reservation`, `Book.return_copy`, `ReservationObserver` and `MemberNotifier`.
7. Track each copy's status: `BookCopy` with `CopyStatus` and its guarded methods.
8. Different permissions: `PermissionPolicy`, `Role` and `Action`.

## 14. Requirement Fit Check

**Setup used in every example:** the loan lasts 14 days, the fee is 10 per day with no grace days, and a new membership lasts 365 days with a limit of 2 books. Librarian `L1`. Members `M1`, `M2` and `M3`. Book `B1` "Clean Code" by Robert Martin, subject Programming, with 2 copies. Book `B2` "Dune" by Frank Herbert, subject Fiction, with 1 copy. Today is 2026-11-08.

1. **Add, update and remove books**
   - *Success:* `add_book("L1", "B1", ..., 2)` returns the Book with copies `B1-1` and `B1-2`. `add_copies("L1", "B1", 1)` adds `B1-3`.
   - *Failure:* `add_book("M1", ...)` returns None because a member lacks `MANAGE_BOOKS`. `remove_book("L1", "B2")` returns False while B2's only copy is borrowed.
2. **Search by title, author or subject**
   - *Success:* `search("M1", TitleContains("clean"))` returns [Clean Code]. `AuthorContains("herbert")` and `SubjectContains("fiction")` both return [Dune].
   - *Failure:* `TitleContains("xyz")` returns an empty list. `TitleContains("")` is refused at construction.
3. **Register and hold a membership**
   - *Success:* `register_member("M1", "Mia", 2026-11-08)` returns a member whose membership ends 2027-11-08, with a limit of 2.
   - *Failure:* `register_member("M1", "Other", ...)` returns None because the id exists. A member whose membership ended before today can't borrow.
4. **Borrow up to a limit with a due date**
   - *Success:* `borrow_book("M1", "B1", 2026-11-08)` returns a Loan due 2026-11-22. `borrow_book("M1", "B2", 2026-11-08)` returns a second loan.
   - *Failure:* a third `borrow_book("M1", ...)` returns None because M1 already has 2 open loans. A librarian calling `borrow_book` gets None because `BORROW` is not granted.
5. **Return with a late fee**
   - *Success:* M1 returns the B1 loan on 2026-11-25, which is 3 days after the due date, so `return_book` returns 30. Returning on 2026-11-22 returns 0.
   - *Failure:* returning the same loan again returns None because it is closed. M2 trying to return M1's loan returns None.
6. **Reserve and be notified**
   - *Success:* B2's only copy is borrowed by M1, and M2 calls `reserve_book("M2", "B2", ...)`, which returns True. When M1 returns B2, the copy becomes `RESERVED` for M2 and `get_messages("M2")` holds a message that B2 is ready. M2's `borrow_book` takes it.
   - *Failure:* `reserve_book("M2", "B1", ...)` returns False while a copy of B1 is available. M2 reserving B2 twice returns False. While the copy is held for M2, `borrow_book("M3", "B2", ...)` returns None.
7. **Track each copy's status**
   - *Success:* a copy goes `AVAILABLE`, `BORROWED` (M1 borrows), `AVAILABLE` (returned). With M2 waiting it goes `BORROWED`, `RESERVED` (held for M2), `BORROWED` (M2 borrows). `mark_copy_lost("L1", "B1", "B1-3")` on an available copy makes it `LOST`.
   - *Failure:* `mark_copy_lost` on a borrowed copy returns False. A `LOST` copy can't be borrowed.
8. **Different permissions**
   - *Success:* librarian L1 can `add_book`, `mark_copy_lost` and `search`. Member M1 can `borrow_book`, `return_book`, `reserve_book` and `search`.
   - *Failure:* M1 calling `add_book` returns None. L1 calling `borrow_book` returns None.
