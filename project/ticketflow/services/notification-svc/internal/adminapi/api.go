// Package adminapi is notification-svc's operator interface, served with Echo.
//
// Echo rather than Gin here, deliberately. The public BFF uses Gin because it
// is the highest-traffic surface and its middleware chain is already built
// around it; this is a small internal admin API where Echo's binding and
// validation ergonomics are pleasant and the traffic is negligible. Using one
// framework everywhere would be defensible too -- but a service boundary is
// exactly where that choice can differ without cost.
package adminapi

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/abhiraj860/ticketflow/services/notification-svc/internal/notifier"
	"github.com/abhiraj860/ticketflow/services/notification-svc/internal/sqsconsumer"
)

// API exposes operator endpoints.
type API struct {
	publisher *notifier.Publisher
	consumer  *sqsconsumer.Consumer
}

func New(publisher *notifier.Publisher, consumer *sqsconsumer.Consumer) *API {
	return &API{publisher: publisher, consumer: consumer}
}

// Register mounts the routes on an Echo instance.
func (a *API) Register(e *echo.Echo) {
	e.Use(middleware.Recover())
	// Structured request logging, matching the JSON the Go services emit so
	// CloudWatch Logs Insights can query both with one expression.
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus: true, LogURI: true, LogLatency: true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			return nil
		},
	}))

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.GET("/metrics", a.metrics)
	e.POST("/admin/test-notification", a.sendTest)
}

func (a *API) metrics(c echo.Context) error {
	pub := a.publisher.Stats()
	con := a.consumer.Stats()

	return c.String(http.StatusOK,
		"notification_sent_total "+itoa(pub.Sent)+"\n"+
			"notification_failed_total "+itoa(pub.Failed)+"\n"+
			"notification_sqs_received_total "+itoa(con.Received)+"\n"+
			"notification_sqs_processed_total "+itoa(con.Processed)+"\n"+
			// A rising failed count with a flat processed count means messages
			// are cycling toward the DLQ.
			"notification_sqs_failed_total "+itoa(con.Failed)+"\n")
}

type testRequest struct {
	UserID  string `json:"user_id" validate:"required"`
	Channel string `json:"channel"`
}

// sendTest publishes a notification without going through the queue, so an
// operator can verify SNS subscriptions are wired without placing a real order.
func (a *API) sendTest(c echo.Context) error {
	var req testRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "malformed request"})
	}
	if req.UserID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id is required"})
	}
	if req.Channel == "" {
		req.Channel = "email"
	}

	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

	err := a.publisher.Send(ctx, notifier.Notification{
		TicketID: "test", OrderID: "test", EventID: "test",
		SeatID: "test", UserID: req.UserID, Channel: req.Channel,
	})
	if err != nil {
		return c.JSON(http.StatusBadGateway, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusAccepted, echo.Map{"status": "sent", "channel": req.Channel})
}

func itoa(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
