// Package auth issues and verifies signed session tokens, and carries the
// authenticated user's ID through a request's context.
//
// There is no password or identity verification yet — IssueToken will sign
// a token for whatever email it's given. What it guarantees is narrower but
// still load-bearing: once issued, a token's subject can't be altered or
// forged by a client, so every later request carrying that token is
// provably from whoever holds it, and handlers can trust the user ID they
// read out of a verified token instead of a client-supplied field.
package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrMissingSecret = errors.New("JWT_SECRET is not set")

const tokenTTL = 24 * time.Hour

type contextKey string

const userIDContextKey contextKey = "userID"

func secret() ([]byte, error) {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return nil, ErrMissingSecret
	}
	return []byte(s), nil
}

// IssueToken mints a signed session token identifying userID, valid for 24 hours.
func IssueToken(userID string) (string, error) {
	key, err := secret()
	if err != nil {
		return "", err
	}

	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// VerifyToken validates a signed token's signature and expiry, and returns
// the user ID it was issued for.
func VerifyToken(tokenString string) (string, error) {
	key, err := secret()
	if err != nil {
		return "", err
	}

	var claims jwt.RegisteredClaims
	_, err = jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid token: %w", err)
	}

	return claims.Subject, nil
}

// ContextWithUserID returns a new context carrying the authenticated user's ID.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

// UserIDFromContext returns the authenticated user's ID set by the auth
// middleware, and false if the context carries none.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDContextKey).(string)
	return id, ok
}
