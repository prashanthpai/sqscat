package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync"
)

func isSendMode() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return (fi.Mode() & os.ModeCharDevice) == 0
}

type nextFunc func() (string, error)

func stdinNextFunc() nextFunc {
	scanner := bufio.NewScanner(os.Stdin)

	return func() (string, error) {
		for scanner.Scan() {
			return scanner.Text(), nil
		}
		if err := scanner.Err(); err != nil {
			return "", err
		} else {
			return "", io.EOF
		}
	}
}

func dispatch(ctx context.Context, sqsClient sqsClient, queueURL *string, next nextFunc, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			body, err := next()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Fatalf("stdin: %v", err)
			}
			if err := sendMessage(ctx, sqsClient, queueURL, &body); err != nil {
				log.Fatalf("sendMessage() failed: %v", err)
			}
		}
	}
}

func dispatchWithLimit(ctx context.Context, sqsClient sqsClient, queueURL *string, next nextFunc, limit int) error {
	for limit > 0 {
		select {
		case <-ctx.Done():
			return nil
		default:
			body, err := next()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			if err := sendMessage(ctx, sqsClient, queueURL, &body); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return err
			}
			limit--
		}
	}

	return nil
}
