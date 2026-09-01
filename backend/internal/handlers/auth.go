package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/auth"
)

type LoginRequest struct {
	Email string `json:"email"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

// Login issues a session token for the given email. There's no password or
// account lookup yet, so this doesn't prove the caller owns that email — it
// establishes a stable, server-signed identity that every later request can
// be authenticated against, in place of trusting a client-supplied user ID.
func (h *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	token, err := auth.IssueToken(req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}
