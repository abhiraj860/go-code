package issuer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	segkafka "github.com/segmentio/kafka-go"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/services/ticket-svc/internal/blob"
	"github.com/abhiraj860/ticketflow/services/ticket-svc/internal/generator"
)

var testKey = []byte("test-signing-key-at-least-32-bytes-long!")

// memStore is an in-memory blob.Store that counts writes, which is how
// idempotency is observed.
type memStore struct {
	mu       sync.Mutex
	objects  map[string][]byte
	putCalls atomic.Int64
	existErr error
}

func newMemStore() *memStore {
	return &memStore{objects: make(map[string][]byte)}
}

func (m *memStore) Put(_ context.Context, key string, data []byte, _ string) error {
	m.putCalls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = data
	return nil
}

func (m *memStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.objects[key]
	if !ok {
		return nil, blob.ErrNotFound
	}
	return v, nil
}

func (m *memStore) Exists(_ context.Context, key string) (bool, error) {
	if m.existErr != nil {
		return false, m.existErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[key]
	return ok, nil
}

func (m *memStore) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.objects)
}

type memPublisher struct {
	mu     sync.Mutex
	topics []string
	err    error
}

func (p *memPublisher) Publish(_ context.Context, topic, _ string, _ []byte, _ map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	p.topics = append(p.topics, topic)
	return nil
}

func (p *memPublisher) published() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.topics)
}

func newIssuer(t *testing.T, store blob.Store, pub Publisher) *Issuer {
	t.Helper()
	gen, err := generator.New(generator.Options{SigningKey: testKey})
	if err != nil {
		t.Fatalf("generator: %v", err)
	}
	return New(Options{
		Generator: gen, Store: store, Publisher: pub,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func orderMsg(t *testing.T, orderID string, seats ...string) segkafka.Message {
	t.Helper()
	env := tfkafka.Envelope[tfkafka.OrderCreated]{
		ID: "evt_" + orderID, Type: tfkafka.TopicOrderCreated,
		AggregateID: orderID, OccurredAt: time.Now(),
		SchemaVersion: tfkafka.CurrentSchemaVersion,
		Payload: tfkafka.OrderCreated{
			OrderID: orderID, UserID: "u1", EventID: "evt-1",
			SeatIDs: seats, TotalMinor: 100000, CurrencyCode: "INR",
		},
	}
	raw, err := tfkafka.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	return segkafka.Message{Value: raw}
}

func TestIssuesOneTicketPerSeat(t *testing.T) {
	store := newMemStore()
	pub := &memPublisher{}
	iss := newIssuer(t, store, pub)

	if err := iss.Handle(context.Background(), orderMsg(t, "ord-1", "S-1", "S-2", "S-3")); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if n := store.count(); n != 3 {
		t.Errorf("stored %d PDFs, want 3", n)
	}
	if got := iss.Stats().Issued; got != 3 {
		t.Errorf("Issued = %d, want 3", got)
	}
	if pub.published() != 3 {
		t.Errorf("published %d ticket.issued events, want 3", pub.published())
	}
}

// THE idempotency test. At-least-once delivery means this runs repeatedly for
// the same order as a matter of course, not as an exception.
func TestRedeliveryDoesNotDuplicateTickets(t *testing.T) {
	store := newMemStore()
	iss := newIssuer(t, store, &memPublisher{})
	msg := orderMsg(t, "ord-1", "S-1", "S-2")

	for i := range 4 {
		if err := iss.Handle(context.Background(), msg); err != nil {
			t.Fatalf("delivery %d: %v", i, err)
		}
	}

	if n := store.count(); n != 2 {
		t.Errorf("stored %d PDFs after 4 deliveries, want 2", n)
	}
	// The rendering itself must be skipped, not merely overwritten -- it is the
	// expensive part of this service.
	if n := store.putCalls.Load(); n != 2 {
		t.Errorf("wrote %d times, want 2 -- replays should skip generation entirely", n)
	}
	if got := iss.Stats().Skipped; got != 6 {
		t.Errorf("Skipped = %d, want 6 (2 seats x 3 replays)", got)
	}
}

// Ticket ids must be a pure function of (order, seat) so a replica with no
// memory of previous work computes the same id.
func TestTicketIDIsDeterministic(t *testing.T) {
	a := generator.TicketID("ord-1", "S-1")
	b := generator.TicketID("ord-1", "S-1")
	if a != b {
		t.Errorf("same inputs produced %q and %q", a, b)
	}
	if generator.TicketID("ord-1", "S-2") == a {
		t.Error("different seats produced the same ticket id")
	}
	if generator.TicketID("ord-2", "S-1") == a {
		t.Error("different orders produced the same ticket id")
	}
}

func TestMalformedMessageIsPermanent(t *testing.T) {
	iss := newIssuer(t, newMemStore(), &memPublisher{})

	err := iss.Handle(context.Background(), segkafka.Message{Value: []byte("{not json")})
	if !errors.Is(err, tfkafka.ErrPermanent) {
		t.Fatalf("err = %v, want ErrPermanent", err)
	}
}

func TestOrderWithNoSeatsIsPermanent(t *testing.T) {
	iss := newIssuer(t, newMemStore(), &memPublisher{})

	if err := iss.Handle(context.Background(), orderMsg(t, "ord-1")); !errors.Is(err, tfkafka.ErrPermanent) {
		t.Fatalf("err = %v, want ErrPermanent", err)
	}
}

// A storage failure must be retryable, never assumed to mean "absent" -- that
// assumption would mint a duplicate ticket.
func TestStorageFailureIsRetryable(t *testing.T) {
	store := newMemStore()
	store.existErr = errors.New("s3 unreachable")
	iss := newIssuer(t, store, &memPublisher{})

	err := iss.Handle(context.Background(), orderMsg(t, "ord-1", "S-1"))
	if err == nil {
		t.Fatal("a storage failure did not fail the message")
	}
	if errors.Is(err, tfkafka.ErrPermanent) {
		t.Error("a transient storage failure was marked permanent; it must be retried")
	}
}

// The ticket is the durable artifact and is already stored, so a failed
// announcement must not fail the message and cause every seat to re-render.
func TestAnnouncementFailureDoesNotFailTheMessage(t *testing.T) {
	store := newMemStore()
	pub := &memPublisher{err: errors.New("broker down")}
	iss := newIssuer(t, store, pub)

	if err := iss.Handle(context.Background(), orderMsg(t, "ord-1", "S-1")); err != nil {
		t.Fatalf("a failed announcement failed the message: %v", err)
	}
	if store.count() != 1 {
		t.Error("the ticket was not stored")
	}
}

func TestPDFIsAWellFormedDocument(t *testing.T) {
	store := newMemStore()
	iss := newIssuer(t, store, &memPublisher{})

	if err := iss.Handle(context.Background(), orderMsg(t, "ord-1", "S-1")); err != nil {
		t.Fatal(err)
	}

	key := generator.PDFKey("evt-1", generator.TicketID("ord-1", "S-1"))
	pdf, err := store.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("reading the stored PDF: %v", err)
	}
	if !strings.HasPrefix(string(pdf), "%PDF-") {
		t.Errorf("stored object is not a PDF (starts with %q)", string(pdf[:min(8, len(pdf))]))
	}
	if len(pdf) < 1000 {
		t.Errorf("PDF is only %d bytes; the QR image is probably missing", len(pdf))
	}
}

// A ticket QR is a bearer token. An unsigned code would let anyone who guesses
// an id walk into the venue.
func TestGateCodeRoundTripsAndRejectsForgery(t *testing.T) {
	gen, err := generator.New(generator.Options{SigningKey: testKey})
	if err != nil {
		t.Fatal(err)
	}

	code := gen.GateCode("tkt-1", "evt-1", "S-1")

	ticketID, eventID, seatID, err := gen.VerifyGateCode(code)
	if err != nil {
		t.Fatalf("verifying a genuine code: %v", err)
	}
	if ticketID != "tkt-1" || eventID != "evt-1" || seatID != "S-1" {
		t.Errorf("round trip = %q/%q/%q", ticketID, eventID, seatID)
	}

	// Tampering with the payload must invalidate the signature.
	tampered := strings.Replace(code, code[:4], "AAAA", 1)
	if _, _, _, err := gen.VerifyGateCode(tampered); err == nil {
		t.Error("a tampered code verified successfully")
	}

	// A code signed with a different key must not verify.
	other, _ := generator.New(generator.Options{SigningKey: []byte("a-completely-different-key-32bytes!!")})
	if _, _, _, err := other.VerifyGateCode(code); err == nil {
		t.Error("a code verified under the wrong signing key")
	}

	for _, bad := range []string{"", "no-dot", "aaa.bbb", "....."} {
		if _, _, _, err := gen.VerifyGateCode(bad); err == nil {
			t.Errorf("malformed code %q verified successfully", bad)
		}
	}
}

func TestGeneratorRequiresAStrongKey(t *testing.T) {
	if _, err := generator.New(generator.Options{SigningKey: []byte("short")}); err == nil {
		t.Error("a 5-byte signing key was accepted")
	}
	if _, err := generator.New(generator.Options{}); err == nil {
		t.Error("an empty signing key was accepted")
	}
}

// Concurrent redeliveries -- the realistic shape when a relay resends and
// several consumer replicas pick it up.
func TestConcurrentDeliveriesAreSafe(t *testing.T) {
	store := newMemStore()
	iss := newIssuer(t, store, &memPublisher{})
	msg := orderMsg(t, "ord-1", "S-1", "S-2")

	var wg sync.WaitGroup
	wg.Add(8)
	for range 8 {
		go func() {
			defer wg.Done()
			_ = iss.Handle(context.Background(), msg)
		}()
	}
	wg.Wait()

	if n := store.count(); n != 2 {
		t.Errorf("stored %d distinct PDFs, want 2", n)
	}
}
