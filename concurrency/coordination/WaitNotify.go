// Solves problem of busy wait : Busy Waiting(anti-pattern)
// Solves problem of sleep-polling : Sleep Polling(anti-pattern) 


// Go idiom: use channels instead of condition variables
// Receive blocks until data is available (like wait)
// Send wakes up receiver (like notify)


package coordination

import "fmt"

type Task struct {
	taskID   int
	taskName string
}

func doWork(task Task) string {
	return fmt.Sprintf("Processed: %s", task.taskName)
}

func ThreadSafeProducerConsumer() {
	taskChannel   := make(chan Task, 100)
	resultChannel := make(chan string, 100)

	go func() {
		task := <-taskChannel       // Blocks until task available
		result := doWork(task)
		resultChannel <- result     // Wakes up receiver
	}()

	taskChannel <- Task{taskID: 1, taskName: "Email sent to John"}
	fmt.Println(<-resultChannel)
}

// Go idiomatically uses channels instead of condition variables. Receiving from a channel blocks until data is available (like wait()). Sending to a channel wakes up a receiver (like notify()). Channels provide cleaner semantics for most coordination problems.