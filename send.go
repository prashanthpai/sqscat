package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
)

func isSendMode() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return (fi.Mode() & os.ModeCharDevice) == 0
}

type inFunc func() (string, error)

func stdinNextFunc() inFunc {
	scanner := bufio.NewScanner(os.Stdin)

	return func() (string, error) {
		for scanner.Scan() {
			return scanner.Text(), nil
		}

		if err := scanner.Err(); err != nil {
			return "", err
		}

		return "", io.EOF
	}
}

func dispatch(ctx context.Context, sqsClient sqsClient, next inFunc) error {
	for {
		if err := dispatchCommon(ctx, sqsClient, next); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}
	}
}

func dispatchWithLimit(ctx context.Context, sqsClient sqsClient, next inFunc, limit int) error {
	for limit > 0 {
		if err := dispatchCommon(ctx, sqsClient, next); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}
		limit--
	}

	return nil
}

func dispatchCommon(ctx context.Context, sqsClient sqsClient, next inFunc) error {
	select {
	case <-ctx.Done():
		return nil
	default:
		body, err := next()
		if err != nil {
			return err
		}

		return sqsClient.SendMessage(ctx, &body)
	}
}
