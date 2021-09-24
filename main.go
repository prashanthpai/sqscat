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
	"time"

	flags "github.com/jessevdk/go-flags"
)

var (
	version = "undefined"
	commit  = "undefined"
)

type opts struct {
	Version     bool `short:"v" long:"version" description:"Print version and exit"`
	Concurrency int  `short:"c" long:"concurrency" description:"Number of concurrent SQS pollers; Defaults to 10 x Num. of CPUs"`
	Delete      bool `short:"d" long:"delete" description:"Delete received messages"`
	NumMessages int  `short:"n" long:"num-messages" description:"Receive specified number of messages and exit; This limits concurrency to 1"`
	Time        int  `short:"t" long:"timeout" description:"Exit after specified number of seconds"`
	Positional  struct {
		QueueName string `positional-arg-name:"queue-name"`
	} `positional-args:"true" required:"true"`
}

func parseOpts(opts *opts) {
	_, err := flags.NewParser(opts, flags.HelpFlag|flags.PassDoubleDash).Parse()
	if opts.Version {
		fmt.Println(version, commit)
		os.Exit(0)
	}
	if err != nil {
		if t, ok := err.(*flags.Error); ok && t.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stdout, err.Error())
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func main() {
	opts := opts{}
	parseOpts(&opts)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if opts.Time > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.Time)*time.Second)
		defer cancel()
	}

	sqsClient, queueURL, err := initSqs(ctx, opts.Positional.QueueName)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		log.Fatalf("initSqs() failed: %v", err)
	}

	sendMode := isSendMode()

	if opts.NumMessages > 0 {
		if sendMode {
			fn := stdinNextFunc()
			if err := dispatchWithLimit(ctx, sqsClient, queueURL, fn, opts.NumMessages); err != nil {
				log.Fatalf("dispatchWithLimit() failed: %v", err)
			}
		} else {
			if err := pollWithLimit(ctx, sqsClient, queueURL, opts.NumMessages, defaultHandler, opts.Delete); err != nil {
				log.Fatalf("pollWithLimit() failed: %v", err)
			}
		}
		return
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 10 * runtime.NumCPU()
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < opts.Concurrency; i++ {
		wg.Add(1)
		if sendMode {
			fn := stdinNextFunc()
			go dispatch(ctx, sqsClient, queueURL, fn, wg)
		} else {
			go poll(ctx, sqsClient, queueURL, defaultHandler, opts.Delete, wg)
		}
	}

	wg.Wait()
}
