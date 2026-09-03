package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/auth"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/model"
	"github.com/ambitechstrous/vod-reviewer-coach/internal/storage"
)

// ChatRequest is the request body for asking the AI coach a question across
// all of the authenticated user's analyzed videos.
type ChatRequest struct {
	Question string `json:"question"`
}

// ChatResponse is the response body containing the AI coach's answer.
type ChatResponse struct {
	Answer string `json:"answer"`
}

// Chat answers a question about patterns across every video the
// authenticated user has had analyzed so far.
func (h *HttpHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Question == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	summaries, err := h.videoSummaries(ctx, userID)
	if err != nil {
		http.Error(w, "failed to gather video summaries: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(summaries) == 0 {
		http.Error(w, "no analyzed videos found for this user yet", http.StatusNotFound)
		return
	}

	answer, err := h.geminiClient.AnalyzeSummaries(ctx, summaries, req.Question)
	if err != nil {
		http.Error(w, "failed to analyze summaries: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ChatResponse{Answer: answer})
}

// videoSummaries collects the analysis summary (analysis.json) for every one
// of userID's videos that has been analyzed. Videos with no analysis yet are
// silently skipped.
func (h *HttpHandler) videoSummaries(ctx context.Context, userID string) ([]string, error) {
	objects, err := h.s3Client.ListObjects(ctx, userID)
	if err != nil {
		return nil, err
	}

	var summaries []string
	for _, obj := range objects {
		parts := strings.Split(obj.Key, "/")
		if len(parts) != 3 || parts[2] != storage.AnalyzerFileName { // userID/videoID/analysis.json
			continue
		}

		data, err := h.s3Client.GetObject(ctx, obj.Key)
		if err != nil {
			return nil, fmt.Errorf("get analysis %q: %w", obj.Key, err)
		}

		var result model.AnalysisResult
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, fmt.Errorf("unmarshal analysis %q: %w", obj.Key, err)
		}

		summaries = append(summaries, result.Summary)
	}

	return summaries, nil
}
