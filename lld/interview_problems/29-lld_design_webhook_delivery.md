# Webhook Delivery System (like Stripe, GitHub, Shopify): Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** publishing an event only creates one `Delivery` per subscribed endpoint and puts it on a queue, so the code that raised the event never waits for anyone's server. Worker threads send each delivery signed with the endpoint's secret, sort every answer into "worked", "try again later" or "give up", and record every attempt. The queue keeps each endpoint's deliveries apart and limits how many workers one endpoint may use, so a slow or broken customer can't hold up the others. An endpoint that keeps failing is switched off automatically, and anything that could not be delivered can be replayed.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. Customers should be able to register webhook endpoints (an HTTPS URL) and choose which event types each endpoint receives.
2. When an event happens in the platform, it should be delivered to every endpoint subscribed to that event type.
3. Each delivery should be sent as an HTTP POST with the event payload, without blocking the code that raised the event.
4. Every delivery should be signed so the customer can check that it came from the platform and has not been replayed.
5. A delivery should count as successful only on a 2xx response within a time limit. Time-outs, 429 and 5xx responses should be retried with growing delays, and other failures should not be retried.
6. After the last retry fails, a delivery should go to a dead-letter list from which it can be replayed.
7. Delivery should be at least once, and each event should carry a unique id so customers can ignore duplicates.
8. One customer's slow or failing endpoint should not delay deliveries to other customers.
9. An endpoint that keeps failing should be switched off automatically, and the customer should be able to switch it back on.
10. The system should record every delivery attempt with its status, response code and timing, and let customers view them.
11. Customers should be able to rotate an endpoint's signing secret without missing any deliveries.

**Types and conventions**
- Times are `int` milliseconds since a fixed start (a Unix timestamp in milliseconds), passed in by the caller or read from a `Clock`. Signature timestamps are whole seconds.
- An event's payload is a `str` of JSON. The body sent is a JSON object with `event_id`, `event_type`, `created_at` and `payload`.
- The signature header is `Webhook-Signature: t=<seconds>,v1=<hex>`, where the hex is HMAC-SHA256 of `"<seconds>.<body>"` keyed with the endpoint's secret. During a secret rotation there is one `v1=` value per valid secret.
- Customers are identified only by a `customer_id: str`. A customer sees only their own endpoints and deliveries.
- Delivery order between events is not guaranteed, because a retrying delivery falls behind newer ones.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put`.
- Sending HTTP requests is the outside world, so it sits behind the `HttpSender` interface.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| DeliveryStatus, EndpointStatus, ResponseClass (enums) | Name the fixed values | none |
| Event | Records one thing that happened | id, type, payload, time |
| SecretEntry | Holds one signing secret and until when it is valid | secret, expiry |
| Endpoint | Holds a customer's URL, event types, secrets and health, and switches itself off | status, failure count, secrets |
| HttpResponse | Records what an endpoint answered | status code, time-out flag, timing |
| HttpSender (interface) | Sends one POST with a time limit | none |
| PayloadSigner | Builds the signature header | none |
| ResponseClassifier | Sorts an answer into success, retry or give up | none |
| RetryPolicy | Says how long to wait before each retry, and when to give up | attempts, delays |
| DeliveryAttempt | Records one try | an unchangeable record |
| Delivery | Tracks one event to one endpoint, its tries and its status | status, tries, next time |
| DeliveryQueue | Hands out due deliveries fairly across endpoints, with a limit per endpoint | the waiting deliveries, work in progress per endpoint |
| WebhookService | Registers endpoints, publishes events, runs a worker step, and answers queries | endpoints, events, deliveries |
| WebhookWorkerPool | Runs several worker threads | the threads |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Customer, endpoint, URL, event type, event, delivery, POST, payload, signature, secret, response, retry, dead-letter list, attempt, status.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Endpoint, Event, Delivery, DeliveryAttempt and SecretEntry are classes. The dead-letter list is a view of deliveries in one status, not a separate store.
2. **Find fixed sets of values.** A delivery's status, an endpoint's status, and the three kinds of answer.
   - *Result:* `DeliveryStatus`, `EndpointStatus`, `ResponseClass`.
3. **Give every fact one owner.** An endpoint's health and secrets belong to the Endpoint. A delivery's tries and status belong to the Delivery. The waiting order belongs to the DeliveryQueue.
   - *Why:* two copies of one fact will eventually disagree. The dead-letter list is computed from delivery statuses, so it can't disagree with them.
4. **Mark what can be computed and do not store it.** The signature, the answer's class, the next retry time and the dead-letter list are all computed.
5. **Give every verb to the class whose data changes.** Recording a try changes the Delivery. Counting failures and switching off changes the Endpoint. Handing out work changes the queue.
6. **Find the action that touches several classes.** One worker step needs the queue, the delivery, the endpoint, the signer, the sender, the classifier and the retry rule, in order. That is `WebhookService.process_next`.
7. **Find the hint phrases for patterns.** "Delivered to every endpoint subscribed" is publish–subscribe. "Without blocking" is a queue with workers. "Switched off automatically" after repeated failures is a circuit breaker.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirements 3 and 8 need it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** An `Event` for publishing, a `bool` for every command, and lists for history and the dead-letter view.

## 4. Pattern Guide

### Used: Publish–Subscribe (fan-out)
- **Meaning:** whoever raises an event doesn't know who listens. Every endpoint that subscribed to that type gets its own copy.
- **Hint phrase:** "it should be delivered to every endpoint subscribed to that event type".
- **How it shows here:** `publish` makes one `Delivery` per subscribed, active endpoint.

### Used: a queue with a pool of workers (not a named design pattern)
- **Meaning:** callers add work and move on, and worker threads take the next due item.
- **Hint phrase:** "without blocking the code that raised the event".

### Used: Circuit Breaker (a reliability pattern, not one of the classic design patterns)
- **Meaning:** after too many failures in a row, stop calling something that is clearly broken, until someone says it is fixed.
- **Hint phrase:** "an endpoint that keeps failing should be switched off automatically".
- **How it shows here:** `Endpoint.record_failure` counts failures in a row, and at a threshold sets `DISABLED`. A success resets the count.

### The HTTP sender interface (a boundary, not a pattern)
- **Why an interface:** real network calls are the outside world. Tests use a fake sender that answers 200, 503, 400 or a time-out on demand.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| Strategy | "a rule that can be replaced" | Not needed. The retry rule is data, and only one signing scheme is asked for. |
| State | "behaves differently depending on its state" | Not used. A delivery's status only decides which change is allowed. |
| Observer | "notify interested objects in the same program" | Publish–subscribe is its cousin across systems, used above. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### DeliveryStatus
- Values: `PENDING`, `RETRYING`, `SUCCEEDED`, `DEAD`
- *Why:* `PENDING` waits for its first try. `RETRYING` failed but will try again. `SUCCEEDED` got a 2xx. `DEAD` is the dead-letter state of requirement 6, reached by running out of tries, by an answer not worth retrying, or by the endpoint being switched off.

### EndpointStatus
- Values: `ACTIVE`, `DISABLED`. *Why:* requirement 9.

### ResponseClass
- Values: `SUCCESS`, `RETRYABLE`, `PERMANENT`
- *Why:* requirement 5 treats answers three ways. Sorting them once means the retry rule never looks at status codes.

## 6. Classes (smallest dependency first)

### Event
**Why this class exists:** requirement 7 gives each event a unique id. An event is a fixed record that every delivery of it shares.
- Attributes: `event_id: str`, `event_type: str`, `payload: str`, `created_at_ms: int`.
- Constructor: `Event(event_id: str, event_type: str, payload: str, created_at_ms: int)`. *Why:* all set once. It refuses empty values.
- Methods: getters, and `to_body() -> str`. *Why `to_body`:* the exact JSON text sent, built the same way every time so the signature matches.

### SecretEntry
**Why this class exists:** requirement 11 rotates secrets. Each secret has an end time, so an old one can keep working for a while.
- Attributes: `secret: str`, `valid_until_ms: int or None`. *Why:* the key, and when it stops being used. None means "current, no end".
- Constructor: `SecretEntry(secret: str, valid_until_ms: int or None)`. *Why:* refuses a secret shorter than 16 characters.
- Methods: `get_secret() -> str`, `is_valid(now_ms: int) -> bool`.

### Endpoint
**Why this class exists:** requirements 1, 9 and 11 are all about one customer URL: what it wants, how healthy it is, and how it is signed. The endpoint owns those and guards them with a lock.
- Attributes:
  - `endpoint_id: str`, `customer_id: str`, `url: str`. *Why:* the identity, the owner and where to send.
  - `event_types: set[str]`. *Why:* what it subscribes to.
  - `secrets: list[SecretEntry]`. *Why:* the current secret, plus an old one during a rotation.
  - `status: EndpointStatus`, `consecutive_failures: int`. *Why:* whether it receives deliveries, and how many tries in a row have failed.
  - `lock: Lock`. *Why:* workers and the customer change these at the same time.
- Constructor: `Endpoint(endpoint_id: str, customer_id: str, url: str, event_types: set[str], secret: str)`. *Why:* refuses a URL that is not `https://`, or an empty set of types. It starts `ACTIVE` with 0 failures.
- Methods (each under the lock):
  - Getters, and `is_active() -> bool`.
  - `subscribes_to(event_type: str) -> bool`.
  - `set_event_types(types: set[str]) -> bool`. *Why:* False for an empty set.
  - `signing_secrets(now_ms: int) -> list[str]`. *Why:* every secret still valid, newest first.
  - `rotate_secret(new_secret: str, now_ms: int, overlap_ms: int) -> bool`. *Why:* the current secret gets an end time of `now_ms + overlap_ms`, and the new one becomes current. Expired ones are dropped. During the overlap, deliveries are signed with both, so a customer who hasn't switched yet still accepts them.
  - `record_success() -> None`. *Why:* resets the failure count.
  - `record_failure(threshold: int) -> bool`. *Why:* adds one to the count. When it reaches `threshold`, the status becomes `DISABLED`. Returns True only on the call that switched it off.
  - `enable() -> bool`. *Why:* `DISABLED` to `ACTIVE`, with the count reset.
  - `disable() -> bool`. *Why:* the customer can switch it off by hand.

### HttpResponse and HttpSender (interface)
**Why they exist:** requirement 3's POST and requirement 5's time limit are the outside world, so they sit behind one method.
- `HttpResponse(status_code: int or None, timed_out: bool, latency_ms: int, error: str or None)`. *Why:* the code (None if no answer came back), whether the time limit was hit, how long it took, and a network error message if any.
- `HttpSender` method: `post(url: str, headers: map{str: str}, body: str, timeout_ms: int) -> HttpResponse`. *Why:* it never throws. Every problem comes back inside the response.

### PayloadSigner
**Why this class exists:** requirement 4. Building the signature in one place means every delivery is signed the same way.
- Attributes: none. Constructor: `PayloadSigner()`.
- Methods: `sign(secrets: list[str], timestamp_s: int, body: str) -> str`. *Why:* returns `t=<timestamp_s>,v1=<hex>` with one `v1=` value per secret, each HMAC-SHA256 of `"<timestamp_s>.<body>"`. *Why the timestamp is inside the signed text:* a customer rejects anything older than a few minutes, so a captured request can't be replayed later.

### ResponseClassifier
**Why this class exists:** requirement 5's rules for what to retry live in one place.
- Methods: `classify(response: HttpResponse) -> ResponseClass`.
  - `SUCCESS` for a 200 to 299 code.
  - `RETRYABLE` for a time-out, no answer at all, 429, or 500 to 599.
  - `PERMANENT` for anything else, such as 400, 401, 404 or 410.

### RetryPolicy
**Why this class exists:** requirement 5's growing delays and the limit on tries belong together.
- Attributes: `max_attempts: int`, `base_delay_ms: int`, `max_delay_ms: int`, `jitter: bool`.
- Constructor: `RetryPolicy(max_attempts: int, base_delay_ms: int, max_delay_ms: int, jitter: bool)`. *Why:* refuses `max_attempts < 1` or a base delay above the maximum.
- Methods: `next_delay(attempts_made: int, random_fraction: float) -> int or None`. *Why:* None once `attempts_made >= max_attempts`. Otherwise `min(max_delay_ms, base_delay_ms * 2^(attempts_made - 1))`, and if `jitter` is on, that times `random_fraction` (a number from 0 to 1 given by the caller). *Why jitter:* without it, every delivery that failed at the same moment retries at the same moment and hits the recovering server all at once. *Why the caller passes the random number:* tests can make it fixed.

### DeliveryAttempt
**Why this class exists:** requirement 10 records every try. Each is a fixed record.
- Attributes: `attempt_number: int`, `at_ms: int`, `status_code: int or None`, `timed_out: bool`, `latency_ms: int`, `result: ResponseClass`, `error: str or None`.
- Constructor: `DeliveryAttempt(...)` with each field. Methods: getters.

### Delivery
**Why this class exists:** requirements 5, 6 and 10 are about one event going to one endpoint. A delivery owns its tries, its status and when to try next, and guards them.
- Attributes:
  - `delivery_id: str`, `event: Event`, `endpoint_id: str`, `customer_id: str`. *Why:* the identity, what is sent, where, and whose it is.
  - `status: DeliveryStatus`. *Why:* the stage.
  - `attempts: list[DeliveryAttempt]`. *Why:* the history for requirement 10.
  - `next_attempt_at_ms: int`. *Why:* the earliest time for the next try.
  - `dead_reason: str or None`. *Why:* why it ended in the dead-letter list.
  - `lock: Lock`.
- Constructor: `Delivery(delivery_id: str, event: Event, endpoint_id: str, customer_id: str, now_ms: int)`. *Why:* starts `PENDING`, due now, with no tries.
- Methods (each under the lock):
  - Getters, and `get_attempts() -> list[DeliveryAttempt]` (a copy).
  - `record_attempt(response: HttpResponse, result: ResponseClass, now_ms: int, policy: RetryPolicy, random_fraction: float) -> DeliveryStatus or None`. *Why:* None if the status is not `PENDING` or `RETRYING`. It adds a `DeliveryAttempt`, then:
    - `SUCCESS` sets `SUCCEEDED`.
    - `PERMANENT` sets `DEAD` with the reason "not retryable".
    - `RETRYABLE`: if `policy.next_delay(number of attempts, random_fraction)` is a delay, it sets `RETRYING` and `next_attempt_at_ms = now_ms + delay`. Otherwise `DEAD` with the reason "retries exhausted".
    - It returns the new status.
  - `mark_dead(reason: str) -> bool`. *Why:* `PENDING` or `RETRYING` to `DEAD`, used when the endpoint is switched off.
  - `reset_for_replay(now_ms: int) -> bool`. *Why:* `DEAD` to `PENDING`, due now. The old tries stay in the history, and the try count for the retry rule starts again from the next attempt.

### DeliveryQueue
**Why this class exists:** requirement 8. If all deliveries shared one line, a slow endpoint with thousands of deliveries would take every worker. The queue keeps one line per endpoint, goes round the endpoints in turn, and lets each endpoint use only a few workers at once.
- Attributes:
  - `lines: map{str: list[(int, str)]}` (an endpoint id to its waiting deliveries as pairs of due time and delivery id, earliest first).
  - `in_flight: map{str: int}` (an endpoint id to how many of its deliveries workers are sending now).
  - `order: list[str]`, `next_index: int`. *Why:* the endpoints in turn, and where the next search starts.
  - `max_in_flight_per_endpoint: int`. *Why:* the most workers one endpoint may use.
  - `lock: Lock`.
- Constructor: `DeliveryQueue(max_in_flight_per_endpoint: int)`. *Why:* refuses a value below 1.
- Methods (each under the lock):
  - `add(endpoint_id: str, delivery_id: str, due_ms: int) -> None`. *Why:* puts it in that endpoint's line in time order, adding the endpoint to `order` if new.
  - `take(now_ms: int) -> (str, str) or None`. *Why:* starting from `next_index`, it finds the first endpoint whose in-flight count is below the limit and whose earliest delivery is due. It removes that delivery, adds 1 to that endpoint's in-flight count, moves `next_index` past it, and returns the endpoint id and delivery id. None if nothing is ready. *Why round-robin:* every endpoint gets a turn, so a long line can't starve the others.
  - `done(endpoint_id: str) -> None`. *Why:* subtracts 1 from that endpoint's in-flight count.
  - `size() -> int`.

### WebhookService (the manager and only entry point)
**Why this class exists:** registering, publishing, delivering and replaying each cross several classes in a fixed order, and endpoints, events and deliveries need one place to be found.
- Attributes:
  - `endpoints: map{str: Endpoint}` (an endpoint id to the Endpoint), `events: map{str: Event}` (an event id to the Event), `deliveries: map{str: Delivery}` (a delivery id to the Delivery). *Why:* the one record of each. Thread-safe maps.
  - `queue: DeliveryQueue`, `sender: HttpSender`, `signer: PayloadSigner`, `classifier: ResponseClassifier`, `retry: RetryPolicy`. *Why:* passed in.
  - `timeout_ms: int`, `failure_threshold: int`, `secret_overlap_ms: int`. *Why:* the time limit for each POST, the failures in a row before switching off, and how long an old secret keeps working. All data.
  - `counter: AtomicInt`. *Why:* makes ids unique.
- Constructor: `WebhookService(queue, sender, signer, classifier, retry, timeout_ms, failure_threshold, secret_overlap_ms)`. *Why:* refuses non-positive numbers.
- Methods (outer ones take ids and check the customer):
  - `register_endpoint(customer_id: str, url: str, event_types: set[str], secret: str) -> Endpoint or None`. *Why:* requirement 1.
  - `update_event_types(customer_id: str, endpoint_id: str, event_types: set[str]) -> bool`.
  - `rotate_secret(customer_id: str, endpoint_id: str, new_secret: str, now_ms: int) -> bool`. *Why:* requirement 11.
  - `enable_endpoint(customer_id: str, endpoint_id: str) -> bool`, `disable_endpoint(customer_id: str, endpoint_id: str) -> bool`. *Why:* requirement 9.
  - `publish(event_type: str, payload: str, now_ms: int) -> Event`. *Why:* requirements 2, 3 and 7. It returns at once.
  - `process_next(now_ms: int, random_fraction: float) -> bool`. *Why:* one worker step. True if it sent something or dealt with an entry.
  - `get_attempts(customer_id: str, delivery_id: str) -> list[DeliveryAttempt] or None`. *Why:* requirement 10. None if the delivery is not that customer's.
  - `list_deliveries(customer_id: str, endpoint_id: str, status: DeliveryStatus or None) -> list[Delivery] or None`. *Why:* requirement 10, newest first.
  - `list_dead_letters(customer_id: str) -> list[Delivery]`. *Why:* requirement 6. Computed as that customer's deliveries in `DEAD`.
  - `replay(customer_id: str, delivery_id: str, now_ms: int) -> bool`. *Why:* requirement 6. Refused while the endpoint is disabled.
  - `replay_all_dead(customer_id: str, endpoint_id: str, now_ms: int) -> int`. *Why:* after re-enabling an endpoint, send everything it missed. Returns how many were queued.
- Private helper: `owned_endpoint(customer_id: str, endpoint_id: str) -> Endpoint or None`. *Why:* one lookup that also checks the owner.

### WebhookWorkerPool
**Why this class exists:** requirement 3's background sending, with several threads.
- Attributes: `size: int`, `service: WebhookService`, `clock: Clock`, `idle_wait_ms: int`, `threads: list[Thread]`, `running: bool`.
- Methods: `start() -> bool` starts `size` threads that repeat `service.process_next(clock.now_millis(), a fresh random number)` and wait `idle_wait_ms` when it returns False. `stop() -> None` lets each thread finish its current delivery.
- `Clock` is the usual small interface (`now_millis() -> int`) with `SystemClock` reading the real time.

## 7. Relationships
- WebhookService **has** Endpoint, Event, Delivery (created inside `register_endpoint` and `publish`).
- Endpoint **has** SecretEntry. Delivery **has** DeliveryAttempt.
- Delivery **uses** Event (shared by every delivery of that event).
- WebhookService **uses** DeliveryQueue, HttpSender, PayloadSigner, ResponseClassifier, RetryPolicy (passed in).
- WebhookWorkerPool **uses** WebhookService, Clock.
- Enums are always **uses**: Endpoint to EndpointStatus, Delivery to DeliveryStatus, DeliveryAttempt to ResponseClass.
- `HttpResponse` appears only as a method input and output, so it has no line.

## 8. Interaction

### Layers
1. **Entry points:** WebhookService (for the platform and customers) and WebhookWorkerPool (which drives sending).
2. **Fair ordering:** DeliveryQueue.
3. **Classes that guard themselves:** Endpoint and Delivery.
4. **Rules and outside world:** PayloadSigner, ResponseClassifier, RetryPolicy, HttpSender.
5. **Plain data:** Event, SecretEntry, DeliveryAttempt, HttpResponse.

### Call chains
- **Publish**
  1. `Platform code -> WebhookService.publish(event_type, payload, now)`.
  2. `WebhookService -> Endpoint.is_active()` and `subscribes_to(event_type)` for each endpoint.
  3. `WebhookService -> Delivery(...)` for each match, then `DeliveryQueue.add(...)`. Return the event at once.
- **Send one delivery (worker step)**
  1. `Worker -> WebhookService.process_next(now, random)`.
  2. `WebhookService -> DeliveryQueue.take(now)`: a fair, due delivery from an endpoint that is under its limit.
  3. `WebhookService -> Endpoint.signing_secrets(now)`, then `PayloadSigner.sign(...)`.
  4. `WebhookService -> HttpSender.post(url, headers, body, timeout)`: with no lock held.
  5. `WebhookService -> ResponseClassifier.classify(response)`, then `Delivery.record_attempt(...)`.
  6. `WebhookService -> Endpoint.record_success()` or `record_failure(threshold)`.
  7. If `RETRYING`: `DeliveryQueue.add(endpoint, delivery, next time)`. Always: `DeliveryQueue.done(endpoint)`.
- **Replay**
  1. `Customer -> WebhookService.replay(customer, delivery, now)`, then `Delivery.reset_for_replay(now)`, then `DeliveryQueue.add(...)`.

### Who may call whom
- Platform code calls `publish` only. Customers call the endpoint, query and replay methods. Worker threads call `process_next` only.
- `WebhookService` calls everything it was given, plus `Endpoint` and `Delivery`.
- `Endpoint`, `Delivery` and the rule classes call no other class. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Register an endpoint
1. `register_endpoint(customer_id, url, event_types, secret)`. Build the `Endpoint` (refused for a non-HTTPS URL, no types, or a short secret). Store and return it.

### Publish (requirements 2, 3, 7)
1. `publish(event_type, payload, now_ms)`. Build the `Event` with a new unique id, and store it.
2. For each endpoint that is active and subscribes to `event_type`: build a `Delivery`, store it, `queue.add(endpoint_id, delivery_id, now_ms)`.
3. Return the event. *Why no sending here:* the caller must never wait for a customer's server.
- **Failure:** an endpoint that is disabled at publish time gets no delivery. An event with no subscribers is stored and simply goes nowhere.

### Send one delivery
1. `process_next(now_ms, random_fraction)`. `item = queue.take(now_ms)`. If None, return False.
2. Look up the delivery and the endpoint. If the endpoint is now disabled: `delivery.mark_dead("endpoint disabled")`, `queue.done(endpoint_id)`, return True. *Why:* nothing is sent to a switched-off endpoint, but nothing is lost either, because it can be replayed.
3. `body = event.to_body()`. `timestamp_s = now_ms // 1000`. Headers: `Webhook-Signature` from `signer.sign(endpoint.signing_secrets(now_ms), timestamp_s, body)`, plus `Webhook-Id` (the event id), `Webhook-Delivery-Id` and `Content-Type: application/json`.
4. `response = sender.post(url, headers, body, timeout_ms)`.
5. `result = classifier.classify(response)`. `status = delivery.record_attempt(response, result, now_ms, retry, random_fraction)`.
6. If `result` is `SUCCESS`, `endpoint.record_success()`. Otherwise `endpoint.record_failure(failure_threshold)`.
7. If `status` is `RETRYING`, `queue.add(endpoint_id, delivery_id, delivery.next_attempt_at_ms)`.
8. `queue.done(endpoint_id)`. Return True.
- **Failure and undo:** nothing to undo. A failure only adds an attempt, may schedule a retry, and may move the delivery to the dead-letter list.

### Replay (requirement 6)
1. `replay(customer_id, delivery_id, now_ms)`. The delivery must be the customer's, `DEAD`, and its endpoint active.
2. `delivery.reset_for_replay(now_ms)`, then `queue.add(...)`.
- **Failure:** returns False for another customer's delivery, a delivery that is not dead, or a disabled endpoint.

### Switch off and back on (requirement 9)
1. Each failed try adds 1 to the endpoint's count, and a success resets it. At the threshold the endpoint becomes `DISABLED`.
2. Deliveries already queued for it become `DEAD` when a worker reaches them, and new events skip it.
3. `enable_endpoint` sets it `ACTIVE` with the count at 0. `replay_all_dead` then queues everything it missed.

### Rotate a secret (requirement 11)
1. `rotate_secret(customer_id, endpoint_id, new_secret, now_ms)`. The old secret stays valid for `secret_overlap_ms`.
2. During that time, every delivery carries two `v1=` signatures, so the customer's code accepts it whether it checks with the old secret or the new one. After that, only the new one is used.

### Concurrency (requirements 3 and 8)
- **Queue:** the lock sits inside `DeliveryQueue`, around every method. Each delivery is handed to exactly one worker, and the in-flight limit is checked and raised in one step.
- **Endpoint:** the lock sits inside `Endpoint`, around its types, secrets, status and failure count.
- **Delivery:** the lock sits inside `Delivery`, around `record_attempt`, `mark_dead` and `reset_for_replay`, so a replay can't overlap a worker's update.
- **Maps and ids:** thread-safe maps and an atomic counter.
- **No lock during the POST:** the HTTP call happens with no lock held, so a slow endpoint holds only its own worker.
- **Isolation:** at most `max_in_flight_per_endpoint` workers can be stuck on one endpoint. The rest keep serving other customers.
- **Lock order:** no method ever holds two locks at once, so no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| Different retry timing | A different `RetryPolicy` | No. It is data. |
| A different time limit, failure threshold or overlap | Different numbers for the service | No. It is data. |
| A real HTTP client, or one with a proxy | A new class implementing `HttpSender` | No. |
| More workers, or more per endpoint | A different pool size or queue limit | No. It is data. |
| A second signing scheme | Make `PayloadSigner` an interface at that point | One change, made only when a second scheme is needed. |

## 11. Interview Notes

### Key decisions
1. **Publishing only queues.**
   - *Why:* the platform's own code never waits for a customer's server, which is requirement 3.
   - *Rejected:* sending inside `publish`, which would make a payment wait on a stranger's slow server.
2. **One line per endpoint, round-robin, with a limit per endpoint.**
   - *Why:* one broken customer can tie up at most a few workers, which is requirement 8.
   - *Rejected:* one shared first-in first-out queue, where a backlog for one endpoint delays everyone.
3. **Answers are sorted once into success, retry or give up.**
   - *Why:* retrying a 400 or 404 wastes effort and never works, while 429, 5xx and time-outs often recover.
   - *Rejected:* retrying every failure.
4. **At least once, with an event id, instead of exactly once.**
   - *Why:* if the customer's server processed a request but the answer was lost, the platform can't know, so it must resend. The event id lets the customer drop the duplicate.
   - *Rejected:* promising exactly once, which can't be guaranteed across a network.
5. **The dead-letter list is a status, not a separate store.**
   - *Why:* one fact in one place, and a replay is just a status change and a new queue entry.
   - *Rejected:* copying failed deliveries into another table.

### Edge cases handled
- A non-HTTPS URL, no event types, or a short secret is refused.
- An event with no subscribers goes nowhere.
- A 400 goes straight to the dead-letter list without retrying.
- A time-out or 503 is retried with a doubling delay (with jitter), up to the limit.
- A response lost after the customer processed it is sent again with the same event id.
- An endpoint switched off while deliveries are queued moves them to the dead-letter list, and they can be replayed later.
- A replay to a disabled endpoint is refused.
- During a secret rotation, both secrets sign every delivery.
- A customer can't see or replay another customer's deliveries.

### Likely follow-up questions
1. **"What if the process crashes with deliveries in the queue?"** Store deliveries in a database and write them in the same transaction as the business change that raised the event (the outbox pattern). Workers then read due deliveries from the database or a durable queue.
2. **"Can you guarantee events arrive in order?"** Only by sending one delivery at a time per endpoint and making later ones wait for a retrying one, which lets one bad event block all the rest. Most platforms don't promise order, and give `created_at` so customers can sort.
3. **"How does the customer verify a request?"** Take the timestamp from the header, refuse it if it is more than five minutes old, compute HMAC-SHA256 of `"<timestamp>.<raw body>"` with their secret, and compare with each `v1` value using a constant-time comparison.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Surviving a crash | See follow-up question 1. |
| Ordered delivery | Not promised. See follow-up question 2. |
| Removing old attempts after a retention period | Not required. A sweep would delete attempts older than the limit. |
| Limits on how many requests per second an endpoint receives | The in-flight limit caps concurrency. A per-second limit would be a rate limiter checked in `take`. |
| Customers choosing payload versions | Not required. It would be a version on the endpoint used by `to_body`. |
| Checking that the URL belongs to the customer | Not required. A challenge request at registration would do it. |

## 13. Requirements Met
1. Register endpoints with event types: `WebhookService.register_endpoint` and `update_event_types`, and `Endpoint`.
2. Deliver to every subscriber: the fan-out in `WebhookService.publish`.
3. POST without blocking: `publish` only queues, and `process_next` sends through `HttpSender`.
4. Signed against forgery and replay: `PayloadSigner.sign` with a timestamp inside the signed text.
5. Success on 2xx, retry only what is worth retrying: `ResponseClassifier`, `RetryPolicy` and `Delivery.record_attempt`.
6. Dead-letter list with replay: `DeliveryStatus.DEAD`, `list_dead_letters`, `replay` and `replay_all_dead`.
7. At least once with a unique event id: retries until success, and `Event.event_id` in the body and the `Webhook-Id` header.
8. One customer can't delay others: `DeliveryQueue` with one line per endpoint, round-robin and `max_in_flight_per_endpoint`.
9. Automatic switch-off and switch-on: `Endpoint.record_failure`, `enable` and `EndpointStatus`.
10. Every attempt recorded and viewable: `DeliveryAttempt`, `get_attempts` and `list_deliveries`.
11. Secret rotation without misses: `Endpoint.rotate_secret`, `signing_secrets` and multiple `v1` values.

## 14. Requirement Fit Check

**Setup used in every example:** endpoint `E1` for customer `C1` at `https://c1.example/hooks` with types {`payment.succeeded`}. Endpoint `E2` for customer `C2` with types {`payment.succeeded`, `order.shipped`}. The retry rule allows 5 attempts with a base delay of 1000 ms and a maximum of 60000 ms, with jitter off for these examples. The POST time limit is 5000 ms. An endpoint switches off after 10 failures in a row. Each endpoint may use 2 workers at once, and the pool has 4. The secret overlap is 24 hours. A fake sender answers as each example says.

1. **Register endpoints with event types**
   - *Success:* `register_endpoint("C1", "https://c1.example/hooks", {"payment.succeeded"}, secret)` returns E1, `ACTIVE`.
   - *Failure:* `http://c1.example/hooks` is refused. An empty set of types is refused.
2. **Deliver to every subscriber**
   - *Success:* `publish("payment.succeeded", ...)` creates one delivery for E1 and one for E2. `publish("order.shipped", ...)` creates one, for E2 only.
   - *Failure:* `publish("refund.created", ...)` creates no deliveries, because nobody subscribes.
3. **POST without blocking**
   - *Success:* `publish` returns the event at once, with both deliveries `PENDING`. The POSTs happen later, in worker threads.
   - *Failure to avoid:* a slow E1 must never slow down `publish`.
4. **Signed against forgery and replay**
   - *Success:* the request carries `Webhook-Signature: t=1731060000,v1=<hex>`. C1 computes HMAC-SHA256 of `"1731060000.<body>"` with its secret, and it matches.
   - *Failure:* a body changed in transit gives a different HMAC, so C1 rejects it. A copy of a valid request replayed ten minutes later is rejected because its timestamp is too old.
5. **Success on 2xx, retry only what is worth retrying**
   - *Success:* E1 answers 503 at time 0, so the delivery is `RETRYING` for 1000. It answers 503 again, so it waits until 1000 + 2000 = 3000. Then it answers 200, so the delivery is `SUCCEEDED` after 3 attempts. A time-out is retried the same way.
   - *Failure:* E1 answers 400, so the delivery goes straight to `DEAD` with "not retryable", after 1 attempt.
6. **Dead-letter list with replay**
   - *Success:* E1 answers 503 five times, so the delivery ends `DEAD` with "retries exhausted". `list_dead_letters("C1")` shows it. `replay("C1", that delivery, now)` makes it `PENDING`, and the next try succeeds.
   - *Failure:* `replay("C2", that delivery, now)` returns False because it is C1's.
7. **At least once with a unique event id**
   - *Success:* E2 processes a delivery, but its answer is lost and the request times out. The platform retries, and C2 receives the same event twice with the same `Webhook-Id`, so it ignores the second.
   - *Failure to avoid:* a delivery is never marked `SUCCEEDED` without a 2xx answer.
8. **One customer can't delay others**
   - *Success:* E1 hangs until every request times out, and 100 deliveries for E1 are queued. At most 2 workers are ever waiting on E1, and the other 2 keep sending E2's deliveries as soon as they are due.
   - *Failure to avoid:* E1's backlog must never take all 4 workers.
9. **Automatic switch-off and switch-on**
   - *Success:* after 10 failed tries in a row across E1's deliveries, E1 becomes `DISABLED`. Queued deliveries for E1 become `DEAD` with "endpoint disabled", and new events skip E1. `enable_endpoint("C1", "E1")` makes it `ACTIVE`, and `replay_all_dead("C1", "E1", now)` queues everything it missed.
   - *Failure:* `replay` on one of those deliveries while E1 is still disabled returns False.
10. **Every attempt recorded and viewable**
    - *Success:* `get_attempts("C1", delivery)` lists 503 (12 ms), 503 (15 ms), 200 (40 ms), each with its time and result.
    - *Failure:* `get_attempts("C2", that delivery)` returns None.
11. **Secret rotation without misses**
    - *Success:* at time T, `rotate_secret("C1", "E1", new_secret, T)`. For the next 24 hours every delivery to E1 has two `v1=` values, one from each secret, so C1 accepts it while switching. After 24 hours only the new secret signs.
    - *Failure to avoid:* no delivery in the overlap is signed with only the new secret, which C1 might not have deployed yet.
