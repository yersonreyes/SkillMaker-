package dto

import (
	"time"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/domain"
)

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

// SessionResponse is one active refresh-token session for the caller.
// ip and userAgent are informational/forensic only — never used to gate auth.
// Fields are omitempty: absent ip/userAgent surfaces as null in JSON.
//
// @Description Active session for the authenticated user.
type SessionResponse struct {
	// ID is the session (refresh token row) UUID.
	ID string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	// IP is the client IP captured at token issuance. May be absent.
	IP *string `json:"ip,omitempty" example:"10.0.0.1"`
	// UserAgent is the HTTP User-Agent captured at token issuance. May be absent.
	UserAgent *string `json:"userAgent,omitempty" example:"Mozilla/5.0"`
	// CreatedAt is the token row creation timestamp (RFC3339).
	CreatedAt time.Time `json:"createdAt"`
	// ExpiresAt is the token expiry timestamp (RFC3339).
	ExpiresAt time.Time `json:"expiresAt"`
	// UsedAt is the last rotation marker; null if this session has never been rotated.
	UsedAt *time.Time `json:"usedAt,omitempty"`
}

// NewSessionResponse maps a domain.RefreshToken to a SessionResponse DTO.
// NEVER exposes token_hash or parent_id.
func NewSessionResponse(rt *domain.RefreshToken) SessionResponse {
	return SessionResponse{
		ID:        rt.ID,
		IP:        rt.IP,
		UserAgent: rt.UserAgent,
		CreatedAt: rt.CreatedAt,
		ExpiresAt: rt.ExpiresAt,
		UsedAt:    rt.UsedAt,
	}
}
