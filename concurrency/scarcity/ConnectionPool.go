// Go channels are the idiomatic blocking queue. A buffered channel make(chan *T, size) blocks senders when full and receivers when empty. Receive with <-ch and send with ch <-.

package main

type ConnectionPool struct {
	connections chan *Connection
}

func NewConnectionPool(poolSize int) *ConnectionPool {
	pool := &ConnectionPool{
		connections: make(chan *Connection, poolSize),
	}
	for i := 0; i < poolSize; i++ {
		pool.connections <- createNewConnection()
	}
	return pool
}

func (p *ConnectionPool) Acquire() *Connection {
	return <-p.connections // Blocks if empty
}

func (p *ConnectionPool) Release(conn *Connection) {
	p.connections <- conn
}

func (p *ConnectionPool) ExecuteQuery(query string) {
	conn := p.Acquire()
	defer p.Release(conn)
	conn.Execute(query)
}
