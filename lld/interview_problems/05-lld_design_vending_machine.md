# Vending Machine: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** every fact has exactly one owner, and the machine's behavior changes by swapping a small state object, not by `if` checks. The machine asks the current state what to do, and the state changes only the data it owns, through checked methods.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The machine should hold multiple products in slots, each with a price and a quantity.
2. The machine should allow a user to select a product and insert money.
3. The machine should behave differently depending on whether it is idle, has money inserted, is dispensing or is out of service.
4. The machine should dispense the product once enough money is inserted and return the change.
5. The machine should allow the user to cancel and get a refund.
6. The machine should refuse a selection when the product is out of stock.
7. An operator should be able to restock products and collect the money.

**Types and conventions**
- Money is `int`, in the smallest currency unit (for example cents). Prices, balances, change and refunds are all `int`.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- No locks are used, because no requirement mentions concurrent use (see Concurrency in section 9).

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Product | Describes one sellable item | name, price |
| Slot | Holds one product and counts what is left | quantity |
| Inventory | Sets up slots and finds a slot by its code | the map of slots |
| CashBox | Keeps money paid by customers until the operator collects it | collected total |
| Session | Tracks one customer's inserted money and chosen slot | balance, selected slot |
| Response | Reports the result of a user action | an unchangeable record |
| MachineState (interface) | Lists every action a state must answer | none |
| IdleState | Behavior while waiting for money | none |
| HasMoneyState | Behavior while the customer has money | none |
| DispensingState | Runs the sale | none |
| OutOfServiceState | Refuses customers | none |
| VendingMachine | Hands each action to the current state, applies the result, guards operator actions | which state it is in |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Machine, product, slot, price, quantity, user, money, state, change, refund, operator, collected money.
   - *Why:* nouns that carry their own data become classes, and nouns that are just a number become attributes.
   - *Result:* Product, Slot, CashBox, Session, VendingMachine. Price and quantity are attributes. Change and refund are numbers, not classes.
2. **Find fixed sets of values.** The four named conditions in requirement 3.
   - *Why:* a fixed set of values is an enum, so a typo like "idel" cannot exist.
   - *Result:* `MachineStateType`.
3. **Give every fact one owner.**
   - Price belongs to Product, quantity to Slot, inserted money and chosen slot to Session, collected money to CashBox, the current state to VendingMachine.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.**
   - "In stock" is `quantity > 0`. "Enough money" is `balance >= price`. Change is `balance - price`.
   - *Why:* a stored "in stock" flag beside quantity would go wrong the moment someone forgets to update it.
5. **Give every verb to the class whose data changes.** Take one item: Slot. Collect money: CashBox. Cancel and refund: Session.
   - *Why:* the owner can check the rule before it changes the data, so nobody can put the class in a bad state.
6. **Find the action that touches several classes.** Dispensing touches Slot, Session and CashBox, and the order matters.
   - *Why:* an action with several steps needs one place that knows the order and the undo.
   - *Result:* DispensingState runs the sale, and VendingMachine decides when it runs.
7. **Find the hint phrase for a pattern.** Requirement 3 says "behave differently depending on...". That is the State pattern.
8. **Pick likely changes** and check each one is a new class or new data (section 10).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Choose one result type for every user action.** `Response` carries accepted or refused, the next state, the product, and the money returned. *Why:* the caller needs one consistent answer shape.

## 4. Pattern Guide

### Used: State
- **Meaning:** each condition of the machine is a small class with its own behavior, and the machine just asks the current one.
- **Hint phrase:** "behave differently depending on whether it is idle, has money inserted, is dispensing or is out of service".
- **Without it:** every method (`insert_money`, `select_product`, `cancel`, and so on) would have an `if` on the state, and adding a state would mean editing all of them.
- **When NOT to use it:** when there are only two simple states with one differing behavior. A boolean is enough there.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| Strategy | "different ways to calculate or pay" | Not used. There is one way to pay (money), so no family of interchangeable rules. |
| Observer | "notify or alert when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. Products are plain data, not different classes. |
| Facade | "a single simple entry point" | Not added as a separate class. `VendingMachine` is already the single entry point because it is the manager. |
| Chain of Responsibility | "pass the request along handlers" | Not used. Change is a plain number, not coin handlers. |
| Singleton | "exactly one instance" | Not used. No requirement says one machine only, and a passed-in machine is easier to test. |

## 5. Enums

### MachineStateType
- Values: `IDLE`, `HAS_MONEY`, `DISPENSING`, `OUT_OF_SERVICE`
- *Why:* these four conditions come straight from requirement 3. A state class returns one of these labels to say where the machine goes next.
- *Why a label and not the next state object:* if each state held the next state object, the states would point at each other in a circle. A label has no such dependency.
- *Extensibility:* a new condition is one new value here, plus a new state class and a map entry.

## 6. Classes (smallest dependency first)

### Product
**Why this class exists:** price belongs to the item, not to the slot, because two slots can hold the same item. It stays separate from Slot so a price is never mixed up with a quantity.
- Attributes:
  - `name: str`. *Why:* the customer-visible name. The slot code is the identity, so no id is needed.
  - `price: int`. *Why:* the price of one unit. It lives here because it describes the item.
- Constructor: `Product(name: str, price: int)`. *Why:* both fields are set once and never change, so a Product can't be altered after it is shared. It refuses an empty name or `price <= 0`.
- Methods:
  - `get_name() -> str`. *Why public:* states need it to report which product was sold.
  - `get_price() -> int`. *Why public:* states compare the balance against it.
  - There are no setters, so a price can't be changed behind a customer's back.
- *Extensibility:* a new product is a new Product value, with no code change.

### Slot
**Why this class exists:** it owns the one fact that changes with every sale, the quantity. It is separate from Product so the item can be shared by several slots.
- Attributes:
  - `code: str`. *Why:* the label a customer types, such as "A1".
  - `product: Product`. *Why:* which item this slot holds.
  - `quantity: int`. *Why:* how many are left. This is the only stock fact. There is no separate "in stock" flag, because that would be a second copy.
- Constructor: `Slot(code: str, product: Product, quantity: int)`. *Why:* a slot is never created without a product or a stock count. It refuses an empty code or a negative quantity.
- Methods:
  - `get_code() -> str`, `get_product() -> Product`, `get_quantity() -> int`. *Why:* read-only views.
  - `is_in_stock() -> bool`. *Why:* computed from `quantity > 0`, so it can never disagree with the quantity.
  - `take_one() -> bool`. *Why:* a checked method instead of `set_quantity`. It refuses at zero, so quantity can never go negative. It returns a bool so the caller knows whether it worked.
  - `restock(count: int) -> bool`. *Why:* the only way to add stock. It refuses `count <= 0`.
- *Extensibility:* more slots are data entries in Inventory, with no code change.

### Inventory
**Why this class exists:** it answers "which slot is code A1?". It is separate from Slot because one slot should not know about its siblings.
- Attributes: `slots: map{str: Slot}` (slot code to the Slot with that code). *Why here:* the collection is a different fact from any one slot.
- Constructor: `Inventory()`. *Why:* it starts with an empty map, ready to use.
- Methods:
  - `add_slot(code: str, product: Product, quantity: int) -> bool`. *Why:* it builds the Slot itself, so no outside code can hold a half-set-up slot. False if the code already exists or the values are invalid.
  - `get_slot(code: str) -> Slot or None`. *Why:* the one id-to-object lookup. Everything after it passes the Slot object.
  - `restock(code: str, count: int) -> bool`. *Why:* it turns the code into a Slot once, then asks the Slot to restock itself. False if the code is unknown or the count is invalid.
- *Extensibility:* a new slot is `add_slot`, which is data and not code.

### CashBox
**Why this class exists:** collected money is a different fact from a customer's inserted money, and the operator collects it. It stays separate so the operator's action touches only this class.
- Attributes: `collected: int`. *Why:* the total of all sales since the last collection.
- Constructor: `CashBox()`. *Why:* it starts at 0.
- Methods:
  - `add(amount: int) -> bool`. *Why:* it refuses `amount <= 0`, so the total can only grow by real sales.
  - `collect() -> int`. *Why:* it returns the total and resets it to 0 in one method, so the same money can't be collected twice. It returns 0 when empty.
- *Extensibility:* none needed for the listed changes.

### Session
**Why this class exists:** "money inserted" belongs to one customer's visit. It is separate from the machine so the machine holds no data that changes with every action.
- Attributes:
  - `balance: int`. *Why:* the money inserted so far.
  - `selected_slot: Slot or None`. *Why:* the chosen slot. Holding the Slot object means the sale never looks it up again.
- Constructor: `Session()`. *Why:* it starts with balance 0 and no selection, which is the idle condition.
- Methods:
  - `add_money(amount: int) -> bool`. *Why:* refuses `amount <= 0`, so the balance can't be reduced or left unchanged by a bad call.
  - `select_slot(slot: Slot) -> bool`. *Why:* refuses when the balance is 0, because a selection with no money is meaningless. It replaces any earlier selection.
  - `get_selected_slot() -> Slot or None`. *Why:* the sale step needs the chosen slot.
  - `can_afford(price: int) -> bool`. *Why:* it is the computed "enough money" check, so no one stores it.
  - `take_payment(price: int) -> int or None`. *Why:* if `balance >= price` it returns the change and resets balance and selection together. Otherwise it returns None and changes nothing. One method does check, deduct and clear, so a half-cleared session can't exist.
  - `refund_all() -> int`. *Why:* returns the full balance and resets the session. It returns 0 if there is nothing to refund.
- *Extensibility:* a card source would add an attribute here, but the listed changes don't need it.
- *Why it is never exposed:* the machine has no getter for it. Only states reach it, so callers can't drive it directly.

### Response
**Why this class exists:** every user action needs the same answer shape. A record keeps the caller from guessing what a None or a bool means.
- Attributes:
  - `accepted: bool`. *Why:* True if the action was done, False if refused with nothing changed.
  - `state_after: MachineStateType`. *Why:* where the machine goes next. On a refusal it is the state's own type, so the machine can always apply it without checking `accepted`.
  - `product_name: str or None`. *Why:* the product sold, or None when no sale happened.
  - `money_returned: int`. *Why:* change or refund. 0 when none.
- Constructor: `Response(accepted: bool, state_after: MachineStateType, product_name: str or None, money_returned: int)`. *Why:* all fields set once, so a result can't be altered after it is made.
- Methods: `is_accepted() -> bool`, `get_state_after() -> MachineStateType`, `get_product_name() -> str or None`, `get_money_returned() -> int`. *Why:* read-only views.
- *Extensibility:* a new result field is added here and nothing else needs to change for existing callers.

### MachineState (interface)
**Why it is an interface:** requirement 3 gives four conditions, so four classes implement the same set of actions. The machine depends on this interface, not on any one state.
- Methods (every one returns a Response unless noted):
  - `insert_money(amount: int) -> Response`
  - `select_product(slot: Slot or None) -> Response`. *Why it takes a Slot:* the machine did the one lookup, so states never look up again. None means an unknown code.
  - `cancel() -> Response`
  - `dispense() -> Response`
  - `take_out_of_service() -> Response`
  - `return_to_service() -> Response`
  - `allows_operator_actions() -> bool`. *Why:* the rule "no restocking in the middle of a customer's purchase" depends on the state, so each state answers it.
- *Why every state has every method:* a state that does not allow an action refuses it with `accepted = False` and its own type, so the machine never needs an `if` per state.
- *Extensibility:* a new state implements these same seven methods.

### IdleState
**Why this class exists:** it defines "waiting for money". It is its own class so idle behavior is not mixed into the others.
- Attributes: `session: Session`. *Why:* the only thing it changes is the customer's money.
- Constructor: `IdleState(session: Session)`. *Why:* it is given what it needs once, so it can't be used without it.
- Behavior:
  - `insert_money`: `session.add_money(amount)`. If True, accept and go to `HAS_MONEY`. If False (bad amount), refuse and stay `IDLE`.
  - `select_product`: refuse, stay `IDLE`. *Why:* money comes first in this design, so a selection needs money.
  - `cancel`: refuse, stay `IDLE`. *Why:* there is nothing to refund.
  - `dispense`: refuse, stay `IDLE`.
  - `take_out_of_service`: accept, go to `OUT_OF_SERVICE`.
  - `return_to_service`: refuse, stay `IDLE`.
  - `allows_operator_actions`: True. *Why:* no customer is in the middle of a purchase.
- *Extensibility:* none needed.

### HasMoneyState
**Why this class exists:** it defines "the customer has money". This is where selection happens and where the sale gets triggered.
- Attributes: `session: Session`. *Why:* it reads and changes the customer's money and selection.
- Constructor: `HasMoneyState(session: Session)`.
- Behavior:
  - `insert_money`: `session.add_money(amount)`. If False, refuse and stay. Otherwise, if a slot is selected and `session.can_afford(price)`, accept and go to `DISPENSING`. Else accept and stay `HAS_MONEY`. *Why:* this lets the customer insert money after selecting.
  - `select_product(slot)`: refuse if `slot` is None (unknown code), or `slot.is_in_stock()` is False (requirement 6), or `session.select_slot(slot)` returns False. Otherwise accept. If `session.can_afford(price)`, go to `DISPENSING`, else stay `HAS_MONEY` and wait for more money. *Why refusing keeps the money:* the customer can pick another product or cancel.
  - `cancel`: `session.refund_all()`, accept, go to `IDLE`, with `money_returned` set to the refund.
  - `dispense`: refuse, stay `HAS_MONEY`. *Why:* only the dispensing state may run a sale.
  - `take_out_of_service`, `return_to_service`: refuse, stay. *Why:* the customer's money must be refunded first.
  - `allows_operator_actions`: False.
- *Extensibility:* none needed.

### DispensingState
**Why this class exists:** it owns the sale steps and their order. It is a state because requirement 3 names it, and it refuses everything else while a sale runs.
- Attributes: `session: Session`, `cash_box: CashBox`. *Why:* a sale changes the customer's money and the collected money, and it needs nothing else.
- Constructor: `DispensingState(session: Session, cash_box: CashBox)`.
- Behavior:
  - `insert_money`, `select_product`, `cancel`, `take_out_of_service`, `return_to_service`: refuse, stay `DISPENSING`. *Why:* nothing may interrupt a sale.
  - `allows_operator_actions`: False.
  - `dispense`: see the sale steps below.
- **Sale steps, in this order (the order is the whole point):**
  1. `slot = session.get_selected_slot()`. If None, or `session.can_afford(price)` is False, refuse and go to `HAS_MONEY`. Nothing has changed yet.
  2. `slot.take_one()`. If False, refuse and go to `HAS_MONEY`. Nothing has changed.
  3. `change = session.take_payment(price)`. If None (it should not happen, since step 1 checked), undo with `slot.restock(1)`, refuse and go to `HAS_MONEY`.
  4. `cash_box.add(price)`.
  5. Return `Response(True, IDLE, product name, change)`.
- *Why this order:* the two steps that can fail are done first, and the one step that can't fail (adding to the cash box) is last, so a failure never leaves money taken without a product.
- *Extensibility:* none needed.

### OutOfServiceState
**Why this class exists:** it defines "not serving customers". It is its own class so customer actions are refused in one place.
- Attributes: none.
- Constructor: `OutOfServiceState()`.
- Behavior:
  - `insert_money`, `select_product`, `cancel`, `dispense`, `take_out_of_service`: refuse, stay `OUT_OF_SERVICE`. *Why money is refused:* the machine can't give a product, so it must not take money.
  - `return_to_service`: accept, go to `IDLE`.
  - `allows_operator_actions`: True.
- *Extensibility:* none needed.

### VendingMachine (the manager and only entry point)
**Why this class exists:** the actions touch several classes and the order matters, so one class must run each action. It holds no sale data of its own, only which state it is in.
- Attributes:
  - `inventory: Inventory`. *Why:* to find slots and set them up.
  - `cash_box: CashBox`. *Why:* the operator collects from it. It is the same object the dispensing state fills.
  - `states: map{MachineStateType: MachineState}` (state type to the state object for that type). *Why:* the machine looks up the real state from the label a state returns.
  - `current_type: MachineStateType`. *Why:* the one place that records which state the machine is in.
- Constructor: `VendingMachine(inventory: Inventory, cash_box: CashBox, states: map{MachineStateType: MachineState})`. *Why:* all collaborators are passed in, so tests can use fakes and no state needs editing to add another. It starts at `IDLE` and refuses a map with no `IDLE` entry.
- Public methods:
  - `insert_money(amount: int) -> Response`. *Why:* it hands the request to the current state, because only the state knows if money is accepted now.
  - `select_product(slot_code: str) -> Response`. *Why it takes an id:* it is the entry point. It looks up the Slot once with `inventory.get_slot` and passes the object inward.
  - `cancel() -> Response`
  - `take_out_of_service() -> Response`
  - `return_to_service() -> Response`
  - `add_slot(slot_code: str, product: Product, quantity: int) -> bool`. *Why:* operator action. False if the current state does not allow operator actions or `inventory.add_slot` refuses.
  - `restock(slot_code: str, count: int) -> bool`. *Why:* operator action, gated the same way.
  - `collect_money() -> int or None`. *Why:* operator action. None if the current state does not allow operator actions. Otherwise it returns `cash_box.collect()`.
  - `get_state() -> MachineStateType`. *Why:* read-only view of the current state.
- Private helper: `finish(response: Response) -> Response`.
  - It sets `current_type` to `response.get_state_after()`.
  - If the new type is `DISPENSING`, it calls the current state's `dispense()`, applies that response the same way, and returns it.
  - Otherwise it returns the response unchanged.
  - *Why it is hidden:* no outside caller should set the state or trigger a dispense directly.
  - *Why the dispense follow-up lives here:* the state only says "go to dispensing". The machine runs the next step, so no state needs to know about another.
- *Extensibility:* a new state needs no change here, because the map is passed in.

## 7. Relationships
- Inventory **has** Slot (slots are created inside `add_slot`).
- Slot **uses** Product (the product exists on its own and can be shared).
- Session **uses** Slot (the slot lives on in the inventory).
- VendingMachine **uses** Inventory, CashBox, MachineState (all created in setup and passed in).
- IdleState **uses** Session.
- HasMoneyState **uses** Session.
- DispensingState **uses** Session, CashBox.
- IdleState, HasMoneyState, DispensingState, OutOfServiceState **implement** MachineState.
- Enums are always **uses**: Response to MachineStateType, VendingMachine to MachineStateType.
- `Response` is only a return type, so it has no line from the states or the machine, because no attribute holds it.

## 8. Interaction

### Layers
1. **Entry point and manager:** VendingMachine. The only class outside code calls.
2. **Behavior:** the four states. Each decides what an action does right now.
3. **Data owners:** Inventory, Slot, Session, CashBox. Each guards its own fact.
4. **Plain data:** Product and Response.

### Call chains
- **Insert money**
  1. `User -> VendingMachine.insert_money(amount)`: the single entry point.
  2. `VendingMachine -> current state.insert_money(amount)`: only the state knows if money is accepted now.
  3. `State -> Session.add_money(amount)`: the session owns the balance.
  4. `VendingMachine.finish(response)`: apply the next state.
- **Select a product**
  1. `User -> VendingMachine.select_product(slot_code)`.
  2. `VendingMachine -> Inventory.get_slot(slot_code)`: the one lookup, before any state runs.
  3. `VendingMachine -> current state.select_product(slot)`.
  4. `HasMoneyState -> Slot.is_in_stock()`: refuse when out of stock.
  5. `HasMoneyState -> Session.select_slot(slot)`, then `Session.can_afford(price)`: decide whether to start the sale.
  6. `VendingMachine.finish(response)`: apply the next state.
- **Automatic dispense (after `finish` sees `DISPENSING`)**
  1. `VendingMachine -> DispensingState.dispense()`: the state now runs the sale.
  2. `DispensingState -> Session.can_afford(price)`, `Slot.take_one()`: the steps that can fail come first.
  3. `DispensingState -> Session.take_payment(price)`: deduct and clear in one step.
  4. `DispensingState -> CashBox.add(price)`: the step that can't fail comes last.
- **Cancel**
  1. `User -> VendingMachine.cancel()`.
  2. `VendingMachine -> current state.cancel()`.
  3. `HasMoneyState -> Session.refund_all()`: the session returns and clears the money.
- **Restock**
  1. `Operator -> VendingMachine.restock(slot_code, count)`.
  2. `VendingMachine -> current state.allows_operator_actions()`: the state decides if it is safe.
  3. `VendingMachine -> Inventory.restock(slot_code, count)`, then `Slot.restock(count)`.
- **Collect money**
  1. `Operator -> VendingMachine.collect_money()`.
  2. `VendingMachine -> current state.allows_operator_actions()`.
  3. `VendingMachine -> CashBox.collect()`.

### Who may call whom
- Users and operators call `VendingMachine` only, and never the states, `Session`, `Slot` or `CashBox`.
- `VendingMachine` calls `Inventory`, `CashBox`, and the current state.
- States call `Session`, `Slot` and (the dispensing state) `CashBox`.
- States must never call `VendingMachine` or another state. This is what keeps dependencies one-way.
- `Slot`, `Session`, `CashBox` and `Product` call no other class.

## 9. Flows

### Insert money
1. `insert_money(amount: int)` on the machine, which hands it to the current state. *Why:* the state decides.
2. The state calls `session.add_money(amount)`. *Why:* the session refuses bad amounts.
3. Return `Response(accepted, state_after, None, 0)`.
- **Failure:** `amount <= 0`, or the state is not Idle or HasMoney. The response is refused, `state_after` is the same state, and nothing changes. There is nothing to undo.

### Select a product
1. `select_product(slot_code: str)`. The machine does `slot: Slot or None = inventory.get_slot(slot_code)`. *Why first:* one lookup, then objects only.
2. The state checks the slot exists and `slot.is_in_stock()`. *Why before changing anything:* a refusal then leaves everything as it was.
3. `session.select_slot(slot)`.
4. If `session.can_afford(price)`, the response says `DISPENSING` and `finish` runs the sale. Otherwise it stays `HAS_MONEY` and waits.
- **Failure:** unknown code, out of stock, or wrong state. The response is refused and the money stays in the session, so the customer can choose again or cancel.

### Dispense (automatic)
1. The five sale steps in `DispensingState`, in the order listed in section 6.
2. Return `Response(True, IDLE, product name, change)`.
- **Failure and undo:**
  - Step 1 or 2 fails: nothing was changed, so the state goes back to `HAS_MONEY` and the money stays.
  - Step 3 fails: the one item already taken is put back with `slot.restock(1)`, and the state goes back to `HAS_MONEY`.
  - *Why this is the right undo:* only the stock was changed before step 3, so putting that back returns everything to its starting point.

### Cancel
1. `cancel()` on the machine, which hands it to the current state.
2. In `HasMoneyState`: `session.refund_all()` returns the whole balance and clears the selection.
3. Return `Response(True, IDLE, None, refund)`.
- **Failure:** in Idle, Dispensing or OutOfService it is refused, and nothing changes. There is nothing to refund, or a sale is running.

### Restock
1. `restock(slot_code: str, count: int)` on the machine.
2. If `current state.allows_operator_actions()` is False, return False.
3. `inventory.restock(slot_code, count)` finds the slot once and asks it to restock.
- **Failure:** wrong state, unknown code, or `count <= 0`. It returns False and the quantity is unchanged.

### Collect money
1. `collect_money()` on the machine.
2. If the current state does not allow operator actions, return None.
3. Return `cash_box.collect()`, which also resets the total to 0.
- **Failure:** wrong state returns None and the total stays.

### Take out of service and return to service
1. `take_out_of_service()` is accepted only in Idle. `return_to_service()` is accepted only in OutOfService.
- **Failure:** any other state refuses and stays as it was. A customer's money is never stranded, because HasMoney refuses until the customer cancels.

### Concurrency
- No requirement mentions concurrent use, so no lock is added.
- The `DISPENSING` state is not a lock. It is a state that refuses every action while the sale runs.
- If concurrency were required later, one lock inside `VendingMachine` around every public method would be enough, because all state is reachable only through it.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| More products and slots | Call `add_slot` with a new Product | No. It is data. |
| Pay by card | A new `CardReader` class that charges the card and calls `insert_money(amount)` | No. |
| Give change in coins | A new `ChangeMaker` class that turns `money_returned` into coin counts | No, as long as coins never run out. |
| A new state, such as MAINTENANCE | A new state class, one new enum value, one new map entry | The one state that moves into it must return the new type, so that return value changes. This is the cost of keeping transitions inside states. |

## 11. Interview Notes

### Key decisions
1. **"In stock" is computed, not stored.**
   - *Why:* quantity is the only stock fact.
   - *Rejected:* a `SlotStatus` enum next to quantity. The two would disagree the moment someone forgot to update one.
2. **State pattern, with states returning a Response.**
   - *Why:* the machine reacts to the label and never exposes a state setter.
   - *Rejected:* an `if` on the state in every method, and states calling a public `set_state` on the machine (anyone could call it).
3. **A separate Session for the customer's money and selection.**
   - *Why:* it is one fact with a check-and-clear method, and states can share it.
   - *Rejected:* balance and selection as machine fields. The machine would hold changing data and mutate it in many methods.
4. **Money first, then selection.**
   - *Why:* it needs no storage of a selection that has no money behind it.
   - *Rejected:* selecting in Idle. It would need another place to keep a pre-money selection.
5. **One `take_payment` method that checks, deducts and clears.**
   - *Why:* a half-cleared session can't exist.
   - *Rejected:* separate deduct and clear calls.

### Edge cases handled
- Exact payment gives change 0. Overpayment gives change.
- Selecting a second product in HasMoney replaces the first.
- Not enough money waits, and later money triggers the sale.
- Out-of-stock selection is refused and the money stays.
- Zero or negative money is refused.
- Restock and collect are refused during a customer's session.
- OutOfService refuses money.

### Likely follow-up questions
1. **"Why is Dispensing a state if the sale is instant?"** It models the physical dispensing time and refuses every action while the sale runs, so nothing can interrupt it.
2. **"How do you give change in coins?"** A `ChangeMaker` class converts the change amount into coins. If coins can run out, the check must go before step 2 of the sale, which means a small edit in `DispensingState`.
3. **"What if two people use it at once?"** One lock inside `VendingMachine` around every public method, since every state is reachable only through it.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Coin and note denominations | Money is one whole-number amount, and change is assumed payable. `ChangeMaker` would add coins. |
| Operator roles and login | No requirement defines them. The four operator methods are only allowed while no customer is mid-purchase. |
| Showing the product list or balance | No requirement asks for a display, so it is a read-only view that can be added. |
| Timeouts for money left in the machine | Money stays until the customer cancels. A timeout would be a new rule in HasMoneyState. |
| Hardware faults such as a jammed item | The sale steps are all-or-nothing in software. A jam would need a hardware step added before step 4. |
| Going out of service when empty | The operator takes it out of service. An automatic rule would be a new transition. |
| Selecting before inserting money | It is refused in Idle by design. |
| Concurrent customers | Section 9 gives the one-lock answer. |

## 13. Requirements Met
1. Products in slots with price and quantity: `Inventory`, `Slot`, `Product`, and `VendingMachine.add_slot`.
2. Select a product and insert money: `VendingMachine.select_product` and `insert_money`, handled by `HasMoneyState` and `Session`.
3. Behavior depends on the condition: `MachineState` with `IdleState`, `HasMoneyState`, `DispensingState`, `OutOfServiceState`.
4. Dispense once enough money and return change: `DispensingState.dispense` with `Session.can_afford` and `take_payment`.
5. Cancel and refund: `HasMoneyState.cancel` with `Session.refund_all`.
6. Refuse selection when out of stock: `HasMoneyState.select_product` with `Slot.is_in_stock`.
7. Restock and collect money: `VendingMachine.restock` and `collect_money`, with `Slot.restock` and `CashBox.collect`.

## 14. Requirement Fit Check

**Setup used in every example:** slot `A1` holds Cola (price 150) with quantity 2. Slot `B2` holds Chips (price 100) with quantity 0. The machine starts in `IDLE`, and the cash box is empty.

1. **Products in slots**
   - *Success:* `add_slot("A1", Cola, 2)` returns True, then `add_slot("B2", Chips, 0)` returns True. The inventory now has two slots with their own prices and quantities.
   - *Failure:* `add_slot("A1", Tea, 5)` returns False because the code exists. Nothing changes.
2. **Select a product and insert money**
   - *Success:* `insert_money(200)` in Idle is accepted, the state is `HAS_MONEY`, and the balance is 200. `select_product("A1")` is accepted.
   - *Failure:* `select_product("A1")` in Idle is refused and the state stays `IDLE`. `insert_money(0)` is refused and nothing changes.
3. **Behavior depends on the condition**
   - *Success:* `take_out_of_service()` in Idle is accepted, the state is `OUT_OF_SERVICE`, and `return_to_service()` brings it back to `IDLE`.
   - *Failure:* in `OUT_OF_SERVICE`, `insert_money(100)` is refused and the state stays. In `HAS_MONEY`, `take_out_of_service()` is refused.
4. **Dispense and return change**
   - *Success (select first):* `insert_money(200)`, `select_product("A1")`. The balance 200 covers 150, so the state goes to `DISPENSING`. The sale runs: `take_one` makes A1's quantity 1, `take_payment(150)` returns change 50, the cash box becomes 150. The response is `(True, IDLE, "Cola", 50)`.
   - *Success (money later):* `insert_money(100)`, `select_product("A1")` waits in `HAS_MONEY` because 100 < 150. `insert_money(50)` makes the balance 150, so it dispenses with change 0.
   - *Failure:* `insert_money(100)`, `select_product("A1")`. It is accepted but waits. Quantity is still 2, nothing is taken, and the balance is 100.
5. **Cancel and refund**
   - *Success:* `insert_money(200)`, then `cancel()` returns `(True, IDLE, None, 200)` and the balance is 0.
   - *Failure:* `cancel()` in Idle is refused. Nothing changes.
6. **Out of stock**
   - *Success (the refusal):* `insert_money(100)`, then `select_product("B2")`. B2's quantity is 0, so the selection is refused. The state stays `HAS_MONEY` and the balance stays 100, so `cancel()` returns 100.
   - *Failure case to check:* `select_product("Z9")` returns a refusal because the lookup is None. Nothing changes.
7. **Restock and collect**
   - *Success:* in `IDLE`, `restock("B2", 5)` returns True and B2's quantity becomes 5. After the sale in example 4, `collect_money()` returns 150 and the cash box resets to 0.
   - *Failure:* in `HAS_MONEY`, `restock("B2", 5)` returns False and `collect_money()` returns None. `restock("Z9", 3)` and `restock("B2", 0)` return False. Quantities and totals are unchanged.
