# Distributed / Unique ID Generator (Snowflake-style): Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** an ID is three numbers packed into one 64-bit integer: the time in the highest bits, then the machine number, then a counter. Two machines can't clash because their machine numbers differ. Two IDs from one machine can't clash because, within one millisecond, the counter differs, and a lock makes every thread take its turn. The time sits in the highest bits, so a bigger ID was generally made later.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The system should generate unique IDs across multiple machines without them talking to each other.
2. The IDs should be roughly ordered by the time they were created.
3. Each ID should combine a timestamp, a machine identifier and a sequence number.
4. A machine should be able to generate many IDs within the same millisecond, and wait for the next millisecond when its sequence runs out.
5. The generator should be safe to use from multiple threads.
6. The generator should handle the clock moving backwards.

**Types and conventions**
- An ID is a 64-bit `int`. The highest bit is always 0, so the ID is never negative.
- The default layout is 41 bits of time (milliseconds since a chosen start date, which lasts about 69 years), 10 bits of machine number (0 to 1023), and 12 bits of counter (0 to 4095 per millisecond).
- Time is `int` milliseconds, read from a `Clock`, so tests can control it.
- Each running process gets its own machine number when it is deployed. Two processes must never share one, and handing out the numbers happens outside this design.
- `or None` means the method can fail and returns None.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| IdLayout | Knows how many bits each part takes, and packs and unpacks IDs | the bit widths |
| IdParts | Holds the three parts of one ID | time, machine, counter |
| Clock (interface) | Says the current time in milliseconds | none |
| SystemClock | Reads the real time | none |
| SnowflakeGenerator | Makes the next ID for one machine, safely across threads | last time used, current counter |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** ID, machine, timestamp, machine identifier, sequence number, millisecond, clock, thread.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* the generator, the layout, the parts and the clock are classes. The timestamp, machine number and counter are values.
2. **Find fixed sets of values.** None.
3. **Give every fact one owner.** The bit widths belong to the layout. The last time used and the counter belong to the generator. The machine number belongs to the generator too, fixed at creation.
   - *Why:* two copies of the counter would hand out the same number twice.
4. **Mark what can be computed and do not store it.** The ID is computed from its parts, and the parts from the ID. The largest counter and largest machine number come from the bit widths.
5. **Give every verb to the class whose data changes.** Making the next ID changes the counter and the last time, so the generator owns it. Packing bits belongs to the layout.
6. **Find the action that touches several classes.** Making an ID reads the clock, updates the counter, and asks the layout to pack. That is `SnowflakeGenerator.next_id`.
7. **Find the hint phrase for a pattern.** None. The clock interface is only a boundary to the outside world.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** Requirement 5 names it, so decide exactly where the lock sits (section 8).
10. **Fix the outputs.** An `int` or None for a new ID, and `IdParts` for decoding.

## 4. Pattern Guide

### Used: none of the named patterns
- **Why:** no requirement asks for a swappable rule, for behavior that changes by state, or for notifications.
- **The clock interface:** it is a boundary to the outside world, the same as in other designs, so tests can make time stand still, jump ahead, or go backwards.

### Common patterns (not used here)
| Pattern | Signal phrase | Why not used here |
|---|---|---|
| Singleton | "exactly one instance" | Not used as a pattern. Each process should create one generator and share it, because two generators with the same machine number would clash. Creating it once at startup and passing it around does that without a global. |
| Strategy | "a rule that can be replaced" | Not used. There is one way to make an ID. |
| State | "behaves differently depending on its state" | Not used. |
| Observer | "notify when something changes" | Not used. |

## 5. Classes (smallest dependency first)

### IdLayout
**Why this class exists:** requirement 3 packs three parts into one number. Keeping all the bit arithmetic in one class means packing and unpacking can never disagree.
- Attributes: `timestamp_bits: int`, `machine_bits: int`, `sequence_bits: int`. *Why:* the width of each part.
- Constructor: `IdLayout(timestamp_bits: int, machine_bits: int, sequence_bits: int)`. *Why:* all set once. It refuses any width below 1, or a total above 63, so the highest bit stays 0.
- Methods:
  - `max_timestamp() -> int`, `max_machine_id() -> int`, `max_sequence() -> int`. *Why:* computed as `2^bits - 1` for each part.
  - `compose(timestamp: int, machine_id: int, sequence: int) -> int or None`. *Why:* `timestamp` shifted left by `machine_bits + sequence_bits`, then `machine_id` shifted left by `sequence_bits`, then `sequence`, combined. None if any part is negative or too big.
  - `decompose(id_value: int) -> IdParts or None`. *Why:* the reverse, using masks and shifts. None for a negative number.
- *Extensibility:* a different split (more machines, fewer per millisecond) is different numbers.

### IdParts
**Why this class exists:** reading an ID back into its three parts is useful for debugging and for checking requirement 3.
- Attributes: `timestamp: int`, `machine_id: int`, `sequence: int`. *Why:* the three parts.
- Constructor: `IdParts(timestamp: int, machine_id: int, sequence: int)`. *Why:* all set once.
- Methods: `get_timestamp() -> int`, `get_machine_id() -> int`, `get_sequence() -> int`. *Why:* read-only views.
- *Extensibility:* none needed.

### Clock (interface) and SystemClock
**Why they exist:** the generator depends on the time, and requirements 4 and 6 are about what happens as time moves (or goes backwards). Asking a `Clock` lets tests control it.
- `Clock` method: `now_millis() -> int`.
- `SystemClock()` returns the real current time in milliseconds.
- *Extensibility:* a test clock is a new class.

### SnowflakeGenerator (the only entry point)
**Why this class exists:** it is the one owner of "what time and counter did I last use", which is the fact that must never repeat.
- Attributes:
  - `layout: IdLayout`. *Why:* how to pack.
  - `machine_id: int`. *Why:* this machine's number, fixed at creation. It is why machines never clash without talking (requirement 1).
  - `epoch_millis: int`. *Why:* the start date. Counting from it rather than from 1970 lets the 41 bits last about 69 years from that date.
  - `clock: Clock`. *Why:* where the time comes from.
  - `max_backward_wait_ms: int`. *Why:* how far back the clock may jump before the generator gives up instead of waiting.
  - `last_timestamp: int`. *Why:* the time part of the last ID made. It starts at −1.
  - `sequence: int`. *Why:* the counter within `last_timestamp`.
  - `lock: Lock`. *Why:* only one thread at a time may read and change the two values above.
- Constructor: `SnowflakeGenerator(layout: IdLayout, machine_id: int, epoch_millis: int, clock: Clock, max_backward_wait_ms: int)`. *Why:* all passed in, so tests can use a small layout and a fake clock. It refuses a machine number outside `0` to `layout.max_machine_id()`, a start date in the future, or a negative wait.
- Methods:
  - `next_id() -> int or None`. *Why:* the next ID. None means it can't safely make one right now (the clock went too far back, or the time part has run out). Steps in section 8.
  - `decode(id_value: int) -> IdParts or None`. *Why:* asks the layout to unpack.
  - `get_machine_id() -> int`. *Why:* read-only view.
- Private helper: `wait_until_after(timestamp: int) -> int`. *Why:* keeps reading the clock, pausing very briefly, until the time is later than `timestamp`, and returns that time.
- *Extensibility:* none needed.

## 6. Relationships
- SnowflakeGenerator **uses** IdLayout, Clock (passed in).
- SystemClock **implements** Clock.
- There are no enums, so there are no enum lines.
- `IdParts` appears only as a return type, so it has no line.

## 7. Interaction

### Layers
1. **Entry point:** SnowflakeGenerator.
2. **Bit arithmetic:** IdLayout.
3. **Outside world:** Clock.
4. **Plain data:** IdParts.

### Call chains
- **Make an ID**
  1. `Caller -> SnowflakeGenerator.next_id()`.
  2. `SnowflakeGenerator -> Clock.now_millis()`, minus `epoch_millis`.
  3. Under the lock: compare with `last_timestamp`, update `sequence` and `last_timestamp`.
  4. `SnowflakeGenerator -> IdLayout.compose(timestamp, machine_id, sequence)`.
- **Read an ID back**
  1. `Caller -> SnowflakeGenerator.decode(id)`, then `IdLayout.decompose(id)`.

### Who may call whom
- Callers call `SnowflakeGenerator` only.
- `SnowflakeGenerator` calls `Clock` and `IdLayout`.
- No class calls back up to its owner, so there are no cycles.

## 8. Flows

### Make the next ID
1. `next_id()`. Take the lock.
2. `now = clock.now_millis() - epoch_millis`.
3. **If the clock went backwards** (`now < last_timestamp`):
   - If the gap is at most `max_backward_wait_ms`, `now = wait_until_after(last_timestamp - 1)`, which waits until the clock is back to at least `last_timestamp`. *Why:* a small jump (a clock correction) is fixed by waiting, and never reuses a time.
   - Otherwise return None. *Why:* waiting a long time would freeze every caller, and making an ID would risk a duplicate. Refusing is the safe choice.
4. **If it is the same millisecond** (`now == last_timestamp`): `sequence = sequence + 1`. If that passes `layout.max_sequence()`, the counter has run out for this millisecond, so `now = wait_until_after(last_timestamp)` and `sequence = 0`.
5. **If it is a new millisecond:** `sequence = 0`.
6. If `now > layout.max_timestamp()`, return None. *Why:* the time part has run out after about 69 years, and wrapping round would repeat old IDs.
7. `last_timestamp = now`. Return `layout.compose(now, machine_id, sequence)`. Release the lock.
- **Why waiting while holding the lock is correct:** every other thread would need the next millisecond anyway, so letting them in early would gain nothing.

### Decode
1. `decode(id_value)` asks `layout.decompose`. None for a negative number.

### Concurrency (requirement 5)
- **Generator:** the lock sits inside `SnowflakeGenerator`, around the whole of steps 2 to 7. Reading the time, changing the counter and saving `last_timestamp` happen as one step, so two threads can never get the same counter in the same millisecond.
- **Layout and parts:** they never change, so they need no lock.
- **Lock order:** there is only one lock, so no deadlock is possible.

## 9. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| More machines, or more IDs per millisecond | A different `IdLayout`, such as 41 / 12 / 10 bits | No. It is data. |
| A different start date | A different `epoch_millis` | No. It is data. |
| More tolerance for clock jumps | A different `max_backward_wait_ms` | No. It is data. |
| Splitting the machine number into data centre and machine | A layout with four parts | Yes, `IdLayout` gains a part. |

## 10. Interview Notes

### Key decisions
1. **Time in the highest bits.**
   - *Why:* comparing two IDs compares their times first, which gives requirement 2's rough order.
   - *Rejected:* the machine number first, which would order IDs by machine.
2. **A machine number fixed at deployment.**
   - *Why:* machines never need to talk to each other, which is requirement 1.
   - *Rejected:* asking a central service for each ID, which makes every ID depend on a network call.
3. **Wait on a small backwards jump, refuse on a large one.**
   - *Why:* a small correction is fixed by waiting a few milliseconds, and a large jump is too dangerous to guess about.
   - *Rejected:* ignoring the clock and carrying on, which can make duplicate IDs.
4. **One lock around the whole step.**
   - *Why:* it is short and simple, and it can still make millions of IDs per second.
   - *Rejected:* a lock-free compare-and-swap loop, which is faster under heavy contention but much harder to get right.
5. **All bit arithmetic in `IdLayout`.**
   - *Why:* packing and unpacking use the same widths, so they can't disagree.
   - *Rejected:* shifts written inline in the generator.

### Edge cases handled
- Many IDs in one millisecond: the counter goes up.
- The counter runs out: the generator waits for the next millisecond.
- The clock goes back a little: the generator waits.
- The clock goes back a lot: `next_id` returns None instead of risking a duplicate.
- The time part runs out after about 69 years: `next_id` returns None.
- A machine number too big for its bits, or a start date in the future, is refused at creation.
- The highest bit is always 0, so IDs are never negative.

### Likely follow-up questions
1. **"How do machines get unique numbers?"** At deployment, from configuration or a coordination service such as ZooKeeper, once at start-up. The design only requires that two running processes never share one.
2. **"Are IDs strictly ordered across machines?"** No, only roughly. Within one millisecond, IDs from different machines are ordered by machine number, and clocks on different machines are never exactly the same.
3. **"What happens after a restart if the clock went backwards while it was down?"** `last_timestamp` starts at −1 after a restart, so the generator can't notice. Saving the last time used at shutdown and refusing to start before it would cover that.

## 11. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Handing out machine numbers | See follow-up question 1. |
| Strict order across machines | See follow-up question 2. |
| Clock jumps across a restart | See follow-up question 3. |
| IDs that hide when they were made | The time is readable from an ID by design. Scrambling it would lose the ordering. |
| More than 4096 IDs per millisecond per machine | A layout with more counter bits. |

## 12. Requirements Met
1. Unique across machines without talking: the fixed `machine_id` in every ID.
2. Roughly ordered by time: the time in the highest bits, set by `IdLayout.compose`.
3. Time, machine and counter in each ID: `IdLayout.compose` and `decompose`, and `IdParts`.
4. Many per millisecond, then wait: the `sequence` counter and `wait_until_after`, in step 4.
5. Safe across threads: the lock inside `SnowflakeGenerator` around the whole step.
6. Clock moving backwards: step 3, wait for a small jump and refuse a large one.

## 13. Requirement Fit Check

**Setup used in every example:** `IdLayout(41, 10, 12)`. A test clock. The epoch is chosen so that the first time read is 1000 milliseconds after it. `max_backward_wait_ms` is 5.

1. **Unique across machines without talking**
   - *Success:* generators for machine 7 and machine 8 both make an ID at time 1000 with counter 0. The IDs differ in the machine bits, so they can't be equal.
   - *Failure:* `SnowflakeGenerator(layout, 1024, ...)` is refused, because 10 bits allow only 0 to 1023.
2. **Roughly ordered by time**
   - *Success:* any ID made at time 1001 is bigger than any ID made at time 1000, on any machine, because the time is in the highest bits.
   - *Failure to avoid:* within the same millisecond, machine 8's IDs are bigger than machine 7's, which is why the order is only rough.
3. **Time, machine and counter in each ID**
   - *Success:* time 1000, machine 7, counter 0 gives `1000 × 2^22 + 7 × 2^12 + 0` = 4194304000 + 28672 = 4194332672. `decode(4194332672)` gives back time 1000, machine 7, counter 0.
   - *Failure:* `decode(-1)` returns None.
4. **Many per millisecond, then wait**
   - *Success:* with the clock held at 1000, 4096 calls give counters 0 to 4095. The 4097th call finds the counter used up, waits until the clock shows 1001, and returns an ID with time 1001 and counter 0.
   - *Failure to avoid:* the 4097th ID must never reuse counter 0 at time 1000.
5. **Safe across threads**
   - *Success:* 8 threads each ask for 10,000 IDs at once. All 80,000 IDs are different, because each call holds the lock for the whole step.
   - *Failure to avoid:* without the lock, two threads could read the same counter in the same millisecond and make the same ID.
6. **Clock moving backwards**
   - *Success:* the last ID used time 2000, then the clock reads 1997. The gap of 3 is within 5, so the generator waits until the clock reaches 2000 and continues.
   - *Failure:* the clock reads 1950, a gap of 50. `next_id` returns None. When the clock passes 2000 again, `next_id` works normally, and no ID was ever repeated.
