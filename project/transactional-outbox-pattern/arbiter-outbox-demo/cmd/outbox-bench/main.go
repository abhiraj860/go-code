// Command outbox-bench measures data loss for two write strategies against
// the same injected "process crash" fault rate:
//
//	NAIVE:  write ticket to MySQL, then separately call Kafka publish.
//	        A crash between the two steps loses the event permanently.
//
//	OUTBOX: write ticket + outbox row atomically, then a relay polls and
//	        publishes with retries across cycles. A crash during a publish
//	        attempt just delays delivery to the next cycle — it does not
//	        lose the event, because the outbox row is durable.
//
// Run:
//
//	go run ./cmd/outbox-bench                       (single run, default params)
//	go run ./cmd/outbox-bench -sweep                 (sweep crash-prob 0..0.5, writes CSV)
//	go run ./cmd/outbox-bench -tickets 5000 -crash-prob 0.05 -relay-cycles 30
package main

import (
	"flag"
	"fmt"
	"os"

	"arbiter-outbox-demo/internal/broker"
	"arbiter-outbox-demo/internal/model"
	"arbiter-outbox-demo/internal/relay"
	"arbiter-outbox-demo/internal/store"
)

type result struct {
	crashProb        float64
	naiveLossPct     float64
	outboxLossPct    float64
	outboxCyclesUsed int
}

func runNaive(numTickets int, crashProb float64, seed int64) (lossPct float64) {
	b := broker.NewFaultyBroker(seed)
	s := store.NewMySQLStore()

	for i := 0; i < numTickets; i++ {
		id := fmt.Sprintf("T-%d", i)
		t := model.Ticket{ID: id, Payload: "payload-" + id}

		// Step 1: durable DB write (this always succeeds in this model —
		// the DB itself isn't what's flaky; the gap between steps is).
		_ = s.WriteTicketOnly(t)

		// Step 2: separate, non-transactional publish call. If the process
		// "crashes" here, nothing durable recorded that this publish was
		// ever supposed to happen -> permanent loss. No retry is possible
		// because there is nothing to retry FROM.
		msg := model.Message{Key: id, Payload: t.Payload}
		b.PublishWithCrashChance(msg, crashProb)
	}

	delivered := b.DeliveredCount()
	lost := numTickets - delivered
	return 100 * float64(lost) / float64(numTickets)
}

func runOutbox(numTickets int, crashProb float64, relayCycles int, batchSize int, seed int64) (lossPct float64, cyclesUsed int) {
	b := broker.NewFaultyBroker(seed)
	s := store.NewMySQLStore()

	for i := 0; i < numTickets; i++ {
		id := fmt.Sprintf("T-%d", i)
		t := model.Ticket{ID: id, Payload: "payload-" + id}
		evt := model.OutboxEvent{
			ID:       "EVT-" + id,
			TicketID: id,
			Payload:  t.Payload,
		}
		// Single atomic transaction: ticket + outbox row together.
		_ = s.WriteTicketWithOutbox(t, evt)
	}

	r := &relay.Relay{Store: s, Broker: b}

	cycle := 0
	for ; cycle < relayCycles; cycle++ {
		if s.PendingCount() == 0 {
			break
		}
		r.RunCycle(batchSize, crashProb)
	}

	delivered := b.DeliveredCount()
	lost := numTickets - delivered
	return 100 * float64(lost) / float64(numTickets), cycle
}

func main() {
	tickets := flag.Int("tickets", 5000, "number of tickets to simulate")
	crashProb := flag.Float64("crash-prob", 0.05, "probability of a process crash at each publish attempt")
	relayCycles := flag.Int("relay-cycles", 50, "max relay poll cycles for the outbox path")
	batchSize := flag.Int("batch-size", 200, "rows the relay pulls per poll cycle")
	sweep := flag.Bool("sweep", false, "sweep crash-prob from 0 to 0.5 and write results/outbox_sweep.csv")
	seed := flag.Int64("seed", 42, "random seed")
	flag.Parse()

	fmt.Println("=== Arbiter Outbox Pattern Benchmark ===")
	fmt.Printf("tickets=%d crash-prob=%.2f relay-cycles=%d seed=%d\n\n", *tickets, *crashProb, *relayCycles, *seed)

	if !*sweep {
		naiveLoss := runNaive(*tickets, *crashProb, *seed)
		outboxLoss, cyclesUsed := runOutbox(*tickets, *crashProb, *relayCycles, *batchSize, *seed)

		fmt.Printf("%-30s %12s %15s\n", "Strategy", "Loss %", "Notes")
		fmt.Printf("%-30s %11.3f%% %15s\n", "Naive dual-write", naiveLoss, "-")
		fmt.Printf("%-30s %11.3f%% %15s\n", "Transactional outbox", outboxLoss, fmt.Sprintf("converged in %d cycles", cyclesUsed))
		fmt.Println()
		if outboxLoss == 0 {
			fmt.Println("Result: outbox pattern achieved ZERO permanent data loss at this crash rate.")
		} else {
			fmt.Printf("Result: outbox pattern still had %.3f%% pending after %d cycles — increase -relay-cycles.\n", outboxLoss, *relayCycles)
		}
		return
	}

	// Sweep mode: vary crash probability, run both strategies at each
	// point, write a CSV you can plot (e.g. in a spreadsheet) to show the
	// naive approach's loss scaling linearly with crash rate while the
	// outbox stays flat at ~0%.
	var results []result
	for cp := 0.0; cp <= 0.5; cp += 0.05 {
		naiveLoss := runNaive(*tickets, cp, *seed)
		outboxLoss, cycles := runOutbox(*tickets, cp, *relayCycles, *batchSize, *seed)
		results = append(results, result{cp, naiveLoss, outboxLoss, cycles})
		fmt.Printf("crash-prob=%.2f  naive-loss=%.3f%%  outbox-loss=%.3f%%  (cycles=%d)\n", cp, naiveLoss, outboxLoss, cycles)
	}

	if err := os.MkdirAll("results", 0755); err != nil {
		fmt.Println("could not create results dir:", err)
		return
	}
	f, err := os.Create("results/outbox_sweep.csv")
	if err != nil {
		fmt.Println("could not write csv:", err)
		return
	}
	defer f.Close()
	fmt.Fprintln(f, "crash_prob,naive_loss_pct,outbox_loss_pct,outbox_cycles_used")
	for _, r := range results {
		fmt.Fprintf(f, "%.2f,%.4f,%.4f,%d\n", r.crashProb, r.naiveLossPct, r.outboxLossPct, r.outboxCyclesUsed)
	}
	fmt.Println("\nWrote results/outbox_sweep.csv")
}
