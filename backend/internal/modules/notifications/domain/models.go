// Package domain contains the core domain types for the notifications module.
// This is a leaf module — it imports nothing from other domain modules.
package domain

import "time"

// Notification represents a single in-app notification for a user.
type Notification struct {
	ID       string    `gorm:"type:uuid;primaryKey"`
	UserID   string    `gorm:"column:user_id;not null"`
	Tipo     string    `gorm:"column:tipo;not null"`
	Titulo   string    `gorm:"column:titulo;not null"`
	Cuerpo   string    `gorm:"column:cuerpo;not null"`
	Leida    bool      `gorm:"column:leida;not null;default:false"`
	RefID    *string   `gorm:"column:ref_id"` // nullable uuid → pointer
	CreadoEn time.Time `gorm:"column:creado_en;not null"`
}

// TableName overrides the GORM default table name.
func (Notification) TableName() string { return "notification" }

// Tipo sentinels — single source for the CHECK constraint values.
const (
	TipoCursoAprobado      = "curso_aprobado"
	TipoCursoRechazado     = "curso_rechazado"
	TipoCertificadoEmitido = "certificado_emitido"
)
