// Command notification-svc bridges ticket.issued into SQS and out over SNS.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/labstack/echo/v4"
	segkafka "github.com/segmentio/kafka-go"

	"github.com/abhiraj860/ticketflow/pkg/awsx"
	"github.com/abhiraj860/ticketflow/pkg/config"
	tfkafka "github.com/abhiraj860/ticketflow/pkg/kafka"
	"github.com/abhiraj860/ticketflow/pkg/logging"
	"github.com/abhiraj860/ticketflow/services/notification-svc/internal/adminapi"
	"github.com/abhiraj860/ticketflow/services/notification-svc/internal/notifier"
	"github.com/abhiraj860/ticketflow/services/notification-svc/internal/sqsconsumer"
)

const serviceName = "notification-svc"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	l := config.New("NOTIFICATION")
	var (
		httpAddr    = l.String("HTTP_ADDR", ":9162")
		brokers     = l.String("KAFKA_BROKERS", "localhost:9092")
		groupID     = l.String("KAFKA_GROUP", "notification-bridge")
		queueName   = l.String("SQS_QUEUE", "ticketflow-notifications")
		topicName   = l.String("SNS_TOPIC", "ticketflow-notifications")
		awsEndpoint = l.String("AWS_ENDPOINT", "http://localhost:4566")
		awsRegion   = l.String("AWS_REGION", "us-east-1")
		awsKey      = l.String("AWS_ACCESS_KEY", "test")
		awsSecret   = l.String("AWS_SECRET_KEY", "test")
		logLevel    = l.String("LOG_LEVEL", "info")
		logFormat   = l.String("LOG_FORMAT", "json")
	)
	if err := l.Err(); err != nil {
		return err
	}

	logger := logging.New(logging.Options{Service: serviceName, Level: logLevel, Format: logging.Format(logFormat)})
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	awsCfg, err := awsx.Load(ctx, awsx.Config{
		Region: awsRegion, Endpoint: awsEndpoint,
		AccessKey: awsKey, SecretKey: awsSecret,
	})
	if err != nil {
		return err
	}
	sqsClient, snsClient := awsx.NewSQS(awsCfg), awsx.NewSNS(awsCfg)

	queueURL, err := sqsconsumer.EnsureQueue(ctx, sqsClient, queueName)
	if err != nil {
		return fmt.Errorf("ensuring queue: %w", err)
	}
	logger.Info("queue ready", slog.String("url", queueURL))

	topic, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String(topicName)})
	if err != nil {
		return fmt.Errorf("ensuring topic: %w", err)
	}
	topicARN := aws.ToString(topic.TopicArn)
	logger.Info("topic ready", slog.String("arn", topicARN))

	publisher := notifier.New(snsClient, topicARN, logger)

	consumer, err := sqsconsumer.New(sqsconsumer.Options{
		Client: sqsClient, QueueURL: queueURL, Logger: logger, Concurrency: 4,
	})
	if err != nil {
		return err
	}

	// Kafka -> SQS bridge. ticket-svc announces on Kafka because that is the
	// event backbone; this puts it on a queue so a slow email provider cannot
	// back-pressure into ticket generation.
	producer, err := tfkafka.NewProducer(tfkafka.ProducerOptions{Brokers: strings.Split(brokers, ",")})
	if err != nil {
		return fmt.Errorf("kafka producer: %w", err)
	}
	defer func() { _ = producer.Close() }()

	kafkaConsumer, err := tfkafka.NewConsumer(tfkafka.ConsumerOptions{
		Brokers: strings.Split(brokers, ","), Topic: tfkafka.TopicTicketIssued,
		GroupID: groupID, Concurrency: 2, BatchSize: 20, MaxAttempts: 3,
		DLQTopic: tfkafka.TopicDLQ, DLQProducer: producer, Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("kafka consumer: %w", err)
	}
	defer func() { _ = kafkaConsumer.Close() }()

	go func() {
		err := kafkaConsumer.Run(ctx, func(c context.Context, msg segkafka.Message) error {
			env, err := tfkafka.Unmarshal[tfkafka.TicketIssued](msg.Value)
			if err != nil {
				return tfkafka.Permanent(err)
			}
			body, err := json.Marshal(env.Payload)
			if err != nil {
				return tfkafka.Permanent(err)
			}
			_, err = sqsClient.SendMessage(c, &sqs.SendMessageInput{
				QueueUrl: aws.String(queueURL), MessageBody: aws.String(string(body)),
			})
			return err
		})
		if err != nil {
			logger.Error("kafka bridge stopped", slog.Any("error", err))
		}
	}()

	// SQS -> SNS delivery.
	go func() {
		err := consumer.Run(ctx, func(c context.Context, body string) error {
			var t tfkafka.TicketIssued
			if err := json.Unmarshal([]byte(body), &t); err != nil {
				// Unparseable will never parse. Returning nil deletes it rather
				// than cycling it to the DLQ forever.
				logger.WarnContext(c, "dropping malformed notification", slog.String("body", body))
				return nil
			}
			return publisher.Send(c, notifier.Notification{
				TicketID: t.TicketID, OrderID: t.OrderID, EventID: t.EventID,
				SeatID: t.SeatID, UserID: t.UserID, PDFKey: t.PDFKey, Channel: "email",
			})
		})
		if err != nil {
			logger.Error("sqs consumer stopped", slog.Any("error", err))
		}
	}()

	e := echo.New()
	e.HideBanner, e.HidePort = true, true
	adminapi.New(publisher, consumer).Register(e)

	go func() {
		logger.Info("echo admin api listening", slog.String("addr", httpAddr))
		if err := e.Start(httpAddr); err != nil {
			logger.Info("echo stopped", slog.Any("reason", err))
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = e.Shutdown(shutdownCtx)

	logger.Info("shutdown complete")
	return nil
}
