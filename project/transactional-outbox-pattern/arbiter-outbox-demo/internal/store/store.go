// Package store simulates the MySQL layer for Arbiter's Priority Engine.
//
// It intentionally exposes TWO write paths so the benchmark can compare
// them head to head:
//
//  1. WriteTicketOnly       -> the NAIVE approach: ticket is written in its
//     own transaction; the caller is expected to separately call the Kafka
//     publish step afterwards. If the process dies between these two calls,
//     the event is gone forever — nothing durable recorded the intent to
//     publish it.
//
//  2. WriteTicketWithOutbox -> the OUTBOX PATTERN: the ticket row AND the
//     outbox_events row are written in a single atomic transaction. Even if
//     the process dies before anything is published to Kafka, the outbox
//     row survives and a relay can pick it up later.
//
// Both paths use the same in-memory map guarded by a mutex to stand in for
// a real MySQL transaction — the important property being modeled is
// atomicity of "both rows commit together, or neither does", which is
// exactly what a real `BEGIN; INSERT ...; INSERT ...; COMMIT;` gives you.
package store

import (
	"fmt"
	"sync"

	"arbiter-outbox-demo/internal/model"
)

type MySQLStore struct {
	mu      sync.Mutex
	tickets map[string]model.Ticket
	outbox  map[string]*model.OutboxEvent
}

func NewMySQLStore() *MySQLStore {
	return &MySQLStore{
		tickets: make(map[string]model.Ticket),
		outbox:  make(map[string]*model.OutboxEvent),
	}
}

// WriteTicketOnly is the NAIVE path: writes the ticket row only.
// Publishing to Kafka is the caller's responsibility, as a separate step.
func (s *MySQLStore) WriteTicketOnly(t model.Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tickets[t.ID] = t
	return nil
}

// WriteTicketWithOutbox is the OUTBOX PATTERN path: writes the ticket row
// and its corresponding outbox row atomically. This is the Go equivalent
// of the single DB transaction shown in incident-service/main.go earlier.
func (s *MySQLStore) WriteTicketWithOutbox(t model.Ticket, evt model.OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Both writes happen while holding the lock -> atomic from the
	// perspective of any other goroutine, mirroring a DB transaction.
	s.tickets[t.ID] = t
	evtCopy := evt
	evtCopy.Status = model.StatusPending
	s.outbox[evt.ID] = &evtCopy
	return nil
}

// PendingOutboxEvents returns up to `limit` pending rows, oldest first
// (insertion order is preserved via the caller's ID scheme in this demo).
func (s *MySQLStore) PendingOutboxEvents(limit int) []*model.OutboxEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	var pending []*model.OutboxEvent
	for _, evt := range s.outbox {
		if evt.Status == model.StatusPending {
			pending = append(pending, evt)
		}
		if len(pending) >= limit {
			break
		}
	}
	return pending
}

// MarkSent flips a row to 'sent' — analogous to
// `UPDATE outbox_events SET status='sent', processed_at=now() WHERE id=$1`.
func (s *MySQLStore) MarkSent(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	evt, ok := s.outbox[id]
	if !ok {
		return fmt.Errorf("outbox event %s not found", id)
	}
	evt.Status = model.StatusSent
	return nil
}

func (s *MySQLStore) TicketCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.tickets)
}

func (s *MySQLStore) PendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, evt := range s.outbox {
		if evt.Status == model.StatusPending {
			n++
		}
	}
	return n
}
