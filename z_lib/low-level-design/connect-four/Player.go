package main

type Player struct {
	Name  string
	Color DiscColor
}

func NewPlayer(name string, color DiscColor) *Player {
	return &Player{
		Name:  name,
		Color: color,
	}
}
