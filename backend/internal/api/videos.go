package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/auth"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/model"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/storage"
	"github.com/go-chi/chi/v5"
)

// GetUserVideos gets a list view of all videos for a given user
func (h *HttpHandler) GetUserVideos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	objects, err := h.s3Client.ListObjects(ctx, userID)
	if err != nil {
		http.Error(w, "failed to list videos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Group object keys by video ID (the path segment right after userID/) so we pull only metadata.json
	response := make([]model.Video, 0, len(objects))
	for _, obj := range objects {
		parts := strings.Split(obj.Key, "/")
		if len(parts) != 3 { // userID/videoID/filename
			continue
		}

		videoID, fileName := parts[1], parts[2]

		// Only pull metadata.json, video itself is not needed for the list view
		if fileName == storage.MetadataFileName {
			data, err := h.s3Client.GetObject(ctx, obj.Key)
			if err != nil {
				http.Error(w, "failed to read video metadata: "+err.Error(), http.StatusInternalServerError)
				return
			}

			var meta model.VideoMetadata
			if err := json.Unmarshal(data, &meta); err != nil {
				http.Error(w, "corrupt video metadata: "+err.Error(), http.StatusInternalServerError)
				return
			}

			video := model.Video{
				ID:         videoID,
				Title:      meta.Title,
				Game:       meta.Game,
				Status:     meta.Status,
				UploadedAt: meta.UploadedAt,
				// Placeholder values; not tracked yet.
				DurationLabel: "12:00",
				ThumbnailHue:  55,
			}

			response = append(response, video)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetVideoDetails gets the details of a specific video for a given user, including the summary and analysis results.
func (h *HttpHandler) GetVideoDetails(w http.ResponseWriter, r *http.Request) {
	videoID := chi.URLParam(r, "videoID")
	ctx := r.Context()
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// TODO: Fetch more metadata like analysis results/summary.
	metaKey := fmt.Sprintf("%s/%s/%s", userID, videoID, storage.MetadataFileName)
	data, err := h.s3Client.GetObject(ctx, metaKey)
	if err != nil {
		http.Error(w, "video not found", http.StatusNotFound)
		return
	}

	// Get metadata about video such as title, game, status, and uploadedAt
	var meta model.VideoMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		http.Error(w, "corrupt video metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := model.Video{
		ID:            videoID,
		Title:         meta.Title,
		Game:          meta.Game,
		Status:        meta.Status,
		UploadedAt:    meta.UploadedAt,
		DurationLabel: "12:00",
		ThumbnailHue:  55,
	}

	// Get a presigned URL for the video file to enable playback without exposing the S3 bucket directly
	videoKey := fmt.Sprintf("%s/%s/%s", userID, videoID, storage.VideoFileName)
	if url, err := h.s3Client.GetPresignedURL(ctx, videoKey, 5*time.Minute); err == nil {
		response.VideoURL = &url
	}

	// Retrieve analysis if available
	analysisBytes, err := h.s3Client.GetObject(ctx, fmt.Sprintf("%s/%s/%s", userID, videoID, storage.AnalyzerFileName))

	// FIXME: error branching feels weird here
	if err == nil {
		var analysisResult model.AnalysisResult
		if err := json.Unmarshal(analysisBytes, &analysisResult); err == nil {
			response.Summary = &analysisResult.Summary
		} else {
			fmt.Printf("failed to unmarshall analysis result: %v", err)
		}
	} else if !strings.Contains(err.Error(), "not found") {
		http.Error(w, "failed to read analysis result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
