package awsx

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Config describes how to reach AWS.
type Config struct {
	Region string
	// Endpoint overrides the AWS endpoint. http://localhost:4566 for
	// LocalStack; empty means real AWS.
	Endpoint string
	// AccessKey/SecretKey exist for LocalStack, which accepts anything. Real
	// deployments leave these empty so the SDK uses the ECS task role -- an IAM
	// role, never a long-lived key in an environment variable.
	AccessKey string
	SecretKey string
}

// Load builds an AWS config, wiring LocalStack when an endpoint is set.
func Load(ctx context.Context, cfg Config) (aws.Config, error) {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}

	opts := []func(*config.LoadOptions) error{config.WithRegion(cfg.Region)}
	if cfg.AccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")))
	}
	if cfg.Endpoint != "" {
		opts = append(opts, config.WithBaseEndpoint(cfg.Endpoint))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("awsx: loading config: %w", err)
	}
	return awsCfg, nil
}

func NewDynamoDB(cfg aws.Config) *dynamodb.Client { return dynamodb.NewFromConfig(cfg) }
func NewSQS(cfg aws.Config) *sqs.Client           { return sqs.NewFromConfig(cfg) }
func NewSNS(cfg aws.Config) *sns.Client           { return sns.NewFromConfig(cfg) }
