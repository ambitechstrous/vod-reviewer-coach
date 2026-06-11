package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/josemvargas94/vod-reviewer-coach/internal/storage"
)

type SamplerHandler struct {
	s3Client *storage.S3Client
}

type SamplerEvent struct {
	VideoID string `json:"video_id"`
}

func NewSamplerHandler(ctx context.Context) (*SamplerHandler, error) {
	s3Client, err := storage.NewS3Client(ctx, "user-vods")
	if err != nil {
		return nil, err
	}
	return &SamplerHandler{s3Client: s3Client}, nil
}

func (h *SamplerHandler) Run(ctx context.Context, event Event) error {
	var samplerEvent SamplerEvent
	if err := json.Unmarshal(event.Payload, &samplerEvent); err != nil {
		return err
	}

	fmt.Println("Got video ID:", samplerEvent.VideoID)

	video, err := h.s3Client.GetVideo(ctx, samplerEvent.VideoID)
	if err != nil {
		return err
	}
	defer video.Body.Close()

	return nil
}
