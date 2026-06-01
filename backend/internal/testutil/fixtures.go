// Package testutil provides shared test helpers for the backend modules.
// fixtures.go contains builder functions for common test data.
// No build tag — compiles in both unit and integration builds.
package testutil

import (
	"time"

	"google.golang.org/api/idtoken"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
)

// ValidPayload builds an *idtoken.Payload with the given claims for use
// in unit tests that stub out the Google token validator.
func ValidPayload(sub, email, name, hd string) *idtoken.Payload {
	claims := map[string]interface{}{
		"sub":   sub,
		"email": email,
		"name":  name,
	}
	if hd != "" {
		claims["hd"] = hd
	}
	return &idtoken.Payload{
		Issuer:   "https://accounts.google.com",
		Audience: "test-client-id",
		Subject:  sub,
		IssuedAt: time.Now().Unix(),
		Expires:  time.Now().Add(time.Hour).Unix(),
		Claims:   claims,
	}
}

// RefreshTokenRow builds a *domain.RefreshToken with sensible defaults.
// Use the functional-options pattern to override individual fields:
//
//	rt := RefreshTokenRow(
//	    WithUserID("user-123"),
//	    WithRevokedAt(time.Now()),
//	)
func RefreshTokenRow(opts ...func(*domain.RefreshToken)) *domain.RefreshToken {
	rt := &domain.RefreshToken{
		ID:        "token-id-fixture",
		UserID:    "user-id-fixture",
		TokenHash: "hash-fixture",
		ExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	for _, opt := range opts {
		opt(rt)
	}
	return rt
}

// WithID sets the token ID.
func WithID(id string) func(*domain.RefreshToken) {
	return func(rt *domain.RefreshToken) { rt.ID = id }
}

// WithUserID sets the UserID field.
func WithUserID(userID string) func(*domain.RefreshToken) {
	return func(rt *domain.RefreshToken) { rt.UserID = userID }
}

// WithTokenHash sets the TokenHash field.
func WithTokenHash(hash string) func(*domain.RefreshToken) {
	return func(rt *domain.RefreshToken) { rt.TokenHash = hash }
}

// WithExpiresAt sets the ExpiresAt field.
func WithExpiresAt(t time.Time) func(*domain.RefreshToken) {
	return func(rt *domain.RefreshToken) { rt.ExpiresAt = t }
}

// WithRevokedAt sets the RevokedAt field to the given time pointer.
func WithRevokedAt(t time.Time) func(*domain.RefreshToken) {
	return func(rt *domain.RefreshToken) { rt.RevokedAt = &t }
}

// WithUsedAt sets the UsedAt field to the given time pointer.
func WithUsedAt(t time.Time) func(*domain.RefreshToken) {
	return func(rt *domain.RefreshToken) { rt.UsedAt = &t }
}
