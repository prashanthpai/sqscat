package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const (
	waitTimeSeconds = 20
)

type sqsClient interface {
	ReceiveMessages(ctx context.Context, batchSize int) ([]types.Message, error)
	DeleteMessages(ctx context.Context, messages []types.Message) error
	SendMessage(ctx context.Context, body *string) error
}

type sqsQueueClient struct {
	client   *sqs.Client
	queueURL *string
}

func initSqs(ctx context.Context, queueName string) (*sqsQueueClient, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("config.LoadDefaultConfig() failed: %w", err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)

	resp, err := sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, fmt.Errorf("sqsClient.GetQueueUrl(%s) failed: %w", queueName, err)
	}

	return &sqsQueueClient{
		sqsClient,
		resp.QueueUrl,
	}, nil
}

func (q *sqsQueueClient) ReceiveMessages(ctx context.Context, batchSize int) ([]types.Message, error) {
	resp, err := q.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            q.queueURL,
		MaxNumberOfMessages: int32(batchSize),
		WaitTimeSeconds:     waitTimeSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("sqsClient.ReceiveMessage() failed: %w", err)
	}

	return resp.Messages, nil
}

func (q *sqsQueueClient) DeleteMessages(ctx context.Context, messages []types.Message) error {
	if len(messages) == 0 {
		return nil
	}

	var arr [maxNumberOfMessages]types.DeleteMessageBatchRequestEntry
	var entries = arr[:0]
	for i, msg := range messages {
		entries = append(entries, types.DeleteMessageBatchRequestEntry{
			Id:            aws.String(strconv.Itoa(i)),
			ReceiptHandle: msg.ReceiptHandle,
		})
	}

	resp, err := q.client.DeleteMessageBatch(context.Background(), &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: q.queueURL,
	})
	if err != nil {
		return fmt.Errorf("sqsClient.DeleteMessageBatch() failed: %w", err)
	}
	for _, msg := range resp.Failed {
		return fmt.Errorf("sqsClient.DeleteMessageBatch() failed: %v", *msg.Message)
	}

	return nil
}

func (q *sqsQueueClient) SendMessage(ctx context.Context, body *string) error {
	_, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    q.queueURL,
		MessageBody: body,
		// TODO: DelaySeconds?
	})
	if err != nil {
		return fmt.Errorf("sqsClient.SendMessage() failed: %w", err)
	}

	return nil
}
