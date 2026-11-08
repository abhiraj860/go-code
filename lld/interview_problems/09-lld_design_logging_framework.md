# Logging Framework (Log4j): Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a log call becomes an entry, the logger throws away entries below its minimum level, and the rest travel down a chain of handlers until the handler for that entry's level writes it to its destinations. Each destination owns its own format and its own lock, so formats and destinations can change without touching the chain, and many threads can log at once without garbling a line.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The system should log messages at different severity levels.
2. A logger should have a minimum level and ignore messages below it.
3. The system should write logs to multiple destinations such as console, file and database at the same time.
4. It should be easy to add new destinations.
5. Each log entry should include a timestamp, a level and a message, and the format should be replaceable.
6. A message should pass along a chain of handlers where each handler deals with its own level.
7. The same logger configuration should be accessible from everywhere in the application.
8. The logger should be safe to use from multiple threads.

**Types and conventions**
- Times are `datetime`. The logger gets them from a `Clock`, so tests can fix the time.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal or failure with nothing else disturbed.
- Logging must never crash the program. A failing destination returns False and the logger carries on.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock.
- Actual input and output (printing, writing a file, writing a database row) happens only inside the three destination classes. Everything else is plain logic.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| LogLevel (enum) | Names a severity and gives it a number to compare | the severity numbers |
| LogEntry | Records one log event | timestamp, level, message |
| LogFormatter (interface) | Turns an entry into text | none |
| PlainFormatter | Makes a one-line readable text | none |
| JsonFormatter | Makes a JSON text | none |
| LogDestination (interface) | Writes one entry somewhere | none |
| ConsoleDestination | Writes to the console | its formatter |
| FileDestination | Writes to a file | file path, formatter |
| DatabaseDestination | Writes to a database table | table name, formatter |
| LogHandler | Is one link in the chain, dealing with one level | its level, its destinations |
| Clock (interface) | Says what time it is | none |
| SystemClock | Reads the real time | none |
| Logger | Filters by minimum level, stamps the entry and starts the chain | minimum level |
| LogManager | Is the one shared place everyone gets the configured logger from | the logger |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Message, severity level, logger, minimum level, destination, console, file, database, entry, timestamp, format, handler, chain, configuration.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Logger, LogEntry, LogHandler and the destinations are classes. Message and timestamp are attributes of the entry.
2. **Find fixed sets of values.** The severity levels.
   - *Why:* a fixed set is an enum, so a typo like "eror" cannot exist.
   - *Result:* `LogLevel`.
3. **Give every fact one owner.** The minimum level belongs to the Logger. Each handler's level and destinations belong to that handler. A destination's format belongs to that destination.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** Whether an entry is below the minimum is computed by comparing severity numbers. The formatted text is produced on demand and not stored on the entry.
5. **Give every verb to the class whose data changes.** Writing changes a destination's file or console, so the destination owns `write`. Passing along changes nothing, so the handler owns `handle`.
6. **Find the action that touches several classes.** One log call needs the clock, the level filter and the chain. That order lives in `Logger`.
7. **Find the hint phrases for patterns.** "A chain of handlers where each deals with its own level" is Chain of Responsibility. "The format should be replaceable" is Strategy. "The same configuration accessible from everywhere" is Singleton.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** Requirement 8 names it, so decide exactly where the locks sit (section 8).
10. **Fix the outputs.** A `bool` from every write and handle call, so a failure never throws into the program.

## 4. Pattern Guide

### Used: Chain of Responsibility
- **Meaning:** a request is passed along a line of handlers, and each handler either deals with it or passes it on.
- **Hint phrase:** "a message should pass along a chain of handlers where each handler deals with its own level".
- **Without it:** one big method would have an `if` for each level, and routing a level somewhere new would mean editing it.
- **When NOT to use it:** when one place can decide everything with a single lookup.

### Used: Strategy
- **Meaning:** a way of doing something is a small object behind a shared interface, and the user of it just uses whichever it is given.
- **Hint phrase:** "the format should be replaceable". The destinations ("it should be easy to add new destinations") follow the same shape.
- **Without it:** each destination would have an `if` on the format, and a new format would mean editing all of them.
- **When NOT to use it:** when there is one fixed way and nobody asked to change it.

### Used: Singleton
- **Meaning:** exactly one shared instance that everyone gets through a single access point.
- **Hint phrase:** "the same logger configuration should be accessible from everywhere in the application".
- **How it is limited here:** only `LogManager` is a singleton. `Logger` and everything else are ordinary classes, so tests can build their own.
- **When NOT to use it:** when sharing is not a requirement. A passed-in object is easier to test.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. Nothing changes behavior with a state. |
| Observer | "notify several interested parties when an event happens" | Not used. A handler simply writes to a fixed list of destinations, which is simpler. |
| Factory | "create different kinds of objects by type" | Not used. The setup builds the exact objects it wants. |
| Facade | "a single simple entry point" | Not added as a separate class. `Logger` is already the single entry point for logging. |

## 5. Enums

### LogLevel
- Values and severity numbers: `DEBUG` 10, `INFO` 20, `WARN` 30, `ERROR` 40, `FATAL` 50.
- *Why:* requirement 1 names levels of severity. The numbers let the logger compare levels without a chain of `if`s.
- *Why numbers are fixed in the enum:* the order of severity is one fact, so it lives in one place.
- Method: `get_severity() -> int`. *Why:* a read-only view.
- *Extensibility:* a new level (TRACE) is one new value here, plus one more handler when the chain is built.

## 6. Classes (smallest dependency first)

### LogEntry
**Why this class exists:** requirement 5 says each entry includes a timestamp, a level and a message. A fixed record means no destination can alter what another one writes.
- Attributes: `timestamp: datetime`, `level: LogLevel`, `message: str`. *Why each:* exactly the three things requirement 5 names.
- Constructor: `LogEntry(timestamp: datetime, level: LogLevel, message: str)`. *Why:* all fields are set once. It refuses a None message. An empty message is allowed.
- Methods: `get_timestamp() -> datetime`, `get_level() -> LogLevel`, `get_message() -> str`. *Why:* read-only views. There are no setters.
- *Extensibility:* an extra field (thread name, source) is added here and old formatters keep working.

### LogFormatter (interface)
**Why it is an interface:** requirement 5 says the format should be replaceable, so destinations depend on one method and not on a specific format.
- Methods: `format(entry: LogEntry) -> str`. *Why it returns text:* every destination ends up writing text.
- *Extensibility:* a new format is a new class implementing this method.

### PlainFormatter (implements LogFormatter)
**Why this class exists:** the first format, readable by people.
- Attributes: none. Constructor: `PlainFormatter()`.
- Methods: `format(entry: LogEntry) -> str`. *Why:* `timestamp [LEVEL] message` on one line, for example `2026-11-08 10:00:00 [ERROR] db down`.
- *Extensibility:* none needed.

### JsonFormatter (implements LogFormatter)
**Why this class exists:** the second format, for machines. It shows the interface really holds more than one format.
- Attributes: none. Constructor: `JsonFormatter()`.
- Methods: `format(entry: LogEntry) -> str`. *Why:* one JSON object with the keys `timestamp`, `level` and `message`, with the text properly escaped.
- *Extensibility:* none needed.

### LogDestination (interface)
**Why it is an interface:** requirement 4 says new destinations must be easy to add, so handlers depend on one method and not on any specific place.
- Methods: `write(entry: LogEntry) -> bool`. *Why a `bool`:* False means the write failed, and the caller carries on instead of crashing the program.
- *Extensibility:* a new destination (a message queue, a remote service) is a new class implementing this method.

### ConsoleDestination (implements LogDestination)
**Why this class exists:** the first destination.
- Attributes: `formatter: LogFormatter`, `lock: Lock`. *Why:* how to turn an entry into text, and a lock so two threads can't mix their lines.
- Constructor: `ConsoleDestination(formatter: LogFormatter)`. *Why:* a destination is useless without a format, so it is passed in.
- Methods: `write(entry: LogEntry) -> bool`. *Why:* under the lock, format the entry and print the line. False if printing fails.
- *Extensibility:* the format is swapped by passing a different formatter.

### FileDestination (implements LogDestination)
**Why this class exists:** the second destination, and the one where interleaved lines would hurt most.
- Attributes: `file_path: str`, `formatter: LogFormatter`, `lock: Lock`. *Why:* the file to append to, the format, and a lock so two threads can't mix their lines.
- Constructor: `FileDestination(file_path: str, formatter: LogFormatter)`. *Why:* both are set once. It refuses an empty path.
- Methods: `write(entry: LogEntry) -> bool`. *Why:* under the lock, format the entry and append one line. False if the file can't be written.
- *Extensibility:* file rotation would be added here and nothing else changes.

### DatabaseDestination (implements LogDestination)
**Why this class exists:** the third destination named in requirement 3.
- Attributes: `table_name: str`, `formatter: LogFormatter`, `lock: Lock`. *Why:* where to insert, the format, and a lock so rows are written one at a time.
- Constructor: `DatabaseDestination(table_name: str, formatter: LogFormatter)`. *Why:* both are set once. It refuses an empty table name.
- Methods: `write(entry: LogEntry) -> bool`. *Why:* under the lock, insert one row holding the formatted text. False if the insert fails.
- *Extensibility:* none needed.

### LogHandler
**Why this class exists:** requirement 6 asks for a chain where each handler deals with its own level. One class with a level as data means a new level needs no new class.
- Attributes:
  - `level: LogLevel`. *Why:* the one level this handler deals with.
  - `destinations: list[LogDestination]`. *Why:* where entries of this level are written. Several destinations make requirement 3's "at the same time" a plain list.
  - `next_handler: LogHandler or None`. *Why:* the next link. None for the last one.
- Constructor: `LogHandler(level: LogLevel, destinations: list[LogDestination], next_handler: LogHandler or None)`. *Why:* a chain is built from its last link back to its first, so the next link already exists when each one is made. The handler keeps its own copy of the destination list, and it refuses an empty list. A chain is never changed once built, so it needs no lock.
- Methods:
  - `handle(entry: LogEntry) -> bool`. *Why:* if the entry's level equals this handler's level, write it to every destination and return True. A failed destination doesn't stop the others. Otherwise pass the entry to `next_handler` and return its answer. If there is no next handler, return False, which means no handler dealt with the level.
- *Extensibility:* routing a level somewhere new is a different destination list passed in when the chain is built.

### Clock (interface)
**Why it is an interface:** the timestamp is a fact about the entry, and tests need to fix it, so the logger asks a clock instead of reading the real time directly.
- Methods: `now() -> datetime`.
- *Extensibility:* a fixed clock for tests is a new class implementing `now`.

### SystemClock (implements Clock)
**Why this class exists:** the real clock used in production.
- Attributes: none. Constructor: `SystemClock()`.
- Methods: `now() -> datetime`. *Why:* returns the current time.
- *Extensibility:* none needed.

### Logger
**Why this class exists:** requirements 1 and 2 need one place that decides whether an entry is worth keeping and stamps it before it starts down the chain.
- Attributes:
  - `min_level: LogLevel`. *Why:* the lowest level kept. It is the one fact about configuration that can change while the program runs.
  - `chain: LogHandler`. *Why:* the first handler in the chain.
  - `clock: Clock`. *Why:* where timestamps come from.
  - `lock: Lock`. *Why:* guards `min_level`, which is read by every thread and changed by some.
- Constructor: `Logger(min_level: LogLevel, chain: LogHandler, clock: Clock)`. *Why:* all three are passed in, so a test can use a fixed clock and a handler that records entries.
- Methods:
  - `log(level: LogLevel, message: str) -> bool`. *Why public:* it is the only thing callers use, and it is safe at any time. It returns False for a message below the minimum, a None message, or a level no handler deals with. It returns True when a handler dealt with the entry.
  - `set_min_level(level: LogLevel) -> None`. *Why:* a named method instead of a raw setter. It cannot fail, because the level is already valid.
  - `get_min_level() -> LogLevel`. *Why:* read-only view.
  - `create_default() -> Logger`. *Why:* a ready-made logger for when no one has configured one: minimum INFO, a chain with one handler per level, each writing to a console with the plain format, and the system clock. So `LogManager` never has to hand out nothing.
- *Why there are no `info()` or `error()` shortcuts:* they would only call `log`. One method in one place is simpler, and shortcuts can be added without changing anything.
- *Extensibility:* the chain, the destinations, the formats and the clock are all passed in.

### LogManager (the only singleton)
**Why this class exists:** requirement 7 wants the same configured logger available from everywhere. A single shared access point does that without passing a logger through every function.
- Attributes:
  - `instance: LogManager or None` (shared by the class). *Why:* the one shared instance, created the first time it is asked for.
  - `logger: Logger`. *Why:* the shared logger. It starts as `Logger.create_default()`.
  - `configured: bool`. *Why:* whether someone has set the logger on purpose. It stops a second setup from silently replacing the first.
  - `lock: Lock`. *Why:* guards the creation of the one instance and the setup.
- Constructor: private. *Why:* only `get_instance` may create it, so there can only be one.
- Methods:
  - `get_instance() -> LogManager`. *Why:* creates the one instance under the lock on first use, so two threads asking at the same moment still get the same object.
  - `initialize(logger: Logger) -> bool`. *Why:* sets the shared logger once. It returns False if it was already set up, and the first setup stays.
  - `get_logger() -> Logger`. *Why:* the shared logger. It is never None, because it starts as the default.
- *Why only this class is a singleton:* the `Logger` itself is an ordinary class, so tests can build and pass their own.
- *Extensibility:* none needed.

## 7. Relationships
- LogManager **has** Logger (the default is created inside, and the set one is kept).
- Logger **has** LogHandler (the head of the chain; the handlers live and die with the logger's configuration).
- LogHandler **uses** LogDestination, LogHandler (the next link). Destinations are passed in.
- Logger **uses** Clock (passed in).
- ConsoleDestination, FileDestination, DatabaseDestination **use** LogFormatter (passed in).
- ConsoleDestination, FileDestination, DatabaseDestination **implement** LogDestination.
- PlainFormatter, JsonFormatter **implement** LogFormatter. SystemClock **implements** Clock.
- Enums are always **uses**: LogEntry to LogLevel, LogHandler to LogLevel, Logger to LogLevel.
- `LogEntry` appears only as a method input, so no attribute holds it and it has no line.

## 8. Interaction

### Layers
1. **Shared access point:** LogManager.
2. **Entry point:** Logger.
3. **Routing:** the chain of LogHandler objects.
4. **Output:** the three destinations, each with a formatter.
5. **Plain data:** LogEntry, LogLevel.

### Call chains
- **Get the shared logger**
  1. `Any code -> LogManager.get_instance().get_logger()`: the same logger, from anywhere.
- **Log a message**
  1. `Any code -> Logger.log(level, message)`.
  2. `Logger -> Logger.min_level` (under its lock): the cheap filter runs first, so a discarded message costs almost nothing.
  3. `Logger -> Clock.now()`: the timestamp is taken only for messages that will be kept.
  4. `Logger -> LogHandler.handle(entry)`: starts the chain.
  5. `LogHandler -> next LogHandler.handle(entry)`: passed along until the handler for this level.
  6. `LogHandler -> LogDestination.write(entry)` for each destination of that handler.
  7. `LogDestination -> LogFormatter.format(entry)`: each destination formats in its own way.
- **Configure once**
  1. `Startup -> LogManager.get_instance().initialize(logger)`.

### Who may call whom
- Application code calls `LogManager.get_instance()` and `Logger.log` only.
- `Logger` calls `Clock` and the first `LogHandler`.
- `LogHandler` calls `LogDestination` and the next `LogHandler`.
- A destination calls its own `LogFormatter` only.
- A handler must never call back to the logger, and a destination must never call a handler. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Log a message
1. `log(level: LogLevel, message: str)`. If `message` is None, return False.
2. Read `min_level` under the lock. If `level.get_severity() < min_level.get_severity()`, return False. *Why first:* a discarded message must cost almost nothing.
3. `timestamp: datetime = clock.now()`. Build `LogEntry(timestamp, level, message)`.
4. `handled: bool = chain.handle(entry)`. Return it.
- **Failure and undo:** there is nothing to undo. A message that is filtered out, has no handler, or hits a failing destination changes no program state. A failing destination returns False and the others still write.

### Pass along the chain
1. In `handle(entry)`: if `entry.level == level`, call `write(entry)` on every destination, and return True.
2. Otherwise, if `next_handler` exists, return `next_handler.handle(entry)`.
3. Otherwise return False.
- **Failure:** a level with no handler is dropped and `log` returns False. A destination that fails does not stop the loop.

### Set up the shared logger
1. `get_instance()` returns the one `LogManager`, creating it under the lock if needed.
2. `initialize(logger: Logger)`: under the lock, if `configured` is False, set `logger` and `configured` and return True. Otherwise return False.
- **Failure:** a second `initialize` returns False and changes nothing. Before any setup, `get_logger()` still returns the default logger, so a log call never fails.

### Change the minimum level
1. `set_min_level(level)` under the logger's lock.
- **Failure:** none. The next `log` call uses the new level.

### Concurrency (requirement 8)
- **Destinations:** the lock sits inside each destination, around the whole of `write`. One line is formatted and written as one step, so lines from different threads are never mixed. Two different destinations don't wait for each other.
- **Logger:** the lock sits inside `Logger`, around `min_level` only. It is released before the chain runs.
- **Chain and formatters:** they have no lock, because a chain is never changed once built and the formatters hold no data.
- **LogManager:** the lock sits inside `LogManager`, around creating the instance and the one-time setup.
- **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
- **Why not one lock around the whole `log`:** every thread would wait for the slowest destination, even when it only needed the console.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new destination (a message queue, a remote service) | A new class implementing `LogDestination`, added to a handler's list when the chain is built | No. |
| A new format (XML, a custom layout) | A new class implementing `LogFormatter`, passed to a destination | No. |
| A new level (TRACE) | One value in `LogLevel`, and one more `LogHandler` in the chain | No. |
| Sending ERROR to a different set of places | A different destination list for the ERROR handler | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **One `LogHandler` class with the level as data.**
   - *Why:* a new level needs no new class, and routing is just a list of destinations.
   - *Rejected:* a subclass per level, which copies the same code five times.
2. **Each destination owns its formatter and its lock.**
   - *Why:* the console can be plain while the file is JSON, and writes to different destinations don't block each other.
   - *Rejected:* one formatter for the whole logger, and one lock for the whole logger.
3. **The minimum level is checked in the logger before the chain.**
   - *Why:* a discarded message never reaches the clock or the chain.
   - *Rejected:* every handler checking the minimum, which repeats the rule.
4. **Only `LogManager` is a singleton.**
   - *Why:* requirement 7 needs shared access, and nothing else needs sharing.
   - *Rejected:* making `Logger` itself a singleton, which would make it hard to test.
5. **A failing destination returns False instead of throwing.**
   - *Why:* a broken log file must never crash the program.
   - *Rejected:* letting the error escape into the application.

### Edge cases handled
- A message below the minimum level is dropped without a timestamp being taken.
- A level with no handler is dropped and `log` returns False.
- One destination failing does not stop the others.
- Two threads asking for the manager at the same moment get the same instance.
- A second `initialize` is refused.
- The default logger is used if no one configures one.
- A None message is refused. An empty message is allowed.
- The minimum level can change while other threads are logging.

### Likely follow-up questions
1. **"How would you make logging not slow the application?"** Put a queue in front of a destination, with a worker thread that writes. That is a new class implementing `LogDestination` that wraps another destination, so nothing else changes.
2. **"How would you rotate a log file?"** Inside `FileDestination.write`: when the file is too big or a day has passed, start a new file, under the same lock.
3. **"How do you test code that uses a singleton?"** Test with a `Logger` you build yourself, because `Logger` is an ordinary class. Only the shared access point is a singleton.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Named loggers per class or package | One shared logger. Named loggers would be a map from a name to a `Logger` in the manager. |
| Changing the chain or destinations while running | A new configuration means a new `Logger`. The chain is built once. |
| Filtering by level per destination | The handler for each level chooses the destinations, which gives per-level routing. |
| Asynchronous logging | See follow-up question 1. |
| Rotating files | See follow-up question 2. |
| Real file and database access | Only the three destination `write` methods touch the outside world. |
| Logging exceptions with stack traces | The message is text. A stack trace would be added to it by the caller. |
| `info()` and `error()` shortcuts | They would only call `log`. They can be added without any other change. |

## 13. Requirements Met
1. Different severity levels: `LogLevel` and `Logger.log`.
2. A minimum level that ignores lower messages: `Logger.min_level` and the check in `Logger.log`.
3. Several destinations at once: the `destinations` list in `LogHandler`, with `ConsoleDestination`, `FileDestination` and `DatabaseDestination`.
4. Easy to add destinations: the `LogDestination` interface.
5. Timestamp, level and message, with a replaceable format: `LogEntry` and the `LogFormatter` interface.
6. A chain of handlers, each for its own level: `LogHandler.handle` and `next_handler`.
7. The same configuration from everywhere: `LogManager.get_instance`, `initialize` and `get_logger`.
8. Safe from many threads: the lock inside each destination, the lock around `min_level`, and the locked creation in `LogManager`.

## 14. Requirement Fit Check

**Setup used in every example:** the minimum level is INFO. The chain, from first to last, is FATAL, ERROR, WARN, INFO, DEBUG. The ERROR handler writes to `FileDestination("app.log", PlainFormatter)` and `DatabaseDestination("logs", JsonFormatter)`. The other handlers write to `ConsoleDestination(PlainFormatter)`. The clock is fixed at 2026-11-08 10:00:00.

1. **Different severity levels**
   - *Success:* `log(INFO, "started")` returns True. `log(ERROR, "db down")` returns True.
   - *Failure:* `log(INFO, None)` returns False and nothing is written.
2. **A minimum level**
   - *Success:* `log(DEBUG, "details")` returns False because 10 is below the minimum of 20, and no timestamp is taken. After `set_min_level(DEBUG)`, the same call returns True and the line is written.
   - *Failure to avoid:* a DEBUG message must not reach any destination while the minimum is INFO.
3. **Several destinations at once**
   - *Success:* the ERROR entry "db down" is written to the file as `2026-11-08 10:00:00 [ERROR] db down` and to the database as a JSON row.
   - *Failure:* if the file can't be written, `FileDestination.write` returns False, but the database still receives the entry and `log` still returns True.
4. **Easy to add destinations**
   - *Success:* a new `SlackDestination` class implements `write` and is added to the ERROR handler's list when the chain is built. No existing class is edited.
   - *Failure:* none. A `write` that fails only returns False.
5. **Timestamp, level, message, replaceable format**
   - *Success:* the plain format gives `2026-11-08 10:00:00 [ERROR] db down`. The JSON format gives `{"timestamp": "2026-11-08 10:00:00", "level": "ERROR", "message": "db down"}`. Passing a `JsonFormatter` to the console destination switches it with no other change.
   - *Failure:* a message with quotes is escaped by the JSON format, so the output stays valid.
6. **A chain of handlers, each for its own level**
   - *Success:* an ERROR entry goes to the FATAL handler, which passes it on. The ERROR handler writes it and stops. An INFO entry passes the FATAL, ERROR and WARN handlers and is written by the INFO handler.
   - *Failure:* if the chain has no WARN handler, a WARN entry reaches the end, `handle` returns False, and the entry is dropped.
7. **The same configuration from everywhere**
   - *Success:* two modules call `LogManager.get_instance()` and receive the same object, and `get_logger()` gives the same logger to both.
   - *Failure:* a second `initialize(other_logger)` returns False, so the first configuration stays. Before any setup, `get_logger()` returns the default logger, not None.
8. **Safe from many threads**
   - *Success:* 50 threads log to the file at the same moment. Each `FileDestination.write` holds the lock for the whole line, so every line in `app.log` is complete and none is mixed with another.
   - *Failure to avoid:* without that lock, two threads could write at the same time and produce half a line from each.
