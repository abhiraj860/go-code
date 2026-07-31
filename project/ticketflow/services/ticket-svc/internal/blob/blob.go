// Package blob stores generated ticket artifacts.
//
// An interface with two implementations rather than calling the AWS SDK
// directly: S3 for real use, and a filesystem store so ticket generation can be
// tested without LocalStack running. The PDF pipeline is the interesting part
// of ticket-svc and should not be untestable because a 1.2GB container is not
// up.
package blob

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// ErrNotFound is returned when a key does not exist.
var ErrNotFound = errors.New("blob: not found")

// Store persists opaque objects by key.
type Store interface {
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Exists(ctx context.Context, key string) (bool, error)
}

// ---------------------------------------------------------------- S3

// S3Store writes to S3, or to anything speaking its API (LocalStack, MinIO).
type S3Store struct {
	client *s3.Client
	bucket string
}

// S3Options configures the client.
type S3Options struct {
	Bucket string
	Region string
	// Endpoint overrides the AWS endpoint. Set to http://localhost:4566 for
	// LocalStack; empty means real AWS.
	Endpoint string
	// AccessKey/SecretKey are for LocalStack, which accepts anything. Real
	// deployments leave these empty and let the SDK use the task role -- an
	// IAM role, never a long-lived key in an environment variable.
	AccessKey string
	SecretKey string
}

func NewS3Store(ctx context.Context, opts S3Options) (*S3Store, error) {
	if opts.Bucket == "" {
		return nil, errors.New("blob: bucket is required")
	}
	if opts.Region == "" {
		opts.Region = "us-east-1"
	}

	loadOpts := []func(*config.LoadOptions) error{config.WithRegion(opts.Region)}
	if opts.AccessKey != "" {
		loadOpts = append(loadOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, "")))
	}

	cfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("blob: loading aws config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if opts.Endpoint != "" {
			o.BaseEndpoint = aws.String(opts.Endpoint)
			// LocalStack and MinIO serve buckets as a path segment rather than
			// a DNS subdomain, which is what the SDK assumes by default.
			o.UsePathStyle = true
		}
	})

	return &S3Store{client: client, bucket: opts.Bucket}, nil
}

// EnsureBucket creates the bucket when absent. Used in development; in
// production the bucket is Terraform's job, not the application's.
func (s *S3Store) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil {
		return nil
	}
	if _, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)}); err != nil {
		// A concurrent replica may have won the race, which is fine.
		if !strings.Contains(err.Error(), "BucketAlreadyOwnedByYou") &&
			!strings.Contains(err.Error(), "BucketAlreadyExists") {
			return fmt.Errorf("blob: creating bucket %q: %w", s.bucket, err)
		}
	}
	return nil
}

func (s *S3Store) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("blob: putting %q: %w", key, err)
	}
	return nil
}

func (s *S3Store) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") || strings.Contains(err.Error(), "NotFound") {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("blob: getting %q: %w", key, err)
	}
	defer func() { _ = out.Body.Close() }()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("blob: reading %q: %w", key, err)
	}
	return data, nil
}

// Exists is the idempotency check: a redelivered order.created must not
// regenerate a PDF that already exists.
func (s *S3Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("blob: heading %q: %w", key, err)
	}
	return true, nil
}

// ---------------------------------------------------------------- filesystem

// FSStore writes under a directory. Used by tests and local development so the
// ticket pipeline does not require LocalStack.
type FSStore struct {
	root string
	mu   sync.RWMutex
}

func NewFSStore(root string) (*FSStore, error) {
	if root == "" {
		return nil, errors.New("blob: root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("blob: creating %q: %w", root, err)
	}
	return &FSStore{root: root}, nil
}

// path maps a key to a file, refusing anything that escapes the root.
func (f *FSStore) path(key string) (string, error) {
	clean := filepath.Clean("/" + key) // strips any ../ prefix
	full := filepath.Join(f.root, clean)
	if !strings.HasPrefix(full, filepath.Clean(f.root)+string(os.PathSeparator)) {
		return "", fmt.Errorf("blob: key %q escapes the store root", key)
	}
	return full, nil
}

func (f *FSStore) Put(_ context.Context, key string, data []byte, _ string) error {
	full, err := f.path(key)
	if err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("blob: creating directory for %q: %w", key, err)
	}
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return fmt.Errorf("blob: writing %q: %w", key, err)
	}
	return nil
}

func (f *FSStore) Get(_ context.Context, key string) ([]byte, error) {
	full, err := f.path(key)
	if err != nil {
		return nil, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := os.ReadFile(full)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("blob: reading %q: %w", key, err)
	}
	return data, nil
}

func (f *FSStore) Exists(_ context.Context, key string) (bool, error) {
	full, err := f.path(key)
	if err != nil {
		return false, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	if _, err := os.Stat(full); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("blob: stating %q: %w", key, err)
	}
	return true, nil
}

var (
	_ Store = (*S3Store)(nil)
	_ Store = (*FSStore)(nil)
)
