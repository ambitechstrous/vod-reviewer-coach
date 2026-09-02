package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/client"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/storage"
)

type AnalyzerHandler struct {
	geminiClient *client.GeminiClient
	s3Client     *storage.S3Client
}

type AnalyzerEvent struct {
	VideoKey string `json:"video_key"` // S3 path (prod)
	FilePath string `json:"file_path"` // local path (testing)
	Prompt   string `json:"prompt"`
}

func NewAnalyzerHandler(ctx context.Context) (*AnalyzerHandler, error) {
	geminiClient, err := client.NewGeminiClient(ctx)
	if err != nil {
		return nil, err
	}

	s3Client, err := storage.NewS3Client(ctx, "user-vods")
	if err != nil {
		return nil, err
	}
	return &AnalyzerHandler{
		geminiClient: geminiClient,
		s3Client:     s3Client,
	}, nil
}

func (h *AnalyzerHandler) Run(ctx context.Context, event Event) error {
	var analyzerEvent AnalyzerEvent
	if err := json.Unmarshal(event.Payload, &analyzerEvent); err != nil {
		return err
	}

	log.Printf("Received analyze request: %+v\n", analyzerEvent)
	reader, err := os.Open(analyzerEvent.FilePath)
	if err != nil {
		return err
	}
	defer reader.Close()

	mimeType := mime.TypeByExtension(filepath.Ext(analyzerEvent.FilePath))
	if mimeType == "" {
		return fmt.Errorf("could not determine MIME type")
	}

	resp, err := h.geminiClient.AnalyzeVideo(ctx, reader, mimeType, analyzerEvent.Prompt)
	if err != nil {
		return err
	}

	log.Printf("Analysis result: %s\n", resp)
	return nil
}
