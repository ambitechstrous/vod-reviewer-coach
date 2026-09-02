package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/auth"
	"github.com/go-chi/chi/v5"
)

const (
	metadataFileName = "metadata.json"
	videoFileName    = "video.mp4"
)

// Video mirrors the frontend's Video type (frontend/src/types.ts) field for
// field, so the frontend can decode this response directly as a Video[]
// with no transformation.
type Video struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Game          string  `json:"game"`
	Status        string  `json:"status"`
	UploadedAt    string  `json:"uploadedAt"`
	DurationLabel string  `json:"durationLabel"`
	ThumbnailHue  int     `json:"thumbnailHue"`
	VideoURL      *string `json:"videoUrl,omitempty"`
	Summary       *string `json:"summary,omitempty"`
}

// VideoMetadata is the shape stored in each video's metadata.json sidecar
// object (userID/videoID/metadata.json).
type VideoMetadata struct {
	Title      string `json:"title"`
	Game       string `json:"game"`
	Status     string `json:"status"`
	UploadedAt string `json:"uploadedAt"`
}

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
	response := make([]Video, 0, len(objects))
	for _, obj := range objects {
		parts := strings.Split(obj.Key, "/")
		if len(parts) != 3 { // userID/videoID/filename
			continue
		}

		videoID, fileName := parts[1], parts[2]

		// Only pull metadata.json, video itself is not needed for the list view
		if fileName == metadataFileName {
			data, err := h.s3Client.GetObject(ctx, obj.Key)
			if err != nil {
				http.Error(w, "failed to read video metadata: "+err.Error(), http.StatusInternalServerError)
				return
			}

			var meta VideoMetadata
			if err := json.Unmarshal(data, &meta); err != nil {
				http.Error(w, "corrupt video metadata: "+err.Error(), http.StatusInternalServerError)
				return
			}

			video := Video{
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
	metaKey := fmt.Sprintf("%s/%s/%s", userID, videoID, metadataFileName)
	data, err := h.s3Client.GetObject(ctx, metaKey)
	if err != nil {
		http.Error(w, "video not found", http.StatusNotFound)
		return
	}

	var meta VideoMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		http.Error(w, "corrupt video metadata: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := Video{
		ID:            videoID,
		Title:         meta.Title,
		Game:          meta.Game,
		Status:        meta.Status,
		UploadedAt:    meta.UploadedAt,
		DurationLabel: "12:00",
		ThumbnailHue:  55,
	}

	videoKey := fmt.Sprintf("%s/%s/%s", userID, videoID, videoFileName)
	if url, err := h.s3Client.GetPresignedURL(ctx, videoKey, 5*time.Minute); err == nil {
		response.VideoURL = &url
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
