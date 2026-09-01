package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-lambda-go/lambda"
)

// Event is the generic input. Payload holds the raw trigger body
// (EventBridge detail, SQS message, S3 notification, etc.) and can be
// unmarshaled into a concrete type once the trigger source is known.
type Event struct {
	Payload json.RawMessage
}

type Handler interface {
	// Run is the entry point for all business logic. It is called by main
	// regardless of the runtime (Lambda, cron binary, etc.) so this package
	// has no dependency on any runtime adapter.
	Run(ctx context.Context, event Event) error
}

// RunHandler handles runtime detection and dispatches to the provided handler.
func RunHandler(newHandlerFn func(ctx context.Context) (Handler, error)) {
	// Lambda sets this variable automatically; use it to detect the runtime.
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(func(ctx context.Context, raw json.RawMessage) error {
			handler, err := newHandlerFn(ctx)
			if err != nil {
				return err
			}
			return handler.Run(ctx, Event{Payload: raw})
		})
		return
	}

	// Standalone: honour OS signals for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Accept an optional JSON payload on stdin so any external trigger
	// (CI job, shell script, another service) can pass structured data.
	var payload json.RawMessage
	if data, err := io.ReadAll(os.Stdin); err == nil && len(data) > 0 {
		payload = data
	}

	handler, err := newHandlerFn(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "handler init: %v\n", err)
		os.Exit(1)
	}

	if err := handler.Run(ctx, Event{Payload: payload}); err != nil {
		fmt.Fprintf(os.Stderr, "handler run: %v\n", err)
		os.Exit(1)
	}
}
