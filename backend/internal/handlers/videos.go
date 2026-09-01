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

// mockVideos is a placeholder for the frontend<->backend connection before
// GetUserVideos/GetVideoDetails scan S3 for a user's real videos. Every
// entry has a VideoURL regardless of Status — the raw file lands in S3 as
// soon as upload completes, independent of analysis, so it's always
// playable; only Summary depends on analysis being done.
var mockVideos = []Video{
	{
		ID:            "vod-1042",
		Title:         "Ranked Placements - Game 5",
		Game:          "Valorant",
		Status:        "processed",
		UploadedAt:    "2026-08-29T18:04:00Z",
		DurationLabel: "38:12",
		ThumbnailHue:  265,
		VideoURL:      new("https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4"),
		Summary:       new("You held angles well in the first half but over-peeked mid on 4 of your 6 deaths, usually right after a teammate died nearby. Comms were sparse after the 8-minute mark — calling rotations even when you're not sure would help your team react faster. Your utility usage on site executes was clean and well-timed."),
	},
	{
		ID:            "vod-1041",
		Title:         "Scrim vs. Nova Esports",
		Game:          "Valorant",
		Status:        "processed",
		UploadedAt:    "2026-08-28T21:30:00Z",
		DurationLabel: "41:05",
		ThumbnailHue:  190,
		VideoURL:      new("https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4"),
		Summary:       new("Strong entry fragging in the first 10 rounds, but your economy management slipped after two lost pistols — you force-bought three rounds in a row instead of saving. Positioning during retakes was consistently late; try rotating on the first callout instead of waiting for confirmation."),
	},
	{
		ID:            "vod-1040",
		Title:         "Solo Queue - Late Night Session",
		Game:          "Apex Legends",
		Status:        "analyzing",
		UploadedAt:    "2026-09-01T02:15:00Z",
		DurationLabel: "22:47",
		ThumbnailHue:  25,
		VideoURL:      new("https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4"),
	},
	{
		ID:            "vod-1039",
		Title:         "Duo Ranked with Kess",
		Game:          "Apex Legends",
		Status:        "analyzing",
		UploadedAt:    "2026-09-01T01:02:00Z",
		DurationLabel: "19:33",
		ThumbnailHue:  340,
		VideoURL:      new("https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4"),
	},
	{
		ID:            "vod-1038",
		Title:         "Tournament Qualifier - Map 2",
		Game:          "Valorant",
		Status:        "uploaded",
		UploadedAt:    "2026-09-01T03:40:00Z",
		DurationLabel: "35:20",
		ThumbnailHue:  145,
		VideoURL:      new("https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4"),
	},
	{
		ID:            "vod-1037",
		Title:         "Warmup Aim Rounds",
		Game:          "CS2",
		Status:        "uploaded",
		UploadedAt:    "2026-09-01T03:52:00Z",
		DurationLabel: "12:08",
		ThumbnailHue:  55,
		VideoURL:      new("https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4"),
	},
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
