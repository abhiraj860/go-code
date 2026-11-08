# Splitwise / Expense Sharing: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** the only facts stored are what actually happened, the expenses and the settlements. Every balance, "who owes whom", and "how to settle with fewest payments" is computed from those records on request, so no balance can ever disagree with the history. The way an expense is split is a small swappable object, so a new way of splitting is a new class.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The system should allow users to register and create groups with members.
2. The system should allow a user to add an expense that one person paid and several people share.
3. An expense should be splittable equally, by exact amounts or by percentages.
4. It should be easy to add new ways of splitting an expense.
5. The system should track how much each user owes or is owed by every other user.
6. Users should be able to view their balances with each person and within each group.
7. Users should be able to settle their dues.
8. The system should reduce the number of payments needed to clear all debts in a group.

**Types and conventions**
- Money is `int`, in the smallest currency unit (for example cents), so there are no rounding errors from decimals. Percentages are whole numbers that must add up to 100.
- A balance is signed. `a owes b: 80` means a owes b 80. A negative number means the other person owes a.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean. Maps keep the order they were given.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- Every expense belongs to a group. Two friends who share costs are simply a two-person group.
- There are no enums, because every condition in this design is a number or a record.
- No locks are used, because no requirement mentions concurrent use (see Concurrency in section 8).

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| User | Describes one registered person | id, name |
| Expense | Records one payment that was shared | payer, amount, each person's share |
| Transfer | Records money moved from one person to another | from, to, amount |
| SplitStrategy (interface) | Turns a total into each person's share | none |
| EqualSplit | Shares a total equally among chosen people | the chosen people |
| ExactSplit | Uses exact amounts per person | the amounts |
| PercentSplit | Uses a percentage per person | the percentages |
| DebtSimplifier | Turns net balances into the fewest payments | none |
| Group | Keeps the members and the expense and transfer history, and computes balances from it | members, expenses, transfers |
| SplitwiseService | Registers users, creates groups, and runs each action in the right order | users, groups |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** User, group, member, expense, payer, share, split, balance, settlement, payment.
   - *Why:* a noun with its own data becomes a class, and a noun that is just a number becomes an attribute.
   - *Result:* User, Group, Expense and Transfer are classes. Balance, share and amount are numbers. "Settlement" and "payment" are both a Transfer.
2. **Find fixed sets of values.** None. The three split types are behaviors, not labels.
   - *Why:* a fixed set of labels is an enum, but behavior that differs by type is a class.
3. **Give every fact one owner.** What was paid belongs to Expense, money moved to Transfer, and the history of both to the Group they happened in.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** Who owes whom, a user's net balance, and the fewest payments are all computed from the history.
   - *Why:* a stored running balance next to the history would be a second copy, and one missed update makes them disagree forever.
5. **Give every verb to the class whose data changes.** Adding an expense and settling change the Group's history, so Group owns them. Computing shares belongs to the strategy.
6. **Find the action that touches several classes.** Showing the fewest payments needs the Group's net balances and then the simplifier. That order lives in the service.
7. **Find the hint phrase for a pattern.** "It should be easy to add new ways of splitting an expense" is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Fix the outputs.** An Expense or None for an add, a bool for a settle, and balance maps for views.

## 4. Pattern Guide

### Used: Strategy
- **Meaning:** a way of calculating is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrase:** "it should be easy to add new ways of splitting an expense" (and "splittable equally, by exact amounts or by percentages").
- **Without it:** the group would have an `if` on the split type, and every new way of splitting would mean editing it.
- **When NOT to use it:** when there is one fixed calculation that nobody asked to extend.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. Nothing in this problem changes behavior with a state. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. The caller chooses and builds the split object itself. |
| Facade | "a single simple entry point" | Not added as a separate class. `SplitwiseService` is already the entry point because it is the manager. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. A passed-in service is easier to test. |

## 5. Classes (smallest dependency first)

### User
**Why this class exists:** a registered person is a thing with its own identity. Groups and expenses refer to people by id, so the id must belong to a real record.
- Attributes:
  - `user_id: str`. *Why:* the identity used everywhere else.
  - `name: str`. *Why:* what other people see.
- Constructor: `User(user_id: str, name: str)`. *Why:* both fields are set once. It refuses an empty id or name.
- Methods: `get_user_id() -> str`, `get_name() -> str`. *Why:* read-only views. There are no setters.
- *Extensibility:* more fields (email, phone) are added here and nothing else changes.

### Expense
**Why this class exists:** requirement 2 is a record of one payment that several people share. It is a fixed record, so history can't be rewritten.
- Attributes:
  - `payer_id: str`. *Why:* who paid the money.
  - `amount: int`. *Why:* the total paid.
  - `shares: map{str: int}` (user id to the amount that user owes for this expense). *Why:* the split result for every participant. It does not record how it was calculated, because only the result matters afterwards.
- Constructor: `Expense(payer_id: str, amount: int, shares: map{str: int})`. *Why:* this is the last line of defense. It refuses `amount <= 0`, an empty `shares` map, a negative share, or shares that do not add up to exactly `amount`. So a buggy strategy can't create money.
- Methods:
  - `get_payer_id() -> str`, `get_amount() -> int`. *Why:* read-only views.
  - `get_shares() -> map{str: int}` (returns a copy). *Why a copy:* so a caller can't edit what was owed.
- *Why the payer need not be in the shares:* a person may pay for others and owe nothing.
- *Extensibility:* none needed.

### Transfer
**Why this class exists:** money moving from one person to another is the same thing whether it records a settlement or suggests a payment. One record for both avoids two near-identical classes.
- Attributes: `from_user_id: str`, `to_user_id: str`, `amount: int`. *Why:* who pays whom and how much.
- Constructor: `Transfer(from_user_id: str, to_user_id: str, amount: int)`. *Why:* all fields set once. It refuses `amount <= 0` or the same person on both sides.
- Methods: `get_from_user_id() -> str`, `get_to_user_id() -> str`, `get_amount() -> int`. *Why:* read-only views.
- *Extensibility:* a date or a note is added here and nothing else changes.

### SplitStrategy (interface)
**Why it is an interface:** requirement 4 asks for new ways of splitting, and requirement 3 already needs three, so the group depends on one method and not on any specific way of splitting.
- Methods: `calculate_shares(total: int) -> map{str: int} or None`.
  - *Why it takes only the total:* each strategy is built with its own inputs (who takes part, the exact amounts, the percentages), so the call looks the same for every one.
  - *Why it can return None:* the inputs may be invalid for this total, such as exact amounts that do not add up. None means refused with nothing changed.
- *Extensibility:* a new way of splitting is a new class implementing this method.

### EqualSplit (implements SplitStrategy)
**Why this class exists:** requirement 3's first way, an equal split.
- Attributes: `participant_ids: list[str]`. *Why:* who takes part, in order.
- Constructor: `EqualSplit(participant_ids: list[str])`. *Why:* set once. It refuses an empty list or a repeated id.
- Methods: `calculate_shares(total: int) -> map{str: int} or None`.
  - *Why:* each person gets `total // n`. The `total % n` leftover units go one each to the first people in the list. So the shares add up to the total exactly, and the result is the same every time.
  - It returns None if `total <= 0`.
- *Extensibility:* none needed.

### ExactSplit (implements SplitStrategy)
**Why this class exists:** requirement 3's second way, exact amounts.
- Attributes: `amounts: map{str: int}` (user id to the exact amount that person owes). *Why:* the amounts the user typed.
- Constructor: `ExactSplit(amounts: map{str: int})`. *Why:* set once. It refuses an empty map.
- Methods: `calculate_shares(total: int) -> map{str: int} or None`. *Why:* returns a copy of the amounts if they add up to `total` and none is negative. Otherwise None.
- *Extensibility:* none needed.

### PercentSplit (implements SplitStrategy)
**Why this class exists:** requirement 3's third way, percentages.
- Attributes: `percentages: map{str: int}` (user id to that person's percentage). *Why:* the percentages the user typed.
- Constructor: `PercentSplit(percentages: map{str: int})`. *Why:* set once. It refuses an empty map.
- Methods: `calculate_shares(total: int) -> map{str: int} or None`.
  - *Why:* None unless the percentages add up to 100 and none is negative. Each share is `total * percent // 100`. The leftover units go one each to the first people in the map, so the shares add up to the total exactly.
- *Extensibility:* none needed.

### DebtSimplifier
**Why this class exists:** requirement 8 is a calculation on balances with no data of its own. It stays separate from Group so the history and the algorithm can change independently.
- Attributes: none.
- Constructor: `DebtSimplifier()`. *Why:* it holds nothing, so it can be shared.
- Methods: `simplify(net_balances: map{str: int}) -> list[Transfer]`.
  - *Why it takes net balances:* a user's net balance (positive means owed money) is all it needs.
  - *How:* repeatedly pair the person owed the most with the person who owes the most, and make one Transfer for the smaller of the two amounts. Someone is cleared each time, so at most (people − 1) transfers result.
  - It returns an empty list if every balance is 0.
- *Extensibility:* a different method of simplifying would be a second class. See the follow-up in section 10.

### Group
**Why this class exists:** requirement 1 groups people, and requirements 5 and 6 need one place that knows the whole history of that group. It owns the members, the expenses and the transfers, and computes every balance from them.
- Attributes:
  - `group_id: str`, `name: str`. *Why:* the identity and the display name.
  - `member_ids: list[str]`. *Why:* the people in the group, by id. The service has already checked they are registered.
  - `expenses: list[Expense]`. *Why:* every shared payment, in order.
  - `transfers: list[Transfer]`. *Why:* every settlement, in order.
- Constructor: `Group(group_id: str, name: str, member_ids: list[str])`. *Why:* a group needs an id, a name and at least one member. It keeps its own copy of the list and starts with no history. It refuses empty values or a repeated member.
- Methods:
  - `get_group_id() -> str`, `get_name() -> str`. *Why:* read-only views.
  - `get_member_ids() -> list[str]` (returns a copy), `is_member(user_id: str) -> bool`. *Why:* read-only views.
  - `add_member(user_id: str) -> bool`. *Why:* False if already a member. A new member starts with a zero balance because they have no history.
  - `add_expense(payer_id: str, amount: int, strategy: SplitStrategy) -> Expense or None`. *Why:* the group checks the payer and every person in the shares are members, so no one outside the group can be charged. None on any refusal.
  - `settle(from_user_id: str, to_user_id: str, amount: int) -> bool`. *Why:* both must be members and `amount` must not exceed what `from` currently owes `to`. Paying more would create a new debt the other way, which is not a settlement. False on any refusal.
  - `get_balance_between(user_a: str, user_b: str) -> int or None`. *Why:* computed from the history. Positive means a owes b. None if either is not a member.
  - `get_balances_of(user_id: str) -> map{str: int} or None`. *Why:* computed. For every other member who is not at 0, the amount this user owes them (negative means they owe this user). None if the user is not a member.
  - `get_net_balances() -> map{str: int}`. *Why:* computed. For each member, what they paid minus what they owe, adjusted by transfers. Positive means they are owed money. The numbers always add up to 0.
- *How balances are computed:* for each expense, every person in the shares other than the payer owes the payer their share. For each transfer, `from` owes `to` less by the amount.
- *Extensibility:* the split strategy is passed in, so new ways of splitting need no change here.

### SplitwiseService (the manager and only entry point)
**Why this class exists:** registering, creating groups and showing the fewest payments each cross several classes or turn an id into an object. Callers use only ids, and this class looks them up once.
- Attributes:
  - `users: map{str: User}` (user id to the User). *Why:* the one record of who is registered.
  - `groups: map{str: Group}` (group id to the Group). *Why:* the one record of every group.
  - `simplifier: DebtSimplifier`. *Why:* the calculation used for the fewest payments, passed in.
- Constructor: `SplitwiseService(simplifier: DebtSimplifier)`. *Why:* the simplifier is passed in so tests can use a fake. It starts with no users or groups.
- Methods:
  - `register_user(user_id: str, name: str) -> User or None`. *Why:* None if the id is taken or the values are invalid.
  - `create_group(group_id: str, name: str, member_ids: list[str]) -> Group or None`. *Why:* None if the id is taken or any member is not a registered user. The service checks this because only it knows who is registered.
  - `add_member(group_id: str, user_id: str) -> bool`. *Why:* looks up the group once, checks the user is registered, then asks the group.
  - `add_expense(group_id: str, payer_id: str, amount: int, strategy: SplitStrategy) -> Expense or None`. *Why:* looks up the group once, then asks the group to add the expense.
  - `settle(group_id: str, from_user_id: str, to_user_id: str, amount: int) -> bool`. *Why:* looks up the group once, then asks the group.
  - `get_group_balances(group_id: str, user_id: str) -> map{str: int} or None`. *Why:* requirement 6, within one group.
  - `get_balances(user_id: str) -> map{str: int} or None`. *Why:* requirement 6, with each person across every group. It adds up the user's balances from each group they are in and leaves out zeros. None if the user is not registered.
  - `get_simplified_payments(group_id: str) -> list[Transfer] or None`. *Why:* requirement 8. It gets the group's net balances, then asks the simplifier. None if the group is unknown.
- *Extensibility:* the simplifier is passed in, so the service doesn't change if its internals do.

## 6. Relationships
- SplitwiseService **has** User, Group (created inside `register_user` and `create_group`).
- Group **has** Expense, Transfer (created inside `add_expense` and `settle`).
- SplitwiseService **uses** DebtSimplifier (passed in).
- EqualSplit, ExactSplit, PercentSplit **implement** SplitStrategy.
- There are no enums, so there are no enum lines.
- `SplitStrategy` and `Transfer` also appear as a method input or output on other classes, but no attribute holds them there, so they have no line from those classes.

## 7. Interaction

### Layers
1. **Entry point and manager:** SplitwiseService. The only class callers use.
2. **Domain class:** Group. Owns the history and computes balances.
3. **Rules passed in:** SplitStrategy with its three implementations, and DebtSimplifier.
4. **Plain records:** User, Expense, Transfer.

### Call chains
- **Register and create a group**
  1. `Caller -> SplitwiseService.register_user(user_id, name)`.
  2. `Caller -> SplitwiseService.create_group(group_id, name, member_ids)`: the service checks every member is registered before any Group is built.
- **Add an expense**
  1. `Caller -> SplitwiseService.add_expense(group_id, payer_id, amount, strategy)`.
  2. `SplitwiseService -> groups.get(group_id)`: the one lookup.
  3. `SplitwiseService -> Group.add_expense(payer_id, amount, strategy)`.
  4. `Group -> SplitStrategy.calculate_shares(amount)`: the strategy decides each share.
  5. `Group -> Expense(...)`: the record checks the shares add up, then the group stores it.
- **View balances**
  1. `Caller -> SplitwiseService.get_balances(user_id)`.
  2. `SplitwiseService -> Group.get_balances_of(user_id)` for each group the user is in, then it adds the results.
- **Settle**
  1. `Caller -> SplitwiseService.settle(group_id, from, to, amount)`.
  2. `SplitwiseService -> Group.settle(from, to, amount)`: the group checks the amount against its own history.
- **Fewest payments**
  1. `Caller -> SplitwiseService.get_simplified_payments(group_id)`.
  2. `SplitwiseService -> Group.get_net_balances()`, then `DebtSimplifier.simplify(net_balances)`.

### Who may call whom
- Callers call `SplitwiseService` only, and never `Group`, the strategies or the simplifier.
- `SplitwiseService` calls `Group` and `DebtSimplifier`.
- `Group` calls the strategy it is given, and builds `Expense` and `Transfer`.
- `Expense`, `Transfer`, `User`, the strategies and the simplifier call no other class.
- No class calls back up to its owner, so there are no cycles.

## 8. Flows

### Register a user
1. `register_user(user_id, name)`. If the id exists or a value is empty, return None. Otherwise store and return the User.
- **Failure:** nothing is stored.

### Create a group
1. `create_group(group_id: str, name: str, member_ids: list[str])`. If the id exists, return None.
2. Check every member id is in `users`. If any is not, return None. *Why before building:* a Group with an unknown member must never exist.
3. Build the Group and store it.
- **Failure:** nothing is created.

### Add an expense
1. `add_expense(group_id: str, payer_id: str, amount: int, strategy: SplitStrategy)`. `group: Group or None = groups.get(group_id)`. If None, return None.
2. In the group: if `amount <= 0` or the payer is not a member, return None.
3. `shares: map{str: int} or None = strategy.calculate_shares(amount)`. If None, return None. *Why:* the strategy refuses inputs that do not add up.
4. If any person in `shares` is not a member, return None. *Why:* nobody outside the group can be charged.
5. Build `Expense(payer_id, amount, shares)`. If it refuses (the shares do not add up), return None. *Why:* a faulty strategy can't create or lose money.
6. Append the expense and return it.
- **Failure and undo:** every check happens before the expense is appended, so a failure changes nothing and there is nothing to undo.

### View balances
1. `get_group_balances(group_id, user_id)` looks up the group once and calls `get_balances_of(user_id)`. None if the group or the membership is unknown.
2. `get_balances(user_id)` adds up `get_balances_of(user_id)` over every group the user is in, and drops zeros.
- **Failure:** an unknown user or group returns None. A user with nothing owed returns an empty map.

### Settle
1. `settle(group_id: str, from_user_id: str, to_user_id: str, amount: int)`. Look up the group once. If None, return False.
2. In the group: both must be members and different, `amount > 0`, and `amount <= get_balance_between(from, to)`. Otherwise return False.
3. Append `Transfer(from, to, amount)` and return True.
- **Failure:** nothing is appended. The check and the append happen in one call, so a refused settle can't half-apply.

### Fewest payments
1. `get_simplified_payments(group_id: str)`. Look up the group once. If None, return None.
2. `net: map{str: int} = group.get_net_balances()`.
3. Return `simplifier.simplify(net)`.
- **Failure:** an unknown group returns None. A fully settled group returns an empty list. Nothing is changed in either case, because the result is only a suggestion.

### Concurrency
- No requirement mentions concurrent use, so no lock is added.
- All of a group's data is changed only inside `Group` methods, and balances are computed from that data.
- If concurrency were required later, one lock inside each `Group` around `add_member`, `add_expense`, `settle` and the balance methods would be enough. The service map would need a thread-safe map.

## 9. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new way of splitting (by shares such as 2:1:1, or by adjustments) | A new class implementing `SplitStrategy`, passed to `add_expense` | No. |
| Equal split among only some people, or leaving the payer out | A different list passed to `EqualSplit` | No. It is data. |
| More users, groups and members | Calls to `register_user`, `create_group` and `add_member` | No. |
| More fields on a user (email, phone) | Add them to `User` | No other class changes. |

## 10. Interview Notes

### Key decisions
1. **Balances are computed from the history and never stored.**
   - *Why:* the expenses and transfers are the only facts, so a balance can't disagree with them.
   - *Rejected:* a running "who owes whom" map updated on every expense and settle. It is a second copy of the history, and one missed update makes it wrong forever.
2. **A split strategy returns shares, and `Expense` checks they add up.**
   - *Why:* the strategy is swappable, and the record is the last line of defense, so no strategy can create or lose money.
   - *Rejected:* an `if` on the split type inside the group.
3. **Money is a whole `int` and leftover units go to the first people in the list.**
   - *Why:* the shares always add up exactly, and the result is the same every time.
   - *Rejected:* decimals. 100 split three ways would give 33.33 three times and lose a cent.
4. **Every expense belongs to a group, and two friends are a two-person group.**
   - *Why:* one path for balances, settlements and simplifying.
   - *Rejected:* a separate expense path outside groups, which would double all the balance logic.
5. **A settlement can't exceed what is owed.**
   - *Why:* paying more would create a debt the other way, which is not settling.
   - *Rejected:* accepting any amount and letting the balance flip silently.

### Edge cases handled
- Exact amounts that do not add up to the total, or percentages that do not add up to 100, are refused.
- The payer is not a member, or a person in the split is not a member, so the expense is refused.
- The payer owes nothing because they are not in the shares.
- Remainder units in an equal or percentage split.
- A settlement larger than the debt, a settlement with oneself, or a zero amount is refused.
- A new member joins with a zero balance.
- A fully settled group has no suggested payments.

### Likely follow-up questions
1. **"Is the fewest-payments result really the minimum?"** It gives at most (people − 1) payments, which is usually the minimum, but the true minimum is a hard problem (it means splitting people into the most groups that each add up to zero). The greedy approach is the practical answer.
2. **"Balances are slow when there are thousands of expenses. What do you do?"** Keep a running balance inside the `Group`, updated by the same methods that add expenses and transfers, and drop it if it is ever in doubt. It is a deliberate second copy kept for speed, so it is a trade-off that should be said out loud.
3. **"How would you swap the simplification method?"** Make `DebtSimplifier` an interface at that point, with the current greedy class as the first implementation. It is passed into the service already, so only the declaration changes.

## 11. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Several currencies | All amounts share one currency. Converting would be a new class applied before the split. |
| Editing or deleting an expense | No requirement asks for it. It would reverse an entry in the history. |
| Descriptions and dates on expenses | No requirement asks for them. They are extra fields on `Expense`. |
| Decimal percentages | Percentages are whole numbers. Finer ones would use a bigger scale such as hundredths of a percent. |
| Removing a member | Not required. A member with a non-zero balance would need to be settled first. |
| Names in balance results | Results use user ids. The caller looks names up. |
| Concurrent users | Section 8 gives the one-lock-per-group answer. |
| Notifications about debts | No requirement asks for them. |

## 12. Requirements Met
1. Register users and create groups: `SplitwiseService.register_user` and `create_group`, with `User` and `Group`.
2. Add an expense paid by one person and shared by several: `SplitwiseService.add_expense`, `Group.add_expense` and `Expense`.
3. Equal, exact or percentage splits: `EqualSplit`, `ExactSplit` and `PercentSplit`.
4. New ways of splitting: the `SplitStrategy` interface.
5. Track what each user owes or is owed: `Group.get_balance_between`, `get_balances_of` and `get_net_balances`, all computed from the expense and transfer history.
6. View balances with each person and within each group: `SplitwiseService.get_balances` and `get_group_balances`.
7. Settle dues: `SplitwiseService.settle` and `Group.settle`.
8. Fewest payments: `SplitwiseService.get_simplified_payments` with `DebtSimplifier.simplify`.

## 13. Requirement Fit Check

**Setup used in every example:** users `A`, `B`, `C` are registered. Group `G1` has members A, B and C.

1. **Register and create groups**
   - *Success:* `register_user("A", "Asha")` returns the User, and `create_group("G1", "Trip", ["A", "B", "C"])` returns the Group.
   - *Failure:* `register_user("A", "Other")` returns None because the id is taken. `create_group("G2", "Gym", ["A", "Z"])` returns None because `Z` is not registered, and no group is created.
2. **Add an expense**
   - *Success:* `add_expense("G1", "A", 300, EqualSplit(["A","B","C"]))` returns an Expense with shares A 100, B 100, C 100. B and C each owe A 100.
   - *Failure:* `add_expense("G1", "Z", 300, ...)` returns None because `Z` is not a member. `add_expense("G1", "A", 0, ...)` returns None. Nothing is stored.
3. **Equal, exact and percentage splits**
   - *Success (exact):* B pays 90 with `ExactSplit({"A": 20, "C": 70})`. Shares are A 20, C 70. *Success (percent):* C pays 200 with `PercentSplit({"A": 50, "B": 30, "C": 20})`. Shares are A 100, B 60, C 40. *Success (remainder):* 100 split equally among A, B, C gives A 34, B 33, C 33.
   - *Failure:* `ExactSplit({"A": 20, "C": 60})` for a total of 90 adds to 80, so the result is None. `PercentSplit({"A": 50, "B": 30})` adds to 80, so the result is None. Nothing is stored.
4. **New ways of splitting**
   - *Success:* a new `ShareSplit({"A": 2, "B": 1, "C": 1})` class implements `calculate_shares` and is passed to the same `add_expense` with no other change.
   - *Failure:* a faulty strategy returns shares that add up to 99 for a total of 100. `Expense` refuses them, so `add_expense` returns None and nothing is stored.
5. **Track who owes whom**
   - *Success:* after the three expenses above, `get_balance_between("B", "A")` is 80 (B owes A 80, from 100 minus 20), between A and C it is 0, and `get_balance_between("C", "B")` is 10. The net balances are A +80, B −70, C −10, which add up to 0.
   - *Failure:* `get_balance_between("A", "Z")` returns None because `Z` is not a member.
6. **View balances**
   - *Success (before the settlement in example 7):* `get_group_balances("G1", "B")` returns {"A": 80, "C": −10}. A second group `G2` with A and B, where A paid 50 split equally, makes B owe A 25 more, so `get_balances("B")` returns {"A": 105, "C": −10}.
   - *Failure:* `get_balances("Z")` returns None. `get_group_balances("G1", "Z")` returns None because `Z` is not in the group.
7. **Settle**
   - *Success:* `settle("G1", "B", "A", 50)` returns True and B now owes A 30.
   - *Failure:* `settle("G1", "B", "A", 100)` returns False because B owes only 30. `settle("G1", "A", "A", 10)` returns False. Nothing is added.
8. **Fewest payments**
   - *Success:* taking the balances before the settlement in example 7 (A +80, B −70, C −10), `get_simplified_payments("G1")` returns two transfers, B pays A 70 and C pays A 10. The group had five separate debts before.
   - *Failure:* for a fully settled group it returns an empty list. For an unknown group it returns None.
