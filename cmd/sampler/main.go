package main

import (
	"context"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/handlers"
)

func main() {
	handlers.RunHandler(func(ctx context.Context) (handlers.Handler, error) {
		return handlers.NewSamplerHandler(ctx)
	})
}
