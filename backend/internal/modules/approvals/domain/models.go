// Package domain contains the GORM models for the approvals module.
// Mirrors the courses/evaluations domain pattern exactly.
package domain

import "time"

// Approval represents an admin decision (approve or reject) on a course submission.
// Maps to the approval table created by migration 0008.
// The resultado column has a DB CHECK constraint: IN ('aprobado', 'rechazado').
type Approval struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	CourseID   string    `gorm:"column:course_id;type:uuid;not null"`
	AdminID    string    `gorm:"column:admin_id;type:uuid;not null"`
	Resultado  string    `gorm:"type:text;not null"`
	Comentario string    `gorm:"type:text;not null;default:''"`
	ResueltoEn time.Time `gorm:"column:resuelto_en;type:timestamptz;default:now()"`
}

// TableName overrides GORM's default pluralisation to match migration 0008.
func (Approval) TableName() string { return "approval" }
