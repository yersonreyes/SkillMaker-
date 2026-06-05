// Package domain contains the GORM models for the certificates module.
// No sentinels here — sentinels live at service level (courses precedent).
package domain

import "time"

// Certificate is the GORM model for the certificate table.
// A certificate is issued once per (user, course) pair — idempotent by UNIQUE constraint.
type Certificate struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	UserID     string    `gorm:"column:user_id;not null"`
	CourseID   string    `gorm:"column:course_id;not null"`
	Codigo     string    `gorm:"column:codigo;not null;uniqueIndex"`
	StorageKey string    `gorm:"column:storage_key;not null"`
	EmitidoEn  time.Time `gorm:"column:emitido_en;not null"`
}

// TableName overrides the GORM default to use the exact SQL table name.
func (Certificate) TableName() string { return "certificate" }

// Badge is the GORM model for the badge table (seed rows from migration 0010).
type Badge struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	Nombre      string    `gorm:"column:nombre;not null;uniqueIndex"`
	Descripcion string    `gorm:"column:descripcion;not null"`
	Umbral      int       `gorm:"column:umbral;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null"`
}

// TableName overrides the GORM default.
func (Badge) TableName() string { return "badge" }

// UserBadge is the GORM model for the user_badge join table.
// Primary key is composite (user_id, badge_id) — mirrors user_role from migration 0001.
type UserBadge struct {
	UserID     string    `gorm:"column:user_id;primaryKey"`
	BadgeID    string    `gorm:"column:badge_id;primaryKey"`
	OtorgadoEn time.Time `gorm:"column:otorgado_en;not null"`
}

// TableName overrides the GORM default.
func (UserBadge) TableName() string { return "user_badge" }
