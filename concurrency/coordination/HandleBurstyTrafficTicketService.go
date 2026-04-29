// Go's select with a timeout channel provides non-blocking queue insertion. If the send doesn't complete within 100ms, the timeout case executes and returns an error. This is idiomatic Go for handling backpressure on bounded channels.

package coordination

import (
    "errors"
	"fmt"
    "time"
)

type PurchaseRequest struct {
    UserID   string
    EventID  string
    Quantity int
}

type TicketService struct {
    // Sized for 10-second burst at 10,000 req/s
    purchaseQueue chan PurchaseRequest
}

func NewTicketService() *TicketService {
    return &TicketService{
        purchaseQueue: make(chan PurchaseRequest, 100000),
    }
}

// API handler (producer) - handles bursts
func (s *TicketService) PurchaseTicket(userID, eventID string, quantity int) error {
    request := PurchaseRequest{userID, eventID, quantity}

    // Enqueue request - returns immediately even during spike
    select {
    case s.purchaseQueue <- request:
        return nil
    case <-time.After(100 * time.Millisecond):
        return errors.New("too many requests, try again")
    }
}

func processPurchase(request PurchaseRequest) {
	fmt.Printf("%v request purchased", request)
}

// Worker pool sized for normal load (100 workers)
func (s *TicketService) PurchaseWorker() {
    for request := range s.purchaseQueue {
        // Process at normal rate - database, payment, inventory
        processPurchase(request)
    }
}
