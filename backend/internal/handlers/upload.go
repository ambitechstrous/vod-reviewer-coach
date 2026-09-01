package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/storage"
)

const (
	// uploadPartSize is the size of every part except the last one. S3
	// requires parts to be at least 5 MiB (except the final part), and this
	// is comfortably above that floor.
	uploadPartSize = 8 * 1024 * 1024 // 8 MiB
	// maxUploadParts is S3's hard limit on parts per multipart upload.
	maxUploadParts = 10000
	// uploadPartURLTTL is how long each presigned part URL stays valid.
	uploadPartURLTTL = 15 * time.Minute
)

// UploadPart contains the presigned URL for a specific part of a multipart upload.
type UploadPart struct {
	PartNumber int32  `json:"part_number"`
	URL        string `json:"url"`
}

// CompletedUploadPart contains the part number and ETag (i.e. checksum) for a part that has been uploaded to S3.
type CompletedUploadPart struct {
	PartNumber int32  `json:"part_number"`
	ETag       string `json:"etag"`
}

// CreateUploadSessionRequest is the request body for starting a multipart upload.
type CreateUploadSessionRequest struct {
	VideoName     string `json:"video_name"`
	ContentType   string `json:"content_type"`
	FileSizeBytes int64  `json:"file_size"`
}

// CompleteUploadRequest is the request body for completing a multipart upload.
type CompleteUploadRequest struct {
	VideoName string                `json:"video_name"`
	UploadID  string                `json:"upload_id"`
	Parts     []CompletedUploadPart `json:"parts"`
}

// AbortUploadRequest is the request body for aborting a multipart upload.
type AbortUploadRequest struct {
	VideoName string `json:"video_name"`
	UploadID  string `json:"upload_id"`
}

// CreateUploadSessionResponse is the response body for starting a multipart upload.
type CreateUploadSessionResponse struct {
	Key      string       `json:"key"`
	UploadID string       `json:"upload_id"`
	PartSize int64        `json:"part_size"`
	Parts    []UploadPart `json:"parts"`
}

// CompleteUploadResponse is the response body for completing a multipart upload.
type CompleteUploadResponse struct {
	Key    string `json:"key"`
	Status string `json:"status"`
}

// CreateUploadSession starts a multipart upload for a video and returns a presigned URL for every part the client will need to PUT directly to S3.
func (h *HttpHandler) CreateUploadSession(w http.ResponseWriter, r *http.Request) {
	var req CreateUploadSessionRequest

	// Read request body to get video name, content type, and file size
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.VideoName == "" || req.ContentType == "" || req.FileSizeBytes <= 0 {
		http.Error(w, "video_name, content_type, and a positive file_size are required", http.StatusBadRequest)
		return
	}

	// File size determines how many parts we need to upload. S3 requires at least one part, and we also enforce a maximum number of parts.
	partCount := max(int32((req.FileSizeBytes+uploadPartSize-1)/uploadPartSize), 1)
	if partCount > maxUploadParts {
		http.Error(w, "file_size is too large for a multipart upload", http.StatusBadRequest)
		return
	}

	// Create multi-part upload session in S3 for this video, which gives us an upload ID.
	ctx := r.Context()
	uploadID, err := h.s3Client.CreateMultipartUpload(ctx, req.VideoName, req.ContentType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate presigned URLs for each part of the upload, which the client will use to upload directly to S3.
	parts := make([]UploadPart, partCount)
	for i := int32(0); i < partCount; i++ {
		partNumber := i + 1
		url, err := h.s3Client.PresignUploadPart(ctx, req.VideoName, uploadID, partNumber, uploadPartURLTTL)
		if err != nil {
			_ = h.s3Client.AbortMultipartUpload(ctx, req.VideoName, uploadID)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		parts[i] = UploadPart{PartNumber: partNumber, URL: url}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CreateUploadSessionResponse{
		Key:      req.VideoName,
		UploadID: uploadID,
		PartSize: uploadPartSize,
		Parts:    parts,
	})
}

// CompleteUpload finalizes a multipart upload once the client has uploaded every part and collected each part's ETag.
func (h *HttpHandler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	var req CompleteUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.VideoName == "" || req.UploadID == "" || len(req.Parts) == 0 {
		http.Error(w, "video_name, upload_id, and at least one part are required", http.StatusBadRequest)
		return
	}

	parts := make([]storage.CompletedPart, len(req.Parts))
	for i, p := range req.Parts {
		parts[i] = storage.CompletedPart{PartNumber: p.PartNumber, ETag: p.ETag}
	}

	if err := h.s3Client.CompleteMultipartUpload(r.Context(), req.VideoName, req.UploadID, parts); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CompleteUploadResponse{
		Key:    req.VideoName,
		Status: "uploaded",
	})
}

// AbortUpload cancels an in-progress multipart upload, e.g. when the client
// gives up partway through.
func (h *HttpHandler) AbortUpload(w http.ResponseWriter, r *http.Request) {
	var req AbortUploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.VideoName == "" || req.UploadID == "" {
		http.Error(w, "video_name and upload_id are required", http.StatusBadRequest)
		return
	}

	if err := h.s3Client.AbortMultipartUpload(r.Context(), req.VideoName, req.UploadID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
