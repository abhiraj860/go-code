# Chess Game Engine: Final LLD

## 1. Reading Guide

**The one idea that ties the design together:** each kind of piece knows its own moves, the board never changes in place (a move produces a new board), and a move is legal only if, after making it on a copy, your own king is not attacked. That one test handles pins, discovered checks, and moving into check, so there is no special code for each case.

**Read in this order:** Class Responsibilities, Design Derivation, Pattern Guide, Enums, Classes, Relationships, Interaction, Flows, Extensibility, Interview Notes, Scope Boundaries, Requirements Met, Requirement Fit Check.

**Requirements (numbered for the mapping sections):**
1. The game should have an 8x8 board with 32 pieces in the standard starting position.
2. Two players should play alternate turns, with white moving first.
3. Each type of piece should move according to its own rules.
4. A player should be able to capture opponent pieces.
5. The game should reject illegal moves such as moving through other pieces, capturing own pieces or leaving the king in check.
6. The game should detect check, checkmate and stalemate.
7. The game should support castling, en passant and pawn promotion.
8. The game should end in a win, a draw or a resignation.
9. The game should record the history of moves.

**Types and conventions**
- A square is written in chess notation such as `e4`. Inside the design it is a `Position` with a file (0 for a to 7 for h) and a rank (0 for 1 to 7 for 8).
- White moves up the board (rank increases) and black moves down.
- `or None` means the method can fail and returns None. A `bool` result means True on success and False on refusal with nothing changed.
- `map{key type: value type}` and `list[type]` always say what the key and value mean.
- A constructor refuses invalid input with an invalid-argument error, so a half-built object never exists.
- No locks are used, because no requirement mentions concurrent use. Players take turns one at a time.

## 2. Class Responsibilities

| Class | One-line job | Fact it owns |
|---|---|---|
| Position | Names one square and finds its neighbours | file, rank |
| Move | Records one move, including what it captured and any special kind | an unchangeable record |
| Piece (abstract) | Holds a piece's colour and whether it has moved, and lists its squares and moves | colour, has-moved |
| SlidingPiece (abstract) | Walks in straight lines until blocked | its directions |
| Rook, Bishop, Queen | Slide along their own directions | none |
| Knight | Jumps in an L shape | none |
| King | Steps one square, and castles | none |
| Pawn | Moves forward, captures diagonally, en passant and promotion | none |
| Board | Holds where every piece stands and finds legal moves | the squares, the last move |
| HistoryEntry | Records one played move and whether it gave check | an unchangeable record |
| Game | Runs turns, checks legality, detects the end, and keeps the history | the current board, turn, status, history |

## 3. Design Derivation (the method, applied here)

1. **Underline the nouns.** Board, square, piece, king, queen, rook, bishop, knight, pawn, player, turn, move, capture, check, castling, en passant, promotion, history.
   - *Why:* a noun with its own data becomes a class, and a noun that is a single value becomes an attribute.
   - *Result:* Board, Piece and each piece kind, Move and Game are classes. A square is a `Position`. Check, capture and the special moves are results or kinds of a move, not classes.
2. **Find fixed sets of values.** The colours, the piece kinds, the kinds of move, and the end conditions.
   - *Why:* a fixed set is an enum, so a typo like "quen" cannot exist.
   - *Result:* `Color`, `PieceType`, `MoveKind`, `GameStatus`.
3. **Give every fact one owner.** Where pieces stand belongs to the Board. Whether a piece has moved belongs to the Piece. Whose turn, the status and the history belong to the Game.
   - *Why:* two copies of one fact will eventually disagree.
4. **Mark what can be computed and do not store it.** Check, the legal moves, checkmate and stalemate are all computed from the board.
   - *Why:* a stored "in check" flag would be wrong after the next move unless someone remembered to update it.
5. **Give every verb to the class whose data changes.** A piece's move rules belong to that piece. Making a move produces a new Board. Ending the game changes the Game.
6. **Find the action that touches several classes.** A turn needs the board's legal moves, the new board, the history and the end check. That order lives in `Game.make_move`.
7. **Find the hint phrase for a pattern.** "Each type of piece should move according to its own rules" is polymorphism: one class per piece kind behind one shared interface.
8. **Pick likely changes** and check each one (section 10).
9. **Check concurrency.** No requirement mentions it, so no locks.
10. **Fix the outputs.** A `bool` for a move or a resignation, and lists for history and legal moves.

## 4. Pattern Guide

### Used: polymorphism through inheritance (a Strategy built into the class hierarchy)
- **Meaning:** every piece answers the same two questions ("which squares do you attack?" and "which moves can you make?"), and each kind answers in its own way.
- **Hint phrase:** "each type of piece should move according to its own rules".
- **Without it:** one big method would have an `if` for each piece kind, and the rules of six pieces would be tangled together.
- **Shared code:** `SlidingPiece` walks a straight line until it is blocked, once, for the rook, bishop and queen. Each of the three only lists its directions.
- **When NOT to use it:** when the kinds differ only by a number. Data is enough then.

### Not a named pattern, but a key choice: a board that never changes in place
- **Meaning:** `with_move` returns a new board instead of changing the old one.
- **Why:** the legality test makes a move on a copy and looks for check. With a board that never changes, no move ever has to be undone, so there are no undo bugs.

### Other common patterns (not used here)
| Pattern | Signal phrase | Why used or not here |
|---|---|---|
| State | "behaves differently depending on its state" | Not used. A finished game only refuses moves, which one check does. |
| Command | "undo and redo actions" | Not used. Moves are recorded as data, and undo is not required. |
| Observer | "notify when something changes" | Not used. No requirement asks for notifications. |
| Factory | "create different kinds of objects by type" | Not a separate class. A small private helper in `Board` builds a piece from its type for the start position and promotion. |
| Singleton | "exactly one instance" | Not used. |

## 5. Enums

### Color
- Values: `WHITE`, `BLACK`. Method: `opposite() -> Color`. *Why:* the two sides of requirement 2, and the other side is needed everywhere.

### PieceType
- Values: `KING`, `QUEEN`, `ROOK`, `BISHOP`, `KNIGHT`, `PAWN`. *Why:* the six kinds of piece, and the choices for promotion.

### MoveKind
- Values: `NORMAL`, `CASTLE`, `EN_PASSANT`, `PROMOTION`. *Why:* requirement 7's special moves change more than one square or change the piece, so the board must know which kind it is making.

### GameStatus
- Values: `ACTIVE`, `CHECKMATE`, `STALEMATE`, `RESIGNED`. *Why:* requirement 8's endings. Checkmate and resignation have a winner, and stalemate is a draw.

## 6. Classes (smallest dependency first)

### Position
**Why this class exists:** a square is two numbers that belong together, and many classes need to read it, compare it, and step from it.
- Attributes: `file: int`, `rank: int`. *Why:* the column (0 to 7 for a to h) and the row (0 to 7 for 1 to 8).
- Constructor: `Position(file: int, rank: int)`. *Why:* both set once. It refuses a value outside 0 to 7, so an off-board square can't exist. Two positions with the same numbers are equal, so a position works as a map key.
- Methods:
  - `get_file() -> int`, `get_rank() -> int`. *Why:* read-only views.
  - `offset(file_step: int, rank_step: int) -> Position or None`. *Why:* the square that many steps away, or None if it is off the board. Every piece uses it, so the edge check is in one place.
  - `to_notation() -> str`. *Why:* such as `e4`.
  - `from_notation(text: str) -> Position or None`. *Why:* turns what a player types into a square. None if the text is not a real square.
- *Extensibility:* none needed.

### Move
**Why this class exists:** a move is data: what moved, from where to where, what it took, and whether it was special. As a fixed record it can be tested, compared, applied to a board and stored in the history.
- Attributes: `color: Color`, `piece_type: PieceType`, `from_pos: Position`, `to_pos: Position`, `kind: MoveKind`, `captured: PieceType or None`, `promotion: PieceType or None`. *Why each:* who moved what, between which squares, the kind of move, what was taken (None if nothing), and the new piece for a promotion.
- Constructor: `Move(color: Color, piece_type: PieceType, from_pos: Position, to_pos: Position, kind: MoveKind, captured: PieceType or None, promotion: PieceType or None)`. *Why:* all set once. It refuses a promotion that is set on a non-promotion move, or a promotion to a king or a pawn.
- Methods: getters for each field, and `to_notation() -> str`. *Why `to_notation`:* such as `e2-e4`, `e5xd6`, `O-O` or `e7-e8=Q`.
- *Extensibility:* none needed.

### Piece (abstract)
**Why this class exists:** every piece has a colour and remembers whether it has moved (castling and the pawn's double step need it), and every piece answers the same two questions in its own way.
- Attributes: `color: Color`, `has_moved: bool`. *Why:* whose piece it is, and whether it has left its first square.
- Constructor: `Piece(color: Color, has_moved: bool)`. *Why:* both set once. A piece never changes. When it moves, the board makes a new one with `has_moved` set.
- Methods:
  - `get_color() -> Color`, `has_moved() -> bool`. *Why:* read-only views.
  - `get_type() -> PieceType` (abstract). *Why:* each kind says what it is.
  - `moved() -> Piece` (abstract). *Why:* returns the same kind of piece with `has_moved` set. This is how a piece changes without anyone editing it.
  - `attacked_squares(board: Board, from_pos: Position) -> list[Position]` (abstract). *Why:* the squares this piece threatens. Check detection uses only this, so it never depends on castling or en passant.
  - `candidate_moves(board: Board, from_pos: Position) -> list[Move]`. *Why:* the moves this piece could make before the king-safety test. By default it turns each attacked square that is empty, or holds an opponent piece, into a `NORMAL` move (with `captured` set for an opponent piece). So a piece can never capture its own side. Pawn and King override it.
- *Extensibility:* a new kind of piece is a new subclass.

### SlidingPiece (abstract, extends Piece)
**Why this class exists:** the rook, bishop and queen all move the same way: along a line until they hit something. Writing that walk once means the "no moving through pieces" rule lives in one place.
- Attributes: none of its own.
- Methods:
  - `directions() -> list[(int, int)]` (abstract). *Why:* each subclass lists its steps, such as (1, 0) for a rook moving towards h.
  - `attacked_squares(board: Board, from_pos: Position) -> list[Position]`. *Why:* for each direction, keep stepping with `offset` and add each square. Stop after the first square that holds any piece, which is included because it can be captured. This is requirement 5's "moving through other pieces" refusal.
- *Extensibility:* none needed.

### Rook, Bishop, Queen (extend SlidingPiece)
**Why these classes exist:** each is a sliding piece with its own directions.
- Constructors: `Rook(color: Color, has_moved: bool)`, `Bishop(...)`, `Queen(...)`.
- Methods: `directions()`. *Why:* the rook uses the four straight lines, the bishop the four diagonals, and the queen all eight. `get_type()` and `moved()` return their own kind.
- *Extensibility:* none needed.

### Knight (extends Piece)
**Why this class exists:** the knight jumps, so pieces in between don't matter.
- Constructor: `Knight(color: Color, has_moved: bool)`.
- Methods: `attacked_squares(...)` returns the up to eight L-shaped squares that are on the board. `candidate_moves` uses the default.
- *Extensibility:* none needed.

### King (extends Piece)
**Why this class exists:** the king steps one square and is the only piece that castles.
- Constructor: `King(color: Color, has_moved: bool)`.
- Methods:
  - `attacked_squares(...)`. *Why:* the up to eight neighbouring squares. Castling is not an attack, so it is not included.
  - `candidate_moves(board: Board, from_pos: Position) -> list[Move]`. *Why:* the default moves, plus a `CASTLE` move on each side where all of these hold: the king has not moved, the rook on that corner exists, is the same colour and has not moved, every square between them is empty, and the king is not in check and does not pass through or land on an attacked square (checked with `board.is_attacked`).
- *Extensibility:* none needed.

### Pawn (extends Piece)
**Why this class exists:** the pawn moves forward but captures diagonally, has a first-move double step, en passant and promotion. It is the one piece whose moves and attacks differ.
- Constructor: `Pawn(color: Color, has_moved: bool)`.
- Methods:
  - `attacked_squares(...)`. *Why:* only the two forward diagonal squares, whatever stands there.
  - `candidate_moves(board: Board, from_pos: Position) -> list[Move]`. *Why:*
    - One square forward if it is empty.
    - Two squares forward if the pawn has not moved and both squares are empty.
    - A diagonal capture onto an opponent piece.
    - `EN_PASSANT` onto the empty diagonal square behind an opponent pawn, only if the board's last move was that pawn's double step landing beside this pawn.
    - Any move that reaches the last rank is turned into four `PROMOTION` moves, one each for queen, rook, bishop and knight.
- *Extensibility:* none needed.

### Board
**Why this class exists:** requirement 1's board is the one owner of where pieces stand. It answers questions about the position, and a move produces a new board.
- Attributes:
  - `squares: map{Position: Piece}` (an occupied square to the piece on it). *Why:* empty squares are simply missing, so there is one record of the position.
  - `last_move: Move or None`. *Why:* en passant depends on the move just made, so the board keeps it.
- Constructor: `Board(squares: map{Position: Piece}, last_move: Move or None)`. *Why:* it keeps its own copy of the map, so nobody can change the board afterwards.
- Methods:
  - `standard() -> Board` (a class method). *Why:* requirement 1. It builds the 32 pieces in their starting squares using a private helper that makes a piece from its type.
  - `get_piece(pos: Position) -> Piece or None`, `get_last_move() -> Move or None`. *Why:* read-only views.
  - `positions_of(color: Color) -> list[Position]`. *Why:* where that side's pieces are.
  - `find_king(color: Color) -> Position`. *Why:* the square of that side's king.
  - `is_attacked(pos: Position, by_color: Color) -> bool`. *Why:* True if any piece of that colour has `pos` in its `attacked_squares`.
  - `is_in_check(color: Color) -> bool`. *Why:* requirement 6. Computed as `is_attacked(find_king(color), color.opposite())`.
  - `with_move(move: Move) -> Board`. *Why:* returns a new board with the move made: the piece goes to its new square as `moved()`, anything there is taken, en passant removes the pawn beside, castling also moves the rook, and promotion puts the new piece on the last rank. `last_move` becomes this move.
  - `legal_moves(color: Color) -> list[Move]`. *Why:* every `candidate_moves` of that side's pieces, keeping only those for which `with_move(move).is_in_check(color)` is False. This is the one test for "leaving the king in check".
- *Extensibility:* a variant start position is another class method like `standard`.

### HistoryEntry
**Why this class exists:** requirement 9 records the moves. Each entry is a fixed record of the move and whether it gave check, so the history can't be rewritten.
- Attributes: `move_number: int`, `move: Move`, `gave_check: bool`. *Why:* the full-move number, the move itself, and whether it checked the opponent.
- Constructor: `HistoryEntry(move_number: int, move: Move, gave_check: bool)`. *Why:* all set once.
- Methods: `get_move_number() -> int`, `get_move() -> Move`, `gave_check() -> bool`, `to_notation() -> str`. *Why `to_notation`:* such as `1. e2-e4` or `1... e7-e5`, with `+` added for check.
- *Extensibility:* none needed.

### Game (the manager and only entry point)
**Why this class exists:** a turn needs the legal moves, the new board, the history and the end check, in that order. It is the one place that knows whose turn it is and whether the game is over.
- Attributes:
  - `board: Board`. *Why:* the current position. It is replaced, never edited, after each move.
  - `players: map{Color: str}` (a colour to the player's name). *Why:* who plays which side.
  - `turn: Color`. *Why:* whose move it is.
  - `status: GameStatus`, `winner: Color or None`. *Why:* how the game stands, and who won. The winner is None while active and after stalemate.
  - `history: list[HistoryEntry]`. *Why:* every move played, in order.
- Constructor: `Game(white_name: str, black_name: str)`. *Why:* it refuses empty or identical names. The board is `Board.standard()`, white moves first, and the status is `ACTIVE`.
- Methods:
  - `make_move(player_name: str, from_square: str, to_square: str, promotion: PieceType or None) -> bool`. *Why:* the outer method takes what a player types. Steps in section 9.
  - `resign(player_name: str) -> bool`. *Why:* requirement 8. False if the game is over or the name is unknown. Otherwise the status is `RESIGNED` and the other side wins.
  - `is_in_check() -> bool`. *Why:* whether the side to move is in check, computed from the board.
  - `get_legal_moves(from_square: str) -> list[Move]`. *Why:* what a player may do with one piece, for a display. Empty if it is not their piece or the game is over.
  - `get_status() -> GameStatus`, `get_winner() -> Color or None`, `get_turn() -> Color`. *Why:* read-only views.
  - `get_history() -> list[HistoryEntry]` (returns a copy). *Why:* requirement 9.
- *Extensibility:* none needed.

## 7. Relationships
- Game **has** Board (the current board belongs to the game), HistoryEntry (created inside `make_move`).
- Board **uses** Piece (pieces never change, so the same piece objects are shared by the old board and the new one).
- Board **uses** Move (the last move is a shared record).
- HistoryEntry **uses** Move.
- Move **uses** Position. Board **uses** Position (the map keys).
- SlidingPiece **extends** Piece. Rook, Bishop, Queen **extend** SlidingPiece. Knight, King, Pawn **extend** Piece.
- Enums are always **uses**: Piece to Color; Move to Color, PieceType and MoveKind; Game to Color and GameStatus.

## 8. Interaction

### Layers
1. **Entry point and manager:** Game.
2. **The position:** Board.
3. **The rules of each piece:** Piece and its subclasses.
4. **Plain data:** Position, Move, HistoryEntry.

### Call chains
- **Make a move**
  1. `Player -> Game.make_move(name, "e2", "e4", None)`.
  2. `Game -> Position.from_notation(...)`: what was typed becomes squares.
  3. `Game -> Board.legal_moves(turn)`: only legal moves are considered.
  4. `Board -> Piece.candidate_moves(board, pos)` for each of the mover's pieces, then `Board.with_move(move).is_in_check(turn)` to drop unsafe ones.
  5. `Board.is_in_check -> Board.is_attacked -> Piece.attacked_squares(...)` for each opponent piece.
  6. `Game -> Board.with_move(chosen)`: the new board replaces the old.
  7. `Game -> Board.is_in_check(opponent)` and `Board.legal_moves(opponent)`: check, checkmate or stalemate.

### Who may call whom
- Players call `Game` only, and never `Board` or a piece.
- `Game` calls `Board` and `Position`.
- `Board` calls the pieces. Pieces read the board (`get_piece`, `get_last_move`, `is_attacked`) but never change it.
- No class calls back up to its owner, so there are no cycles.

## 9. Flows

### Make a move
1. `make_move(player_name: str, from_square: str, to_square: str, promotion: PieceType or None)`. If the status is not `ACTIVE`, return False.
2. If `player_name` is not the player of `turn`, return False. *Why:* white first, then strict turns.
3. Turn the two squares into `Position`s. If either is None, return False.
4. `moves = board.legal_moves(turn)`. Keep the moves from `from_pos` to `to_pos`. If none, return False. *Why:* this one check refuses wrong piece rules, blocked paths, own captures, and leaving the king in check.
5. If those moves are promotions, pick the one whose `promotion` equals the request, or the queen if `promotion` is None. If the request matches none, return False.
6. `board = board.with_move(move)`. Add a `HistoryEntry` with `gave_check = board.is_in_check(turn.opposite())`. Then `turn` becomes the opponent.
7. If the side to move has no legal moves: the status becomes `CHECKMATE` with the previous mover as winner if that side is in check, or `STALEMATE` if not.
8. Return True.
- **Failure and undo:** steps 1 to 5 change nothing. After step 6 nothing can fail. Because the board is never changed in place, testing a move never needs an undo.

### Resign
1. `resign(player_name: str)`. If the game is over or the name is unknown, return False. Otherwise `RESIGNED`, and the other side wins.

### Concurrency
- No requirement mentions concurrent use, so no lock is added.
- The board never changes in place, so it is safe to share and to read from several threads. Only `Game` changes, and one lock around `make_move` and `resign` would be enough if needed.

## 10. Extensibility

| Likely change | What to add | Existing code edited? |
|---|---|---|
| A new kind of piece (a chess variant) | A new subclass of `Piece`, one new `PieceType` value, and one line in the board's private piece-building helper | The helper gains one line. The rest is new code. |
| A different start position (Chess960) | A new class method on `Board` beside `standard` | No logic changes. |
| Other draw rules (repetition, 50 moves) | A new check in step 7 of `make_move`, and a counter or a list of past positions in `Game` | Yes, step 7. It is the one place endings are decided. |
| A computer opponent | A new class that reads `get_legal_moves` and calls `make_move` like a player | No. |

## 11. Interview Notes

### Key decisions
1. **A move is legal only if your king is safe after it, tested on a copy.**
   - *Why:* one test covers pins, discovered checks, moving into check and en passant that exposes the king.
   - *Rejected:* separate rules for each case, which are easy to get wrong.
2. **The board never changes in place.**
   - *Why:* testing a move needs no undo, and a bug can't leave half a move on the board.
   - *Rejected:* make and unmake on one board. It is faster, but forgetting to restore one detail (a captured piece, a castling flag) breaks the game.
3. **`attacked_squares` is separate from `candidate_moves`.**
   - *Why:* a pawn attacks differently from how it moves, and castling is not an attack. Check detection uses only attacks, so it can't loop back into castling.
   - *Rejected:* using the move list to detect check.
4. **Pieces never change. Moving makes a new piece with `has_moved` set.**
   - *Why:* castling rights and the pawn's double step come from one fact, and old boards stay valid.
   - *Rejected:* a flag set on the existing piece.
5. **Sliding is written once in `SlidingPiece`.**
   - *Why:* "no moving through pieces" lives in one place for three kinds.
   - *Rejected:* the same walk in the rook, bishop and queen.

### Edge cases handled
- A pinned piece can't move off the line to its king.
- En passant is allowed only on the move right after the double step.
- Castling is refused if the king or that rook has moved, a square between is occupied, or the king is in, passes through, or lands in check.
- A pawn reaching the last rank must promote, to a queen if no choice is given. A king or pawn as the choice is refused.
- Checkmate and stalemate both mean no legal moves. Check decides which.
- Moving out of turn, moving the other side's piece, or moving after the game ends is refused.

### Likely follow-up questions
1. **"Isn't copying the board for every test slow?"** It costs one small map copy per candidate move, which is fine for a game between people. An engine would switch to make and unmake, or to bitboards, behind the same `Board` methods.
2. **"How would you add draw by repetition?"** Keep a count of each position in `Game`, using a key built from the board and whose turn it is, and add a check in step 7.
3. **"How would you add undo?"** Keep the list of past boards in `Game`. Because boards never change, undo is just going back to the previous one.

## 12. Scope Boundaries

| Not covered | One-line answer if asked |
|---|---|
| Draw by agreement, repetition, 50 moves or too little material | Stalemate is the draw covered. See follow-up question 2. |
| Clocks and time limits | No requirement asks for them. |
| Standard algebraic notation (Nf3) | The history uses coordinate notation such as `g1-f3`. |
| Undo | See follow-up question 3. |
| A computer opponent | See the last row of section 10. |
| Concurrent games | Each `Game` is separate, and boards are safe to share. |

## 13. Requirements Met
1. An 8x8 board in the standard position: `Board.standard` and `Position`.
2. Alternate turns, white first: `Game.turn` and the check in `make_move`.
3. Each piece's own rules: `Piece` and its subclasses.
4. Capturing: `Move.captured`, the default `candidate_moves`, and `Board.with_move`.
5. Rejecting illegal moves: `SlidingPiece.attacked_squares` (blocking), `candidate_moves` (own pieces), and `Board.legal_moves` (king safety).
6. Check, checkmate and stalemate: `Board.is_in_check`, `legal_moves`, and step 7 of `make_move`.
7. Castling, en passant and promotion: `King.candidate_moves`, `Pawn.candidate_moves`, `MoveKind`, and `Board.with_move`.
8. Win, draw or resignation: `GameStatus`, `winner` and `Game.resign`.
9. History: `HistoryEntry` and `Game.get_history`.

## 14. Requirement Fit Check

**Setup used in every example:** `Game("Alice", "Bob")`, so Alice is white and Bob is black.

1. **Standard 8x8 start**
   - *Success:* the board has 32 pieces: the white king on e1, the black queen on d8, white pawns on a2 to h2, black pawns on a7 to h7.
   - *Failure:* `Position.from_notation("i9")` returns None, so a square off the board can't be named.
2. **Alternate turns, white first**
   - *Success:* `make_move("Alice", "e2", "e4", None)` returns True, and it is now black's turn.
   - *Failure:* `make_move("Bob", "e7", "e5", None)` at the start returns False because white moves first. After e2-e4, `make_move("Alice", "d2", "d4", None)` returns False because it is black's turn.
3. **Each piece's own rules**
   - *Success:* `g1` to `f3` (a knight's L) returns True.
   - *Failure:* `g1` to `g3` returns False because a knight can't move straight. `a1` to `a3` returns False because the rook is blocked by the pawn on a2.
4. **Capturing**
   - *Success:* after 1. e2-e4 d7-d5, `make_move("Alice", "e4", "d5", None)` captures the pawn, and the history shows `e4xd5`.
   - *Failure:* `a1` to `a2` returns False because the rook can't capture its own pawn.
5. **Rejecting illegal moves**
   - *Success:* after 1. e4 e5 2. Nf3 d6 3. Bb5, black is in check along b5-c6-d7-e8. `c7` to `c6` blocks it and returns True.
   - *Failure:* in that position `a7` to `a6` returns False because it leaves the king in check. At the start, `f1` to `a6` returns False because the bishop can't pass through the pawn on e2.
6. **Check, checkmate, stalemate**
   - *Success (checkmate):* 1. f2-f3 e7-e5 2. g2-g4 d8-h4. White is in check with no legal moves, so the status is `CHECKMATE` and the winner is black.
   - *Success (stalemate):* the ten-move line 1. e3 a5 2. Qh5 Ra6 3. Qxa5 h5 4. h4 Rah6 5. Qxc7 f6 6. Qxd7+ Kf7 7. Qxb7 Qd3 8. Qxb8 Qh7 9. Qxc8 Kg6 10. Qe6 leaves black not in check with no legal moves, so the status is `STALEMATE` with no winner.
   - *Failure:* after 3. Bb5, `is_in_check()` is True but black has legal moves, so the game stays `ACTIVE`.
7. **Castling, en passant, promotion**
   - *Success (castling):* after 1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5, `make_move("Alice", "e1", "g1", None)` castles: the king ends on g1 and the rook on f1.
   - *Success (en passant):* after 1. e4 a6 2. e5 d5, `make_move("Alice", "e5", "d6", None)` captures en passant, and the black pawn on d5 is removed.
   - *Success (promotion):* a white pawn on b7 moving to b8 with `promotion = KNIGHT` becomes a knight. With `None` it becomes a queen.
   - *Failure:* castling after the king has moved and come back returns False. The same en passant one move later returns False. A promotion to `KING` returns False.
8. **Win, draw or resignation**
   - *Success:* the checkmate above is a win for black. The stalemate above is a draw. `resign("Bob")` makes the status `RESIGNED` with white as the winner.
   - *Failure:* `make_move` and `resign` after the game is over both return False.
9. **History**
   - *Success:* after the checkmate in example 6, `get_history()` has four entries: `1. f2-f3`, `1... e7-e5`, `2. g2-g4`, `2... d8-h4+`.
   - *Failure:* a refused move adds nothing to the history.
