package model

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

// VideoMetadata is the shape stored in each video's metadata.json sidecar object (userID/videoID/metadata.json).
type VideoMetadata struct {
	Title      string `json:"title"`
	Game       string `json:"game"`
	Status     string `json:"status"`
	UploadedAt string `json:"uploadedAt"`
}
