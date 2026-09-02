package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/client"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/model"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/storage"
)

type AnalyzerHandler struct {
	geminiClient *client.GeminiClient
	s3Client     *storage.S3Client
}

type AnalyzerEvent struct {
	UserID  string `json:"user_id"`   // ID of the user who uploaded the video
	VideoID string `json:"video_key"` // ID of the video to analyze (used alongside userID as the S3 key prefix)
	Prompt  string `json:"prompt"`
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

	// userID and videoID are used as the key prefix in S3, in the format "userID/videoID/filename"
	keyPrefix := fmt.Sprintf("%s/%s", analyzerEvent.UserID, analyzerEvent.VideoID)
	metadataFileKey := fmt.Sprintf("%s/%s", keyPrefix, storage.MetadataFileName)

	// Update status in metadata.json to indicate run has started.
	if err := h.s3Client.UpdateAnalyzerStatus(ctx, metadataFileKey, storage.AnalyzerStatusAnalyzing); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Retrieve video to analyze from S3 using the userID and videoID as the key prefix.
	videoFile, err := h.s3Client.GetVideo(ctx, fmt.Sprintf("%s/%s", keyPrefix, storage.VideoFileName))
	if err != nil {
		return fmt.Errorf("failed to get video file from S3: %w", err)
	}
	defer videoFile.Body.Close()

	// Call the Gemini API to analyze the video with the provided prompt.
	resp, err := h.geminiClient.AnalyzeVideo(ctx, videoFile.Body, videoFile.ContentType, analyzerEvent.Prompt)
	if err != nil {
		return err
	}

	analysisResult := model.AnalysisResult{
		Summary: resp,
	}

	// Upload the analysis result to S3 as a JSON file under the same userID/videoID prefix.
	if err = h.s3Client.PutJSON(ctx, fmt.Sprintf("%s/%s", keyPrefix, storage.AnalyzerFileName), analysisResult); err != nil {
		return fmt.Errorf("failed to upload analysis result: %w", err)
	}

	// Update the status in metadata.json to "analyzed" after successful analysis and upload.
	if err := h.s3Client.UpdateAnalyzerStatus(ctx, fmt.Sprintf("%s/%s", keyPrefix, storage.MetadataFileName), storage.AnalyzerStatusAnalyzed); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	log.Printf("Analysis result uploaded to S3 at %s/%s\n", keyPrefix, storage.MetadataFileName)
	return nil
}
