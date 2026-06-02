// Package dto contains the wire shapes for the users module API.
// DTOs are kept separate from GORM models and service read-models so that
// JSON concerns do not leak into domain or service layers.
package dto

import (
	"time"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── Request bodies ─────────────────────────────────────────────────────────────

// RolesPatchRequest is the body for PATCH /api/users/:id/roles.
// Add and Remove are sets of role names to apply as a delta.
type RolesPatchRequest struct {
	Add    []string `json:"add"    binding:"omitempty,dive,required"`
	Remove []string `json:"remove" binding:"omitempty,dive,required"`
}

// ActivePatchRequest is the body for PATCH /api/users/:id/active.
// Active is a pointer so that `{"active":false}` is accepted and a missing
// field produces a 400 (avoids the Go zero-value trap).
type ActivePatchRequest struct {
	Active *bool `json:"active" binding:"required"`
}

// SupervisionCreateRequest is the body for POST /api/supervisions (PR-B).
type SupervisionCreateRequest struct {
	SupervisorID string `json:"supervisorId" binding:"required,uuid"`
	EmpleadoID   string `json:"empleadoId"   binding:"required,uuid"`
}

// ── Response shapes ────────────────────────────────────────────────────────────

// UserListItem is the compact user representation used in paginated list responses.
type UserListItem struct {
	ID     string   `json:"id"`
	Email  string   `json:"email"`
	Nombre string   `json:"nombre"`
	Activo bool     `json:"activo"`
	Roles  []string `json:"roles"`
}

// UserDetail is the full user representation used in single-user responses.
type UserDetail struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Nombre    string    `json:"nombre"`
	Activo    bool      `json:"activo"`
	Roles     []string  `json:"roles"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SupervisionItem is the wire shape for a supervision relation (PR-B, declared now
// so the facade type alias can reference it without a PR-B stub).
type SupervisionItem struct {
	ID           string    `json:"id"`
	SupervisorID string    `json:"supervisorId"`
	EmpleadoID   string    `json:"empleadoId"`
	CreadoEn     time.Time `json:"creadoEn"`
}

// ── Mapping functions ─────────────────────────────────────────────────────────

// ToUserListItem converts a service.UserDetailModel to a UserListItem wire shape.
func ToUserListItem(m *service.UserDetailModel) UserListItem {
	roles := m.Roles
	if roles == nil {
		roles = []string{}
	}

	return UserListItem{
		ID:     m.ID,
		Email:  m.Email,
		Nombre: m.Nombre,
		Activo: m.Activo,
		Roles:  roles,
	}
}

// ToUserDetail converts a service.UserDetailModel to a UserDetail wire shape.
func ToUserDetail(m *service.UserDetailModel) UserDetail {
	roles := m.Roles
	if roles == nil {
		roles = []string{}
	}

	return UserDetail{
		ID:        m.ID,
		Email:     m.Email,
		Nombre:    m.Nombre,
		Activo:    m.Activo,
		Roles:     roles,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

// ToUserListPage maps a pagination.Page[service.UserDetailModel] to a
// pagination.Page[UserListItem], converting each item with ToUserListItem.
func ToUserListPage(p pagination.Page[service.UserDetailModel]) pagination.Page[UserListItem] {
	items := make([]UserListItem, 0, len(p.Items))
	for i := range p.Items {
		items = append(items, ToUserListItem(&p.Items[i]))
	}

	return pagination.Page[UserListItem]{
		Items:      items,
		Page:       p.Page,
		Size:       p.Size,
		Total:      p.Total,
		TotalPages: p.TotalPages,
	}
}

// ToSupervisionItem converts a service.SupervisionModel to a SupervisionItem wire shape.
func ToSupervisionItem(m service.SupervisionModel) SupervisionItem {
	return SupervisionItem{
		ID:           m.ID,
		SupervisorID: m.SupervisorID,
		EmpleadoID:   m.EmpleadoID,
		CreadoEn:     m.CreadoEn,
	}
}
