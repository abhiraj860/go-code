package main

import (
	"container/heap"
	"fmt"
	"math"
)

// ─────────────────────────────────────────────
// SPLITWISE: Minimize number of transactions (priority-queue version)
//
// Why a heap instead of the slice-scan version:
//   The original settle() walked creditors/debtors with two index
//   pointers (i, j) over UNSORTED slices. That's not actually greedy —
//   "pair index i with index j" has no relationship to "pair the
//   biggest creditor with the biggest debtor." It happened to produce
//   *a* valid settlement (balances still net to zero) but not
//   necessarily the locally-greedy one, and result order was at the
//   mercy of Go's map iteration, which is randomized.
//
//   A max-heap on each side gives O(log n) access to the current
//   largest creditor/debtor on every step, so the greedy invariant
//   (always settle biggest-against-biggest) actually holds.
//
// Step 1: Compute net balance per person.
// Step 2: Push positives into a creditor max-heap, negatives into a
//         debtor max-heap (ordered by absolute value).
// Step 3: Pop top of each, settle min(|debt|, |credit|), push back
//         whichever side has leftover balance, repeat.
//
// Time:  O(n log n)  — n pushes/pops, each O(log n)
// Space: O(n)
// ─────────────────────────────────────────────

const epsilon = 1e-9 // floating point tolerance for "settled to zero"

type Transaction struct {
	from   string
	to     string
	amount float64
}

// entry is a heap node. Both heaps reuse this type; each heap only ever
// holds entries of one sign (all-positive or all-negative), so ordering
// by absolute value is equivalent to ordering by "size of the balance"
// within that group.
type entry struct {
	name   string
	amount float64
}

// maxHeap orders entries by descending absolute amount — the heap.Interface
// the stdlib expects is a min-heap by default, so Less is inverted here.
type maxHeap []*entry

func (h maxHeap) Len() int            { return len(h) }
func (h maxHeap) Less(i, j int) bool  { return math.Abs(h[i].amount) > math.Abs(h[j].amount) }
func (h maxHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *maxHeap) Push(x interface{}) { *h = append(*h, x.(*entry)) }
func (h *maxHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

func settle(balances map[string]float64) []Transaction {
	creditors := &maxHeap{}
	debtors := &maxHeap{}

	for name, bal := range balances {
		switch {
		case bal > epsilon:
			heap.Push(creditors, &entry{name, bal})
		case bal < -epsilon:
			heap.Push(debtors, &entry{name, bal})
		}
	}

	var result []Transaction

	for creditors.Len() > 0 && debtors.Len() > 0 {
		creditor := heap.Pop(creditors).(*entry)
		debtor := heap.Pop(debtors).(*entry)

		amount := math.Min(creditor.amount, -debtor.amount)

		result = append(result, Transaction{
			from:   debtor.name,
			to:     creditor.name,
			amount: amount,
		})

		creditor.amount -= amount
		debtor.amount += amount // debtor.amount is negative, so this moves it toward 0

		// Whoever still has a nonzero balance goes back in their heap
		// to be re-compared against the next round's max on the other side.
		if creditor.amount > epsilon {
			heap.Push(creditors, creditor)
		}
		if debtor.amount < -epsilon {
			heap.Push(debtors, debtor)
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
		"Bob":     -300,
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
		{"Alice", 900, []string{"Alice", "Bob", "Charlie"}},   // each owes 300
		{"Bob", 600, []string{"Bob", "Charlie"}},              // each owes 300
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
		"A": 0,
		"B": 0,
		"C": 0,
	}
	txns3 := settle(balances3)
	if len(txns3) == 0 {
		fmt.Println("No transactions needed — all settled!")
	}

	// ── Example 4: Where greedy-by-size actually matters ──────────
	// Demonstrates why "biggest creditor vs biggest debtor" (not index order)
	// minimizes transaction count: 4 people, but it still resolves in 3 txns
	// (n-1 is the theoretical floor for n non-zero balances).
	fmt.Println("\n=== Example 4: Mixed magnitudes ===")

	balances4 := map[string]float64{
		"Dave":  500,
		"Erin":  -50,
		"Frank": -200,
		"Grace": -250,
	}
	txns4 := settle(balances4)
	printTransactions(txns4)
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