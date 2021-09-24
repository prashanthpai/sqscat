package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"time"

	flags "github.com/jessevdk/go-flags"
	"golang.org/x/sync/errgroup"
)

var (
	version = "undefined"
	commit  = "undefined"
)

type opts struct {
	Version     bool `short:"v" long:"version" description:"Print version and exit"`
	Concurrency int  `short:"c" long:"concurrency" description:"Number of concurrent SQS receivers/senders; Defaults to 10 x Num. of CPUs"`
	Delete      bool `short:"d" long:"delete" description:"Delete received messages"`
	NumMessages int  `short:"n" long:"num-messages" description:"Receive/send specified number of messages and exit; This limits concurrency to 1"`
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

func run(ctx context.Context, opts opts) error {
	if opts.Time > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(opts.Time)*time.Second)
		defer cancel()
	}

	sqsClient, err := initSqs(ctx, opts.Positional.QueueName)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		return err
	}

	sendMode := isSendMode()

	if opts.NumMessages > 0 {
		if sendMode {
			fn := stdinNextFunc()
			if err := dispatchWithLimit(ctx, sqsClient, fn, opts.NumMessages); err != nil {
				return err
			}
		} else {
			if err := pollWithLimit(ctx, sqsClient, opts.NumMessages, defaultHandler, opts.Delete); err != nil {
				return err
			}
		}
		return nil
	}

	if opts.Concurrency <= 0 {
		opts.Concurrency = 10 * runtime.NumCPU()
	}

	g, ctx := errgroup.WithContext(ctx)

	if sendMode {
		fn := stdinNextFunc()
		for i := 0; i < opts.Concurrency; i++ {
			g.Go(func() error {
				return dispatch(ctx, sqsClient, fn)
			})
		}
	} else {
		fn := defaultHandler
		for i := 0; i < opts.Concurrency; i++ {
			g.Go(func() error {
				return poll(ctx, sqsClient, fn, opts.Delete)
			})
		}
	}

	return g.Wait()
}

func main() {
	opts := opts{}
	parseOpts(&opts)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, opts); err != nil {
		log.Fatalf("run() failed: %v", err)
	}
}
