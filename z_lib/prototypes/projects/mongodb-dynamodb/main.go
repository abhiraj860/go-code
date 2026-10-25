package main

import (
	"context" // carries cancellation and timeout info between calls
	"fmt"     // print formatted output
	"log"     // print messages and quit
	"os"      // read environment variables
	"time"    // durations and timestamps

	// ---- DynamoDB (AWS SDK v2) ----
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	// attributevalue converts Go structs <-> DynamoDB's odd data format.
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	// types holds the enums and small structs the API needs.
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	// ---- MongoDB ----
	"go.mongodb.org/mongo-driver/bson"    // Mongo's binary JSON format
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const tableName = "orders"

// One order. Two sets of tags because each database names fields its
// own way. The backtick parts are "tags" — metadata attached to a
// field telling each library what to call it.
type Order struct {
	// dynamodbav is DynamoDB's tag; bson is MongoDB's.
	CustomerID string    `dynamodbav:"customer_id" bson:"customer_id"`
	OrderID    string    `dynamodbav:"order_id"    bson:"order_id"`
	Status     string    `dynamodbav:"status"      bson:"status"`
	Total      float64   `dynamodbav:"total"       bson:"total"`
	CreatedAt  time.Time `dynamodbav:"created_at"  bson:"created_at"`
	// A nested list. Mongo stores this naturally; DynamoDB flattens
	// it into its own nested format.
	Items []Item `dynamodbav:"items" bson:"items"`
}

type Item struct {
	SKU      string `dynamodbav:"sku"      bson:"sku"`
	Quantity int    `dynamodbav:"quantity" bson:"quantity"`
}

func main() {
	// An empty context — a required argument carrying "stop now"
	// signals. Background() is the do-nothing version.
	ctx := context.Background()

	fmt.Println("=== DYNAMODB ===")
	runDynamo(ctx)

	fmt.Println("\n=== MONGODB ===")
	runMongo(ctx)
}

// ================= DYNAMODB =================

func runDynamo(ctx context.Context) {
	endpoint := os.Getenv("DYNAMO_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}

	// Load AWS settings. Region is required even locally, and the
	// fake credentials are accepted by DynamoDB Local.
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("fake", "fake", ""),
		),
	)
	if err != nil {
		log.Fatal(err) // print the error and quit
	}

	// Build the client. The function passed in modifies the options —
	// o is a pointer, so changing o.X changes the real settings.
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		// Point at the local container instead of real AWS.
		// aws.String returns the ADDRESS of a string, because the SDK
		// uses *string to tell "not set" (nil) from "set to empty".
		o.BaseEndpoint = aws.String(endpoint)
	})

	createTable(ctx, client)
	putOrder(ctx, client)
	getOrder(ctx, client)
	queryByCustomer(ctx, client)
	updateStatus(ctx, client)
}

func createTable(ctx context.Context, c *dynamodb.Client) {
	// The & means "address of" — we build a struct and pass its
	// address, which is what the method expects.
	_, err := c.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),

		// Only KEY attributes are declared. Everything else is
		// schemaless — different rows can have completely different
		// fields, and DynamoDB doesn't care.
		AttributeDefinitions: []types.AttributeDefinition{
			// S means string. N is number, B is binary.
			{AttributeName: aws.String("customer_id"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("order_id"), AttributeType: types.ScalarAttributeTypeS},
		},

		// The key decides EVERYTHING about how you can query.
		// HASH (partition key) picks which physical partition the row
		// lives on. RANGE (sort key) orders rows within it.
		// You can only query by partition key — never by other fields
		// without a secondary index. This is the whole design.
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("customer_id"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("order_id"), KeyType: types.KeyTypeRange},
		},

		// PAY_PER_REQUEST means no capacity planning. The alternative
		// is PROVISIONED, where you pre-buy read/write units.
		BillingMode: types.BillingModePayPerRequest,
	})
	// The _ discards the first return value (table details) since we
	// don't need it.

	if err != nil {
		// Running twice is normal — the table already exists.
		log.Println("create table:", err)
		return
	}
	fmt.Println("created table", tableName)
}

func putOrder(ctx context.Context, c *dynamodb.Client) {
	o := Order{
		CustomerID: "cust-1",
		OrderID:    "ord-100",
		Status:     "pending",
		Total:      49.99,
		CreatedAt:  time.Now(),
		Items: []Item{
			{SKU: "widget-a", Quantity: 2},
			{SKU: "widget-b", Quantity: 1},
		},
	}

	// MarshalMap converts our struct into DynamoDB's format, where
	// every value is wrapped in a type marker. You almost never build
	// this by hand.
	item, err := attributevalue.MarshalMap(o)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	})
	if err != nil {
		log.Fatal("put:", err)
	}
	fmt.Println("put order", o.OrderID)

	// A second order for the same customer, so the query below has
	// more than one result.
	o2 := o
	o2.OrderID = "ord-101"
	o2.Total = 15.00
	o2.Status = "shipped"
	item2, _ := attributevalue.MarshalMap(o2)
	c.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item2,
	})
	fmt.Println("put order", o2.OrderID)
}

func getOrder(ctx context.Context, c *dynamodb.Client) {
	// GetItem needs the FULL key — both parts. This is the fastest
	// and cheapest operation DynamoDB offers.
	res, err := c.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		// map[string]types.AttributeValue means "field name -> value".
		Key: map[string]types.AttributeValue{
			// The & builds a pointer to the string-typed value.
			"customer_id": &types.AttributeValueMemberS{Value: "cust-1"},
			"order_id":    &types.AttributeValueMemberS{Value: "ord-100"},
		},
	})
	if err != nil {
		log.Fatal("get:", err)
	}

	// A missing item is NOT an error — Item is simply empty.
	if len(res.Item) == 0 {
		fmt.Println("not found")
		return
	}

	var o Order
	// UnmarshalMap converts back into our struct. The & is required
	// because it WRITES INTO o, so it needs o's memory address.
	if err := attributevalue.UnmarshalMap(res.Item, &o); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("got %s: %s, $%.2f, %d items\n", o.OrderID, o.Status, o.Total, len(o.Items))
}

func queryByCustomer(ctx context.Context, c *dynamodb.Client) {
	// Query works on ONE partition key value. It's efficient because
	// DynamoDB knows exactly which partition to read.
	res, err := c.Query(ctx, &dynamodb.QueryInput{
		TableName: aws.String(tableName),

		// The condition, written with placeholders. #c and :c are
		// substituted below — needed because some field names collide
		// with reserved words.
		KeyConditionExpression: aws.String("#c = :c"),
		ExpressionAttributeNames: map[string]string{
			"#c": "customer_id",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":c": &types.AttributeValueMemberS{Value: "cust-1"},
		},
	})
	if err != nil {
		log.Fatal("query:", err)
	}

	var orders []Order
	// UnmarshalListOfMaps handles a whole list at once.
	if err := attributevalue.UnmarshalListOfMaps(res.Items, &orders); err != nil {
		log.Fatal(err)
	}

	fmt.Println("orders for cust-1:")
	// range gives the index and the value; _ discards the index.
	for _, o := range orders {
		fmt.Printf("  %s  %s  $%.2f\n", o.OrderID, o.Status, o.Total)
	}
}

func updateStatus(ctx context.Context, c *dynamodb.Client) {
	_, err := c.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"customer_id": &types.AttributeValueMemberS{Value: "cust-1"},
			"order_id":    &types.AttributeValueMemberS{Value: "ord-100"},
		},
		// SET changes a field. #s is used because "status" is a
		// reserved word in DynamoDB's expression language.
		UpdateExpression: aws.String("SET #s = :new"),
		ExpressionAttributeNames: map[string]string{
			"#s": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":new": &types.AttributeValueMemberS{Value: "shipped"},
			":old": &types.AttributeValueMemberS{Value: "pending"},
		},
		// Only apply the change if the status is still "pending".
		// This is optimistic locking — it stops two writers from
		// overwriting each other silently.
		ConditionExpression: aws.String("#s = :old"),
	})
	if err != nil {
		log.Println("update (condition may have failed):", err)
		return
	}
	fmt.Println("updated ord-100 to shipped")
}

// ================= MONGODB =================

func runMongo(ctx context.Context) {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	// Connect. options.Client() builds a settings object; ApplyURI
	// fills it from the connection string.
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("mongo connect:", err)
	}
	// defer means "run this when the function ends, whatever happens".
	defer client.Disconnect(ctx)

	// Connect is lazy and doesn't actually dial. Ping forces a real
	// connection so we fail loudly now rather than on the first query.
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("mongo unreachable:", err)
	}

	// Database and collection are created automatically on first
	// write — no CREATE statement needed.
	coll := client.Database("shop").Collection("orders")

	createIndexes(ctx, coll)
	insertOrders(ctx, coll)
	findOne(ctx, coll)
	findMany(ctx, coll)
	updateOne(ctx, coll)
	aggregate(ctx, coll)
}

func createIndexes(ctx context.Context, coll *mongo.Collection) {
	// Without an index Mongo scans every document. Unlike DynamoDB,
	// you can index ANY field at any time — that's the core
	// difference in flexibility.
	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// bson.D is an ORDERED list of key-value pairs. Order
			// matters for compound indexes, which is why it's a list
			// and not a map.
			// 1 means ascending, -1 descending.
			Keys: bson.D{{Key: "customer_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			// A unique index prevents duplicate order IDs.
			Keys:    bson.D{{Key: "order_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			// You can even index inside a nested array — impossible
			// in DynamoDB without restructuring your data.
			Keys: bson.D{{Key: "items.sku", Value: 1}},
		},
	})
	if err != nil {
		log.Println("indexes:", err)
		return
	}
	fmt.Println("created indexes")
}

func insertOrders(ctx context.Context, coll *mongo.Collection) {
	orders := []any{ // any means "a value of any type"
		Order{
			CustomerID: "cust-1", OrderID: "ord-200", Status: "pending",
			Total: 89.99, CreatedAt: time.Now(),
			Items: []Item{{SKU: "widget-a", Quantity: 3}},
		},
		Order{
			CustomerID: "cust-1", OrderID: "ord-201", Status: "shipped",
			Total: 25.50, CreatedAt: time.Now().Add(-24 * time.Hour),
			Items: []Item{{SKU: "widget-c", Quantity: 1}},
		},
		Order{
			CustomerID: "cust-2", OrderID: "ord-202", Status: "pending",
			Total: 150.00, CreatedAt: time.Now(),
			Items: []Item{{SKU: "widget-a", Quantity: 5}},
		},
	}

	// InsertMany sends them all in one round trip.
	res, err := coll.InsertMany(ctx, orders)
	if err != nil {
		// Re-running hits the unique index — expected, not fatal.
		log.Println("insert (may be duplicates):", err)
		return
	}
	fmt.Println("inserted", len(res.InsertedIDs), "orders")
}

func findOne(ctx context.Context, coll *mongo.Collection) {
	var o Order

	// A filter is just a document describing what to match.
	// bson.M is an unordered map — fine for simple filters.
	// Decode writes into o, so it needs o's address via &.
	err := coll.FindOne(ctx, bson.M{"order_id": "ord-200"}).Decode(&o)

	// ErrNoDocuments is the specific "nothing matched" error.
	if err == mongo.ErrNoDocuments {
		fmt.Println("not found")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("found %s: %s, $%.2f\n", o.OrderID, o.Status, o.Total)
}

func findMany(ctx context.Context, coll *mongo.Collection) {
	// Query by ANY field combination — no index required for it to
	// work, though one makes it fast. DynamoDB would need a whole
	// secondary index for this.
	// $gt means "greater than"; $ prefixes all Mongo operators.
	filter := bson.M{
		"status": "pending",
		"total":  bson.M{"$gt": 50.0},
	}

	// Sort newest first and cap the results.
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(10)

	cur, err := coll.Find(ctx, filter, opts)
	if err != nil {
		log.Fatal(err)
	}
	// A cursor streams results; closing it frees server resources.
	defer cur.Close(ctx)

	var orders []Order
	// All drains the cursor into the slice at once. For huge results
	// you'd loop with cur.Next() instead to avoid loading everything.
	if err := cur.All(ctx, &orders); err != nil {
		log.Fatal(err)
	}

	fmt.Println("pending orders over $50:")
	for _, o := range orders {
		fmt.Printf("  %s  %s  $%.2f\n", o.OrderID, o.CustomerID, o.Total)
	}
}

func updateOne(ctx context.Context, coll *mongo.Collection) {
	// $set changes only the listed fields, leaving the rest alone.
	// Without $set you'd REPLACE the whole document — a classic
	// beginner mistake that silently deletes fields.
	res, err := coll.UpdateOne(ctx,
		bson.M{"order_id": "ord-200"},
		bson.M{"$set": bson.M{"status": "shipped"}},
	)
	if err != nil {
		log.Fatal(err)
	}
	// MatchedCount is how many matched the filter; ModifiedCount is
	// how many actually changed. They differ when the value was
	// already what you set.
	fmt.Printf("matched %d, modified %d\n", res.MatchedCount, res.ModifiedCount)
}

func aggregate(ctx context.Context, coll *mongo.Collection) {
	// An aggregation pipeline: documents flow through stages, each
	// transforming them. This is Mongo's biggest advantage — there's
	// no DynamoDB equivalent, you'd read everything and compute in
	// your app.
	pipeline := mongo.Pipeline{
		// Stage 1: keep only pending orders.
		bson.D{{Key: "$match", Value: bson.M{"status": "pending"}}},

		// Stage 2: group by customer, summing and counting.
		// _id is what you group BY. $total means "the total field".
		bson.D{{Key: "$group", Value: bson.M{
			"_id":     "$customer_id",
			"revenue": bson.M{"$sum": "$total"},
			"count":   bson.M{"$sum": 1}, // 1 per document = a count
		}}},

		// Stage 3: sort by the computed revenue field.
		bson.D{{Key: "$sort", Value: bson.M{"revenue": -1}}},
	}

	cur, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		log.Fatal(err)
	}
	defer cur.Close(ctx)

	// The output shape is new, so decode into maps rather than Order.
	var results []bson.M
	if err := cur.All(ctx, &results); err != nil {
		log.Fatal(err)
	}

	fmt.Println("pending revenue by customer:")
	for _, r := range results {
		fmt.Printf("  %v: $%.2f (%v orders)\n", r["_id"], r["revenue"], r["count"])
	}
}