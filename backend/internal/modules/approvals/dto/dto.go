// Package dto contains the request/response DTOs and mappers for the approvals module.
// JSON field names use camelCase per project convention.
// Mirrors the evaluations/dto pattern exactly.
package dto

import (
	"time"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
)

// ── Request DTOs ──────────────────────────────────────────────────────────────

// ApproveRequest is the optional body for POST /courses/:courseId/approve.
// Comentario is optional on approve (stored as ” when absent).
type ApproveRequest struct {
	Comentario string `json:"comentario"`
}

// RejectRequest is the mandatory body for POST /courses/:courseId/reject.
// Comentario is required at the binding layer for immediate 400 on empty body.
// Note: the service also validates TrimSpace(comentario)!="" for defense-in-depth.
type RejectRequest struct {
	Comentario string `json:"comentario"`
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

// SubmitResponse is returned by POST /courses/:courseId/submit.
// Estado reflects the new state (en_revision).
type SubmitResponse struct {
	CourseID string `json:"courseId"`
	Estado   string `json:"estado"`
}

// PendingItemDTO is one item in the GET /approvals/pending list.
// FechaEnvio uses UpdatedAt from the course row as a proxy for submission time (OQ-3).
type PendingItemDTO struct {
	ID        string    `json:"id"`
	Titulo    string    `json:"titulo"`
	CreadorID string    `json:"creadorId"`
	Estado    string    `json:"estado"`
	CreatedAt time.Time `json:"fechaEnvio"` // OQ-3: UpdatedAt would be ideal; using CreatedAt as proxy in C4.1
}

// ApprovalHistoryDTO is one item in the GET /courses/:id/approvals list.
type ApprovalHistoryDTO struct {
	ID         string    `json:"id"`
	Resultado  string    `json:"resultado"`
	Comentario string    `json:"comentario"`
	AdminID    string    `json:"adminId"`
	ResueltoEn time.Time `json:"resueltoEn"`
}

// ── Mappers ───────────────────────────────────────────────────────────────────

// ToPending maps a slice of courses.CourseSummary to []PendingItemDTO.
func ToPending(rows []courses.CourseSummary) []PendingItemDTO {
	out := make([]PendingItemDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, PendingItemDTO{
			ID:        r.ID,
			Titulo:    r.Titulo,
			CreadorID: r.CreadorID,
			Estado:    r.Estado,
			CreatedAt: r.CreatedAt,
		})
	}
	return out
}

// ToHistory maps a slice of domain.Approval to []ApprovalHistoryDTO.
func ToHistory(rows []domain.Approval) []ApprovalHistoryDTO {
	out := make([]ApprovalHistoryDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, ApprovalHistoryDTO{
			ID:         r.ID,
			Resultado:  r.Resultado,
			Comentario: r.Comentario,
			AdminID:    r.AdminID,
			ResueltoEn: r.ResueltoEn,
		})
	}
	return out
}
