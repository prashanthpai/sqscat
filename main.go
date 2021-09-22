package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	flags "github.com/jessevdk/go-flags"
)

const (
	maxNumberOfMessages = 10
	waitTimeSeconds     = 20
)

type opts struct {
	Concurrency int  `short:"c" long:"concurrency" description:"Number of concurrent SQS pollers; Defaults to 10 x Num. of CPUs"`
	Delete      bool `short:"d" long:"delete" description:"Delete received messages"`
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

	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("config.LoadDefaultConfig() failed: %v", err)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)

	resp, err := sqsClient.GetQueueUrl(context.Background(), &sqs.GetQueueUrlInput{
		QueueName: aws.String(opts.Positional.QueueName),
	})
	if err != nil {
		log.Fatalf("sqsClient.GetQueueUrl(%s) failed: %v", opts.Positional.QueueName, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if opts.Concurrency <= 0 {
		opts.Concurrency = 10 * runtime.NumCPU()
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		go poll(ctx, sqsClient, resp.QueueUrl, os.Stdout, opts.Delete, wg)
	}

	wg.Wait()
}

func poll(ctx context.Context, sqsClient *sqs.Client, queueURL *string, f *os.File, shouldDelete bool, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			resp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            queueURL,
				MaxNumberOfMessages: maxNumberOfMessages,
				WaitTimeSeconds:     waitTimeSeconds,
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				log.Fatalf("sqsClient.ReceiveMessage() failed: %v", err)
			}

			for _, msg := range resp.Messages {
				// sequential is ok, poller is concurrent
				handleMessage(f, *msg.Body)
			}

			if shouldDelete && len(resp.Messages) > 0 {
				deleteMessages(context.Background(), sqsClient, queueURL, resp.Messages)
			}
		}
	}
}

var outputDelimiter = []byte("\n")

func handleMessage(f *os.File, body string) {
	if _, err := f.WriteString(body); err != nil {
		log.Fatalf("WriteString() failed: %v", err)
	}
	if _, err := f.Write(outputDelimiter); err != nil {
		log.Fatalf("Write() failed: %v", err)
	}
}

func deleteMessages(ctx context.Context, sqsClient *sqs.Client, queueURL *string, messages []types.Message) {
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
		log.Fatalf("sqsClient.DeleteMessageBatch() failed: %v", err)
	}
	for _, msg := range resp.Failed {
		log.Fatalf("message delete failed: msg id = %s", *msg.Id)
	}
}
