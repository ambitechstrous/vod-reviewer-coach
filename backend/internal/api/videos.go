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

	// Get metadata for each video and build a response list. Each video is stored in S3 under a prefix of userID/videoID/, and the metadata is stored in a file named metadata.json.
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

	// Get the analysis summary if available (i.e. video has been analyzed already)
	if response.Status == string(storage.AnalyzerStatusAnalyzed) {
		analysisKey := fmt.Sprintf("%s/%s/%s", userID, videoID, storage.AnalyzerFileName)
		analysisData, err := h.s3Client.GetObject(ctx, analysisKey)

		// If the analysis result is not found, but the status is analyzed, this is a legitimate error, either with S3 or the data in it.
		if err != nil {
			http.Error(w, "failed to read analysis result: "+err.Error(), http.StatusInternalServerError)
		}

		// Unmarshall analysis result and pull the summary into the response object
		var analysisResult model.AnalysisResult
		if err := json.Unmarshal(analysisData, &analysisResult); err != nil {
			http.Error(w, "failed to unmarshal analysis result: "+err.Error(), http.StatusInternalServerError)
		}

		response.Summary = &analysisResult.Summary
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
