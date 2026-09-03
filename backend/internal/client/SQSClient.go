package client

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// SQSClient provides a small wrapper around the AWS SQS client.
type SQSClient struct {
	client   *sqs.Client
	queueURL string
}

// NewSQSClient creates a new SQS client with the provided AWS configuration.
func NewSQSClient(ctx context.Context) (*SQSClient, error) {
	// Get queue URL from params, falling back to mocked out URL
	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		queueURL = "mock-queue"
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return &SQSClient{
		client:   sqs.NewFromConfig(cfg),
		queueURL: queueURL,
	}, nil
}

// SendMessage sends a message to the configured queue.
func (c *SQSClient) SendMessage(ctx context.Context, body string) (*sqs.SendMessageOutput, error) {
	if c.client == nil {
		return nil, fmt.Errorf("sqs client is not initialized")
	}

	return c.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(c.queueURL),
		MessageBody: aws.String(body),
	})
}
