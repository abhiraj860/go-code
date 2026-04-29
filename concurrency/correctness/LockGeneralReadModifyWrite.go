package correctness

import "sync"

type BankAccount struct {
	mu      sync.Mutex
	balance int
}

func (ba *BankAccount) Deposit(amount int) {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.balance = ba.balance + amount
}

func (ba *BankAccount) Withdraw(amount int) {
	ba.mu.Lock()
	defer ba.mu.Unlock()
	ba.balance = ba.balance - amount
}
