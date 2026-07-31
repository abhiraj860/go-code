package indexer

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	segkafka "github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	catalogv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/catalog/v1"
	"github.com/abhiraj860/ticketflow/services/search-svc/internal/index"
)

type stubCatalog struct {
	event *catalogv1.Event
	err   error
}

func (s *stubCatalog) GetEvent(context.Context, *catalogv1.GetEventRequest, ...grpc.CallOption) (*catalogv1.GetEventResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &catalogv1.GetEventResponse{Event: s.event}, nil
}
func (s *stubCatalog) ListEvents(context.Context, *catalogv1.ListEventsRequest, ...grpc.CallOption) (*catalogv1.ListEventsResponse, error) {
	return &catalogv1.ListEventsResponse{}, nil
}
func (s *stubCatalog) GetSeatMap(context.Context, *catalogv1.GetSeatMapRequest, ...grpc.CallOption) (*catalogv1.GetSeatMapResponse, error) {
	return &catalogv1.GetSeatMapResponse{}, nil
}
func (s *stubCatalog) GetEventContent(context.Context, *catalogv1.GetEventContentRequest, ...grpc.CallOption) (*catalogv1.GetEventContentResponse, error) {
	return &catalogv1.GetEventContentResponse{}, nil
}

type stubWriter struct {
	mu      sync.Mutex
	indexed []index.Document
	deleted []string
	err     error
}

func (w *stubWriter) IndexDocument(_ context.Context, doc index.Document) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	w.indexed = append(w.indexed, doc)
	return nil
}

func (w *stubWriter) DeleteDocument(_ context.Context, id string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	w.deleted = append(w.deleted, id)
	return nil
}

func msgFor(t *testing.T, eventID string) segkafka.Message {
	t.Helper()
	raw, err := tfkafka.Marshal(tfkafka.Envelope[tfkafka.CatalogEventUpdated]{
		ID: "evt_1", Type: tfkafka.TopicCatalogEventUpdated,
		SchemaVersion: tfkafka.CurrentSchemaVersion,
		Payload:       tfkafka.CatalogEventUpdated{EventID: eventID, Version: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	return segkafka.Message{Value: raw}
}

func sampleEvent(status catalogv1.EventStatus) *catalogv1.Event {
	return &catalogv1.Event{
		Id: "evt-1", Title: "Coldplay", Kind: catalogv1.EventKind_EVENT_KIND_CONCERT,
		Status:   status,
		Venue:    &catalogv1.Venue{Id: "v1", Name: "NSCI", City: "Mumbai", CountryCode: "IN"},
		StartsAt: timestamppb.New(time.Now().Add(time.Hour)),
		Tags:     []string{"Rock", " Live-Music "},
		Version:  3,
		PricingTiers: []*catalogv1.PricingTier{
			{Id: "t1", Price: &catalogv1.Money{AmountMinor: 900000, CurrencyCode: "INR"}},
			{Id: "t2", Price: &catalogv1.Money{AmountMinor: 350000, CurrencyCode: "INR"}},
		},
	}
}

func newIndexer(cat catalogv1.CatalogServiceClient, w Writer) *Indexer {
	return New(cat, w, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestIndexesEventReadBackFromCatalog(t *testing.T) {
	w := &stubWriter{}
	idx := newIndexer(&stubCatalog{event: sampleEvent(catalogv1.EventStatus_EVENT_STATUS_ON_SALE)}, w)

	if err := idx.Handle(context.Background(), msgFor(t, "evt-1")); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(w.indexed) != 1 {
		t.Fatalf("indexed %d docs, want 1", len(w.indexed))
	}

	doc := w.indexed[0]
	if doc.Title != "Coldplay" || doc.City != "Mumbai" {
		t.Errorf("doc = %+v", doc)
	}
	// The cheapest tier is denormalised so a listing needs no second call.
	if doc.MinPriceMinor != 350000 {
		t.Errorf("MinPriceMinor = %d, want 350000 (the cheapest tier)", doc.MinPriceMinor)
	}
	// Tags are normalised, or faceting shows "Rock" and "rock" as two filters.
	for _, want := range []string{"rock", "live-music"} {
		var found bool
		for _, got := range doc.Tags {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("tags %v missing normalised %q", doc.Tags, want)
		}
	}
}

// A cancelled event must leave the index entirely, not sit there with a
// status field nobody filters on.
func TestCancelledEventIsDeleted(t *testing.T) {
	w := &stubWriter{}
	idx := newIndexer(&stubCatalog{event: sampleEvent(catalogv1.EventStatus_EVENT_STATUS_CANCELLED)}, w)

	if err := idx.Handle(context.Background(), msgFor(t, "evt-1")); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(w.indexed) != 0 {
		t.Error("a cancelled event was indexed")
	}
	if len(w.deleted) != 1 || w.deleted[0] != "evt-1" {
		t.Errorf("deleted = %v, want [evt-1]", w.deleted)
	}
}

// An event catalog no longer has must leave the index, or it stays searchable
// forever.
func TestDeletedEventIsRemoved(t *testing.T) {
	w := &stubWriter{}
	idx := newIndexer(&stubCatalog{err: status.Error(codes.NotFound, "gone")}, w)

	if err := idx.Handle(context.Background(), msgFor(t, "evt-1")); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(w.deleted) != 1 {
		t.Errorf("deleted = %v, want the event removed", w.deleted)
	}
}

// Anything other than NotFound is transient -- catalog restarting, a network
// blip -- and must be retried rather than dropping the event from search.
func TestCatalogFailureIsRetryable(t *testing.T) {
	w := &stubWriter{}
	idx := newIndexer(&stubCatalog{err: status.Error(codes.Unavailable, "restarting")}, w)

	err := idx.Handle(context.Background(), msgFor(t, "evt-1"))
	if err == nil {
		t.Fatal("a catalog outage did not fail the message")
	}
	if errors.Is(err, tfkafka.ErrPermanent) {
		t.Error("a transient catalog failure was marked permanent")
	}
	if len(w.deleted) != 0 {
		t.Error("a transient failure removed the event from the index")
	}
}

func TestMalformedMessageIsPermanent(t *testing.T) {
	idx := newIndexer(&stubCatalog{}, &stubWriter{})

	if err := idx.Handle(context.Background(), segkafka.Message{Value: []byte("{bad")}); !errors.Is(err, tfkafka.ErrPermanent) {
		t.Fatalf("err = %v, want ErrPermanent", err)
	}
}

func TestEmptyEventIDIsPermanent(t *testing.T) {
	idx := newIndexer(&stubCatalog{}, &stubWriter{})

	raw, _ := tfkafka.Marshal(tfkafka.Envelope[tfkafka.CatalogEventUpdated]{SchemaVersion: 1})
	if err := idx.Handle(context.Background(), segkafka.Message{Value: raw}); !errors.Is(err, tfkafka.ErrPermanent) {
		t.Fatalf("err = %v, want ErrPermanent", err)
	}
}

// Redelivery is routine under at-least-once. Because the indexer re-reads
// current state, replaying converges rather than corrupting.
func TestRedeliveryConverges(t *testing.T) {
	w := &stubWriter{}
	idx := newIndexer(&stubCatalog{event: sampleEvent(catalogv1.EventStatus_EVENT_STATUS_ON_SALE)}, w)
	msg := msgFor(t, "evt-1")

	for range 3 {
		if err := idx.Handle(context.Background(), msg); err != nil {
			t.Fatal(err)
		}
	}
	if len(w.indexed) != 3 {
		t.Errorf("indexed %d times, want 3 -- last write wins, so each delivery writes", len(w.indexed))
	}
	for _, doc := range w.indexed {
		if doc.Title != "Coldplay" {
			t.Errorf("a replay wrote %q", doc.Title)
		}
	}
}
