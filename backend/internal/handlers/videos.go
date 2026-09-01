package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/auth"
	"github.com/go-chi/chi/v5"
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

// GetUserVideos gets a list view of all videos for a given user
func (h *HttpHandler) GetUserVideos(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	videos, err := h.s3Client.ListVideos(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to list videos: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]Video, 0, len(videos))
	for _, v := range videos {
		name := strings.Split(v.Key, "/")[1]
		response = append(response, Video{
			ID:            name,
			Title:         name,            // Placeholder; in a real app, you'd extract a title from metadata or a database
			Game:          "Rocket League", // Placeholder; in a real app, you'd extract the game from metadata or a database
			Status:        "uploaded",      // Placeholder; in a real app, you'd determine status based on analysis results
			UploadedAt:    v.LastModified,
			DurationLabel: "12:00", // Placeholder; in a real app, you'd extract duration from metadata or a database
			ThumbnailHue:  55,      // Placeholder; in a real app, you'd extract thumbnail hue from metadata or a database
			VideoURL:      &v.URL,  // FIXME: Don't need a video URL here, only the thumbnail
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetVideoDetails gets the details of a specific video for a given user, including the summary and analysis results.
func (h *HttpHandler) GetVideoDetails(w http.ResponseWriter, r *http.Request) {
	videoName := chi.URLParam(r, "videoName")
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// TODO: Fetch more metadata like video game and analysis results.
	prefix := fmt.Sprintf("%s/%s", userID, videoName)
	url, err := h.s3Client.GetPresignedURL(r.Context(), prefix, 5*time.Minute)
	if err != nil {
		fmt.Printf("failed to get presigned URL for video %s: %v\n", prefix, err)
		http.Error(w, "video not found", http.StatusNotFound)
		return
	}

	response := Video{
		ID:       videoName,
		VideoURL: &url,
		// Placeholder values; in a real app, you'd fetch these from metadata or a database
		Title:         videoName,
		Game:          "Rocket League",
		Status:        "uploaded",
		UploadedAt:    "2026-09-01T03:52:00Z",
		DurationLabel: "12:00",
		ThumbnailHue:  55,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
