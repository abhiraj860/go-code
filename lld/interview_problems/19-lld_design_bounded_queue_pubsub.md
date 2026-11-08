# Thread-Safe Bounded Queue / In-Memory Pub-Sub Engine: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** both parts are built on one tool, a lock with condition variables (a "monitor"). Waiting threads sleep on a condition and are woken when the thing they wait for changes. The queue hands each item to exactly one consumer. The pub-sub engine keeps one ordered log per topic, and each subscriber only keeps its own position in that log, so every subscriber reads every message at its own pace, and nobody copies or waits for anybody else.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The queue should hold a fixed number of items.
2. Producers should add items, and should wait when the queue is full.
3. Consumers should remove items in the order they were added, and should wait when the queue is empty.
4. Multiple producers and consumers should work at the same time without losing or duplicating items.
5. Adding and removing should support an optional timeout.
6. Publishers should be able to publish messages to named topics.
7. Subscribers should be able to subscribe to and unsubscribe from topics.
8. Every subscriber of a topic should receive each message published after it subscribed.
9. Each subscriber should consume at its own pace without affecting others.
10. Messages within a topic should be delivered in the order they were published.
11. The system should shut down cleanly.

**Types and conventions**
- Timeouts are `int` milliseconds, or None to wait without a limit.
- `T` is the item type of the queue. An item is never None, so None can mean "nothing" (a timeout or a shut-down).
- A message payload is a `str`.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal or timeout with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. A `Condition` is a waiting room tied to a lock: a thread holding the lock can `wait` (optionally with a timeout), which releases the lock while it sleeps, and another thread can `notify_all` to wake every waiter. A woken thread always checks its reason again, because a wake can happen with nothing changed.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| BoundedBlockingQueue | Holds up to a fixed number of items, in order, making producers and consumers wait when needed | the items, closed flag |
| Message | Records one published message | an unchangeable record |
| Topic | Keeps one topic's ordered log and each subscriber's position in it | the log, the positions |
| Broker | Creates topics and routes publish, subscribe, poll and shutdown to them | the topics |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Queue, item, producer, consumer, timeout, publisher, message, topic, subscriber, order, shutdown.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* BoundedBlockingQueue, Message, Topic and Broker are classes. Producers, consumers, publishers and subscribers are callers and ids, not classes.
2. **Find fixed sets of values.** None. Open and closed is one flag.
3. **Give every fact one owner.** The queue's items belong to the queue. A topic's messages and each subscriber's position belong to the Topic. The set of topics belongs to the Broker.
   - *Why:* two copies of one fact will eventually disagree. A subscriber's position is one number in one place.
4. **Mark what can be computed and do not store it.** "Is it full" and "is it empty" come from the item count. "Does this subscriber have something to read" comes from its position and the log's end.
5. **Give every verb to the class whose data changes.** Putting and taking change the queue. Publishing, subscribing and reading change the topic.
6. **Choose the waiting tool.** One lock per queue or topic, and conditions to sleep on. This is the monitor approach.
7. **Choose how subscribers get messages.** Requirement 9 rules out pushing into each subscriber's own bounded queue, because a full one would make the publisher wait and slow everyone. A shared log with a position per subscriber lets everyone read at their own pace.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** It is the whole problem, so section 8 says exactly where each lock sits.
10. **Fix the outputs.** A `bool` for put, an item or None for take, an offset or None for publish, a message or None for poll.

## 4. Pattern Guide

### Used: Producer–Consumer (with a monitor)
- **Meaning:** producers put work into a shared buffer and consumers take it out. The buffer makes each side wait when it can't proceed.
- **Hint phrases:** "producers should add items, and should wait when the queue is full" and "consumers should ... wait when the queue is empty".
- **How the waiting works:** one lock and two conditions, `not_full` for producers and `not_empty` for consumers.

### Used: Publish–Subscribe, read by position (the "log" variant of Observer)
- **Meaning:** publishers send to a named topic without knowing who listens, and every subscriber gets every message.
- **Hint phrases:** "publish messages to named topics", "subscribe to and unsubscribe from topics", and "each subscriber should consume at its own pace".
- **Why reading by position and not pushing:** the classic Observer pushes each message to every listener, so a slow listener slows the publisher. Here the publisher only appends to the log, and each subscriber pulls from its own position.
- **When NOT to use it:** when there is one consumer per message. Use the queue then.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| Strategy | "a rule that can be replaced" | Not used. Nothing asks for a swappable rule. |
| State | "behaves differently depending on its state" | Not used. Closed is one flag checked in each method. |
| Factory | "create different kinds of objects by type" | Not used. |
| Singleton | "exactly one instance" | Not used. |

## 5. Classes (smallest dependency first)

### BoundedBlockingQueue (part 1)
**Why this class exists:** requirements 1 to 5 describe one shared buffer. It owns its items and does all the waiting, so producers and consumers only call `put` and `take`.
- Attributes:
  - `capacity: int`. *Why:* the fixed limit of requirement 1.
  - `items: list[T]`. *Why:* the waiting items, oldest first. Used as a ring buffer (or a double-ended list) so adding at the back and taking from the front are both constant time.
  - `closed: bool`. *Why:* once true, no new items are accepted and nobody waits any more.
  - `lock: Lock`. *Why:* only one thread changes the items at a time.
  - `not_full: Condition`, `not_empty: Condition`. *Why:* producers sleep on the first and consumers on the second, so each side is woken only by the change it needs.
- Constructor: `BoundedBlockingQueue(capacity: int)`. *Why:* it refuses `capacity < 1`. It starts empty and open, and creates the lock and both conditions on that lock.
- Methods (each holds the lock):
  - `put(item: T, timeout_ms: int or None) -> bool`. *Why:* False at once if `item` is None or the queue is closed. While the queue is full and open, wait on `not_full` for whatever time is left. If the time runs out, return False. If it was closed while waiting, return False. Otherwise add the item at the back, `not_empty.notify_all()`, and return True.
  - `take(timeout_ms: int or None) -> T or None`. *Why:* while the queue is empty and open, wait on `not_empty` for whatever time is left, returning None if the time runs out. If it is empty and closed, return None. Otherwise remove the front item, `not_full.notify_all()`, and return it.
  - `close() -> None`. *Why:* sets `closed`, then wakes everyone on both conditions so no thread waits forever. Items already inside can still be taken.
  - `size() -> int`, `is_closed() -> bool`. *Why:* read-only views, under the lock.
- *Why "while" and not "if" around each wait:* a woken thread re-checks, so it never acts on a change another thread already used up.
- *Extensibility:* none needed.

### Message (part 2)
**Why this class exists:** a published message is a fixed record with its place in the topic's order, so readers can't alter it and positions are easy to compare.
- Attributes: `topic: str`, `offset: int`, `payload: str`. *Why:* which topic, its position in that topic (0, 1, 2, ...), and the content.
- Constructor: `Message(topic: str, offset: int, payload: str)`. *Why:* all set once. It refuses a negative offset or a None payload.
- Methods: `get_topic() -> str`, `get_offset() -> int`, `get_payload() -> str`. *Why:* read-only views.
- *Extensibility:* a time or a key is an extra field here.

### Topic (part 2)
**Why this class exists:** requirements 8 to 10 are all about one topic's messages and who has read how far. One class with one lock owns both, so order and positions can't disagree.
- Attributes:
  - `name: str`. *Why:* the topic's identity.
  - `log: list[Message]`. *Why:* the messages still needed by some subscriber, in publish order.
  - `base_offset: int`. *Why:* the offset of the first message still in `log`. Older ones have been trimmed.
  - `next_offset: int`. *Why:* the offset the next published message gets.
  - `positions: map{str: int}` (a subscriber id to the offset of the next message that subscriber will read). *Why:* the only per-subscriber fact. It is one number, so a slow subscriber costs nothing extra.
  - `closed: bool`. *Why:* after shutdown, no new messages and no waiting.
  - `lock: Lock`, `has_new: Condition`. *Why:* one lock guards the log and the positions together. Readers sleep on `has_new`.
- Constructor: `Topic(name: str)`. *Why:* it refuses an empty name. Both offsets start at 0, with an empty log and no subscribers.
- Methods (each holds the lock):
  - `publish(payload: str) -> int or None`. *Why:* None if closed. Otherwise append `Message(name, next_offset, payload)`, add 1 to `next_offset`, `has_new.notify_all()`, and return the message's offset. The lock gives each message a unique place, which is requirement 10's order.
  - `subscribe(subscriber_id: str) -> bool`. *Why:* False if already subscribed or closed. Otherwise the position is `next_offset`, so the subscriber gets exactly the messages published from now on (requirement 8).
  - `unsubscribe(subscriber_id: str) -> bool`. *Why:* False if not subscribed. Otherwise remove the position and trim.
  - `read(subscriber_id: str, timeout_ms: int or None) -> Message or None`. *Why:* None if not subscribed. While the subscriber's position equals `next_offset` and the topic is open, wait on `has_new` for whatever time is left, returning None if it runs out. If there is nothing to read and the topic is closed, return None. Otherwise take the message at `position - base_offset`, add 1 to the position, trim, and return it.
  - `close() -> None`. *Why:* sets `closed` and wakes every reader. Messages already published can still be read.
  - Private `trim() -> None`. *Why:* drops messages below the smallest position, or all of them if there are no subscribers, and moves `base_offset` up. Memory then holds only what someone still needs.
- *Extensibility:* none needed.

### Broker (the manager and only entry point of part 2)
**Why this class exists:** publishers and subscribers name topics by string, and shutdown must reach every topic. The broker owns the set of topics and turns a name into a topic once.
- Attributes:
  - `topics: map{str: Topic}` (a topic name to the Topic). *Why:* the one record of topics. It is a thread-safe map.
  - `closed: bool`, `lock: Lock`. *Why:* shutdown happens once, and new topics can't appear during it.
- Constructor: `Broker()`. *Why:* it starts with no topics and open.
- Methods:
  - `create_topic(name: str) -> bool`. *Why:* False if the name exists or the broker is closed.
  - `publish(topic_name: str, payload: str) -> int or None`. *Why:* looks up the topic once and asks it. None if the topic is unknown or closed.
  - `subscribe(topic_name: str, subscriber_id: str) -> bool`, `unsubscribe(topic_name: str, subscriber_id: str) -> bool`. *Why:* look up once and ask the topic.
  - `poll(topic_name: str, subscriber_id: str, timeout_ms: int or None) -> Message or None`. *Why:* looks up once and asks the topic to read. Each subscriber polls on its own thread, at its own pace.
  - `shutdown() -> None`. *Why:* requirement 11. Under the lock it marks the broker closed, then closes every topic, which wakes every waiting reader. Calling it again does nothing.
- *Extensibility:* none needed.

## 6. Relationships
- Broker **has** Topic (created inside `create_topic`).
- Topic **has** Message (created inside `publish`, and dropped by trimming).
- BoundedBlockingQueue stands alone. It holds items of type `T`, not other classes of this design.
- There are no enums, so there are no enum lines.

## 7. Interaction

### Layers
1. **Part 1 entry point:** BoundedBlockingQueue, called directly by producer and consumer threads.
2. **Part 2 entry point:** Broker, called by publisher and subscriber threads.
3. **Part 2 data owner:** Topic, which guards its log and positions.
4. **Plain data:** Message.

### Call chains
- **Queue**
  1. `Producer -> BoundedBlockingQueue.put(item, timeout)`: waits on `not_full` if needed, then wakes consumers.
  2. `Consumer -> BoundedBlockingQueue.take(timeout)`: waits on `not_empty` if needed, then wakes producers.
- **Publish**
  1. `Publisher -> Broker.publish(topic_name, payload)`.
  2. `Broker -> Topic.publish(payload)`: appends and wakes readers. It never waits for a subscriber.
- **Subscribe and poll**
  1. `Subscriber -> Broker.subscribe(topic_name, subscriber_id)`, then `Topic.subscribe(...)`: the position starts at the end.
  2. `Subscriber -> Broker.poll(topic_name, subscriber_id, timeout)`, then `Topic.read(...)`: waits on `has_new` if needed, then returns the next message and trims.
- **Shutdown**
  1. `Owner -> Broker.shutdown()`, then `Topic.close()` for each topic.

### Who may call whom
- Producer and consumer threads call `BoundedBlockingQueue` only.
- Publisher and subscriber threads call `Broker` only, and never `Topic`.
- `Broker` calls `Topic`. `Topic` builds `Message`.
- No class calls back up to its owner, so there are no cycles.

## 8. Flows

### Put (producer)
1. `put(item: T, timeout_ms: int or None)`. Take the lock. If `item` is None or the queue is closed, return False.
2. While full and open: wait on `not_full` for the time left. If no time is left, return False.
3. If closed now, return False. Otherwise add at the back, wake consumers, and return True.
- **Failure:** a timeout or a shutdown returns False and the item was never added, so nothing is half-done.

### Take (consumer)
1. `take(timeout_ms: int or None)`. Take the lock.
2. While empty and open: wait on `not_empty` for the time left. If no time is left, return None.
3. If empty and closed, return None. Otherwise remove the front item, wake producers, and return it.
- **Why nothing is lost or duplicated:** the item is removed and returned while the lock is held, so exactly one consumer gets it.

### Publish
1. `publish(topic_name: str, payload: str)`. Look up the topic once. If None, return None.
2. In the topic, under its lock: if closed, return None. Append with the next offset, wake readers, and return the offset.
- **Why the publisher never waits:** it only appends. A slow subscriber is just a smaller number in `positions`.

### Poll (subscriber)
1. `poll(topic_name: str, subscriber_id: str, timeout_ms: int or None)`. Look up the topic once. If None, return None.
2. In the topic: if not subscribed, return None. While nothing is new and the topic is open, wait on `has_new` for the time left. If none is left, return None.
3. If nothing is new and the topic is closed, return None. Otherwise return the message at the subscriber's position, move the position on, and trim.
- **Why the order holds:** offsets were given out under the lock in publish order, and each subscriber reads them one after another.

### Unsubscribe
1. Remove the subscriber's position, then trim. Messages that only they still needed are freed.

### Shut down cleanly
1. `queue.close()`: no more puts, waiting producers get False, consumers drain what is left and then get None.
2. `broker.shutdown()`: no more topics or messages, waiting readers are woken, and each subscriber can drain what it has not read yet and then gets None.
- **What "clean" means here:** no thread is left waiting forever, nothing already accepted is lost, and nothing new is accepted.

### Concurrency
- **Queue:** the lock sits inside `BoundedBlockingQueue`, around every method. Two conditions on that lock make producers and consumers wait separately.
- **Topic:** the lock sits inside each `Topic`, around its log and positions. Different topics never wait for each other.
- **Broker:** the lock sits inside `Broker`, around creating topics and shutdown only. It is released before a topic is touched. `topics` is a thread-safe map.
- **Lock order:** no method ever holds two locks at once, apart from `shutdown`, which always takes the broker lock and then one topic lock at a time, and no method takes them in the other order. So no deadlock is possible.
- **Spurious wake-ups:** every wait is inside a `while` that re-checks the reason, so a thread never acts on a change that is not there.

## 9. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| More topics | Calls to `create_topic` | No. It is data. |
| A bigger or smaller queue | A different `capacity` | No. It is data. |
| A subscriber that starts from the oldest message still kept | A `subscribe` variant that sets the position to `base_offset` | One new method on `Topic`. |
| Another kind of item in the queue | Use the queue with a different `T` | No. |

## 10. Interview Notes

### Key decisions
1. **One lock and two conditions for the queue.**
   - *Why:* producers wake only consumers and consumers wake only producers, so threads are not woken for nothing.
   - *Rejected:* busy-waiting in a loop, which burns the processor, and one condition for both sides.
2. **A topic is one log, and each subscriber is one position in it.**
   - *Why:* each message is stored once, every subscriber gets it, and a slow subscriber costs one number and never slows the publisher.
   - *Rejected:* a bounded queue per subscriber. A full one would either block the publisher (breaking requirement 9) or drop messages (breaking requirement 8).
3. **The log is trimmed to the slowest position.**
   - *Why:* memory holds only what someone still needs.
   - *Rejected:* keeping every message forever.
4. **Offsets are given out under the topic's lock.**
   - *Why:* every subscriber sees the same order, even with many publishers.
   - *Rejected:* giving out offsets outside the lock, which could put messages in two different orders.
5. **Shutdown closes and wakes, and lets readers drain.**
   - *Why:* no thread waits forever and nothing accepted is lost.
   - *Rejected:* clearing everything at once, which would lose accepted items.

### Edge cases handled
- A full queue with a timeout returns False when the time runs out.
- An empty queue with a timeout returns None when the time runs out.
- A wake with nothing changed is re-checked.
- A subscriber that joins late gets only messages published after it joined.
- A slow subscriber keeps old messages in memory but never slows the publisher or other subscribers.
- Unsubscribing frees messages only that subscriber still needed.
- Publishing to a topic with no subscribers stores nothing.
- Polling an unknown topic or as a non-subscriber returns None.
- After shutdown, puts and publishes are refused, waiting threads are woken, and what was already accepted can still be read.

### Likely follow-up questions
1. **"What if one subscriber never reads?"** The log grows from its position. The usual fix is a limit on how far behind a subscriber may fall, after which it is removed (or skipped forward) and told it lost messages. It is a trade-off between requirement 8 and memory.
2. **"Why `notify_all` and not `notify`?"** It is the safe choice with timeouts and shutdown. It can wake more threads than needed, and each re-checks its reason. `notify` is faster but easy to get wrong.
3. **"How would you deliver to subscribers by callback instead of polling?"** Run one thread per subscriber that polls and calls the callback. The topic stays the same, and a slow callback still affects only its own subscriber.

## 11. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| A limit on how far behind a subscriber may fall | See follow-up question 1. |
| Saving messages to disk | Everything is in memory. A durable log would replace `Topic.log`. |
| Delivery by callback | See follow-up question 3. |
| Several readers sharing one subscription (a consumer group) | Each subscriber id has its own position. A shared position would need its own lock. |
| Deleting topics | Not required. It would close the topic and remove it from the map. |
| Message keys or filtering | Not required. |

## 12. Requirements Met
1. A fixed number of items: `BoundedBlockingQueue.capacity`.
2. Producers wait when full: `put` and the `not_full` condition.
3. Consumers take in order and wait when empty: `take`, the front of `items`, and the `not_empty` condition.
4. Many producers and consumers, nothing lost or duplicated: the queue's lock around every change.
5. Optional timeouts: `timeout_ms` on `put` and `take`.
6. Publish to named topics: `Broker.publish` and `Topic.publish`.
7. Subscribe and unsubscribe: `Broker.subscribe`, `unsubscribe`, and `Topic.positions`.
8. Every subscriber gets every later message: the starting position at `next_offset`, and the shared log.
9. Each at its own pace: one position per subscriber, and a publisher that only appends.
10. Order within a topic: offsets given out under the topic's lock.
11. Clean shutdown: `BoundedBlockingQueue.close` and `Broker.shutdown`, which wake every waiter and let them drain.

## 13. Requirement Fit Check

**Setup used in every example:** a queue with capacity 2. A broker with topic `orders`.

1. **A fixed number of items**
   - *Success:* `put("A", None)` and `put("B", None)` both return True and `size()` is 2.
   - *Failure:* `BoundedBlockingQueue(0)` is refused at construction.
2. **Producers wait when full**
   - *Success:* with A and B inside, a producer calls `put("C", None)` and waits. A consumer takes A, the producer is woken, and C is added.
   - *Failure (timeout):* `put("C", 100)` with no consumer returns False after 100 ms, and C is never added.
3. **Consumers take in order and wait when empty**
   - *Success:* `take` returns A, then B, then C, in the order they were added. A consumer calling `take(None)` on an empty queue waits until a producer adds something.
   - *Failure (timeout):* `take(50)` on an empty queue returns None after 50 ms.
4. **Many producers and consumers**
   - *Success:* 3 producers each put 1000 numbered items and 2 consumers take until all are done. The consumers receive exactly 3000 items, each one once.
   - *Failure to avoid:* without the lock, two consumers could take the same front item, or an item could be overwritten.
5. **Optional timeouts**
   - *Success:* `put` and `take` with `None` wait as long as needed. With a number they give up after that many milliseconds, as shown in examples 2 and 3.
   - *Failure:* none. A timeout is a normal answer of False or None.
6. **Publish to named topics**
   - *Success:* `publish("orders", "o-1")` returns offset 0, and `publish("orders", "o-2")` returns 1.
   - *Failure:* `publish("refunds", "x")` returns None because that topic was never created.
7. **Subscribe and unsubscribe**
   - *Success:* `subscribe("orders", "S1")` returns True. `unsubscribe("orders", "S1")` returns True.
   - *Failure:* `subscribe("orders", "S1")` twice returns False on the second call. `unsubscribe` of an unknown subscriber returns False.
8. **Every subscriber gets every later message**
   - *Success:* S1 subscribes, then `o-1` and `o-2` are published. Then S2 subscribes, and `o-3` is published. S1 polls `o-1`, `o-2`, `o-3`. S2 polls only `o-3`.
   - *Failure to avoid:* S2 must not receive `o-1` or `o-2`, which were published before it joined.
9. **Each at its own pace**
   - *Success:* S2 reads `o-3` at once while S1 has not polled anything. The publisher keeps publishing without waiting. When S1 finally polls, it gets every message from its own position.
   - *Failure to avoid:* a slow S1 must never make the publisher or S2 wait.
10. **Order within a topic**
    - *Success:* two publishers publish at the same moment. The topic's lock gives one offset 3 and the other offset 4, and every subscriber reads them as 3, then 4.
    - *Failure to avoid:* two subscribers must never see the same two messages in different orders.
11. **Clean shutdown**
    - *Success:* `close()` on the queue: a producer waiting on a full queue gets False, and consumers take what is left and then get None. `broker.shutdown()`: S1, waiting in `poll(..., None)`, is woken, reads any messages it had not read yet, then gets None. `publish` now returns None.
    - *Failure to avoid:* no thread is left waiting forever, and no accepted item or message is lost.
