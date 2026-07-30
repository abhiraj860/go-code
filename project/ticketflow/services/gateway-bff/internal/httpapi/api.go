// Package httpapi is the public REST surface.
//
// It owns no data. Its whole job is to fan out to the internal gRPC services,
// merge their answers into what a browser actually needs, and translate errors
// into HTTP status codes.
package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	catalogv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/catalog/v1"
	inventoryv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/inventory/v1"
	"github.com/abhiraj860/ticketflow/services/gateway-bff/internal/session"
)

// API holds the handler dependencies.
type API struct {
	catalog   catalogv1.CatalogServiceClient
	inventory inventoryv1.InventoryServiceClient
	sessions  *session.Store
	logger    *slog.Logger

	// upstreamTimeout bounds each fan-out call. Without it, one slow service
	// holds a BFF request open until the client gives up, and during a drop
	// that exhausts the BFF's own connection pool -- a slow dependency becomes
	// a total outage.
	upstreamTimeout time.Duration
}

type Options struct {
	Catalog         catalogv1.CatalogServiceClient
	Inventory       inventoryv1.InventoryServiceClient
	Sessions        *session.Store
	Logger          *slog.Logger
	UpstreamTimeout time.Duration
}

func New(opts Options) *API {
	if opts.UpstreamTimeout <= 0 {
		opts.UpstreamTimeout = 2 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &API{
		catalog:         opts.Catalog,
		inventory:       opts.Inventory,
		sessions:        opts.Sessions,
		logger:          opts.Logger,
		upstreamTimeout: opts.UpstreamTimeout,
	}
}

const sessionCookie = "tf_session"

// Register mounts every route.
func (a *API) Register(r *gin.Engine) {
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	v1 := r.Group("/v1")
	v1.Use(a.sessionMiddleware())

	v1.GET("/events", a.listEvents)
	v1.GET("/events/:id", a.getEvent)
	v1.GET("/events/:id/seatmap", a.getSeatMap)
	v1.POST("/holds", a.createHold)
	v1.DELETE("/holds/:id", a.releaseHold)
}

// sessionMiddleware attaches a session to every request, minting one when the
// cookie is absent. Anonymous browsing still needs an identity, because a hold
// belongs to a buyer before they have logged in.
func (a *API) sessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		if id, err := c.Cookie(sessionCookie); err == nil {
			if sess, err := a.sessions.Get(ctx, id); err == nil {
				c.Set("session", sess)
				c.Next()
				return
			}
		}

		// An empty user id makes the store derive an anonymous one. It must be
		// assigned before persisting, or a returning request would load a
		// session with no identity and be unable to hold seats.
		sess, err := a.sessions.Create(ctx, "")
		if err != nil {
			// Redis being down must not stop someone browsing. Continue without
			// a session; only the hold endpoints actually require one.
			a.logger.WarnContext(ctx, "could not create session", slog.Any("error", err))
			c.Next()
			return
		}

		c.SetCookie(sessionCookie, sess.ID, int(24*time.Hour/time.Second), "/", "", false, true)
		c.Set("session", sess)
		c.Next()
	}
}

func sessionFrom(c *gin.Context) (session.Session, bool) {
	v, ok := c.Get("session")
	if !ok {
		return session.Session{}, false
	}
	sess, ok := v.(session.Session)
	return sess, ok
}

func (a *API) listEvents(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), a.upstreamTimeout)
	defer cancel()

	pageSize, err := intQuery(c, "page_size", 20)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := a.catalog.ListEvents(ctx, &catalogv1.ListEventsRequest{
		City:      c.Query("city"),
		PageSize:  int32(pageSize),
		PageToken: c.Query("page_token"),
	})
	if err != nil {
		a.writeGRPCError(c, err)
		return
	}

	// Browse results are safe to cache briefly at the CDN: the list changes
	// only when an event is added or its status flips.
	c.Header("Cache-Control", "public, max-age=30")
	c.JSON(http.StatusOK, gin.H{
		"events":          resp.GetEvents(),
		"next_page_token": resp.GetNextPageToken(),
	})
}

// getEvent returns an event together with a live availability summary.
//
// THE FAN-OUT. Catalog and inventory are independent, so calling them
// sequentially would make the page wait for the sum of both latencies. errgroup
// runs them concurrently and returns the first error, so the response takes as
// long as the slower call rather than both.
//
// Note the asymmetry in how failures are handled: catalog failing is fatal to
// the response (there is no page without the event), but inventory failing is
// not -- the page still renders with availability omitted. Encoding that
// difference is the point of doing the fan-out by hand rather than with a
// generic helper.
func (a *API) getEvent(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), a.upstreamTimeout)
	defer cancel()

	eventID := c.Param("id")

	var (
		eventResp *catalogv1.GetEventResponse
		availResp *inventoryv1.GetAvailabilityResponse
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		eventResp, err = a.catalog.GetEvent(gctx, &catalogv1.GetEventRequest{EventId: eventID})
		return err
	})

	g.Go(func() error {
		resp, err := a.inventory.GetAvailability(gctx, &inventoryv1.GetAvailabilityRequest{
			EventId: eventID,
		})
		if err != nil {
			// Swallowed deliberately: a buyer can still read the event page
			// without a seat count. Returning the error would cancel the
			// catalog call too, via the shared errgroup context.
			a.logger.WarnContext(gctx, "availability unavailable, rendering without it",
				slog.String("event_id", eventID), slog.Any("error", err))
			return nil
		}
		availResp = resp
		return nil
	})

	if err := g.Wait(); err != nil {
		a.writeGRPCError(c, err)
		return
	}

	event := eventResp.GetEvent()

	// ETag from the event's version counter, which increments on every mutation.
	// A repeat visitor gets a 304 with no body -- the cheapest cache there is.
	etag := fmt.Sprintf(`W/"evt-%s-v%d"`, event.GetId(), event.GetVersion())
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("ETag", etag)
	// Availability is embedded, so this response must not be shared between
	// users or held for long.
	c.Header("Cache-Control", "private, max-age=5")

	c.JSON(http.StatusOK, gin.H{
		"event":        event,
		"availability": summarise(availResp),
	})
}

// getSeatMap merges static seat geometry with live availability.
//
// The two halves have opposite caching characteristics, which is exactly why
// they live in different services: the map is cached for hours in catalog,
// while availability is read through to Postgres on every request.
func (a *API) getSeatMap(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), a.upstreamTimeout)
	defer cancel()

	eventID := c.Param("id")

	eventResp, err := a.catalog.GetEvent(ctx, &catalogv1.GetEventRequest{EventId: eventID})
	if err != nil {
		a.writeGRPCError(c, err)
		return
	}

	var (
		mapResp   *catalogv1.GetSeatMapResponse
		availResp *inventoryv1.GetAvailabilityResponse
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		mapResp, err = a.catalog.GetSeatMap(gctx, &catalogv1.GetSeatMapRequest{
			SeatMapId: eventResp.GetEvent().GetSeatMapId(),
		})
		return err
	})

	g.Go(func() error {
		var err error
		availResp, err = a.inventory.GetAvailability(gctx, &inventoryv1.GetAvailabilityRequest{
			EventId: eventID,
		})
		return err
	})

	if err := g.Wait(); err != nil {
		a.writeGRPCError(c, err)
		return
	}

	// Seat state is the volatile half; never let a cache hold this response.
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"seat_map":     mapResp.GetSeatMap(),
		"availability": availResp.GetSeats(),
		"sequence":     availResp.GetSequence(),
	})
}

type createHoldRequest struct {
	EventID string   `json:"event_id" binding:"required"`
	SeatIDs []string `json:"seat_ids" binding:"required,min=1"`
	TTL     int32    `json:"ttl_seconds"`
}

func (a *API) createHold(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), a.upstreamTimeout)
	defer cancel()

	var req createHoldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sess, ok := sessionFrom(c)
	if !ok || sess.UserID == "" {
		// Holds need an owner; browsing does not.
		c.JSON(http.StatusUnauthorized, gin.H{"error": "a session is required to hold seats"})
		return
	}

	// The client supplies the idempotency key, because only the client knows
	// which retries are the same logical attempt. Generating one here would
	// make every retry a fresh request and defeat the purpose.
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required"})
		return
	}

	resp, err := a.inventory.HoldSeats(ctx, &inventoryv1.HoldSeatsRequest{
		EventId:        req.EventID,
		SeatIds:        req.SeatIDs,
		UserId:         sess.UserID,
		IdempotencyKey: idempotencyKey,
		TtlSeconds:     req.TTL,
	})
	if err != nil {
		a.writeGRPCError(c, err)
		return
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, gin.H{
		"hold_id":           resp.GetHoldId(),
		"held_seat_ids":     resp.GetHeldSeatIds(),
		"rejected_seat_ids": resp.GetRejectedSeatIds(),
		"expires_at":        resp.GetExpiresAt().AsTime(),
	})
}

func (a *API) releaseHold(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), a.upstreamTimeout)
	defer cancel()

	resp, err := a.inventory.ReleaseHold(ctx, &inventoryv1.ReleaseHoldRequest{
		HoldId: c.Param("id"),
	})
	if err != nil {
		a.writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"released_seat_count": resp.GetReleasedSeatCount()})
}

// availabilitySummary is a count per status, which is all a listing page needs.
// Sending twenty thousand seat records to render "1,204 seats left" would be
// absurd; the full map is a separate, explicit request.
type availabilitySummary struct {
	Available int `json:"available"`
	Held      int `json:"held"`
	Sold      int `json:"sold"`
	Blocked   int `json:"blocked"`
	Total     int `json:"total"`
}

func summarise(resp *inventoryv1.GetAvailabilityResponse) *availabilitySummary {
	if resp == nil {
		// Distinguishable from a zero summary: null means "inventory did not
		// answer", not "no seats exist".
		return nil
	}
	var s availabilitySummary
	for _, seat := range resp.GetSeats() {
		s.Total++
		switch seat.GetStatus() {
		case inventoryv1.SeatStatus_SEAT_STATUS_AVAILABLE:
			s.Available++
		case inventoryv1.SeatStatus_SEAT_STATUS_HELD:
			s.Held++
		case inventoryv1.SeatStatus_SEAT_STATUS_SOLD:
			s.Sold++
		case inventoryv1.SeatStatus_SEAT_STATUS_BLOCKED:
			s.Blocked++
		}
	}
	return &s
}

// writeGRPCError maps upstream gRPC codes to HTTP status codes.
//
// The mapping is deliberate rather than mechanical: ResourceExhausted from
// inventory means "those seats are gone", which is 409 Conflict to a browser,
// not 429 Too Many Requests.
func (a *API) writeGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		a.logger.ErrorContext(c.Request.Context(), "non-grpc upstream error", slog.Any("error", err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "upstream unavailable"})
		return
	}

	switch st.Code() {
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
	case codes.ResourceExhausted:
		c.JSON(http.StatusConflict, gin.H{"error": "those seats are no longer available"})
	case codes.FailedPrecondition:
		c.JSON(http.StatusConflict, gin.H{"error": st.Message()})
	case codes.DeadlineExceeded, codes.Canceled:
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "upstream timed out"})
	case codes.Unavailable:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "upstream unavailable"})
	default:
		a.logger.ErrorContext(c.Request.Context(), "upstream error",
			slog.String("code", st.Code().String()), slog.String("message", st.Message()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func intQuery(c *gin.Context, name string, def int) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New(name + " must be an integer")
	}
	return v, nil
}
