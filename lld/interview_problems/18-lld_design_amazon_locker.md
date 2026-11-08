# Amazon Locker System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** each compartment guards whether it holds a package with its own lock, so two agents can never be given the same compartment. A package waiting in a locker is tracked by one record that joins its code, its package and its compartment, and that record decides once whether the package was collected or expired.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The locker station should have compartments of different sizes.
2. A delivery agent should be able to deposit a package, and the system should assign the smallest free compartment that fits it.
3. The system should generate a unique access code for each package and send it to the customer.
4. The customer should be able to open the compartment by entering the code and collect the package.
5. The code should expire after a set time, and the package should then be removed and the compartment freed.
6. A compartment should track whether it is available, occupied or out of service.
7. Two packages must never be assigned to the same compartment at the same time.

**Types and conventions**
- Times are `datetime`, passed in by the caller, so the design never reads a clock. The code lifetime is in whole minutes.
- A package fits a compartment if the compartment's size is the same as or bigger than the package's.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. A "thread-safe map" supports an atomic `put_if_absent`, which stores a value only if the key is empty, and an atomic `remove`.
- Making random codes and sending messages are the only contacts with the outside world, so each sits behind a small interface that a test can replace.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Size (enum) | Names the sizes and orders them | the size ranks |
| Package | Describes one package | id, size, customer |
| Compartment | Holds at most one package and guards that with a lock | the package inside, out-of-service flag |
| CodeGenerator (interface) | Makes a candidate access code | none |
| RandomCodeGenerator | Makes random digit codes | code length |
| CustomerNotifier (interface) | Sends a code to a customer | none |
| InboxNotifier | Keeps the messages sent to each customer | the messages |
| Assignment | Joins a code to a package and its compartment, and decides collected or expired once | status, expiry time |
| LockerStation | Deposits, collects and expires packages, and manages compartments | compartments, active codes |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Locker station, compartment, size, delivery agent, package, access code, customer, expiry, status.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* LockerStation, Compartment, Package and Assignment (a code joined to a package and compartment) are classes. The agent and the customer are just callers and ids.
2. **Find fixed sets of values.** Sizes, the conditions of a compartment, and the outcome of a waiting package.
   - *Why:* a fixed set is an enum, so a typo like "meduim" cannot exist.
   - *Result:* `Size`, `CompartmentStatus`, `AssignmentStatus`.
3. **Give every fact one owner.** What is inside a compartment, and whether it is out of service, belong to the Compartment. Whether a waiting package was collected or expired belongs to its Assignment. Which codes are live belongs to the station.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** A compartment's status is computed from "is it out of service" and "does it hold a package". "Has the code expired" is computed from the time.
   - *Why:* a stored `OCCUPIED` status beside the package field would disagree the moment one is updated and the other is not.
5. **Give every verb to the class whose data changes.** Occupying and freeing change the Compartment. Collecting and expiring change the Assignment.
6. **Find the action that touches several classes.** A deposit needs a free compartment, a unique code and a message, in an order that can be undone. That lives in `LockerStation.deposit`.
7. **Find the hint phrase for a pattern.** None. The requirements describe one fixed way to choose a compartment and one way to send a code.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirement 7 names it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A compartment id or None for a deposit, a Package or None for a collection, and a list of removed packages for the expiry sweep.

## 4. Pattern Guide

### Used: none of the named patterns
- **Why:** no requirement asks for a rule to be swapped, for behavior that changes by state, or for many parties to be told.
- **The two small interfaces:** `CodeGenerator` and `CustomerNotifier` are boundaries to the outside world (randomness and messaging), the same way a clock is. They make the design testable, not more flexible.

### Common patterns (not used here)
| Pattern | Signal phrase | Why not used here |
|---|---|---|
| Strategy | "a rule that can be replaced" | Not used. "Smallest free compartment that fits" is one fixed rule. |
| State | "behaves differently depending on its state" | Not used. A compartment's status only decides whether a deposit is allowed. |
| Observer | "notify several interested parties" | Not used. Only the customer is told. |
| Factory | "create different kinds of objects by type" | Not used. |
| Facade | "a single simple entry point" | Not added as a separate class. `LockerStation` is already the single entry point. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### Size
- Values: `SMALL`, `MEDIUM`, `LARGE`, with ranks 1, 2 and 3. Method: `get_rank() -> int`.
- *Why:* requirement 1 has compartments of different sizes, and the ranks decide what fits.

### CompartmentStatus
- Values: `AVAILABLE`, `OCCUPIED`, `OUT_OF_SERVICE`
- *Why:* exactly the three conditions named in requirement 6. It is reported, never stored.

### AssignmentStatus
- Values: `WAITING`, `COLLECTED`, `EXPIRED`
- *Why:* a waiting package ends in one of two ways (requirements 4 and 5), and only one may win.

## 6. Classes (smallest dependency first)

### Package
**Why this class exists:** a package is a thing with its own identity, size and owner.
- Attributes: `package_id: str`, `size: Size`, `customer_id: str`. *Why:* the identity, what it needs to fit in, and who it belongs to.
- Constructor: `Package(package_id: str, size: Size, customer_id: str)`. *Why:* all set once. It refuses empty ids.
- Methods: `get_package_id() -> str`, `get_size() -> Size`, `get_customer_id() -> str`. *Why:* read-only views.
- *Extensibility:* none needed.

### Compartment
**Why this class exists:** requirement 7 needs one place where "only one package goes here" is decided. The compartment owns what is inside it and its lock.
- Attributes:
  - `compartment_id: str`, `size: Size`. *Why:* the identity, and what fits.
  - `package: Package or None`. *Why:* the single fact about occupancy.
  - `out_of_service: bool`. *Why:* the single fact about maintenance.
  - `lock: Lock`. *Why:* makes "check free, then occupy" one step.
- Constructor: `Compartment(compartment_id: str, size: Size)`. *Why:* a new compartment is empty and in service, and creates its lock. It refuses an empty id.
- Methods (each under the lock):
  - `get_compartment_id() -> str`, `get_size() -> Size`. *Why:* read-only views.
  - `get_status() -> CompartmentStatus`. *Why:* computed. `OUT_OF_SERVICE` if the flag is set, `OCCUPIED` if it holds a package, otherwise `AVAILABLE`.
  - `fits(size: Size) -> bool`. *Why:* computed from the ranks.
  - `try_occupy(package: Package) -> bool`. *Why:* True only if it is in service, empty, and the package fits. Then it stores the package. This is what stops two packages sharing it.
  - `release(package_id: str) -> Package or None`. *Why:* removes and returns the package, but only if it is that exact package. None otherwise, so a wrong call can't empty someone else's compartment.
  - `take_out_of_service() -> bool`. *Why:* only when empty, so a package is never trapped.
  - `return_to_service() -> bool`. *Why:* only when it was out of service.
- *Extensibility:* none needed.

### CodeGenerator (interface) and RandomCodeGenerator
**Why they exist:** requirement 3 needs codes that are hard to guess. Randomness is an outside effect, so it sits behind one method that a test can replace with a fixed sequence.
- `CodeGenerator` method: `next_code() -> str`.
- `RandomCodeGenerator(length: int)`: attribute `length: int`. *Why:* how many digits. It refuses `length < 4`. `next_code()` returns a random string of that many digits.
- *Why uniqueness is not the generator's job:* only the station knows which codes are live, so it checks and retries.
- *Extensibility:* a different code format is a new class.

### CustomerNotifier (interface) and InboxNotifier
**Why they exist:** requirement 3 sends the code to the customer. Sending is an outside effect, so it sits behind one method.
- `CustomerNotifier` method: `send_code(customer_id: str, code: str, compartment_id: str, expires_at: datetime) -> bool`. *Why a `bool`:* False means the message could not be sent, and the station undoes the deposit.
- `InboxNotifier()`: attribute `messages: map{str: list[str]}` (a customer id to the messages sent to that customer, oldest first). `send_code` adds a message such as `Your code 482913 opens locker M1 until 2026-11-11 10:00` and returns True. `get_messages(customer_id: str) -> list[str]` returns a copy. *Why:* the in-memory record. A real SMS or email service would be called here.
- *Extensibility:* a new channel is a new class.

### Assignment
**Why this class exists:** a waiting package is a code, a package and a compartment joined together, with a deadline. It owns whether the package was collected or expired, so the two can't both happen.
- Attributes:
  - `code: str`. *Why:* what the customer types.
  - `package: Package`, `compartment: Compartment`. *Why:* what is waiting and where.
  - `deposited_at: datetime`, `expires_at: datetime`. *Why:* the deadline of requirement 5.
  - `status: AssignmentStatus`. *Why:* the single fact about how it ended.
  - `lock: Lock`. *Why:* a collection and an expiry racing at the deadline can't both win.
- Constructor: `Assignment(code: str, package: Package, compartment: Compartment, deposited_at: datetime, expires_at: datetime)`. *Why:* all set at creation, and the status starts `WAITING`. It refuses `expires_at <= deposited_at`.
- Methods (each under the lock):
  - `get_code() -> str`, `get_package() -> Package`, `get_compartment() -> Compartment`, `get_status() -> AssignmentStatus`, `get_expires_at() -> datetime`. *Why:* read-only views.
  - `is_expired(now: datetime) -> bool`. *Why:* computed as `now >= expires_at`.
  - `collect(now: datetime) -> bool`. *Why:* `WAITING` and not expired. Sets `COLLECTED`. False otherwise.
  - `expire(now: datetime) -> bool`. *Why:* `WAITING` and expired. Sets `EXPIRED`. False otherwise.
  - `cancel() -> None`. *Why:* used only to undo a deposit whose message failed. It sets `EXPIRED` so the record can never be collected.
- *Extensibility:* none needed.

### LockerStation (the manager and only entry point)
**Why this class exists:** a deposit needs a compartment, a unique code and a message, in an order that can be undone. Collection and expiry need the code lookup and the compartment. No single class can do that.
- Attributes:
  - `compartments: list[Compartment]`. *Why:* kept sorted by size rank, then id, so the first one that accepts is the smallest that fits.
  - `active: map{str: Assignment}` (a live code to its Assignment). *Why:* the single record of which codes are live. A thread-safe map, so `put_if_absent` makes a new code unique in one step.
  - `generator: CodeGenerator`, `notifier: CustomerNotifier`. *Why:* the two outside services, passed in.
  - `code_ttl_minutes: int`. *Why:* how long a code lasts. It is data.
  - `max_code_tries: int`. *Why:* how many fresh codes to try if one is already live.
  - `lock: Lock`. *Why:* guards only the compartment list while one is added.
- Constructor: `LockerStation(generator: CodeGenerator, notifier: CustomerNotifier, code_ttl_minutes: int, max_code_tries: int)`. *Why:* the services are passed in so tests can use fakes. It refuses non-positive numbers and starts with no compartments.
- Methods:
  - `add_compartment(compartment_id: str, size: Size) -> bool`. *Why:* setup. False if the id exists. It keeps the list sorted.
  - `deposit(package_id: str, size: Size, customer_id: str, now: datetime) -> str or None`. *Why:* requirement 2. It returns the compartment id for the agent to open. None if nothing fits or the customer could not be told.
  - `collect(code: str, now: datetime) -> Package or None`. *Why:* requirement 4. None for a wrong, used or expired code.
  - `expire_stale(now: datetime) -> list[Package]`. *Why:* requirement 5. It removes every package whose code has expired, frees the compartments, and returns the removed packages for staff to take back.
  - `take_out_of_service(compartment_id: str) -> bool`, `return_to_service(compartment_id: str) -> bool`. *Why:* look up once and ask the compartment.
  - `get_compartment_status(compartment_id: str) -> CompartmentStatus or None`. *Why:* requirement 6. None if the id is unknown.
- Private helper: `find_compartment(compartment_id: str) -> Compartment or None`. *Why:* one lookup by id.
- *Extensibility:* the outside services and the numbers are all passed in.

## 7. Relationships
- LockerStation **has** Compartment (created inside `add_compartment`), Assignment (created inside `deposit`).
- Compartment **uses** Package (the package lives on after it leaves).
- Assignment **uses** Package, Compartment.
- LockerStation **uses** CodeGenerator, CustomerNotifier (passed in).
- RandomCodeGenerator **implements** CodeGenerator. InboxNotifier **implements** CustomerNotifier.
- Enums are always **uses**: Package to Size, Compartment to Size, Assignment to AssignmentStatus.
- `CompartmentStatus` appears only as a return type, so it has no line.

## 8. Interaction

### Layers
1. **Entry point and manager:** LockerStation.
2. **Classes that guard themselves:** Compartment and Assignment.
3. **Outside services:** CodeGenerator and CustomerNotifier.
4. **Plain data:** Package.

### Call chains
- **Deposit**
  1. `Agent -> LockerStation.deposit(package_id, size, customer_id, now)`.
  2. `LockerStation -> Compartment.try_occupy(package)`, smallest first: the compartment decides.
  3. `LockerStation -> CodeGenerator.next_code()`, then `active.put_if_absent(code, assignment)`: retried until the code is unique.
  4. `LockerStation -> CustomerNotifier.send_code(...)`: the customer is told last, once everything is in place.
- **Collect**
  1. `Customer -> LockerStation.collect(code, now)`.
  2. `LockerStation -> active.get(code)`, then `Assignment.collect(now)`: the record decides.
  3. `LockerStation -> Compartment.release(package_id)`, then `active.remove(code)`.
- **Expire**
  1. `Scheduler -> LockerStation.expire_stale(now)`.
  2. For each live code: `Assignment.expire(now)`, then `Compartment.release(package_id)`, then `active.remove(code)`.

### Who may call whom
- Agents, customers, staff and the sweep call `LockerStation` only.
- `LockerStation` calls `Compartment`, `Assignment`, `CodeGenerator` and `CustomerNotifier`.
- `Assignment` and `Compartment` call no other class.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Deposit
1. `deposit(package_id: str, size: Size, customer_id: str, now: datetime)`. Build the `Package` (refused if ids are empty).
2. Go through `compartments` smallest first. Stop at the first where `try_occupy(package)` is True. If none, return None. *Why:* the compartment's own lock decides, so two agents can't both get it.
3. Up to `max_code_tries` times: `code = generator.next_code()`, build the `Assignment` with `expires_at = now + code_ttl_minutes`, and `active.put_if_absent(code, assignment)`. Stop on the first success.
4. If no unique code was found, `compartment.release(package_id)` and return None. *Undone:* the compartment.
5. `notifier.send_code(customer_id, code, compartment id, expires_at)`. If False, `assignment.cancel()`, `active.remove(code)`, `compartment.release(package_id)`, and return None. *Undone:* the code and the compartment, so a customer who was never told can't have a package stuck in a locker.
6. Return the compartment id.

### Collect
1. `collect(code: str, now: datetime)`. `assignment = active.get(code)`. If None, return None. *Why:* a wrong or used code is refused.
2. If `assignment.collect(now)` is False: if the code has expired, run the expiry steps for this one assignment. Return None.
3. `package = compartment.release(package_id)`, then `active.remove(code)`. Return the package.
- **Failure:** a wrong, used or expired code opens nothing.

### Expire stale packages
1. `expire_stale(now: datetime)`. Take a copy of the live assignments.
2. For each, if `assignment.expire(now)` is True: `compartment.release(package_id)`, `active.remove(code)`, and add the package to the result.
3. Return the result.
- **Failure:** an assignment collected meanwhile makes `expire` False, so it is skipped. A package can't be both collected and removed.

### Out of service
1. `take_out_of_service(compartment_id)` is accepted only for an empty compartment. `return_to_service` only for one that is out of service.
- **Failure:** False, with no change. Deposits skip an out-of-service compartment because `try_occupy` refuses it.

### Concurrency (requirement 7)
- **Compartment:** the lock sits inside `Compartment`, around `try_occupy`, `release` and the service changes. "Check free, then occupy" is one step, so two agents can never be given the same compartment. The loser just tries the next one.
- **Assignment:** the lock sits inside `Assignment`, around `collect` and `expire`, so a collection and an expiry at the deadline can't both succeed.
- **Codes:** `active` is a thread-safe map, and `put_if_absent` makes a new code unique in one step, even if two deposits draw the same code.
- **Station:** the lock sits inside `LockerStation`, around the compartment list only, and is released before any compartment is touched.
- **Lock order:** no method ever holds two locks at once, so no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| More compartments or new stations | Calls to `add_compartment`, or more `LockerStation` objects | No. It is data. |
| A different code lifetime | A different `code_ttl_minutes` | No. It is data. |
| Sending codes by a new channel | A new class implementing `CustomerNotifier` | No. |
| A different code format (letters, longer) | A new class implementing `CodeGenerator`, or a different length | No. |

## 11. Interview Notes

### Key decisions
1. **Each compartment guards itself with a lock.**
   - *Why:* only one compartment is ever contested at a time, so only it waits.
   - *Rejected:* one lock on the whole station, which would make every agent wait for every other.
2. **A compartment's status is computed, not stored.**
   - *Why:* occupancy is one field and maintenance is one flag, so the status can't disagree with either.
   - *Rejected:* a stored `OCCUPIED` status beside the package.
3. **`Assignment` decides collected or expired, once.**
   - *Why:* at the deadline, a collection and the sweep can race, and exactly one must win.
   - *Rejected:* the station checking the time and then acting, which two callers could both pass.
4. **The code is made unique with `put_if_absent`, and retried.**
   - *Why:* checking and claiming the code is one step.
   - *Rejected:* checking the map and then adding, which two deposits could both pass with the same code.
5. **The customer is told last, and a failed message undoes the deposit.**
   - *Why:* a package is never left in a locker that nobody can open.
   - *Rejected:* telling the customer first, which could send a code for a compartment that was never secured.

### Edge cases handled
- A small package goes into a small compartment even when bigger ones are free.
- No compartment fits, so the deposit is refused with nothing changed.
- Two agents racing for the last compartment: one gets it, the other tries the next.
- A generated code that is already live is retried.
- A message that can't be sent undoes the whole deposit.
- A wrong, used or expired code opens nothing.
- A collection at the very moment of expiry: exactly one of the two wins.
- An occupied compartment can't be taken out of service.

### Likely follow-up questions
1. **"What stops someone guessing codes?"** Lock out a station, or a customer, after a few wrong codes in a row. It would be a counter checked at the start of `collect`. It is not required here.
2. **"How does expiry run on its own?"** A scheduler calls `expire_stale(now)` every few minutes. `collect` also checks the deadline itself, so an expired code never works even before the sweep runs.
3. **"Why not put a package in a bigger compartment when the right size is full?"** It already does. "Smallest free that fits" moves on to the next size when every small one is taken.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Locking out after wrong codes | See follow-up question 1. |
| Several packages for one customer under one code | One code per package, as requirement 3 says. |
| Returning a package to the sender | `expire_stale` returns the removed packages. Sending them back is outside the locker. |
| Running the expiry sweep on a timer | Infrastructure outside the design. It only needs to call `expire_stale(now)`. |
| Choosing between several stations | One station. A choice of station would be a layer above. |
| A real SMS or email service | `CustomerNotifier.send_code` is the one place it goes. |

## 13. Requirements Met
1. Compartments of different sizes: `Compartment`, `Size`, and `LockerStation.add_compartment`.
2. Deposit into the smallest free compartment that fits: `LockerStation.deposit`, the sorted list, and `Compartment.try_occupy`.
3. A unique code sent to the customer: `CodeGenerator`, `put_if_absent` on `active`, and `CustomerNotifier.send_code`.
4. Open with the code and collect: `LockerStation.collect` and `Assignment.collect`.
5. Expiry, removal and freeing: `Assignment.expire` and `LockerStation.expire_stale`.
6. Available, occupied or out of service: `Compartment.get_status`, `take_out_of_service`, `return_to_service`.
7. Never two packages in one compartment: the lock inside `Compartment.try_occupy`.

## 14. Requirement Fit Check

**Setup used in every example:** compartments `S1` and `S2` (SMALL), `M1` (MEDIUM), `L1` (LARGE). The code lifetime is 4320 minutes (3 days). A test generator returns 111111, then 111111 again, then 222222, then 333333. An `InboxNotifier` is used. The deposit time is 2026-11-08 10:00.

1. **Compartments of different sizes**
   - *Success:* `add_compartment` for S1, S2, M1 and L1 all return True. The list is ordered S1, S2, M1, L1.
   - *Failure:* `add_compartment("S1", SMALL)` again returns False.
2. **Smallest free compartment that fits**
   - *Success:* a SMALL package `P1` goes to S1, not M1. A MEDIUM package `P2` goes to M1. With M1 full, a MEDIUM package `P3` goes to L1.
   - *Failure:* with L1 now full, a LARGE package `P4` returns None, and no compartment changes.
3. **A unique code, sent to the customer**
   - *Success:* P1 gets code 111111, and customer C1's inbox has the message for S1. P2's first drawn code is 111111 again, which `put_if_absent` refuses because it is live, so P2 gets 222222.
   - *Failure:* if the notifier fails for P3, the deposit is undone: the code is removed, L1 is free again, and `deposit` returns None.
4. **Open with the code and collect**
   - *Success:* `collect("111111", 2026-11-09 09:00)` returns P1, and S1 is `AVAILABLE` again.
   - *Failure:* `collect("999999", ...)` returns None. Using 111111 again returns None because it was used.
5. **Expiry, removal and freeing**
   - *Success:* P2 expires at 2026-11-11 10:00. `expire_stale(2026-11-11 11:00)` returns [P2], M1 is freed, and code 222222 is no longer live.
   - *Failure:* `collect("222222", 2026-11-11 10:30)` before the sweep runs returns None, because `collect` checks the deadline itself and expires the package.
6. **Available, occupied, out of service**
   - *Success:* S1 is `AVAILABLE`, then `OCCUPIED` with P1, then `AVAILABLE` after collection. `take_out_of_service("S2")` makes S2 `OUT_OF_SERVICE`, and the next SMALL deposit skips S2.
   - *Failure:* `take_out_of_service("M1")` while it holds P2 returns False.
7. **Never two packages in one compartment**
   - *Success:* two agents deposit SMALL packages at the same moment with only S1 free among the small ones. Both call `try_occupy` on S1, and its lock lets one succeed.
   - *Failure to avoid:* the other agent's `try_occupy` on S1 returns False, so it moves on to M1. S1 never holds two packages.
