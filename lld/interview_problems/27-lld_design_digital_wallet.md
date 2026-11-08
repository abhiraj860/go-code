# Digital Wallet & Payment Gateway Integration: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** a wallet's balance is changed only by the wallet itself, under its own lock, and a transfer locks both wallets in a fixed order so the money moves in one step. Every request carries an idempotency key, and the key is claimed before anything happens, so a repeated request returns the first result instead of moving money again. Outside payment providers sit behind one interface, and anything that fails after money has left a wallet is either retried or paid back, and recorded either way.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. Users should have a wallet with a balance.
2. Users should be able to add money from a bank account or card.
3. Users should be able to transfer money to other wallets and pay merchants.
4. Users should be able to view their transaction history with the status of each transaction.
5. The system should support multiple payment methods and gateways that can be swapped without changing the rest of the system.
6. A transfer should either debit and credit both wallets or change neither.
7. The system should reject a payment when the balance is not enough.
8. Balances should stay correct when many transactions happen at the same time.
9. Failed payments should be refunded or retried.
10. A repeated request should not be processed twice.

**Types and conventions**
- Money is `int`, in the smallest currency unit, so there are no rounding errors. Times are `datetime`, passed in by the caller.
- Each request from a client carries an idempotency key, a string the client makes once per intended action and resends unchanged on a retry.
- One wallet per user.
- Retries against a gateway happen straight away, a fixed number of times, and use the same idempotency key, so the gateway itself won't charge twice.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock. `AtomicInt` is a thread-safe counter. A "thread-safe map" supports an atomic `put_if_absent`, which stores a value only if the key is empty and returns whichever value is now stored.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| TransactionType, TransactionStatus, MethodType, GatewayOutcome (enums) | Name the fixed values | none |
| Wallet | Holds a balance and guards every change to it, including transfers | balance |
| PaymentMethod (interface) | Describes a source or destination of outside money | none |
| CardMethod | A card | masked card number, holder |
| BankAccountMethod | A bank account | masked account number, bank code |
| GatewayResult | Records what a gateway answered | outcome, reference |
| PaymentGateway (interface) | Charges or pays out through an outside provider | none |
| CardGateway | Talks to the card provider | none |
| BankGateway | Talks to the bank provider | none |
| GatewayRouter | Picks the gateway for a payment method | the method-to-gateway table |
| Merchant | Describes a merchant and where they are paid | id, name, settlement account |
| Transaction | Records one money movement and its status | type, parties, amount, status |
| WalletService | Runs every money movement in the right order, once per key | wallets, merchants, transactions, keys |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** User, wallet, balance, bank account, card, transfer, merchant, payment, transaction, history, status, payment method, gateway, refund, request.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Wallet, Merchant, Transaction, the payment methods and gateways are classes. Balance and status are values. A refund is a status and a credit back, not a class.
2. **Find fixed sets of values.** Transaction types, transaction statuses, method types, gateway outcomes.
   - *Result:* `TransactionType`, `TransactionStatus`, `MethodType`, `GatewayOutcome`.
3. **Give every fact one owner.** The balance belongs to the Wallet. A money movement and its status belong to its Transaction. "This key was already used" belongs to the service's key map.
   - *Why:* if a transaction also stored "the balance after", it could disagree with the wallet. The history is the transactions, and the balance is the wallet.
4. **Mark what can be computed and do not store it.** A wallet's history is the list of transactions that name it, computed on request.
5. **Give every verb to the class whose data changes.** Debit, credit and transfer change wallets, so `Wallet` owns them, including the two-wallet transfer. A status change belongs to the Transaction.
6. **Find the action that touches several classes.** Paying a merchant needs the key check, a debit, a gateway payout that can fail, and a refund if it does. That lives in `WalletService.pay_merchant`.
7. **Find the hint phrase for a pattern.** "Multiple payment methods and gateways that can be swapped without changing the rest of the system" is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** Requirement 8 names it, so decide exactly where the locks sit (section 9).
10. **Fix the outputs.** A `Transaction` (with its status) or None for every money movement. None means the request was invalid and nothing was recorded.

## 4. Pattern Guide

### Used: Strategy (gateways and payment methods)
- **Meaning:** a way of doing something is a small object behind a shared interface, and the service uses whichever one it is given.
- **Hint phrase:** "multiple payment methods and gateways that can be swapped without changing the rest of the system".
- **Without it:** the service would have an `if` for each provider, and swapping one would mean editing it.

### Used, though not a named design pattern: idempotency keys
- **Meaning:** each request names itself with a key, and the first request to claim the key is the only one processed.
- **Hint phrase:** "a repeated request should not be processed twice".

### Used, though not a named design pattern: compensation (pay back on failure)
- **Meaning:** when step two of a money movement fails after step one took the money, step one is undone with a matching credit.
- **Hint phrase:** "failed payments should be refunded or retried".

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A transaction's status only decides which transition is allowed. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. The router looks a gateway up by method type. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### TransactionType
- Values: `TOP_UP`, `TRANSFER`, `MERCHANT_PAYMENT`. *Why:* the three movements in requirements 2 and 3.

### TransactionStatus
- Values: `PENDING`, `SUCCESS`, `FAILED`, `REFUNDED`
- *Why:* requirement 4 shows each status. `FAILED` means no money moved. `REFUNDED` means money left a wallet and was put back.

### MethodType
- Values: `CARD`, `BANK_ACCOUNT`. *Why:* requirement 2's two sources, and the key the router uses to pick a gateway.

### GatewayOutcome
- Values: `SUCCESS`, `DECLINED`, `ERROR`
- *Why:* requirement 9 retries only what might work next time. `ERROR` (a timeout, a network problem) is retried, and `DECLINED` (no funds, a blocked card) is not.

## 6. Classes (smallest dependency first)

### Wallet
**Why this class exists:** requirement 1's balance must have one owner that checks every change. Putting the two-wallet transfer here too means the locking rule for balances lives in one class.
- Attributes: `wallet_id: str`, `owner_id: str`, `balance: int`, `lock: Lock`. *Why:* the identity, the user, the money, and the lock that guards it.
- Constructor: `Wallet(wallet_id: str, owner_id: str)`. *Why:* a new wallet has a balance of 0. It refuses empty ids.
- Methods:
  - `get_wallet_id() -> str`, `get_owner_id() -> str`, `get_balance() -> int`. *Why:* read-only views. `get_balance` reads under the lock.
  - `credit(amount: int) -> bool`. *Why:* under the lock, adds the amount. False if `amount <= 0`.
  - `debit(amount: int) -> bool`. *Why:* under the lock, False if `amount <= 0` or the balance is lower. Otherwise subtracts. This is requirement 7.
  - `transfer(from_wallet: Wallet, to_wallet: Wallet, amount: int) -> bool` (a class method). *Why:* requirement 6. It refuses the same wallet on both sides or `amount <= 0`. It takes the two wallets' locks in order of wallet id, then, if the sender has enough, subtracts from one and adds to the other, and releases both. Either both balances change or neither does. *Why the order by id:* two transfers going opposite ways between the same two wallets then take the locks in the same order, so they can't deadlock.
- *Why there is no setter:* every change goes through a checked method under the lock.
- *Extensibility:* none needed.

### PaymentMethod (interface), CardMethod and BankAccountMethod
**Why they exist:** requirement 2 brings money in from a card or a bank account, and requirement 3 pays merchants out to an account. The service handles both the same way.
- `PaymentMethod` methods: `method_type() -> MethodType`, `describe() -> str` (a masked, safe description such as `card ending 4242`).
- `CardMethod(card_token: str, last_four: str, holder: str)`: *why a token:* the real card number is held by the card provider, never by the wallet. It refuses empty values.
- `BankAccountMethod(account_token: str, last_four: str, bank_code: str)`: the same, for a bank account.
- *Extensibility:* a new method (UPI) is a new class and a new `MethodType` value.

### GatewayResult
**Why this class exists:** a gateway answers with an outcome and the provider's reference, which belong together.
- Attributes: `outcome: GatewayOutcome`, `reference: str or None`. *Why:* what happened, and the provider's id for it, kept for reconciling.
- Constructor: `GatewayResult(outcome: GatewayOutcome, reference: str or None)`.
- Methods: `get_outcome() -> GatewayOutcome`, `get_reference() -> str or None`.

### PaymentGateway (interface), CardGateway and BankGateway
**Why they exist:** requirement 5. Outside providers are swappable, so the service depends on two methods and not on any provider.
- `PaymentGateway` methods:
  - `charge(method: PaymentMethod, amount: int, idempotency_key: str) -> GatewayResult`. *Why:* takes money in. The key is passed on, so the provider ignores a repeat.
  - `payout(method: PaymentMethod, amount: int, idempotency_key: str) -> GatewayResult`. *Why:* sends money out to a merchant.
  - Neither method throws. A problem comes back as `ERROR`.
- `CardGateway()` and `BankGateway()` are where calls to the real providers go.
- *Extensibility:* a new provider is a new class.

### GatewayRouter
**Why this class exists:** requirement 5 swaps gateways "without changing the rest of the system". The choice of gateway for each kind of method is one table, so swapping is a data change.
- Attributes: `routes: map{MethodType: PaymentGateway}` (a method type to the gateway that handles it).
- Constructor: `GatewayRouter(routes: map{MethodType: PaymentGateway})`. *Why:* refuses a table missing any method type.
- Methods: `gateway_for(method: PaymentMethod) -> PaymentGateway`. *Why:* the lookup.
- *Extensibility:* swapping a provider is a new table entry.

### Merchant
**Why this class exists:** requirement 3 pays merchants. A merchant has an identity and an account to be paid into.
- Attributes: `merchant_id: str`, `name: str`, `settlement_method: PaymentMethod`.
- Constructor: `Merchant(merchant_id: str, name: str, settlement_method: PaymentMethod)`. *Why:* all set once. It refuses empty values.
- Methods: getters for each field.

### Transaction
**Why this class exists:** requirement 4 shows each money movement and its status. A transaction is that record, and it guards its own status so it can't be both refunded and successful.
- Attributes:
  - `transaction_id: str`, `idempotency_key: str`, `transaction_type: TransactionType`. *Why:* the identity, the request it came from, and what kind it is.
  - `from_wallet_id: str or None`, `to_wallet_id: str or None`, `merchant_id: str or None`. *Why:* who pays and who receives. A top-up has no `from`, and a merchant payment has no `to` wallet.
  - `amount: int`, `created_at: datetime`. *Why:* how much and when.
  - `status: TransactionStatus`, `gateway_reference: str or None`, `note: str`. *Why:* the outcome, the provider's id, and a short reason (such as "insufficient balance" or "card declined").
  - `lock: Lock`. *Why:* status changes run one at a time.
- Constructor: `Transaction(transaction_id: str, idempotency_key: str, transaction_type: TransactionType, from_wallet_id: str or None, to_wallet_id: str or None, merchant_id: str or None, amount: int, created_at: datetime)`. *Why:* the status starts `PENDING`. It refuses `amount <= 0`.
- Methods (each under the lock):
  - Getters for each field, and `involves(wallet_id: str) -> bool`.
  - `succeed(reference: str or None) -> bool`. *Why:* `PENDING` to `SUCCESS`.
  - `fail(note: str) -> bool`. *Why:* `PENDING` to `FAILED`.
  - `mark_refunded(note: str) -> bool`. *Why:* `PENDING` to `REFUNDED`, for money that left a wallet and was put back.
- *Extensibility:* none needed.

### WalletService (the manager and only entry point)
**Why this class exists:** every money movement crosses several classes in a fixed order, must happen once per key, and must leave a record. No single wallet or gateway can do that.
- Attributes:
  - `wallets: map{str: Wallet}` (a wallet id to the Wallet), `owners: map{str: str}` (a user id to their wallet id), `merchants: map{str: Merchant}` (a merchant id to the Merchant), `transactions: map{str: Transaction}` (a transaction id to the Transaction). *Why:* the one record of each. All are thread-safe maps.
  - `keys: map{str: str}` (an idempotency key to the transaction id it created). *Why:* the single record of which requests were already handled. Claimed with `put_if_absent`.
  - `router: GatewayRouter`. *Why:* passed in.
  - `max_gateway_attempts: int`. *Why:* how many times to try a gateway on `ERROR`. It is data.
  - `counter: AtomicInt`. *Why:* makes wallet and transaction ids unique.
- Constructor: `WalletService(router: GatewayRouter, max_gateway_attempts: int)`. *Why:* refuses attempts below 1.
- Methods:
  - `create_wallet(owner_id: str) -> Wallet or None`. *Why:* requirement 1. None if that user already has one.
  - `add_merchant(merchant_id: str, name: str, settlement_method: PaymentMethod) -> bool`. *Why:* setup.
  - `top_up(key: str, wallet_id: str, method: PaymentMethod, amount: int, now: datetime) -> Transaction or None`. *Why:* requirement 2.
  - `transfer(key: str, from_wallet_id: str, to_wallet_id: str, amount: int, now: datetime) -> Transaction or None`. *Why:* requirements 3 and 6.
  - `pay_merchant(key: str, wallet_id: str, merchant_id: str, amount: int, now: datetime) -> Transaction or None`. *Why:* requirements 3 and 9.
  - `get_balance(wallet_id: str) -> int or None`.
  - `get_history(wallet_id: str) -> list[Transaction] or None`. *Why:* requirement 4. Computed, newest first.
- Private helpers:
  - `claim(key: str, build: a new Transaction) -> (Transaction, bool)`. *Why:* stores the new transaction's id under the key with `put_if_absent`. If the key was already used, returns the existing transaction and False, so the caller stops. This is requirement 10.
  - `call_with_retries(call) -> GatewayResult`. *Why:* calls the gateway up to `max_gateway_attempts` times while the outcome is `ERROR`, with the same key each time.
- *Extensibility:* gateways and their table are passed in.

## 7. Relationships
- WalletService **has** Wallet, Transaction, Merchant (created inside its methods).
- Merchant **uses** PaymentMethod (passed in).
- GatewayRouter **uses** PaymentGateway (passed in).
- WalletService **uses** GatewayRouter (passed in).
- CardMethod, BankAccountMethod **implement** PaymentMethod. CardGateway, BankGateway **implement** PaymentGateway.
- Enums are always **uses**: Transaction to TransactionType and TransactionStatus, PaymentMethod to MethodType, GatewayResult to GatewayOutcome, GatewayRouter to MethodType.
- `GatewayResult` appears only as a return type, so it has no line from the service.

## 8. Interaction

### Layers
1. **Entry point and manager:** WalletService.
2. **Classes that guard themselves:** Wallet (balances) and Transaction (status).
3. **Outside providers:** GatewayRouter, PaymentGateway and its implementations.
4. **Plain data:** PaymentMethod and its implementations, Merchant, GatewayResult.

### Call chains
- **Top up**
  1. `User -> WalletService.top_up(key, wallet, method, amount, now)`.
  2. `WalletService -> claim(key, ...)`: a repeated key returns the first result here.
  3. `WalletService -> GatewayRouter.gateway_for(method)`, then `PaymentGateway.charge(...)`, retried on `ERROR`.
  4. On `SUCCESS`: `Wallet.credit(amount)`, then `Transaction.succeed(reference)`.
- **Transfer**
  1. `User -> WalletService.transfer(key, from, to, amount, now)`, then `claim(key, ...)`.
  2. `WalletService -> Wallet.transfer(from_wallet, to_wallet, amount)`: one locked step.
  3. `Transaction.succeed` or `fail`.
- **Pay a merchant**
  1. `User -> WalletService.pay_merchant(...)`, then `claim(key, ...)`.
  2. `WalletService -> Wallet.debit(amount)`: the money leaves only if there is enough.
  3. `WalletService -> PaymentGateway.payout(merchant method, amount, key)`, retried on `ERROR`.
  4. On failure: `Wallet.credit(amount)` to pay it back, then `Transaction.mark_refunded`.

### Who may call whom
- Users call `WalletService` only.
- `WalletService` calls `Wallet`, `Transaction`, `GatewayRouter` and the gateways.
- Gateways call only their outside providers. Wallets and transactions call no other class.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Top up (requirement 2)
1. `top_up(key, wallet_id, method, amount, now)`. If the wallet is unknown or `amount <= 0`, return None.
2. `claim(key, new TOP_UP transaction)`. If the key was already used, return that transaction.
3. `result = call_with_retries(gateway.charge(method, amount, key))`.
4. `SUCCESS`: `wallet.credit(amount)`, then `transaction.succeed(reference)`.
5. `DECLINED`, or `ERROR` after every attempt: `transaction.fail(...)`. The wallet is untouched.
- **Why credit only after the charge succeeds:** the wallet never shows money that was not really collected.

### Transfer (requirements 3, 6, 7)
1. `transfer(key, from_wallet_id, to_wallet_id, amount, now)`. If either wallet is unknown, they are the same, or `amount <= 0`, return None.
2. `claim(key, new TRANSFER transaction)`. If already used, return that transaction.
3. `Wallet.transfer(from_wallet, to_wallet, amount)`. True means `succeed`. False means `fail("insufficient balance")`.
- **Failure and undo:** nothing to undo. Both balances change in one locked step or neither does.

### Pay a merchant (requirements 3, 7, 9)
1. `pay_merchant(key, wallet_id, merchant_id, amount, now)`. If the wallet or merchant is unknown or `amount <= 0`, return None.
2. `claim(key, new MERCHANT_PAYMENT transaction)`. If already used, return that transaction.
3. `wallet.debit(amount)`. If False, `fail("insufficient balance")` and stop. *Why first:* the money is held before anyone is paid.
4. `result = call_with_retries(gateway.payout(merchant method, amount, key))`.
5. `SUCCESS`: `transaction.succeed(reference)`.
6. Otherwise: `wallet.credit(amount)` and `transaction.mark_refunded(...)`. *Undone:* the debit, with a matching credit. The user's balance is as it was, and the history shows the payment was refunded.

### Repeated request (requirement 10)
1. The client resends with the same key. `claim` finds the key and returns the original transaction, whatever its status. No money moves again.
2. Two copies arriving at the same moment both call `put_if_absent`. Exactly one stores its id, and the other gets that id back.

### History (requirement 4)
1. `get_history(wallet_id)` filters every transaction that `involves(wallet_id)`, newest first. None if the wallet is unknown.

### Concurrency (requirement 8)
- **Wallet:** the lock sits inside each `Wallet`, around `credit`, `debit` and balance reads. A check and a change are one step, so two debits can never both pass a balance check meant for one.
- **Transfer:** `Wallet.transfer` takes both wallets' locks, always in order of wallet id. Opposite transfers between the same two wallets can't deadlock, and no one ever sees money that has left one wallet but not reached the other.
- **Transaction:** the lock sits inside each `Transaction`, around status changes.
- **Keys and maps:** `keys` uses `put_if_absent`, so only one request per key goes on. All maps are thread-safe, and ids come from an atomic counter.
- **No lock during a gateway call:** gateway calls happen with no wallet lock held, so a slow provider never blocks other users' balances. A merchant payment's money is already debited, so it can't be spent twice while the payout runs.
- **Lock order:** at most two wallet locks, always in id order, and never together with a transaction lock. So no deadlock is possible.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new payment provider for cards | A new class implementing `PaymentGateway`, set in the router's table | No. |
| A new payment method (UPI) | A new `MethodType` value, a new `PaymentMethod` class, and a router entry | No logic changes. |
| More retry attempts | A different `max_gateway_attempts` | No. It is data. |
| New merchants | Calls to `add_merchant` | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **Balances change only inside `Wallet`, under its lock.**
   - *Why:* every check-then-change is one step, so concurrent debits can't overdraw.
   - *Rejected:* the service reading a balance and then setting a new one, which loses updates under concurrency.
2. **A transfer locks both wallets in order of id.**
   - *Why:* it is atomic, and the fixed order prevents deadlock.
   - *Rejected:* a debit then a separate credit, where a reader could see the money missing for a moment, and a failure in between needs fixing up.
3. **The idempotency key is claimed before anything else.**
   - *Why:* a repeated request, even one arriving at the same moment, is answered with the first result.
   - *Rejected:* checking for a key after processing, which lets two copies both move money.
4. **Money held first, payout second, and a credit back if the payout fails.**
   - *Why:* the user's money can't be spent twice while the payout is in progress, and a failure leaves the balance as it was.
   - *Rejected:* paying out first and debiting after, which can pay a merchant from an empty wallet.
5. **The gateway for each method type is a table.**
   - *Why:* swapping a provider is a data change, as requirement 5 asks.
   - *Rejected:* an `if` on the provider inside the service.

### Edge cases handled
- A transfer to the same wallet, or of 0 or less, is refused.
- A transfer or payment larger than the balance fails and is recorded as failed.
- A declined card is not retried. A gateway error is retried a set number of times.
- A merchant payout that fails every attempt is paid back, and the transaction shows `REFUNDED`.
- A repeated key returns the first transaction, even if it failed.
- Opposite transfers between two wallets at the same moment don't deadlock.
- A user can't open a second wallet.
- A card or account is kept as a provider token, not as a real number.

### Likely follow-up questions
1. **"What if the gateway charged the card but its answer was lost?"** The retry uses the same key, so the provider returns the original charge instead of charging again. A daily reconciliation against the provider's records catches anything left over.
2. **"How would you scale this across many servers?"** Balances move to a database, the wallet lock becomes a row lock (or a version check), and the key map becomes a table with a unique key column.
3. **"Why keep a balance at all instead of adding up the history?"** Adding up the history on every payment is slow. A real system keeps a double-entry ledger as the record and the balance as a checked summary of it.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Waiting between retries | Retries happen straight away. A delay would be added in `call_with_retries`. |
| Reconciling with providers | See follow-up question 1. |
| Withdrawing money to a bank | Not required. It would be a payout from the user's own wallet. |
| A full double-entry ledger | See follow-up question 3. |
| Limits, fees and fraud checks | Not required. They would be checks before the debit. |
| Expiring old idempotency keys | Keys are kept forever here. A real system would drop them after a set time. |

## 13. Requirements Met
1. A wallet with a balance: `Wallet` and `WalletService.create_wallet`.
2. Add money from a bank account or card: `WalletService.top_up` with `CardMethod` or `BankAccountMethod`.
3. Transfer and pay merchants: `WalletService.transfer` and `pay_merchant`.
4. History with statuses: `Transaction`, `TransactionStatus` and `WalletService.get_history`.
5. Swappable methods and gateways: `PaymentMethod`, `PaymentGateway` and `GatewayRouter`.
6. All or nothing transfers: `Wallet.transfer`.
7. Reject when the balance is not enough: `Wallet.debit` and `Wallet.transfer`.
8. Correct under concurrency: the lock inside each `Wallet`, ordered locking in `transfer`, and `put_if_absent` on keys.
9. Refund or retry failed payments: `call_with_retries` and the credit back in `pay_merchant`.
10. No double processing: the `keys` map and `claim`.

## 14. Requirement Fit Check

**Setup used in every example:** wallets `W1` (alice) and `W2` (bob), both starting at 0. Merchant `M1` "Cafe" with a bank account. Gateways allow 3 attempts. A test card gateway approves card `...4242` and declines `...0002`. A test bank gateway can be set to answer `ERROR`.

1. **A wallet with a balance**
   - *Success:* `create_wallet("alice")` returns `W1` with a balance of 0.
   - *Failure:* `create_wallet("alice")` again returns None.
2. **Add money**
   - *Success:* `top_up("k1", "W1", card 4242, 1000, now)` returns a `SUCCESS` transaction and W1 is 1000. A top-up from a bank account works the same way.
   - *Failure:* `top_up("k5", "W1", card 0002, 500, now)` returns a `FAILED` transaction ("card declined"), and W1 is still 1000.
3. **Transfer and pay merchants**
   - *Success:* `transfer("k2", "W1", "W2", 300, now)` is `SUCCESS`, so W1 is 700 and W2 is 300. `pay_merchant("k3", "W1", "M1", 200, now)` is `SUCCESS`, so W1 is 500.
   - *Failure:* `transfer("k6", "W1", "W1", 10, now)` returns None because it is the same wallet.
4. **History with statuses**
   - *Success:* `get_history("W1")` shows, newest first, the merchant payment (`SUCCESS`), the declined top-up (`FAILED`), the transfer (`SUCCESS`) and the first top-up (`SUCCESS`).
   - *Failure:* `get_history("W9")` returns None.
5. **Swappable gateways**
   - *Success:* the router maps `CARD` to `CardGateway` and `BANK_ACCOUNT` to `BankGateway`. Replacing the card entry with a new provider class changes how cards are charged, with no other edit.
   - *Failure:* a router missing the `BANK_ACCOUNT` entry is refused at construction.
6. **All or nothing transfers**
   - *Success:* the transfer of 300 changes both balances in one locked step.
   - *Failure:* a transfer to an unknown wallet returns None and changes nothing. A reader never sees W1 down 300 while W2 is not yet up 300.
7. **Reject when the balance is not enough**
   - *Success:* `transfer("k4", "W2", "W1", 5000, now)` returns a `FAILED` transaction ("insufficient balance").
   - *Failure to avoid:* W2 must still be 300, and W1 unchanged.
8. **Correct under concurrency**
   - *Success:* 100 transfers of 10 run at once, half from W1 to W2 and half the other way. All finish without deadlock, and W1 plus W2 is still 800 (1000 minus the 200 paid to the merchant).
   - *Failure to avoid:* two debits of 400 from a wallet holding 500 at the same moment can't both succeed, because each check and change is one step.
9. **Refund or retry**
   - *Success (retry):* the card gateway answers `ERROR` once, then `SUCCESS`, so the top-up retries with the same key and ends `SUCCESS`, credited once.
   - *Success (refund):* the bank gateway answers `ERROR` three times on a merchant payout of 100. W1 is debited 100, then credited back 100, and the transaction ends `REFUNDED`.
10. **No double processing**
    - *Success:* `transfer("k2", "W1", "W2", 300, now)` is sent again. It returns the original transaction, and no money moves.
    - *Failure to avoid:* two copies of `k2` arriving at the same moment can't both transfer, because only one `put_if_absent` succeeds.
