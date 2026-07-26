// Command idempotency-bench measures wasted embedding-generation compute
// from duplicate Kafka redeliveries, with and without a consumer-side
// idempotency (dedup) check before writing to Milvus.
//
// Run:
//
//	go run ./cmd/idempotency-bench
//	go run ./cmd/idempotency-bench -sweep
//	go run ./cmd/idempotency-bench -events 5000 -crash-interval 500 -redelivery-window 20
package main

import (
	"flag"
	"fmt"
	"os"

	"arbiter-outbox-demo/internal/consumer"
	"arbiter-outbox-demo/internal/vectorstore"
)

type result struct {
	crashInterval   int
	redeliverWindow int
	naiveCalls      int
	dedupCalls      int
	wastedPct       float64
}

func runOnce(numEvents, crashInterval, redeliveryWindow int) (naiveCalls, dedupCalls int) {
	stream := consumer.GenerateStreamWithRedeliveries(numEvents, crashInterval, redeliveryWindow)

	naiveVS := vectorstore.NewVectorStore()
	consumer.RunNaiveConsumer(stream, naiveVS)

	dedupVS := vectorstore.NewVectorStore()
	consumer.RunIdempotentConsumer(stream, dedupVS)

	return naiveVS.EmbedCallCount(), dedupVS.EmbedCallCount()
}

func main() {
	events := flag.Int("events", 5000, "number of unique ticket events")
	crashInterval := flag.Int("crash-interval", 500, "consumer commits offset (and may crash) every N messages")
	redeliveryWindow := flag.Int("redelivery-window", 20, "number of uncommitted messages redelivered on crash/rebalance")
	sweep := flag.Bool("sweep", false, "sweep crash-interval and write results/idempotency_sweep.csv")
	flag.Parse()

	fmt.Println("=== Arbiter Vector-Ingestion Idempotency Benchmark ===")
	fmt.Printf("events=%d crash-interval=%d redelivery-window=%d\n\n", *events, *crashInterval, *redeliveryWindow)

	if !*sweep {
		naiveCalls, dedupCalls := runOnce(*events, *crashInterval, *redeliveryWindow)
		wasted := naiveCalls - dedupCalls
		wastedPct := 100 * float64(wasted) / float64(naiveCalls)

		fmt.Printf("%-35s %10d\n", "Embedding calls (no dedup)", naiveCalls)
		fmt.Printf("%-35s %10d\n", "Embedding calls (with dedup)", dedupCalls)
		fmt.Printf("%-35s %10d\n", "Wasted/duplicate calls avoided", wasted)
		fmt.Printf("%-35s %9.2f%%\n", "Compute savings from dedup", wastedPct)
		return
	}

	// Sweep mode: vary crash-interval (i.e. how often the consumer
	// commits offsets) to show how wasted-compute % changes with
	// consumer restart/rebalance frequency. A smaller crash-interval
	// (more frequent restarts relative to a fixed redelivery window)
	// produces a higher wasted % — this lets you locate the interval
	// that matches your real production rebalance frequency and read
	// off the expected savings number.
	var results []result
	for _, ci := range []int{100, 200, 300, 500, 750, 1000, 1500, 2000} {
		naiveCalls, dedupCalls := runOnce(*events, ci, *redeliveryWindow)
		wasted := naiveCalls - dedupCalls
		wastedPct := 100 * float64(wasted) / float64(naiveCalls)
		results = append(results, result{ci, *redeliveryWindow, naiveCalls, dedupCalls, wastedPct})
		fmt.Printf("crash-interval=%-6d naive=%-6d dedup=%-6d wasted=%6.2f%%\n", ci, naiveCalls, dedupCalls, wastedPct)
	}

	if err := os.MkdirAll("results", 0755); err != nil {
		fmt.Println("could not create results dir:", err)
		return
	}
	f, err := os.Create("results/idempotency_sweep.csv")
	if err != nil {
		fmt.Println("could not write csv:", err)
		return
	}
	defer f.Close()
	fmt.Fprintln(f, "crash_interval,redelivery_window,naive_calls,dedup_calls,wasted_pct")
	for _, r := range results {
		fmt.Fprintf(f, "%d,%d,%d,%d,%.4f\n", r.crashInterval, r.redeliverWindow, r.naiveCalls, r.dedupCalls, r.wastedPct)
	}
	fmt.Println("\nWrote results/idempotency_sweep.csv")
}
