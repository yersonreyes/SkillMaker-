// Package domain contains the GORM models for the evaluations module.
// These structs are only used inside the evaluations module; handlers and
// cross-module callers use service-layer read models instead.
package domain

import "time"

// TipoPregunta is the question-type discriminator.
// Only two values are valid: opcion_multiple and verdadero_falso.
type TipoPregunta string

const (
	// TipoOpcionMultiple is a multiple-choice question with user-defined options.
	TipoOpcionMultiple TipoPregunta = "opcion_multiple"
	// TipoVerdaderoFalso is a true/false question with two auto-created options.
	TipoVerdaderoFalso TipoPregunta = "verdadero_falso"
)

// Valid reports whether the tipo value is one of the two defined constants.
func (t TipoPregunta) Valid() bool {
	return t == TipoOpcionMultiple || t == TipoVerdaderoFalso
}

// Evaluation is the GORM model for the evaluation table.
// One evaluation maps to exactly one course (UNIQUE course_id).
type Evaluation struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	CourseID    string    `gorm:"column:course_id;type:uuid;not null"`
	NotaMinima  int       `gorm:"column:nota_minima;not null;default:70"`
	IntentosMax int       `gorm:"column:intentos_max;not null;default:0"`
	CreatedAt   time.Time `gorm:"type:timestamptz;default:now()"`
}

// TableName overrides the GORM default ("evaluations") with the actual SQL table name.
func (Evaluation) TableName() string { return "evaluation" }

// Question is the GORM model for the question table.
type Question struct {
	ID           string       `gorm:"type:uuid;primaryKey"`
	EvaluationID string       `gorm:"column:evaluation_id;type:uuid;not null"`
	Enunciado    string       `gorm:"type:text;not null"`
	Tipo         TipoPregunta `gorm:"type:text;not null"`
	Puntaje      int          `gorm:"not null;default:1"`
	Orden        int          `gorm:"not null;default:0"`
	CreatedAt    time.Time    `gorm:"type:timestamptz;default:now()"`
}

// TableName overrides the GORM default with the actual SQL table name.
func (Question) TableName() string { return "question" }

// Option is the GORM model for the question_option table.
// TableName() returns "question_option" to match the migration.
type Option struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	QuestionID string    `gorm:"column:question_id;type:uuid;not null"`
	Texto      string    `gorm:"type:text;not null"`
	Correcta   bool      `gorm:"not null;default:false"`
	Orden      int       `gorm:"not null;default:0"`
	CreatedAt  time.Time `gorm:"type:timestamptz;default:now()"`
}

// TableName overrides the GORM default with the actual SQL table name.
func (Option) TableName() string { return "question_option" }

// Attempt is SCHEMA-ONLY in C3.1 — endpoints are added in C3.2.
// The struct exists so GORM is aware of the table for any future auto-migrate.
type Attempt struct {
	ID           string     `gorm:"type:uuid;primaryKey"`
	UserID       string     `gorm:"column:user_id;type:uuid;not null"`
	EvaluationID string     `gorm:"column:evaluation_id;type:uuid;not null"`
	Numero       int        `gorm:"not null;default:1"`
	Puntaje      int        `gorm:"not null;default:0"`
	Aprobado     bool       `gorm:"not null;default:false"`
	IniciadoEn   time.Time  `gorm:"column:iniciado_en;type:timestamptz;default:now()"`
	FinalizadoEn *time.Time `gorm:"column:finalizado_en;type:timestamptz"`
}

// TableName overrides the GORM default with the actual SQL table name.
func (Attempt) TableName() string { return "attempt" }

// Answer is SCHEMA-ONLY in C3.1 — endpoints are added in C3.2.
type Answer struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	AttemptID  string `gorm:"column:attempt_id;type:uuid;not null"`
	QuestionID string `gorm:"column:question_id;type:uuid;not null"`
	OptionID   string `gorm:"column:option_id;type:uuid;not null"`
	Correcta   bool   `gorm:"not null;default:false"`
}

// TableName overrides the GORM default with the actual SQL table name.
func (Answer) TableName() string { return "answer" }
