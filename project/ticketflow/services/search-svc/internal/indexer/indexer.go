// Package indexer keeps the search index in step with catalog.
package indexer

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"

	segkafka "github.com/segmentio/kafka-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	catalogv1 "github.com/abhiraj860/ticketflow/proto/gen/ticketflow/catalog/v1"
	"github.com/abhiraj860/ticketflow/services/search-svc/internal/index"
)

// Writer is the index side, kept an interface so handler logic can be tested
// without ElasticSearch running.
type Writer interface {
	IndexDocument(ctx context.Context, doc index.Document) error
	DeleteDocument(ctx context.Context, id string) error
}

// Indexer consumes catalog.event.updated and refreshes the index.
type Indexer struct {
	catalog catalogv1.CatalogServiceClient
	writer  Writer
	logger  *slog.Logger

	indexed, deleted, failed atomic.Uint64
}

func New(catalog catalogv1.CatalogServiceClient, writer Writer, logger *slog.Logger) *Indexer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Indexer{catalog: catalog, writer: writer, logger: logger}
}

// Handle reindexes one event.
//
// The message carries only an id and a version, never the event body. That is
// deliberate: a body embedded in a message goes stale the moment it waits in a
// queue, so after a retry storm the index could be rewritten from an old
// payload -- and the index would silently disagree with catalog with nothing
// to indicate it. Reading back means the index reflects catalog's state at
// processing time, whenever that happens to be.
func (i *Indexer) Handle(ctx context.Context, msg segkafka.Message) error {
	env, err := tfkafka.Unmarshal[tfkafka.CatalogEventUpdated](msg.Value)
	if err != nil {
		return tfkafka.Permanent(err)
	}
	if env.Payload.EventID == "" {
		return tfkafka.Permanent(fmt.Errorf("indexer: message carries no event id"))
	}

	resp, err := i.catalog.GetEvent(ctx, &catalogv1.GetEventRequest{EventId: env.Payload.EventID})
	if err != nil {
		// An event catalog no longer has must leave the index, or a deleted
		// event stays searchable forever.
		if status.Code(err) == codes.NotFound {
			if delErr := i.writer.DeleteDocument(ctx, env.Payload.EventID); delErr != nil {
				return delErr
			}
			i.deleted.Add(1)
			return nil
		}
		// Anything else is transient -- catalog restarting, a network blip --
		// so let the retry handle it rather than dropping the event from search.
		i.failed.Add(1)
		return fmt.Errorf("indexer: reading event %q: %w", env.Payload.EventID, err)
	}

	doc := toDocument(resp.GetEvent())

	// Cancelled events are removed rather than indexed, so they cannot surface
	// in a browse listing at all.
	if doc.Status == statusCancelled {
		if err := i.writer.DeleteDocument(ctx, doc.ID); err != nil {
			return err
		}
		i.deleted.Add(1)
		return nil
	}

	if err := i.writer.IndexDocument(ctx, doc); err != nil {
		i.failed.Add(1)
		return err
	}

	i.indexed.Add(1)
	i.logger.Debug("event indexed",
		slog.String("event_id", doc.ID), slog.Int64("version", doc.Version))
	return nil
}

// statusCancelled mirrors catalogv1.EVENT_STATUS_CANCELLED.
const statusCancelled int16 = 4

func toDocument(e *catalogv1.Event) index.Document {
	doc := index.Document{
		ID:          e.GetId(),
		Title:       e.GetTitle(),
		Kind:        int16(e.GetKind()),
		Status:      int16(e.GetStatus()),
		VenueID:     e.GetVenue().GetId(),
		VenueName:   e.GetVenue().GetName(),
		City:        e.GetVenue().GetCity(),
		CountryCode: e.GetVenue().GetCountryCode(),
		StartsAt:    e.GetStartsAt().AsTime(),
		SaleOpensAt: e.GetSaleOpensAt().AsTime(),
		Tags:        e.GetTags(),
		PosterURL:   e.GetPosterUrl(),
		Version:     e.GetVersion(),
	}

	// Denormalise the cheapest tier, so a listing renders "from Rs X" without a
	// second call into catalog. Search engines cannot join, so the duplication
	// is the price of a one-round-trip query.
	for _, tier := range e.GetPricingTiers() {
		amount := tier.GetPrice().GetAmountMinor()
		if doc.MinPriceMinor == 0 || amount < doc.MinPriceMinor {
			doc.MinPriceMinor = amount
			doc.CurrencyCode = tier.GetPrice().GetCurrencyCode()
		}
	}

	// Normalise tags so faceting buckets "Live-Music" and "live-music"
	// together rather than showing the user two of what is one filter.
	for j, tag := range doc.Tags {
		doc.Tags[j] = strings.ToLower(strings.TrimSpace(tag))
	}
	return doc
}

// Stats reports indexing activity.
type Stats struct {
	Indexed uint64
	Deleted uint64
	Failed  uint64
}

func (i *Indexer) Stats() Stats {
	return Stats{
		Indexed: i.indexed.Load(),
		Deleted: i.deleted.Load(),
		Failed:  i.failed.Load(),
	}
}
