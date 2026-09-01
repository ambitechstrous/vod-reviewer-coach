package handlers

import (
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

type AnalyzeRequest struct {
	VideoKey string `json:"video_key"` // S3 path (prod)
	FilePath string `json:"file_path"` // local path (testing)
	Prompt   string `json:"prompt"`
}

func (h *HttpHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Received analyze request: %+v\n", req)
	reader, err := os.Open(req.FilePath)
	if err != nil {
		http.Error(w, "failed to open file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	mimeType := mime.TypeByExtension(filepath.Ext(req.FilePath))
	if mimeType == "" {
		http.Error(w, "could not determine MIME type", http.StatusBadRequest)
		return
	}

	resp, err := h.geminiClient.AnalyzeVideo(r.Context(), reader, mimeType, req.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"response": resp})
}
