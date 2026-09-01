package handlers

import (
	"net/http"
	"strings"

	"github.com/ambitechstrous/vod-reviewer-coach/internal/auth"
)

// requireAuth verifies the request's "Authorization: Bearer <token>" header
// and, on success, injects the authenticated user ID into the request
// context for downstream handlers to read via auth.UserIDFromContext.
func requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		userID, err := auth.VerifyToken(token)
		if err != nil {
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r.WithContext(auth.ContextWithUserID(r.Context(), userID)))
	})
}
