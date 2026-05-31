// Package dto defines the request and response shapes for the auth module.
// JSON tags follow OAuth 2.0 conventions for token fields (snake_case)
// and the frontend contract for request bodies (camelCase).
package dto

// GoogleLoginRequest is the body expected by POST /api/auth/google.
type GoogleLoginRequest struct {
	IDToken string `json:"idToken" binding:"required"`
}

// RefreshRequest is the body expected by POST /api/auth/refresh and
// POST /api/auth/logout.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}
