package handlers

import (
	"encoding/json"
	"net/http"
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

// mockVideos mirrors frontend/src/data/mockVideos.ts exactly, so the list
// view renders identically whether it reads local mock data or this
// endpoint. This is a placeholder for wiring up the frontend<->backend
// connection before GetUserVideos scans S3 for a user's real videos.
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
	},
	{
		ID:            "vod-1039",
		Title:         "Duo Ranked with Kess",
		Game:          "Apex Legends",
		Status:        "analyzing",
		UploadedAt:    "2026-09-01T01:02:00Z",
		DurationLabel: "19:33",
		ThumbnailHue:  340,
	},
	{
		ID:            "vod-1038",
		Title:         "Tournament Qualifier - Map 2",
		Game:          "Valorant",
		Status:        "uploaded",
		UploadedAt:    "2026-09-01T03:40:00Z",
		DurationLabel: "35:20",
		ThumbnailHue:  145,
	},
	{
		ID:            "vod-1037",
		Title:         "Warmup Aim Rounds",
		Game:          "CS2",
		Status:        "uploaded",
		UploadedAt:    "2026-09-01T03:52:00Z",
		DurationLabel: "12:08",
		ThumbnailHue:  55,
	},
}

// GetUserVideos gets a list view of all videos for a given user
func (h *HttpHandler) GetUserVideos(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mockVideos)
}
