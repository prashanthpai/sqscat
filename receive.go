package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
)

const (
	maxNumberOfMessages = 10
)

var (
	outputDelimiter = []byte("\n")
)

type handlerFunc func(string) error

func defaultHandler(body string) error {
	if _, err := os.Stdout.WriteString(body); err != nil {
		return fmt.Errorf("WriteString() failed: %w", err)
	}

	if _, err := os.Stdout.Write(outputDelimiter); err != nil {
		return fmt.Errorf("Write() failed: %w", err)
	}

	return nil
}

func poll(ctx context.Context, sqsClient sqsClient, queueURL *string, fn handlerFunc, shouldDelete bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if _, err := receiveMessages(ctx, sqsClient, queueURL, maxNumberOfMessages, fn, shouldDelete); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				log.Fatalf("sqsClient.ReceiveMessage() failed: %v", err)
			}
		}
	}
}

func pollWithLimit(ctx context.Context, sqsClient sqsClient, queueURL *string, limit int, fn handlerFunc, shouldDelete bool) error {
	for limit > 0 {
		batchSize := maxNumberOfMessages
		if limit < batchSize {
			batchSize = limit
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			rcvd, err := receiveMessages(ctx, sqsClient, queueURL, batchSize, fn, shouldDelete)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return err
			}
			limit -= rcvd
		}
	}

	return nil
}
