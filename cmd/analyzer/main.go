package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/handlers"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "analyzer: failed to load .env file: %v\n", err)
		os.Exit(1)
	}

	httpHandler, err := handlers.NewHttpHandler(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "analyzer: failed to create HTTP handler: %v\n", err)
		os.Exit(1)
	}

	srv := httpHandler.SetUpRouter()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go httpHandler.Start()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "analyzer shutdown: %v\n", err)
	}
}
