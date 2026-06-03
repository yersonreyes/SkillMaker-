// Package domain contains the GORM models for the courses module.
// Only Course is exercised in C2.1; the other 4 structs are schema-mapped for
// future use by C2.2–C2.4 (TableName + minimal tags only).
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
	ID          string    `gorm:"type:uuid;primaryKey"`
	CreadorID   string    `gorm:"type:uuid;not null"`
	Titulo      string    `gorm:"type:text;not null"`
	Descripcion string    `gorm:"type:text;not null;default:''"`
	Estado      Estado    `gorm:"type:text;not null;default:'borrador'"`
	CreatedAt   time.Time `gorm:"type:timestamptz;default:now()"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;default:now()"`
}

// TableName overrides GORM's default pluralisation to match migration 0003.
func (Course) TableName() string { return "course" }

// Section is schema-only in C2.1 (no endpoints). Tags map to migration 0003.
type Section struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	CourseID  string    `gorm:"type:uuid;not null"`
	Titulo    string    `gorm:"type:text;not null"`
	Orden     int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"type:timestamptz;default:now()"`
}

func (Section) TableName() string { return "section" }

// Video is schema-only in C2.1.
type Video struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	SectionID  string    `gorm:"type:uuid;not null"`
	Titulo     string    `gorm:"type:text;not null"`
	StorageKey string    `gorm:"type:text;not null"`
	DuracionS  int       `gorm:"column:duracion_s;not null;default:0"`
	Orden      int       `gorm:"not null;default:0"`
	CreatedAt  time.Time `gorm:"type:timestamptz;default:now()"`
}

func (Video) TableName() string { return "video" }

// Material is schema-only in C2.1.
type Material struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	CourseID   string    `gorm:"type:uuid;not null"`
	Titulo     string    `gorm:"type:text;not null"`
	StorageKey string    `gorm:"type:text;not null"`
	CreatedAt  time.Time `gorm:"type:timestamptz;default:now()"`
}

func (Material) TableName() string { return "material" }

// Enrollment is schema-only in C2.1.
// The composite UNIQUE(user_id, course_id) is enforced by migration 0003.
type Enrollment struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	UserID     string    `gorm:"type:uuid;not null"`
	CourseID   string    `gorm:"type:uuid;not null"`
	InscritoEn time.Time `gorm:"column:inscrito_en;type:timestamptz;default:now()"`
}

func (Enrollment) TableName() string { return "enrollment" }
