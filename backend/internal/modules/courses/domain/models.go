// Package domain contains the GORM models for the courses module.
// Course is fully exercised since C2.1. Section and Video are activated in C2.2.
// Material and Enrollment remain schema-only (C2.3/C2.4 will add endpoints).
package domain

import "time"

// Estado is a typed string representing the lifecycle state of a course.
// The DB enforces this via a CHECK constraint; this type provides defense-in-depth
// at the Go layer.
type Estado string

const (
	EstadoBorrador   Estado = "borrador"
	EstadoEnRevision Estado = "en_revision"
	EstadoAprobado   Estado = "aprobado"
	EstadoRechazado  Estado = "rechazado"
)

// Valid reports whether e is one of the four recognised course states.
func (e Estado) Valid() bool {
	switch e {
	case EstadoBorrador, EstadoEnRevision, EstadoAprobado, EstadoRechazado:
		return true
	}
	return false
}

// Course is the aggregate root for the courses domain.
// creador_id references "user"(id) ON DELETE RESTRICT (migration 0003).
type Course struct {
	ID          string     `gorm:"type:uuid;primaryKey"`
	CreadorID   string     `gorm:"type:uuid;not null"`
	Titulo      string     `gorm:"type:text;not null"`
	Descripcion string     `gorm:"type:text;not null;default:''"`
	Estado      Estado     `gorm:"type:text;not null;default:'borrador'"`
	PublicadoEn *time.Time `gorm:"column:publicado_en;type:timestamptz"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;default:now()"`
	UpdatedAt   time.Time  `gorm:"type:timestamptz;default:now()"`
}

// TableName overrides GORM's default pluralisation to match migration 0003.
func (Course) TableName() string { return "course" }

// Section maps to the section table (migration 0003). Activated in C2.2.
type Section struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	CourseID  string    `gorm:"type:uuid;not null"`
	Titulo    string    `gorm:"type:text;not null"`
	Orden     int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:now()"`
}

func (Section) TableName() string { return "section" }

// Video maps to the video table (migration 0004 corrects schema: url+proveedor replace storage_key).
type Video struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	SectionID string    `gorm:"type:uuid;not null"`
	Titulo    string    `gorm:"type:text;not null"`
	URL       string    `gorm:"column:url;type:text;not null"`
	Proveedor string    `gorm:"column:proveedor;type:text;not null"`
	DuracionS int       `gorm:"column:duracion_s;not null;default:0"`
	Orden     int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:now()"`
}

func (Video) TableName() string { return "video" }

// Material represents an uploaded file attachment for a course (C2.3).
// MimeType and TamanoBytes are added by migration 0005.
// The wire label for Titulo is "nombre" (D1: titulo is the persisted contract).
type Material struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	CourseID    string    `gorm:"type:uuid;not null"`
	Titulo      string    `gorm:"type:text;not null"`
	StorageKey  string    `gorm:"type:text;not null"`
	MimeType    string    `gorm:"column:mime_type;type:text;not null"`
	TamanoBytes int64     `gorm:"column:tamano_bytes;not null"`
	CreatedAt   time.Time `gorm:"type:timestamptz;default:now()"`
}

func (Material) TableName() string { return "material" }

// Enrollment maps to the enrollment table (migration 0003 schema, completado added in 0009).
// The composite UNIQUE(user_id, course_id) is enforced by migration 0003.
// Completado is set to true when the student passes an evaluation (EnrollmentCompleter seam, C2.4).
type Enrollment struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	UserID     string    `gorm:"type:uuid;not null"`
	CourseID   string    `gorm:"type:uuid;not null"`
	InscritoEn time.Time `gorm:"column:inscrito_en;type:timestamptz;default:now()"`
	Completado bool      `gorm:"column:completado;not null;default:false" json:"completado"`
}

func (Enrollment) TableName() string { return "enrollment" }
