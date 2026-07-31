package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/abhiraj860/ticketflow/pkg/testsupport"
	catalogv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/catalog/v1"
	inventoryv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/inventory/v1"
	"github.com/abhiraj860/ticketflow/services/gateway-bff/internal/session"
)

// stubCatalog implements the generated client interface so handlers can be
// exercised without running a gRPC server.
type stubCatalog struct {
	event      *catalogv1.Event
	eventErr   error
	eventDelay time.Duration

	content    *structpb.Struct
	contentErr error

	seatMap    *catalogv1.SeatMap
	seatMapErr error

	list    *catalogv1.ListEventsResponse
	listErr error
}

func (s *stubCatalog) GetEvent(ctx context.Context, _ *catalogv1.GetEventRequest, _ ...grpc.CallOption) (*catalogv1.GetEventResponse, error) {
	if s.eventDelay > 0 {
		select {
		case <-time.After(s.eventDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if s.eventErr != nil {
		return nil, s.eventErr
	}
	return &catalogv1.GetEventResponse{Event: s.event}, nil
}

func (s *stubCatalog) ListEvents(context.Context, *catalogv1.ListEventsRequest, ...grpc.CallOption) (*catalogv1.ListEventsResponse, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.list != nil {
		return s.list, nil
	}
	return &catalogv1.ListEventsResponse{}, nil
}

func (s *stubCatalog) GetSeatMap(context.Context, *catalogv1.GetSeatMapRequest, ...grpc.CallOption) (*catalogv1.GetSeatMapResponse, error) {
	if s.seatMapErr != nil {
		return nil, s.seatMapErr
	}
	return &catalogv1.GetSeatMapResponse{SeatMap: s.seatMap}, nil
}

func (s *stubCatalog) GetEventContent(context.Context, *catalogv1.GetEventContentRequest, ...grpc.CallOption) (*catalogv1.GetEventContentResponse, error) {
	if s.contentErr != nil {
		return nil, s.contentErr
	}
	return &catalogv1.GetEventContentResponse{Content: s.content}, nil
}

type stubInventory struct {
	avail    *inventoryv1.GetAvailabilityResponse
	availErr error

	hold    *inventoryv1.HoldSeatsResponse
	holdErr error

	release    *inventoryv1.ReleaseHoldResponse
	releaseErr error
}

func (s *stubInventory) GetAvailability(context.Context, *inventoryv1.GetAvailabilityRequest, ...grpc.CallOption) (*inventoryv1.GetAvailabilityResponse, error) {
	if s.availErr != nil {
		return nil, s.availErr
	}
	return s.avail, nil
}

func (s *stubInventory) HoldSeats(context.Context, *inventoryv1.HoldSeatsRequest, ...grpc.CallOption) (*inventoryv1.HoldSeatsResponse, error) {
	if s.holdErr != nil {
		return nil, s.holdErr
	}
	return s.hold, nil
}

func (s *stubInventory) ReleaseHold(context.Context, *inventoryv1.ReleaseHoldRequest, ...grpc.CallOption) (*inventoryv1.ReleaseHoldResponse, error) {
	if s.releaseErr != nil {
		return nil, s.releaseErr
	}
	return s.release, nil
}

func (s *stubInventory) ConfirmHold(context.Context, *inventoryv1.ConfirmHoldRequest, ...grpc.CallOption) (*inventoryv1.ConfirmHoldResponse, error) {
	return &inventoryv1.ConfirmHoldResponse{}, nil
}

// newTestServer wires the handlers to stubs and a real Redis session store
// (DB 13). Sessions are exercised for real because the last session bug lived
// precisely in the persistence round trip, which a fake would have hidden.
func newTestServer(t *testing.T, cat catalogv1.CatalogServiceClient, inv inventoryv1.InventoryServiceClient) *gin.Engine {
	t.Helper()

	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", DB: 13, DialTimeout: 500 * time.Millisecond,
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		testsupport.SkipOrFail(t, "redis not reachable (run `make up`): %v", err)
	}
	_ = client.FlushDB(ctx).Err()
	t.Cleanup(func() { _ = client.Close() })

	api := New(Options{
		Catalog:         cat,
		Inventory:       inv,
		Sessions:        session.NewStore(session.Options{Client: client, TTL: time.Minute}),
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		UpstreamTimeout: time.Second,
	})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	api.Register(r)
	return r
}

func sampleEvent() *catalogv1.Event {
	return &catalogv1.Event{
		Id: "e1", Title: "Concert", Version: 7,
		Venue: &catalogv1.Venue{Id: "v1", City: "Mumbai"},
	}
}

func do(t *testing.T, r *gin.Engine, method, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("response was not JSON: %v\nbody: %s", err, w.Body.String())
	}
	return out
}

// The asymmetry that justifies hand-writing the fan-out: inventory failing must
// NOT kill the page, because a buyer can still read an event without a seat
// count.
func TestEventPageRendersWhenInventoryFails(t *testing.T) {
	r := newTestServer(t,
		&stubCatalog{event: sampleEvent()},
		&stubInventory{availErr: status.Error(codes.Unavailable, "inventory down")},
	)

	w := do(t, r, http.MethodGet, "/v1/events/e1", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 -- inventory failing must not fail the page", w.Code)
	}

	body := decode(t, w)
	if body["event"] == nil {
		t.Error("event missing from the response")
	}
	// Null rather than a zero summary: "inventory did not answer" is not the
	// same claim as "this event has no seats".
	if body["availability"] != nil {
		t.Errorf("availability = %v, want null when inventory failed", body["availability"])
	}
}

// Catalog failing IS fatal: without the event there is no page.
func TestEventPageFailsWhenCatalogFails(t *testing.T) {
	r := newTestServer(t,
		&stubCatalog{eventErr: status.Error(codes.NotFound, "no such event")},
		&stubInventory{avail: &inventoryv1.GetAvailabilityResponse{}},
	)

	w := do(t, r, http.MethodGet, "/v1/events/missing", "", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// Most events have no content document, so a NotFound from content must be
// invisible rather than an error.
func TestEventPageRendersWithoutContent(t *testing.T) {
	r := newTestServer(t,
		&stubCatalog{event: sampleEvent(), contentErr: status.Error(codes.NotFound, "no content")},
		&stubInventory{avail: &inventoryv1.GetAvailabilityResponse{}},
	)

	w := do(t, r, http.MethodGet, "/v1/events/e1", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if body := decode(t, w); body["content"] != nil {
		t.Errorf("content = %v, want null", body["content"])
	}
}

func TestAvailabilitySummary(t *testing.T) {
	avail := &inventoryv1.GetAvailabilityResponse{Seats: []*inventoryv1.SeatAvailability{
		{SeatId: "1", Status: inventoryv1.SeatStatus_SEAT_STATUS_AVAILABLE},
		{SeatId: "2", Status: inventoryv1.SeatStatus_SEAT_STATUS_AVAILABLE},
		{SeatId: "3", Status: inventoryv1.SeatStatus_SEAT_STATUS_HELD},
		{SeatId: "4", Status: inventoryv1.SeatStatus_SEAT_STATUS_SOLD},
		{SeatId: "5", Status: inventoryv1.SeatStatus_SEAT_STATUS_BLOCKED},
	}}

	r := newTestServer(t, &stubCatalog{event: sampleEvent()}, &stubInventory{avail: avail})

	w := do(t, r, http.MethodGet, "/v1/events/e1", "", nil)
	summary, ok := decode(t, w)["availability"].(map[string]any)
	if !ok {
		t.Fatalf("availability missing: %s", w.Body.String())
	}

	for field, want := range map[string]float64{
		"available": 2, "held": 1, "sold": 1, "blocked": 1, "total": 5,
	} {
		if got := summary[field]; got != want {
			t.Errorf("%s = %v, want %v", field, got, want)
		}
	}
}

// The cheapest cache is the one that sends no body at all.
func TestETagReturns304(t *testing.T) {
	r := newTestServer(t, &stubCatalog{event: sampleEvent()},
		&stubInventory{avail: &inventoryv1.GetAvailabilityResponse{}})

	first := do(t, r, http.MethodGet, "/v1/events/e1", "", nil)
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatal("no ETag on the first response")
	}

	second := do(t, r, http.MethodGet, "/v1/events/e1", "", map[string]string{"If-None-Match": etag})
	if second.Code != http.StatusNotModified {
		t.Errorf("status = %d, want 304", second.Code)
	}
	if second.Body.Len() != 0 {
		t.Errorf("304 carried a %d-byte body, want none", second.Body.Len())
	}
}

// The ETag is derived from the version counter, so an edit must invalidate it.
func TestETagChangesWithEventVersion(t *testing.T) {
	cat := &stubCatalog{event: sampleEvent()}
	r := newTestServer(t, cat, &stubInventory{avail: &inventoryv1.GetAvailabilityResponse{}})

	oldETag := do(t, r, http.MethodGet, "/v1/events/e1", "", nil).Header().Get("ETag")

	cat.event = &catalogv1.Event{Id: "e1", Title: "Concert", Version: 8,
		Venue: &catalogv1.Venue{Id: "v1"}}

	w := do(t, r, http.MethodGet, "/v1/events/e1", "", map[string]string{"If-None-Match": oldETag})
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 -- a stale ETag must not match after an edit", w.Code)
	}
}

// Seat state is the volatile half of the seat-map response and must never be
// cached by anything in between.
func TestSeatMapIsNoStore(t *testing.T) {
	r := newTestServer(t,
		&stubCatalog{event: sampleEvent(), seatMap: &catalogv1.SeatMap{Id: "m1"}},
		&stubInventory{avail: &inventoryv1.GetAvailabilityResponse{Sequence: 42}},
	)

	w := do(t, r, http.MethodGet, "/v1/events/e1/seatmap", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
}

// Without an idempotency key a network retry silently buys a second set of
// seats, so the header is mandatory rather than optional.
func TestHoldRequiresIdempotencyKey(t *testing.T) {
	r := newTestServer(t, &stubCatalog{}, &stubInventory{})

	w := do(t, r, http.MethodPost, "/v1/holds",
		`{"event_id":"e1","seat_ids":["S-1"]}`, nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Idempotency-Key") {
		t.Errorf("error does not mention the missing header: %s", w.Body.String())
	}
}

func TestHoldSucceeds(t *testing.T) {
	r := newTestServer(t, &stubCatalog{}, &stubInventory{
		hold: &inventoryv1.HoldSeatsResponse{
			HoldId: "h1", HeldSeatIds: []string{"S-1"}, RejectedSeatIds: []string{"S-2"},
		},
	})

	w := do(t, r, http.MethodPost, "/v1/holds",
		`{"event_id":"e1","seat_ids":["S-1","S-2"]}`,
		map[string]string{"Idempotency-Key": "k1"})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", w.Code, w.Body.String())
	}
	body := decode(t, w)
	if body["hold_id"] != "h1" {
		t.Errorf("hold_id = %v", body["hold_id"])
	}
	// A partial result must be surfaced, not hidden.
	rejected, _ := body["rejected_seat_ids"].([]any)
	if len(rejected) != 1 || rejected[0] != "S-2" {
		t.Errorf("rejected_seat_ids = %v, want [S-2]", body["rejected_seat_ids"])
	}
}

func TestHoldRejectsMalformedBody(t *testing.T) {
	r := newTestServer(t, &stubCatalog{}, &stubInventory{})

	for _, body := range []string{`{}`, `{"event_id":"e1"}`, `{"event_id":"e1","seat_ids":[]}`, `not json`} {
		w := do(t, r, http.MethodPost, "/v1/holds", body,
			map[string]string{"Idempotency-Key": "k"})
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %q -> status %d, want 400", body, w.Code)
		}
	}
}

// The mapping is deliberate, not mechanical. ResourceExhausted from inventory
// means "those seats are gone", which is 409 to a browser -- not 429.
func TestGRPCErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		code codes.Code
		want int
	}{
		{"not found", codes.NotFound, http.StatusNotFound},
		{"invalid argument", codes.InvalidArgument, http.StatusBadRequest},
		{"seats gone", codes.ResourceExhausted, http.StatusConflict},
		{"hold expired", codes.FailedPrecondition, http.StatusConflict},
		{"upstream timeout", codes.DeadlineExceeded, http.StatusGatewayTimeout},
		{"upstream down", codes.Unavailable, http.StatusServiceUnavailable},
		{"anything else", codes.Internal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestServer(t, &stubCatalog{}, &stubInventory{
				holdErr: status.Error(tt.code, "upstream said no"),
			})

			w := do(t, r, http.MethodPost, "/v1/holds",
				`{"event_id":"e1","seat_ids":["S-1"]}`,
				map[string]string{"Idempotency-Key": "k1"})

			if w.Code != tt.want {
				t.Errorf("%v -> HTTP %d, want %d", tt.code, w.Code, tt.want)
			}
		})
	}
}

// An internal error must not leak the upstream message, which can disclose
// schema details.
func TestInternalErrorDoesNotLeakUpstreamMessage(t *testing.T) {
	r := newTestServer(t, &stubCatalog{}, &stubInventory{
		holdErr: status.Error(codes.Internal, `pq: relation "seat_allocation" does not exist`),
	})

	w := do(t, r, http.MethodPost, "/v1/holds",
		`{"event_id":"e1","seat_ids":["S-1"]}`,
		map[string]string{"Idempotency-Key": "k1"})

	if strings.Contains(w.Body.String(), "seat_allocation") {
		t.Errorf("response leaked the upstream error: %s", w.Body.String())
	}
}

// A slow upstream must surface as a 504 from our own handler rather than
// hanging until the client gives up.
func TestSlowUpstreamTimesOut(t *testing.T) {
	r := newTestServer(t,
		&stubCatalog{event: sampleEvent(), eventDelay: 2 * time.Second},
		&stubInventory{avail: &inventoryv1.GetAvailabilityResponse{}},
	)

	start := time.Now()
	w := do(t, r, http.MethodGet, "/v1/events/e1", "", nil)
	elapsed := time.Since(start)

	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("status = %d, want 504", w.Code)
	}
	if elapsed > 1500*time.Millisecond {
		t.Errorf("took %v; the 1s upstream timeout was not enforced", elapsed)
	}
}

// Browsing issues a session cookie, because a hold needs an owner before the
// buyer has logged in.
func TestSessionCookieIsIssued(t *testing.T) {
	r := newTestServer(t, &stubCatalog{}, &stubInventory{})

	w := do(t, r, http.MethodGet, "/v1/events", "", nil)

	var found bool
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			found = true
			if !c.HttpOnly {
				t.Error("session cookie is not HttpOnly -- it is readable by scripts")
			}
		}
	}
	if !found {
		t.Error("no session cookie was issued")
	}
}

// The regression this guards: a returning session must still be able to hold
// seats. The earlier bug persisted sessions with an empty UserID, so only the
// very first request worked.
func TestReturningSessionCanStillHold(t *testing.T) {
	r := newTestServer(t, &stubCatalog{}, &stubInventory{
		hold: &inventoryv1.HoldSeatsResponse{HoldId: "h1", HeldSeatIds: []string{"S-1"}},
	})

	first := do(t, r, http.MethodGet, "/v1/events", "", nil)
	var cookie string
	for _, c := range first.Result().Cookies() {
		if c.Name == sessionCookie {
			cookie = c.Value
		}
	}
	if cookie == "" {
		t.Fatal("no session cookie issued")
	}

	w := do(t, r, http.MethodPost, "/v1/holds",
		`{"event_id":"e1","seat_ids":["S-1"]}`,
		map[string]string{
			"Idempotency-Key": "k1",
			"Cookie":          sessionCookie + "=" + cookie,
		})

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 -- a returning session must be able to hold: %s",
			w.Code, w.Body.String())
	}
}

func TestListEventsRejectsBadPageSize(t *testing.T) {
	r := newTestServer(t, &stubCatalog{}, &stubInventory{})

	w := do(t, r, http.MethodGet, "/v1/events?page_size=lots", "", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHealthz(t *testing.T) {
	r := newTestServer(t, &stubCatalog{}, &stubInventory{})

	if w := do(t, r, http.MethodGet, "/healthz", "", nil); w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}
