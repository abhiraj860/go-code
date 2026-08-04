package main

import (
	"context" // carries cancellation and timeout info between calls
	"fmt"     // print formatted output
	"io"      // read streams of bytes
	"log"     // print messages and quit
	"strings" // turn a string into something readable, like a tiny file

	// The AWS SDK v2 packages. aws holds shared types, config loads
	// settings, s3 is the actual S3 client.
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// A bucket is S3's top-level container, roughly like a folder that
// holds files. Names must be globally unique on real AWS.
const bucket = "my-test-bucket"

// Where LocalStack is listening. On real AWS you'd delete this line
// entirely and the SDK would find Amazon's servers automatically.
const endpoint = "http://localhost:4566"

func main() {
	// An empty context — a required argument that carries "stop now"
	// signals. We have nothing to cancel, so Background() is the
	// do-nothing version.
	ctx := context.Background()

	// Load AWS settings. LoadDefaultConfig normally reads your real
	// credentials file; here we override the pieces we need.
	cfg, err := config.LoadDefaultConfig(ctx,
		// Region is required even for LocalStack, which ignores it.
		config.WithRegion("us-east-1"),

		// Fake credentials. LocalStack accepts anything non-empty.
		// NewStaticCredentialsProvider takes key, secret, and a
		// session token we leave blank with "".
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		),
	)
	if err != nil {
		log.Fatal(err) // print the error and quit
	}

	// Build the S3 client. The function passed in modifies the
	// options — o is a pointer to them, so changing o.X changes the
	// real settings, not a copy.
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// Point at LocalStack instead of Amazon.
		// aws.String returns the ADDRESS of a string, because the SDK
		// uses *string to tell "not set" (nil) from "set to empty".
		o.BaseEndpoint = aws.String(endpoint)

		// Path style means URLs look like localhost:4566/bucket/key
		// instead of bucket.localhost:4566/key. LocalStack needs this
		// because the fancy subdomain form doesn't work on localhost.
		o.UsePathStyle = true
	})

	// Run each operation in order.
	createBucket(ctx, client)
	putObject(ctx, client, "hello.txt", "hello from go")
	putObject(ctx, client, "logs/app.log", "line one\nline two")
	listObjects(ctx, client)
	getObject(ctx, client, "hello.txt")
	deleteObject(ctx, client, "hello.txt")
	listObjects(ctx, client)
}

// ---------------- CREATE BUCKET ----------------

func createBucket(ctx context.Context, c *s3.Client) {
	// Every SDK call takes a struct of parameters. The & gives its
	// address, which is what the method expects.
	_, err := c.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	// The _ discards the first return value (details about the new
	// bucket) since we don't need it.

	if err != nil {
		// Creating a bucket that already exists is an error, but a
		// harmless one — so log it and carry on instead of quitting.
		log.Println("create bucket:", err)
		return
	}
	fmt.Println("created bucket", bucket)
}

// ---------------- PUT (upload) ----------------

func putObject(ctx context.Context, c *s3.Client, key, content string) {
	// "key" is S3's word for the object's name, e.g. "logs/app.log".
	// S3 has no real folders — the slashes are just part of the name.
	_, err := c.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),

		// Body must be something readable, like a file. strings.NewReader
		// wraps our text so it can be read from. For a real file you'd
		// pass the result of os.Open here instead.
		Body: strings.NewReader(content),
	})
	if err != nil {
		log.Fatal("put:", err)
	}
	fmt.Println("uploaded", key)
}

// ---------------- GET (download) ----------------

func getObject(ctx context.Context, c *s3.Client, key string) {
	res, err := c.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Fatal("get:", err)
	}
	// defer means "run this when the function ends, whatever happens".
	// Closing the body returns the network connection for reuse.
	defer res.Body.Close()

	// The body is a stream, not text. ReadAll pulls it all into memory.
	// Fine for small objects; for large ones you'd io.Copy it straight
	// to a file instead of holding it all at once.
	data, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// string(data) converts the raw bytes into readable text.
	fmt.Printf("downloaded %s: %q\n", key, string(data))

	// ContentLength is a *int64 — a pointer, so it might be nil.
	// The * before it reads the value at that address.
	if res.ContentLength != nil {
		fmt.Println("  size:", *res.ContentLength, "bytes")
	}
}

// ---------------- LIST ----------------

func listObjects(ctx context.Context, c *s3.Client) {
	// V2 is the newer version of the list operation; use it over V1.
	res, err := c.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		log.Fatal("list:", err)
	}

	fmt.Println("objects in bucket:")

	// res.Contents is a slice (list) of objects. If the bucket is
	// empty it's simply empty, and the loop body never runs.
	// The _ discards the index number; we only want each value.
	for _, obj := range res.Contents {
		// Both Key and Size are pointers, so * reads their values.
		fmt.Printf("  %s (%d bytes)\n", *obj.Key, *obj.Size)
	}

	// S3 returns at most 1000 objects per call. IsTruncated being true
	// means there are more, and you'd re-call with a continuation
	// token to fetch the next page.
	if res.IsTruncated != nil && *res.IsTruncated {
		fmt.Println("  (more objects exist — this is only the first page)")
	}
}

// ---------------- DELETE ----------------

func deleteObject(ctx context.Context, c *s3.Client, key string) {
	_, err := c.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Fatal("delete:", err)
	}
	fmt.Println("deleted", key)
}
