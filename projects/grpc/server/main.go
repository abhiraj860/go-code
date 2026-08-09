package main

import (
	"context" // carries cancellation info; every gRPC method gets one
	"fmt"
	"log"
	"net"  // listen on a TCP port directly — gRPC isn't net/http
	"sync" // a lock, so concurrent calls don't corrupt the map
	"time"

	"server/shoppb" // the generated package

	"google.golang.org/grpc"
	// These two turn Go errors into proper gRPC status codes.
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Our server implementation. Embedding the generated
// UnimplementedOrderServiceServer means that if we forget a method,
// it still compiles and returns "unimplemented" instead of crashing.
// It's also what lets new methods be added to the proto without
// breaking this file.
type server struct {
	shoppb.UnimplementedOrderServiceServer

	// A lock protecting the map below. gRPC handles each call in its
	// own goroutine, so two calls can touch this at the same time.
	mu     sync.Mutex
	orders map[string]*shoppb.Order
	nextID int
}

// GetOrder implements the rpc of the same name.
// The (s *server) part is the receiver — the value the method runs
// on. The * means it's a pointer, so changes stick.
func (s *server) GetOrder(ctx context.Context, req *shoppb.GetOrderRequest) (*shoppb.Order, error) {
	// Lock before touching shared state; Unlock when we return.
	// defer means "run this when the function ends, whatever happens".
	s.mu.Lock()
	defer s.mu.Unlock()

	// Two-value map read: the second value says whether it was found.
	o, ok := s.orders[req.GetOrderId()]
	if !ok {
		// Return a gRPC status code, NOT a plain error. The client
		// gets a typed code it can branch on — this is the direct
		// equivalent of an HTTP 404.
		return nil, status.Errorf(codes.NotFound,
			"no order %s", req.GetOrderId())
	}

	// nil in the error slot means success.
	return o, nil
}

func (s *server) CreateOrder(ctx context.Context, req *shoppb.CreateOrderRequest) (*shoppb.Order, error) {
	// Validation errors get InvalidArgument — the 400 equivalent.
	if req.GetCustomer() == "" {
		return nil, status.Error(codes.InvalidArgument, "customer is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	// The & means "address of" — protobuf messages are always used
	// as pointers.
	o := &shoppb.Order{
		OrderId:  fmt.Sprintf("ord-%d", s.nextID),
		Customer: req.GetCustomer(),
		Total:    req.GetTotal(),
	}
	s.orders[o.OrderId] = o

	log.Println("created", o.OrderId)
	return o, nil
}

// WatchOrders is a STREAMING method. Note the different signature:
// no return value, because results go out through the stream object.
func (s *server) WatchOrders(req *shoppb.WatchRequest, stream shoppb.OrderService_WatchOrdersServer) error {
	log.Println("client watching orders for", req.GetCustomer())

	// Send five updates, one per second, over a single connection.
	for i := 0; i < 5; i++ {
		// Send pushes one message to the client right now. The
		// client receives it immediately — it doesn't wait for the
		// call to finish.
		err := stream.Send(&shoppb.Order{
			OrderId:  fmt.Sprintf("live-%d", i),
			Customer: req.GetCustomer(),
			Total:    float64(i) * 10,
		})
		if err != nil {
			return err // client hung up
		}

		// Context() lets us notice if the client disconnected.
		// select waits on several channels and takes whichever fires
		// first, so we stop promptly instead of sleeping pointlessly.
		select {
		case <-stream.Context().Done():
			log.Println("client disconnected")
			return nil
		case <-time.After(time.Second):
		}
	}
	// Returning nil ends the stream cleanly.
	return nil
}

func main() {
	// gRPC listens on a raw TCP socket, not through net/http.
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	// Build the gRPC server.
	s := grpc.NewServer()

	// Register our implementation against the generated interface.
	// If our type is missing a method, this line fails to compile —
	// the schema is enforced at build time.
	shoppb.RegisterOrderServiceServer(s, &server{
		// make(...) creates an empty map ready to use.
		orders: make(map[string]*shoppb.Order),
	})

	log.Println("grpc server on :50051")
	// Blocks forever, serving connections.
	log.Fatal(s.Serve(lis))
}
