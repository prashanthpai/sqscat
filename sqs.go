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
	ReceiveMessage(context.Context, *sqs.ReceiveMessageInput, ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessageBatch(context.Context, *sqs.DeleteMessageBatchInput, ...func(*sqs.Options)) (*sqs.DeleteMessageBatchOutput, error)
}

func initSqs(ctx context.Context, queueName string) (*sqs.Client, *string, error) {
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("config.LoadDefaultConfig() failed: %w", err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)

	resp, err := sqsClient.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("sqsClient.GetQueueUrl(%s) failed: %w", queueName, err)
	}

	return sqsClient, resp.QueueUrl, nil
}

func receiveMessages(ctx context.Context, sqsClient sqsClient, queueURL *string, batchSize int, fn handlerFunc, shouldDelete bool) (int, error) {
	resp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            queueURL,
		MaxNumberOfMessages: int32(batchSize),
		WaitTimeSeconds:     waitTimeSeconds,
	})
	if err != nil {
		return 0, fmt.Errorf("sqsClient.ReceiveMessage() failed: %w", err)
	}
	rcvd := len(resp.Messages)

	for _, msg := range resp.Messages {
		if err := fn(*msg.Body); err != nil {
			return rcvd, err
		}
	}

	if shouldDelete && len(resp.Messages) > 0 {
		if err := deleteMessages(context.Background(), sqsClient, queueURL, resp.Messages); err != nil {
			return rcvd, err
		}
	}

	return rcvd, nil
}

func deleteMessages(ctx context.Context, sqsClient sqsClient, queueURL *string, messages []types.Message) error {
	var arr [maxNumberOfMessages]types.DeleteMessageBatchRequestEntry
	var entries = arr[:0]
	for i, msg := range messages {
		entries = append(entries, types.DeleteMessageBatchRequestEntry{
			Id:            aws.String(strconv.Itoa(i)),
			ReceiptHandle: msg.ReceiptHandle,
		})
	}

	resp, err := sqsClient.DeleteMessageBatch(context.Background(), &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: queueURL,
	})
	if err != nil {
		return fmt.Errorf("sqsClient.DeleteMessageBatch() failed: %w", err)
	}
	for _, msg := range resp.Failed {
		return fmt.Errorf("sqsClient.DeleteMessageBatch() failed: %v", *msg.Message)
	}

	return nil
}
