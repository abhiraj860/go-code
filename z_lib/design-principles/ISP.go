// SOLID Principles
// ISP - Keep interfaces clean and focussed

package main

type Workable interface {
	Work()
}

type Feedable interface {
	Eat()
}

type Restable interface {
	Sleep()
}
 

type Human struct{}

func (Human) Work() {}
func (Human) Eat() {}
func (Human) Sleep() {}

type Robot struct{}

func (Robot) Work() {}
