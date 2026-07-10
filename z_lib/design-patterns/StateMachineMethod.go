// Behavioral Patterns
// State Machine: Use when an object's behavior depends on its current state and transitions get messy. 

package main

import "fmt"

type VendingMachineState interface {
	InsertCoin(machine *VendingMachine)
	SelectProduct(machine * VendingMachine)
	Dispense(machine * VendingMachine)
}

type NoCoinState struct{}

func (NoCoinState) InsertCoin(machine *VendingMachine) {
	fmt.Println("Coin inserted")
	machine.setState(HasCoinState{})
}

func (NoCoinState) SelectProduct(machine *VendingMachine) {
	fmt.Println("Insert coin first")
}

func (NoCoinState) Dispense(machine *VendingMachine) {
	fmt.Println("Insert coin first")
}

type HasCoinState struct{}

func (HasCoinState) InsertCoin(machine *VendingMachine) {
	fmt.Println("Coin already inserted")
}

func (HasCoinState) SelectProduct(machine * VendingMachine) {
	fmt.Println("Product Selected")
	machine.setState(DispenseState{})
}

func (HasCoinState) Dispense(machine *VendingMachine) {
	fmt.Println("Select product state")
}

type DispenseState struct{}

func (DispenseState) InsertCoin(machine *VendingMachine) {
	fmt.Println("Please, wait, dispensing")
}

func (DispenseState) SelectProduct(machine *VendingMachine) {
	fmt.Println("Please wait, dispensing")
}

func (DispenseState) Dispense(machine *VendingMachine) {
	fmt.Println("Dispensing product")
	machine.setState(NoCoinState{})
}

type VendingMachine struct {
	currentState VendingMachineState
}

func NewVendingMachine() *VendingMachine {
	return &VendingMachine{currentState : NoCoinState{}}
}

func (v *VendingMachine) InsertCoin() {
	v.currentState.InsertCoin(v)
}

func (v *VendingMachine) SelectProduct() {
	v.currentState.SelectProduct(v)
}

func (v *VendingMachine) Dispense() {
	v.currentState.Dispense(v)
}

func (v *VendingMachine) setState(state VendingMachineState) {
	v.currentState = state
}

// Usage
// machine := NewVendingMachine()
// machine.SelectProduct() //"Insert a coin"
// machine.InsertCoin() // "Coin Inserted"
// machine.SelectProduct() // "Product Selected"
// machine.Dispense() // "Dispensing Product"
