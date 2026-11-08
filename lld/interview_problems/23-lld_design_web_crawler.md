# Web Crawler: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a URL is cleaned into one standard form and claimed in a shared "seen" set before it is queued, so it is fetched at most once. The queue of pages to visit is grouped by site, and each site may only be hit after its waiting time has passed, so politeness is enforced in one place. Workers repeat one small step: take a ready page, check the site's rules, fetch it, hand it to every processor, and queue its new links.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The crawler should start from a set of seed URLs and fetch their pages.
2. The crawler should extract links from each page and add them to the list of pages to visit.
3. The crawler should never visit the same URL twice.
4. The crawler should be able to limit how deep it goes and which domains it visits.
5. Multiple threads should crawl at the same time.
6. The crawler should limit how often it hits the same domain and follow the site's crawling rules.
7. It should be easy to add different ways of processing the fetched content.
8. Failed fetches should be retried and should time out.

**Types and conventions**
- URLs and domains are `str`. A domain is the host part, such as `a.com`, in lower case.
- Time is `int` milliseconds. Workers read it from a `Clock`, so tests can control it.
- A page's depth is 0 for a seed, and one more for each link followed.
- "The site's crawling rules" means its `robots.txt` file: paths it disallows, and an optional delay between requests.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. A "thread-safe set" supports an atomic `add_if_absent` that returns False if the value was already there. A "thread-safe map" supports an atomic `put_if_absent`.
- Fetching pages is the only contact with the outside world, so it sits behind an interface that a test can replace.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| FetchStatus (enum) | Names the outcomes of a fetch | none |
| UrlNormalizer | Turns any URL into one standard form and finds its domain | none |
| CrawlTask | Records one URL waiting to be fetched | url, depth, attempts, earliest time |
| FetchResult | Records the outcome of one fetch | status, content |
| Page | Holds one fetched page for the processors | url, depth, content |
| PageFetcher (interface) | Fetches a URL with a time limit | none |
| HttpFetcher | Fetches over the network | none |
| LinkExtractor | Finds the links in a page | none |
| RobotsRules | Holds one site's rules and checks a path | disallowed paths, delay |
| RobotsCache | Fetches each site's rules once and keeps them | the rules per domain |
| CrawlPolicy | Says how deep to go and which domains are allowed | depth limit, allowed domains |
| RetryPolicy | Says how long to wait before each retry, and when to give up | max attempts, base delay |
| ContentProcessor (interface) | Does something with a fetched page | none |
| ArchiveProcessor | Keeps a copy of every fetched page | the pages kept |
| WordCountProcessor | Counts the words on every fetched page | the counts |
| Frontier | Hands out the next page to fetch, one site at a time, respecting each site's wait | the waiting tasks, each site's next allowed time |
| Crawler | Runs one crawl step and holds the seen set | the seen URLs, failed URLs |
| CrawlerWorkerPool | Runs several worker threads until the crawl is done | the threads |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Seed URL, page, link, URL, depth, domain, thread, crawling rules, content, fetch, retry, timeout.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* CrawlTask, Page, RobotsRules, the frontier and the crawler are classes. Depth, domain and timeout are values.
2. **Find fixed sets of values.** The outcomes of a fetch.
   - *Result:* `FetchStatus`.
3. **Give every fact one owner.** "Has this URL been seen" belongs to the crawler's seen set. "When may this site be hit next" belongs to the frontier. A site's rules belong to the robots cache.
   - *Why:* if "seen" were checked in two places, a URL could slip through both.
4. **Mark what can be computed and do not store it.** A URL's standard form and its domain are computed. Whether a page may be followed is computed from its depth and domain. "Is the crawl finished" is computed from the queue and the pages in flight.
5. **Give every verb to the class whose data changes.** Claiming a URL changes the seen set. Handing out work changes the frontier. Processing a page is each processor's own job.
6. **Find the action that touches several classes.** One crawl step needs the frontier, the rules, the fetcher, the processors, the link extractor and the seen set, in order. That is `Crawler.process_next`.
7. **Find the hint phrase for a pattern.** "Easy to add different ways of processing the fetched content" is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirements 5 and 6 name it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A count of accepted seeds, a `bool` per step, and lists of pages for the processors.

## 4. Pattern Guide

### Used: Strategy (content processors)
- **Meaning:** a way of handling something is a small object behind a shared interface, and the crawler uses every one it is given.
- **Hint phrase:** "it should be easy to add different ways of processing the fetched content".
- **Without it:** the crawler would contain the indexing, saving and counting code itself, and every new use would mean editing it.
- **When NOT to use it:** when there is only ever one thing to do with a page.

### Used: a queue with a pool of workers (not a named design pattern)
- **Meaning:** workers keep taking the next ready task from a shared queue.
- **Hint phrase:** "multiple threads should crawl at the same time".

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| Observer | "notify several parties when something happens" | Close, but the processors are called directly in a fixed list by the crawler, which is simpler than a separate publisher. |
| State | "behaves differently depending on its state" | Not used. |
| Factory | "create different kinds of objects by type" | Not used. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### FetchStatus
- Values: `OK`, `NOT_FOUND`, `ERROR`, `TIMEOUT`
- *Why:* requirement 8 retries failures, but only the ones worth retrying. `ERROR` and `TIMEOUT` may succeed later. `NOT_FOUND` will not.

## 6. Classes (smallest dependency first)

### UrlNormalizer
**Why this class exists:** requirement 3 is only true if two spellings of the same URL count as one. Putting the cleaning rules in one place makes "same URL" mean one thing everywhere.
- Attributes: none. Constructor: `UrlNormalizer()`.
- Methods:
  - `normalize(url: str, base: str or None) -> str or None`. *Why:* turns a relative link into a full URL using `base`, lower-cases the scheme and host, removes the `#fragment`, removes the default port, and makes an empty path `/`. None if it is not an `http` or `https` URL.
  - `domain_of(url: str) -> str or None`. *Why:* the lower-case host. None if there is none.
  - `path_of(url: str) -> str`. *Why:* the path part, which the site's rules are checked against.
- *Extensibility:* a new cleaning rule (sorting query parameters) is one more step here.

### CrawlTask
**Why this class exists:** a URL waiting to be fetched carries its depth, how many times it has been tried, and the earliest time it may be tried again. One fixed record keeps those together.
- Attributes: `url: str`, `depth: int`, `attempts: int`, `not_before_ms: int`. *Why:* what to fetch, how far from a seed it is, tries so far, and the earliest time for a retry.
- Constructor: `CrawlTask(url: str, depth: int, attempts: int, not_before_ms: int)`. *Why:* all set once. A retry is a new task. It refuses a negative depth or attempts.
- Methods: getters for each field.
- *Extensibility:* none needed.

### FetchResult
**Why this class exists:** a fetch has an outcome and, if it worked, content. One record keeps them together.
- Attributes: `status: FetchStatus`, `content: str or None`. *Why:* what happened, and the page text when `OK`.
- Constructor: `FetchResult(status: FetchStatus, content: str or None)`. *Why:* refuses `OK` with no content.
- Methods: `get_status() -> FetchStatus`, `get_content() -> str or None`.
- *Extensibility:* a content type or final URL after redirects would be extra fields.

### Page
**Why this class exists:** processors need the URL, depth and content of a fetched page, and nothing else about the crawl.
- Attributes: `url: str`, `depth: int`, `content: str`. *Why:* what a processor needs.
- Constructor: `Page(url: str, depth: int, content: str)`. *Why:* all set once, so one processor can't change what the next one sees.
- Methods: `get_url() -> str`, `get_depth() -> int`, `get_content() -> str`.
- *Extensibility:* none needed.

### PageFetcher (interface) and HttpFetcher
**Why they exist:** fetching is the outside world. An interface lets tests serve pages from memory, and puts requirement 8's time limit in one method.
- `PageFetcher` method: `fetch(url: str, timeout_ms: int) -> FetchResult`. *Why:* it never throws. A problem comes back as `ERROR`, and a slow answer as `TIMEOUT`.
- `HttpFetcher()` does a real network request with that time limit.
- *Extensibility:* a fetcher with a browser for script-heavy pages is a new class.

### LinkExtractor
**Why this class exists:** requirement 2 finds links in a page. Parsing HTML is its own job, apart from crawling.
- Attributes: `normalizer: UrlNormalizer`. *Why:* links come out already in standard form.
- Constructor: `LinkExtractor(normalizer: UrlNormalizer)`.
- Methods: `extract(page_url: str, content: str) -> list[str]`. *Why:* the standard form of every `href` link, with duplicates and invalid links removed.
- *Extensibility:* none needed.

### RobotsRules
**Why this class exists:** requirement 6 follows each site's rules. One record holds what a site allows and how long to wait between requests.
- Attributes: `disallowed_prefixes: list[str]`, `crawl_delay_ms: int or None`. *Why:* paths starting with any of these may not be fetched, and the site's requested wait, if any.
- Constructor: `RobotsRules(disallowed_prefixes: list[str], crawl_delay_ms: int or None)`. *Why:* all set once.
- Methods:
  - `parse(text: str or None) -> RobotsRules` (a class method). *Why:* reads the lines for all crawlers. Missing or unreadable text gives "allow everything, no delay".
  - `allows(path: str) -> bool`. *Why:* False if the path starts with any disallowed prefix.
  - `get_crawl_delay_ms() -> int or None`. *Why:* read-only view.
- *Extensibility:* rules for a named crawler would be one more parsing branch.

### RobotsCache
**Why this class exists:** each site's rules should be fetched once, not before every page. The cache owns them.
- Attributes: `rules: map{str: RobotsRules}` (a domain to its rules), `fetcher: PageFetcher`, `timeout_ms: int`. *Why:* the thread-safe map of rules, how to fetch `robots.txt`, and its time limit.
- Constructor: `RobotsCache(fetcher: PageFetcher, timeout_ms: int)`.
- Methods: `get(domain: str) -> RobotsRules`. *Why:* returns the cached rules. On a miss, it fetches `https://domain/robots.txt`, parses it (a failed fetch means "allow everything"), and stores it with `put_if_absent`. If two workers fetch at once, both use whichever was stored first.
- *Extensibility:* rules that expire after a day would be a time stored with each entry.

### CrawlPolicy
**Why this class exists:** requirement 4's limits are two settings that belong together.
- Attributes: `max_depth: int`, `allowed_domains: set[str] or None`. *Why:* how far from a seed to go, and the only domains allowed (None means any).
- Constructor: `CrawlPolicy(max_depth: int, allowed_domains: set[str] or None)`. *Why:* refuses a negative depth.
- Methods:
  - `allows_domain(domain: str) -> bool`. *Why:* True if there is no list, or the domain is on it.
  - `allows_following(depth: int) -> bool`. *Why:* True if `depth < max_depth`, meaning the page's links may be queued.
- *Extensibility:* different limits are data.

### RetryPolicy
**Why this class exists:** requirement 8 retries failures. How many tries and how long to wait belong together.
- Attributes: `max_attempts: int`, `base_delay_ms: int`.
- Constructor: `RetryPolicy(max_attempts: int, base_delay_ms: int)`. *Why:* refuses `max_attempts < 1` or a negative delay.
- Methods: `next_delay(attempts_made: int) -> int or None`. *Why:* None once `attempts_made >= max_attempts`. Otherwise `base_delay_ms * 2^(attempts_made - 1)`.
- *Extensibility:* different numbers are data.

### ContentProcessor (interface)
**Why it is an interface:** requirement 7 says new ways of processing must be easy to add, so the crawler depends on one method and not on any use of the page.
- Methods: `process(page: Page) -> None`. *Why:* called once per fetched page, possibly from several threads at once, so each processor must be safe for that. A processor that fails must not stop the others, so the crawler guards each call.
- *Extensibility:* a new use (indexing, saving to disk, building a link graph) is a new class.

### ArchiveProcessor and WordCountProcessor (implement ContentProcessor)
**Why these classes exist:** two real uses, which show the interface holds more than one.
- `ArchiveProcessor()`: attributes `pages: map{str: str}` (a URL to its content) and `lock: Lock`. `process` stores the content. `get(url: str) -> str or None` reads it back.
- `WordCountProcessor()`: attributes `counts: map{str: int}` (a URL to its number of words) and `lock: Lock`. `process` counts words separated by spaces. `get_count(url: str) -> int or None` reads it.
- *Extensibility:* none needed.

### Frontier
**Why this class exists:** requirement 6 limits how often each site is hit. The frontier holds the waiting pages grouped by site and only hands out a page whose site's wait is over, so politeness lives in one place.
- Attributes:
  - `queues: map{str: list[CrawlTask]}` (a domain to its waiting tasks, oldest first). *Why:* tasks are grouped by site.
  - `next_allowed_ms: map{str: int}` (a domain to the earliest time it may be hit again). *Why:* the one fact about politeness.
  - `default_delay_ms: int`. *Why:* the wait between hits to one site when the site asks for none.
  - `domain_delays: map{str: int}` (a domain to its own wait). *Why:* a site's `robots.txt` may ask for a longer wait.
  - `in_flight: int`. *Why:* how many tasks have been handed out and not finished, so the end of the crawl can be detected.
  - `lock: Lock`. *Why:* many workers add and take at once.
- Constructor: `Frontier(default_delay_ms: int)`. *Why:* refuses a negative delay and starts empty.
- Methods (each under the lock):
  - `add(domain: str, task: CrawlTask) -> None`. *Why:* adds to the end of that domain's queue.
  - `set_domain_delay(domain: str, delay_ms: int) -> None`. *Why:* stores the larger of the site's request and the default.
  - `take(now_ms: int) -> CrawlTask or None`. *Why:* finds a domain whose queue is not empty, whose `next_allowed_ms` has come, and whose first task's `not_before_ms` has come. Among them it picks the one waiting longest. It removes that task, sets the domain's `next_allowed_ms` to `now_ms` plus its wait, adds 1 to `in_flight`, and returns the task. None if nothing is ready. Setting the next time here, at take time, means two workers can never hit one site back to back.
  - `done() -> None`. *Why:* subtracts 1 from `in_flight` when a task is finished.
  - `is_finished() -> bool`. *Why:* True if every queue is empty and `in_flight` is 0. A task in flight may still add links, so empty queues alone are not the end.
- *Extensibility:* none needed.

### Crawler (the manager and entry point for work)
**Why this class exists:** one crawl step touches the frontier, the rules, the fetcher, the processors and the seen set, in a fixed order. It also owns the seen set, the one fact behind requirement 3.
- Attributes:
  - `frontier: Frontier`, `fetcher: PageFetcher`, `extractor: LinkExtractor`, `normalizer: UrlNormalizer`, `robots: RobotsCache`, `policy: CrawlPolicy`, `retry: RetryPolicy`, `processors: list[ContentProcessor]`. *Why:* everything a step needs, all passed in.
  - `fetch_timeout_ms: int`. *Why:* requirement 8's time limit for each fetch.
  - `seen: set[str]`. *Why:* every URL ever queued, in standard form. It is a thread-safe set, and adding to it is how a URL is claimed.
  - `failed: list[str]`. *Why:* URLs given up on, for reporting. Guarded by `lock`.
  - `lock: Lock`. *Why:* guards only `failed`.
- Constructor: `Crawler(frontier: Frontier, fetcher: PageFetcher, extractor: LinkExtractor, normalizer: UrlNormalizer, robots: RobotsCache, policy: CrawlPolicy, retry: RetryPolicy, processors: list[ContentProcessor], fetch_timeout_ms: int)`. *Why:* everything is passed in so tests can use fakes. It refuses a timeout below 1.
- Methods:
  - `add_seeds(urls: list[str]) -> int`. *Why:* requirement 1. Each seed is cleaned, checked against the allowed domains, claimed in `seen`, and queued at depth 0. Returns how many were accepted.
  - `process_next(now_ms: int) -> bool`. *Why:* one worker step. False if nothing was ready. Steps in section 9.
  - `is_finished() -> bool`. *Why:* asks the frontier.
  - `get_failed() -> list[str]`. *Why:* a copy of the URLs given up on.
- Private helper: `enqueue(url: str, depth: int) -> bool`. *Why:* the one gate for new URLs. It checks the domain, then `seen.add_if_absent(url)`, then adds a task. Every path for adding a URL goes through it, so the seen check can never be skipped.
- *Extensibility:* processors, the fetcher and the policies are all passed in.

### CrawlerWorkerPool
**Why this class exists:** requirement 5 crawls with several threads. The pool owns them and stops them when the crawl is finished.
- Attributes: `size: int`, `crawler: Crawler`, `clock: Clock`, `idle_wait_ms: int`, `threads: list[Thread]`, `running: bool`.
- Constructor: `CrawlerWorkerPool(size: int, crawler: Crawler, clock: Clock, idle_wait_ms: int)`. *Why:* refuses `size < 1` or a wait below 1.
- Methods:
  - `start() -> bool`. *Why:* starts `size` threads. Each repeats `crawler.process_next(clock.now_millis())`, waits `idle_wait_ms` when nothing was ready, and stops when `crawler.is_finished()` or `running` is False. False if already started.
  - `stop() -> None`. *Why:* sets `running` to False and waits for each thread to finish its current page.
  - `await_finished() -> None`. *Why:* waits until every thread has stopped because the crawl is done.
- `Clock` is the same small interface as in other designs: `now_millis() -> int`, with `SystemClock` reading the real time.
- *Extensibility:* more threads is a different `size`.

## 7. Relationships
- Frontier **has** CrawlTask (tasks wait in its queues until taken; the crawler creates them).
- RobotsCache **has** RobotsRules (created inside `get`).
- Crawler **uses** Frontier, PageFetcher, LinkExtractor, UrlNormalizer, RobotsCache, CrawlPolicy, RetryPolicy, ContentProcessor (all passed in).
- LinkExtractor **uses** UrlNormalizer. RobotsCache **uses** PageFetcher.
- CrawlerWorkerPool **uses** Crawler, Clock.
- HttpFetcher **implements** PageFetcher. ArchiveProcessor, WordCountProcessor **implement** ContentProcessor.
- Enums are always **uses**: FetchResult to FetchStatus.
- `Page` and `FetchResult` appear only as method inputs and outputs, so no attribute holds them and they have no line.

## 8. Interaction

### Layers
1. **Entry points:** Crawler (seeds and steps) and CrawlerWorkerPool (which drives the steps).
2. **Ordering and politeness:** Frontier.
3. **Rules:** CrawlPolicy, RobotsCache with RobotsRules, RetryPolicy.
4. **Outside world and parsing:** PageFetcher, LinkExtractor, UrlNormalizer.
5. **Uses of pages:** ContentProcessor and its implementations.
6. **Plain data:** CrawlTask, FetchResult, Page.

### Call chains
- **Seed**
  1. `Owner -> Crawler.add_seeds(urls)`, then for each: `UrlNormalizer.normalize`, `CrawlPolicy.allows_domain`, `seen.add_if_absent`, `Frontier.add`.
- **One crawl step**
  1. `Worker thread -> Crawler.process_next(now)`.
  2. `Crawler -> Frontier.take(now)`: a page whose site may be hit now.
  3. `Crawler -> RobotsCache.get(domain)`, then `RobotsRules.allows(path)`, and `Frontier.set_domain_delay` with the site's delay.
  4. `Crawler -> PageFetcher.fetch(url, timeout)`.
  5. On success: `ContentProcessor.process(page)` for each processor, then `LinkExtractor.extract`, then `enqueue` for each link.
  6. On a failure worth retrying: a new `CrawlTask` with one more attempt and a later `not_before_ms`, added straight to the frontier.
  7. `Crawler -> Frontier.done()`.

### Who may call whom
- Worker threads call `Crawler.process_next` only. The owner calls `add_seeds` and the pool.
- `Crawler` calls everything it was given.
- `RobotsCache` calls `PageFetcher`. `LinkExtractor` calls `UrlNormalizer`.
- Processors call no crawler class. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Add seeds
1. `add_seeds(urls)`. For each URL, `enqueue(normalized, 0)`. Count the ones accepted.
- **Failure:** an invalid URL, a domain not allowed, or a repeat of another seed is skipped.

### One crawl step
1. `process_next(now_ms: int)`. `task = frontier.take(now_ms)`. If None, return False.
2. `domain = normalizer.domain_of(url)`. `rules = robots.get(domain)`. If the site asks for a delay, `frontier.set_domain_delay(domain, delay)`.
3. If `rules.allows(normalizer.path_of(url))` is False, skip to the last step. *Why:* the site's rules are followed before any request is made.
4. `result = fetcher.fetch(url, fetch_timeout_ms)`.
5. If `OK`: build a `Page` and call each processor's `process`, guarding each so one failure doesn't stop the others. Then, if `policy.allows_following(depth)`, call `enqueue(link, depth + 1)` for each link from `extractor.extract`.
6. If `ERROR` or `TIMEOUT`: `delay = retry.next_delay(attempts + 1)`. If there is a delay, add a new task with one more attempt and `not_before_ms = now_ms + delay` to the frontier. *Why straight to the frontier and not through `enqueue`:* the URL is already in `seen`, and `enqueue` would refuse it. If there is no delay, add the URL to `failed`.
7. If `NOT_FOUND`: add it to `failed`. It is not retried.
8. `frontier.done()`. Return True.
- **Failure and undo:** nothing to undo. A failed fetch changes only the frontier (a retry) or the failed list.

### Detect the end
1. `is_finished()` is True when every queue is empty and no task is in flight. A worker that is processing a page still counts, because that page may add new links.

### Concurrency (requirements 5 and 6)
- **Seen set:** `seen` is a thread-safe set, and `add_if_absent` claims a URL in one step. Two workers finding the same link at once can't both queue it.
- **Frontier:** the lock sits inside `Frontier`, around every method. `take` reserves the site's next allowed time while holding the lock, so two workers can't hit the same site too soon.
- **Robots cache:** the map is thread-safe, and `put_if_absent` keeps one set of rules per site even if two workers fetch at once.
- **Processors:** each processor guards its own data with its own lock, because pages arrive from many threads.
- **Crawler:** the lock sits inside `Crawler`, around `failed` only.
- **No lock while fetching:** the network call happens with no lock held, so a slow site never blocks other workers.
- **Lock order:** no method ever holds two locks at once, so no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new use of pages (search index, link graph, save to disk) | A new class implementing `ContentProcessor`, added to the list | No. |
| A different fetcher (a headless browser) | A new class implementing `PageFetcher` | No. |
| Different depth, domains, delays or retries | Different numbers and sets in the policies and the frontier | No. It is data. |
| More threads | A different `size` for the pool | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **A URL is cleaned and claimed in `seen` before it is queued, not when it is fetched.**
   - *Why:* a URL can't be queued twice, so the queue never holds duplicates.
   - *Rejected:* checking when fetching. The same URL could sit in the queue many times.
2. **The frontier groups tasks by site and reserves each site's next time when it hands out a task.**
   - *Why:* politeness is decided in one locked place, and many workers can't hit one site at once.
   - *Rejected:* each worker sleeping before a request, which can still hit a site twice when two workers wake together.
3. **A site's rules are fetched once and cached.**
   - *Why:* a site is asked for `robots.txt` once, not before every page.
   - *Rejected:* fetching the rules before each page.
4. **Processors are passed in as a list.**
   - *Why:* new uses of pages are new classes, and the crawler never changes.
   - *Rejected:* indexing or saving code inside the crawler.
5. **The end of the crawl counts tasks in flight.**
   - *Why:* empty queues while a page is being processed are not the end, because that page may add links.
   - *Rejected:* stopping as soon as the queue is empty.

### Edge cases handled
- Two spellings of one URL (`https://A.com/#top` and `https://a.com/`) count as one.
- A link back to a page already seen is ignored.
- A disallowed path is never fetched.
- A domain not on the list is never queued.
- A page at the depth limit is fetched but its links are not followed.
- A slow page times out and is retried with a growing wait, and given up after the last try.
- A missing page is not retried.
- A processor that fails doesn't stop the other processors or the crawl.
- A site's `robots.txt` that can't be fetched means "allow everything".

### Likely follow-up questions
1. **"How would you crawl billions of pages?"** Split the work across machines by domain, so each site is handled by one machine and politeness stays local. The seen set moves to a shared store or a Bloom filter, and the frontier to a durable queue.
2. **"Why not one queue for all sites?"** A big site at the front would block every worker on that site's delay. Grouping by site lets workers move on to sites that are ready.
3. **"How would you avoid crawler traps (endless calendar pages)?"** The depth limit already caps them. More defences are a limit on pages per site and on URL length, both new rules in `enqueue`.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Crawling across many machines | See follow-up question 1. |
| Saving the crawl to resume after a crash | The frontier and the seen set are in memory. Durable versions would replace them. |
| Pages built by scripts | A browser-based `PageFetcher`. |
| Recrawling pages that change | Not required. It would need a time per URL in `seen`. |
| Robots rules for named crawlers, and rules that expire | The rules for all crawlers are read once. |
| Limits on pages per site | See follow-up question 3. |

## 13. Requirements Met
1. Start from seeds and fetch: `Crawler.add_seeds` and `process_next`, with `PageFetcher`.
2. Extract and queue links: `LinkExtractor.extract` and `Crawler.enqueue`.
3. Never the same URL twice: `UrlNormalizer.normalize` and the thread-safe `seen` set, checked in `enqueue`.
4. Limit depth and domains: `CrawlPolicy.allows_following` and `allows_domain`.
5. Many threads: `CrawlerWorkerPool` and the locks in section 9.
6. Politeness and site rules: `Frontier.take` with `next_allowed_ms`, `RobotsCache` and `RobotsRules`.
7. Easy to add processing: the `ContentProcessor` interface.
8. Retries and time limits: `RetryPolicy`, `CrawlTask.attempts` and `not_before_ms`, and `fetch_timeout_ms`.

## 14. Requirement Fit Check

**Setup used in every example:** the seed is `https://a.com/`. A test fetcher serves: `a.com/` links to `/x`, `/y`, `https://b.com/` and `https://evil.com/`. `a.com/x` links to `/` and `/z`. `a.com/robots.txt` disallows `/private` and asks for a delay of 2000 ms. The default delay is 1000 ms. The policy allows depth 2 and the domains `a.com` and `b.com`. Retries are 3 attempts with a base delay of 500 ms. The fetch time limit is 5000 ms. Time starts at 0.

1. **Start from seeds and fetch**
   - *Success:* `add_seeds(["https://a.com/"])` returns 1, and the first step fetches `https://a.com/`.
   - *Failure:* `add_seeds(["ftp://a.com/"])` returns 0 because it is not an http URL.
2. **Extract and queue links**
   - *Success:* after `a.com/` is fetched, `/x`, `/y` and `https://b.com/` are queued at depth 1.
   - *Failure:* `https://evil.com/` is not queued because its domain is not allowed.
3. **Never the same URL twice**
   - *Success:* `a.com/x` links back to `/`, which is already in `seen`, so it is not queued again.
   - *Failure to avoid:* `add_seeds(["https://a.com", "https://A.com/#top"])` accepts only 1, because both clean to `https://a.com/`.
4. **Limit depth and domains**
   - *Success:* `a.com/z` is reached at depth 2 and fetched, but its links are not followed because 2 is not below the limit of 2.
   - *Failure:* nothing from `evil.com` is ever fetched.
5. **Many threads**
   - *Success:* with a pool of 4, one worker fetches from `a.com` while another fetches `https://b.com/` at the same moment.
   - *Failure to avoid:* two workers finding `a.com/y` at once can't both queue it, because only one `add_if_absent` succeeds.
6. **Politeness and site rules**
   - *Success:* `a.com/` is taken at time 0, and its rules set the delay to 2000 ms, so the next `a.com` page can't be taken before time 2000. Meanwhile `b.com` pages can be taken.
   - *Failure:* a link to `https://a.com/private/report` is queued but never fetched, because the rules disallow `/private`.
7. **Easy to add processing**
   - *Success:* with `ArchiveProcessor` and `WordCountProcessor` in the list, every fetched page is stored and counted. A new `LinkGraphProcessor` class added to the list starts working with no other change.
   - *Failure:* if `WordCountProcessor` fails on a page, `ArchiveProcessor` still stores it and the crawl continues.
8. **Retries and time limits**
   - *Success:* `https://b.com/` times out on its first try at time 0, so a new task with attempt 1 and `not_before_ms` 500 is added. The second try succeeds.
   - *Failure:* a page that times out three times is given up after the third try and appears in `get_failed()`. A page that returns `NOT_FOUND` is put in `get_failed()` at once, with no retry.
