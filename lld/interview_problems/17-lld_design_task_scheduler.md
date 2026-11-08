# Task Scheduler / Cron Executor: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a job is a record of what to run and when, a queue hands out due jobs in the right order, and a pool of workers pulls from the queue. Each job guards its own stage, and every change to a job raises its version number, so an old queue entry for a job that was updated or cancelled is simply skipped. Nobody has to dig through the queue to remove it.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The system should allow scheduling a task to run once at a given time or repeatedly at a fixed interval.
2. Users should be able to cancel or update a scheduled task.
3. The system should run tasks when they become due, earliest first.
4. Multiple tasks should run at the same time using a pool of workers.
5. Tasks should be able to have priorities.
6. A task should move through the stages of scheduled, running, completed and failed.
7. Failed tasks should be retried.
8. It should be easy to add new kinds of tasks.
9. The scheduler should stay correct when tasks are added while others are running.

**Types and conventions**
- Times are `datetime`. Workers get them from a `Clock`, so tests can control time. Intervals and delays are whole seconds.
- "Task" means the work itself. "Job" means a scheduled task with its timing, priority and stage.
- Among jobs that are all due, the higher priority runs first. Among equal priorities, the one that became due earliest runs first.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put`.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| RetryPolicy | Says how long to wait before each retry, and when to give up | max attempts, base delay |
| Schedule (interface) | Says when a job first runs and when it runs next | none |
| OneTimeSchedule | Runs once at one time | the time |
| FixedIntervalSchedule | Runs at a start time and then every interval | start, interval |
| Task (interface) | Does one piece of work | none |
| PrintTask | Prints a message | the message |
| FlakyTask | Fails a set number of times, then succeeds | the failures left |
| Clock (interface) | Says what time it is | none |
| SystemClock | Reads the real time | none |
| QueueEntry | Is one ticket in the queue for one run of a job | an unchangeable record |
| JobQueue | Hands out due entries, highest priority first | the waiting entries |
| ScheduledJob | Holds one job's timing and stage, and guards every change | stage, next run, version |
| Scheduler | Adds, updates and cancels jobs, and runs the next due one | the jobs |
| WorkerPool | Runs several worker threads that keep asking the scheduler for work | the threads |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Task, time, interval, schedule, scheduler, worker, pool, priority, stage, retry.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Task, Schedule, ScheduledJob, Scheduler, WorkerPool are classes. Time, interval and priority are values.
2. **Find fixed sets of values.** The stages and the priorities.
   - *Why:* a fixed set is an enum, so a typo like "runing" cannot exist.
   - *Result:* `JobStatus`, `Priority`.
3. **Give every fact one owner.** A job's stage, next run time and version belong to the ScheduledJob. The order of waiting work belongs to the JobQueue. The set of jobs belongs to the Scheduler.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** The next run of a repeating job is computed by its schedule. Whether a queue entry is out of date is computed by comparing versions.
5. **Give every verb to the class whose data changes.** Starting, finishing, cancelling and updating change the ScheduledJob, so it owns them. Ordering belongs to the queue.
6. **Find the action that touches several classes.** Running the next job needs the queue, the job, the task and the retry rule, in order. That is `Scheduler.run_next`.
7. **Find the hint phrases for patterns.** "Easy to add new kinds of tasks" is Command. "Once at a given time or repeatedly at a fixed interval" is Strategy for the schedule. "A pool of workers" is a queue with workers.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirements 4 and 9 name it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A job id or None for scheduling, a `bool` for cancel and update, a `JobStatus` for tracking.

## 4. Pattern Guide

### Used: Command (tasks)
- **Meaning:** a piece of work is wrapped in an object with one method, so it can be stored, queued and run later by someone who knows nothing about it.
- **Hint phrase:** "it should be easy to add new kinds of tasks".
- **Without it:** the scheduler would have an `if` on the kind of work, and every new kind would mean editing it.
- **When NOT to use it:** when the work is always the same, so a plain method call is enough.

### Used: Strategy (schedules)
- **Meaning:** a rule is a small object behind a shared interface, and the job uses whichever one it is given.
- **Hint phrase:** "run once at a given time or repeatedly at a fixed interval".
- **Without it:** the job would have an `if` on one-time or repeating, and adding a cron-style rule would mean editing it.
- **When NOT to use it:** when there is only one kind of timing.

### Used: a queue with a pool of workers (not a named design pattern)
- **Meaning:** callers add jobs and move on, and worker threads keep taking the next due job.
- **Hint phrase:** "multiple tasks should run at the same time using a pool of workers".

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A job's stage only decides which method is allowed, and each method checks it. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. The caller builds the task it wants. |
| Singleton | "exactly one instance" | Not used. A passed-in scheduler is easier to test. |

## 5. Enums

### JobStatus
- Values: `SCHEDULED`, `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED`
- *Why:* the four stages of requirement 6, plus `CANCELLED` for requirement 2, so a cancelled job is not mistaken for a finished one.
- A repeating job goes back to `SCHEDULED` after each run. Only a one-time job ends `COMPLETED` or `FAILED`.

### Priority
- Values: `LOW`, `MEDIUM`, `HIGH`, with ranks 1, 2 and 3. Method: `get_rank() -> int`.
- *Why:* requirement 5. The ranks live in the enum because the order of importance is one fact.

## 6. Classes (smallest dependency first)

### RetryPolicy
**Why this class exists:** requirement 7 retries failures. How many tries and how long to wait are two numbers that belong together.
- Attributes: `max_attempts: int`, `base_delay_seconds: int`. *Why:* the total number of tries for one run, and the first wait.
- Constructor: `RetryPolicy(max_attempts: int, base_delay_seconds: int)`. *Why:* both set once. It refuses `max_attempts < 1` or a negative delay.
- Methods: `next_delay(attempts_made: int) -> int or None`. *Why:* None once `attempts_made >= max_attempts`, meaning give up. Otherwise `base_delay_seconds * 2^(attempts_made - 1)`, so each wait doubles.
- *Extensibility:* different numbers are data.

### Schedule (interface)
**Why it is an interface:** requirement 1 has two kinds of timing, so a job depends on two questions and not on a specific kind.
- Methods:
  - `first_run() -> datetime`. *Why:* when the job runs first.
  - `next_run_after(planned: datetime) -> datetime or None`. *Why:* the next run after a planned run time. None means there is no next run. It is based on the planned time, not the time the run finished, so a repeating job does not drift.
- *Extensibility:* a cron-style rule is a new class implementing these two methods.

### OneTimeSchedule (implements Schedule)
**Why this class exists:** the first kind: run once.
- Attributes: `run_at: datetime`. *Why:* the one time.
- Constructor: `OneTimeSchedule(run_at: datetime)`. *Why:* set once.
- Methods: `first_run()` returns `run_at`. `next_run_after(...)` returns None.
- *Extensibility:* none needed.

### FixedIntervalSchedule (implements Schedule)
**Why this class exists:** the second kind: run at a start time and then every interval.
- Attributes: `start_at: datetime`, `interval_seconds: int`. *Why:* the first time and the gap.
- Constructor: `FixedIntervalSchedule(start_at: datetime, interval_seconds: int)`. *Why:* both set once. It refuses `interval_seconds < 1`.
- Methods: `first_run()` returns `start_at`. `next_run_after(planned)` returns `planned + interval_seconds`.
- *Extensibility:* none needed.

### Task (interface)
**Why it is an interface:** requirement 8 says new kinds of tasks must be easy to add, so the scheduler depends on one method and not on any kind of work.
- Methods:
  - `execute() -> bool`. *Why:* True on success, False on failure. An error thrown by a task is treated as False, so one bad task can't kill a worker.
  - `get_name() -> str`. *Why:* a label for logs and displays.
- *Extensibility:* a new kind of work is a new class implementing these two methods.

### PrintTask (implements Task)
**Why this class exists:** a simple real task.
- Attributes: `message: str`. *Why:* what to print.
- Constructor: `PrintTask(message: str)`. *Why:* set once. It refuses an empty message.
- Methods: `execute()` prints the message and returns True. `get_name()` returns "print".
- *Extensibility:* none needed.

### FlakyTask (implements Task)
**Why this class exists:** a second kind that fails a set number of times and then succeeds. It shows the retry path and is used in tests.
- Attributes: `failures_left: int`. *Why:* how many more times it will fail.
- Constructor: `FlakyTask(failures: int)`. *Why:* refuses a negative number.
- Methods: `execute()` returns False and reduces `failures_left` while it is above 0, then returns True. `get_name()` returns "flaky".
- *Extensibility:* none needed.

### Clock (interface) and SystemClock
**Why they exist:** worker threads need the current time. Asking a `Clock` means tests can use a clock they control.
- `Clock` method: `now() -> datetime`.
- `SystemClock` implements it with the real time. Constructor: `SystemClock()`.
- *Extensibility:* a test clock is a new class implementing `now`.

### QueueEntry
**Why this class exists:** the queue holds tickets, not jobs. A ticket records the job's version when it was queued, so a ticket for a job that has since been updated or cancelled is recognised as out of date and skipped.
- Attributes: `job_id: str`, `version: int`, `run_at: datetime`, `priority: Priority`, `sequence: int`. *Why each:* which job, which version, when it is due, how urgent, and the order it arrived (to break ties).
- Constructor: `QueueEntry(job_id: str, version: int, run_at: datetime, priority: Priority, sequence: int)`. *Why:* all set once.
- Methods: getters for each field.
- *Extensibility:* none needed.

### JobQueue
**Why this class exists:** requirements 3 and 5 decide which job runs next. The queue owns that order, and keeps not-yet-due entries apart.
- Attributes:
  - `waiting: list[QueueEntry]`. *Why:* entries not yet due, ordered by `run_at`.
  - `ready: list[QueueEntry]`. *Why:* entries already due, ordered by priority rank (highest first), then `run_at`, then `sequence`.
  - `next_sequence: int`. *Why:* stamps each entry with its arrival order.
  - `lock: Lock`. *Why:* callers add while workers take.
- Constructor: `JobQueue()`. *Why:* both lists empty, `next_sequence` at 0, and the lock created.
- Methods (each under the lock):
  - `add(job_id: str, version: int, run_at: datetime, priority: Priority) -> None`. *Why:* builds the `QueueEntry` with the next sequence number and puts it in `waiting`.
  - `take_ready(now: datetime) -> QueueEntry or None`. *Why:* first moves every waiting entry with `run_at <= now` into `ready`. Then removes and returns the best ready entry. None if nothing is due.
  - `size() -> int`. *Why:* read-only view.
- *Extensibility:* none needed.

### ScheduledJob
**Why this class exists:** a job's timing, stage, retries and version must change together and one at a time. The job owns them all and guards every change.
- Attributes:
  - `job_id: str`. *Why:* the identity.
  - `task: Task`. *Why:* the work.
  - `schedule: Schedule`. *Why:* the timing rule.
  - `priority: Priority`. *Why:* requirement 5.
  - `status: JobStatus`. *Why:* the stage.
  - `planned_run_at: datetime`. *Why:* the planned time of the current run, which the next run is counted from.
  - `next_try_at: datetime`. *Why:* when to try next. It equals `planned_run_at` unless a retry is waiting.
  - `attempts: int`. *Why:* how many times the current run has been tried.
  - `version: int`. *Why:* goes up whenever the job is queued again, updated or cancelled, so older queue entries are known to be stale.
  - `lock: Lock`. *Why:* makes each change one step.
- Constructor: `ScheduledJob(job_id: str, task: Task, schedule: Schedule, priority: Priority)`. *Why:* the stage starts `SCHEDULED`, both times start at `schedule.first_run()`, attempts at 0 and version at 1.
- Methods (each under the lock):
  - Getters: `get_job_id() -> str`, `get_status() -> JobStatus`, `get_priority() -> Priority`, `get_next_try_at() -> datetime`, `get_version() -> int`, `get_task() -> Task`.
  - `start_run(version: int) -> bool`. *Why:* `SCHEDULED` to `RUNNING`, only if `version` is the current one. False otherwise. This is what stops a stale entry, or a second worker, from running the job.
  - `finish_run(success: bool, now: datetime, policy: RetryPolicy) -> bool`. *Why:* `RUNNING` only. It returns True if the job must be queued again.
    - On success: attempts become 0. If `schedule.next_run_after(planned_run_at)` gives a time, that becomes both times, the stage is `SCHEDULED` and the version goes up. Otherwise the stage is `COMPLETED`.
    - On failure: attempts go up. If `policy.next_delay(attempts)` gives a delay, `next_try_at` is `now` plus the delay, the stage is `SCHEDULED` and the version goes up.
    - When retries run out: attempts become 0. A repeating job moves on to its next planned run and is `SCHEDULED`. A one-time job is `FAILED`.
  - `cancel() -> bool`. *Why:* `SCHEDULED` only. Sets `CANCELLED` and raises the version. A running job finishes its current run first, so cancel is refused while it runs.
  - `update(schedule: Schedule or None, priority: Priority or None) -> bool`. *Why:* `SCHEDULED` only. It replaces whichever is given, resets both times to the new schedule's first run, sets attempts to 0, and raises the version. False if both are None.
- *Extensibility:* none needed.

### Scheduler (the manager and only entry point)
**Why this class exists:** adding a job means storing it and queuing it, and running the next one needs the queue, the job, the task and the retry rule in a fixed order.
- Attributes:
  - `jobs: map{str: ScheduledJob}` (a job id to the job). *Why:* the one record of every job. It is a thread-safe map.
  - `queue: JobQueue`. *Why:* the order of waiting work.
  - `policy: RetryPolicy`. *Why:* the retry rule, passed in.
  - `counter: AtomicInt`. *Why:* makes job ids unique when jobs are added at once.
- Constructor: `Scheduler(queue: JobQueue, policy: RetryPolicy)`. *Why:* both are passed in so tests can use fakes. It starts with no jobs.
- Methods:
  - `schedule(task: Task, schedule: Schedule, priority: Priority) -> str`. *Why:* builds the job, stores it, queues its first run, and returns the new id. The two kinds of timing in requirement 1 are the two schedule classes.
  - `cancel(job_id: str) -> bool`. *Why:* looks up once and asks the job. The queued entry becomes stale and is skipped later.
  - `update(job_id: str, schedule: Schedule or None, priority: Priority or None) -> bool`. *Why:* looks up once and asks the job. If it agrees, the job is queued again under its new version.
  - `get_status(job_id: str) -> JobStatus or None`. *Why:* requirement 6. None if the id is unknown.
  - `run_next(now: datetime) -> bool`. *Why:* the single step each worker repeats. True if it ran a job. Steps in section 9.
- *Extensibility:* tasks, schedules and the retry rule are all passed in.

### WorkerPool
**Why this class exists:** requirement 4 runs several jobs at once. The pool owns the worker threads and how they start and stop.
- Attributes: `size: int`, `scheduler: Scheduler`, `clock: Clock`, `idle_wait_ms: int`, `running: bool`, `threads: list[Thread]`. *Why:* how many workers, where work comes from, where the time comes from, how long an idle worker waits before asking again, whether to keep going, and the threads themselves.
- Constructor: `WorkerPool(size: int, scheduler: Scheduler, clock: Clock, idle_wait_ms: int)`. *Why:* refuses `size < 1` or a wait below 1. No threads run yet.
- Methods:
  - `start() -> bool`. *Why:* starts `size` threads. Each repeats: if `scheduler.run_next(clock.now())` is False, wait `idle_wait_ms`, until `running` is False. False if already started.
  - `shutdown() -> None`. *Why:* sets `running` to False and waits for each thread to finish its current job, so no job is cut off halfway.
- *Extensibility:* a bigger pool is a different `size`.

## 7. Relationships
- Scheduler **has** ScheduledJob (created inside `schedule`).
- JobQueue **has** QueueEntry (created inside `add`).
- ScheduledJob **uses** Task, Schedule (passed in, and they live on).
- Scheduler **uses** JobQueue, RetryPolicy (passed in).
- WorkerPool **uses** Scheduler, Clock (passed in).
- OneTimeSchedule, FixedIntervalSchedule **implement** Schedule. PrintTask, FlakyTask **implement** Task. SystemClock **implements** Clock.
- Enums are always **uses**: ScheduledJob to JobStatus and Priority, QueueEntry to Priority.

## 8. Interaction

### Layers
1. **Entry points:** Scheduler (for callers) and WorkerPool (which drives the work).
2. **Ordering:** JobQueue and QueueEntry.
3. **Domain record that guards itself:** ScheduledJob.
4. **Rules and work passed in:** Schedule, Task, RetryPolicy, Clock.

### Call chains
- **Schedule a job**
  1. `Caller -> Scheduler.schedule(task, schedule, priority)`.
  2. `Scheduler -> ScheduledJob(...)`, then `jobs.put`, then `JobQueue.add(job_id, version, first run, priority)`.
- **A worker runs the next job**
  1. `Worker thread -> Scheduler.run_next(clock.now())`.
  2. `Scheduler -> JobQueue.take_ready(now)`: the best due entry.
  3. `Scheduler -> ScheduledJob.start_run(entry version)`: stale or busy entries are refused here.
  4. `Scheduler -> Task.execute()`: run with no lock held.
  5. `Scheduler -> ScheduledJob.finish_run(success, now, policy)`, then, if it must run again, `JobQueue.add(...)`.
- **Cancel or update**
  1. `Caller -> Scheduler.cancel(job_id)` or `update(...)`, then `ScheduledJob.cancel()` or `update(...)`.
  2. On an update, `Scheduler -> JobQueue.add(...)` with the new version.

### Who may call whom
- Callers call `Scheduler` only. Worker threads call `Scheduler.run_next` only.
- `Scheduler` calls `JobQueue`, `ScheduledJob` and `Task`.
- `ScheduledJob` calls `Schedule` and `RetryPolicy`.
- Tasks call no scheduler class. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Schedule a job
1. `schedule(task: Task, schedule: Schedule, priority: Priority)`. The id comes from `counter`.
2. Build the `ScheduledJob`, store it, then `queue.add(job_id, 1, schedule.first_run(), priority)`. Return the id.
- **Failure:** invalid inputs are refused when the schedule or the job is built, so nothing is stored or queued.

### Run the next job (one worker step)
1. `run_next(now: datetime)`. `entry: QueueEntry or None = queue.take_ready(now)`. If None, return False.
2. `job = jobs.get(entry.get_job_id())`. If None, return True (the entry is dropped).
3. `job.start_run(entry.get_version())`. If False, the entry is stale or the job is not waiting, so return True and drop it. *Why this is safe:* the job's own lock decides, so only one worker can ever start a given run.
4. `success: bool = job.get_task().execute()`. An error counts as False. *Why no lock is held here:* a slow task must not block other jobs or callers.
5. `again: bool = job.finish_run(success, now, policy)`. If True, `queue.add(job_id, job version, job next_try_at, job priority)`.
6. Return True.
- **Failure and undo:** a failed task changes only its own job record, and the retry or next run is queued with a new version. A stale entry is dropped without touching anything.

### Cancel
1. `cancel(job_id: str)`. Look up once. If None, return False. Return `job.cancel()`.
2. The old queue entry stays in the queue, and is dropped in step 3 of `run_next` because its version is old.
- **Failure:** a running, completed, failed or already cancelled job is refused.

### Update
1. `update(job_id: str, schedule: Schedule or None, priority: Priority or None)`. Look up once. If None, return False.
2. If `job.update(schedule, priority)` is True, add a new queue entry with the new version and time. Return its result.
- **Failure:** a job that is not `SCHEDULED` is refused, with no change.

### Concurrency (requirements 4 and 9)
- **Queue:** the lock sits inside `JobQueue`, around `add` and `take_ready`. Callers can add jobs while workers take them, and each entry is handed to exactly one worker.
- **Job:** the lock sits inside `ScheduledJob`, around every change. `start_run` checks the version and the stage and changes the stage as one step, so a job can never run twice at once.
- **Map and ids:** `jobs` is a thread-safe map, and ids come from an atomic counter.
- **No lock while working:** a task runs with no lock held.
- **Lock order:** no method ever holds two locks at once. The scheduler finishes with the queue before it touches a job, so no deadlock is possible.
- **Why stale entries are skipped instead of removed:** removing an entry from the middle of a priority queue is slow and needs extra locking. A version check is one comparison.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new kind of task (send a report, clean files) | A new class implementing `Task`, passed to `schedule` | No. |
| A new kind of timing (cron expression, weekdays only) | A new class implementing `Schedule` | No. |
| More or fewer workers | A different `size` for `WorkerPool` | No. It is data. |
| Different retry timing | A different `RetryPolicy` | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **Version numbers instead of removing entries from the queue.**
   - *Why:* cancel and update stay fast, and a stale entry is recognised with one comparison.
   - *Rejected:* searching the queue to remove the old entry on every cancel or update.
2. **`start_run` checks the version and the stage as one step.**
   - *Why:* a job can never run twice at once, even if two entries for it are picked by two workers.
   - *Rejected:* checking the stage in the scheduler and then setting it, which two workers could both pass.
3. **A repeating job's next run is counted from the planned time.**
   - *Why:* the job does not drift later and later because of slow runs.
   - *Rejected:* `finish time + interval`.
4. **A task runs with no lock held.**
   - *Why:* a slow task can't block other jobs or callers.
   - *Rejected:* holding the job lock during the whole run, which would make `cancel` and `get_status` wait.
5. **Due jobs are ordered by priority first, then by time.**
   - *Why:* "earliest first" applies among jobs that are equally important, and an urgent job is not stuck behind an older routine one.
   - *Rejected:* time first, which would let a pile of overdue low-priority jobs delay an urgent one.

### Edge cases handled
- A cancelled or updated job's old entry is skipped.
- A job can't be cancelled or updated while it runs.
- A failed run retries with a doubling delay, and the last failure ends a one-time job as `FAILED`.
- A repeating job whose run keeps failing moves on to its next planned run.
- A task that throws an error counts as a failure and doesn't kill the worker.
- A repeating job can't overlap itself, because it is `RUNNING` until its run ends.
- A worker that falls behind runs each missed repeating run once and then catches up to the schedule.
- Jobs added while workers are busy are queued and picked up when due.

### Likely follow-up questions
1. **"Why do idle workers wait and check again instead of being woken?"** It is simple and correct. A better version uses a condition variable that `add` signals, and a wait until the next `run_at`, so no worker checks too often.
2. **"What if the process crashes?"** Jobs are in memory and are lost. A durable store would keep jobs and entries, and `Scheduler` would reload them on start.
3. **"How would you stop a slow task from running forever?"** Give each run a time limit and treat a timeout as a failure, so it goes through the same retry path.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Cancelling a task while it runs | A running task finishes its current run. Stopping it mid-run needs the task to support it. |
| Surviving a crash | See follow-up question 2. |
| Time limits on a run | See follow-up question 3. |
| Cron expressions | A new `Schedule` class. |
| Jobs that depend on other jobs | No requirement asks for it. |
| A history of every run | `get_status` shows the current stage. A run log would be a list kept by the scheduler. |
| Waking workers instead of polling | See follow-up question 1. |

## 13. Requirements Met
1. Once or at a fixed interval: `Scheduler.schedule` with `OneTimeSchedule` or `FixedIntervalSchedule`.
2. Cancel or update: `Scheduler.cancel` and `update`, with `ScheduledJob.cancel` and `update` and the version check.
3. Due jobs, earliest first: `JobQueue.take_ready` and its ordering.
4. A pool of workers: `WorkerPool` and `Scheduler.run_next`.
5. Priorities: `Priority` and the ordering of `ready` in `JobQueue`.
6. The stages: `JobStatus` and the guarded methods of `ScheduledJob`.
7. Retry failures: `RetryPolicy` and `ScheduledJob.finish_run`.
8. New kinds of tasks: the `Task` interface.
9. Correct while jobs are added during runs: the locks in `JobQueue` and `ScheduledJob`, the thread-safe map, and the version check in `start_run`.

## 14. Requirement Fit Check

**Setup used in every example:** the `RetryPolicy` is 3 attempts with a base delay of 10 seconds. A `WorkerPool` of size 2. Times are on one day, starting at 10:00:00.

1. **Once or at a fixed interval**
   - *Success:* `schedule(PrintTask("hi"), OneTimeSchedule(10:05), MEDIUM)` returns `J1`, due at 10:05. `schedule(PrintTask("tick"), FixedIntervalSchedule(10:00, 60), LOW)` returns `J2`, which runs at 10:00, 10:01, 10:02, and so on.
   - *Failure:* `FixedIntervalSchedule(10:00, 0)` is refused at construction, so no job is created.
2. **Cancel or update**
   - *Success:* `update("J1", OneTimeSchedule(10:10), None)` returns True and J1's version becomes 2. At 10:05 the old version-1 entry is taken and dropped by `start_run`, and J1 runs at 10:10. `cancel("J2")` returns True, and J2's queued entry is later dropped.
   - *Failure:* `cancel` on a job that is running returns False. `cancel` on a completed job returns False. `update("J1", None, None)` returns False.
3. **Due jobs, earliest first**
   - *Success:* J3 is due at 10:01 and J4 at 10:02, both MEDIUM. At 10:03 a worker gets J3 first, then J4.
   - *Failure to avoid:* at 10:00, `take_ready` returns None and nothing runs early.
4. **A pool of workers**
   - *Success:* three jobs are due at 10:00. The two workers each take one at the same time. The third waits in the queue until a worker is free.
   - *Failure to avoid:* one job is never handed to two workers, because each entry leaves the queue once.
5. **Priorities**
   - *Success:* at 10:00 a LOW job (due 09:59) and a HIGH job (due 10:00) are both due. The HIGH job runs first.
   - *Failure to avoid:* the HIGH job must not wait just because the LOW one became due earlier.
6. **The stages**
   - *Success:* J1 goes `SCHEDULED`, then `RUNNING` when a worker starts it, then `COMPLETED`.
   - *Failure:* a one-time job whose every try fails ends `FAILED`. `get_status("J99")` returns None.
7. **Retry failures**
   - *Success:* `FlakyTask(1)` as a one-time job fails at 10:00. It is `SCHEDULED` again for 10:00:10, succeeds then, and ends `COMPLETED`.
   - *Failure:* `FlakyTask(5)` fails at 10:00, at 10:00:10 and at 10:00:30. After the third failure there are no retries left, and it is `FAILED`.
8. **New kinds of tasks**
   - *Success:* a new `ReportTask` class implements `execute` and `get_name`, and is passed to `schedule`. The scheduler, the queue and the workers are unchanged.
   - *Failure:* a task that throws an error is treated as a failed run and retried, and the worker keeps going.
9. **Correct while jobs are added during runs**
   - *Success:* while both workers run jobs, a caller schedules J6. The queue lock makes the add safe, and J6 runs when it is due and a worker is free.
   - *Failure to avoid:* two workers taking two entries for the same job can't both run it, because `start_run` lets only the one with the current version and the `SCHEDULED` stage through.
