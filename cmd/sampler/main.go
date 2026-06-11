package main

import (
	"context"

	"github.com/josemvargas94/vod-reviewer-coach/internal/handlers"
)

func main() {
	handlers.RunHandler(func(ctx context.Context) (handlers.Handler, error) {
		return handlers.NewSamplerHandler(ctx)
	})
}
