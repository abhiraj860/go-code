# ATM Machine System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** the bank owns the money and the card's PIN attempts, the cash dispenser owns the notes, and the ATM only asks a small state object what each button does right now. A withdrawal always checks first and changes last: the machine's cash is checked without changing it, then the account is debited, then the notes are taken, with an undo if the last step fails.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The user should insert a card and enter a PIN to be authenticated.
2. The machine should behave differently depending on whether it is idle, has a card inserted, has an authenticated user or is dispensing cash.
3. The user should be able to check the balance, withdraw cash and deposit cash.
4. A withdrawal should succeed only if the account has enough balance and the machine has enough cash.
5. Cash should be dispensed by passing the amount through note denominations, largest first.
6. The card should be blocked after repeated wrong PIN entries.
7. The user should be able to cancel at any step and get the card back.
8. Every transaction should be recorded and a receipt provided.

**Types and conventions**
- Money and note counts are `int`, in whole currency units. Times are `datetime`, passed in by the caller, so the design never reads a clock.
- `map{int: int}` for notes always means note value to count. `map{key type: value type}` and `list[type]` always say what the key and value mean.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- No locks are used, because no requirement mentions concurrent use. One person uses a machine at a time.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Card | Checks a PIN and counts wrong attempts | PIN hash, wrong-attempt count |
| Account | Holds a balance and guards changes to it | balance |
| BankService | Creates accounts and cards, links them, and authenticates | the accounts, cards and links |
| NoteDispenser | Is one link in the chain, for one note value | its note count |
| CashDispenser | Builds the chain and plans and takes cash | the chain, the dispensers |
| Transaction | Records one operation and prints a receipt | an unchangeable record |
| TransactionLog | Keeps every transaction | the history |
| Session | Holds the card in the machine and the requested withdrawal | card number, requested amount |
| Response | Reports the result of a user action | an unchangeable record |
| ATMState (interface) | Lists every action a state must answer | none |
| IdleState | Behavior with no card | none |
| HasCardState | Behavior with a card, before the PIN | none |
| AuthenticatedState | Behavior after a correct PIN | none |
| DispensingState | Runs a withdrawal | none |
| ATM | Hands each action to the current state and applies the result | which state it is in |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Machine, card, PIN, user, state, balance, account, cash, note denomination, wrong PIN entry, transaction, receipt.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Card, Account, BankService, the dispensers, Transaction and ATM are classes. PIN and balance are attributes. A receipt is a printout of a transaction, not a separate fact.
2. **Find fixed sets of values.** The operations, the four conditions of the machine, and the result of a PIN check.
   - *Why:* a fixed set is an enum, so a typo like "withdrwa" cannot exist.
   - *Result:* `OperationType`, `ATMStateType`, `AuthResult`.
3. **Give every fact one owner.** The balance belongs to Account, the wrong-attempt count to Card, the notes to the dispensers, the history to the log.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** "Is the card blocked" is `wrong attempts >= allowed`. A receipt is printed from a transaction. The notes to hand out are planned, not stored.
   - *Why:* a stored "blocked" flag beside the count would go wrong the moment one is updated and the other is not.
5. **Give every verb to the class whose data changes.** Debiting changes Account. Counting a wrong PIN changes Card. Taking notes changes the dispensers.
6. **Find the action that touches several classes.** A withdrawal touches the dispensers, the account and the log, and the order matters. That is `DispensingState`.
7. **Find the hint phrases for patterns.** "Behave differently depending on whether it is idle, has a card, has an authenticated user or is dispensing" is State. "Passing the amount through note denominations, largest first" is Chain of Responsibility.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Fix the outputs.** A `Response` for every user action, and a `Transaction` inside it when money or a balance was involved.

## 4. Pattern Guide

### Used: State
- **Meaning:** each condition of the machine is a small class that decides what an action does, and the machine just asks the current one.
- **Hint phrase:** "behave differently depending on whether it is idle, has a card inserted, has an authenticated user or is dispensing cash".
- **Without it:** every action would have an `if` on the condition, and a new condition would mean editing all of them.
- **When NOT to use it:** when there are two simple conditions with one differing behavior. A boolean is enough then.

### Used: Chain of Responsibility
- **Meaning:** a request is passed along a line of handlers, and each handler takes what it can and passes the rest on.
- **Hint phrase:** "passing the amount through note denominations, largest first".
- **Without it:** one method would loop over a sorted list of notes with the logic written once, which also works, but the requirement asks for a chain and a chain makes each denomination its own link.
- **When NOT to use it:** when one place can decide everything with a single lookup.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| Strategy | "different ways to calculate or choose" | Not used. There is one way to authenticate and one way to dispense. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. |
| Facade | "a single simple entry point" | Not added as a separate class. `ATM` is already the single entry point because it is the manager. |
| Singleton | "exactly one instance" | Not used. A passed-in machine is easier to test. |

## 5. Enums

### OperationType
- Values: `CHECK_BALANCE`, `WITHDRAW`, `DEPOSIT`
- *Why:* requirement 3 names exactly three operations.

### ATMStateType
- Values: `IDLE`, `HAS_CARD`, `AUTHENTICATED`, `DISPENSING`
- *Why:* the four conditions of requirement 2. A state class returns one of these labels to say where the machine goes next, so states never hold each other and cannot form a circle.
- *Extensibility:* a new condition is one new value here, plus a new state class and a map entry.

### AuthResult
- Values: `SUCCESS`, `WRONG_PIN`, `BLOCKED`
- *Why:* requirement 6 needs the machine to tell a wrong PIN from a blocked card, because a blocked card is taken out of the machine.

## 6. Classes (smallest dependency first)

### Card
**Why this class exists:** requirement 6 needs one owner for "how many wrong PINs so far". The card owns it, so the count follows the card between visits.
- Attributes:
  - `card_number: str`. *Why:* the identity of the card.
  - `pin_hash: str`. *Why:* a hash of the PIN, never the PIN itself.
  - `max_attempts: int`. *Why:* how many wrong entries block the card. It is data, so it can differ per card or bank.
  - `failed_attempts: int`. *Why:* the single fact. "Blocked" is computed from it.
- Constructor: `Card(card_number: str, pin: str, max_attempts: int)`. *Why:* it stores only the hash of the PIN. It refuses an empty number, a PIN that is not digits, or `max_attempts < 1`. The count starts at 0.
- Methods:
  - `get_card_number() -> str`. *Why:* read-only view.
  - `is_blocked() -> bool`. *Why:* computed as `failed_attempts >= max_attempts`, so it can't disagree with the count.
  - `try_pin(pin: str) -> AuthResult`. *Why:* the only way to check a PIN, so every wrong entry is counted. If the card is already blocked it returns `BLOCKED` without checking. A correct PIN resets the count to 0 and returns `SUCCESS`. A wrong PIN adds 1 and returns `BLOCKED` if that blocks the card, else `WRONG_PIN`.
- *Extensibility:* none needed.

### Account
**Why this class exists:** requirement 4 needs one owner for the balance, so "enough money" is decided where the money is.
- Attributes: `account_number: str`, `balance: int`. *Why:* the identity, and the single fact about the money.
- Constructor: `Account(account_number: str, initial_balance: int)`. *Why:* it refuses an empty number or a negative balance.
- Methods:
  - `get_balance() -> int`. *Why:* read-only view.
  - `deposit(amount: int) -> bool`. *Why:* False if `amount <= 0`.
  - `withdraw(amount: int) -> bool`. *Why:* False if `amount <= 0` or the balance is lower. The check and the subtraction are one step, so the balance can never go negative. There is no setter.
- *Extensibility:* none needed.

### BankService
**Why this class exists:** a card is linked to an account, and that link, and the PIN check, belong to the bank. It is separate from the ATM because the bank is not part of the machine.
- Attributes:
  - `accounts: map{str: Account}` (account number to the Account). *Why:* the one record of accounts.
  - `cards: map{str: Card}` (card number to the Card). *Why:* the one record of cards.
  - `card_account: map{str: str}` (card number to the account number it draws on). *Why:* the one place the link is stored, so neither Card nor Account holds a copy.
- Constructor: `BankService()`. *Why:* it starts with three empty maps.
- Methods:
  - `create_account(account_number: str, initial_balance: int) -> Account or None`. *Why:* None if the number exists or the values are invalid.
  - `create_card(card_number: str, pin: str, max_attempts: int, account_number: str) -> Card or None`. *Why:* None if the card exists or the account is unknown.
  - `is_card_usable(card_number: str) -> bool`. *Why:* True if the card exists and is not blocked. The machine checks it before accepting a card.
  - `authenticate(card_number: str, pin: str) -> AuthResult`. *Why:* looks up the card once and asks it to try the PIN. An unknown card gives `WRONG_PIN`.
  - `get_balance(card_number: str) -> int or None`. *Why:* None if the card or its account is unknown.
  - `deposit(card_number: str, amount: int) -> bool`, `withdraw(card_number: str, amount: int) -> bool`. *Why:* each follows the card to its account once and asks the account. False if anything is unknown or the account refuses.
- *Extensibility:* more accounts and cards are data.

### NoteDispenser
**Why this class exists:** requirement 5 passes the amount through the denominations, so each denomination is one link that takes what it can and passes the rest on.
- Attributes:
  - `value: int`. *Why:* the note value this link hands out.
  - `count: int`. *Why:* how many of these notes are loaded. It is the only copy.
  - `next_dispenser: NoteDispenser or None`. *Why:* the next, smaller link. None for the last.
- Constructor: `NoteDispenser(value: int, next_dispenser: NoteDispenser or None)`. *Why:* the chain is built from the smallest value up, so the next link already exists. It refuses `value <= 0`. The count starts at 0.
- Methods:
  - `get_value() -> int`, `get_count() -> int`. *Why:* read-only views.
  - `plan(amount: int) -> map{int: int} or None`. *Why:* a dry run that changes nothing. It takes `n = min(amount // value, count)` notes, and the remainder is `amount - n * value`. If the remainder is 0 it returns `{value: n}`. Otherwise it asks the next link for the rest and joins the answers. If there is no next link and a remainder is left, it returns None.
  - `remove(n: int) -> bool`. *Why:* False if `n <= 0` or `n > count`, so the count can't go negative.
  - `add(n: int) -> bool`. *Why:* False if `n <= 0`.
- *Extensibility:* none needed.

### CashDispenser
**Why this class exists:** the machine's cash is one fact with one owner, and the chain's order (largest first) is built in one place.
- Attributes:
  - `head: NoteDispenser`. *Why:* the first link, with the largest value, where a plan starts.
  - `dispensers: map{int: NoteDispenser}` (a note value to its link). *Why:* finds the right link by value for loading, taking and restoring.
- Constructor: `CashDispenser(denominations: list[int])`. *Why:* it sorts the values from largest to smallest and builds the chain from the smallest up. It refuses an empty list, a repeated value or a value of 0 or less.
- Methods:
  - `has_denomination(value: int) -> bool`. *Why:* the machine accepts only notes it has a link for.
  - `add_notes(value: int, count: int) -> bool`. *Why:* loads or restocks notes. False if the value has no link or the count is invalid.
  - `plan(amount: int) -> map{int: int} or None`. *Why:* a dry run through the chain from the head. None if `amount <= 0` or the chain can't pay it exactly.
  - `take(plan: map{int: int}) -> bool`. *Why:* all or nothing. It first checks every count is enough, and only then removes the notes. False changes nothing.
  - `restore(plan: map{int: int}) -> bool`. *Why:* the undo of `take`. It puts the notes back.
- *Extensibility:* a new denomination is one more number in the list.

### Transaction
**Why this class exists:** requirement 8 asks for every transaction to be recorded and a receipt given. A fixed record can't be altered afterwards, and the receipt is computed from it.
- Attributes: `transaction_id: int`, `card_number: str`, `operation: OperationType`, `amount: int`, `balance_after: int or None`, `success: bool`, `timestamp: datetime`. *Why each:* the identity, who, what, how much (for a balance check, the balance is shown in `balance_after` and `amount` is 0), the balance afterwards if known, whether it worked, and when.
- Constructor: `Transaction(transaction_id: int, card_number: str, operation: OperationType, amount: int, balance_after: int or None, success: bool, timestamp: datetime)`. *Why:* all fields set once.
- Methods: `get_transaction_id() -> int`, `get_operation() -> OperationType`, `get_amount() -> int`, `get_balance_after() -> int or None`, `is_success() -> bool`. *Why:* read-only views.
  - `to_receipt() -> str`. *Why:* computed text: the id, the time, the card number with all but the last four digits hidden, the operation, the amount, the balance, and OK or FAILED.
- *Extensibility:* none needed.

### TransactionLog
**Why this class exists:** the history is a fact with one owner, so no other class keeps a second list.
- Attributes: `transactions: list[Transaction]` (every transaction, oldest first), `next_id: int`. *Why:* the history, and the next number to give out so ids are unique and in order.
- Constructor: `TransactionLog()`. *Why:* it starts empty with `next_id` at 1.
- Methods:
  - `record(card_number: str, operation: OperationType, amount: int, balance_after: int or None, success: bool, timestamp: datetime) -> Transaction`. *Why:* the log builds the Transaction itself, so it controls the id. Failed attempts are recorded too, because requirement 8 says every transaction.
  - `get_all() -> list[Transaction]` (returns a copy). *Why:* read-only view.
- *Extensibility:* none needed.

### Session
**Why this class exists:** the card in the machine and the amount asked for belong to one visit. They are separate from the states so the states can share them.
- Attributes: `card_number: str or None`, `requested_amount: int`. *Why:* which card is in, and the withdrawal waiting to be dispensed. No card is None, and no request is 0.
- Constructor: `Session()`. *Why:* it starts with no card and no request, which is the idle condition.
- Methods:
  - `get_card_number() -> str or None`, `get_requested_amount() -> int`. *Why:* read-only views.
  - `start(card_number: str) -> bool`. *Why:* False if a card is already in.
  - `request_withdrawal(amount: int) -> bool`. *Why:* False if there is no card or `amount <= 0`.
  - `clear_request() -> None`. *Why:* resets the request to 0 after a withdrawal.
  - `clear() -> None`. *Why:* the card comes out, and the card and the request are both reset.
- *Why it is never exposed:* the machine has no getter for it. Only the states reach it.
- *Extensibility:* none needed.

### Response
**Why this class exists:** every user action needs the same answer shape, so the caller never has to guess what a None or a bool means.
- Attributes:
  - `accepted: bool`. *Why:* True if the action was done, False if refused.
  - `state_after: ATMStateType`. *Why:* where the machine goes next. On a refusal it is the state's own type, so the machine can always apply it. A blocked card shows as `IDLE`, because the card was taken out.
  - `transaction: Transaction or None`. *Why:* the transaction made, with its receipt. None when no money or balance was involved.
- Constructor: `Response(accepted: bool, state_after: ATMStateType, transaction: Transaction or None)`. *Why:* all fields set once.
- Methods: `is_accepted() -> bool`, `get_state_after() -> ATMStateType`, `get_transaction() -> Transaction or None`. *Why:* read-only views.
- *Extensibility:* a new field is added here and old callers keep working.

### ATMState (interface)
**Why it is an interface:** requirement 2 gives four conditions, so four classes implement the same set of actions. The machine depends on this interface and not on any one state.
- Methods (every one returns a Response):
  - `insert_card(card_number: str)`, `enter_pin(pin: str)`
  - `select_operation(operation: OperationType, amount: int, notes: map{int: int}, now: datetime)`. *Why both `amount` and `notes`:* a withdrawal uses the amount and a deposit uses the notes. The other is ignored.
  - `dispense(now: datetime)`, `cancel()`
- *Why every state has every method:* a state that does not allow an action refuses it with `accepted = False` and its own type, so the machine never needs an `if` per state.
- *Extensibility:* a new state implements these five methods.

### IdleState (implements ATMState)
**Why this class exists:** it defines "no card in the machine".
- Attributes: `session: Session`, `bank_service: BankService`. *Why:* it starts a visit, and asks the bank if the card may be used.
- Constructor: `IdleState(session: Session, bank_service: BankService)`.
- Behavior:
  - `insert_card`: if `bank_service.is_card_usable(card_number)` is False (unknown or blocked), refuse and stay `IDLE`. Otherwise `session.start(card_number)`, accept, go to `HAS_CARD`.
  - `enter_pin`, `select_operation`, `dispense`, `cancel`: refuse and stay `IDLE`. *Why `cancel` is refused:* there is no card to return.
- *Extensibility:* none needed.

### HasCardState (implements ATMState)
**Why this class exists:** it defines "a card is in, but the PIN is not yet proved".
- Attributes: `session: Session`, `bank_service: BankService`.
- Constructor: `HasCardState(session: Session, bank_service: BankService)`.
- Behavior:
  - `enter_pin(pin)`: `result = bank_service.authenticate(session card number, pin)`.
    - `SUCCESS`: accept, go to `AUTHENTICATED`.
    - `WRONG_PIN`: refuse, stay `HAS_CARD`.
    - `BLOCKED`: `session.clear()` (the card comes out), refuse, go to `IDLE`.
  - `cancel`: `session.clear()`, accept, go to `IDLE`. This is requirement 7.
  - `insert_card`, `select_operation`, `dispense`: refuse and stay `HAS_CARD`.
- *Extensibility:* none needed.

### AuthenticatedState (implements ATMState)
**Why this class exists:** it defines "the user is proved and can choose an operation".
- Attributes: `session: Session`, `bank_service: BankService`, `cash_dispenser: CashDispenser`, `log: TransactionLog`. *Why each:* the visit, the balance and deposit, the cash taken in by a deposit, and the record.
- Constructor: `AuthenticatedState(session: Session, bank_service: BankService, cash_dispenser: CashDispenser, log: TransactionLog)`.
- Behavior:
  - `select_operation`:
    - `CHECK_BALANCE`: `balance = bank_service.get_balance(card)`. Record a transaction with `amount` 0 and `balance_after` set, accept, stay `AUTHENTICATED`.
    - `DEPOSIT`: total is the sum of `value * count` over `notes`. Refuse if any note value fails `cash_dispenser.has_denomination`, or the total is 0 or less. Otherwise `bank_service.deposit(card, total)`. If True, call `cash_dispenser.add_notes` for each note value, record a success, accept. If False, record a failure and refuse. *Why the notes go into the machine:* deposited cash is then available for later withdrawals.
    - `WITHDRAW`: `session.request_withdrawal(amount)`. If False, refuse. Otherwise accept and go to `DISPENSING`.
  - `cancel`: `session.clear()`, accept, go to `IDLE`.
  - `insert_card`, `enter_pin`, `dispense`: refuse and stay `AUTHENTICATED`.
- *Extensibility:* a new operation is a new branch here. That is a real edit, noted in section 10.

### DispensingState (implements ATMState)
**Why this class exists:** a withdrawal is the one multi-step action that touches money and cash together, and requirement 2 names the condition. While it runs, it refuses everything else.
- Attributes: `session: Session`, `bank_service: BankService`, `cash_dispenser: CashDispenser`, `log: TransactionLog`.
- Constructor: `DispensingState(session: Session, bank_service: BankService, cash_dispenser: CashDispenser, log: TransactionLog)`.
- Behavior:
  - `insert_card`, `enter_pin`, `select_operation`, `cancel`: refuse and stay `DISPENSING`. *Why `cancel` is refused:* the cash is already being handed out.
  - `dispense(now)`: see the withdrawal steps below.
- **Withdrawal steps, in this order (checks first, changes last):**
  1. `amount = session.get_requested_amount()`, `card = session.get_card_number()`.
  2. `plan = cash_dispenser.plan(amount)`. If None (the machine can't pay it), go to failure.
  3. `bank_service.withdraw(card, amount)`. If False (not enough balance), go to failure.
  4. `cash_dispenser.take(plan)`. If False (it should not happen), undo with `bank_service.deposit(card, amount)` and go to failure.
  5. Record a success with the balance after, `session.clear_request()`, and return `Response(True, AUTHENTICATED, transaction)`.
  - *Failure:* record a failed transaction, `session.clear_request()`, and return `Response(False, AUTHENTICATED, transaction)`.
- *Why this order:* the machine's cash is checked without changing anything, then the account (which can fail) is debited, and only then are notes taken. The one step after the debit that could fail is undone by paying the money back.
- *Extensibility:* none needed.

### ATM (the manager and only entry point)
**Why this class exists:** it is the one place that hands each action to the current state, applies the answer, and runs the follow-up dispense. It holds only the one fact that changes between actions: which state it is in.
- Attributes:
  - `states: map{ATMStateType: ATMState}` (state type to the state object for that type). *Why:* the machine looks up the real state from the label a state returns.
  - `current_type: ATMStateType`. *Why:* the single record of which state the machine is in.
- Constructor: `ATM(states: map{ATMStateType: ATMState})`. *Why:* the states are passed in, so a new state needs no change here and tests can use fakes. It starts at `IDLE` and refuses a map with no entry for each of the four types.
- Methods:
  - `insert_card(card_number: str) -> Response`, `enter_pin(pin: str) -> Response`, `cancel() -> Response`. *Why:* each hands the action to the current state and applies the answer.
  - `select_operation(operation: OperationType, amount: int, notes: map{int: int}, now: datetime) -> Response`. *Why:* the same, and if the answer says `DISPENSING`, it then runs the dispense.
  - `get_state() -> ATMStateType`. *Why:* read-only view.
- Private helper: `finish(response: Response, now: datetime) -> Response`. It sets `current_type` to the response's `state_after`. If that is `DISPENSING`, it calls the current state's `dispense(now)`, applies that response the same way, and returns it. Otherwise it returns the response unchanged. *Why hidden:* no outside caller should be able to set the state or trigger a dispense directly. *Why the dispense follow-up lives here:* the state only says "go to dispensing", and the machine runs the next step, so no state needs to know about another.
- *Extensibility:* a new state needs no change here.

## 7. Relationships
- BankService **has** Account, Card (created inside `create_account` and `create_card`).
- CashDispenser **has** NoteDispenser (the chain is built inside the constructor).
- NoteDispenser **uses** NoteDispenser (the next link).
- TransactionLog **has** Transaction (created inside `record`).
- ATM **uses** ATMState (the states passed in).
- IdleState **uses** Session, BankService.
- HasCardState **uses** Session, BankService.
- AuthenticatedState **uses** Session, BankService, CashDispenser, TransactionLog.
- DispensingState **uses** Session, BankService, CashDispenser, TransactionLog.
- IdleState, HasCardState, AuthenticatedState, DispensingState **implement** ATMState.
- Enums are always **uses**: Transaction to OperationType, Response to ATMStateType, ATM to ATMStateType.
- `Response` and `AuthResult` appear only as return types, so no attribute holds them and they have no line.

## 8. Interaction

### Layers
1. **Entry point and manager:** ATM. The only class users call.
2. **Behavior:** the four states.
3. **Data owners that guard themselves:** Card, Account, the dispensers, Session.
4. **Bank and record keeping:** BankService, TransactionLog.
5. **Plain data:** Transaction, Response.

### Call chains
- **Insert a card and enter the PIN**
  1. `User -> ATM.insert_card(card_number)`, then `ATM -> current state.insert_card(...)`.
  2. `IdleState -> BankService.is_card_usable(card_number)`: blocked or unknown cards never get in.
  3. `User -> ATM.enter_pin(pin)`, then `HasCardState -> BankService.authenticate(card, pin)`.
  4. `BankService -> Card.try_pin(pin)`: the card counts the wrong entries.
- **Check the balance or deposit**
  1. `User -> ATM.select_operation(...)`, then `AuthenticatedState -> BankService.get_balance` or `deposit`.
  2. `AuthenticatedState -> CashDispenser.add_notes` for a deposit, then `TransactionLog.record`.
- **Withdraw**
  1. `User -> ATM.select_operation(WITHDRAW, amount, ...)`, then `AuthenticatedState -> Session.request_withdrawal(amount)`.
  2. `ATM.finish -> DispensingState.dispense(now)`: the machine runs the next step.
  3. `DispensingState -> CashDispenser.plan(amount)`, which goes `NoteDispenser.plan` down the chain: the machine's cash is checked without changing it.
  4. `DispensingState -> BankService.withdraw(card, amount)`, then `CashDispenser.take(plan)`, then `TransactionLog.record`.
- **Cancel**
  1. `User -> ATM.cancel()`, then `current state.cancel()`, then `Session.clear()`: the card comes out.

### Who may call whom
- Users call `ATM` only, and never a state, the session or the dispensers.
- `ATM` calls the current state only.
- States call `Session`, `BankService`, `CashDispenser` and `TransactionLog`.
- `BankService` calls `Card` and `Account`. `CashDispenser` calls `NoteDispenser`.
- States must never call `ATM` or another state. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Insert a card and authenticate
1. `insert_card(card_number: str)`. If the card is unknown or blocked, refuse. Otherwise the session starts and the state is `HAS_CARD`.
2. `enter_pin(pin: str)`: `authenticate` returns `SUCCESS`, `WRONG_PIN` or `BLOCKED`. The state becomes `AUTHENTICATED`, stays `HAS_CARD`, or goes to `IDLE` with the card taken out.
- **Failure and undo:** a wrong PIN changes only the card's count, and a correct PIN resets it. A blocked card is ejected, and can't be inserted again.

### Check the balance
1. `select_operation(CHECK_BALANCE, 0, empty, now)`. The balance is read and a transaction recorded with a receipt.
- **Failure:** an unknown account refuses. Nothing changes.

### Deposit
1. `select_operation(DEPOSIT, 0, notes, now)`. Every note value must be a denomination the machine has, and the total must be above 0.
2. `bank_service.deposit(card, total)`, then `add_notes` for each note value, then record a success.
- **Failure and undo:** the validation happens before any change. If the bank refuses the deposit, no notes are added, and a failed transaction is recorded.

### Withdraw
1. `select_operation(WITHDRAW, amount, empty, now)` stores the request and moves to `DISPENSING`. A request of 0 or less is refused first.
2. `dispense(now)` runs the five steps in section 6, then the state returns to `AUTHENTICATED`.
- **Failure and undo:**
  - The machine can't pay the amount: nothing has changed, so there is nothing to undo.
  - The balance is too low: nothing was taken from the machine, so there is nothing to undo.
  - Taking the notes fails: the account was already debited, so the amount is paid back with a deposit.
  - In every failure a failed transaction is recorded, and the user stays authenticated.

### Cancel
1. `cancel()` in `HAS_CARD` or `AUTHENTICATED` clears the session, so the card comes out, and the state is `IDLE`.
- **Failure:** in `IDLE` there is no card, and in `DISPENSING` the cash is being handed out, so it is refused.

### Concurrency
- No requirement mentions concurrent use, so no lock is added.
- If it were needed later, a lock inside `Account` around `deposit` and `withdraw`, and one around `CashDispenser.take`, would protect the two shared facts.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new note denomination (200) | One more number in the list passed to `CashDispenser`, then `add_notes` | No. It is data. |
| A different number of allowed wrong PINs | A different `max_attempts` in `create_card` | No. It is data. |
| More accounts and cards | Calls to `create_account` and `create_card` | No. |
| A new operation (transfer) | A new `OperationType` value and a new branch in `AuthenticatedState.select_operation` | Yes, one branch. That is where operations live. |
| A new state, such as OUT_OF_SERVICE | A new state class, one new enum value, one new map entry | The one state that moves into it must return the new type, so that return value changes. |

## 11. Interview Notes

### Key decisions
1. **A withdrawal checks first and changes last.**
   - *Why:* the dry-run plan changes nothing, the account debit is the step that can fail, and taking the notes comes after, so the only undo needed is paying one amount back.
   - *Rejected:* taking the notes first, which would need a restock when the account refuses.
2. **The card owns the wrong-PIN count, and "blocked" is computed.**
   - *Why:* one fact, and the count follows the card between visits.
   - *Rejected:* a stored `blocked` flag beside the count.
3. **States return a `Response`, and the machine applies it.**
   - *Why:* no state holds another state, and nothing exposes a public state setter.
   - *Rejected:* states calling a `change_state` method on the machine, which anyone could call.
4. **The link from a card to an account is stored once, in `BankService`.**
   - *Why:* one fact in one place.
   - *Rejected:* the card holding its account and the account holding its cards.
5. **Failed transactions are recorded too.**
   - *Why:* requirement 8 says every transaction.
   - *Rejected:* recording only successes, which hides refused withdrawals.

### Edge cases handled
- A wrong PIN is counted, and a correct PIN resets the count.
- A blocked card is ejected, and a blocked or unknown card is refused at insertion.
- Cancel works in the card and authenticated states, and is refused when idle or dispensing.
- A withdrawal of 0 or less is refused.
- The account has too little money: refused, with the machine's cash untouched.
- The machine has too little cash, or can't make the amount exactly: refused, with the account untouched.
- A deposit with an unknown note value is refused.
- A failed `take` after the debit pays the money back.

### Likely follow-up questions
1. **"Doesn't largest-first fail for some amounts?"** Yes. With notes 100, 50 and 20, an amount of 60 is payable as 20+20+20, but largest-first takes one 50 and has 10 left, so the plan is None and the withdrawal is refused. The requirement asks for largest first. A fix is to search the combinations (backtracking or dynamic programming) inside `CashDispenser.plan`, which changes only that method.
2. **"What if two ATMs use the same account?"** Put a lock inside `Account` around `deposit` and `withdraw`, so the check and the change are one step.
3. **"Why is Dispensing a state if it is instant?"** It names the condition from requirement 2 and refuses every action, including cancel, while the cash is handed out.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Amounts that largest-first can't make | They are refused. See follow-up question 1. |
| Keeping or swallowing a blocked card | The card comes out and can't be inserted again. |
| A daily withdrawal limit | No requirement asks for it. It would be a rule in `DispensingState` before step 3. |
| Changing a PIN | Not required. It would be a method on `Card`. |
| Loading cash into the machine | The machine's cash is loaded at setup with `add_notes`, and by deposits. |
| Printing the receipt | `to_receipt()` gives the text. Printing is a hardware concern. |
| Transfers between accounts | See the new-operation row in section 10. |
| Concurrent users of one account | See follow-up question 2. |

## 13. Requirements Met
1. Insert a card and enter a PIN: `IdleState.insert_card`, `HasCardState.enter_pin`, `BankService.authenticate`.
2. Behavior by condition: the `ATMState` interface with its four implementations, applied by `ATM`.
3. Balance, withdraw and deposit: `AuthenticatedState.select_operation` and `DispensingState.dispense`.
4. Enough balance and enough cash: `Account.withdraw` and `CashDispenser.plan`, checked in order in `DispensingState.dispense`.
5. Largest first through the denominations: `CashDispenser` and the chain of `NoteDispenser`.
6. Block after repeated wrong PINs: `Card.try_pin` and `Card.is_blocked`.
7. Cancel at any step and get the card back: `cancel` in the card and authenticated states, and `Session.clear`.
8. Record every transaction and give a receipt: `TransactionLog.record` and `Transaction.to_receipt`.

## 14. Requirement Fit Check

**Setup used in every example:** card `C1` has number 1111222233334444, PIN 4321, 3 allowed attempts, and draws on account `ACC1` with balance 500. Card `C2` draws on `ACC2` with balance 5000. The `CashDispenser` has denominations 100, 50 and 20, loaded with 100 × 5, 50 × 5 and 20 × 5, so 1000 in total. The time is 2026-11-08 10:00.

1. **Insert a card and authenticate**
   - *Success:* `insert_card("1111222233334444")` moves `IDLE` to `HAS_CARD`. `enter_pin("4321")` returns `SUCCESS` and the state is `AUTHENTICATED`.
   - *Failure:* `insert_card("9999")` is refused because the card is unknown. `enter_pin("0000")` returns `WRONG_PIN`, and the state stays `HAS_CARD`.
2. **Behavior by condition**
   - *Success:* in `IDLE`, `insert_card` is accepted. In `HAS_CARD`, `enter_pin` is accepted. In `AUTHENTICATED`, `select_operation` is accepted.
   - *Failure:* in `IDLE`, `enter_pin` and `select_operation` are refused. In `HAS_CARD`, `select_operation` is refused. In `DISPENSING`, every action is refused.
3. **Check balance, withdraw, deposit**
   - *Success:* `CHECK_BALANCE` records a transaction showing 500. `DEPOSIT` of {100: 2} adds 200, so the balance is 700 and the machine holds 7 notes of 100. The withdrawal is in example 4.
   - *Failure:* a `DEPOSIT` of {10: 3} is refused because the machine has no link for 10. A `WITHDRAW` of −5 is refused and nothing is recorded.
4. **Enough balance and enough cash**
   - *Success:* with a balance of 700, `WITHDRAW` 270 plans {100: 2, 50: 1, 20: 1}, debits 270 (balance 430), and takes the notes.
   - *Failure (balance):* with a balance of 100, `WITHDRAW` 200 plans fine, but `Account.withdraw` refuses. No notes are taken, and a failed transaction is recorded.
   - *Failure (cash):* card C2 has 5000, but `WITHDRAW` 1500 needs more than the machine's 1000, so `plan` is None. The account is untouched.
5. **Largest first through the denominations**
   - *Success:* 270 goes through 100 (two notes, 70 left), then 50 (one note, 20 left), then 20 (one note). The plan is {100: 2, 50: 1, 20: 1}.
   - *Failure:* 30 goes through 100 (none), 50 (none), 20 (one note, 10 left), and the chain has no more links, so the plan is None and the withdrawal is refused. An amount of 60 is also refused, even though 20 + 20 + 20 would work, because largest-first takes a 50 first.
6. **Block after repeated wrong PINs**
   - *Success:* wrong PINs on C1 are counted: 1, 2, then the third makes `failed_attempts` equal 3, so `try_pin` returns `BLOCKED`. The card comes out and the state is `IDLE`.
   - *Failure:* `insert_card` with C1 again is refused because it is blocked. A correct PIN after one wrong entry resets the count to 0.
7. **Cancel at any step and get the card back**
   - *Success:* `cancel()` in `HAS_CARD` or in `AUTHENTICATED` clears the session, returns the card, and the state is `IDLE`.
   - *Failure:* `cancel()` in `IDLE` is refused because there is no card. `cancel()` in `DISPENSING` is refused.
8. **Record every transaction and give a receipt**
   - *Success:* after a balance check, a deposit and a withdrawal, the log holds three transactions with ids 1, 2, 3. The last receipt reads like `#3 2026-11-08 10:05 ************4444 WITHDRAW 270 balance 430 OK`.
   - *Failure to avoid:* the refused withdrawal from example 4 is still recorded, with `success` False, and its receipt says FAILED.
