# Snake & Ladder: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** the board only knows where snakes and ladders are, the dice only know how to roll, and the game is the one place that knows whose turn it is and where everybody stands. A turn is always the same four steps: roll, try to move, follow a snake or ladder, check for a win.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The game should have a numbered board of fixed size with a start and a final cell.
2. The board should have snakes that move a player down and ladders that move a player up.
3. Two or more players should take turns.
4. On each turn a player should roll the dice and move forward by the rolled number.
5. A player should not move past the final cell.
6. The first player to reach the final cell should win.
7. The number and type of dice should be configurable.

**Types and conventions**
- Cells are numbered `1` to `size`. The start cell is cell 1 and the final cell is cell `size`. Every player starts on cell 1.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- No locks are used, because no requirement mentions concurrent use. Players take turns one at a time.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| GameStatus (enum) | Names the two conditions of a game | none |
| Player | Identifies one player | id, name |
| TurnResult | Reports what happened in one turn | an unchangeable record |
| Board | Knows the size and where snakes and ladders are | size, snakes, ladders |
| Dice (interface) | Rolls one die | none |
| StandardDice | Rolls a random die with a chosen number of sides | the number of sides |
| SequenceDice | Rolls a fixed list of values in order | the list and its position |
| DiceSet | Rolls all dice and adds them up | the list of dice |
| Game | Runs turns, tracks positions and decides the winner | positions, whose turn, status |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Game, board, cell, start, final cell, snake, ladder, player, turn, dice, roll, winner.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Board, Player, Dice, Game are classes. Cell, start and final cell are numbers. A snake and a ladder are each just two numbers, so they are entries in a board map, not classes.
2. **Find fixed sets of values.** The two conditions of a game.
   - *Why:* a fixed set is an enum, so a typo like "finnished" cannot exist.
   - *Result:* `GameStatus`.
3. **Give every fact one owner.** Snake and ladder positions belong to the Board. Each player's cell, whose turn it is, and the winner belong to the Game.
   - *Why:* two copies of one fact will eventually disagree. A player's cell is not stored on `Player`, so there is no second copy to keep in step.
4. **Mark what can be computed and do not store it.** The start and final cell come from the size. The current player is found from the turn number. Where a landing cell leads is looked up, not stored.
5. **Give every verb to the class whose data changes.** Adding a snake changes the Board. Rolling is the dice's. Moving a player changes the Game's positions.
6. **Find the action that touches several classes.** A turn needs the dice, the board and the positions. That is `Game.take_turn`.
7. **Find the hint phrase for a pattern.** "The number and type of dice should be configurable" is Strategy for the type, and a list for the number.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Fix the outputs.** A `TurnResult` for a turn, and None once the game is over.

## 4. Pattern Guide

### Used: Strategy
- **Meaning:** a way of doing something is a small object behind a shared interface, and the caller uses whichever one it is given.
- **Hint phrase:** "the number and type of dice should be configurable". The type is the strategy and the number is the length of a list.
- **Without it:** the game would have an `if` on the kind of die, and a new kind would mean editing it.
- **When NOT to use it:** when there is one fixed kind of die that nobody asked to change.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A finished game only refuses turns, which one check does. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. The caller builds the dice it wants. |
| Facade | "a single simple entry point" | Not added as a separate class. `Game` is already the single entry point. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### GameStatus
- Values: `IN_PROGRESS`, `FINISHED`
- *Why:* requirement 6 ends the game when a player wins, and a finished game must refuse more turns.

## 6. Classes (smallest dependency first)

### Player
**Why this class exists:** a player is a thing with its own identity. It holds no position, because the position is a fact about the game.
- Attributes: `player_id: str`, `name: str`. *Why:* the identity used as a key, and the display name.
- Constructor: `Player(player_id: str, name: str)`. *Why:* both set once. It refuses empty values.
- Methods: `get_player_id() -> str`, `get_name() -> str`. *Why:* read-only views. There are no setters.
- *Extensibility:* none needed.

### TurnResult
**Why this class exists:** a turn has several results (the roll, where the player landed, where the snake or ladder sent them, whether they won). One record carries them all, so the caller never has to guess.
- Attributes: `player_id: str`, `roll: int`, `start_cell: int`, `landed_cell: int`, `end_cell: int`, `won: bool`. *Why each:* who played, what was rolled, where they began, where the roll took them, where they finished after a snake or ladder, and whether the game ended.
- Constructor: `TurnResult(player_id: str, roll: int, start_cell: int, landed_cell: int, end_cell: int, won: bool)`. *Why:* all fields set once, so a result can't be altered.
- Methods: `get_player_id() -> str`, `get_roll() -> int`, `get_start_cell() -> int`, `get_landed_cell() -> int`, `get_end_cell() -> int`, `has_won() -> bool`. *Why:* read-only views.
- *Extensibility:* a new field is added here and old callers keep working.

### Board
**Why this class exists:** requirements 1 and 2 are about the board alone. It knows the size and where each snake and ladder is, and nothing about players, so it never has to change when the rules for players do.
- Attributes:
  - `size: int`. *Why:* the number of cells. It is the final cell.
  - `snakes: map{int: int}` (a snake's head cell to its tail cell). *Why:* a snake moves a player down, so the tail is lower than the head.
  - `ladders: map{int: int}` (a ladder's bottom cell to its top cell). *Why:* a ladder moves a player up, so the top is higher than the bottom.
- Constructor: `Board(size: int)`. *Why:* a board is useless without a size. It refuses `size < 2`, and starts with no snakes or ladders.
- Methods:
  - `get_size() -> int`, `get_start_cell() -> int`, `get_final_cell() -> int`. *Why:* the start is always 1 and the final cell is the size, computed and not stored.
  - `add_snake(head: int, tail: int) -> bool`. *Why:* False unless `1 <= tail < head < size`, and the head cell has no snake or ladder already. A snake can't start on the final cell, or nobody could win.
  - `add_ladder(bottom: int, top: int) -> bool`. *Why:* False unless `1 <= bottom < top <= size`, and the bottom cell has no snake or ladder already.
  - `resolve(cell: int) -> int`. *Why:* returns where a cell leads: the tail if it is a snake head, the top if it is a ladder bottom, otherwise the cell itself. It is applied once, so a destination is never followed again and a loop is impossible.
- *Extensibility:* more snakes and ladders, or a different size, are data.

### Dice (interface)
**Why it is an interface:** requirement 7 says the type of dice is configurable, so the game depends on one method and not on any kind of die.
- Methods: `roll() -> int`. *Why:* returns a whole number of at least 1.
- *Extensibility:* a new kind of die (weighted, with 8 sides) is a new class implementing `roll`.

### StandardDice (implements Dice)
**Why this class exists:** the usual die: a random whole number from 1 to the number of sides.
- Attributes: `sides: int`. *Why:* how many faces the die has.
- Constructor: `StandardDice(sides: int)`. *Why:* set once. It refuses `sides < 2`.
- Methods: `roll() -> int`. *Why:* a random whole number from 1 to `sides`.
- *Extensibility:* a different number of sides is data.

### SequenceDice (implements Dice)
**Why this class exists:** a die that returns a fixed list of values in order. It is a real second kind of die, and it makes a game repeatable for tests and demos.
- Attributes: `values: list[int]`, `index: int`. *Why:* the values to return, and where in the list it is.
- Constructor: `SequenceDice(values: list[int])`. *Why:* it keeps its own copy. It refuses an empty list or any value below 1. The index starts at 0.
- Methods: `roll() -> int`. *Why:* returns the value at the index and moves on, going back to the start after the last one.
- *Extensibility:* none needed.

### DiceSet
**Why this class exists:** requirement 7 also says the number of dice is configurable. One class that rolls every die and adds them up keeps `Game` from knowing how many there are.
- Attributes: `dice: list[Dice]`. *Why:* the dice to roll. One die or several is just the length of this list.
- Constructor: `DiceSet(dice: list[Dice])`. *Why:* it keeps its own copy and refuses an empty list.
- Methods: `roll_total() -> int`. *Why:* rolls each die once and returns the sum.
- *Extensibility:* more dice are a longer list.

### Game (the manager and only entry point)
**Why this class exists:** a turn touches the dice, the board and the positions, in a fixed order. It is also the one place that knows whose turn it is.
- Attributes:
  - `board: Board`. *Why:* where the snakes and ladders are.
  - `dice_set: DiceSet`. *Why:* how a roll is made. It is passed in, so it can be configured.
  - `players: list[Player]`. *Why:* the players in turn order.
  - `positions: map{str: int}` (a player id to that player's current cell). *Why:* the single record of where everyone is.
  - `current_index: int`. *Why:* whose turn it is. The current player is found from it, never stored twice.
  - `status: GameStatus`. *Why:* whether the game can still take turns.
  - `winner: Player or None`. *Why:* who won. None until someone does.
- Constructor: `Game(board: Board, dice_set: DiceSet, players: list[Player])`. *Why:* it refuses fewer than two players or a repeated player id. Everyone starts on the start cell, the first player moves first, and the status is `IN_PROGRESS`.
- Methods:
  - `take_turn() -> TurnResult or None`. *Why:* plays one turn for the current player. None if the game is finished. Steps in section 8.
  - `get_status() -> GameStatus`, `get_winner() -> Player or None`. *Why:* read-only views.
  - `get_current_player() -> Player or None`. *Why:* computed from the index. None once the game is finished.
  - `get_position(player_id: str) -> int or None`. *Why:* read-only view. None for an unknown id.
- *Extensibility:* the board and the dice are passed in, so neither requires a change here.

## 7. Relationships
- Game **has** Player (the game owns the list of players it was given).
- Game **uses** Board, DiceSet (both passed in).
- Board **uses** nothing else. DiceSet **uses** Dice (passed in).
- StandardDice, SequenceDice **implement** Dice.
- Enums are always **uses**: Game to GameStatus.
- `TurnResult` appears only as a return type, so no attribute holds it and it has no line.

## 8. Interaction

### Layers
1. **Entry point and manager:** Game.
2. **Rules of the world:** Board and DiceSet.
3. **The dice:** Dice and its implementations.
4. **Plain data:** Player, TurnResult.

### Call chains
- **Take a turn**
  1. `Caller -> Game.take_turn()`: the single entry point.
  2. `Game -> DiceSet.roll_total()`: the roll comes first, because everything depends on it.
  3. `DiceSet -> Dice.roll()` for each die.
  4. `Game -> Board.resolve(cell)`: asked only after the move is known to be allowed.
  5. `Game -> positions`: the position is updated last, once the final cell is known.

### Who may call whom
- Callers call `Game` only.
- `Game` calls `DiceSet` and `Board`.
- `DiceSet` calls `Dice`. `Board` and the dice call no other class.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Take a turn
1. `take_turn()`. If `status` is `FINISHED`, return None.
2. `player = players[current_index]`. `roll: int = dice_set.roll_total()`. `start: int = positions[player id]`. `target = start + roll`.
3. If `target > board.get_final_cell()`, the player does not move: `landed = start` and `end = start`. *Why:* requirement 5. The player needs a roll that fits, and this is the simplest rule.
4. Otherwise `landed = target` and `end = board.resolve(target)`, and the player's position becomes `end`. *Why resolve after landing:* a snake or ladder only matters where the player lands.
5. If `end` is the final cell, set `winner` to the player, `status` to `FINISHED`, and do not move the turn on. Otherwise `current_index` moves to the next player, wrapping round.
6. Return `TurnResult(player id, roll, start, landed, end, won)`.
- **Failure and undo:** the only refusal is a finished game, which changes nothing. Every step before the position is updated only reads, so there is nothing to undo.

### Add a snake or a ladder
1. `add_snake(head, tail)` or `add_ladder(bottom, top)` on the board.
- **Failure:** returns False and nothing is added, for example a ladder pointing downward or two jumps on one cell.

### Concurrency
- No requirement mentions concurrent use, so no lock is added. All state changes happen inside `take_turn`.
- If it were needed later, one lock inside `Game` around `take_turn` would be enough.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new kind of die (weighted, 8 sides) | A new class implementing `Dice`, or a `StandardDice` with a different number of sides | No. |
| More or fewer dice | A longer or shorter list passed to `DiceSet` | No. It is data. |
| A bigger board or different snakes and ladders | A different size and more `add_snake` and `add_ladder` calls | No. It is data. |
| More players | A longer list passed to `Game` | No. It is data. |

## 11. Interview Notes

### Key decisions
1. **Positions live in `Game`, not on `Player`.**
   - *Why:* a position is a fact about this game, so there is one owner and no setter.
   - *Rejected:* a `position` field on Player that the game updates. Anyone could set it, and a player in two games would have two positions.
2. **The board knows snakes and ladders, but not players.**
   - *Why:* it can be built and tested alone.
   - *Rejected:* the board moving players. It would need to know about the game.
3. **Snakes and ladders are two small maps, not classes.**
   - *Why:* each is two numbers, and one lookup tells where a cell leads.
   - *Rejected:* a `Snake` and `Ladder` class with identical fields.
4. **`DiceSet` rolls and adds.**
   - *Why:* the number of dice is just a list, and `Game` never knows how many there are.
   - *Rejected:* the game looping over dice itself.
5. **`take_turn` returns a record of what happened.**
   - *Why:* the roll, the landing and the final cell can all be shown without the caller reading game state.
   - *Rejected:* returning only a bool.

### Edge cases handled
- A roll that would pass the final cell leaves the player where they are, and the turn passes.
- Landing on a snake head or ladder bottom follows it once.
- A ladder that goes to the final cell wins.
- A turn after the game ends returns None.
- A snake starting on the final cell is refused.
- Two jumps starting on the same cell are refused.
- One player, or two players with the same id, is refused.
- A die that could return less than 1 can't be built.

### Likely follow-up questions
1. **"How would you give an extra turn for rolling a six?"** In `take_turn`, do not move `current_index` on when the roll is the maximum. Because it is a rule about turns, a `TurnRule` object passed in would keep it out of `Game`.
2. **"What if the player must bounce back instead of staying?"** Change step 3: the overshoot is subtracted from the final cell. It is the one place the final-cell rule lives.
3. **"How do you test a game with random dice?"** Use `SequenceDice` so every roll is known in advance.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Needing an exact roll, or bouncing back, at the end | The player stays put. See follow-up question 2. |
| An extra turn for a six | See follow-up question 1. |
| Several players sharing a cell, or capturing | Players can share a cell. |
| Players joining during the game | The player list is fixed when the game is created. |
| Showing the board | `get_position` gives each player's cell. Drawing it is a display concern. |
| Concurrent turns | One lock inside `Game` around `take_turn` would be enough. |

## 13. Requirements Met
1. A numbered board with a start and final cell: `Board`, with `get_start_cell` and `get_final_cell`.
2. Snakes down and ladders up: `Board.add_snake`, `add_ladder` and `resolve`.
3. Two or more players taking turns: `Game` with `current_index`, and the constructor check.
4. Roll and move forward: `Game.take_turn` with `DiceSet.roll_total`.
5. Not past the final cell: step 3 of `take_turn`.
6. First to the final cell wins: `Game.winner`, `status` and `take_turn` step 5.
7. Configurable dice: the `Dice` interface and the list in `DiceSet`.

## 14. Requirement Fit Check

**Setup used in every example:** `Board(100)` with a ladder from 4 to 14, a snake from 16 to 6, and a snake from 17 to 7. Players `A` and `B`. The dice is `DiceSet([SequenceDice([3, 2, 2])])`, so the rolls are 3, 2, 2, then 3 again.

1. **A numbered board with a start and final cell**
   - *Success:* `Board(100)` has start cell 1 and final cell 100.
   - *Failure:* `Board(1)` is refused. `add_snake(100, 50)` returns False because a snake can't start on the final cell.
2. **Snakes and ladders**
   - *Success:* A rolls 3 from cell 1 and lands on 4, a ladder bottom, so A ends on 14. B rolls 2 and ends on 3. A rolls 2 from 14 and lands on 16, a snake head, so A ends on 6.
   - *Failure:* `add_ladder(5, 3)` returns False because the top is below the bottom. `add_ladder(16, 30)` returns False because cell 16 already has a snake.
3. **Two or more players take turns**
   - *Success:* the turns go A, B, A, B, and so on.
   - *Failure:* a game with one player is refused at construction. Two players with the same id are refused.
4. **Roll and move forward**
   - *Success:* with the roll 3 and a start on cell 1, the player lands on 4. With `DiceSet([SequenceDice([3]), SequenceDice([4])])` the roll is 7.
   - *Failure:* a `SequenceDice` with a value of 0 can't be built, so a roll below 1 is impossible.
5. **Not past the final cell**
   - *Success:* A stands on 98 and rolls 5. The target 103 is past 100, so A stays on 98 and the turn passes. A later roll of 2 reaches 100.
   - *Failure to avoid:* A must never end on 103.
6. **First to the final cell wins**
   - *Success:* A reaches cell 100. The status becomes `FINISHED` and `get_winner()` is A.
   - *Failure:* `take_turn()` after that returns None, and no position changes.
7. **Configurable dice**
   - *Success:* building the game with `DiceSet([StandardDice(6), StandardDice(6)])` plays with two random six-sided dice. `DiceSet([StandardDice(8)])` plays with one eight-sided die. The `Game` code is the same.
   - *Failure:* `StandardDice(1)` and an empty `DiceSet` are both refused at construction.
