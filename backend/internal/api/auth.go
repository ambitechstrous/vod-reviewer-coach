package api

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

type VerifyResponse struct {
	Email string `json:"email"`
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

	// TODO: Check an actual database of email/passwords
	token, err := auth.IssueToken(req.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: token})
}

// Verify confirms the caller's token is still valid. It sits behind
// requireAuth, so simply reaching this handler (rather than getting a 401
// from the middleware) is the check — this just echoes back who the token
// identifies. Frontends use it on load to detect a stale/invalid stored
// session (e.g. one saved before this endpoint existed) and force a
// re-login instead of letting every subsequent request fail with 401.
func (h *HttpHandler) Verify(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(VerifyResponse{Email: userID})
}
