package main

type DiscColor string

const (
	Red    DiscColor = "RED"
	Yellow DiscColor = "YELLOW"
)

type Board struct {
	rows int
	cols int
	grid [][]*DiscColor
}

func NewBoard(rows, cols int) *Board {
	grid := make([][]*DiscColor, rows)
	for r := 0; r < rows; r++ {
		grid[r] = make([]*DiscColor, cols)
	}
	return &Board{
		rows: rows,
		cols: cols,
		grid: grid,
	}
}

func (b *Board) GetRows() int {
	return b.rows
}

func (b *Board) GetCols() int {
	return b.cols
}

func (b *Board) CanPlace(column int) bool {
	if column < 0 || column >= b.cols {
		return false
	}
	return b.grid[0][column] == nil
}

func (b *Board) PlaceDisc(column int, color DiscColor) int {
	if !b.CanPlace(column) {
		return -1
	}

	for row := b.rows - 1; row >= 0; row-- {
		if b.grid[row][column] == nil {
			colorCopy := color
			b.grid[row][column] = &colorCopy
			return row
		}
	}
	return -1
}

func (b *Board) CheckWin(row, column int, color DiscColor) bool {
	if !b.inBounds(row, column) {
		return false
	}

	cell := b.grid[row][column]
	if cell == nil || *cell != color {
		return false
	}

	directions := [][2]int{
		{0, 1},
		{1, 0},
		{1, 1},
		{-1, 1},
	}

	for _, dir := range directions {
		count := 1
		count += b.countInDirection(row, column, dir[0], dir[1], color)
		count += b.countInDirection(row, column, -dir[0], -dir[1], color)
		if count >= 4 {
			return true
		}
	}
	return false
}

func (b *Board) IsFull() bool {
	for c := 0; c < b.cols; c++ {
		if b.grid[0][c] == nil {
			return false
		}
	}
	return true
}

func (b *Board) GetCell(row, column int) *DiscColor {
	if !b.inBounds(row, column) {
		return nil
	}
	return b.grid[row][column]
}

func (b *Board) countInDirection(row, column, dr, dc int, color DiscColor) int {
	count := 0
	r := row + dr
	c := column + dc
	for b.inBounds(r, c) && b.grid[r][c] != nil && *b.grid[r][c] == color {
		count++
		r += dr
		c += dc
	}
	return count
}

func (b *Board) inBounds(row, column int) bool {
	return row >= 0 && row < b.rows && column >= 0 && column < b.cols
}
