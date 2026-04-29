// Go's channels are a natural fit for actors. The buffered channel is the mailbox, and range over a channel provides the message loop. Closing the channel signals shutdown, causing the range to exit cleanly. This pattern is idiomatic Go.\

package coordination

type Actor[T any] struct {
    mailbox chan T
    handler func(T)
    done    chan struct{}
}

func NewActor[T any](handler func(T)) *Actor[T] {
    a := &Actor[T]{
        mailbox: make(chan T, 100),
        handler: handler,
        done:    make(chan struct{}),
    }
    go a.run()
    return a
}

func (a *Actor[T]) Send(message T) {
    a.mailbox <- message
}

func (a *Actor[T]) Stop() {
    close(a.mailbox)
    <-a.done
}

func (a *Actor[T]) run() {
    for message := range a.mailbox {
        a.handler(message)
    }
    close(a.done)
}
