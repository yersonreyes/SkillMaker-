package dto

import "time"

// LoginResponse is the body returned by POST /api/auth/google and
// POST /api/auth/refresh. Field names follow OAuth 2.0 token response
// conventions (snake_case) per spec SCAFFOLD-RF-03/04.
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         UserDTO   `json:"user"`
}

// UserDTO is the caller-facing representation of the authenticated user
// included in every LoginResponse.
type UserDTO struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Nombre string   `json:"nombre"`
	Roles  []string `json:"roles"`
}
