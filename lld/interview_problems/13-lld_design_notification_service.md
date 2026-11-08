# Notification Service: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** sending is split into two halves. The fast half takes the request, checks the user's choices, builds the message from a template, and puts one delivery job per channel on a queue, then returns at once. The slow half is a worker that takes jobs from the queue, calls the real channel, and records the outcome, retrying failures. Every delivery tracks its own status, so one slow or broken channel never hides what happened on the others.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The system should send notifications to users through multiple channels such as email, SMS and push.
2. It should be easy to add new channels.
3. Users should be able to choose which channels and types of notifications they want.
4. Notification content should be created from templates with placeholders.
5. Notifications should have types and priorities.
6. Several interested parties should be notified automatically when an event happens.
7. Failed deliveries should be retried.
8. The system should track the delivery status of each notification.
9. Notifications should be sent without blocking the caller.

**Types and conventions**
- Times are `datetime`, passed in by the caller, so the design never reads a clock. Delays are whole seconds.
- A placeholder in a template is written `{name}`.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put_if_absent`, which stores a value only if the key is empty.
- Talking to the real email, SMS or push service happens only inside the three channel classes. Everything else is plain logic.
- Requirement 9 means several threads act at once: callers send, and workers deliver. A background thread calls the worker's `process_next` in a loop. Starting those threads is infrastructure outside this design.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| RenderedContent | Holds a finished subject and body | subject, body |
| Template | Turns values into a finished message | the subject and body patterns |
| RetryPolicy | Says how long to wait before each retry, and when to give up | max attempts, base delay |
| User | Holds a person's contacts and choices | contacts, enabled channels, enabled types |
| NotificationChannel (interface) | Delivers one message through one kind of channel | none |
| EmailChannel | Sends email | none |
| SmsChannel | Sends SMS | none |
| PushChannel | Sends push notifications | none |
| ChannelRegistry | Finds the channel for a channel type | the channels |
| Delivery | Tracks one notification on one channel | status, attempts, next attempt time |
| Notification | Holds one message for one user and its deliveries | the message, its deliveries |
| DeliveryQueue | Hands out ready deliveries, highest priority first | the waiting jobs |
| DeliveryWorker | Takes one job, delivers it and records the result | none |
| Event | Describes something that happened and who to tell | an unchangeable record |
| EventListener (interface) | Reacts to an event | none |
| NotificationListener | Turns an event into notifications | none |
| EventPublisher | Tells every listener about an event | the listeners |
| NotificationService | Builds and queues notifications, and holds users, templates and notifications | the users, templates, notifications |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Notification, channel, email, SMS, push, user, type, priority, template, placeholder, event, interested party, delivery, status, retry.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Notification, Template, User, the channels, Delivery, Event are classes. Placeholder and status are values.
2. **Find fixed sets of values.** Notification types, priorities, channel kinds, delivery statuses.
   - *Why:* a fixed set is an enum, so a typo like "pending" for "PENDING" cannot exist.
   - *Result:* `NotificationType`, `Priority`, `ChannelType`, `DeliveryStatus`.
3. **Give every fact one owner.** The user's choices belong to the User. The status and attempts of one delivery belong to that Delivery. The message belongs to the Notification.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** A user's deliverable channels are the enabled ones with a contact. A notification's overall status is read from its deliveries. A rendered message is produced from a template and values.
   - *Why:* a stored "overall status" beside the deliveries would go wrong the moment one delivery changes.
5. **Give every verb to the class whose data changes.** Retrying changes a Delivery. Opting out changes a User. Sending through the real service is the channel's.
6. **Find the action that touches several classes.** One send needs the user, the template, the registry and the queue, in order. That is `NotificationService.send`.
7. **Find the hint phrases for patterns.** "Easy to add new channels" is a swappable interface. "Several interested parties notified automatically when an event happens" is Observer. "Without blocking the caller" is a queue with a worker.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirement 9 names it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A `Notification` or None for a send, and a map of statuses for tracking.

## 4. Pattern Guide

### Used: Strategy (channels)
- **Meaning:** a way of doing something is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrase:** "it should be easy to add new channels".
- **Without it:** the worker would have an `if` on the channel kind, and every new channel would mean editing it.
- **When NOT to use it:** when there is one fixed way and nobody asked to add another.

### Used: Observer (events)
- **Meaning:** a publisher keeps a list of listeners and tells every one of them when something happens, without knowing what they do.
- **Hint phrase:** "several interested parties should be notified automatically when an event happens".
- **Without it:** the code that raises an event would have to call each interested party by name, and a new one would mean editing it.
- **When NOT to use it:** when only one party ever cares.

### Used: a queue with a worker (not a named design pattern)
- **Meaning:** the caller puts a job on a queue and moves on, and a separate worker takes jobs off.
- **Hint phrase:** "notifications should be sent without blocking the caller".
- **Without it:** the caller would wait for the slowest channel.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A delivery's status decides which transitions are allowed, and one method checks it. |
| Factory | "create different kinds of objects by type" | Not used. The registry looks a channel up by its type. |
| Facade | "a single simple entry point" | Not added as a separate class. `NotificationService` is already the single entry point. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. A passed-in service is easier to test. |

## 5. Enums

### NotificationType
- Values: `TRANSACTIONAL`, `PROMOTIONAL`, `ALERT`
- *Why:* requirement 5 gives notifications types, and requirement 3 lets users choose the types they want.
- *Extensibility:* a new type is one new value here.

### Priority
- Values: `LOW`, `MEDIUM`, `HIGH`, with ranks 1, 2 and 3.
- Method: `get_rank() -> int`. *Why:* the queue compares ranks to hand out the most important job first.
- *Why ranks live in the enum:* the order of importance is one fact, so it lives in one place.

### ChannelType
- Values: `EMAIL`, `SMS`, `PUSH`
- *Why:* requirement 1 names these channels, and a user's contact details are kept per channel type.
- *Extensibility:* a new channel is one new value here, a new channel class, and one registry entry.

### DeliveryStatus
- Values: `PENDING`, `RETRYING`, `DELIVERED`, `FAILED`
- *Why:* requirement 8 tracks each delivery. `PENDING` is waiting for a first attempt. `RETRYING` has failed at least once and will try again. `DELIVERED` and `FAILED` are final.

## 6. Classes (smallest dependency first)

### RenderedContent
**Why this class exists:** a finished subject and body travel together. As a fixed record, nothing can alter them after they are built.
- Attributes: `subject: str`, `body: str`. *Why:* the two parts of a message. A channel that has no subject, such as SMS, uses only the body.
- Constructor: `RenderedContent(subject: str, body: str)`. *Why:* both set once. It refuses a None value.
- Methods: `get_subject() -> str`, `get_body() -> str`. *Why:* read-only views.
- *Extensibility:* none needed.

### Template
**Why this class exists:** requirement 4 builds content from templates with placeholders. The template owns the patterns and the rule for filling them.
- Attributes: `name: str`, `subject_pattern: str`, `body_pattern: str`. *Why:* the identity and the two patterns, with `{placeholder}` marks.
- Constructor: `Template(name: str, subject_pattern: str, body_pattern: str)`. *Why:* all set once. It refuses an empty name or body.
- Methods:
  - `get_name() -> str`. *Why:* read-only view.
  - `render(values: map{str: str}) -> RenderedContent or None`. *Why:* the map is a placeholder name to its value. It replaces every `{name}` in both patterns. If any placeholder has no value, it returns None, so a half-filled message is never sent.
- *Extensibility:* a new template is data.

### RetryPolicy
**Why this class exists:** requirement 7 retries failed deliveries. How many times and how long to wait are two numbers that belong together, so they live in one place.
- Attributes: `max_attempts: int`, `base_delay_seconds: int`. *Why:* the total number of tries, and the first wait.
- Constructor: `RetryPolicy(max_attempts: int, base_delay_seconds: int)`. *Why:* both set once. It refuses `max_attempts < 1` or a negative delay.
- Methods: `next_delay(attempts_made: int) -> int or None`. *Why:* None once `attempts_made >= max_attempts`, which means give up. Otherwise `base_delay_seconds * 2^(attempts_made - 1)`, so each wait is twice the last.
- *Extensibility:* different numbers are data.

### User
**Why this class exists:** requirement 3 lets a user choose channels and types. The user owns those choices and their contact details, and answers which channels can actually be used.
- Attributes:
  - `user_id: str`, `name: str`. *Why:* the identity and the display name.
  - `contacts: map{ChannelType: str}` (a channel type to the address on it, such as an email address, a phone number or a device token). *Why:* a channel can't be used without an address.
  - `enabled_channels: set[ChannelType]`. *Why:* the channels the user wants. All are on at first.
  - `enabled_types: set[NotificationType]`. *Why:* the types the user wants. All are on at first.
  - `lock: Lock`. *Why:* a user can change a choice while a send is reading it.
- Constructor: `User(user_id: str, name: str)`. *Why:* it refuses empty values, starts with no contacts, with every channel and type on, and creates its lock.
- Methods (each under the lock):
  - `get_user_id() -> str`. *Why:* read-only view.
  - `set_contact(channel_type: ChannelType, address: str) -> bool`. *Why:* False if the address is empty.
  - `set_channel_enabled(channel_type: ChannelType, enabled: bool) -> None`, `set_type_enabled(notification_type: NotificationType, enabled: bool) -> None`. *Why:* the named way to opt in or out. They cannot fail.
  - `wants(notification_type: NotificationType) -> bool`. *Why:* True if that type is enabled.
  - `get_deliverable_channels() -> list[ChannelType]`. *Why:* computed. Channels that are enabled and have an address.
  - `get_address(channel_type: ChannelType) -> str or None`. *Why:* the address for a channel, or None.
- *Extensibility:* none needed.

### NotificationChannel (interface)
**Why it is an interface:** requirement 2 says new channels must be easy to add, so the worker depends on one method and not on any specific channel.
- Methods: `send(address: str, content: RenderedContent) -> bool`. *Why a `bool`:* True if the channel accepted the message. False means it failed, and the worker decides whether to retry.
- *Extensibility:* a new channel is a new class implementing `send`.

### EmailChannel (implements NotificationChannel)
**Why this class exists:** the first channel.
- Attributes: none. Constructor: `EmailChannel()`.
- Methods: `send(address: str, content: RenderedContent) -> bool`. *Why:* sends the subject and body to an email address. This is where a real mail service is called, and a failure returns False.
- *Extensibility:* none needed.

### SmsChannel (implements NotificationChannel)
**Why this class exists:** the second channel.
- Attributes: none. Constructor: `SmsChannel()`.
- Methods: `send(address: str, content: RenderedContent) -> bool`. *Why:* sends only the body to a phone number. This is where a real SMS service is called, and a failure returns False.
- *Extensibility:* none needed.

### PushChannel (implements NotificationChannel)
**Why this class exists:** the third channel.
- Attributes: none. Constructor: `PushChannel()`.
- Methods: `send(address: str, content: RenderedContent) -> bool`. *Why:* sends the subject as a title and the body as text to a device token. This is where a real push service is called, and a failure returns False.
- *Extensibility:* none needed.

### ChannelRegistry
**Why this class exists:** both the service (does this channel exist?) and the worker (give me the channel to send with) need to find a channel by its type. One owner means one answer.
- Attributes: `channels: map{ChannelType: NotificationChannel}` (a channel type to the object that sends on it). *Why:* the single record of which channels exist.
- Constructor: `ChannelRegistry()`. *Why:* it starts with an empty map.
- Methods:
  - `register(channel_type: ChannelType, channel: NotificationChannel) -> bool`. *Why:* False if that type already has a channel.
  - `supports(channel_type: ChannelType) -> bool`, `get(channel_type: ChannelType) -> NotificationChannel or None`. *Why:* the lookup by type.
- *Extensibility:* a new channel is one `register` call.

### Delivery
**Why this class exists:** requirements 7 and 8 retry and track per channel. A notification sent by email, SMS and push can succeed on one and fail on another, so each channel gets its own record.
- Attributes:
  - `delivery_id: str`. *Why:* the identity.
  - `notification_id: str`, `priority: Priority`. *Why:* which notification it belongs to, and how urgent it is, so the queue can order it and the worker can find the message.
  - `channel_type: ChannelType`, `address: str`. *Why:* where to send it.
  - `content: RenderedContent`. *Why:* what to send. It is a shared fixed record.
  - `status: DeliveryStatus`. *Why:* the single fact about progress.
  - `attempts: int`. *Why:* how many tries have been made.
  - `next_attempt_at: datetime or None`. *Why:* the earliest time for the next try. None for a first try.
  - `lock: Lock`. *Why:* makes recording an outcome one step.
- Constructor: `Delivery(delivery_id: str, notification_id: str, priority: Priority, channel_type: ChannelType, address: str, content: RenderedContent)`. *Why:* a delivery is never created without where and what. The status starts `PENDING`, with 0 attempts.
- Methods:
  - `get_delivery_id() -> str`, `get_notification_id() -> str`, `get_priority() -> Priority`, `get_channel_type() -> ChannelType`, `get_address() -> str`, `get_content() -> RenderedContent`, `get_status() -> DeliveryStatus`, `get_attempts() -> int`, `get_next_attempt_at() -> datetime or None`. *Why:* read-only views.
  - `record_attempt(success: bool, now: datetime, policy: RetryPolicy) -> DeliveryStatus or None`. *Why:* the only way a status changes. None if the status is already final. It adds 1 to `attempts`. On success the status is `DELIVERED`. On failure, if `policy.next_delay(attempts)` is None the status is `FAILED`. Otherwise it is `RETRYING` and `next_attempt_at` is `now` plus the delay. It returns the new status.
- *Extensibility:* none needed.

### Notification
**Why this class exists:** one message for one user belongs together with its deliveries, and tracking asks about the notification as a whole.
- Attributes:
  - `notification_id: str`, `user_id: str`. *Why:* the identity and who it is for.
  - `notification_type: NotificationType`, `priority: Priority`. *Why:* requirement 5.
  - `content: RenderedContent`. *Why:* the message, built once from the template.
  - `created_at: datetime`. *Why:* when it was made.
  - `deliveries: list[Delivery]`. *Why:* one per channel used.
- Constructor: `Notification(notification_id: str, user_id: str, notification_type: NotificationType, priority: Priority, content: RenderedContent, created_at: datetime, deliveries: list[Delivery])`. *Why:* a notification always has at least one delivery. It keeps its own copy of the list and refuses an empty list.
- Methods:
  - `get_notification_id() -> str`, `get_user_id() -> str`, `get_priority() -> Priority`. *Why:* read-only views.
  - `get_deliveries() -> list[Delivery]` (returns a copy). *Why a copy:* a caller can't change the list.
  - `get_statuses() -> map{ChannelType: DeliveryStatus}`. *Why:* computed from the deliveries (a channel type to its delivery's status), so it can never disagree with them.
- *Extensibility:* none needed.

### DeliveryQueue
**Why this class exists:** requirement 9 separates taking the request from delivering it, and requirement 5's priorities decide which job goes first. The queue owns the order and the waiting.
- Attributes:
  - `ready: list[Delivery]`. *Why:* jobs that may be tried now, ordered by priority rank (highest first), then by the order they arrived.
  - `delayed: list[Delivery]`. *Why:* jobs waiting for a retry time, ordered by `next_attempt_at`.
  - `lock: Lock`. *Why:* many senders add jobs while workers take them.
- Constructor: `DeliveryQueue()`. *Why:* both lists start empty and the lock is created.
- Methods (each under the lock):
  - `put(delivery: Delivery, now: datetime) -> None`. *Why:* if the delivery has no `next_attempt_at`, or it is not after `now`, it joins `ready`. Otherwise it joins `delayed`.
  - `take_ready(now: datetime) -> Delivery or None`. *Why:* first moves every delayed job whose time has come into `ready`. Then removes and returns the best ready job. None if nothing is ready.
  - `size() -> int`. *Why:* read-only view of how many jobs wait.
- *Extensibility:* none needed.

### DeliveryWorker
**Why this class exists:** the slow half of sending. It is a single step, so any number of background threads can run it, and tests can run it one step at a time.
- Attributes: `queue: DeliveryQueue`, `channels: ChannelRegistry`, `policy: RetryPolicy`. *Why each:* where jobs come from, how to find the channel, and the retry rule. All are passed in.
- Constructor: `DeliveryWorker(queue: DeliveryQueue, channels: ChannelRegistry, policy: RetryPolicy)`.
- Methods:
  - `process_next(now: datetime) -> bool`. *Why:* True if it handled a job, False if nothing was ready. Steps in section 9.
- *Extensibility:* the channels, the queue and the retry rule are all passed in.

### Event
**Why this class exists:** requirement 6 needs a fixed record of "something happened and these people should hear about it".
- Attributes:
  - `event_name: str`. *Why:* what happened, such as "order_shipped".
  - `notification_type: NotificationType`, `priority: Priority`. *Why:* how listeners should treat it.
  - `template_name: str`, `values: map{str: str}` (a placeholder name to its value). *Why:* the template to use and what fills it.
  - `recipient_ids: list[str]`. *Why:* the interested parties.
  - `occurred_at: datetime`. *Why:* when it happened. Listeners use it as the time for sending.
- Constructor: `Event(event_name: str, notification_type: NotificationType, priority: Priority, template_name: str, values: map{str: str}, recipient_ids: list[str], occurred_at: datetime)`. *Why:* all set once. It keeps its own copies of the list and the map, and refuses an empty name or template name.
- Methods: `get_event_name() -> str`, `get_notification_type() -> NotificationType`, `get_priority() -> Priority`, `get_template_name() -> str`, `get_values() -> map{str: str}` (returns a copy), `get_recipient_ids() -> list[str]` (returns a copy), `get_occurred_at() -> datetime`. *Why:* read-only views.
- *Extensibility:* none needed.

### EventListener (interface)
**Why it is an interface:** requirement 6 notifies several interested parties, so the publisher depends on one method and not on any specific party.
- Methods: `on_event(event: Event) -> None`. *Why:* it is called when something happens. A listener that fails must not stop the others, so the publisher guards each call.
- *Extensibility:* a new interested party is a new class implementing `on_event`.

### NotificationListener (implements EventListener)
**Why this class exists:** the first interested party: it turns an event into notifications.
- Attributes: `service: NotificationService`. *Why:* the thing that builds and queues notifications.
- Constructor: `NotificationListener(service: NotificationService)`.
- Methods: `on_event(event: Event) -> None`. *Why:* for each recipient id, it calls `service.send(...)` with the event's type, priority, template and values, using `event.occurred_at` as the time.
- *Extensibility:* none needed.

### EventPublisher
**Why this class exists:** requirement 6 needs one place that keeps the list of listeners and tells them all.
- Attributes: `listeners: list[EventListener]`, `lock: Lock`. *Why:* the interested parties, and a lock because listeners can be added while an event is published.
- Constructor: `EventPublisher()`. *Why:* an empty list and a lock.
- Methods:
  - `subscribe(listener: EventListener) -> bool`, `unsubscribe(listener: EventListener) -> bool`. *Why:* False if it is already there, or not there.
  - `publish(event: Event) -> int`. *Why:* copies the list under the lock, then calls `on_event` on each. A listener that fails is skipped. It returns how many listeners handled the event without failing.
- *Extensibility:* a new listener is one `subscribe` call.

### NotificationService (the manager and only entry point)
**Why this class exists:** a send needs the user, the template, the registry and the queue in a fixed order, and notifications need one place to be stored and found by id.
- Attributes:
  - `users: map{str: User}` (user id to the User), `templates: map{str: Template}` (template name to the Template), `notifications: map{str: Notification}` (notification id to the Notification). *Why:* the one record of each. All are thread-safe maps.
  - `channels: ChannelRegistry`. *Why:* to check a channel exists before queuing a delivery. It is the same registry the worker uses.
  - `queue: DeliveryQueue`. *Why:* where deliveries are put. It is the same queue the worker uses.
  - `counter: AtomicInt`. *Why:* makes ids unique when sends arrive at once.
- Constructor: `NotificationService(channels: ChannelRegistry, queue: DeliveryQueue)`. *Why:* both are passed in so the worker can share them, and tests can use fakes. It starts with no users, templates or notifications.
- Methods (outer ones take ids and look up once):
  - `register_user(user_id: str, name: str) -> User or None`. *Why:* None if the id exists or the values are invalid.
  - `set_contact(user_id: str, channel_type: ChannelType, address: str) -> bool`, `set_channel_enabled(user_id: str, channel_type: ChannelType, enabled: bool) -> bool`, `set_type_enabled(user_id: str, notification_type: NotificationType, enabled: bool) -> bool`. *Why:* each looks up the user once and asks them. False if the user is unknown.
  - `add_template(name: str, subject_pattern: str, body_pattern: str) -> bool`. *Why:* builds the Template. False if the name exists.
  - `send(user_id: str, notification_type: NotificationType, priority: Priority, template_name: str, values: map{str: str}, now: datetime) -> Notification or None`. *Why:* the fast half. None means nothing was queued. It returns at once, with every delivery `PENDING`.
  - `get_statuses(notification_id: str) -> map{ChannelType: DeliveryStatus} or None`. *Why:* requirement 8. None if the id is unknown.
- *Extensibility:* channels, templates and users are all data or passed in.

## 7. Relationships
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

## 8. Interaction

### Layers
1. **Entry points:** NotificationService (sending and tracking) and EventPublisher (events).
2. **Slow half:** DeliveryQueue and DeliveryWorker.
3. **Channels:** ChannelRegistry and the three channels.
4. **Domain records:** User, Template, Notification, Delivery.
5. **Plain data:** RenderedContent, RetryPolicy, Event.

### Call chains
- **Send (fast half)**
  1. `Caller -> NotificationService.send(...)`.
  2. `NotificationService -> User.wants(type)`, then `User.get_deliverable_channels()`: the user's choices decide first.
  3. `NotificationService -> Template.render(values)`: the message is built once.
  4. `NotificationService -> ChannelRegistry.supports(channel_type)`: only usable channels get a delivery.
  5. `NotificationService -> DeliveryQueue.put(delivery, now)`: one job per channel, then it returns.
- **Deliver (slow half)**
  1. `Worker thread -> DeliveryWorker.process_next(now)`.
  2. `DeliveryWorker -> DeliveryQueue.take_ready(now)`: the best ready job.
  3. `DeliveryWorker -> ChannelRegistry.get(channel_type)`, then `NotificationChannel.send(address, content)`.
  4. `DeliveryWorker -> Delivery.record_attempt(success, now, policy)`, and on `RETRYING`, `DeliveryQueue.put(delivery, now)`.
- **Event**
  1. `Event source -> EventPublisher.publish(event)`.
  2. `EventPublisher -> EventListener.on_event(event)` for each listener.
  3. `NotificationListener -> NotificationService.send(...)` for each recipient.

### Who may call whom
- Callers call `NotificationService` and `EventPublisher` only. Worker threads call `DeliveryWorker` only.
- `NotificationService` calls `User`, `Template`, `ChannelRegistry` and `DeliveryQueue`.
- `DeliveryWorker` calls `DeliveryQueue`, `ChannelRegistry`, the channels and `Delivery`.
- `EventPublisher` calls listeners. `NotificationListener` calls `NotificationService`.
- Channels, templates and records call no other class. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Send (fast half)
1. `send(user_id: str, notification_type: NotificationType, priority: Priority, template_name: str, values: map{str: str}, now: datetime)`. `user: User or None = users.get(user_id)`. If None, return None.
2. If `user.wants(notification_type)` is False, return None. *Why:* the user opted out of that type, and sending is not an error.
3. `template = templates.get(template_name)`. If None, return None.
4. `content: RenderedContent or None = template.render(values)`. If None (a placeholder has no value), return None. *Why before queuing:* a half-filled message must never be sent.
5. `channel_types = user.get_deliverable_channels()` filtered by `channels.supports(...)`. If the list is empty, return None.
6. Build one `Delivery` per channel, then the `Notification` with them, and store it in `notifications`. Ids come from `counter`.
7. `queue.put(delivery, now)` for each delivery. Return the Notification at once.
- **Failure and undo:** every check happens before the first queue entry, so a refusal queues nothing and there is nothing to undo.

### Deliver (slow half)
1. `process_next(now: datetime)`. `delivery: Delivery or None = queue.take_ready(now)`. If None, return False.
2. `channel: NotificationChannel or None = channels.get(delivery.get_channel_type())`. If None, treat the attempt as a failure.
3. `success: bool = channel.send(address, content)`. If the channel raises an error, treat it as False.
4. `status = delivery.record_attempt(success, now, policy)`.
5. If the status is `RETRYING`, call `queue.put(delivery, now)`, where the delay in `next_attempt_at` keeps it waiting. Return True.
- **Failure and undo:** a failed send changes only the delivery's own record. `FAILED` after the last attempt is a final state, and it stays visible through tracking.

### Retry timeline
1. With a policy of 3 attempts and a base delay of 30 seconds, the first failure waits 30 seconds, the second waits 60 seconds, and the third failure ends in `FAILED`.

### Track status
1. `get_statuses(notification_id: str)`. Look up once. Return `notification.get_statuses()`.
- **Failure:** an unknown id returns None.

### Publish an event
1. `publish(event: Event)` copies the listener list under the lock, calls `on_event` on each, and counts the ones that did not fail.
2. `NotificationListener.on_event` calls `send` for each recipient. A recipient who opted out, or an unknown user, just gets None and the loop continues.
- **Failure:** a failing listener is skipped and the others still run.

### Concurrency (requirement 9)
- **Queue:** the lock sits inside `DeliveryQueue`, around `put` and `take_ready`. Many senders and many workers can use it at once, and a job is handed to exactly one worker.
- **Delivery:** the lock sits inside `Delivery`, around `record_attempt`, so a status changes one step at a time.
- **User:** the lock sits inside `User`, around reading and changing choices, so a send never sees a half-changed set.
- **Maps and ids:** the three maps are thread-safe maps, and ids come from an atomic counter.
- **Publisher:** the lock sits inside `EventPublisher`, around its list only, and is released before any listener runs.
- **Not blocking the caller:** `send` only does quick work and puts jobs on the queue. The calls to the real channels happen only in `process_next`, on worker threads.
- **Lock order:** no method ever holds two locks at once. The worker releases the queue lock before it takes a delivery lock, so no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new channel (WhatsApp) | One new `ChannelType` value, a new class implementing `NotificationChannel`, and one `register` call | No logic changes. It is an enum value, a class and data. |
| A new notification type | One new `NotificationType` value | No. |
| A new template or new wording | A new `add_template` call | No. It is data. |
| Different retry timing | A different `RetryPolicy` passed to the worker | No. It is data. |
| Another party interested in events (audit, analytics) | A new class implementing `EventListener`, then one `subscribe` call | No. |

## 11. Interview Notes

### Key decisions
1. **A `Delivery` per channel is the unit of retry and status.**
   - *Why:* one notification can succeed by email and fail by SMS, and each needs its own tries and status.
   - *Rejected:* one status per notification. A retry would resend on channels that already worked.
2. **A notification's overall status is computed from its deliveries.**
   - *Why:* one fact in one place.
   - *Rejected:* a stored status on the notification, which would disagree after one delivery changed.
3. **`send` queues and returns, and a worker delivers.**
   - *Why:* the caller never waits for a slow channel.
   - *Rejected:* sending inside `send`, which would block the caller on the slowest gateway.
4. **The queue orders by priority, and holds retries until their time.**
   - *Why:* an urgent alert is not stuck behind a thousand promotions, and a retry does not hammer a failing channel.
   - *Rejected:* a plain first-in first-out list with a sleep before each retry.
5. **A user's choices live in `User`, with usable channels computed.**
   - *Why:* a channel with no address, or switched off, is simply not in the list.
   - *Rejected:* the service checking preferences held in several places.

### Edge cases handled
- A user who opted out of a type gets nothing, and `send` returns None.
- A channel with no address, or no registered class, is skipped.
- A template placeholder with no value refuses the send before anything is queued.
- A retry waits, with the wait doubling, and the last failure ends in `FAILED`.
- A high-priority job is taken before older low-priority ones.
- A listener that fails does not stop the others.
- A channel that throws an error counts as a failed attempt.
- An unknown user, template or notification id returns None.

### Likely follow-up questions
1. **"Can a notification be delivered twice?"** Yes, if a channel accepted a message but the answer was lost, the worker retries it. The fix is an idempotency key sent with each delivery so the channel can ignore a repeat.
2. **"What if the process crashes with jobs on the queue?"** The queue is in memory, so those jobs are lost. A durable queue (a database or message broker) would replace `DeliveryQueue`, and `DeliveryWorker` would not change.
3. **"How would you stop a user being flooded?"** Add a limit per user and channel, checked in `send` before queuing, or a digest that groups promotional notifications. It is a new rule, because nothing in the requirements limits volume.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Real email, SMS and push services | Only the three `send` methods touch the outside world. |
| A saved queue that survives a crash | See follow-up question 2. |
| Sending at a later time | Not required. It would be a `next_attempt_at` on the first delivery. |
| Alerts that ignore opt-out | A user's choice always wins. A rule such as "ALERT can't be turned off" would go into `User.wants`. |
| Duplicates after a lost answer | See follow-up question 1. |
| Limits on how many a user receives | See follow-up question 3. |
| Starting the worker threads | Infrastructure outside the design. They only need to call `process_next` in a loop. |
| Different wording per language | A template per language, chosen by the caller. |

## 13. Requirements Met
1. Multiple channels: `NotificationChannel` with `EmailChannel`, `SmsChannel`, `PushChannel`, and one `Delivery` per channel.
2. Easy to add channels: the `NotificationChannel` interface and `ChannelRegistry.register`.
3. Users choose channels and types: `User.set_channel_enabled`, `set_type_enabled` and `get_deliverable_channels`.
4. Templates with placeholders: `Template.render`.
5. Types and priorities: `NotificationType`, `Priority`, and the ordering in `DeliveryQueue`.
6. Several parties notified when an event happens: `EventPublisher`, `EventListener` and `NotificationListener`.
7. Retry failed deliveries: `RetryPolicy`, `Delivery.record_attempt` and `DeliveryWorker.process_next`.
8. Track delivery status: `Delivery.status`, `Notification.get_statuses` and `NotificationService.get_statuses`.
9. Sending without blocking the caller: `NotificationService.send` queues, and `DeliveryWorker` delivers.

## 14. Requirement Fit Check

**Setup used in every example:** the `RetryPolicy` is 3 attempts with a base delay of 30 seconds. Template `order_shipped` has the subject `Order {order_id} shipped` and the body `Hi {name}, your order {order_id} is on the way.` User `U1` (Asha) has an email address, a phone number and a device token, with everything enabled. User `U2` (Ben) has an email address and a phone number, with SMS and push switched off and PROMOTIONAL switched off. The time is 10:00.

1. **Multiple channels**
   - *Success:* `send("U1", TRANSACTIONAL, MEDIUM, "order_shipped", {name: Asha, order_id: 42}, 10:00)` returns a Notification with three deliveries (EMAIL, SMS, PUSH), all `PENDING`. The worker delivers each, and each becomes `DELIVERED`.
   - *Failure:* `send("U9", ...)` returns None because the user is unknown. Nothing is queued.
2. **Easy to add channels**
   - *Success:* a new `WhatsAppChannel` class implements `send`. With one new `ChannelType` value and `register(WHATSAPP, channel)`, any user with a WhatsApp address and the channel enabled gets a fourth delivery. No existing logic is edited.
   - *Failure:* a channel type with no registered class is skipped by `supports`, so no delivery is created for it.
3. **Users choose channels and types**
   - *Success:* for U2, `get_deliverable_channels()` returns only EMAIL, so a TRANSACTIONAL send creates one delivery.
   - *Failure:* a PROMOTIONAL send to U2 returns None because U2 turned that type off. `set_type_enabled("U9", ...)` returns False.
4. **Templates with placeholders**
   - *Success:* rendering gives the subject `Order 42 shipped` and the body `Hi Asha, your order 42 is on the way.`
   - *Failure:* values without `order_id` make `render` return None, so `send` returns None and nothing is queued.
5. **Types and priorities**
   - *Success:* the queue holds a LOW promotion that arrived at 09:58 and a HIGH alert that arrived at 09:59. `take_ready(10:00)` returns the HIGH alert first.
   - *Failure to avoid:* the HIGH alert must not wait behind the older LOW job.
6. **Interested parties notified on an event**
   - *Success:* an `EventPublisher` has a `NotificationListener` and an audit listener. `publish(Event order_shipped, recipients [U1, U2])` calls both listeners, and the notification listener sends one notification to each recipient. It returns 2.
   - *Failure:* if the audit listener fails, the notification listener still runs, and `publish` returns 1.
7. **Retry failed deliveries**
   - *Success:* the SMS channel fails at 10:00. `record_attempt` makes the delivery `RETRYING` with the next attempt at 10:00:30. At 10:00:10 `take_ready` finds nothing. At 10:00:30 the retry succeeds and the status is `DELIVERED` after 2 attempts.
   - *Failure:* if the third attempt also fails, `next_delay(3)` is None and the status is `FAILED`. It is not retried again.
8. **Track delivery status**
   - *Success:* `get_statuses(notification_id)` returns {EMAIL: DELIVERED, SMS: RETRYING, PUSH: DELIVERED} while the SMS retry is waiting.
   - *Failure:* `get_statuses("N999")` returns None.
9. **Not blocking the caller**
   - *Success:* `send` returns the Notification with three `PENDING` deliveries straight away. The real sending happens later, when worker threads call `process_next`.
   - *Failure to avoid:* a slow SMS gateway must not delay the caller of `send`.
