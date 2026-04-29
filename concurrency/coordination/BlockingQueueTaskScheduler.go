// Go uses buffered channels as blocking queues. make(chan T, capacity) creates a bounded channel. Sends block when full, receives block when empty. Channels are the idiomatic Go solution for producer-consumer coordination.

// Go's channel receive operation (<-ch) blocks when the channel is empty and returns once a value is sent. It returns the zero value and false if the channel is closed.


package coordination

type TaskScheduler struct {
    queue chan func()
}

func NewTaskScheduler() *TaskScheduler {
    return &TaskScheduler{
        queue: make(chan func(), 1000),  // Buffered channel acts as bounded queue
    }
}

func (s *TaskScheduler) SubmitTask(task func()) {
    s.queue <- task  // Blocks if channel is full
}

func (s *TaskScheduler) WorkerLoop() {
    for task := range s.queue {  // Blocks if channel is empty
        task()
    }
}
