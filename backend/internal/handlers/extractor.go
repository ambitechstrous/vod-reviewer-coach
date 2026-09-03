package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/storage"
)

type ExtractorHandler struct {
	s3Client *storage.S3Client
}

type ExtractorEvent struct {
	MatchID string `json:"match_id"`
}

func NewExtractorHandler(ctx context.Context) (*ExtractorHandler, error) {
	s3Client, err := storage.NewS3Client(ctx)
	if err != nil {
		return nil, err
	}
	return &ExtractorHandler{s3Client: s3Client}, nil
}

func (h *ExtractorHandler) Run(ctx context.Context, event Event) error {
	var extractorEvent ExtractorEvent
	if err := json.Unmarshal(event.Payload, &extractorEvent); err != nil {
		return err
	}

	fmt.Println("Got match ID:", extractorEvent.MatchID)

	return nil
}
