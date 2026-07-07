package main

type Compartment struct {
	size     Size
	occupied bool
}

func NewCompartment(size Size) *Compartment {
	return &Compartment{
		size:     size,
		occupied: false,
	}
}

func (c *Compartment) GetSize() Size {
	return c.size
}

func (c *Compartment) IsOccupied() bool {
	return c.occupied
}

func (c *Compartment) MarkOccupied() {
	c.occupied = true
}

func (c *Compartment) MarkFree() {
	c.occupied = false
}

func (c *Compartment) Open() {
}

