package main

import (
	"fmt"
)

type PaymentMethod interface {
	Process(amount float64) bool
}

type CreditCardPayment struct{}
func (CreditCardPayment) Process(amount float64) bool { return true } 

type PayPalPayment struct{}
func (PayPalPayment) Process(amount float64) bool {return true}

func Checkout(method PaymentMethod, amount float64) {
	method.Process(amount)
}

type Room struct {}

func (r *Room) Error() string {
	return "Erer"
}


func (r *Room) Book() error {
	return r 
}
type LogLevel struct {}

func (l LogLevel) String() string {
	return ""
}

type SpotType int

const (
	SPOT_MOTORCYCLE SpotType = iota
	SPOT_CAR 
	SPOT_Large
)

type TaskStatus string

const (
	Pending TaskStatus = "PENDING"
	InProgress TaskStatus = "IN_PROGRESS"
)

type ParkingSpot struct {
	ID string
	SpotType SpotType
	occupied bool // lowercase = encapsulation
}

type BankAccount struct {
	Balance float64
}

func (a *BankAccount) Deposit(amount float64) { a.Balance += amount }

type SavingsAccount struct {
	BankAccount
	InterestRate float64
}





func NewParkingSpot(id string, spotType SpotType) *ParkingSpot {
	return &ParkingSpot{ID : id, SpotType: spotType}
}

func main() {

	fmt.Println(SPOT_CAR)
}

