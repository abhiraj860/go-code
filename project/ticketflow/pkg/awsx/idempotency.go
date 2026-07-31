// Package awsx holds the AWS-facing helpers shared by services and Lambdas.
package awsx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var (
	// ErrInFlight means another request with this key is still running. The
	// caller should tell the client to retry shortly rather than proceeding,
	// because proceeding is exactly how two tickets get issued.
	ErrInFlight = errors.New("idempotency: a request with this key is already in flight")

	// ErrKeyMismatch means the key was reused with a different request body.
	// That is a client bug, and serving the first response for a different
	// request would be worse than refusing.
	ErrKeyMismatch = errors.New("idempotency: key reused with a different request")
)

// Record is a stored request outcome.
type Record struct {
	Key         string
	Status      string
	RequestHash string
	Response    []byte
	StatusCode  int32
	CreatedAt   time.Time
}

// Status values.
const (
	StatusInFlight = "in_flight"
	StatusComplete = "complete"
)

// Store is the DynamoDB-backed idempotency table.
//
// WHY DYNAMODB AND NOT POSTGRES.
//
// Inventory and order already enforce idempotency with unique constraints, and
// that remains the authoritative layer. This is a second, earlier one that runs
// at the API edge inside a Lambda -- and a Lambda cannot hold a Postgres
// connection pool. Under a drop, thousands of concurrent invocations would each
// open a connection and exhaust max_connections in seconds. DynamoDB is
// connectionless and scales horizontally, which is exactly the shape of the
// problem at the edge.
//
// It also catches retries earlier: a duplicate rejected here never reaches the
// BFF, the gRPC services or the database at all.
type Store struct {
	client *dynamodb.Client
	table  string
	ttl    time.Duration
}

type Options struct {
	Client *dynamodb.Client
	Table  string
	// TTL bounds how long a completed response is replayable. Long enough to
	// cover any client retry window, short enough that the table does not grow
	// forever -- DynamoDB expires the items itself.
	TTL time.Duration
}

func NewStore(opts Options) (*Store, error) {
	if opts.Client == nil {
		return nil, errors.New("idempotency: client is required")
	}
	if opts.Table == "" {
		return nil, errors.New("idempotency: table is required")
	}
	if opts.TTL <= 0 {
		opts.TTL = 24 * time.Hour
	}
	return &Store{client: opts.Client, table: opts.Table, ttl: opts.TTL}, nil
}

// Begin claims a key for a new request.
//
// THE CORE OPERATION, and it is a conditional write for the same reason
// inventory's seat claim is: a read-then-write from application code has a
// window in which two concurrent requests both see "absent" and both proceed.
//
//	ConditionExpression: attribute_not_exists(pk) OR (#s = complete AND ...)
//
// DynamoDB evaluates that atomically on a single item, so exactly one caller
// wins the claim no matter how many arrive together.
//
// Returns:
//   - (nil, nil)          the caller won and should do the work
//   - (record, nil)       a completed response exists; replay it verbatim
//   - (nil, ErrInFlight)  someone else is doing the work right now
func (s *Store) Begin(ctx context.Context, key, requestHash string) (*Record, error) {
	if key == "" {
		return nil, errors.New("idempotency: key is required")
	}

	now := time.Now()
	item := map[string]types.AttributeValue{
		"pk":           &types.AttributeValueMemberS{Value: key},
		"status":       &types.AttributeValueMemberS{Value: StatusInFlight},
		"request_hash": &types.AttributeValueMemberS{Value: requestHash},
		"created_at":   &types.AttributeValueMemberN{Value: fmt.Sprint(now.Unix())},
		// DynamoDB's own TTL reaps these. Deleting them from application code
		// would mean a scan, which on a hot table is both slow and expensive.
		"expires_at": &types.AttributeValueMemberN{Value: fmt.Sprint(now.Add(s.ttl).Unix())},
	}

	_, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.table),
		Item:      item,
		// Claim the key only if nothing holds it.
		ConditionExpression: aws.String("attribute_not_exists(pk)"),
	})
	if err == nil {
		return nil, nil // won the claim
	}

	var conditionFailed *types.ConditionalCheckFailedException
	if !errors.As(err, &conditionFailed) {
		return nil, fmt.Errorf("idempotency: claiming key: %w", err)
	}

	// Someone holds it. Read what they left.
	existing, err := s.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		// Claimed and expired between our write and our read. Treat as in
		// flight: the safe direction is to make the client retry rather than
		// risk doing the work twice.
		return nil, ErrInFlight
	}

	// A key reused with a different body is a client bug. Replaying the first
	// response would answer a question that was never asked.
	if existing.RequestHash != "" && requestHash != "" && existing.RequestHash != requestHash {
		return nil, ErrKeyMismatch
	}

	if existing.Status == StatusComplete {
		return existing, nil
	}
	return nil, ErrInFlight
}

// Complete stores the response so later retries replay it.
func (s *Store) Complete(ctx context.Context, key string, statusCode int32, response []byte) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.table),
		Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: key}},
		UpdateExpression: aws.String(
			"SET #s = :complete, #r = :response, #c = :code"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status", "#r": "response", "#c": "status_code",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":complete": &types.AttributeValueMemberS{Value: StatusComplete},
			":response": &types.AttributeValueMemberB{Value: response},
			":code":     &types.AttributeValueMemberN{Value: fmt.Sprint(statusCode)},
		},
	})
	if err != nil {
		return fmt.Errorf("idempotency: completing key %q: %w", key, err)
	}
	return nil
}

// Abandon releases a claim after a failure, so the client can retry
// immediately rather than waiting out the TTL.
//
// Only in-flight claims are deleted: a completed record must survive, because
// its whole purpose is to be replayed.
func (s *Store) Abandon(ctx context.Context, key string) error {
	_, err := s.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName:                aws.String(s.table),
		Key:                      map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: key}},
		ConditionExpression:      aws.String("#s = :inflight"),
		ExpressionAttributeNames: map[string]string{"#s": "status"},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inflight": &types.AttributeValueMemberS{Value: StatusInFlight},
		},
	})
	var conditionFailed *types.ConditionalCheckFailedException
	if errors.As(err, &conditionFailed) {
		return nil // already completed; leaving it is correct
	}
	if err != nil {
		return fmt.Errorf("idempotency: abandoning key %q: %w", key, err)
	}
	return nil
}

// Get reads a record, returning nil when absent.
func (s *Store) Get(ctx context.Context, key string) (*Record, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key:       map[string]types.AttributeValue{"pk": &types.AttributeValueMemberS{Value: key}},
		// Strongly consistent: the default eventually-consistent read can miss
		// a write from milliseconds ago, which under a retry storm is exactly
		// when this is being called.
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("idempotency: reading key %q: %w", key, err)
	}
	if len(out.Item) == 0 {
		return nil, nil
	}

	rec := &Record{Key: key}
	if v, ok := out.Item["status"].(*types.AttributeValueMemberS); ok {
		rec.Status = v.Value
	}
	if v, ok := out.Item["request_hash"].(*types.AttributeValueMemberS); ok {
		rec.RequestHash = v.Value
	}
	if v, ok := out.Item["response"].(*types.AttributeValueMemberB); ok {
		rec.Response = v.Value
	}
	if v, ok := out.Item["status_code"].(*types.AttributeValueMemberN); ok {
		var code int32
		_, _ = fmt.Sscan(v.Value, &code)
		rec.StatusCode = code
	}
	return rec, nil
}

// EnsureTable creates the table when absent. Development convenience; in
// production the table is Terraform's job.
func (s *Store) EnsureTable(ctx context.Context) error {
	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(s.table),
	})
	if err == nil {
		return nil
	}

	_, err = s.client.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(s.table),
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: types.KeyTypeHash},
		},
		// On-demand billing: a ticket drop is the definition of spiky traffic,
		// and provisioned capacity sized for the peak would idle at 1% between
		// drops while still being billed.
		BillingMode: types.BillingModePayPerRequest,
	})
	if err != nil {
		var inUse *types.ResourceInUseException
		if errors.As(err, &inUse) {
			return nil // another replica won the race
		}
		return fmt.Errorf("idempotency: creating table %q: %w", s.table, err)
	}
	return nil
}
