package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

const (
	queueURL    = "https://sqs.us-east-1.amazonaws.com/xxxxxxxx/yyyyyyyyy"
	concurrency = 10
)

var (
	outputDelimiter = []byte("\n")
)

func main() {
	awsCfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("config.LoadDefaultConfig() failed: %v", err)
	}
	sqsClient := sqs.NewFromConfig(awsCfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	wg := &sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go poll(ctx, sqsClient, os.Stdout, wg)
	}

	wg.Wait()
}

func poll(ctx context.Context, sqsClient *sqs.Client, f *os.File, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Printf("poller exited")
			return
		default:
			resp, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
				QueueUrl:            aws.String(queueURL),
				MaxNumberOfMessages: int32(10),
				WaitTimeSeconds:     int32(20),
			})
			if err != nil {
				if errors.Is(err, context.Canceled) {
					log.Printf("poller exited")
					return
				}
				log.Fatalf("sqsClient.ReceiveMessage() failed: %v", err)
			}

			log.Printf("received messages; count = %d\n", len(resp.Messages))

			for _, msg := range resp.Messages {
				// sequential is ok, poller is concurrent
				handleMessage(f, *msg.Body)
			}
		}
	}
}

func handleMessage(f *os.File, body string) {
	if _, err := f.WriteString(body); err != nil {
		log.Fatalf("WriteString() failed: %v", err)
	}
	if _, err := f.Write(outputDelimiter); err != nil {
		log.Fatalf("Write() failed: %v", err)
	}
}
