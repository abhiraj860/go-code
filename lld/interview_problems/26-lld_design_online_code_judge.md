# Online Code Judge / IDE System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** submitting only records the code and puts it on a queue, so the user gets an answer at once. Worker threads take submissions one by one, and for each one a fresh, sealed sandbox is created, the code is compiled and run against every test, and the sandbox is thrown away. A language is just data (a file name, a compile command and a run command), so adding one needs no code. Leaderboards are computed from the judged submissions, never stored.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. Users should be able to browse problems with a statement, difficulty, tags and test cases.
2. Users should be able to submit a solution in a language of their choice.
3. The system should compile and run each submission against the test cases within time and memory limits.
4. Each submission should receive a verdict such as accepted, wrong answer, time limit exceeded, runtime error or compile error.
5. It should be easy to add support for new languages.
6. Submissions should be queued and processed in the background by multiple workers.
7. Each submission should run in isolation from the others.
8. Users should be able to view their submission history.
9. Contests should rank participants on a leaderboard by score and time.

**Types and conventions**
- Times are `datetime`, passed in by the caller. Time limits are milliseconds and memory limits are megabytes, both `int`.
- Users are identified only by a `user_id: str`.
- When browsing, users see only the sample test cases. The hidden ones are used for judging.
- Output is compared after removing spaces at the end of each line and empty lines at the end.
- In a contest, a solved problem scores its points, and the time is the minutes from the contest start to the accepted submission plus a penalty for each earlier wrong submission on it. More points ranks higher, and equal points rank by less time.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put`.
- Running untrusted code is the outside world, so it sits behind the `Sandbox` interfaces.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Difficulty, Verdict, SubmissionStatus (enums) | Name the fixed values | none |
| TestCase | Holds one input and its expected output | input, output, sample flag |
| Problem | Holds a problem and its limits | statement, difficulty, tags, tests, limits |
| ProblemCatalog | Stores problems and filters them for browsing | the problems |
| LanguageSpec | Describes how to build and run one language | file name, commands |
| ExecResult | Records what happened when a command ran | exit code, output, time, memory, limit flags |
| Sandbox (interface) | Runs commands sealed off from everything else | none |
| SandboxFactory (interface) | Makes a fresh sandbox | none |
| JudgeResult | Records the outcome of judging | verdict, tests passed, time, memory, message |
| Judge | Compiles and runs one submission in its own sandbox | none |
| Submission | Holds one submission and its progress | status, result |
| SubmissionQueue | Hands out waiting submissions, oldest first | the waiting ids |
| LeaderboardRow | Gives one line of a leaderboard | an unchangeable record |
| Contest | Holds a contest's problems, points, times and participants, and ranks them | times, points, participants |
| JudgeService | Takes submissions, runs a worker step, and answers history and leaderboard queries | languages, submissions, contests |
| JudgeWorkerPool | Runs several worker threads | the threads |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Problem, statement, difficulty, tag, test case, user, solution, language, submission, time limit, memory limit, verdict, worker, history, contest, leaderboard, score.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Problem, TestCase, Submission, LanguageSpec, Contest are classes. Statement, tag and score are values.
2. **Find fixed sets of values.** Difficulty, verdicts, and a submission's progress.
   - *Result:* `Difficulty`, `Verdict`, `SubmissionStatus`.
3. **Give every fact one owner.** Test cases and limits belong to the Problem. A submission's progress and result belong to the Submission. A contest's points and times belong to the Contest.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** A user's history and a contest's leaderboard are computed from submissions.
   - *Why:* a stored leaderboard would need updating after every judged submission, and one missed update makes it wrong.
5. **Give every verb to the class whose data changes.** Starting and finishing change the Submission. Judging is the Judge's work, which changes nothing outside its sandbox.
6. **Find the action that touches several classes.** A worker step needs the queue, the submission, the problem, the language and the judge, in order. That is `JudgeService.process_next`.
7. **Find the hint phrases.** "Easy to add support for new languages" points to Strategy, but languages here differ only in their commands, so data is enough (section 4). "Each submission should run in isolation" is the sandbox boundary. "Queued and processed in the background by multiple workers" is a queue with workers.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirement 6 names it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A submission id or None for submitting, a `Verdict` inside a `JudgeResult`, lists for history and leaderboards.

## 4. Pattern Guide

### Used: a queue with a pool of workers (not a named design pattern)
- **Meaning:** callers add work and move on, and worker threads take the next item.
- **Hint phrase:** "submissions should be queued and processed in the background by multiple workers".

### Considered: Strategy for languages, and why data was chosen instead
- **Hint phrase:** "it should be easy to add support for new languages".
- **Why data is enough:** every language here is "write the code to a file, maybe run a compile command, then run a run command". Only the file name and the two commands differ, so a `LanguageSpec` record covers them, and a new language is one new record.
- **When Strategy would be right:** if a language needed different steps (special output checking, a different build process), `LanguageSpec` would become an interface with one class per odd language.

### The sandbox interfaces (a boundary, not a pattern)
- **Hint phrase:** "each submission should run in isolation from the others".
- **Why interfaces:** real isolation (containers, jails) is outside this design. The judge depends on `Sandbox` and `SandboxFactory`, and tests can use a fake.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A submission's status only decides which method is allowed. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Only as the `SandboxFactory` boundary, which makes one kind of thing. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### Difficulty
- Values: `EASY`, `MEDIUM`, `HARD`. *Why:* requirement 1.

### Verdict
- Values: `ACCEPTED`, `WRONG_ANSWER`, `TIME_LIMIT_EXCEEDED`, `MEMORY_LIMIT_EXCEEDED`, `RUNTIME_ERROR`, `COMPILE_ERROR`
- *Why:* requirement 4 lists these "such as". Memory is included because requirement 3 sets memory limits.

### SubmissionStatus
- Values: `QUEUED`, `RUNNING`, `JUDGED`
- *Why:* a user can see whether their submission is waiting, being judged, or done. The verdict exists only once it is `JUDGED`.

## 6. Classes (smallest dependency first)

### TestCase
**Why this class exists:** an input and its expected output belong together, and some are shown to users while others are hidden.
- Attributes: `input_text: str`, `expected_output: str`, `is_sample: bool`. *Why:* what is fed in, what must come out, and whether users may see it.
- Constructor: `TestCase(input_text: str, expected_output: str, is_sample: bool)`. *Why:* all set once.
- Methods: getters for each field.
- *Extensibility:* a weight per test, for partial scores, would be an extra field.

### Problem
**Why this class exists:** requirement 1 browses problems. A problem owns its text, its tests and its limits, so judging and browsing read the same facts.
- Attributes: `problem_id: str`, `title: str`, `statement: str`, `difficulty: Difficulty`, `tags: set[str]`, `test_cases: list[TestCase]`, `time_limit_ms: int`, `memory_limit_mb: int`. *Why each:* exactly what requirements 1 and 3 name.
- Constructor: `Problem(...)` with each field. *Why:* all set once. It refuses empty text, no test cases, no sample test case, or limits of 0 or less.
- Methods:
  - Getters for each field.
  - `get_sample_cases() -> list[TestCase]`. *Why:* what users may see. The full list is only for the judge.
  - `has_tag(tag: str) -> bool`. *Why:* used by browsing filters.
- *Extensibility:* none needed.

### ProblemCatalog
**Why this class exists:** browsing is about the set of problems, which no single problem knows.
- Attributes: `problems: map{str: Problem}` (a problem id to the Problem). *Why:* the one record of problems.
- Constructor: `ProblemCatalog()`.
- Methods:
  - `add_problem(problem: Problem) -> bool`. *Why:* False if the id exists.
  - `get_problem(problem_id: str) -> Problem or None`.
  - `list_problems(difficulty: Difficulty or None, tag: str or None) -> list[Problem]`. *Why:* requirement 1. None means "any". Sorted by id.
- *Extensibility:* more filters are added here.

### LanguageSpec
**Why this class exists:** requirement 5. Each language is described by data, so adding one needs no code.
- Attributes: `language_id: str`, `source_file_name: str`, `compile_command: str or None`, `run_command: str`. *Why:* the name users pick, the file the code is saved as, the build command (None for languages run directly), and the run command.
- Constructor: `LanguageSpec(language_id: str, source_file_name: str, compile_command: str or None, run_command: str)`. *Why:* all set once. It refuses empty values.
- Methods: getters, and `needs_compile() -> bool`.
- *Extensibility:* a new language is a new `LanguageSpec`.

### ExecResult
**Why this class exists:** running a command in a sandbox gives several facts at once, which the judge turns into a verdict.
- Attributes: `exit_code: int`, `stdout: str`, `stderr: str`, `time_ms: int`, `memory_mb: int`, `timed_out: bool`, `memory_exceeded: bool`.
- Constructor: `ExecResult(...)` with each field. *Why:* all set once.
- Methods: getters for each field.
- *Extensibility:* none needed.

### Sandbox (interface) and SandboxFactory (interface)
**Why they exist:** requirement 7. Each submission must run sealed off: its own files, no network, and hard time and memory limits. The real mechanism (containers, jails) is outside the design, so the judge depends only on these methods.
- `Sandbox` methods:
  - `write_file(name: str, content: str) -> bool`.
  - `execute(command: str, input_text: str, time_limit_ms: int, memory_limit_mb: int) -> ExecResult`. *Why:* the sandbox stops the command at the limits and reports which limit was hit.
  - `destroy() -> None`. *Why:* removes everything the submission created.
- `SandboxFactory` method: `create() -> Sandbox`. *Why:* a brand-new sandbox for every submission.
- *Extensibility:* a different isolation technology is a new pair of classes.

### JudgeResult
**Why this class exists:** judging produces a verdict plus details a user wants to see.
- Attributes: `verdict: Verdict`, `passed: int`, `total: int`, `max_time_ms: int`, `max_memory_mb: int`, `message: str`. *Why:* the outcome, how many tests passed out of how many, the worst time and memory, and a short note (such as the compiler's error or the first failed test number).
- Constructor: `JudgeResult(...)` with each field. *Why:* all set once.
- Methods: getters for each field.
- *Extensibility:* none needed.

### Judge
**Why this class exists:** judging one submission is a self-contained job. It holds no data of its own, so any number of workers can share one judge.
- Attributes: `sandbox_factory: SandboxFactory`, `compile_time_limit_ms: int`, `compile_memory_limit_mb: int`. *Why:* how to get a sandbox, and fixed limits for the compile step.
- Constructor: `Judge(sandbox_factory: SandboxFactory, compile_time_limit_ms: int, compile_memory_limit_mb: int)`. *Why:* refuses limits of 0 or less.
- Methods: `evaluate(source: str, language: LanguageSpec, problem: Problem) -> JudgeResult`. *Why:* steps in section 9. It always destroys its sandbox, even if something goes wrong.
- *Extensibility:* special output checking (any order accepted, decimal tolerance) would be a checker passed to the judge.

### Submission
**Why this class exists:** a submission is the record of one attempt and its progress. It guards its own status so two workers can't judge it at once.
- Attributes:
  - `submission_id: str`, `user_id: str`, `problem_id: str`, `language_id: str`, `source: str`. *Why:* what was sent.
  - `submitted_at: datetime`, `contest_id: str or None`. *Why:* when, and which contest it counts for, if any.
  - `status: SubmissionStatus`, `result: JudgeResult or None`. *Why:* progress, and the result once judged.
  - `lock: Lock`. *Why:* status changes run one at a time.
- Constructor: `Submission(submission_id: str, user_id: str, problem_id: str, language_id: str, source: str, submitted_at: datetime, contest_id: str or None)`. *Why:* the status starts `QUEUED` and the result None. It refuses empty source.
- Methods (each under the lock):
  - Getters for each field, and `get_verdict() -> Verdict or None`.
  - `start() -> bool`. *Why:* `QUEUED` to `RUNNING`. False otherwise.
  - `finish(result: JudgeResult) -> bool`. *Why:* `RUNNING` to `JUDGED`, storing the result. False otherwise.
- *Extensibility:* none needed.

### SubmissionQueue
**Why this class exists:** requirement 6 queues submissions. The queue owns the waiting order and lets workers sleep until there is work.
- Attributes: `ids: list[str]` (submission ids, oldest first), `lock: Lock`, `not_empty: Condition`, `closed: bool`.
- Constructor: `SubmissionQueue()`.
- Methods (each under the lock):
  - `put(submission_id: str) -> bool`. *Why:* False if closed. Otherwise add at the back and wake one waiting worker.
  - `take(timeout_ms: int) -> str or None`. *Why:* waits while empty and open, up to the timeout. None on timeout or when closed and empty.
  - `close() -> None`. *Why:* wakes every waiting worker so they can stop.
- *Extensibility:* priority for contests would be a second queue taken first.

### LeaderboardRow
**Why this class exists:** a leaderboard line is a fixed record for display.
- Attributes: `rank: int`, `user_id: str`, `score: int`, `time_minutes: int`, `solved: int`.
- Constructor: `LeaderboardRow(...)` with each field.
- Methods: getters for each field.

### Contest
**Why this class exists:** requirement 9 ranks participants. A contest owns its problems, points, times and participants, and the scoring rule.
- Attributes:
  - `contest_id: str`, `name: str`, `start: datetime`, `end: datetime`.
  - `points: map{str: int}` (a problem id to the points it is worth). *Why:* which problems count, and how much.
  - `penalty_minutes: int`. *Why:* time added for each wrong submission before a problem is solved.
  - `participants: set[str]`, `lock: Lock`. *Why:* who is registered, guarded because people register while others submit.
- Constructor: `Contest(contest_id: str, name: str, start: datetime, end: datetime, points: map{str: int}, penalty_minutes: int)`. *Why:* refuses `end <= start`, no problems, points of 0 or less, or a negative penalty.
- Methods:
  - `register(user_id: str) -> bool`. *Why:* False if already registered or the contest has ended.
  - `accepts(user_id: str, problem_id: str, at: datetime) -> bool`. *Why:* True if the user is registered, the problem is in the contest, and `start <= at < end`.
  - `build_leaderboard(submissions: list[Submission]) -> list[LeaderboardRow]`. *Why:* computed. For each participant and problem, take the judged submissions in time order. The first `ACCEPTED` solves it: add its points, and add its minutes since the start plus `penalty_minutes` for each earlier non-accepted one. Rank by score (higher first), then time (lower first), then user id.
- *Extensibility:* a different scoring rule would be a scorer passed to `build_leaderboard`.

### JudgeService (the manager and only entry point)
**Why this class exists:** submitting, judging, history and leaderboards each cross several classes, and submissions, languages and contests need one place to be found.
- Attributes:
  - `catalog: ProblemCatalog`, `judge: Judge`, `queue: SubmissionQueue`. *Why:* passed in.
  - `languages: map{str: LanguageSpec}` (a language id to its description). *Why:* the one record of supported languages.
  - `submissions: map{str: Submission}` (a submission id to the Submission), `contests: map{str: Contest}` (a contest id to the Contest). *Why:* the one record of each. Thread-safe maps.
  - `counter: AtomicInt`. *Why:* makes submission ids unique.
- Constructor: `JudgeService(catalog: ProblemCatalog, judge: Judge, queue: SubmissionQueue)`.
- Methods (outer ones take ids and look up once):
  - `add_language(spec: LanguageSpec) -> bool`. *Why:* requirement 5. False if the id exists.
  - `submit(user_id: str, problem_id: str, language_id: str, source: str, contest_id: str or None, now: datetime) -> str or None`. *Why:* requirement 2. Returns the new id at once, with the submission `QUEUED`. None if the problem or language is unknown, or the contest does not accept it.
  - `process_next(timeout_ms: int) -> bool`. *Why:* one worker step. True if it judged something.
  - `get_submission(submission_id: str, user_id: str) -> Submission or None`. *Why:* a user sees only their own.
  - `get_history(user_id: str) -> list[Submission]`. *Why:* requirement 8. Computed, newest first.
  - `create_contest(contest: Contest) -> bool`, `register_for_contest(contest_id: str, user_id: str) -> bool`.
  - `get_leaderboard(contest_id: str) -> list[LeaderboardRow] or None`. *Why:* requirement 9. It passes the contest's judged submissions to `Contest.build_leaderboard`.
- *Extensibility:* languages, problems and contests are data. The sandbox is passed in through the judge.

### JudgeWorkerPool
**Why this class exists:** requirement 6's "multiple workers". The pool owns the threads.
- Attributes: `size: int`, `service: JudgeService`, `wait_ms: int`, `threads: list[Thread]`, `running: bool`.
- Constructor: `JudgeWorkerPool(size: int, service: JudgeService, wait_ms: int)`. *Why:* refuses `size < 1`.
- Methods: `start() -> bool` starts `size` threads that repeat `service.process_next(wait_ms)` while running. `stop() -> None` sets `running` to False, closes the queue to wake them, and waits for each to finish its current submission.

## 7. Relationships
- ProblemCatalog **has** Problem (added and kept there). Problem **has** TestCase.
- JudgeService **has** Submission, Contest (created inside `submit` and `create_contest`).
- JudgeService **uses** ProblemCatalog, Judge, SubmissionQueue, LanguageSpec.
- Judge **uses** SandboxFactory. Submission **uses** JudgeResult.
- JudgeWorkerPool **uses** JudgeService.
- Enums are always **uses**: Problem to Difficulty, Submission to SubmissionStatus, JudgeResult to Verdict.
- `Sandbox`, `ExecResult` and `LeaderboardRow` appear only as method inputs and outputs, so no attribute holds them and they have no line.

## 8. Interaction

### Layers
1. **Entry points:** JudgeService and JudgeWorkerPool.
2. **Work order:** SubmissionQueue.
3. **Domain records that guard themselves:** Submission, Contest.
4. **Judging:** Judge, with LanguageSpec and the sandbox interfaces.
5. **Plain data:** Problem, TestCase, ExecResult, JudgeResult, LeaderboardRow.

### Call chains
- **Submit**
  1. `User -> JudgeService.submit(...)`.
  2. `JudgeService -> ProblemCatalog.get_problem`, `languages.get`, and `Contest.accepts` if it is for a contest.
  3. `JudgeService -> Submission(...)`, store it, then `SubmissionQueue.put(id)`. Return the id.
- **Judge one submission (worker step)**
  1. `Worker -> JudgeService.process_next(wait)`.
  2. `JudgeService -> SubmissionQueue.take(wait)`, then `Submission.start()`.
  3. `JudgeService -> Judge.evaluate(source, language, problem)`.
  4. `Judge -> SandboxFactory.create()`, then `Sandbox.write_file`, `Sandbox.execute` (compile, then each test), then `Sandbox.destroy()`.
  5. `JudgeService -> Submission.finish(result)`.
- **Leaderboard**
  1. `User -> JudgeService.get_leaderboard(contest_id)`, then `Contest.build_leaderboard(contest submissions)`.

### Who may call whom
- Users call `JudgeService` only. Worker threads call `process_next` only.
- `JudgeService` calls the catalog, the queue, submissions, contests and the judge.
- `Judge` calls the sandbox interfaces only.
- Submitted code can reach only its own sandbox. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Submit
1. `submit(user_id, problem_id, language_id, source, contest_id, now)`. If the problem or language is unknown, or `source` is empty, return None.
2. If `contest_id` is given, the contest must exist and `accepts(user_id, problem_id, now)` must be True. Otherwise return None.
3. Build the `Submission`, store it, `queue.put(id)`, and return the id.
- **Failure:** nothing is stored or queued.

### Judge one submission
1. `process_next(timeout_ms)`. `id = queue.take(timeout_ms)`. If None, return False.
2. `submission = submissions.get(id)`. If `submission.start()` is False, return True (it was already picked up).
3. Look up the problem and language once, then `result = judge.evaluate(source, language, problem)`.
4. `submission.finish(result)`. Return True.

### Inside `Judge.evaluate`
1. `sandbox = sandbox_factory.create()`. Write the source as `language.source_file_name`.
2. If the language needs compiling, run the compile command with the compile limits. A non-zero exit or a limit hit gives `COMPILE_ERROR`, with the compiler's error as the message.
3. For each test case in order, run the run command with the input and the problem's limits:
   - `timed_out` gives `TIME_LIMIT_EXCEEDED`.
   - `memory_exceeded` gives `MEMORY_LIMIT_EXCEEDED`.
   - A non-zero exit gives `RUNTIME_ERROR`.
   - Output that differs after trimming gives `WRONG_ANSWER`.
   - The first failing test stops the run. *Why stop:* the verdict is already known, and it saves worker time.
4. If every test passes, `ACCEPTED`.
5. Always `sandbox.destroy()`, whatever happened. *Why:* nothing from one submission may remain for the next.

### History and leaderboard
1. `get_history(user_id)` filters `submissions` by user, newest first.
2. `get_leaderboard(contest_id)` passes the judged submissions with that contest id to `build_leaderboard`.
- **Failure:** an unknown contest returns None.

### Concurrency (requirement 6)
- **Queue:** the lock sits inside `SubmissionQueue`, around `put`, `take` and `close`, with a condition so idle workers sleep. Each id is handed to exactly one worker.
- **Submission:** the lock sits inside `Submission`, around `start` and `finish`, so it is judged at most once.
- **Contest:** the lock sits inside `Contest`, around `participants`.
- **Judge:** it holds no changing data, so all workers share one. Each submission gets its own sandbox.
- **Maps and ids:** thread-safe maps and an atomic counter.
- **No lock while judging:** compiling and running happen with no lock held, so a slow submission never blocks others.
- **Lock order:** no method ever holds two locks at once, so no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new language | A new `LanguageSpec` passed to `add_language` | No. It is data. |
| More workers | A different `size` for `JudgeWorkerPool` | No. It is data. |
| A different isolation technology | A new class implementing `Sandbox` and `SandboxFactory` | No. |
| New problems or contests | Calls to `add_problem` and `create_contest` | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **A language is data, not a class.**
   - *Why:* every language follows the same steps and differs only in commands, so a new one is a record.
   - *Rejected:* one class per language, which would be the same code with different strings.
2. **A fresh sandbox per submission, always destroyed.**
   - *Why:* requirement 7's isolation is guaranteed by construction. One submission can't see another's files.
   - *Rejected:* reusing sandboxes between submissions, which leaks files and state.
3. **Submitting only queues.**
   - *Why:* the user gets an answer at once, and judging runs at the workers' pace.
   - *Rejected:* judging inside `submit`, which would make the user wait for compilation and every test.
4. **The leaderboard is computed from judged submissions.**
   - *Why:* it is always right, including after a rejudge.
   - *Rejected:* a stored score table updated on every verdict.
5. **Judging stops at the first failing test.**
   - *Why:* the verdict is known and worker time is saved.
   - *Rejected:* running every test anyway, which only helps if partial scores are wanted.

### Edge cases handled
- An unknown problem or language, or empty code, is refused.
- A submission for a contest the user didn't join, or outside its times, is refused.
- Code that doesn't compile gets `COMPILE_ERROR` and is never run.
- A run that hits the time or memory limit gets the matching verdict.
- Spaces at the ends of lines don't cause a wrong answer.
- A submission is judged at most once, even if its id is taken twice.
- The sandbox is destroyed even when judging fails.
- Wrong submissions after a problem is solved don't add penalty.

### Likely follow-up questions
1. **"How would you scale to thousands of submissions a minute?"** Move the queue to a message broker and run workers on many machines. `JudgeService.submit` and `process_next` stay the same around it.
2. **"How would you rejudge after a test case is fixed?"** Reset the affected submissions to `QUEUED` and put them back on the queue. The leaderboard updates itself because it is computed.
3. **"How would you support problems with many correct answers?"** Pass a checker program to the judge, run it in the sandbox, and use its result instead of a text comparison.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| The real isolation technology | It is behind `Sandbox`. Containers or jails would implement it. |
| Partial scores per test | Judging stops at the first failure. Partial scores need every test run and a weight per test. |
| Custom checkers | See follow-up question 3. |
| Rejudging | See follow-up question 2. |
| User accounts | Users are only an id. |
| Showing hidden test cases | Only sample tests are shown, by design. |
| Freezing the leaderboard near the end | Not required. It would hide submissions after a cut-off time. |

## 13. Requirements Met
1. Browse problems: `ProblemCatalog.list_problems` and `get_problem`, with `Problem` and `get_sample_cases`.
2. Submit in a chosen language: `JudgeService.submit` with `languages`.
3. Compile and run within limits: `Judge.evaluate` with `Sandbox.execute` and the problem's limits.
4. Verdicts: `Verdict` and `JudgeResult`.
5. New languages easily: `LanguageSpec` and `add_language`.
6. Queued and processed by many workers: `SubmissionQueue`, `JudgeService.process_next` and `JudgeWorkerPool`.
7. Isolation: a new `Sandbox` from `SandboxFactory` per submission, always destroyed.
8. Submission history: `JudgeService.get_history`.
9. Contest leaderboard: `Contest.build_leaderboard` and `JudgeService.get_leaderboard`.

## 14. Requirement Fit Check

**Setup used in every example:** problem `P1` "Sum" (EASY, tag `math`, 1000 ms, 256 MB) with a sample test `1 2` giving `3` and hidden tests `5 7` giving `12` and `1000000000 1000000000` giving `2000000000`. Languages `python3` (file `main.py`, no compile, run `python3 main.py`) and `cpp17` (file `main.cpp`, compile `g++ -O2 -o main main.cpp`, run `./main`). A fake sandbox reports results as described. A pool of 3 workers.

1. **Browse problems**
   - *Success:* `list_problems(None, "math")` returns [P1]. Its sample cases are only `1 2` giving `3`.
   - *Failure:* `get_problem("P9")` returns None.
2. **Submit in a chosen language**
   - *Success:* `submit("U1", "P1", "python3", code, None, now)` returns `S1`, which is `QUEUED`.
   - *Failure:* `submit(..., "cobol", ...)` returns None because the language is unknown.
3. **Compile and run within limits**
   - *Success:* a correct C++ solution is compiled once, then run on all three tests within 1000 ms and 256 MB.
   - *Failure:* a solution that loops forever is stopped at 1000 ms on the first test.
4. **Verdicts**
   - *Success:* the correct solution gets `ACCEPTED` with 3 of 3 passed.
   - *Failure:* printing `4` for the sample gives `WRONG_ANSWER` at test 1. The endless loop gives `TIME_LIMIT_EXCEEDED`. Dividing by zero gives `RUNTIME_ERROR`. A C++ syntax error gives `COMPILE_ERROR` with the compiler's message. A huge array gives `MEMORY_LIMIT_EXCEEDED`.
5. **New languages easily**
   - *Success:* `add_language(LanguageSpec("java17", "Main.java", "javac Main.java", "java Main"))` returns True, and Java submissions work. No code was edited.
   - *Failure:* adding `python3` again returns False.
6. **Queued and processed by many workers**
   - *Success:* 10 submissions arrive at once. Each `submit` returns at once, and the 3 workers judge 3 at a time until the queue is empty.
   - *Failure to avoid:* one submission is never judged twice, because `start` lets only one worker through.
7. **Isolation**
   - *Success:* S2 writes a file `secret.txt`. S3 is judged in a new sandbox and can't see it. Both sandboxes are destroyed afterwards.
   - *Failure to avoid:* a crashed run still has its sandbox destroyed.
8. **Submission history**
   - *Success:* `get_history("U1")` returns S3 (`ACCEPTED`), then S2 (`WRONG_ANSWER`), newest first.
   - *Failure:* `get_submission("S2", "U2")` returns None because S2 belongs to U1.
9. **Contest leaderboard**
   - *Success:* contest `C1` runs 10:00 to 12:00 with P1 worth 100 and P2 worth 200, and a penalty of 20 minutes. Alice gets a wrong answer on P1, then solves it at 10:10 (score 100, time 10 + 20 = 30), then solves P2 at 11:00 (score 300, time 30 + 60 = 90). Bob solves P2 at 10:30 and P1 at 10:40 (score 300, time 30 + 40 = 70). The leaderboard is Bob first, then Alice.
   - *Failure:* a submission at 12:05 is refused for the contest. A user who didn't register can't submit to it.
