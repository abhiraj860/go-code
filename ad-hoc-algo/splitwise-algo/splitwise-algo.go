package main

import (
	"fmt"
	"math"
)

// ─────────────────────────────────────────────
// SPLITWISE: Minimize number of transactions
//
// Step 1: Compute net balance per person
//         net[i] = total_received - total_paid
//         positive = creditor (owed money)
//         negative = debtor  (owes money)
//
// Step 2: Greedy — match max creditor with max debtor
//         settle min(|debt|, |credit|) in one transaction
//         repeat until all balances are zero
//
// Time:  O(n^2) in worst case (n = number of people)
// Space: O(n)
// ─────────────────────────────────────────────

type Transaction struct {
	from   string
	to     string
	amount float64
}

func settle(balances map[string]float64) []Transaction {
	// Separate into creditors (positive) and debtors (negative)
	// Using slices of [name, amount] for in-place mutation
	type entry struct {
		name   string
		amount float64
	}

	var creditors, debtors []entry
	for name, bal := range balances {
		if bal > 0 {
			creditors = append(creditors, entry{name, bal})
		} else if bal < 0 {
			debtors = append(debtors, entry{name, bal})
		}
	}

	var result []Transaction

	i, j := 0, 0
	for i < len(creditors) && j < len(debtors) {
		creditor := &creditors[i]
		debtor := &debtors[j]

		// Settle the smaller of the two amounts
		amount := math.Min(creditor.amount, -debtor.amount)

		result = append(result, Transaction{
			from:   debtor.name,
			to:     creditor.name,
			amount: amount,
		})

		creditor.amount -= amount
		debtor.amount += amount // debtor.amount is negative, so adding brings it toward 0

		// Fully settled — move pointer
		if creditor.amount == 0 {
			i++
		}
		if debtor.amount == 0 {
			j++
		}
	}

	return result
}

// computeNetBalances takes raw expense records and returns net balance per person.
// positive = owed money, negative = owes money
func computeNetBalances(expenses []struct {
	paidBy string
	amount float64
	split  []string // people splitting this expense equally
}) map[string]float64 {
	balances := make(map[string]float64)

	for _, e := range expenses {
		share := e.amount / float64(len(e.split))
		balances[e.paidBy] += e.amount // payer gets full credit
		for _, person := range e.split {
			balances[person] -= share // each splitter owes their share
		}
	}

	return balances
}

func main() {
	// ── Example 1: Direct balances ──────────────────
	// Alice is owed 600, Bob owes 300, Charlie owes 300
	fmt.Println("=== Example 1: Direct net balances ===")

	balances := map[string]float64{
		"Alice":   600,
		"Bob":    -300,
		"Charlie": -300,
	}

	txns := settle(balances)
	printTransactions(txns)

	// ── Example 2: Raw expenses ──────────────────────
	// Alice paid 900 for dinner split 3 ways
	// Bob paid 600 for cab split 2 ways (Bob + Charlie)
	// Charlie paid 300 for drinks split 3 ways
	fmt.Println("\n=== Example 2: From raw expenses ===")

	expenses := []struct {
		paidBy string
		amount float64
		split  []string
	}{
		{"Alice", 900, []string{"Alice", "Bob", "Charlie"}}, // each owes 300
		{"Bob", 600, []string{"Bob", "Charlie"}},            // each owes 300
		{"Charlie", 300, []string{"Alice", "Bob", "Charlie"}}, // each owes 100
	}

	balances2 := computeNetBalances(expenses)
	fmt.Println("Net balances:")
	for name, bal := range balances2 {
		fmt.Printf("  %-10s %+.2f\n", name, bal)
	}

	txns2 := settle(balances2)
	fmt.Println("Transactions:")
	printTransactions(txns2)

	// ── Example 3: Circular debt ─────────────────────
	// A owes B 10, B owes C 10, C owes A 10 → all cancel out
	fmt.Println("\n=== Example 3: Circular debt (net zero) ===")

	balances3 := map[string]float64{
		"A": 0, // owes 10 to B, gets 10 from C → net 0
		"B": 0,
		"C": 0,
	}
	txns3 := settle(balances3)
	if len(txns3) == 0 {
		fmt.Println("No transactions needed — all settled!")
	}
}

func printTransactions(txns []Transaction) {
	if len(txns) == 0 {
		fmt.Println("  No transactions needed.")
		return
	}
	for _, t := range txns {
		fmt.Printf("  %s pays %s %.2f\n", t.from, t.to, t.amount)
	}
}
