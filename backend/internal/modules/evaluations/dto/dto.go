// Package dto contains the wire shapes for the evaluations module API.
// DTOs are kept separate from GORM models (domain) and service read-models
// so that JSON concerns do not leak into the domain or service layers.
// Mirrors the courses dto pattern exactly (camelCase JSON, binding tags).
package dto

import (
	"time"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
)

// ── Request bodies ─────────────────────────────────────────────────────────────

// EvaluationCreateRequest is the body for POST /api/courses/:courseId/evaluation.
type EvaluationCreateRequest struct {
	NotaMinima  *int `json:"notaMinima"  binding:"omitempty,min=0,max=100"`
	IntentosMax *int `json:"intentosMax" binding:"omitempty,min=0"`
}

// EvaluationUpdateRequest is the body for PATCH /api/evaluations/:id.
type EvaluationUpdateRequest struct {
	NotaMinima  *int `json:"notaMinima"  binding:"omitempty,min=0,max=100"`
	IntentosMax *int `json:"intentosMax" binding:"omitempty,min=0"`
}

// QuestionCreateRequest is the body for POST /api/evaluations/:id/questions.
type QuestionCreateRequest struct {
	Enunciado string `json:"enunciado" binding:"required,min=1,max=1000"`
	Tipo      string `json:"tipo"      binding:"required,oneof=opcion_multiple verdadero_falso"`
	Puntaje   *int   `json:"puntaje"   binding:"omitempty,min=1"`
}

// QuestionUpdateRequest is the body for PATCH /api/questions/:id.
// Tipo is intentionally absent — it is immutable after creation.
type QuestionUpdateRequest struct {
	Enunciado *string `json:"enunciado" binding:"omitempty,min=1,max=1000"`
	Puntaje   *int    `json:"puntaje"   binding:"omitempty,min=1"`
	Orden     *int    `json:"orden"     binding:"omitempty,min=0"`
}

// OptionCreateRequest is the body for POST /api/questions/:id/options.
type OptionCreateRequest struct {
	Texto    string `json:"texto"    binding:"required,min=1,max=500"`
	Correcta *bool  `json:"correcta" binding:"omitempty"`
}

// OptionUpdateRequest is the body for PATCH /api/options/:id.
type OptionUpdateRequest struct {
	Texto    *string `json:"texto"    binding:"omitempty,min=1,max=500"`
	Correcta *bool   `json:"correcta" binding:"omitempty"`
}

// ── Response shapes ────────────────────────────────────────────────────────────

// EvaluationResponse is the flat evaluation representation returned by create/update.
type EvaluationResponse struct {
	ID          string    `json:"id"`
	CourseID    string    `json:"courseId"`
	NotaMinima  int       `json:"notaMinima"`
	IntentosMax int       `json:"intentosMax"`
	CreatedAt   time.Time `json:"createdAt"`
}

// OptionResponse is the option representation.
type OptionResponse struct {
	ID         string `json:"id"`
	QuestionID string `json:"questionId"`
	Texto      string `json:"texto"`
	Correcta   bool   `json:"correcta"`
	Orden      int    `json:"orden"`
}

// QuestionResponse is the flat question representation.
type QuestionResponse struct {
	ID           string `json:"id"`
	EvaluationID string `json:"evaluationId"`
	Enunciado    string `json:"enunciado"`
	Tipo         string `json:"tipo"`
	Puntaje      int    `json:"puntaje"`
	Orden        int    `json:"orden"`
}

// QuestionDetail is a question with its options nested.
type QuestionDetail struct {
	QuestionResponse
	Options []OptionResponse `json:"options"`
}

// EvaluationDetail is the full nested representation returned by GET evaluation.
type EvaluationDetail struct {
	ID          string           `json:"id"`
	CourseID    string           `json:"courseId"`
	NotaMinima  int              `json:"notaMinima"`
	IntentosMax int              `json:"intentosMax"`
	Questions   []QuestionDetail `json:"questions"`
}

// ── Mappers ────────────────────────────────────────────────────────────────────

// ToEvaluation converts a service EvaluationModel to a flat EvaluationResponse.
func ToEvaluation(m *service.EvaluationModel) EvaluationResponse {
	return EvaluationResponse{
		ID:          m.ID,
		CourseID:    m.CourseID,
		NotaMinima:  m.NotaMinima,
		IntentosMax: m.IntentosMax,
	}
}

// ToOption converts a service OptionModel to an OptionResponse.
func ToOption(m *service.OptionModel) OptionResponse {
	return OptionResponse{
		ID:         m.ID,
		QuestionID: m.QuestionID,
		Texto:      m.Texto,
		Correcta:   m.Correcta,
		Orden:      m.Orden,
	}
}

// ToQuestionDetail converts a service QuestionWithOptionsModel to a QuestionDetail.
func ToQuestionDetail(m *service.QuestionWithOptionsModel) QuestionDetail {
	options := make([]OptionResponse, 0, len(m.Options))
	for i := range m.Options {
		options = append(options, ToOption(&m.Options[i]))
	}
	return QuestionDetail{
		QuestionResponse: QuestionResponse{
			ID:           m.ID,
			EvaluationID: m.EvaluationID,
			Enunciado:    m.Enunciado,
			Tipo:         m.Tipo,
			Puntaje:      m.Puntaje,
			Orden:        m.Orden,
		},
		Options: options,
	}
}

// ToEvaluationDetail converts a service EvaluationDetailModel to a nested EvaluationDetail.
func ToEvaluationDetail(m *service.EvaluationDetailModel) EvaluationDetail {
	questions := make([]QuestionDetail, 0, len(m.Questions))
	for i := range m.Questions {
		questions = append(questions, ToQuestionDetail(&m.Questions[i]))
	}
	return EvaluationDetail{
		ID:          m.ID,
		CourseID:    m.CourseID,
		NotaMinima:  m.NotaMinima,
		IntentosMax: m.IntentosMax,
		Questions:   questions,
	}
}

// ToQuestion converts a service QuestionModel to a flat QuestionResponse.
func ToQuestion(m *service.QuestionModel) QuestionResponse {
	return QuestionResponse{
		ID:           m.ID,
		EvaluationID: m.EvaluationID,
		Enunciado:    m.Enunciado,
		Tipo:         m.Tipo,
		Puntaje:      m.Puntaje,
		Orden:        m.Orden,
	}
}
