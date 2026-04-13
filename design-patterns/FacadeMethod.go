// Structural Patterns
// Facade: Use when you want to hide internal complexity behind a simple entry point.
// It's a coordinator that hides the complexity. Below the "Game" struct is a Facade Pattern.

package main

type GameState int

const (
	InProgress GameState = iota
	Won
	Draw
)

type Board struct{}

func (b *Board) PlaceMark(row, col int, mark string) bool {
	// Placeholder for actual placement logic
	return true
}

func (b *Board) CheckWin(row, col int) bool {
	// Placeholder for win detection
	return false
}

func (b *Board) IsFull() bool {
	// Placeholder for board fullness check
	return false
}

type Player struct{
	mark string
}

func NewPlayer(mark string) *Player {
	return &Player{mark: mark}
}

func (p *Player) GetMark() string {
	return p.mark
}

type Game struct {
	board *Board
	playerX *Player
	playerO *Player
	currentPlayer *Player
	state GameState
}

func NewGame() *Game {
	playerX := NewPlayer("X")
	playerO := NewPlayer("O")
	return &Game{
		board: &Board{},
		playerX : playerX,
		playerO : playerO,
		currentPlayer : playerX,
		state : InProgress,
	}
}

func (g *Game) MakeMove(row, col int) bool {
	if g.state != InProgress {
		return false
	}
	if !g.board.PlaceMark(row, col, g.currentPlayer.GetMark()) {
		return false 
	}
	if g.board.CheckWin(row, col) {
		g.state = Won
	} else if g.board.IsFull() {
		g.state = Draw
	} else {
		if g.currentPlayer == g.playerX {
			g.currentPlayer = g.playerO
		} else {
			g.currentPlayer = g.playerX
		}
	}
	return true
}

// Usage
// func main() {
// 	game := NewGame()
// 	game.MakeMove(0, 1)
// 	game.MakeMove(1, 1)
// }