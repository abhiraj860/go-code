// Inheritance: Compose behaviour, don't inherit it. Reach for interface first, use inheritance only when sharing stable implementation

package main

type BankAccount struct {
	Balance float64
}

func (a *BankAccount) Deposit(amount float64) {
	a.Balance += amount
}

func (a *BankAccount) Withdraw(amount float64) bool {
	if a.Balance < amount {
		return false
	}
	a.Balance -= amount
	return true
}

func (a *BankAccount) GetBalance() float64 {
	return a.Balance
}

type SavingsAccount struct {
	BankAccount
	InterestRate float64
}

type CheckingAccount struct {
	BankAccount
	OverdraftLimit float64
}