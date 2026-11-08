# Food Delivery App (Swiggy/Zomato): Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** an order is a record that runs its own stages and tells its watchers at every change, so no path can change an order without the user being told. Scarce things are guarded where they live: a delivery partner is reserved before money is taken, so a paid order always has someone to deliver it, and every failure after that point has a clear undo.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. Users should be able to browse and search restaurants and their menus.
2. Users should be able to add items from one restaurant to a cart.
3. Users should be able to place an order with a delivery address and pay for it.
4. A restaurant should be able to accept or reject an order and mark it as being prepared.
5. The system should assign a delivery partner to each order.
6. An order should move through the stages of placed, accepted, preparing, picked up, delivered and cancelled, and the user should be notified at each change.
7. Users should be able to track their order.
8. The system should support coupons and discounts on the order total.
9. The system should support multiple payment methods.
10. Users should be able to rate the restaurant and the delivery partner.

**Types and conventions**
- Money is `int`, in the smallest currency unit. Quantities and counts are `int`.
- Users are identified only by a `user_id: str`. No requirement asks for registration, so there is no User class.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- No locks are used, because no requirement mentions concurrent use. The reservation of a partner is still a checked method, so a lock inside `DeliveryPartner` would be the only change needed (section 9).

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| MenuItem | Describes one dish | id, name, price |
| ReputationScore | Adds ratings into an average | total stars, count |
| Restaurant | Holds a menu and a rating, and says if it matches a search | the menu, rating |
| RestaurantCatalog | Finds restaurants by id or by search text | the restaurants |
| Cart | Holds one user's chosen items from one restaurant | restaurant, item quantities |
| OrderLine | Records one ordered dish at the price it was bought | name, price, quantity |
| DeliveryPartner | Is a delivery person who guards their own availability | status, rating |
| DiscountStrategy (interface) | Turns an order subtotal into a discount | none |
| PercentDiscount | Takes a percentage off, up to a cap | percent, cap |
| FlatDiscount | Takes a fixed amount off | amount |
| Coupon | Joins a code, a minimum order and a discount rule | code, minimum, rule |
| PaymentMethod (interface) | Takes and returns a payment | none |
| CardPayment | Pays and refunds by card | amount paid |
| UpiPayment | Pays and refunds by UPI | amount paid |
| OrderObserver (interface) | Reacts to an order changing stage | none |
| UserNotifier | Keeps the messages sent to each user | the messages |
| OrderTracking | Gives a read-only picture of one order | a copy of key facts |
| Order | Holds one order and runs its own stages, refunds and ratings | status, partner, payment |
| OrderService | Runs the ordering flow and holds the carts, orders, partners and coupons | carts, orders, partners, coupons |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** User, restaurant, menu, item, cart, order, delivery address, payment, delivery partner, stage, coupon, discount, rating.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Restaurant, MenuItem, Cart, Order, DeliveryPartner, Coupon are classes. Delivery address and rating are values.
2. **Find fixed sets of values.** The order stages and the partner's availability.
   - *Why:* a fixed set is an enum, so a typo like "delivred" cannot exist.
   - *Result:* `OrderStatus`, `PartnerStatus`.
3. **Give every fact one owner.** The menu belongs to the Restaurant, the cart's contents to the Cart, the stage, partner and payment of an order to the Order, a partner's availability to the DeliveryPartner.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** The order total, "is the order paid", and a rating average are all computed.
   - *Why:* a stored total beside the lines would go wrong the moment one changes.
5. **Give every verb to the class whose data changes.** Accepting, preparing and cancelling change the Order. Reserving a partner changes the DeliveryPartner.
6. **Find the action that touches several classes.** Placing an order needs the cart, a coupon, a partner and a payment, in an order that can be undone. That is `OrderService.place_order`.
7. **Find the hint phrases for patterns.** "The user should be notified at each change" is Observer. "Coupons and discounts" is Strategy for the discount rule. "Multiple payment methods" is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 10).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Fix the outputs.** An `Order` or None for placing, a `bool` for every command, and an `OrderTracking` for tracking.

## 4. Pattern Guide

### Used: Observer (order changes)
- **Meaning:** an object keeps a list of watchers and tells every one of them when something changes, without knowing what they do.
- **Hint phrase:** "the user should be notified at each change".
- **Without it:** every method that changes an order would have to call the notification code, and a missed call would leave a change unannounced.
- **When NOT to use it:** when only one party ever cares.

### Used: Strategy (discounts and payment)
- **Meaning:** a way of doing something is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrases:** "support coupons and discounts on the order total" and "support multiple payment methods".
- **Without it:** the coupon and the order would each have an `if` on the kind, and every new kind would mean editing them.
- **When NOT to use it:** when there is one fixed way and nobody asked to add another.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. The stages only decide which method is allowed, and each method checks the status itself. |
| Factory | "create different kinds of objects by type" | Not used. |
| Facade | "a single simple entry point" | Not added as a separate class. `OrderService` and `RestaurantCatalog` are already the entry points. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### OrderStatus
- Values: `PLACED`, `ACCEPTED`, `PREPARING`, `PICKED_UP`, `DELIVERED`, `CANCELLED`
- *Why:* exactly the six stages named in requirement 6. A restaurant rejection ends as `CANCELLED`.

### PartnerStatus
- Values: `AVAILABLE`, `BUSY`
- *Why:* requirement 5 assigns a partner to each order, so the system must know who is free.

## 6. Classes (smallest dependency first)

### MenuItem
**Why this class exists:** a dish is a thing with its own identity and price. It is separate from Restaurant so the menu is a plain collection of dishes.
- Attributes: `item_id: str`, `name: str`, `price: int`. *Why:* the identity, the display name, and the price of one unit.
- Constructor: `MenuItem(item_id: str, name: str, price: int)`. *Why:* all set once. It refuses an empty id or name, or `price <= 0`.
- Methods: `get_item_id() -> str`, `get_name() -> str`, `get_price() -> int`. *Why:* read-only views. There are no setters.
- *Extensibility:* more fields (a description, a photo) are added here.

### ReputationScore
**Why this class exists:** restaurants and partners are rated, and the averaging rule is needed twice. One class means one place for the rule.
- Attributes: `total_stars: int`, `count: int`. *Why:* a sum and a count are enough to compute an average.
- Constructor: `ReputationScore()`. *Why:* starts at 0 and 0.
- Methods:
  - `add(stars: int) -> bool`. *Why:* refuses stars outside 1 to 5.
  - `get_average() -> float or None`. *Why:* computed as the total divided by the count. None if there are no ratings.
  - `get_count() -> int`. *Why:* read-only view.
- *Extensibility:* none needed.

### Restaurant
**Why this class exists:** requirement 1 browses restaurants and menus. A restaurant owns its menu and answers whether it matches a search.
- Attributes:
  - `restaurant_id: str`, `name: str`. *Why:* the identity and display name.
  - `menu: map{str: MenuItem}` (an item id to the MenuItem). *Why:* gives a lookup of a dish by id.
  - `reputation: ReputationScore`. *Why:* the restaurant's rating.
- Constructor: `Restaurant(restaurant_id: str, name: str)`. *Why:* a restaurant starts with an empty menu and no ratings. It refuses empty values.
- Methods:
  - `get_restaurant_id() -> str`, `get_name() -> str`. *Why:* read-only views.
  - `add_item(item_id: str, name: str, price: int) -> bool`. *Why:* the restaurant builds the MenuItem itself. False if the id exists or the values are invalid.
  - `get_menu() -> list[MenuItem]`. *Why:* requirement 1. It returns a copy of the menu in the order items were added.
  - `get_item(item_id: str) -> MenuItem or None`. *Why:* the lookup by id.
  - `matches(text: str) -> bool`. *Why:* computed. True if the text appears (ignoring case) in the restaurant's name or in any dish name. False for empty text.
  - `add_rating(stars: int) -> bool`, `get_average_rating() -> float or None`. *Why:* the only way the rating changes or is read.
- *Extensibility:* none needed.

### RestaurantCatalog
**Why this class exists:** browsing and searching are about the whole set of restaurants, which no single restaurant knows.
- Attributes: `restaurants: map{str: Restaurant}` (a restaurant id to the Restaurant). *Why:* the one record of restaurants.
- Constructor: `RestaurantCatalog()`. *Why:* it starts with an empty map.
- Methods:
  - `add_restaurant(restaurant_id: str, name: str) -> Restaurant or None`. *Why:* None if the id exists.
  - `get_restaurant(restaurant_id: str) -> Restaurant or None`. *Why:* the lookup by id.
  - `list_all() -> list[Restaurant]`. *Why:* browsing. A copy, sorted by name.
  - `search(text: str) -> list[Restaurant]`. *Why:* the restaurants for which `matches(text)` is True, sorted by name. Empty if nothing matches.
- *Extensibility:* none needed.

### Cart
**Why this class exists:** requirement 2 gathers items from one restaurant before ordering. The cart owns what is in it and refuses a second restaurant.
- Attributes:
  - `user_id: str`. *Why:* whose cart it is.
  - `restaurant: Restaurant or None`. *Why:* the one restaurant the items come from. None while empty.
  - `quantities: map{str: int}` (an item id to the quantity wanted). *Why:* it holds only ids and counts, so prices are read from the live menu at order time and never go stale.
- Constructor: `Cart(user_id: str)`. *Why:* starts empty. It refuses an empty id.
- Methods:
  - `get_restaurant() -> Restaurant or None`, `get_quantities() -> map{str: int}` (returns a copy), `is_empty() -> bool`. *Why:* read-only views.
  - `add_item(restaurant: Restaurant, item_id: str, quantity: int) -> bool`. *Why:* False if `quantity <= 0`, the item is not on that restaurant's menu, or the cart already holds another restaurant's items. An item added again adds to its quantity.
  - `remove_item(item_id: str) -> bool`. *Why:* False if it is not in the cart. When the last item goes, the restaurant becomes None.
  - `clear() -> None`. *Why:* empties the cart after an order. It cannot fail.
- *Extensibility:* none needed.

### OrderLine
**Why this class exists:** an order must keep the dish name and price as they were when it was bought, even if the menu changes later.
- Attributes: `item_name: str`, `unit_price: int`, `quantity: int`. *Why:* a snapshot of what was ordered.
- Constructor: `OrderLine(item_name: str, unit_price: int, quantity: int)`. *Why:* all set once. It refuses `unit_price <= 0` or `quantity <= 0`.
- Methods: `get_item_name() -> str`, `get_unit_price() -> int`, `get_quantity() -> int`, `get_line_total() -> int`. *Why:* `get_line_total` is computed as the price times the quantity.
- *Extensibility:* none needed.

### DeliveryPartner
**Why this class exists:** requirement 5 assigns one partner to each order, so a partner must never be given two orders at once. The partner owns that decision.
- Attributes:
  - `partner_id: str`, `name: str`. *Why:* the identity and display name.
  - `status: PartnerStatus`. *Why:* the single fact about availability.
  - `reputation: ReputationScore`. *Why:* the partner's rating.
- Constructor: `DeliveryPartner(partner_id: str, name: str)`. *Why:* a new partner starts `AVAILABLE` with no ratings. It refuses empty values.
- Methods:
  - `get_partner_id() -> str`, `get_name() -> str`, `get_status() -> PartnerStatus`. *Why:* read-only views.
  - `try_reserve() -> bool`. *Why:* `AVAILABLE` to `BUSY`. False otherwise. This is what stops a partner being given two orders.
  - `release() -> bool`. *Why:* `BUSY` to `AVAILABLE`. False if already available.
  - `add_rating(stars: int) -> bool`, `get_average_rating() -> float or None`. *Why:* the only way the rating changes or is read.
- *Extensibility:* none needed.

### DiscountStrategy (interface)
**Why it is an interface:** requirement 8 supports coupons and discounts, which come in several kinds, so a coupon depends on one method and not on a specific kind.
- Methods: `discount_for(subtotal: int) -> int`. *Why:* the amount to take off the subtotal.
- *Extensibility:* a new kind (buy one get one) is a new class implementing this method.

### PercentDiscount (implements DiscountStrategy)
**Why this class exists:** the usual coupon: a percentage off, with a cap.
- Attributes: `percent: int`, `max_discount: int`. *Why:* the percentage and the most it may take off.
- Constructor: `PercentDiscount(percent: int, max_discount: int)`. *Why:* both set once. It refuses a percent outside 1 to 100 or `max_discount <= 0`.
- Methods: `discount_for(subtotal: int) -> int`. *Why:* `min(subtotal * percent // 100, max_discount)`.
- *Extensibility:* none needed.

### FlatDiscount (implements DiscountStrategy)
**Why this class exists:** the second kind: a fixed amount off.
- Attributes: `amount: int`. *Why:* what is taken off.
- Constructor: `FlatDiscount(amount: int)`. *Why:* set once. It refuses `amount <= 0`.
- Methods: `discount_for(subtotal: int) -> int`. *Why:* `min(amount, subtotal)`, so a discount never goes below zero total.
- *Extensibility:* none needed.

### Coupon
**Why this class exists:** a coupon joins a code, a minimum order and a discount rule. It owns the minimum-order rule, so the order does not.
- Attributes: `code: str`, `min_order_total: int`, `strategy: DiscountStrategy`. *Why:* the code a user types, the smallest subtotal it works on, and how the discount is worked out.
- Constructor: `Coupon(code: str, min_order_total: int, strategy: DiscountStrategy)`. *Why:* all set once. It refuses an empty code or a negative minimum.
- Methods: `get_code() -> str`. *Why:* read-only view.
  - `calculate_discount(subtotal: int) -> int or None`. *Why:* None if `subtotal < min_order_total`. Otherwise the strategy's discount, never more than the subtotal.
- *Extensibility:* a new coupon is data.

### PaymentMethod (interface)
**Why it is an interface:** requirement 9 asks for multiple payment methods, and a cancelled order needs the money back.
- Methods: `pay(amount: int) -> bool`, `refund(amount: int) -> bool`. *Why:* a `bool` is all the order needs to know.
- *Extensibility:* a new method is a new class implementing both.

### CardPayment (implements PaymentMethod)
**Why this class exists:** the first way to pay.
- Attributes: `card_number: str`, `paid_amount: int`. *Why:* the card to charge, and how much was paid, so a refund can only return what was paid.
- Constructor: `CardPayment(card_number: str)`. *Why:* `paid_amount` starts at 0. It refuses an empty number. One object is used for one order.
- Methods:
  - `pay(amount: int) -> bool`. *Why:* False if `amount <= 0` or it was already paid. This is where a real card gateway call goes, and a decline returns False.
  - `refund(amount: int) -> bool`. *Why:* True only if `paid_amount >= amount > 0`, then it reduces `paid_amount`.
- *Extensibility:* none needed.

### UpiPayment (implements PaymentMethod)
**Why this class exists:** the second implementation, which shows the interface holds more than one way to pay.
- Attributes: `upi_id: str`, `paid_amount: int`. *Why:* the same as the card.
- Constructor: `UpiPayment(upi_id: str)`. *Why:* `paid_amount` starts at 0. It refuses an empty id.
- Methods: `pay(amount: int) -> bool` and `refund(amount: int) -> bool`, with the same rules as the card.
- *Extensibility:* none needed.

### OrderObserver (interface)
**Why it is an interface:** requirement 6 notifies the user at each change, and others may want to listen later, so the order depends on one method and not on any watcher.
- Methods: `on_status_changed(order_id: str, user_id: str, status: OrderStatus) -> None`. *Why these inputs:* plain ids and the new stage, so a watcher never holds the order itself. A watcher that fails must not stop an order from changing, so the order guards each call.
- *Extensibility:* a new watcher is a new class implementing this method.

### UserNotifier (implements OrderObserver)
**Why this class exists:** the first watcher: it keeps the messages sent to each user.
- Attributes: `messages: map{str: list[str]}` (a user id to the messages sent to that user, oldest first). *Why:* the one record of what each user was told. A real push or SMS service would be called here.
- Constructor: `UserNotifier()`. *Why:* it starts with an empty map.
- Methods:
  - `on_status_changed(order_id: str, user_id: str, status: OrderStatus) -> None`. *Why:* adds a message such as `Order O1: PICKED_UP`.
  - `get_messages(user_id: str) -> list[str]`. *Why:* read-only view. A copy. Empty if none.
- *Extensibility:* none needed.

### OrderTracking
**Why this class exists:** requirement 7 shows a user how their order stands without handing out the order itself, which could be changed.
- Attributes: `order_id: str`, `status: OrderStatus`, `restaurant_name: str`, `partner_name: str or None`, `total: int`. *Why:* what a user wants to see. The partner name is None only if the order is cancelled before pick-up.
- Constructor: `OrderTracking(order_id: str, status: OrderStatus, restaurant_name: str, partner_name: str or None, total: int)`. *Why:* all set once.
- Methods: `get_order_id() -> str`, `get_status() -> OrderStatus`, `get_restaurant_name() -> str`, `get_partner_name() -> str or None`, `get_total() -> int`. *Why:* read-only views.
- *Extensibility:* a new field is added here.

### Order
**Why this class exists:** an order is the record of one purchase and its stage. It runs its own transitions and tells its watchers, so a direct call from anywhere is still safe and still announced.
- Attributes:
  - `order_id: str`, `user_id: str`. *Why:* the identity and the buyer.
  - `restaurant: Restaurant`. *Why:* who is cooking.
  - `lines: list[OrderLine]`. *Why:* what was ordered, as a snapshot.
  - `delivery_address: str`. *Why:* where to deliver.
  - `coupon_code: str or None`, `discount: int`. *Why:* the coupon used and what it took off. Stored once at placement, because the coupon's rules may change later.
  - `status: OrderStatus`. *Why:* the stage.
  - `partner: DeliveryPartner`. *Why:* the assigned partner, reserved before the order exists.
  - `payment: PaymentMethod or None`. *Why:* what paid for it, kept so a cancel can refund. None if the total was 0.
  - `observers: list[OrderObserver]`. *Why:* who to tell at each change.
  - `restaurant_rated: bool`, `partner_rated: bool`. *Why:* so each rating can be given once.
- Constructor: `Order(order_id: str, user_id: str, restaurant: Restaurant, lines: list[OrderLine], delivery_address: str, coupon_code: str or None, discount: int, partner: DeliveryPartner, payment: PaymentMethod or None, observers: list[OrderObserver])`. *Why:* the partner is already reserved and the payment already taken by the caller. The status starts `PLACED`, the order keeps its own copies of the lists, and it tells the observers it is placed. It refuses an empty address, no lines, or a discount that is more than the subtotal.
- Methods:
  - `get_order_id() -> str`, `get_user_id() -> str`, `get_status() -> OrderStatus`, `get_partner() -> DeliveryPartner`, `get_restaurant() -> Restaurant`. *Why:* read-only views.
  - `get_subtotal() -> int`, `get_total() -> int`. *Why:* computed. The subtotal is the sum of the line totals, and the total is the subtotal minus the discount.
  - `track() -> OrderTracking`. *Why:* builds the read-only picture.
  - `accept(restaurant_id: str) -> bool`. *Why:* `PLACED` only, and only the order's restaurant. Sets `ACCEPTED`.
  - `reject(restaurant_id: str) -> bool`. *Why:* `PLACED` only, and only the order's restaurant. It cancels the order (see `cancel`'s refund steps).
  - `start_preparing(restaurant_id: str) -> bool`. *Why:* `ACCEPTED` only, and only the order's restaurant. Sets `PREPARING`.
  - `pick_up(partner_id: str) -> bool`. *Why:* `PREPARING` only, and only the assigned partner. Sets `PICKED_UP`.
  - `deliver(partner_id: str) -> bool`. *Why:* `PICKED_UP` only, and only the assigned partner. It releases the partner and sets `DELIVERED`.
  - `cancel(user_id: str) -> bool`. *Why:* only the buyer, and only in `PLACED` or `ACCEPTED`. Once cooking starts it can't be cancelled. It refunds the payment (False if the refund fails), releases the partner, and sets `CANCELLED`.
  - `rate_restaurant(user_id: str, stars: int) -> bool`, `rate_partner(user_id: str, stars: int) -> bool`. *Why:* only the buyer, only after `DELIVERED`, each at most once. The flag is set only if the rating was accepted.
- *Why every stage change tells the observers:* it is the last step of each method, so no change is silent. A failing observer is skipped and the change stands.
- *Extensibility:* none needed.

### OrderService (the manager and only entry point)
**Why this class exists:** placing an order needs the cart, a coupon, a partner and a payment, in an order that can be undone. No single class can do that, and carts, orders and partners need one place to be found.
- Attributes:
  - `catalog: RestaurantCatalog`. *Why:* to find a restaurant by id.
  - `carts: map{str: Cart}` (a user id to that user's Cart), `orders: map{str: Order}` (an order id to the Order), `partners: map{str: DeliveryPartner}` (a partner id to the partner), `coupons: map{str: Coupon}` (a code to the Coupon). *Why:* the one record of each.
  - `observers: list[OrderObserver]`. *Why:* handed to each new Order.
  - `next_order_number: int`. *Why:* makes order ids unique.
- Constructor: `OrderService(catalog: RestaurantCatalog, observers: list[OrderObserver])`. *Why:* both are passed in, so tests can use fakes. It starts with no carts, orders, partners or coupons.
- Methods (outer ones take ids and look up once):
  - `add_partner(partner_id: str, name: str) -> DeliveryPartner or None`. *Why:* None if the id exists.
  - `add_coupon(code: str, min_order_total: int, strategy: DiscountStrategy) -> bool`. *Why:* False if the code exists.
  - `add_to_cart(user_id: str, restaurant_id: str, item_id: str, quantity: int) -> bool`. *Why:* looks up the restaurant once and the user's cart (creating it if needed), then asks the cart. False if the restaurant is unknown or the cart refuses.
  - `remove_from_cart(user_id: str, item_id: str) -> bool`. *Why:* looks up the cart once and asks it.
  - `view_cart(user_id: str) -> list[OrderLine] or None`. *Why:* the lines at current menu prices, computed. None if the cart is empty or missing.
  - `place_order(user_id: str, delivery_address: str, payment: PaymentMethod, coupon_code: str or None) -> Order or None`. *Why:* requirements 3, 5 and 8. None means nothing was changed.
  - `respond_to_order(order_id: str, restaurant_id: str, accept: bool) -> bool`, `start_preparing(order_id: str, restaurant_id: str) -> bool`, `pick_up(order_id: str, partner_id: str) -> bool`, `deliver(order_id: str, partner_id: str) -> bool`, `cancel_order(order_id: str, user_id: str) -> bool`. *Why:* each looks up the order once and asks it. The order checks that the right person is acting.
  - `track_order(order_id: str, user_id: str) -> OrderTracking or None`. *Why:* requirement 7. None if the order is unknown or belongs to another user.
  - `rate_restaurant(order_id: str, user_id: str, stars: int) -> bool`, `rate_partner(order_id: str, user_id: str, stars: int) -> bool`. *Why:* requirement 10, looked up once and handed to the order.
- Private helper: `reserve_partner() -> DeliveryPartner or None`. *Why hidden:* no outside caller should reserve a partner without making an order. It tries partners in id order and returns the first for which `try_reserve()` is True.
- *Extensibility:* coupons, partners and restaurants are data. The watchers are passed in.

## 7. Relationships
- RestaurantCatalog **has** Restaurant (created inside `add_restaurant`).
- Restaurant **has** MenuItem, ReputationScore. DeliveryPartner **has** ReputationScore.
- Order **has** OrderLine (the lines die with the order).
- OrderService **has** Cart, Order, DeliveryPartner, Coupon (created inside the add, cart and place methods).
- Cart **uses** Restaurant (the restaurant lives on without the cart).
- Order **uses** Restaurant, DeliveryPartner, PaymentMethod, OrderObserver.
- Coupon **uses** DiscountStrategy (passed in).
- OrderService **uses** RestaurantCatalog, OrderObserver (passed in).
- PercentDiscount, FlatDiscount **implement** DiscountStrategy. CardPayment, UpiPayment **implement** PaymentMethod. UserNotifier **implements** OrderObserver.
- Enums are always **uses**: Order to OrderStatus, DeliveryPartner to PartnerStatus.
- `OrderTracking` appears only as a return type, so no attribute holds it and it has no line.

## 8. Interaction

### Layers
1. **Entry points and managers:** RestaurantCatalog (browsing) and OrderService (ordering).
2. **Domain classes that guard themselves:** Order, DeliveryPartner, Cart.
3. **Rules passed in:** DiscountStrategy, PaymentMethod and OrderObserver with their implementations.
4. **Plain data:** MenuItem, OrderLine, ReputationScore, OrderTracking.

### Call chains
- **Browse and search**
  1. `User -> RestaurantCatalog.search(text)` or `list_all()`.
  2. `RestaurantCatalog -> Restaurant.matches(text)`, then `Restaurant.get_menu()`.
- **Fill the cart**
  1. `User -> OrderService.add_to_cart(user_id, restaurant_id, item_id, quantity)`.
  2. `OrderService -> RestaurantCatalog.get_restaurant(restaurant_id)`: the one lookup.
  3. `OrderService -> Cart.add_item(restaurant, item_id, quantity)`.
- **Place an order**
  1. `User -> OrderService.place_order(...)`.
  2. `OrderService -> Cart`, then `Restaurant.get_item(...)` for each entry: lines at live prices.
  3. `OrderService -> Coupon.calculate_discount(subtotal)`, then `DiscountStrategy.discount_for(subtotal)`.
  4. `OrderService -> DeliveryPartner.try_reserve()`: the scarce thing is secured before money moves.
  5. `OrderService -> PaymentMethod.pay(total)`, then `Order(...)`, which calls `OrderObserver.on_status_changed(...)`.
- **Restaurant and partner steps**
  1. `Restaurant -> OrderService.respond_to_order(...)`, then `Order.accept(restaurant_id)` or `Order.reject(restaurant_id)`.
  2. `Order -> PaymentMethod.refund(total)` and `DeliveryPartner.release()` on a rejection.
  3. `Order -> OrderObserver.on_status_changed(...)` after every change.

### Who may call whom
- Users, restaurants and partners call `OrderService` and `RestaurantCatalog` only.
- `OrderService` calls `RestaurantCatalog`, `Cart`, `Coupon`, `DeliveryPartner` and `Order`.
- `Order` calls `DeliveryPartner`, `PaymentMethod` and the observers.
- `Cart` and `Coupon` call `Restaurant` and `DiscountStrategy`.
- Observers must never call back into `Order`. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Browse and search
1. `list_all()` and `search(text)` only read.
- **Failure:** no match returns an empty list. Nothing changes.

### Add to the cart
1. `add_to_cart(user_id: str, restaurant_id: str, item_id: str, quantity: int)`. `restaurant = catalog.get_restaurant(restaurant_id)`. If None, return False.
2. Find or create the user's cart, then `cart.add_item(restaurant, item_id, quantity)`.
- **Failure:** the cart refuses a second restaurant, a dish not on the menu, or a quantity of 0 or less, and nothing changes.

### Place an order
1. `place_order(user_id: str, delivery_address: str, payment: PaymentMethod, coupon_code: str or None)`. `cart = carts.get(user_id)`. If it is missing or empty, or the address is empty, return None.
2. Build `lines: list[OrderLine]` from the cart, reading each dish from the restaurant's current menu. If a dish is no longer on the menu, return None. `subtotal` is the sum of the line totals.
3. If a `coupon_code` was given: `coupon = coupons.get(code)`. If None, return None. `discount = coupon.calculate_discount(subtotal)`. If None (below the minimum), return None. Without a coupon the discount is 0. `total = subtotal - discount`.
4. `partner: DeliveryPartner or None = reserve_partner()`. If None, return None. *Why before paying:* a paid order must always have someone to deliver it.
5. If `total > 0`: `payment.pay(total)`. If False, call `partner.release()` and return None. *Undone:* only the partner. If `total` is 0, no payment is taken and the order keeps no payment.
6. Build the `Order`, store it in `orders`, call `cart.clear()`, and return the order. The order tells its observers it is placed.
- **Failure and undo:** steps 1 to 3 change nothing. A declined payment undoes the one change made so far, the partner reservation.

### Restaurant steps
1. `respond_to_order(order_id: str, restaurant_id: str, accept: bool)`. Look up the order once. If None, return False.
2. If `accept`: `order.accept(restaurant_id)`. Otherwise `order.reject(restaurant_id)`, which refunds the payment, releases the partner and sets `CANCELLED`.
3. `start_preparing(order_id, restaurant_id)` moves `ACCEPTED` to `PREPARING`.
- **Failure:** a restaurant that does not own the order, or the wrong stage, is refused. If a refund fails, the rejection is refused and the order stays `PLACED`.

### Pick up, deliver and cancel
1. `pick_up` moves `PREPARING` to `PICKED_UP`. `deliver` moves `PICKED_UP` to `DELIVERED` and releases the partner. `cancel_order` works only in `PLACED` or `ACCEPTED`, and refunds and releases.
- **Failure:** the wrong person or the wrong stage is refused with no change. An order being cooked can't be cancelled by the user.

### Track and rate
1. `track_order(order_id, user_id)` returns `order.track()` if the order belongs to that user.
2. `rate_restaurant` and `rate_partner` need `DELIVERED`, are allowed once each, and add the stars to the rated party.
- **Failure:** another user's order, a rating before delivery, a second rating, or stars outside 1 to 5 is refused. A bad star value does not use up the one rating.

### Concurrency
- No requirement mentions concurrent use, so no lock is added.
- `try_reserve` is already a checked step. If concurrency were required, one lock inside `DeliveryPartner` around its status changes would make it safe, and the order stages would need a lock inside `Order`.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new kind of coupon (buy one get one) | A new class implementing `DiscountStrategy`, used in a new `Coupon` | No. |
| A new payment method (wallet) | A new class implementing `PaymentMethod`, passed to `place_order` | No. |
| Another party told about order changes (the restaurant, analytics) | A new class implementing `OrderObserver`, passed to the service | No. |
| More restaurants, dishes, partners and coupons | Calls to the add methods | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **A partner is reserved before the payment is taken.**
   - *Why:* a paid order always has a partner, and the only undo is releasing one partner.
   - *Rejected:* paying first and assigning later. A paid order with no partner would need a refund.
2. **`Order` runs its own stages and tells its observers.**
   - *Why:* every change is announced, whoever calls it.
   - *Rejected:* the service calling the notifier after each change. A direct call on the order would be silent.
3. **The cart stores ids and counts, and prices are read from the menu at order time.**
   - *Why:* a price change can't leave a stale cart.
   - *Rejected:* a cart that copies prices.
4. **The order stores the discount it got, once.**
   - *Why:* the coupon's rules may change later, and the order must always show what the customer was actually charged.
   - *Rejected:* recomputing the discount from the coupon each time.
5. **The coupon owns its minimum-order rule, and a strategy owns the discount.**
   - *Why:* a new kind of discount needs only a new class.
   - *Rejected:* an `if` on the coupon kind inside the order.

### Edge cases handled
- A cart can hold only one restaurant's items.
- A dish removed from the menu after it was added refuses the order.
- A coupon that is unknown, or whose minimum is not met, refuses the order.
- A coupon never takes the total below zero. If the total is 0, no payment is taken.
- No free partner means no order and no payment.
- A declined payment releases the reserved partner.
- A rejection or a cancel refunds the payment and releases the partner.
- An order being cooked can't be cancelled.
- Another user can't track or cancel someone's order.
- A failing observer does not stop an order from changing.

### Likely follow-up questions
1. **"What if the restaurant never responds?"** Add a time limit and a sweep that rejects stale `PLACED` orders, the same pattern as expiring a hold. It is not built here because no requirement asks for it.
2. **"How would you pick the nearest partner?"** Replace `reserve_partner`'s id order with a matching rule that needs locations, passed in as a strategy. Today no requirement mentions distance.
3. **"What if two orders race for the last partner?"** `try_reserve` is one checked step. With real concurrency, a lock inside `DeliveryPartner` makes it atomic, and the loser tries the next partner or gets None.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| User accounts | Users are only an id. Accounts would be a new class. |
| Choosing the nearest partner | See follow-up question 2. |
| A time limit for the restaurant | See follow-up question 1. |
| A history of stage changes with times | `track_order` shows the current stage. Observers received each change, so a history observer could record them. |
| Coupon limits (one use per user) | Not required. It would be a rule inside `Coupon`. |
| Delivery fees and taxes | Not required. They would be added to the total computation in `Order`. |
| Several restaurants in one cart | Requirement 2 says one. |
| Real payment and notification services | `pay`, `refund` and the notifier are the one place each outside call goes. |

## 13. Requirements Met
1. Browse and search: `RestaurantCatalog.list_all` and `search`, `Restaurant.get_menu` and `matches`.
2. A cart from one restaurant: `Cart.add_item`, `OrderService.add_to_cart`.
3. Place an order with an address and pay: `OrderService.place_order`.
4. Accept, reject and prepare: `Order.accept`, `reject` and `start_preparing`.
5. Assign a delivery partner: `DeliveryPartner.try_reserve` and `OrderService.reserve_partner`.
6. The stages, with notifications: `OrderStatus`, the guarded methods of `Order`, and `OrderObserver` with `UserNotifier`.
7. Track an order: `OrderService.track_order` and `Order.track`.
8. Coupons and discounts: `Coupon`, `DiscountStrategy`, `PercentDiscount` and `FlatDiscount`.
9. Multiple payment methods: `PaymentMethod` with `CardPayment` and `UpiPayment`.
10. Rate the restaurant and partner: `Order.rate_restaurant`, `rate_partner`, and `ReputationScore`.

## 14. Requirement Fit Check

**Setup used in every example:** restaurant `R1` "Spice Hub" has dishes `I1` "Paneer Tikka" at 250 and `I2` "Naan" at 40. Restaurant `R2` "Pizza Point" has `I3` "Margherita" at 300. Partners `P1` "Ravi" and `P2` "Sam" are available. Coupon `SAVE10` is 10 percent off, capped at 50, with a minimum of 200. Coupon `FLAT50` is 50 off with a minimum of 100. User `U1`'s cart holds `I1` once and `I2` twice, so the subtotal is 250 + 80 = 330. A `UserNotifier` is the only observer.

1. **Browse and search**
   - *Success:* `search("pizza")` returns [Pizza Point]. `search("naan")` returns [Spice Hub] because a dish name matches. `list_all()` returns both.
   - *Failure:* `search("sushi")` returns an empty list, and `search("")` returns an empty list.
2. **A cart from one restaurant**
   - *Success:* `add_to_cart("U1", "R1", "I1", 1)` and `add_to_cart("U1", "R1", "I2", 2)` both return True.
   - *Failure:* `add_to_cart("U1", "R2", "I3", 1)` returns False because the cart already holds Spice Hub's items. `add_to_cart("U1", "R1", "I9", 1)` returns False because the dish is not on the menu.
3. **Place an order and pay**
   - *Success:* `place_order("U1", "12 Park St", CardPayment("4111"), "SAVE10")`. The discount is 10 percent of 330, which is 33, so the total is 297. P1 is reserved, the card pays 297, and the order is `PLACED`.
   - *Failure:* an empty address returns None. A declined card returns None, and P1 is released. An empty cart returns None.
4. **Accept, reject, prepare**
   - *Success:* `respond_to_order(order, "R1", True)` makes it `ACCEPTED`, and `start_preparing(order, "R1")` makes it `PREPARING`.
   - *Failure:* `respond_to_order(order, "R2", True)` returns False because R2 does not own the order. A rejection by R1 refunds 297, releases P1, and sets `CANCELLED`.
5. **Assign a delivery partner**
   - *Success:* the first order gets P1. A second order, placed while P1 is busy, gets P2.
   - *Failure:* a third order finds no free partner, so `place_order` returns None, no payment is taken, and no order exists.
6. **Stages, with notifications**
   - *Success:* the order goes `PLACED`, `ACCEPTED`, `PREPARING`, `PICKED_UP` (by P1), `DELIVERED` (by P1). `get_messages("U1")` holds one message for each of the five stages.
   - *Failure:* `cancel_order` once the order is `PREPARING` returns False. `pick_up` by P2 returns False because P2 is not the assigned partner.
7. **Track an order**
   - *Success:* `track_order(order, "U1")` returns status `PREPARING`, restaurant "Spice Hub", partner "Ravi", total 297.
   - *Failure:* `track_order(order, "U2")` returns None because the order belongs to U1.
8. **Coupons and discounts**
   - *Success:* with `FLAT50` the discount is 50, so the total is 280. With `SAVE10` the discount is 33.
   - *Failure:* a cart with a subtotal of 40 and `SAVE10` returns None because 40 is below the minimum of 200. An unknown code `NOPE` returns None.
9. **Multiple payment methods**
   - *Success:* `place_order(..., UpiPayment("u1@upi"), None)` follows exactly the same steps as the card and the total is 330.
   - *Failure:* a declined UPI payment returns None and releases the partner, the same as a declined card.
10. **Rate the restaurant and partner**
    - *Success:* after `DELIVERED`, `rate_restaurant(order, "U1", 5)` makes Spice Hub's average 5.0, and `rate_partner(order, "U1", 4)` makes Ravi's average 4.0.
    - *Failure:* rating before delivery returns False, a second rating returns False, and 6 stars returns False without using up the rating.
