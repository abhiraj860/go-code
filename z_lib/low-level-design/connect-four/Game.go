package main

type GameState string

const (
	InProgress GameState = "IN_PROGRESS"
	Won        GameState = "WON"
	Draw       GameState = "DRAW"
)

type Game struct {
	board         *Board
	player1       *Player
	player2       *Player
	currentPlayer *Player
	state         GameState
	winner        *Player
}

func NewGame(player1, player2 *Player) *Game {
	return &Game{
		board:         NewBoard(6, 7),
		player1:       player1,
		player2:       player2,
		currentPlayer: player1,
		state:         InProgress,
	}
}

func (g *Game) MakeMove(player *Player, column int) bool {
	if g.state != InProgress {
		return false
	}
	if player != g.currentPlayer {
		return false
	}

	row := g.board.PlaceDisc(column, player.Color)
	if row == -1 {
		return false
	}

	if g.board.CheckWin(row, column, player.Color) {
		g.state = Won
		g.winner = player
	} else if g.board.IsFull() {
		g.state = Draw
	} else {
		if g.currentPlayer == g.player1 {
			g.currentPlayer = g.player2
		} else {
			g.currentPlayer = g.player1
		}
	}
	return true
}

func (g *Game) GetCurrentPlayer() *Player {
	return g.currentPlayer
}

func (g *Game) GetGameState() GameState {
	return g.state
}

func (g *Game) GetWinner() *Player {
	return g.winner
}

func (g *Game) GetBoard() *Board {
	return g.board
}
