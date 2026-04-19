// SOLID Principles
// LSP - Prevent brittle hierarchies that break at runtime

package main

type BirdGood interface{
	Eat()
}

type FlyingBird interface {
	BirdGood
	Fly()
}

type Sparrow struct{}

func (Sparrow) Eat() {}
func (Sparrow) Fly() {}

type PenguinGood struct{}

func (PenguinGood) Eat() {} 