package main

import (
	"context"
	"errors"
	"fmt"
	"os"
)

const (
	maxNumberOfMessages = 10
)

var (
	outputDelimiter = []byte("\n")
)

type outFunc func(string) error

func defaultHandler(body string) error {
	if _, err := os.Stdout.WriteString(body); err != nil {
		return fmt.Errorf("WriteString() failed: %w", err)
	}

	if _, err := os.Stdout.Write(outputDelimiter); err != nil {
		return fmt.Errorf("Write() failed: %w", err)
	}

	return nil
}

func poll(ctx context.Context, sqsClient sqsClient, fn outFunc, shouldDelete bool) error {
	for {
		_, err := pollCommon(ctx, sqsClient, fn, maxNumberOfMessages, shouldDelete)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}

			return err
		}
	}
}

func pollWithLimit(ctx context.Context, sqsClient sqsClient, fn outFunc, limit int, shouldDelete bool) error {
	for limit > 0 {
		batchSize := maxNumberOfMessages
		if limit < batchSize {
			batchSize = limit
		}

		count, err := pollCommon(ctx, sqsClient, fn, batchSize, shouldDelete)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}

			return err
		}

		limit -= count
	}

	return nil
}

func pollCommon(ctx context.Context, sqsClient sqsClient, fn outFunc, batchSize int, shouldDelete bool) (int, error) {
	select {
	case <-ctx.Done():
		return 0, nil
	default:
		messages, err := sqsClient.ReceiveMessages(ctx, batchSize)
		if err != nil {
			return 0, err
		}

		for _, msg := range messages {
			if err := fn(*msg.Body); err != nil {
				return len(messages), err
			}
		}

		if shouldDelete {
			if err := sqsClient.DeleteMessages(context.Background(), messages); err != nil {
				return len(messages), err
			}
		}

		return len(messages), nil
	}
}
