# Elevator Control System: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** one `Car` object owns the physical facts of a single elevator (floor, door, load, pending stops) and the rules for moving it. A small, shared `ElevatorState` object only decides where the elevator heads next. A controller picks which elevator takes a call, using a replaceable rule, and one lock per elevator keeps simultaneous requests from clashing.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The building should have multiple elevators serving multiple floors.
2. Users should be able to call an elevator from a floor by choosing a direction, and choose a destination floor once inside.
3. Each elevator should move up or down, stop at requested floors along its direction, and then reverse.
4. The system should assign each request to an elevator using a scheduling rule that can be replaced.
5. An elevator should behave differently depending on whether it is idle, moving up, moving down or out of service.
6. Doors should open and close at every stop.
7. An elevator should not move when it is over its weight limit.
8. Multiple requests should be handled at the same time without conflicts.

**Types and conventions**
- Floors, weights and capacities are `int`. Weight is in one fixed unit (for example kg).
- Time moves in ticks. One tick lets an elevator do exactly one thing: close or hold a door, open a door at a stop, or move one floor. The caller decides when ticks happen, so the design never reads a clock.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}`, `list[type]` and `set[type]` (a collection with no repeats) always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- `Lock` is a mutual-exclusion lock.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| ElevatorInfo | Gives a read-only picture of one elevator | a copy of its key facts |
| Car | Owns one elevator's floor, door, load and pending stops, and guards moving it | floor, door, load, stops |
| ElevatorState (interface) | Lists the actions a state must answer | none |
| IdleState | Behavior when waiting with no stops | none |
| MovingUpState | Behavior when heading up | none |
| MovingDownState | Behavior when heading down | none |
| OutOfServiceState | Refuses calls and does not move | none |
| Elevator | Runs one elevator: asks the current state, applies the result, and holds the lock | which state it is in |
| SchedulingStrategy (interface) | Chooses which elevator takes a call | none |
| NearestElevatorStrategy | Picks the best elevator by position and direction | none |
| ElevatorController | Takes calls, picks an elevator, and advances all elevators | the elevators |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Building, elevator, floor, user, call, direction, destination, door, weight limit, scheduling rule, state.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Car, Elevator and ElevatorController are classes. Floor, direction, destination and weight are values. A "call" needs no class, because it is just a floor and a direction.
2. **Find fixed sets of values.** Up or down, door open or closed, the four conditions of an elevator.
   - *Why:* a fixed set is an enum, so a typo like "opne" cannot exist.
   - *Result:* `Direction`, `DoorStatus`, `ElevatorStateType`.
3. **Give every fact one owner.** Floor, door, load and pending stops belong to Car. The current state belongs to Elevator. The set of elevators belongs to the controller.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** "Is overloaded" is `load > capacity`. "Stops above" is a check on the stop set. The direction of travel is the state.
   - *Why:* a stored "overloaded" flag beside the load would go wrong the moment one is updated and the other is not.
5. **Give every verb to the class whose data changes.** Move, open the door and add a stop change Car. Choosing where to head next is the state's decision. Choosing an elevator for a call is the strategy's.
6. **Find the action that touches several classes.** A call needs a view of every elevator, then a choice, then an assignment that may fail. That order lives in the controller.
7. **Find the hint phrases for patterns.** "Behave differently depending on whether it is idle, moving up, moving down or out of service" is State. "A scheduling rule that can be replaced" is Strategy.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** Requirement 8 names it, so decide exactly where the locks sit (section 8).
10. **Fix the outputs.** A `str` or None for a call, a `bool` for each command, and a read-only `ElevatorInfo` for what the scheduler sees.

## 4. Pattern Guide

### Used: State
- **Meaning:** each condition of the elevator is a small class that decides what an action does, and the elevator just asks the current one.
- **Hint phrase:** "an elevator should behave differently depending on whether it is idle, moving up, moving down or out of service".
- **Without it:** every action would have an `if` on the condition, and adding a condition would mean editing all of them.
- **When NOT to use it:** when there are two simple conditions with one differing behavior. A boolean is enough then.

### Used: Strategy
- **Meaning:** a rule is a small object behind a shared interface, and the controller uses whichever one it is given.
- **Hint phrase:** "using a scheduling rule that can be replaced".
- **Without it:** the controller would hold one fixed choice rule, and changing it would mean editing the controller.
- **When NOT to use it:** when nobody asked to change the rule.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| Observer | "notify when something changes" | Not used. No requirement asks for notifications or display updates. |
| Factory | "create different kinds of objects by type" | Not used. All elevators are the same class with different numbers. |
| Facade | "a single simple entry point" | Not added as a separate class. `ElevatorController` is already the single entry point because it is the manager. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. A passed-in controller is easier to test. |

## 5. Enums

### Direction
- Values: `UP`, `DOWN`
- *Why:* requirement 2 has the user choose a direction when calling. The car also moves one floor in a direction.

### DoorStatus
- Values: `OPEN`, `CLOSED`
- *Why:* requirement 6 needs the door to be open or closed, and an elevator must not move while it is open.

### ElevatorStateType
- Values: `IDLE`, `MOVING_UP`, `MOVING_DOWN`, `OUT_OF_SERVICE`
- *Why:* these are the four conditions from requirement 5. A state class returns one of these labels to say where the elevator goes next.
- *Why a label and not the next state object:* if each state held the next state, the states would point at each other in a circle. A label has no such dependency.
- *Extensibility:* a new condition is one new value here, plus a new state class and a map entry.

## 6. Classes (smallest dependency first)

### ElevatorInfo
**Why this class exists:** the scheduler needs to see every elevator without being able to change it. A copy made at one moment is also consistent, because it can't change while the scheduler reads it.
- Attributes: `elevator_id: str`, `current_floor: int`, `state_type: ElevatorStateType`, `stop_count: int`, `min_floor: int`, `max_floor: int`. *Why each:* exactly what a rule needs to judge how near, how busy and how able an elevator is to serve a floor.
- Constructor: `ElevatorInfo(elevator_id: str, current_floor: int, state_type: ElevatorStateType, stop_count: int, min_floor: int, max_floor: int)`. *Why:* all fields are set once, so the picture can't be altered.
- Methods: `get_elevator_id() -> str`, `get_current_floor() -> int`, `get_state_type() -> ElevatorStateType`, `get_stop_count() -> int`, `get_min_floor() -> int`, `get_max_floor() -> int`. *Why:* read-only views.
- *Extensibility:* a new field for a new rule is added here and nothing else changes for existing rules.

### Car
**Why this class exists:** the facts and the physical rules of one elevator (floor, door, load, stops) must have one owner that checks every change. It stays separate from the states so the states can be shared by every elevator.
- Attributes:
  - `car_id: str`. *Why:* the identity of this elevator.
  - `capacity: int`. *Why:* the weight limit from requirement 7.
  - `min_floor: int`, `max_floor: int`. *Why:* requirement 1 says elevators serve multiple floors, and different elevators may serve different ranges.
  - `current_floor: int`. *Why:* where the car is. It is the only copy.
  - `load: int`. *Why:* the last weight reported by the sensor. There is no stored "overloaded" flag, because that would be a second copy.
  - `door: DoorStatus`. *Why:* whether the door is open.
  - `stops: set[int]`. *Why:* the floors it must stop at. A set can't hold the same floor twice, so repeated requests collapse into one stop.
- Constructor: `Car(car_id: str, capacity: int, min_floor: int, max_floor: int, start_floor: int)`. *Why:* it refuses a capacity of 0 or less, `min_floor >= max_floor` (an elevator must serve multiple floors), or a start floor outside the range. The door starts CLOSED, the load at 0 and there are no stops, which is the idle condition.
- Methods:
  - `get_car_id() -> str`, `get_current_floor() -> int`, `get_door() -> DoorStatus`, `get_load() -> int`. *Why:* read-only views.
  - `can_serve(floor: int) -> bool`. *Why:* computed from the floor range.
  - `is_overloaded() -> bool`. *Why:* computed as `load > capacity`, so it can't disagree with the load.
  - `report_load(weight: int) -> bool`. *Why:* a named method instead of a setter. It refuses a negative weight.
  - `add_stop(floor: int) -> bool`. *Why:* refuses a floor the car can't serve. Adding a floor already in the set succeeds and changes nothing.
  - `has_stops() -> bool`, `has_stops_above() -> bool`, `has_stops_below() -> bool`, `get_stop_count() -> int`. *Why:* computed from the stop set. The states use them to choose a direction.
  - `serve_stop() -> bool`. *Why:* if the current floor is a stop, it removes it and opens the door, then returns True. Otherwise False. This is how requirement 6 happens at every stop.
  - `handle_door() -> bool`. *Why:* the one place for the door and load rule. If the door is OPEN, it tries to close it. If the car is overloaded it stays open. If the door is CLOSED but the car is overloaded, it opens the door. It returns True if this tick was used on the door or the load, and False if the car is free to act.
  - `move(direction: Direction) -> bool`. *Why:* moves one floor. It refuses when the door is OPEN (requirement 6), when the car is overloaded (requirement 7), or at the end of its floor range.
- *Why it has no lock:* the Elevator that owns it holds the lock, and the car is never given to anyone else.
- *Extensibility:* a different weight limit is a different `capacity` passed in.

### ElevatorState (interface)
**Why it is an interface:** requirement 5 gives four conditions, so four classes implement the same set of actions. The elevator depends on this interface and not on any one state.
- Methods:
  - `add_stop(car: Car, floor: int) -> ElevatorStateType or None`. *Why it returns None:* None means this state refuses the stop, and nothing changed.
  - `step(car: Car) -> ElevatorStateType`. *Why it always returns a type:* every tick ends in some state, even if it is the same one.
  - `take_out_of_service(car: Car) -> ElevatorStateType or None`.
  - `return_to_service(car: Car) -> ElevatorStateType or None`.
- *Why the states take the car as an input:* then they hold no data of their own, so one set of states is shared by every elevator, and no state points back at an elevator.
- *Extensibility:* a new state implements these four methods.

### IdleState (implements ElevatorState)
**Why this class exists:** it defines "waiting with nothing to do".
- Attributes: none. Constructor: `IdleState()`.
- Behavior:
  - `add_stop`: if `car.add_stop(floor)` is False, return None. Otherwise return `MOVING_UP` if the floor is above the car, `MOVING_DOWN` if below, and `IDLE` if it is the current floor (the next tick opens the door).
  - `step`: if `car.handle_door()` is True, return `IDLE`. If `car.serve_stop()` is True, return `IDLE`. If there are stops above, return `MOVING_UP`. If below, `MOVING_DOWN`. Otherwise `IDLE`.
  - `take_out_of_service`: if the door is CLOSED and there are no stops, return `OUT_OF_SERVICE`. Otherwise None. *Why:* nobody should be inside when the elevator stops serving.
  - `return_to_service`: None.
- *Extensibility:* none needed.

### MovingUpState (implements ElevatorState)
**Why this class exists:** it defines "heading up", where a car serves stops ahead before turning.
- Attributes: none. Constructor: `MovingUpState()`.
- Behavior:
  - `add_stop`: if `car.add_stop(floor)` is False, return None. Otherwise return `MOVING_UP`. *Why:* a stop behind the car is kept and served after the car reverses.
  - `step`:
    1. If `car.handle_door()` is True, return `MOVING_UP`.
    2. If `car.serve_stop()` is True, return `MOVING_UP` if there are stops above, else `MOVING_DOWN` if there are stops below, else `IDLE`.
    3. If there are stops above, call `car.move(UP)` and return `MOVING_UP`.
    4. If there are stops below, return `MOVING_DOWN` (reverse).
    5. Otherwise return `IDLE`.
  - `take_out_of_service`, `return_to_service`: None. *Why:* a moving car finishes its stops first.
- *Extensibility:* none needed.

### MovingDownState (implements ElevatorState)
**Why this class exists:** it is the mirror of MovingUpState for the other direction. They are two classes because requirement 5 names them as two conditions.
- Attributes: none. Constructor: `MovingDownState()`.
- Behavior: the same as `MovingUpState`, with above and below swapped and `move(DOWN)` used.
- *Extensibility:* none needed.

### OutOfServiceState (implements ElevatorState)
**Why this class exists:** it defines "not serving", so refusing everything is in one place.
- Attributes: none. Constructor: `OutOfServiceState()`.
- Behavior:
  - `add_stop`: None. `step`: return `OUT_OF_SERVICE` and do nothing. `take_out_of_service`: None.
  - `return_to_service`: return `IDLE`.
- *Extensibility:* none needed.

### Elevator
**Why this class exists:** one elevator needs one place that asks its current state, applies the answer, and makes simultaneous requests wait their turn. It holds only the one fact that changes between actions: which state it is in.
- Attributes:
  - `car: Car`. *Why:* the physical facts and rules for this elevator.
  - `states: map{ElevatorStateType: ElevatorState}` (state type to the state object for that type). *Why:* the elevator looks up the real state from the label a state returns. The map is shared by all elevators because the states hold no data.
  - `current_type: ElevatorStateType`. *Why:* the single record of which state the elevator is in.
  - `lock: Lock`. *Why:* makes every action on this elevator run one at a time (requirement 8).
- Constructor: `Elevator(car: Car, states: map{ElevatorStateType: ElevatorState})`. *Why:* the car and states are passed in so tests can use fakes, and no state needs editing to add another. It starts at `IDLE`, creates the lock, and refuses a map with no `IDLE` entry.
- Methods (each runs under the lock):
  - `get_elevator_id() -> str`. *Why:* read-only view.
  - `add_stop(floor: int) -> bool`. *Why:* asks the current state. If the state returns None, return False. Otherwise store the returned type and return True.
  - `step() -> None`. *Why:* asks the current state to act for one tick and stores the type it returns.
  - `take_out_of_service() -> bool`, `return_to_service() -> bool`. *Why:* the same pattern, True only if the state allowed it.
  - `report_load(weight: int) -> bool`. *Why:* passes the sensor reading to the car. It is under the lock so a reading can't land in the middle of a tick.
  - `snapshot() -> ElevatorInfo`. *Why:* builds a consistent copy of the key facts under the lock, so the scheduler never reads a half-changed elevator.
- *Why the car is never exposed:* nobody can reach the car's methods without going through the lock.
- *Extensibility:* a new state needs no change here, because the map is passed in.

### SchedulingStrategy (interface)
**Why it is an interface:** requirement 4 says the rule must be replaceable, so the controller depends on this one method and not on a specific rule.
- Methods: `select(candidates: list[ElevatorInfo], floor: int, direction: Direction) -> ElevatorInfo or None`.
  - *Why it takes read-only copies:* a rule can judge elevators but can't change them.
  - *Why it returns None:* None means no candidate is suitable.
- *Extensibility:* a new rule is a new class implementing this method.

### NearestElevatorStrategy (implements SchedulingStrategy)
**Why this class exists:** the first rule: send the elevator that will reach the caller soonest.
- Attributes: none. Constructor: `NearestElevatorStrategy()`.
- Methods: `select(candidates: list[ElevatorInfo], floor: int, direction: Direction) -> ElevatorInfo or None`.
  - *How:* an elevator is "on the way" if it is idle, or is heading up with `current_floor <= floor` for an UP call, or heading down with `current_floor >= floor` for a DOWN call. It picks the elevator on the way first, then the smallest distance to the floor, then the fewest pending stops, then the lowest id. It returns None for an empty list.
  - *Why this order:* an elevator already going the right way picks up the caller without turning, so being on the way matters more than distance. The last two tie-breakers make the result the same every time.
- *Extensibility:* none needed.

### ElevatorController (the manager and only entry point)
**Why this class exists:** a call needs a view of every elevator, a choice, and an assignment that can fail, in that order. No single elevator or rule can do that.
- Attributes:
  - `elevators: map{str: Elevator}` (elevator id to the Elevator). *Why:* the one record of every elevator.
  - `strategy: SchedulingStrategy`. *Why:* the replaceable rule, passed in.
  - `states: map{ElevatorStateType: ElevatorState}` (state type to the state object). *Why:* shared by every elevator the controller builds. The states hold no data, so sharing is safe.
  - `lock: Lock`. *Why:* guards only the `elevators` map while one is added.
- Constructor: `ElevatorController(strategy: SchedulingStrategy, states: map{ElevatorStateType: ElevatorState})`. *Why:* the rule and the states are passed in. It starts with no elevators and refuses a map with no `IDLE` entry.
- Methods:
  - `add_elevator(elevator_id: str, capacity: int, min_floor: int, max_floor: int, start_floor: int) -> bool`. *Why:* administrator setup. It builds the Car and the Elevator. False if the id exists or the values are invalid.
  - `request_elevator(floor: int, direction: Direction) -> str or None`. *Why:* requirement 2's call from a floor. It returns the id of the assigned elevator, or None if none can take it.
  - `select_destination(elevator_id: str, floor: int) -> bool`. *Why it takes an id:* it is an entry point. It looks up the elevator once and asks it to add the stop. False if the id is unknown or the elevator refuses.
  - `report_load(elevator_id: str, weight: int) -> bool`. *Why:* the weight sensor reports through the controller, so no one needs a direct reference.
  - `take_out_of_service(elevator_id: str) -> bool`, `return_to_service(elevator_id: str) -> bool`. *Why:* look up once and ask the elevator.
  - `step_all() -> None`. *Why:* the tick. It asks every elevator to take one step. A timer, or one thread per elevator, calls it.
  - `get_status() -> list[ElevatorInfo]`. *Why:* a read-only view for display.
- *Extensibility:* the scheduling rule and the states are both passed in, so neither requires a change here.

## 7. Relationships
- ElevatorController **has** Elevator (created inside `add_elevator`).
- Elevator **has** Car (the car dies with its elevator).
- ElevatorController **uses** SchedulingStrategy, ElevatorState (both passed in).
- Elevator **uses** ElevatorState (the shared state objects, passed in).
- IdleState, MovingUpState, MovingDownState, OutOfServiceState **implement** ElevatorState.
- NearestElevatorStrategy **implements** SchedulingStrategy.
- Enums are always **uses**: Car to DoorStatus, Elevator to ElevatorStateType, ElevatorController to ElevatorStateType, ElevatorInfo to ElevatorStateType.
- `Direction` and `ElevatorInfo` appear only as method inputs or outputs, so no attribute holds them and they have no line.

## 8. Interaction

### Layers
1. **Entry point and manager:** ElevatorController. The only class users call.
2. **Per-elevator control:** Elevator holds the lock and the current state.
3. **Decisions:** the four states, and the scheduling strategy.
4. **Physical data and rules:** Car.
5. **Plain data:** ElevatorInfo.

### Call chains
- **Call an elevator**
  1. `User -> ElevatorController.request_elevator(floor, direction)`.
  2. `ElevatorController -> Elevator.snapshot()` for each elevator: a consistent copy of each.
  3. `ElevatorController -> SchedulingStrategy.select(candidates, floor, direction)`: the rule chooses.
  4. `ElevatorController -> Elevator.add_stop(floor)`: the elevator's own state decides whether to accept.
  5. If step 4 is refused, drop that candidate and go back to step 3.
- **Choose a destination**
  1. `User -> ElevatorController.select_destination(elevator_id, floor)`.
  2. `ElevatorController -> Elevator.add_stop(floor)`: the one lookup by id happens before this.
  3. `Elevator -> current state.add_stop(car, floor)`, then `Car.add_stop(floor)`.
- **Tick**
  1. `Timer -> ElevatorController.step_all()`.
  2. `ElevatorController -> Elevator.step()` for each elevator.
  3. `Elevator -> current state.step(car)`.
  4. `State -> Car.handle_door()`, `Car.serve_stop()`, `Car.move(direction)`: each tick does one of these.
- **Weight sensor**
  1. `Sensor -> ElevatorController.report_load(elevator_id, weight)`.
  2. `ElevatorController -> Elevator.report_load(weight)`, then `Car.report_load(weight)`.
- **Take out of service**
  1. `Operator -> ElevatorController.take_out_of_service(elevator_id)`.
  2. `ElevatorController -> Elevator.take_out_of_service()`, then `current state.take_out_of_service(car)`.

### Who may call whom
- Users, sensors, operators and the timer call `ElevatorController` only.
- `ElevatorController` calls `Elevator`, `SchedulingStrategy`.
- `Elevator` calls its state and, for load reports and snapshots, its `Car`.
- States call `Car` only.
- `SchedulingStrategy` reads `ElevatorInfo` only and never calls an elevator.
- States must never call `Elevator`, another state, or the controller. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Call an elevator
1. `request_elevator(floor: int, direction: Direction)`. Take `infos: list[ElevatorInfo]` by calling `snapshot()` on every elevator.
2. `candidates` is every info whose state is not `OUT_OF_SERVICE` and whose range includes `floor`. *Why the controller filters:* then no rule has to repeat the filter.
3. `chosen: ElevatorInfo or None = strategy.select(candidates, floor, direction)`. If None, return None.
4. `elevator: Elevator = elevators.get(chosen.get_elevator_id())`, looked up once.
5. `elevator.add_stop(floor)`. If True, return the elevator id.
6. If False, the elevator changed between the snapshot and now (for example, it went out of service). Remove it from `candidates` and go back to step 3.
- **Failure and undo:** a refused `add_stop` changes nothing, so there is nothing to undo. When no candidate is left, the result is None.

### Choose a destination
1. `select_destination(elevator_id: str, floor: int)`. Look up the elevator once. If None, return False.
2. `elevator.add_stop(floor)`. False if the elevator is out of service or does not serve the floor.
- **Failure:** returns False and nothing changes.

### Tick (one elevator, one step)
1. `step_all()` calls `elevator.step()` on every elevator.
2. The current state's `step(car)` does exactly one of these, in this order:
   1. `car.handle_door()` is True: the door closed this tick, or is held open because of overload, or was opened because of overload.
   2. `car.serve_stop()` is True: the car is at a stop, so the door opened.
   3. Move one floor toward the next stop.
   4. Reverse direction if there are no stops ahead but there are stops behind.
   5. Go idle.
- **Why this order:** the door is handled first, so a car never moves with an open door or an overload.
- **Failure and undo:** none of these steps can leave a half-done state. Each is one change, made by one method.

### Weight limit
1. `report_load(elevator_id: str, weight: int)`. Look up once. False if the id is unknown or the weight is negative.
2. If the new load is over the capacity, the next tick opens the door and the car holds. Each later tick keeps the door open until the load drops, then the door closes and the car moves again.
- **Failure:** a refused report leaves the old load.

### Out of service
1. `take_out_of_service(elevator_id: str)` is accepted only in `IDLE` with a closed door and no pending stops.
2. `return_to_service(elevator_id: str)` is accepted only in `OUT_OF_SERVICE`, and returns to `IDLE`.
- **Failure:** any other state refuses and stays as it was. A moving car must finish its stops first.

### Concurrency (requirement 8)
- **Elevator:** the lock sits inside `Elevator`, around `add_stop`, `step`, `take_out_of_service`, `return_to_service`, `report_load` and `snapshot`. Every change to one elevator runs one at a time, so two requests can't corrupt its stop set or state.
- **Car and states:** they have no lock. Only the Elevator, holding its lock, reaches them.
- **Controller:** the lock sits inside `ElevatorController`, around the `elevators` map only. It is never held while calling an elevator.
- **Assignment race:** the snapshot may be stale by the time `add_stop` runs. A refused `add_stop` makes the controller drop that candidate and choose again, so no request is lost and no elevator is assigned something it refuses.
- **Lock order:** no method ever holds two locks at once, so no deadlock is possible.
- **Why not one big lock:** it would make every elevator wait for every other, which is slower and unnecessary because requests for different elevators don't conflict.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new scheduling rule (least busy, zoned elevators, express) | A new class implementing `SchedulingStrategy`, passed to the controller | No. |
| More elevators, or elevators serving different floors | Calls to `add_elevator` with different numbers | No. It is data. |
| A different weight limit per elevator | A different `capacity` in `add_elevator` | No. It is data. |
| A new state, such as MAINTENANCE or FIRE_MODE | A new state class, one new enum value, one new map entry | The one state that moves into it must return the new type, so that return value changes. This is the cost of keeping transitions inside states. |

## 11. Interview Notes

### Key decisions
1. **`Car` owns the physical facts and rules, and the states hold no data.**
   - *Why:* states can be shared by every elevator, and no state points back at an elevator.
   - *Rejected:* states that hold the elevator. They would point at it while it points at them, in a circle.
2. **One lock per elevator, and a retry if the chosen elevator refuses.**
   - *Why:* requests for different elevators never wait for each other, and a stale snapshot is handled by choosing again.
   - *Rejected:* one lock for the whole controller, which would serialize everything.
3. **One tick does one thing.**
   - *Why:* doors opening and closing at each stop, and an elevator stopping for overload, are all visible and testable step by step.
   - *Rejected:* a single `step` that moves, opens and closes at once. The door rule would not be observable.
4. **The scheduler sees `ElevatorInfo` copies, not live elevators.**
   - *Why:* a rule can't change an elevator, and the copy is consistent.
   - *Rejected:* passing elevators to the rule. A rule could modify one, and reading several facts could see a half-changed elevator.
5. **"Overloaded" is computed from the load.**
   - *Why:* one fact in one place.
   - *Rejected:* an `overloaded` flag next to the load. They would disagree after one missed update.

### Edge cases handled
- A call for a floor no elevator serves, or when every elevator is out of service, returns None.
- An elevator going out of service between the snapshot and the assignment: the controller picks another.
- The same floor requested twice becomes one stop.
- A stop at the floor the car is already on opens the door on the next tick.
- Stops behind a moving elevator wait until it reverses.
- Overload with the door closed: the door opens and the car holds. Overload with the door open: it stays open.
- A destination outside an elevator's range is refused.
- Taking a moving elevator out of service is refused.

### Likely follow-up questions
1. **"How would you pick elevators more cleverly?"** Write another `SchedulingStrategy`, such as a zoned or least-busy rule. Nothing else changes.
2. **"Why is each elevator driven by ticks?"** It makes the behavior deterministic and testable. A timer or one thread per elevator can call `step`, and the elevator's lock keeps it safe either way.
3. **"What happens when two calls arrive at the same instant?"** Both take snapshots and may pick the same elevator. Each `add_stop` runs under that elevator's lock, so both stops are added. If the elevator refuses one, the controller chooses again.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Fire mode or emergency stop | A new state class. See the new-state row in section 10. |
| Door obstruction sensors | The door closes when not overloaded. A sensor would add a reason `handle_door` keeps the door open. |
| Serving stops first-come-first-served | Stops are served by direction order. It is the standard behavior and meets requirement 3. |
| The direction of a hall call after assignment | It only helps choose an elevator. Once assigned, the floor is just a stop. |
| Cancelling a request | No requirement asks for it. It would be a `remove_stop` on `Car`. |
| Travel time and speed | One floor per tick. The caller sets how often ticks happen. |
| Button lights and floor displays | No requirement asks for them. `get_status` provides what a display needs. |
| Weight measured by the design | The sensor reports the weight through `report_load`. |

## 13. Requirements Met
1. Multiple elevators serving multiple floors: `ElevatorController.add_elevator`, with `Car.min_floor` and `max_floor`.
2. Call with a direction, then choose a destination: `ElevatorController.request_elevator` and `select_destination`.
3. Move, stop along the direction, then reverse: `MovingUpState.step`, `MovingDownState.step` and `Car.serve_stop`.
4. A replaceable scheduling rule: the `SchedulingStrategy` interface with `NearestElevatorStrategy`.
5. Behavior by condition: the `ElevatorState` interface with its four implementations.
6. Doors open and close at every stop: `Car.serve_stop` opens the door and `Car.handle_door` closes it on the next tick.
7. No movement when overloaded: `Car.is_overloaded`, `Car.handle_door` and `Car.move`.
8. Simultaneous requests without conflicts: the lock inside `Elevator` and the retry loop in `request_elevator`.

## 14. Requirement Fit Check

**Setup used in every example:** the strategy is `NearestElevatorStrategy`. Elevator `E1` has capacity 600, floors 0 to 10, and starts at floor 0. Elevator `E2` has capacity 600, floors 0 to 10, and starts at floor 8.

1. **Multiple elevators, multiple floors**
   - *Success:* `add_elevator("E1", 600, 0, 10, 0)` and `add_elevator("E2", 600, 0, 10, 8)` both return True.
   - *Failure:* `add_elevator("E1", ...)` again returns False because the id exists. `add_elevator("E3", 600, 5, 5, 5)` returns False because it serves only one floor.
2. **Call and choose a destination**
   - *Success:* `request_elevator(3, UP)`. Both are idle, E1 is 3 floors away and E2 is 5, so E1 is chosen. `E1.add_stop(3)` is accepted, E1 becomes `MOVING_UP`, and the result is "E1". Then `select_destination("E1", 7)` is accepted.
   - *Failure:* `request_elevator(15, UP)` returns None because no elevator serves floor 15. `select_destination("E9", 4)` returns False. `select_destination("E1", 20)` returns False.
3. **Move, stop along the way, reverse**
   - *Success:* E1 at floor 0 has stops {3, 7} and is `MOVING_UP`. Ticks 1 to 3 move it to floors 1, 2, 3. Tick 4 serves the stop at 3 (door opens). Tick 5 closes the door. Ticks 6 to 9 move it to 7. Tick 10 serves the stop at 7, and with no stops left it goes `IDLE`. Tick 11 closes the door.
   - *Reverse:* E1 at floor 7, moving up, has stops {9, 2}. It serves 9 first. With no stops above and one below, it becomes `MOVING_DOWN` and serves 2.
   - *Failure to avoid:* a stop behind the car is not served until the upward stops are done.
4. **Replaceable scheduling rule**
   - *Success:* E1 is at floor 6 `MOVING_UP` and E2 is `IDLE` at floor 8. `request_elevator(5, DOWN)`. E1 is closer, but heading up is not on the way for a DOWN call. E2 is idle, so E2 is chosen. A different `SchedulingStrategy` passed to the controller changes the rule with nothing else edited.
   - *Failure:* with every elevator out of service, the candidate list is empty and the result is None.
5. **State-dependent behavior**
   - *Success:* `take_out_of_service("E2")` on an idle elevator with a closed door returns True. Calls then skip E2, and `E2.add_stop` is refused. `return_to_service("E2")` returns True and E2 is `IDLE`.
   - *Failure:* `take_out_of_service("E1")` while E1 is `MOVING_UP` returns False and nothing changes.
6. **Doors at every stop**
   - *Success:* when E1 reaches floor 3 and serves the stop, the door is OPEN. The next tick closes it. Only then can it move.
   - *Failure to avoid:* while the door is OPEN, `move` refuses, so the car can never travel with an open door.
7. **No movement when overloaded**
   - *Success:* `report_load("E1", 650)` returns True. On the next tick the door is closed but the car is overloaded, so the door opens and the car does not move, even with pending stops. `report_load("E1", 500)` lets the next tick close the door, and the car moves again.
   - *Failure:* `report_load("E1", -5)` returns False and the old load stays.
8. **Simultaneous requests**
   - *Success:* `request_elevator(3, UP)` and `request_elevator(4, UP)` arrive at the same moment. Both snapshots show both elevators idle, and both choose E1. Each `E1.add_stop` runs under E1's lock, so both are True and the stops are {3, 4}.
   - *Failure and recovery:* E1 goes out of service between a request's snapshot and its `add_stop`. E1 refuses, the controller drops E1 and chooses again, and the result is "E2". No request is lost.
