package main

import (
	"context"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/handlers"
)

// TODO: Migrate analyzer to a Lambda handler and point this to that
func main() {
	handlers.RunHandler(func(ctx context.Context) (handlers.Handler, error) {
		return handlers.NewExtractorHandler(ctx)
	})
}
