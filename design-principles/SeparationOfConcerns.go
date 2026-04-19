// General Principles
// Separation of Concerns - Enable independent testing and changes

package main

type Board interface {
	HasWinner() bool
	MakeMove(move string)
	GetWinner() string
}

type Display interface {
	Render(board Board)
	ShowWinner(winner string)
}

type InputHandler interface {
	NextMove() string
}

type TicTacToeGood struct {
	board Board
	display Display
	inputHandler InputHandler
}

func NewTicTacToeGood(board Board, display Display, inputHandler InputHandler) *TicTacToeGood {
	return &TicTacToeGood{
		board: board,
		display: display,
		inputHandler: inputHandler,
	}
}

func (g *TicTacToeGood) Play() {
	for !g.board.HasWinner() {
		g.display.Render(g.board)
		move := g.inputHandler.NextMove()
		g.board.MakeMove(move)
	}
	g.display.ShowWinner(g.board.GetWinner())
}








