// Package domain contains the core data model for the auth module.
// The RefreshToken struct is owned exclusively by this module — no other
// module should import it directly.
package domain

import "time"

// RefreshToken represents a persisted opaque refresh token.
// The token itself is never stored — only its SHA-256 hash (token_hash).
// Rotation chains are tracked via parent_id: when a token is refreshed,
// the old token gets revoked and the new token sets parent_id to the old ID.
type RefreshToken struct {
	ID        string     `gorm:"type:uuid;primaryKey"`
	UserID    string     `gorm:"type:uuid;not null;index"`
	TokenHash string     `gorm:"type:text;not null;uniqueIndex"`
	ParentID  *string    `gorm:"type:uuid"`
	ExpiresAt time.Time  `gorm:"type:timestamptz;not null"`
	RevokedAt *time.Time `gorm:"type:timestamptz"`
	UsedAt    *time.Time `gorm:"type:timestamptz"`
	UserAgent *string    `gorm:"type:text"`
	IP        *string    `gorm:"type:inet"`
	CreatedAt time.Time  `gorm:"type:timestamptz;default:now()"`
}

// TableName overrides GORM's default pluralized table name.
func (RefreshToken) TableName() string { return "refresh_token" }
