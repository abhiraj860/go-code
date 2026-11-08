# Tic-Tac-Toe: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** the board is the only place that knows what is in each cell and whether a move is legal, and the game is the only place that knows whose turn it is and what the result is. A win is found by looking only at the lines through the cell that was just played, so nothing extra has to be stored.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The game should have a square board, 3x3 by default.
2. Two players with different symbols should play alternate turns.
3. A player should place a symbol on an empty cell, and invalid moves should be rejected.
4. A player should win by filling a full row, column or diagonal with their symbol.
5. The game should end in a draw when the board is full and nobody has won.
6. The game should track whether it is in progress, won or drawn.

**Types and conventions**
- Rows and columns are numbered from 0. A board of size `n` has rows and columns `0` to `n - 1`.
- A winning line is a full row, a full column, or one of the two full diagonals.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `list[type]` always says what the elements are.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- No locks are used, because no requirement mentions concurrent use. Players take turns one at a time.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Symbol (enum) | Names the two symbols | none |
| GameStatus (enum) | Names the three conditions of a game | none |
| Player | Identifies one player and their symbol | name, symbol |
| Board | Holds the cells and checks moves and lines | the cells |
| Game | Runs turns, decides the result and tracks the status | whose turn, status, winner |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Game, board, cell, player, symbol, turn, move, row, column, diagonal, draw, win.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Board, Player and Game are classes. Row, column and diagonal are just ways to read the board. Win and draw are results of the status.
2. **Find fixed sets of values.** The two symbols, and the three conditions of a game.
   - *Why:* a fixed set is an enum, so a typo like "in progres" cannot exist.
   - *Result:* `Symbol` and `GameStatus`.
3. **Give every fact one owner.** What is in each cell belongs to the Board. Whose turn, the status and the winner belong to the Game.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** "Is the board full" and "did this move complete a line" are computed by reading the cells. The current player is found from a turn number.
   - *Why:* a stored count of filled cells beside the cells could go wrong the moment one is updated and the other is not.
5. **Give every verb to the class whose data changes.** Placing a symbol changes the Board, so the Board checks the move. Changing whose turn it is, and the status, belongs to the Game.
6. **Find the action that touches several classes.** A move touches the Board, the turn, and the result. That order is `Game.make_move`.
7. **Find the hint phrase for a pattern.** None. "Tracks whether it is in progress, won or drawn" is a status field, not behavior that differs by status.
8. **Pick likely changes** and check each is a new class or new data (section 9).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Fix the outputs.** A `bool` for a move, because the caller only needs to know if it was accepted.

## 4. Pattern Guide

### Used: none
- **Why none:** no requirement says a rule must be swapped, or that behavior changes by state. The status only decides whether a move is accepted, and one check does that.

### Common patterns (not used here)
| Pattern | Signal phrase | Why not used here |
|---|---|---|
| Strategy | "different ways to calculate or choose" | Not used. There is one way to win and one way to move. |
| State | "behaves differently depending on its state" | Not used. A finished game only refuses moves. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not used. |
| Facade | "a single simple entry point" | Not added as a separate class. `Game` is already the single entry point. |
| Chain of Responsibility | "pass along handlers" | Not used. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### Symbol
- Values: `X`, `O`
- *Why:* requirement 2 gives two players different symbols. A fixed set means a cell holds one of two things or is empty.

### GameStatus
- Values: `IN_PROGRESS`, `WON`, `DRAWN`
- *Why:* exactly the three conditions named in requirement 6.

## 6. Classes (smallest dependency first)

### Player
**Why this class exists:** a player is a thing with its own identity and a symbol. It is separate from the Game so the game can tell the two apart.
- Attributes: `name: str`, `symbol: Symbol`. *Why:* the name is the identity used to say who is moving, and the symbol is what they place.
- Constructor: `Player(name: str, symbol: Symbol)`. *Why:* both are set once, so a player can't change symbol mid-game. It refuses an empty name.
- Methods: `get_name() -> str`, `get_symbol() -> Symbol`. *Why:* read-only views. There are no setters.
- *Extensibility:* none needed.

### Board
**Why this class exists:** requirements 1, 3, 4 and 5 are all about the cells. The board owns them and refuses a bad move itself, so no other class can put the cells in a bad state.
- Attributes:
  - `size: int`. *Why:* the board is `size` by `size`.
  - `cells: list[list[Symbol or None]]` (rows of cells, each a symbol or None for empty). *Why:* the single record of what is on the board.
- Constructor: `Board(size: int)`. *Why:* a board is useless without a size. It refuses `size < 3`, and starts with every cell empty.
- Methods:
  - `get_size() -> int`. *Why:* read-only view.
  - `get_cell(row: int, col: int) -> Symbol or None`. *Why:* a read-only view of one cell. None means empty. It also returns None for a cell outside the board.
  - `place(row: int, col: int, symbol: Symbol) -> bool`. *Why:* False if the cell is outside the board or already filled. Otherwise it fills the cell. This is requirement 3's rejection of invalid moves, in the one place that owns the cells. There is no setter.
  - `completes_line(row: int, col: int) -> bool`. *Why:* True if the symbol in that cell now fills its whole row, its whole column, or (if the cell is on one) a whole diagonal. It only looks at the lines through that cell, so it is computed from the cells and needs no stored counters. False for an empty or outside cell.
  - `is_full() -> bool`. *Why:* computed by checking that no cell is empty, so it can never disagree with the cells.
- *Extensibility:* a bigger board is a different `size`.

### Game (the manager and only entry point)
**Why this class exists:** a move touches the board, the turn and the result, in a fixed order. It is also the one place that knows whose turn it is.
- Attributes:
  - `board: Board`. *Why:* the cells.
  - `players: list[Player]`. *Why:* the two players, in turn order.
  - `current_index: int`. *Why:* whose turn it is. The current player is found from it, never stored twice.
  - `status: GameStatus`. *Why:* the single fact about the result.
  - `winner: Player or None`. *Why:* who won. None until someone does.
- Constructor: `Game(first: Player, second: Player, board_size: int)`. *Why:* the board size is passed in, and callers use 3 for the default. It refuses two players with the same name or the same symbol, and a size below 3. The first player moves first, and the status is `IN_PROGRESS`.
- Methods:
  - `make_move(player_name: str, row: int, col: int) -> bool`. *Why:* it takes the player's name because that is how a caller identifies someone. It returns False for a move that is refused. Steps in section 8.
  - `get_status() -> GameStatus`, `get_winner() -> Player or None`. *Why:* read-only views.
  - `get_current_player() -> Player or None`. *Why:* computed from the index. None once the game is over.
  - `get_cell(row: int, col: int) -> Symbol or None`. *Why:* a read-only view of one cell. It is the only way to see the board, because handing out the board itself would let a caller place a symbol and skip the turn rules.
- *Extensibility:* the board size is passed in, so a different board needs no change here.

## 7. Relationships
- Game **has** Board (the board is created for the game and dies with it).
- Game **has** Player (the game owns the two players it was given).
- Board **uses** Symbol (the cells hold symbols, which are shared values).
- Player **uses** Symbol. Game **uses** GameStatus.
- Enums are always **uses**.

## 8. Interaction

### Layers
1. **Entry point and manager:** Game.
2. **Domain class that guards itself:** Board.
3. **Plain data:** Player and the enums.

### Call chains
- **Make a move**
  1. `Caller -> Game.make_move(player_name, row, col)`.
  2. `Game -> players[current_index]`: only the current player may move.
  3. `Game -> Board.place(row, col, symbol)`: the board decides if the cell is valid and empty.
  4. `Game -> Board.completes_line(row, col)`: only asked after a successful placement.
  5. `Game -> Board.is_full()`: only asked if there was no win.

### Who may call whom
- Callers call `Game` only, and never the board.
- `Game` calls `Board`.
- `Board` calls no other class. No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Make a move
1. `make_move(player_name: str, row: int, col: int)`. If `status` is not `IN_PROGRESS`, return False. *Why:* a finished game accepts no moves.
2. If `player_name` is not the name of the current player, return False. *Why:* a player can't move out of turn.
3. `ok: bool = board.place(row, col, current symbol)`. If False, return False. *Why:* the board refuses a cell that is outside or filled.
4. If `board.completes_line(row, col)`: `winner` becomes the player, `status` becomes `WON`. Return True.
5. Otherwise, if `board.is_full()`: `status` becomes `DRAWN`. Return True.
6. Otherwise `current_index` moves to the other player. Return True.
- **Failure and undo:** steps 1 to 3 refuse before anything changes. After step 3 succeeds nothing can fail, so there is nothing to undo.

### Concurrency
- No requirement mentions concurrent use, so no lock is added.
- If it were needed later, one lock inside `Game` around `make_move` would be enough, because the board is reachable only through the game.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A bigger board (4x4, 5x5) | A different `board_size` passed to `Game` | No. It is data. |
| A computer opponent | A new class that chooses a cell and calls `make_move` like any player | No. |
| Showing the board | A display class that reads `get_cell` | No. |

## 11. Interview Notes

### Key decisions
1. **A win is found by checking only the lines through the last move.**
   - *Why:* it needs nothing stored, so nothing can disagree with the cells.
   - *Rejected:* a counter for every row, column and diagonal. It is faster, but it is a second copy of the board that must be updated in step.
2. **`is_full` is computed by looking at the cells.**
   - *Why:* one fact in one place.
   - *Rejected:* a stored count of filled cells.
3. **The board refuses bad moves itself.**
   - *Why:* no caller can put it in a bad state.
   - *Rejected:* the game checking and then writing to the cells.
4. **`Game` gives read-only access to cells instead of the board.**
   - *Why:* handing out the board would let a caller place a symbol out of turn.
   - *Rejected:* a `get_board` method.
5. **The player is identified by name.**
   - *Why:* callers know a name, and the constructor refuses two equal names.
   - *Rejected:* passing a `Player` object into `make_move`, which a caller could fake.

### Edge cases handled
- A move outside the board, or on a filled cell, is refused.
- A move out of turn is refused.
- A move after a win or a draw is refused.
- A win on the last empty cell is a win, not a draw.
- Two players with the same symbol or name are refused.
- A board smaller than 3x3 is refused.

### Likely follow-up questions
1. **"How would you make it faster for a big board?"** Keep a counter per row, column and diagonal, updated in `place`. It makes the win check constant time, at the price of a second copy of the data that must be kept in step.
2. **"How would you support K in a row on a larger board?"** `completes_line` would count a run of the same symbol through the cell, up to a `win_length` passed to the board. That is the one place the rule lives.
3. **"How would you add undo?"** Keep a list of moves in the game and clear the last cell. That is a new rule and a new `clear` method on the board, because the board has no way to empty a cell today.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Telling the player why a move was refused | `make_move` returns only a bool. A reason would be a new return type. |
| K in a row | See follow-up question 2. |
| Undo or move history | See follow-up question 3. |
| More than two players | Requirement 2 says two. |
| Showing the board | `get_cell` gives what a display needs. |
| Concurrent moves | One lock inside `Game` around `make_move` would be enough. |

## 13. Requirements Met
1. A square board, 3x3 by default: `Board` with `size`, and the `board_size` passed to `Game`.
2. Two players, different symbols, alternate turns: `Player`, the constructor checks in `Game`, and `current_index`.
3. Place on an empty cell, reject invalid moves: `Board.place` and `Game.make_move`.
4. A winning row, column or diagonal: `Board.completes_line`.
5. A draw when full with no winner: `Board.is_full` in `make_move` step 5.
6. Track in progress, won or drawn: `GameStatus` and `Game.status`.

## 14. Requirement Fit Check

**Setup used in every example:** the players are `Asha` with symbol X, who moves first, and `Ben` with symbol O.

1. **A square board, 3x3 by default**
   - *Success:* `Game(Asha, Ben, 3)` makes a 3x3 board with nine empty cells. `Game(Asha, Ben, 4)` makes a 4x4 board.
   - *Failure:* `Game(Asha, Ben, 2)` is refused at construction.
2. **Two players, different symbols, alternate turns**
   - *Success:* `make_move("Asha", 0, 0)` returns True, and now it is Ben's turn.
   - *Failure:* `make_move("Asha", 0, 1)` straight away returns False because it is Ben's turn. `Player("Ben", X)` next to `Asha` with X is refused when the game is built.
3. **Place on an empty cell, reject invalid moves**
   - *Success:* `make_move("Ben", 1, 1)` returns True and cell (1, 1) holds O.
   - *Failure:* `make_move("Asha", 0, 0)` on the filled cell returns False. `make_move("Asha", 3, 3)` returns False because the cell is outside a 3x3 board. Nothing changes in either case, and it is still Asha's turn.
4. **A winning row, column or diagonal**
   - *Success:* X plays (0, 0), (0, 1), (0, 2) with O elsewhere. The third X completes the top row, so the status is `WON` and `get_winner()` is Asha. A column such as (0, 0), (1, 0), (2, 0), or a diagonal such as (0, 0), (1, 1), (2, 2), wins the same way.
   - *Failure:* two in a row is not a win and the game stays `IN_PROGRESS`. After a win, `make_move` returns False.
5. **A draw when full with no winner**
   - *Success:* the moves in order are X (0,0), O (0,1), X (0,2), O (1,0), X (1,1), O (2,0), X (1,2), O (2,2), X (2,1). The board is then X O X, O X X, O X O. It is full, no line is complete, and the status is `DRAWN`.
   - *Failure:* if the ninth move had completed a line, the status would be `WON`, not `DRAWN`, because the win is checked first.
6. **Track in progress, won or drawn**
   - *Success:* after the first move the status is `IN_PROGRESS`. After a winning move it is `WON`. After a drawing move it is `DRAWN`.
   - *Failure:* once the status is `WON` or `DRAWN`, it never goes back to `IN_PROGRESS`, because `make_move` refuses every further move.
