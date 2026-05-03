// Use a select statement with time.After(duration) to implement timeouts on channel operations. The select picks whichever case is ready first.

package main

import (
	"errors"
	"time"
)

type Connection struct{}

func (c *Connection) Execute(query string) error {
	return nil
}

func createNewConnection() *Connection {
	return &Connection{}
}

type ConnectionPoolWithTimeout struct {
	connections chan *Connection
	timeout     time.Duration
}

func NewConnectionPoolWithTimeout(poolSize int, timeout time.Duration) *ConnectionPoolWithTimeout {
	pool := &ConnectionPoolWithTimeout{
		connections: make(chan *Connection, poolSize),
		timeout:     timeout,
	}
	for i := 0; i < poolSize; i++ {
	    pool.connections <- createNewConnection()
	}
	return pool
}

func (p *ConnectionPoolWithTimeout) Acquire() (*Connection, error) {
	select {
	case conn := <-p.connections:
		return conn, nil
	case <-time.After(p.timeout):
		return nil, errors.New("no connection available within timeout")
	}
}

func (p *ConnectionPoolWithTimeout) ExecuteQuery(query string) error {
	conn, err := p.Acquire()
	if err != nil {
		return err
	}
	defer func() { p.connections <- conn }()
	return conn.Execute(query)
}
