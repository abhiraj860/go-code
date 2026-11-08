# In-Memory Key-Value Store with TTL & Transactions: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** the shared store holds only committed data. Each client works through its own session, which keeps a private stack of change layers, one per open transaction. Reads look through the client's own layers first and then the store, so a client sees its own changes and nobody else does. Committing the inner layer merges it into the one below, and committing the last layer applies everything to the store in one locked step. Expiry is checked on every read and cleaned up in the background.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The store should allow setting, getting and deleting keys.
2. A key should optionally have a time to live, and an expired key should not be returned.
3. Expired keys should be cleaned up automatically.
4. The store should support beginning, committing and rolling back a transaction.
5. Transactions should be allowed to nest.
6. Changes made inside a transaction should stay hidden from others until it is committed and be discarded on rollback.
7. The store should stay correct when accessed by multiple threads.

**Types and conventions**
- Keys and values are `str`. A value is never None, so None can mean "missing".
- A time to live is a whole number of seconds, or None for "never expires". The store reads the time from a `Clock`, so tests can control it.
- A key expires at the moment `now >= expires_at`.
- "Others" means other sessions. One session is used by one thread at a time.
- When two sessions commit changes to the same key, the later commit wins.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| ChangeType (enum) | Names the two kinds of change | none |
| Clock (interface) | Says what time it is | none |
| SystemClock | Reads the real time | none |
| Entry | Holds one committed value and when it expires | value, expiry time |
| Change | Records one pending set or delete | kind, value, expiry time |
| TransactionLayer | Holds the pending changes of one open transaction | its changes |
| KeyValueStore | Holds committed data, applies commits in one step, and removes expired keys | the data, the expiry queue |
| Session | Gives one client its own view, and runs begin, commit and rollback | its stack of layers |
| ExpiryCleaner | Runs the cleanup in the background on a timer | its thread |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Store, key, value, time to live, transaction, nested transaction, others, thread.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* KeyValueStore, Entry, Change, TransactionLayer and Session are classes. "Others" are other sessions.
2. **Find fixed sets of values.** A pending change is a set or a delete.
   - *Result:* `ChangeType`.
3. **Give every fact one owner.** Committed data belongs to the store. Pending changes belong to the layer of the transaction that made them. The stack of open transactions belongs to the session.
   - *Why:* two copies of one fact will eventually disagree. A pending change is never written to the store early.
4. **Mark what can be computed and do not store it.** "Is this key expired" is computed from the time. The value a session sees is computed by looking through its layers and then the store.
5. **Give every verb to the class whose data changes.** Applying a commit changes the store. Recording a pending change changes a layer. Opening and closing transactions changes the session.
6. **Find the action that touches several classes.** A read needs the session's layers, then the store, then the expiry check, in order. That lives in `Session.get`.
7. **Find the hint phrases for patterns.** None names a classic pattern. The key structure is a stack of overlays (section 4).
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** Requirement 7 names it, so decide exactly where the locks sit (section 8).
10. **Fix the outputs.** A value or None for a read, a `bool` for every change and every transaction command.

## 4. Pattern Guide

### Used: none of the named patterns. The key structure is a stack of change layers.
- **Meaning:** each open transaction is a small map of pending changes sitting on top of the ones below it, like sheets of tracing paper over a drawing. A read looks down through the sheets until it finds the key.
- **Why it fits:** nesting (requirement 5) is just one more sheet. Rollback throws away the top sheet. Commit presses the top sheet onto the one below, or onto the store if it is the last.
- **Why changes are not written to the store early:** other sessions would see them, which breaks requirement 6.

### Common patterns (not used here)
| Pattern | Signal phrase | Why not used here |
|---|---|---|
| Memento | "save and restore a snapshot" | Not used. A layer records only what changed, which is smaller than a snapshot of everything. |
| Command | "undo and redo actions" | Not used. Rollback discards a layer instead of undoing each change. |
| Strategy | "a rule that can be replaced" | Not used. |
| Observer | "notify when something changes" | Not used. |
| Singleton | "exactly one instance" | Not used. A passed-in store is easier to test. |

## 5. Enums

### ChangeType
- Values: `SET`, `DELETE`
- *Why:* a pending delete must hide a committed value from the session, so it is stored as a change of its own.

## 6. Classes (smallest dependency first)

### Clock (interface) and SystemClock
**Why they exist:** expiry depends on the time, and the background cleaner needs it too. Asking a `Clock` means tests can move time forward themselves.
- `Clock` method: `now() -> datetime`.
- `SystemClock()` implements it with the real time.
- *Extensibility:* a test clock is a new class.

### Entry
**Why this class exists:** a committed value and its expiry belong together.
- Attributes: `value: str`, `expires_at: datetime or None`. *Why:* the value, and when it stops counting. None means never.
- Constructor: `Entry(value: str, expires_at: datetime or None)`. *Why:* both set once. It refuses a None value.
- Methods: `get_value() -> str`, `get_expires_at() -> datetime or None`, `is_expired(now: datetime) -> bool`. *Why `is_expired`:* computed, so no stored "expired" flag can be wrong.
- *Extensibility:* none needed.

### Change
**Why this class exists:** a pending change must record whether it is a set or a delete, and for a set, the value and expiry.
- Attributes: `change_type: ChangeType`, `value: str or None`, `expires_at: datetime or None`. *Why:* a delete has no value.
- Constructor: `Change(change_type: ChangeType, value: str or None, expires_at: datetime or None)`. *Why:* all set once. It refuses a `SET` with no value, and a `DELETE` with a value.
- Methods: `get_change_type() -> ChangeType`, `get_value() -> str or None`, `get_expires_at() -> datetime or None`, `is_expired(now: datetime) -> bool`. *Why:* the same expiry rule as an entry.
- *Extensibility:* none needed.

### TransactionLayer
**Why this class exists:** requirement 5 nests transactions, and each one's pending changes must be kept apart so a rollback throws away exactly its own.
- Attributes: `changes: map{str: Change}` (a key to the last pending change to that key in this transaction). *Why:* only the last change to a key matters.
- Constructor: `TransactionLayer()`. *Why:* starts empty.
- Methods:
  - `record(key: str, change: Change) -> None`. *Why:* replaces any earlier change to that key in this layer.
  - `lookup(key: str) -> Change or None`. *Why:* None means this layer did not touch the key.
  - `merge_into(lower: TransactionLayer) -> None`. *Why:* used by an inner commit. Each change here replaces the lower layer's change to the same key.
  - `get_changes() -> map{str: Change}` (returns a copy). *Why:* used by the outermost commit.
- *Extensibility:* none needed.

### KeyValueStore
**Why this class exists:** committed data is shared by every session, so it needs one owner with one lock. It also owns expiry cleanup, because only it knows when each key expires.
- Attributes:
  - `data: map{str: Entry}` (a key to its committed entry). *Why:* the one record of committed data.
  - `expiry_queue: list[(datetime, str)]`. *Why:* a min-heap of (expiry time, key), so the cleaner finds the next key to expire without scanning every key. An entry in it may be stale if the key was changed later. The cleaner checks before deleting.
  - `clock: Clock`. *Why:* where the time comes from.
  - `lock: Lock`. *Why:* every read and change of `data` and `expiry_queue` runs one at a time.
- Constructor: `KeyValueStore(clock: Clock)`. *Why:* the clock is passed in. It starts empty.
- Methods (each under the lock):
  - `get(key: str) -> str or None`. *Why:* None if missing or expired.
  - `set(key: str, value: str, ttl_seconds: int or None) -> bool`. *Why:* requirement 1 without a transaction. False if `ttl_seconds` is 0 or less, or the value is None. If there is a time to live, the expiry is `now + ttl_seconds` and it is pushed onto `expiry_queue`.
  - `delete(key: str) -> bool`. *Why:* True if a live key was removed.
  - `apply(changes: map{str: Change}) -> None`. *Why:* the outermost commit. All changes are applied under one hold of the lock, so other readers see either none or all of them. A `SET` that has already expired is applied as a delete, because the set replaced the old value and is itself gone.
  - `purge_expired() -> int`. *Why:* requirement 3. While the earliest time in `expiry_queue` is not after now, pop it. If `data` still holds that key with that same expiry, delete it. Returns how many keys were removed.
  - `size() -> int`. *Why:* the number of stored entries, for monitoring.
- *Extensibility:* none needed.

### Session
**Why this class exists:** requirement 6 needs each client to see its own uncommitted changes while others don't. A session is that client's private view, and it owns its stack of open transactions.
- Attributes:
  - `store: KeyValueStore`. *Why:* where committed data lives, and where the outermost commit goes.
  - `layers: list[TransactionLayer]`. *Why:* the open transactions, innermost last. Empty means no transaction is open.
  - `clock: Clock`. *Why:* to stamp expiry times on pending sets. It is the store's clock.
- Constructor: `Session(store: KeyValueStore, clock: Clock)`. *Why:* starts with no open transaction.
- Methods:
  - `get(key: str) -> str or None`. *Why:* looks through `layers` from innermost to outermost. The first layer that has a change for the key decides: a `DELETE` or an expired `SET` gives None, a live `SET` gives its value. If no layer touched the key, it asks the store.
  - `set(key: str, value: str, ttl_seconds: int or None) -> bool`. *Why:* with no open transaction, it goes straight to `store.set`. Otherwise it records a `SET` change, with its expiry computed now, in the innermost layer. False for a None value or a time to live of 0 or less.
  - `delete(key: str) -> bool`. *Why:* with no open transaction, it goes straight to `store.delete`. Otherwise, if `get(key)` is None it returns False, and if not it records a `DELETE` in the innermost layer and returns True.
  - `begin() -> None`. *Why:* pushes a new empty layer. Any depth is allowed.
  - `commit() -> bool`. *Why:* False if no transaction is open. Pops the innermost layer. If another layer is still open, it merges into it, so the changes stay hidden. If it was the last, `store.apply(changes)`. Returns True.
  - `rollback() -> bool`. *Why:* False if no transaction is open. Pops the innermost layer and throws it away.
  - `in_transaction() -> bool`, `depth() -> int`. *Why:* read-only views.
- *Why a session has no lock:* it belongs to one client thread. Its layers are private, so no other thread ever touches them.
- *Extensibility:* none needed.

### ExpiryCleaner
**Why this class exists:** requirement 3 says expired keys are cleaned up automatically. Something must call `purge_expired` regularly without the users doing it.
- Attributes: `store: KeyValueStore`, `interval_ms: int`, `running: bool`, `thread: Thread or None`. *Why:* what to clean, how often, and the background thread.
- Constructor: `ExpiryCleaner(store: KeyValueStore, interval_ms: int)`. *Why:* refuses an interval below 1.
- Methods:
  - `start() -> bool`. *Why:* starts one thread that repeats `store.purge_expired()` and then waits `interval_ms`. False if already started.
  - `stop() -> None`. *Why:* sets `running` to False and waits for the thread to finish.
- *Extensibility:* a different interval is data.

## 7. Relationships
- KeyValueStore **has** Entry (entries are created inside `set` and `apply`).
- TransactionLayer **has** Change (changes live and die with their layer).
- Session **has** TransactionLayer (created by `begin`, dropped by `commit` and `rollback`).
- Session **uses** KeyValueStore, Clock. KeyValueStore **uses** Clock. ExpiryCleaner **uses** KeyValueStore.
- SystemClock **implements** Clock.
- Enums are always **uses**: Change to ChangeType.

## 8. Interaction

### Layers
1. **Entry points:** Session (one per client) and ExpiryCleaner (the background job).
2. **Shared committed data:** KeyValueStore.
3. **Private pending data:** TransactionLayer and Change.
4. **Plain data and time:** Entry, Clock.

### Call chains
- **Read**
  1. `Client -> Session.get(key)`.
  2. `Session -> TransactionLayer.lookup(key)` from innermost to outermost: the client's own changes win.
  3. If none: `Session -> KeyValueStore.get(key)`, which checks expiry under the lock.
- **Write inside a transaction**
  1. `Client -> Session.set(...)` or `delete(...)`, then `TransactionLayer.record(key, change)`. The store is not touched.
- **Commit**
  1. `Client -> Session.commit()`.
  2. Inner: `TransactionLayer.merge_into(lower)`. Outermost: `KeyValueStore.apply(changes)`.
- **Clean up**
  1. `ExpiryCleaner thread -> KeyValueStore.purge_expired()` every interval.

### Who may call whom
- Clients call their own `Session` only, and never another client's session or the layers.
- `Session` calls its layers and the store.
- `ExpiryCleaner` calls the store only.
- The store never calls a session. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Set, get and delete without a transaction
1. `session.set(key, value, ttl)` goes straight to `store.set`, so it is visible to everyone at once. `get` and `delete` work the same way.
- **Failure:** a None value or a time to live of 0 or less is refused with no change.

### Read with expiry
1. `get(key)` finds the first layer that touched the key, or the store.
2. If that change or entry has expired, the result is None. *Why on every read:* an expired key must never be returned, even if the cleaner has not run yet.

### Begin, commit and roll back
1. `begin()` pushes an empty layer.
2. `commit()` pops the innermost. If a layer is left, `merge_into` it. If none is left, `store.apply(changes)`, which runs under the store's lock in one step.
3. `rollback()` pops the innermost and throws it away.
- **Failure and undo:** commit or rollback with no open transaction returns False. A rollback needs no undo, because nothing was ever written outside the layer.

### Automatic cleanup
1. Each interval, `purge_expired()` pops every queue entry whose time has come.
2. For each, the key is deleted only if it still has that exact expiry. *Why:* if the key was set again since, the old queue entry is stale and must not delete the new value.

### Concurrency (requirement 7)
- **Store:** the lock sits inside `KeyValueStore`, around `get`, `set`, `delete`, `apply` and `purge_expired`. A commit's changes are applied in one hold, so no reader sees half of a commit.
- **Sessions:** no lock. Each session belongs to one thread, and its layers are private, which is what keeps uncommitted changes hidden.
- **Cleaner:** it takes the same store lock, so cleanup never interleaves with a commit.
- **Lock order:** there is only one lock, so no deadlock is possible.
- **Two sessions changing the same key:** the later commit wins. Detecting the clash would be optimistic versioning (follow-up question 1).

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A different cleanup interval | A different `interval_ms` | No. It is data. |
| Values other than text | Use another value type in `Entry` and `Change` | Type change only. |
| A test clock | A new class implementing `Clock` | No. |
| Reading a key's remaining time to live | A new read-only method on `KeyValueStore` and `Session` | Additions only. |

## 11. Interview Notes

### Key decisions
1. **Pending changes live in private layers, and only commits touch the store.**
   - *Why:* other sessions can't see uncommitted changes, and a rollback has nothing to undo.
   - *Rejected:* writing to the store and keeping an undo log. Other sessions would see the changes before the commit.
2. **Each nested transaction is one more layer.**
   - *Why:* nesting, inner commit and inner rollback all fall out of a stack.
   - *Rejected:* copying the whole data set at each `begin`, which costs memory in proportion to the store.
3. **The outermost commit is applied under one hold of the store's lock.**
   - *Why:* readers see a commit all at once or not at all.
   - *Rejected:* applying changes one by one with the lock taken each time.
4. **Expiry is checked on every read and cleaned in the background.**
   - *Why:* correctness never depends on when the cleaner runs, and memory is still reclaimed.
   - *Rejected:* relying on the cleaner alone, which could return an expired key.
5. **The cleaner uses a min-heap of expiry times and skips stale entries.**
   - *Why:* it never scans every key, and a key set again is never deleted by its old expiry.
   - *Rejected:* scanning the whole map each time.

### Edge cases handled
- A key set again without a time to live is not deleted by its old expiry.
- A delete inside a transaction hides the committed value from that session only.
- A key set with a time to live inside a transaction that expires before commit is applied as a delete.
- Commit or rollback with no open transaction returns False.
- An inner commit keeps changes hidden until the outermost commit.
- Rolling back an inner transaction leaves the outer one's changes in place.
- Deleting a missing or expired key returns False.
- A time to live of 0 or less is refused.

### Likely follow-up questions
1. **"How would you detect two sessions changing the same key?"** Give each entry a version number. Record the version each read saw, and at commit, refuse if any of those keys has a newer version. That is optimistic concurrency.
2. **"Can reads run in parallel?"** Replace the store's lock with a read-write lock, so reads share it and only changes, commits and cleanup take it alone.
3. **"What if one session is shared by several threads?"** Add a lock inside `Session` around every method. Today a session belongs to one client thread.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Detecting conflicting commits | The later commit wins. See follow-up question 1. |
| Parallel reads | See follow-up question 2. |
| A session shared by threads | See follow-up question 3. |
| Saving to disk | It is in memory, as the requirement says. |
| A memory limit or eviction | Not required. It would be an eviction policy like an LRU cache. |
| Changing only the time to live of a key | Not required. It is a new store method. |

## 13. Requirements Met
1. Set, get, delete: `KeyValueStore.set`, `get` and `delete`, through `Session`.
2. Optional time to live, expired keys not returned: `ttl_seconds`, `Entry.is_expired`, `Change.is_expired`, and the checks in every `get`.
3. Automatic cleanup: `KeyValueStore.purge_expired`, the `expiry_queue`, and `ExpiryCleaner`.
4. Begin, commit, roll back: `Session.begin`, `commit` and `rollback`.
5. Nesting: the stack of `TransactionLayer` and `merge_into`.
6. Hidden until commit, discarded on rollback: private layers per session, and `KeyValueStore.apply` only at the outermost commit.
7. Correct under many threads: the lock inside `KeyValueStore` and one-step `apply`.

## 14. Requirement Fit Check

**Setup used in every example:** a test clock starting at 10:00:00, one `KeyValueStore`, and sessions `X` and `Y` on it.

1. **Set, get, delete**
   - *Success:* `X.set("a", "1", None)`, then `Y.get("a")` returns `1` because there is no transaction. `X.delete("a")` returns True, and `Y.get("a")` returns None.
   - *Failure:* `X.delete("a")` again returns False. `X.set("b", None, None)` returns False.
2. **Time to live**
   - *Success:* at 10:00:00, `X.set("s", "x", 10)`. At 10:00:05, `get("s")` returns `x`.
   - *Failure:* at 10:00:10, `get("s")` returns None even if the cleaner has not run. `set("t", "1", 0)` returns False.
3. **Automatic cleanup**
   - *Success:* at 10:00:11, `purge_expired()` removes `s` and returns 1.
   - *Failure to avoid:* `set("t", "1", 10)` at 10:00:00, then `set("t", "2", None)`. At 10:00:11 the queue entry for `t` is stale, so `t` is kept with value `2`.
4. **Begin, commit, roll back**
   - *Success:* `X.begin()`, `X.set("a", "2", None)`, `X.commit()` returns True, and the store has `a = 2`. `X.begin()`, `X.set("b", "9", None)`, `X.rollback()` returns True, and `b` does not exist.
   - *Failure:* `X.commit()` with no open transaction returns False.
5. **Nesting**
   - *Success:* `X.begin()` (T1), `X.set("a", "5", None)`, `X.begin()` (T2), `X.set("a", "6", None)`. `X.get("a")` returns `6`. `X.rollback()` drops T2, and `X.get("a")` returns `5`. `X.begin()` (T3), `X.set("c", "1", None)`, `X.commit()` merges T3 into T1. `X.commit()` then applies `a = 5` and `c = 1` to the store.
   - *Failure to avoid:* after T3's commit but before T1's, `Y.get("c")` still returns None.
6. **Hidden until commit, discarded on rollback**
   - *Success:* with `a = 2` in the store, `X.begin()` and `X.set("a", "7", None)`. `X.get("a")` returns `7`, while `Y.get("a")` still returns `2`. After `X.commit()`, `Y.get("a")` returns `7`.
   - *Failure to avoid:* if X had rolled back instead, Y would never have seen `7`.
7. **Correct under many threads**
   - *Success:* X commits `{x = 1, y = 1}` while Y keeps reading `x` and `y` on another thread. The commit is applied under one hold of the lock, so Y sees either both old values or both new ones, never one of each.
   - *Failure to avoid:* without the single locked `apply`, Y could see `x = 1` with the old `y`.
