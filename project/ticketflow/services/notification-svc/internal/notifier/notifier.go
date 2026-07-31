// Package notifier delivers ticket notifications.
//
// The pipeline is Kafka -> SQS -> SNS, and each hop earns its place:
//
//   - Kafka carries ticket.issued, because that is the event backbone every
//     service already speaks
//   - SQS decouples delivery from issuance. Email and SMS providers are slow
//     and rate-limited; without a queue in front, a provider outage would
//     back-pressure into ticket generation and eventually into checkout
//   - SNS fans one notification out to email, SMS and push subscribers without
//     this service knowing who they are. Adding a channel becomes a
//     subscription change rather than a code change.
package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// Notification is what a subscriber receives.
type Notification struct {
	TicketID string `json:"ticket_id"`
	OrderID  string `json:"order_id"`
	EventID  string `json:"event_id"`
	SeatID   string `json:"seat_id"`
	UserID   string `json:"user_id"`
	PDFKey   string `json:"pdf_key"`
	// Channel lets a subscriber filter without parsing the body, via an SNS
	// message attribute.
	Channel string `json:"channel"`
}

// Publisher fans notifications out over SNS.
type Publisher struct {
	client   *sns.Client
	topicARN string
	logger   *slog.Logger

	sent, failed atomic.Uint64
}

func New(client *sns.Client, topicARN string, logger *slog.Logger) *Publisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Publisher{client: client, topicARN: topicARN, logger: logger}
}

// Send publishes one notification.
//
// The message attribute is what makes fan-out selective: an SNS subscription
// filter policy on `channel` lets the SMS subscriber ignore email traffic
// without this service maintaining a list of who wants what.
func (p *Publisher) Send(ctx context.Context, n Notification) error {
	body, err := json.Marshal(n)
	if err != nil {
		p.failed.Add(1)
		return fmt.Errorf("notifier: marshalling notification: %w", err)
	}

	_, err = p.client.Publish(ctx, &sns.PublishInput{
		TopicArn: aws.String(p.topicARN),
		Message:  aws.String(string(body)),
		Subject:  aws.String("Your TicketFlow ticket"),
		MessageAttributes: map[string]snstypes.MessageAttributeValue{
			"channel":  {DataType: aws.String("String"), StringValue: aws.String(n.Channel)},
			"event_id": {DataType: aws.String("String"), StringValue: aws.String(n.EventID)},
		},
	})
	if err != nil {
		p.failed.Add(1)
		return fmt.Errorf("notifier: publishing to sns: %w", err)
	}

	p.sent.Add(1)
	p.logger.InfoContext(ctx, "notification sent",
		slog.String("ticket_id", n.TicketID), slog.String("channel", n.Channel))
	return nil
}

// Stats reports delivery counts.
type Stats struct {
	Sent   uint64
	Failed uint64
}

func (p *Publisher) Stats() Stats {
	return Stats{Sent: p.sent.Load(), Failed: p.failed.Load()}
}
