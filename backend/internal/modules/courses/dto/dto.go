// Package dto contains the wire shapes for the courses module API.
// DTOs are kept separate from GORM models (domain) and service read-models
// so that JSON concerns do not leak into the domain or service layers.
package dto

import (
	"time"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── Course request bodies ──────────────────────────────────────────────────────

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

// ── Section request bodies ─────────────────────────────────────────────────────

// SectionCreateRequest is the body for POST /api/courses/:courseId/sections.
type SectionCreateRequest struct {
	Titulo string `json:"titulo" binding:"required,min=1,max=200"`
}

// SectionUpdateRequest is the body for PATCH /api/sections/:id.
// Pointer — nil means "not provided, do not update".
type SectionUpdateRequest struct {
	Titulo *string `json:"titulo" binding:"omitempty,min=1,max=200"`
}

// ReorderRequest is the body for PATCH /api/courses/:id/sections/reorder.
// ids must contain the EXACT full set of section IDs for the course.
type ReorderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

// ── Video request bodies ───────────────────────────────────────────────────────

// VideoCreateRequest is the body for POST /api/sections/:sectionId/videos.
// proveedor is validated by the DTO (oneof=youtube vimeo).
// url/proveedor cross-validation is done at the service layer (design §4 D4).
type VideoCreateRequest struct {
	Titulo    string `json:"titulo"    binding:"required,min=1,max=200"`
	URL       string `json:"url"       binding:"required,url"`
	Proveedor string `json:"proveedor" binding:"required,oneof=youtube vimeo"`
	DuracionS int    `json:"duracionS" binding:"omitempty,min=0"`
}

// VideoUpdateRequest is the body for PATCH /api/videos/:id.
// All fields are optional — nil means "not provided, do not update".
type VideoUpdateRequest struct {
	Titulo    *string `json:"titulo"    binding:"omitempty,min=1,max=200"`
	URL       *string `json:"url"       binding:"omitempty,url"`
	Proveedor *string `json:"proveedor" binding:"omitempty,oneof=youtube vimeo"`
	DuracionS *int    `json:"duracionS" binding:"omitempty,min=0"`
}

// ── Course response shapes ─────────────────────────────────────────────────────

// CourseDetail is the full course representation returned by POST and GET single.
// hasContent (D5) is computed via svc.HasContent on GET detail; false on Create/Update.
type CourseDetail struct {
	ID          string    `json:"id"`
	CreadorID   string    `json:"creadorId"`
	Titulo      string    `json:"titulo"`
	Descripcion string    `json:"descripcion"`
	Estado      string    `json:"estado"`
	HasContent  bool      `json:"hasContent"`
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

// ── Section response shapes ────────────────────────────────────────────────────

// SectionResponse is the wire shape for a section returned by the API.
type SectionResponse struct {
	ID        string    `json:"id"`
	CourseID  string    `json:"courseId"`
	Titulo    string    `json:"titulo"`
	Orden     int       `json:"orden"`
	CreatedAt time.Time `json:"createdAt"`
}

// ── Video response shapes ──────────────────────────────────────────────────────

// VideoResponse is the wire shape for a video returned by the API.
type VideoResponse struct {
	ID        string    `json:"id"`
	SectionID string    `json:"sectionId"`
	Titulo    string    `json:"titulo"`
	URL       string    `json:"url"`
	Proveedor string    `json:"proveedor"`
	DuracionS int       `json:"duracionS"`
	Orden     int       `json:"orden"`
	CreatedAt time.Time `json:"createdAt"`
}

// ── Mapping functions ──────────────────────────────────────────────────────────

// ToCourseDetail converts a service.CourseModel to the full CourseDetail wire shape.
// hasContent must be computed by the handler via svc.HasContent before calling this.
// Create/Update handlers pass hasContent=false (borrador — content state is re-fetched on reload).
func ToCourseDetail(m *service.CourseModel, hasContent bool) CourseDetail {
	return CourseDetail{
		ID:          m.ID,
		CreadorID:   m.CreadorID,
		Titulo:      m.Titulo,
		Descripcion: m.Descripcion,
		Estado:      string(m.Estado),
		HasContent:  hasContent,
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

// ToSection converts a service.SectionModel to the SectionResponse wire shape.
func ToSection(m *service.SectionModel) SectionResponse {
	return SectionResponse{
		ID:        m.ID,
		CourseID:  m.CourseID,
		Titulo:    m.Titulo,
		Orden:     m.Orden,
		CreatedAt: m.CreatedAt,
	}
}

// ToVideo converts a service.VideoModel to the VideoResponse wire shape.
func ToVideo(m *service.VideoModel) VideoResponse {
	return VideoResponse{
		ID:        m.ID,
		SectionID: m.SectionID,
		Titulo:    m.Titulo,
		URL:       m.URL,
		Proveedor: m.Proveedor,
		DuracionS: m.DuracionS,
		Orden:     m.Orden,
		CreatedAt: m.CreatedAt,
	}
}
