// Package dto contains the wire shapes for the courses module API.
// DTOs are kept separate from GORM models (domain) and service read-models
// so that JSON concerns do not leak into the domain or service layers.
package dto

import (
	"time"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── Request bodies ─────────────────────────────────────────────────────────────

// CreateCourseRequest is the body for POST /api/courses.
// titulo is required (min=3). descripcion is optional.
// estado is intentionally absent — the service always forces borrador.
type CreateCourseRequest struct {
	Titulo      string `json:"titulo"      binding:"required,min=3,max=200"`
	Descripcion string `json:"descripcion" binding:"omitempty,max=5000"`
}

// UpdateCourseRequest is the body for PATCH /api/courses/:id.
// Both fields use pointers to distinguish "not provided" (nil) from "set to empty".
// This enables true partial updates at the service layer.
type UpdateCourseRequest struct {
	Titulo      *string `json:"titulo"      binding:"omitempty,min=3,max=200"`
	Descripcion *string `json:"descripcion" binding:"omitempty,max=5000"`
}

// ── Response shapes ────────────────────────────────────────────────────────────

// CourseDetail is the full course representation returned by POST and GET single.
type CourseDetail struct {
	ID          string    `json:"id"`
	CreadorID   string    `json:"creadorId"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	Estado      string    `json:"estado"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CourseListItem is the compact representation used in paginated list responses.
type CourseListItem struct {
	ID        string    `json:"id"`
	Titulo    string    `json:"titulo"`
	Estado    string    `json:"estado"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ── Mapping functions ──────────────────────────────────────────────────────────

// ToCourseDetail converts a service.CourseModel to the full CourseDetail wire shape.
func ToCourseDetail(m *service.CourseModel) CourseDetail {
	return CourseDetail{
		ID:          m.ID,
		CreadorID:   m.CreadorID,
		Titulo:      m.Titulo,
		Descripcion: m.Descripcion,
		Estado:      string(m.Estado),
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ToCourseListItem converts a service.CourseModel to the compact CourseListItem shape.
func ToCourseListItem(m *service.CourseModel) CourseListItem {
	return CourseListItem{
		ID:        m.ID,
		Titulo:    m.Titulo,
		Estado:    string(m.Estado),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// ToCourseListPage maps a pagination.Page[service.CourseModel] to a
// pagination.Page[CourseListItem], converting each item with ToCourseListItem.
func ToCourseListPage(p pagination.Page[service.CourseModel]) pagination.Page[CourseListItem] {
	items := make([]CourseListItem, 0, len(p.Items))
	for i := range p.Items {
		items = append(items, ToCourseListItem(&p.Items[i]))
	}
	return pagination.Page[CourseListItem]{
		Items:      items,
		Page:       p.Page,
		Size:       p.Size,
		Total:      p.Total,
		TotalPages: p.TotalPages,
	}
}
