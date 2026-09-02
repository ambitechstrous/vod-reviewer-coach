package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/handlers"
	"github.com/joho/godotenv"
)

// TODO: Migrate analyzer to a Lambda handler and point this to that
func main() {
	handlers.RunHandler(func(ctx context.Context) (handlers.Handler, error) {
		err := godotenv.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "analyzer: failed to load .env file: %v\n", err)
			os.Exit(1)
		}

		return handlers.NewAnalyzerHandler(ctx)
	})
}
