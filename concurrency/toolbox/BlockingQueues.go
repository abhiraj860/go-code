package toolbox

import "fmt"

type Task struct {
	taskID int
	taskName string
	taskType string
}


// Go uses buffered channels as blocking queues
func ThreadSafeProducerConsumer() {
	task := Task{
		taskID : 12312312,
		taskName : "Email sent to John",
		taskType : "Send Email",
	}
	queue := make(chan Task, 100)
	queue <- task // Blocks if channel is full
	t := <-queue // Blocks if channel is empty
	fmt.Println(t.taskName)
}
