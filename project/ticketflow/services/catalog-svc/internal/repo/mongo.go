package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/abhiraj860/ticketflow/services/catalog-svc/internal/domain"
)

// ContentRepo reads editorial event content from MongoDB.
//
// Why Mongo and not Postgres, given catalog already owns a relational database:
// this content genuinely has no fixed shape. A concert document carries a
// setlist and support acts; a football fixture carries squads and a league
// table; a theatre run carries a cast and an interval length. In Postgres that
// is either a wide sparse table, an EAV schema, or a jsonb column that gets
// none of Mongo's indexing over nested fields.
//
// The split is drawn at a clear line: anything queried, sorted, filtered or
// joined lives in Postgres; anything only ever fetched whole, by event id,
// lives here.
type ContentRepo struct {
	coll *mongo.Collection
}

const (
	contentDB         = "ticketflow"
	contentCollection = "event_content"
)

func NewContentRepo(client *mongo.Client) *ContentRepo {
	return &ContentRepo{coll: client.Database(contentDB).Collection(contentCollection)}
}

// contentDoc is the stored shape. Only _id and updated_at are fixed; the body
// is deliberately untyped.
type contentDoc struct {
	EventID   string    `bson:"_id"`
	Kind      int16     `bson:"kind"`
	Body      bson.M    `bson:"body"`
	UpdatedAt time.Time `bson:"updated_at"`
}

// GetContent returns an event's content document.
//
// Keyed by event id as _id, so every read is a primary-key lookup and the
// collection needs no secondary index at all.
func (r *ContentRepo) GetContent(ctx context.Context, eventID string) (domain.EventContent, error) {
	var doc contentDoc

	err := r.coll.FindOne(ctx, bson.M{"_id": eventID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return domain.EventContent{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.EventContent{}, fmt.Errorf("catalog: fetching content for %q: %w", eventID, err)
	}

	return domain.EventContent{
		EventID:   doc.EventID,
		Kind:      domain.EventKind(doc.Kind),
		Body:      bsonToMap(doc.Body),
		UpdatedAt: doc.UpdatedAt,
	}, nil
}

// UpsertContent writes an event's content. Upsert rather than insert so a
// re-run of a content import is idempotent.
func (r *ContentRepo) UpsertContent(ctx context.Context, c domain.EventContent) error {
	doc := bson.M{
		"kind":       int16(c.Kind),
		"body":       c.Body,
		"updated_at": time.Now().UTC(),
	}

	_, err := r.coll.UpdateOne(ctx,
		bson.M{"_id": c.EventID},
		bson.M{"$set": doc},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("catalog: upserting content for %q: %w", c.EventID, err)
	}
	return nil
}

// bsonToMap converts a decoded BSON document into plain Go types.
//
// Necessary because bson.M values nest as bson.M and bson.A rather than
// map[string]any and []any, and structpb -- which the gRPC layer needs -- only
// understands the plain forms.
func bsonToMap(m bson.M) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = bsonValue(v)
	}
	return out
}

func bsonValue(v any) any {
	switch t := v.(type) {
	case bson.M:
		return bsonToMap(t)
	case bson.D:
		// Ordered documents decode as bson.D; flatten to a map since JSON has
		// no ordering guarantee anyway.
		out := make(map[string]any, len(t))
		for _, e := range t {
			out[e.Key] = bsonValue(e.Value)
		}
		return out
	case bson.A:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = bsonValue(e)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = bsonValue(e)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, e := range t {
			out[k] = bsonValue(e)
		}
		return out
	case bson.DateTime:
		return t.Time().UTC().Format(time.RFC3339)
	default:
		return v
	}
}

// ConnectMongo dials MongoDB and verifies reachability, so a bad URI fails at
// startup rather than on the first content read.
func ConnectMongo(ctx context.Context, uri string, timeout time.Duration) (*mongo.Client, error) {
	if uri == "" {
		return nil, errors.New("catalog: mongo URI is required")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri).SetConnectTimeout(timeout))
	if err != nil {
		return nil, fmt.Errorf("catalog: connecting to mongo: %w", err)
	}

	if err := client.Ping(dialCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("catalog: pinging mongo: %w", err)
	}
	return client, nil
}
