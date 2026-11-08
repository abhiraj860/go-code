# Rate Limiter (In-Memory): Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a rate limiter is a lookup of the right limit, plus a tiny counter for each client and endpoint, and each counter checks and updates itself under its own lock. The way of counting is a swappable object, so changing the algorithm never touches the code that calls the limiter.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The system should limit how many requests a client can make within a time window.
2. The system should decide for each request whether to allow or reject it based on the client's recent usage.
3. The system should support multiple limiting algorithms that can be swapped without changing the callers.
4. Different clients or endpoints should be able to have different limits.
5. The limits should be configurable.
6. The limiter should stay correct when many requests arrive at the same time.

**Types and conventions**
- Time is a `float` number of seconds. The caller passes the current time (`now`) in, so the design never reads a clock and is easy to test. Time is assumed never to go backwards.
- A limit is a whole number of requests per window. A window is a number of seconds.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. A "thread-safe map" supports an atomic `put_if_absent`, which stores a value only if the key is empty and returns whichever value is now stored.
- There are no enums, because every condition in this design is a number.
- A counter is kept for each pair of client and endpoint. A rule chooses the numbers, but counting is never shared between endpoints.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| RateLimitRule | Describes one limit | the limit and the window |
| RuleRegistry | Finds the right rule for a client and endpoint, and lets rules be changed | the rules |
| UsageTracker (interface) | Decides allow or reject for one client and endpoint | none |
| FixedWindowTracker | Counts requests in fixed windows | window start, count |
| SlidingWindowLogTracker | Remembers request times inside the window | the recent request times |
| TokenBucketTracker | Spends and refills tokens | tokens, last refill time |
| LimitAlgorithm (interface) | Creates a fresh tracker of one kind | none |
| FixedWindowAlgorithm | Creates fixed-window trackers | none |
| SlidingWindowLogAlgorithm | Creates sliding-log trackers | none |
| TokenBucketAlgorithm | Creates token-bucket trackers | none |
| RateLimiter | Finds the rule and the tracker for each request and asks the tracker | the trackers |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Limiter, client, request, time window, limit, algorithm, endpoint, usage.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* RateLimitRule, RuleRegistry, the trackers and RateLimiter are classes. Client, endpoint and request are just ids and timestamps, so they are not classes.
2. **Find fixed sets of values.** None. The algorithms are behaviors, not labels.
   - *Why:* a fixed set of labels is an enum, but behavior that differs by kind is a class.
3. **Give every fact one owner.** Each limit belongs to a rule, the set of rules to the registry, each client's usage to its own tracker, and the set of trackers to the limiter.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** Whether a request is allowed is computed from the tracker's state and the rule. Which rule applies is computed from the registry.
   - *Why:* a stored "currently blocked" flag would be wrong the moment the window moves on.
5. **Give every verb to the class whose data changes.** Counting changes a tracker's state, so the tracker owns `try_acquire`. Changing a limit changes the registry.
6. **Find the action that touches several classes.** One request needs the rule, then the tracker, then the check. That order lives in `RateLimiter`.
7. **Find the hint phrase for a pattern.** "Multiple limiting algorithms that can be swapped without changing the callers" is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** Requirement 6 names it, so decide exactly where the locks sit (section 8).
10. **Fix the outputs.** A `bool` for every decision, because that is all the caller needs.

## 4. Pattern Guide

### Used: Strategy
- **Meaning:** a way of doing something is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrase:** "support multiple limiting algorithms that can be swapped without changing the callers".
- **Without it:** the limiter would have an `if` on the algorithm in the middle of its check, and every new algorithm would mean editing it.
- **When NOT to use it:** when there is one fixed way and nobody asked to add another.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A tracker's numbers change, but its behavior does not. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not added as a separate pattern. Each algorithm creates its own tracker, which is part of the strategy itself. |
| Facade | "a single simple entry point" | Not added as a separate class. `RateLimiter` is already the single entry point because it is the manager. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "one shared configuration accessible everywhere" | Not used. No requirement says so, and a passed-in limiter is easier to test. |

## 5. Classes (smallest dependency first)

### RateLimitRule
**Why this class exists:** a limit is two numbers that belong together. As one unchangeable record it can be replaced whole, so a half-changed limit can never be seen.
- Attributes:
  - `limit: int`. *Why:* how many requests are allowed in the window.
  - `window_seconds: float`. *Why:* how long the window is.
- Constructor: `RateLimitRule(limit: int, window_seconds: float)`. *Why:* both fields are set once. It refuses `limit <= 0` or `window_seconds <= 0`, so a meaningless limit cannot exist.
- Methods: `get_limit() -> int`, `get_window_seconds() -> float`. *Why:* read-only views. There are no setters, so changing a limit means replacing the rule.
- *Extensibility:* an extra setting (for example a burst size) is a new field here.

### RuleRegistry
**Why this class exists:** requirements 4 and 5 need one place that says which limit applies to which client and endpoint, and that can be changed while the limiter runs.
- Attributes:
  - `default_rule: RateLimitRule`. *Why:* the limit for anyone with no specific rule, so a rule always exists.
  - `rules: map{str: RateLimitRule}` (a rule key to the rule). *Why:* the specific rules. The key is the client id and the endpoint joined with `|`, and `*` stands for "any". `alice|/search` means one client and one endpoint. `alice|*` means one client on any endpoint. `*|/login` means any client on one endpoint.
  - `lock: Lock`. *Why:* guards the map and the default, so a change and a lookup can't overlap.
- Constructor: `RuleRegistry(default_rule: RateLimitRule)`. *Why:* a registry is useless without a default. It starts with no specific rules and creates its lock.
- Methods:
  - `get_rule(client_id: str, endpoint: str) -> RateLimitRule`. *Why:* it never fails, because the default always exists. It returns the first match of: the exact pair, then the client on any endpoint, then any client on the endpoint, then the default. *Why client before endpoint:* a rule written for one client is more specific than one written for everybody.
  - `set_rule(client_id: str or None, endpoint: str or None, rule: RateLimitRule) -> bool`. *Why:* None means "any". It returns False if both are None (use `set_default_rule` for that) or if a given id is an empty string.
  - `remove_rule(client_id: str or None, endpoint: str or None) -> bool`. *Why:* False if no such rule exists, so the caller knows nothing changed.
  - `set_default_rule(rule: RateLimitRule) -> None`. *Why:* it cannot fail, because the rule object is already valid.
- *Extensibility:* all limits are data here, so changing one needs no code change.

### UsageTracker (interface)
**Why it is an interface:** requirement 3 asks for several ways of counting, so the limiter depends on this one method and not on any specific way.
- Methods: `try_acquire(rule: RateLimitRule, now: float) -> bool`.
  - *Why the rule is passed in on every call:* the tracker keeps only its counting state, so a rule change takes effect on the very next request.
  - *Why it returns a `bool`:* True means allowed and the request was counted. False means rejected and nothing was counted.
- *Extensibility:* a new way of counting is a new class implementing this method.

### FixedWindowTracker (implements UsageTracker)
**Why this class exists:** the simplest algorithm: count requests in fixed blocks of time.
- Attributes:
  - `window_start: float or None`. *Why:* when the current window began. None before the first request.
  - `count: int`. *Why:* requests allowed in the current window.
  - `lock: Lock`. *Why:* makes "check the count, then add one" a single step.
- Constructor: `FixedWindowTracker()`. *Why:* it starts with no window and a count of 0, and creates its lock.
- Methods: `try_acquire(rule: RateLimitRule, now: float) -> bool`.
  - *How, under the lock:* if there is no window, or `now - window_start >= window_seconds`, start a new window at `now` with count 0. Then if `count < limit`, add one and return True. Otherwise return False.
- *Extensibility:* none needed.

### SlidingWindowLogTracker (implements UsageTracker)
**Why this class exists:** a more exact algorithm: remember when each recent request happened.
- Attributes:
  - `timestamps: list[float]`. *Why:* the times of allowed requests inside the window, oldest first.
  - `lock: Lock`. *Why:* makes "drop old, check, add" a single step.
- Constructor: `SlidingWindowLogTracker()`. *Why:* it starts with an empty list and creates its lock.
- Methods: `try_acquire(rule: RateLimitRule, now: float) -> bool`.
  - *How, under the lock:* drop every timestamp at or before `now - window_seconds`. If fewer than `limit` remain, append `now` and return True. Otherwise return False.
- *Extensibility:* none needed.

### TokenBucketTracker (implements UsageTracker)
**Why this class exists:** a smoother algorithm: a bucket that holds up to `limit` tokens and refills steadily.
- Attributes:
  - `tokens: float or None`. *Why:* tokens in the bucket now. None before the first request, when the bucket starts full.
  - `last_refill: float`. *Why:* when the tokens were last topped up.
  - `lock: Lock`. *Why:* makes "refill, check, spend" a single step.
- Constructor: `TokenBucketTracker()`. *Why:* it starts with no token count yet and creates its lock.
- Methods: `try_acquire(rule: RateLimitRule, now: float) -> bool`.
  - *How, under the lock:* on the first call, set the tokens to `limit` and `last_refill` to `now`. Otherwise add `(now - last_refill) * limit / window_seconds` tokens, never going above `limit` and never below the old value if the time difference is negative. Set `last_refill` to `now`. If `tokens >= 1`, take one and return True. Otherwise return False.
- *Extensibility:* none needed.

### LimitAlgorithm (interface)
**Why it is an interface:** requirement 3 says the algorithm must be swappable without changing callers. The limiter asks this interface for a fresh tracker whenever it meets a new client and endpoint.
- Methods: `create_tracker() -> UsageTracker`. *Why it needs no inputs:* a tracker holds only counting state, and the rule is passed in on every call.
- *Extensibility:* a new algorithm is a new class implementing this method, plus its tracker.

### FixedWindowAlgorithm (implements LimitAlgorithm)
**Why this class exists:** it chooses fixed-window counting.
- Attributes: none. Constructor: `FixedWindowAlgorithm()`.
- Methods: `create_tracker() -> UsageTracker`. *Why:* returns a new `FixedWindowTracker`.
- *Extensibility:* none needed.

### SlidingWindowLogAlgorithm (implements LimitAlgorithm)
**Why this class exists:** it chooses sliding-log counting.
- Attributes: none. Constructor: `SlidingWindowLogAlgorithm()`.
- Methods: `create_tracker() -> UsageTracker`. *Why:* returns a new `SlidingWindowLogTracker`.
- *Extensibility:* none needed.

### TokenBucketAlgorithm (implements LimitAlgorithm)
**Why this class exists:** it chooses token-bucket counting.
- Attributes: none. Constructor: `TokenBucketAlgorithm()`.
- Methods: `create_tracker() -> UsageTracker`. *Why:* returns a new `TokenBucketTracker`.
- *Extensibility:* none needed.

### RateLimiter (the manager and only entry point)
**Why this class exists:** one request needs the rule, then the tracker for that client and endpoint, then the decision, in that order. No tracker or rule can do that alone.
- Attributes:
  - `rules: RuleRegistry`. *Why:* which limit applies. It is passed in, so an administrator who holds the same registry can change limits while the limiter runs.
  - `algorithm: LimitAlgorithm`. *Why:* the swappable way of counting, passed in.
  - `trackers: map{str: UsageTracker}` (a key made of the client id and the endpoint joined with `|`, to the tracker for that pair). *Why:* the one record of every pair's usage. It is a thread-safe map.
- Constructor: `RateLimiter(rules: RuleRegistry, algorithm: LimitAlgorithm)`. *Why:* both collaborators are passed in, so a test can use a small registry and a fake algorithm. It starts with no trackers.
- Methods:
  - `allow(client_id: str, endpoint: str, now: float) -> bool`. *Why public:* it is the only thing callers use, and it is safe at any time. An empty client id or endpoint is rejected with False, because nothing can be counted for it.
- *Extensibility:* the algorithm and the rules are both passed in, so neither requires a change here.

## 6. Relationships
- RuleRegistry **uses** RateLimitRule (rules are unchangeable records that can be shared, and the registry just holds the ones it is given).
- RateLimiter **has** UsageTracker (trackers are created inside `allow`, kept in the map, and die with the limiter).
- RateLimiter **uses** RuleRegistry, LimitAlgorithm (both passed in).
- FixedWindowTracker, SlidingWindowLogTracker, TokenBucketTracker **implement** UsageTracker.
- FixedWindowAlgorithm, SlidingWindowLogAlgorithm, TokenBucketAlgorithm **implement** LimitAlgorithm.
- There are no enums, so there are no enum lines.
- Each algorithm creates its tracker and each tracker takes a rule as a method input, but no attribute holds those classes there, so they have no line.

## 7. Interaction

### Layers
1. **Entry point and manager:** RateLimiter. The only class callers use.
2. **Configuration:** RuleRegistry and RateLimitRule.
3. **Rules passed in:** LimitAlgorithm and its three implementations.
4. **Counting state:** the trackers.

### Call chains
- **Check a request**
  1. `Caller -> RateLimiter.allow(client_id, endpoint, now)`.
  2. `RateLimiter -> RuleRegistry.get_rule(client_id, endpoint)`: the limit is found first.
  3. `RateLimiter -> trackers.get(key)`: the tracker for this pair, if one exists.
  4. If none exists: `RateLimiter -> LimitAlgorithm.create_tracker()`, then `trackers.put_if_absent(key, tracker)`: the map keeps exactly one winner.
  5. `RateLimiter -> UsageTracker.try_acquire(rule, now)`: the tracker decides and counts under its own lock.
- **Change a limit**
  1. `Administrator -> RuleRegistry.set_rule(client_id, endpoint, rule)`. The limiter is not involved, and the next request sees the new rule.

### Who may call whom
- Callers call `RateLimiter.allow` only. Administrators call `RuleRegistry` only.
- `RateLimiter` calls `RuleRegistry`, `LimitAlgorithm` and `UsageTracker`.
- Algorithms call nothing but the tracker constructors. Trackers call no other class.
- No class calls back up to its owner, so there are no cycles.

## 8. Flows

### Check a request
1. `allow(client_id: str, endpoint: str, now: float)`. If either id is empty, return False.
2. `rule: RateLimitRule = rules.get_rule(client_id, endpoint)`. *Why first:* the tracker needs it, and it can never fail.
3. `key: str` is the client id and the endpoint joined with `|`.
4. `tracker: UsageTracker or None = trackers.get(key)`. If None, create one with `algorithm.create_tracker()` and store it with `put_if_absent`. Use whatever tracker `put_if_absent` returns. *Why:* if two requests race to create the first tracker, both end up using the same one.
5. Return `tracker.try_acquire(rule, now)`.
- **Failure and undo:** the only failure is the empty-id refusal at step 1, which changed nothing. A rejection at step 5 counts nothing, so there is nothing to undo.

### Change or remove a limit
1. `set_rule(client_id, endpoint, rule)` or `remove_rule(client_id, endpoint)` on the registry, under its lock.
- **Failure:** returns False and nothing changes. Existing trackers keep their state and use the new rule from their next call.

### Concurrency (requirement 6)
- **Tracker:** the lock sits inside each tracker, around the whole of `try_acquire`. "Check the count, then add one" is one step, so two simultaneous requests can never both take the last slot.
- **Different keys:** each client and endpoint has its own tracker and lock, so requests for different keys never wait for each other.
- **Tracker creation:** `trackers` is a thread-safe map, and `put_if_absent` makes first-time creation safe. A tracker that loses the race is simply discarded.
- **Registry:** the lock sits inside `RuleRegistry`, around its map and default. A rule change and a lookup can't overlap.
- **Lock order:** no method ever holds two locks at once. `allow` finishes with the registry lock before it takes a tracker lock, so no deadlock is possible.

## 9. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new algorithm (leaky bucket, sliding window counter) | A new tracker class and a new `LimitAlgorithm` class, passed to the `RateLimiter` constructor | No. |
| A different limit for a client, an endpoint, or a plan (free versus paid) | Calls to `set_rule` | No. It is data. |
| Changing a limit while the limiter runs | A new rule passed to `set_rule` or `set_default_rule` | No. It is data. |
| A different default limit | A different `RateLimitRule` passed to the `RuleRegistry` constructor | No. It is data. |

## 10. Interview Notes

### Key decisions
1. **Two levels: the algorithm creates a tracker, and the limiter manages keys and rules.**
   - *Why:* finding the rule, creating the tracker and handling races are written once, and an algorithm only has to count.
   - *Rejected:* a separate limiter class per algorithm, each with its own map and its own rule lookup. That code would be copied three times.
2. **A tracker holds only counting state, and the rule is passed in on every call.**
   - *Why:* a changed limit applies on the very next request, with no need to tell the trackers.
   - *Rejected:* a tracker that stores its rule. It would keep using the old limit after a change.
3. **One lock per tracker.**
   - *Why:* only requests for the same client and endpoint ever compete.
   - *Rejected:* one lock for the whole limiter. Every request would wait for every other.
4. **A rule is an unchangeable record that is replaced whole.**
   - *Why:* a request can never see a limit of one rule and a window of another.
   - *Rejected:* changing `limit` and `window_seconds` in place.
5. **`put_if_absent` creates a tracker.**
   - *Why:* when two first requests race, both use the one tracker that won.
   - *Rejected:* check-then-insert, which could give each request its own tracker and let both through.

### Edge cases handled
- The first request for a pair creates its tracker.
- A window that has ended starts a new one.
- A request with an empty client id or endpoint is rejected.
- A rule changed in the middle of a window takes effect at once.
- A client with no specific rule gets the default.
- A token bucket with a fraction of a token refuses until a whole token is available.
- Many simultaneous requests for one pair: exactly the allowed number succeed.
- A client on a different endpoint has a separate count.

### Likely follow-up questions
1. **"Doesn't a fixed window allow a double burst at the boundary?"** Yes. A client can use the whole limit at the end of one window and again at the start of the next. The sliding log and the token bucket avoid this. The sliding log costs memory in proportion to the limit, and the token bucket keeps just two numbers.
2. **"What happens to trackers for clients that never come back?"** They stay in memory. A fix is to record each tracker's last-used time and sweep out idle ones with a periodic method. It is not built here because no requirement asks for it.
3. **"How would this work across many servers?"** The in-memory trackers would move to a shared store, with the check-and-update done as one atomic operation there. `UsageTracker` is already the boundary where that change goes.

## 11. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| A limit shared across all of a client's endpoints | Counting is per client and endpoint pair. A shared quota would need a tracker keyed by client alone. |
| Telling the caller when to retry or how many requests remain | `allow` returns only allow or reject. A richer answer would be a new return record. |
| Removing idle trackers | See follow-up question 2. |
| Changing the algorithm while running | The algorithm is chosen when the limiter is built. Switching would start fresh counts, so a new limiter replaces the old one. |
| Time going backwards | Time is assumed never to go backwards. |
| Fractional limits | A limit is a whole number of requests. |
| Many servers | See follow-up question 3. |
| Reading the clock | The caller passes the time in, so tests need no clock. |

## 12. Requirements Met
1. Limit requests per client in a window: `RateLimiter.allow` with `RateLimitRule` and the trackers.
2. Allow or reject from recent usage: `UsageTracker.try_acquire`, which returns a `bool`.
3. Swappable algorithms: the `LimitAlgorithm` and `UsageTracker` interfaces, with the three implementations of each.
4. Different limits per client or endpoint: `RuleRegistry.get_rule` and `set_rule`.
5. Configurable limits: `RuleRegistry.set_rule`, `remove_rule` and `set_default_rule`, with `RateLimitRule` as data.
6. Correct under simultaneous requests: the lock inside each tracker, and `put_if_absent` for tracker creation.

## 13. Requirement Fit Check

**Setup used in every example:** the default rule is 3 requests per 10 seconds. Times are in seconds. Unless said otherwise the algorithm is `FixedWindowAlgorithm`.

1. **Limit requests per window**
   - *Success:* `bob` calls `/home` at times 0, 1 and 2, and all three are allowed.
   - *Failure (the rejection):* at time 3 the fourth call is rejected, because the window that started at 0 already has 3. At time 10 a new window starts and the call is allowed.
2. **Decide from recent usage**
   - *Success:* with `SlidingWindowLogAlgorithm`, `bob` is allowed at 0, 1 and 2. At 10.5 the time 0 has aged out, so two times remain (1 and 2) and the call is allowed.
   - *Failure:* at 3 the sliding log holds three times, so the call is rejected.
3. **Swappable algorithms**
   - *Success:* building `RateLimiter(rules, SlidingWindowLogAlgorithm())` instead of `FixedWindowAlgorithm()` changes how requests are counted, and callers still call `allow(...)` exactly the same way. With `TokenBucketAlgorithm`, `bob` is allowed three times at time 0 because the bucket starts full. At time 3 there are only 0.9 tokens (refill is 0.3 per second), so the call is rejected. At time 4 there are 1.2 tokens, so it is allowed.
   - *Failure:* `allow("", "/home", 0)` returns False because the client id is empty. Nothing is counted.
4. **Different limits per client or endpoint**
   - *Success:* with `set_rule("alice", "/search", 2 per 10)`, `alice` is allowed twice on `/search` and then rejected, while `alice` on `/home` still gets 3. With `set_rule("vip", None, 5 per 10)`, `vip` gets 5 on any endpoint. With `set_rule(None, "/login", 1 per 10)`, any other client gets 1 on `/login`.
   - *Failure:* `vip` on `/login` gets 5, not 1, because a client rule is more specific than an endpoint rule. `bob` on `/login` is rejected on the second call within the window.
5. **Configurable limits**
   - *Success:* `set_rule("bob", "/home", 10 per 10)` while `bob` is mid-window. His very next call uses the limit of 10.
   - *Failure:* `RateLimitRule(0, 10)` is refused at construction. `set_rule(None, None, rule)` returns False, because the default has its own method.
6. **Correct under simultaneous requests**
   - *Success:* 100 threads call `allow("bob", "/home", 0)` at the same moment with a limit of 3. Each call takes the tracker's lock for the whole check and count, so exactly 3 return True and 97 return False. If two threads race to create `bob`'s first tracker, `put_if_absent` leaves one winner and both use it.
   - *Failure to avoid:* without the lock, two threads could both read a count of 2 and both be allowed, giving 4 in a window of 3.
