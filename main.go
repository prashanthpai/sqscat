package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"

	flags "github.com/jessevdk/go-flags"
)

const (
	maxNumberOfMessages = 10
)

type handlerFunc func(string) error

var outputDelimiter = []byte("\n")

func defaultHandler(body string) error {
	if _, err := os.Stdout.WriteString(body); err != nil {
		return fmt.Errorf("WriteString() failed: %w", err)
	}

	if _, err := os.Stdout.Write(outputDelimiter); err != nil {
		return fmt.Errorf("Write() failed: %w", err)
	}

	return nil
}

type opts struct {
	Concurrency int  `short:"c" long:"concurrency" description:"Number of concurrent SQS pollers; Defaults to 10 x Num. of CPUs"`
	Delete      bool `short:"d" long:"delete" description:"Delete received messages"`
	NumMessages int  `short:"n" long:"num-messages" description:"Receive specified number of messages and exit; This limits concurrency to 1"`
	Positional  struct {
		QueueName string `positional-arg-name:"queue-name"`
	} `positional-args:"true" required:"true"`
}

func main() {
	opts := opts{}
	if _, err := flags.Parse(&opts); err != nil {
		if e, ok := err.(*flags.Error); ok {
			if e.Type == flags.ErrHelp {
				os.Exit(0)
			} else {
				os.Exit(1)
			}
		}
	}

	sqsClient, queueUrl, err := initSqs(opts.Positional.QueueName)
	if err != nil {
		log.Fatalf("initSqs() failed: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if opts.NumMessages > 0 {
		// sync with limit
		if err := pollWithLimit(ctx, sqsClient, queueUrl, opts.NumMessages, defaultHandler, opts.Delete); err != nil {
			log.Fatalf("pollWithLimit() failed: %v", err)
		}
		return
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 10 * runtime.NumCPU()
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go poll(ctx, sqsClient, queueUrl, defaultHandler, opts.Delete, wg)
	}

	wg.Wait()
}

func poll(ctx context.Context, sqsClient sqsClient, queueURL *string, fn handlerFunc, shouldDelete bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if _, err := receiveMessages(ctx, sqsClient, queueURL, maxNumberOfMessages, fn, shouldDelete); err != nil {
				if errors.Is(err, context.Canceled) {
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
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return err
			}
			limit -= rcvd
		}
	}

	return nil
}
