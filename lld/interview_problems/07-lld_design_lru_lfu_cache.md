# LRU / LFU Cache: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** the cache owns the values and the capacity, and a separate eviction policy owns only the order or the use counts of the keys. The cache tells the policy about every insert and every use, and asks it who to evict, so LRU and LFU are the same cache with a different policy. Every step is a hash-map lookup or a linked-list splice, so everything is constant time.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The cache should store key-value pairs up to a fixed capacity.
2. The cache should return the value for a key, or indicate that the key is missing.
3. The cache should allow inserting or updating a key, and evict an entry when it is full.
4. The least recently used cache should evict the entry that was used longest ago.
5. The least frequently used cache should evict the entry with the fewest uses, breaking ties by oldest use.
6. Both reads and writes should run in constant time.
7. The eviction rule should be replaceable.
8. The cache should stay correct when accessed by multiple threads.

**Types and conventions**
- `K` is the key type, and it must be usable as a map key. `V` is the value type, and a value is never None, so None can mean "missing".
- Capacity and counts are `int`.
- A "use" is any read or update of a key. Inserting a new key counts as its first use.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock.
- There are no enums, because every condition in this design is a number.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| UsageNode | Is one link in an ordered chain of keys | its key and its two neighbours |
| UsageList | Keeps keys in order, oldest first, with constant-time add, remove and take-oldest | the order of its keys |
| EvictionPolicy (interface) | Tracks key usage and names the key to evict | none |
| LruPolicy | Evicts the key used longest ago | the recency order |
| LfuPolicy | Evicts the key with the fewest uses, oldest first among ties | each key's use count |
| Cache | Stores the values, enforces the capacity, and keeps the policy informed | the values, the capacity |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Cache, key, value, capacity, entry, eviction rule, thread.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Cache and the policy classes. Key, value and capacity are attributes. An "entry" is just a key and its value in the cache's map.
2. **Find fixed sets of values.** None. The two eviction rules are behaviors, not labels.
   - *Why:* a fixed set of labels is an enum, but behavior that differs by kind is a class.
3. **Give every fact one owner.** Values and capacity belong to Cache. The order or count of uses belongs to the policy.
   - *Why:* two copies of one fact will eventually disagree. The values are only in the cache, so a policy swap never moves data.
4. **Mark what can be computed and do not store it.** "Is the cache full" is `size >= capacity`. The victim is read off the policy's order, not stored.
5. **Give every verb to the class whose data changes.** Storing and evicting a value change the cache's map. Reordering keys or counting uses changes the policy.
6. **Find the structure that makes it constant time.** Ordered keys with O(1) add, remove and take-oldest need a hash map plus a doubly linked list. That is `UsageList`, and both policies reuse it.
7. **Find the hint phrase for a pattern.** "The eviction rule should be replaceable" is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** Requirement 8 names it, so decide exactly where the lock sits (section 8).
10. **Fix the outputs.** `V or None` for a read, `bool` for a write.

## 4. Pattern Guide

### Used: Strategy
- **Meaning:** a rule is a small object behind a shared interface, and the cache uses whichever one it is given.
- **Hint phrase:** "the eviction rule should be replaceable" (and requirements 4 and 5, which name two rules).
- **Without it:** the cache would have an `if` on the rule inside `put`, and a third rule would mean editing the cache.
- **When NOT to use it:** when there is one fixed rule that nobody asked to change.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. The cache does not change behavior with a state. |
| Observer | "notify when something changes" | Not used. No requirement asks for eviction notifications. |
| Factory | "create different kinds of objects by type" | Not used. The caller builds the policy it wants. |
| Facade | "a single simple entry point" | Not added as a separate class. `Cache` is already the single entry point. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. No requirement says one cache only, and a passed-in cache is easier to test. |

## 5. Classes (smallest dependency first)

### UsageNode
**Why this class exists:** a doubly linked list needs a link object that knows its two neighbours. Taking a node out of the middle in constant time is only possible if the node knows both sides.
- Attributes:
  - `key: K or None`. *Why:* which key this link stands for. The two end markers hold no key.
  - `previous: UsageNode or None`, `next: UsageNode or None`. *Why:* the neighbours on each side.
- Constructor: `UsageNode(key: K or None)`. *Why:* a node starts with no neighbours. The list connects them.
- Methods: `get_key() -> K or None`, `get_previous() -> UsageNode or None`, `get_next() -> UsageNode or None`, `set_previous(node: UsageNode or None) -> None`, `set_next(node: UsageNode or None) -> None`.
- *Why this class has setters:* it is the one deliberate exception to "no raw setters". A node is an internal piece of `UsageList`, which is the only class that ever holds one, so no outside caller can misuse the setters.
- *Extensibility:* none needed.

### UsageList
**Why this class exists:** both policies need "keys in order, oldest first, with constant-time add, remove and take-oldest". Writing it once means the two policies can't have different bugs in it.
- Attributes:
  - `nodes: map{K: UsageNode}` (a key to the node for that key). *Why:* finds a key's node in constant time, so removing or moving it needs no search.
  - `head: UsageNode`, `tail: UsageNode`. *Why:* two empty end markers, so adding and removing never has to check for an empty list.
- Constructor: `UsageList()`. *Why:* it connects `head` to `tail`, so the list starts empty and valid.
- Methods:
  - `add_newest(key: K) -> bool`. *Why:* puts the key at the newest end. False if the key is already there, so one key never appears twice.
  - `remove(key: K) -> bool`. *Why:* takes the key out from anywhere in constant time. False if it is not there.
  - `touch(key: K) -> bool`. *Why:* moves an existing key to the newest end (remove, then add). False if it is not there.
  - `pop_oldest() -> K or None`. *Why:* removes and returns the oldest key. None if the list is empty.
  - `contains(key: K) -> bool`, `size() -> int`. *Why:* read-only views.
- *Extensibility:* none needed.

### EvictionPolicy (interface)
**Why it is an interface:** requirement 7 says the rule must be replaceable, and requirements 4 and 5 already name two. The cache depends on these three methods and not on any one rule.
- Methods:
  - `on_insert(key: K) -> bool`. *Why:* the cache calls it when a new key is stored. False if the key is already tracked.
  - `on_access(key: K) -> bool`. *Why:* the cache calls it on every read and update. False if the key is not tracked.
  - `evict() -> K or None`. *Why:* picks the key to remove, stops tracking it, and returns it. None if nothing is tracked.
- *Why the policy never sees values:* it only needs keys. The values stay in the cache, so changing the rule never moves data.
- *Extensibility:* a new rule (first in first out, most recently used) is a new class implementing these three methods.

### LruPolicy (implements EvictionPolicy)
**Why this class exists:** requirement 4. The key used longest ago is the one at the oldest end of a single ordered list.
- Attributes: `usage: UsageList`. *Why:* the recency order, oldest first. It is the only fact the policy needs.
- Constructor: `LruPolicy()`. *Why:* it starts with an empty list.
- Methods:
  - `on_insert(key: K) -> bool`. *Why:* `usage.add_newest(key)`. A new key is the most recently used.
  - `on_access(key: K) -> bool`. *Why:* `usage.touch(key)`. A used key becomes the most recent.
  - `evict() -> K or None`. *Why:* `usage.pop_oldest()`. The oldest key is the one used longest ago.
- *Extensibility:* none needed.

### LfuPolicy (implements EvictionPolicy)
**Why this class exists:** requirement 5. It needs the use count of each key, and among keys with the same count, the one used longest ago.
- Attributes:
  - `frequency: map{K: int}` (a key to its use count). *Why:* the count is the single fact the rule is based on.
  - `buckets: map{int: UsageList}` (a use count to the keys that have that count, oldest use first). *Why:* keys with the same count stay in the order they were last used, so the oldest one is always at the front. A key moving to a higher count joins the newest end of that bucket.
  - `min_frequency: int`. *Why:* the lowest count currently held, so the victim's bucket is found without scanning. It is 0 when nothing is tracked.
- Constructor: `LfuPolicy()`. *Why:* it starts with empty maps and a minimum of 0.
- Methods:
  - `on_insert(key: K) -> bool`. *Why:* False if the key is tracked. Otherwise the count is 1, the key is added to the newest end of bucket 1 (created if missing), and `min_frequency` becomes 1.
  - `on_access(key: K) -> bool`. *Why:* False if untracked. Otherwise take the key out of its bucket. If that bucket is now empty, delete it, and if it was the minimum, the minimum becomes the count plus 1. Then the count goes up by 1 and the key joins the newest end of the next bucket (created if missing).
  - `evict() -> K or None`. *Why:* None if nothing is tracked. Otherwise take the oldest key out of the `min_frequency` bucket, delete that bucket if it is now empty, forget the key's count, and return it. If `min_frequency` points at a bucket that no longer exists, it steps up until it finds one. That only happens if `evict` is called twice in a row with no insert between, which the cache never does, so the cache's path stays constant time.
- *Extensibility:* none needed.

### Cache (the manager and only entry point)
**Why this class exists:** it is the one place that keeps the values and the policy in step. Every insert, use and eviction goes through it, so the policy can never track a key the cache does not hold.
- Attributes:
  - `capacity: int`. *Why:* the fixed limit on entries, from requirement 1.
  - `store: map{K: V}` (a key to its value). *Why:* the only home of the values.
  - `policy: EvictionPolicy`. *Why:* the replaceable rule, passed in.
  - `lock: Lock`. *Why:* makes every cache operation run one at a time (requirement 8). It is needed even for reads, because a read changes the policy's order or counts.
- Constructor: `Cache(capacity: int, policy: EvictionPolicy)`. *Why:* the policy is passed in so the same cache is LRU or LFU, and so tests can use a fake. It refuses `capacity <= 0`. The policy must be new and track no keys.
- Methods (each runs under the lock):
  - `get(key: K) -> V or None`. *Why:* None if the key is missing, with nothing changed. Otherwise it tells the policy the key was used and returns the value.
  - `put(key: K, value: V) -> bool`. *Why:* False if the key or the value is None. If the key exists, it replaces the value and counts a use. If it is new, it makes room first (evicting if the cache is full), then stores it and tells the policy. True when stored.
  - `size() -> int`, `get_capacity() -> int`. *Why:* read-only views.
- *Extensibility:* the policy is passed in, so a new rule needs no change here.

## 6. Relationships
- Cache **uses** EvictionPolicy (passed in, and can be any rule).
- LruPolicy **has** UsageList (created inside the constructor).
- LfuPolicy **has** UsageList (one is created inside for each count).
- UsageList **has** UsageNode (nodes are created inside `add_newest`).
- LruPolicy, LfuPolicy **implement** EvictionPolicy.
- There are no enums, so there are no enum lines.

## 7. Interaction

### Layers
1. **Entry point and manager:** Cache. The only class callers use.
2. **Rule passed in:** EvictionPolicy with LruPolicy and LfuPolicy.
3. **Ordered storage:** UsageList and UsageNode.

### Call chains
- **Read**
  1. `Caller -> Cache.get(key)`.
  2. `Cache -> store`: is the key there? If not, return None.
  3. `Cache -> EvictionPolicy.on_access(key)`: the policy records the use.
  4. `LruPolicy -> UsageList.touch(key)` or `LfuPolicy -> UsageList.remove / add_newest`.
- **Write a new key when full**
  1. `Caller -> Cache.put(key, value)`.
  2. `Cache -> EvictionPolicy.evict()`: the policy names and forgets the victim.
  3. `Cache -> store`: the victim's value is removed, and then the new pair is stored.
  4. `Cache -> EvictionPolicy.on_insert(key)`: the policy starts tracking the new key.

### Who may call whom
- Callers call `Cache` only, and never the policy or the lists.
- `Cache` calls the policy only.
- Policies call `UsageList` only. `UsageList` calls `UsageNode` only.
- A policy must never read or write the cache's values, and never call the cache. No class calls back up to its owner, so there are no cycles.

## 8. Flows

### Read (`get`)
1. `get(key: K)`, under the lock.
2. If `key` is not in `store`, return None. *Why before anything else:* a miss must change nothing, or a missing key could disturb the order.
3. `policy.on_access(key)`.
4. Return `store[key]`.
- **Failure and undo:** the only failure is the miss, which changed nothing.

### Write (`put`)
1. `put(key: K, value: V)`, under the lock. If `key` or `value` is None, return False.
2. If `key` is already in `store`: replace the value, call `policy.on_access(key)`, return True. *Why:* an update is a use, and it never needs an eviction.
3. Otherwise, if `size() >= capacity`: `victim: K or None = policy.evict()`. If None, return False. Remove `victim` from `store`. *Why evict first:* the size can never go above the capacity, even for an instant.
4. `store[key] = value`, then `policy.on_insert(key)`. If `on_insert` returns False (it should not), remove the key from `store` again and return False.
5. Return True.
- **Failure and undo:** steps 1 and 3 refuse before changing anything. In step 4 the undo removes exactly the one entry just stored.

### Why everything is constant time
- **Read:** one map lookup, then `on_access`. LRU does one remove and one add on the list. LFU does one remove and one add on a bucket, and at most one bucket deletion and one minimum update.
- **Write:** one map lookup, at most one `evict` (one `pop_oldest` and a few map operations), one map store, one `on_insert` (one list add). Nothing ever scans the keys.
- **Why the list is the key:** `UsageList` finds a node through the `nodes` map, then unlinks and relinks two neighbours. That is constant time at any size.

### Concurrency (requirement 8)
- **Cache:** the lock sits inside `Cache`, around the whole of `get` and `put`. Each operation checks and changes the map and the policy as one step, so two threads can't both see room for the last entry, and the policy's lists can't be corrupted.
- **Why reads are locked too:** a read changes the order or the counts, so it is not a read-only action.
- **Policies and lists:** they have no lock. Only `Cache`, holding its lock, reaches them.
- **Lock order:** only one lock exists, so no deadlock is possible.

## 9. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new eviction rule (first in first out, most recently used) | A new class implementing `EvictionPolicy`, passed to the `Cache` constructor | No. |
| A different capacity | A different number passed to the `Cache` constructor | No. It is data. |
| Both LRU and LFU in the same program | Two caches built with different policies | No. |
| Different key and value types | Use the cache with other types (`K` and `V` are generic) | No. |

## 10. Interview Notes

### Key decisions
1. **Values live only in `Cache`, and the policy tracks only keys.**
   - *Why:* swapping the rule never moves data, and the policy stays small.
   - *Rejected:* the classic design where each list node also holds the value. The policy would then own the data, and a second rule would need its own copy of the storage.
2. **Three tiny policy methods that the cache calls in step.**
   - *Why:* "something was added", "something was used", "who goes". That is all any rule needs.
   - *Rejected:* a policy that inspects the cache. It would depend on the cache while the cache depends on it.
3. **One `UsageList` used by both policies.**
   - *Why:* constant-time ordered keys are written and tested once.
   - *Rejected:* a separate structure for each policy.
4. **LFU uses count buckets and a minimum count.**
   - *Why:* it gives constant-time eviction and breaks ties by oldest use, because each bucket is in order of use.
   - *Rejected:* a heap, which costs time proportional to the logarithm of the size, and a scan for the minimum, which is linear.
5. **One lock on the whole cache, even for reads.**
   - *Why:* a read changes the order, so a reader/writer lock would not help.
   - *Rejected:* a lock only around writes. Two reads could both reorder the list at once and corrupt it.

### Edge cases handled
- Updating an existing key never evicts.
- A miss changes nothing, and does not count as a use.
- A capacity of 0 or less is refused.
- A value of None is refused, so None always means missing.
- A capacity of 1 evicts on every new key.
- LFU: keys with the same count are evicted oldest-use first.
- A new key in LFU starts at a count of 1 and resets the minimum to 1.
- The size never goes above the capacity.

### Likely follow-up questions
1. **"Why does a read need the lock?"** Because a read moves the key in the order or raises its count. Two threads doing that at once could corrupt the linked list.
2. **"How would you add a time to live?"** Store an expiry time with each value and treat an expired key as missing on read. That means a small change inside `Cache`, because the values and their times must be kept together. Eviction by rule and expiry by time are separate reasons for removal.
3. **"Can reads run in parallel?"** Only by splitting the cache into several independent caches by key, each with its own lock. The trade-off is that the capacity is divided, so it is no longer one global limit.

## 11. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Deleting or invalidating a key | No requirement asks for it. It would be a `remove` on `Cache` and on the policy. |
| Time to live | See follow-up question 2. |
| Storing None as a value | Values are never None, so None can mean missing. A wrapper type would lift this. |
| Hit and miss statistics | No requirement asks for them. They would be counters in `Cache`. |
| Capacity measured in bytes | The capacity counts entries. A size would be a new field with each value. |
| Telling someone when an entry is evicted | No requirement asks for it. It would be a listener called after step 3 of `put`. |
| Reading in parallel | See follow-up question 3. |

## 12. Requirements Met
1. Fixed capacity: `Cache.capacity`, and the eviction in `put` before every new insert.
2. Return a value or indicate missing: `Cache.get`, which returns None for a missing key.
3. Insert or update, and evict when full: `Cache.put` with `EvictionPolicy.evict`.
4. LRU eviction: `LruPolicy`.
5. LFU eviction with ties by oldest use: `LfuPolicy`, with its count buckets.
6. Constant-time reads and writes: the `store` map, `UsageList`, and the count buckets with `min_frequency`.
7. A replaceable eviction rule: the `EvictionPolicy` interface.
8. Correct under many threads: the lock inside `Cache`.

## 13. Requirement Fit Check

**Setup used in every example:** the capacity is 2, and keys and values are shown as letters and numbers.

1. **Fixed capacity**
   - *Success:* `put("A", 1)` and `put("B", 2)` both return True, and `size()` is 2.
   - *Failure:* `Cache(0, LruPolicy())` is refused at construction. After a third `put`, `size()` is still 2.
2. **Return a value or indicate missing**
   - *Success:* `get("A")` returns 1.
   - *Failure:* `get("Z")` returns None, and the order and the counts are unchanged. `put("C", None)` returns False.
3. **Insert or update, and evict when full**
   - *Success (update):* `put("A", 10)` on a full cache replaces the value and evicts nothing. *Success (insert and evict):* the next new key evicts exactly one entry first.
   - *Failure:* the size never goes above 2, even during the eviction, because the victim is removed before the new key is stored.
4. **LRU eviction**
   - *Success:* with `LruPolicy`, `put("A", 1)`, `put("B", 2)`, `get("A")` make A the most recent. `put("C", 3)` finds the cache full and evicts B, the one used longest ago. `get("B")` returns None.
   - *Failure to avoid:* evicting A would be wrong, because A was just used.
5. **LFU eviction, ties by oldest use**
   - *Success (fewest uses):* with `LfuPolicy`, `put("A", 1)`, `put("B", 2)`, `get("A")`. A has 2 uses and B has 1, so `put("C", 3)` evicts B.
   - *Success (tie):* `put("A", 1)`, `put("B", 2)` and nothing else. Both have 1 use, and A was used first, so `put("C", 3)` evicts A.
6. **Constant time**
   - *Success:* for `get("A")` under LRU the work is one map lookup, one list remove and one list add. For `put("C", 3)` under LFU it is one lookup, one `pop_oldest` from the lowest bucket, one map store, and one add to bucket 1. No step looks at every key.
   - *Failure to avoid:* scanning the keys to find the lowest count would be linear time.
7. **A replaceable eviction rule**
   - *Success:* run `put("A", 1)`, `put("B", 2)`, `get("A")`, `get("A")`, `get("B")`, then `put("C", 3)`. With `LruPolicy` A was used longest ago, so A is evicted. With `LfuPolicy` A has 3 uses and B has 2, so B is evicted. Only the policy passed to the constructor differs.
   - *Failure:* none of the caller code changes between the two caches.
8. **Correct under many threads**
   - *Success:* 100 threads call `put` and `get` at the same moment. Each call holds the cache's lock for the whole operation, so the map and the policy always agree and `size()` never exceeds 2.
   - *Failure to avoid:* without the lock, two `put` calls for new keys could both see a size of 1, and both store, giving a size of 3.
