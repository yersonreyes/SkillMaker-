// Package service contains the business logic for the courses module.
// It is HTTP-agnostic: it returns domain sentinels and CourseModel read-models.
// Handlers are responsible for mapping sentinels → HTTP status codes.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ── Read models ────────────────────────────────────────────────────────────────

// CourseModel is the service-layer read model returned to handlers and DTOs.
// Callers never import the domain package directly; they use CourseModel.
type CourseModel struct {
	ID          string
	CreadorID   string
	Titulo      string
	Descripcion string
	Estado      domain.Estado
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ── Request types ──────────────────────────────────────────────────────────────

// CreateRequest carries the caller-supplied data for creating a course.
// Estado is intentionally absent — the service always forces borrador (AC1).
type CreateRequest struct {
	Titulo      string
	Descripcion string
}

// UpdateRequest carries optional fields for a partial course update.
// Nil pointer = field not supplied by the caller = leave as-is in DB.
// This matches design §4: field-map so zero-value strings can be written.
type UpdateRequest struct {
	Titulo      *string
	Descripcion *string
}

// ── Service interface ──────────────────────────────────────────────────────────

// Service is the public interface of the courses domain.
// Other modules (handlers) depend on this interface — never on serviceImpl.
type Service interface {
	// Create persists a new course with estado=borrador (forced, regardless of
	// any client input). creadorID is always taken from the JWT, not from the body.
	Create(ctx context.Context, creadorID string, req CreateRequest) (*CourseModel, error)

	// GetByID returns the course for the given owner.
	// Returns ErrNotOwner if creadorID does not match the course's creador_id.
	// Returns ErrCourseNotFound if no course with that id exists.
	GetByID(ctx context.Context, id, creadorID string) (*CourseModel, error)

	// UpdateByID partially updates a course the caller owns.
	// Ownership is checked BEFORE the estado transition guard (D5 ordering).
	// Returns ErrNotOwner for non-owners; ErrInvalidTransition when estado ∉ {borrador, rechazado}.
	UpdateByID(ctx context.Context, id, creadorID string, req UpdateRequest) (*CourseModel, error)

	// ListByCreator returns a paginated page of courses owned by creadorID.
	ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[CourseModel], error)
}

// ── concrete implementation ────────────────────────────────────────────────────

type serviceImpl struct {
	repo repository.Repository
}

// New creates a Service backed by the given Repository.
func New(repo repository.Repository) Service {
	return &serviceImpl{repo: repo}
}

// Create forces estado=borrador and sets creadorID from the argument (never from request).
func (s *serviceImpl) Create(ctx context.Context, creadorID string, req CreateRequest) (*CourseModel, error) {
	c := &domain.Course{
		ID:          uuid.New().String(),
		CreadorID:   creadorID,
		Titulo:      req.Titulo,
		Descripcion: req.Descripcion,
		Estado:      domain.EstadoBorrador, // FORCED — client cannot influence this
	}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return toModel(c), nil
}

// GetByID fetches the course then enforces ownership.
func (s *serviceImpl) GetByID(ctx context.Context, id, creadorID string) (*CourseModel, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}
	if c.CreadorID != creadorID {
		return nil, ErrNotOwner // handler maps to 404 (hides existence)
	}
	return toModel(c), nil
}

// UpdateByID enforces ownership FIRST, then transition guard, then applies partial update.
// Ordering rationale: a non-owner editing an aprobado course must get ErrNotOwner (403),
// not ErrInvalidTransition (409) — authz outranks state. See LOAD-BEARING-B.
func (s *serviceImpl) UpdateByID(ctx context.Context, id, creadorID string, req UpdateRequest) (*CourseModel, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, wrapNotFound(err)
	}

	// 1. Ownership check FIRST (handler → 403 on PATCH).
	if c.CreadorID != creadorID {
		return nil, ErrNotOwner
	}

	// 2. Transition guard: only borrador and rechazado allow edits.
	if c.Estado != domain.EstadoBorrador && c.Estado != domain.EstadoRechazado {
		return nil, ErrInvalidTransition
	}

	// 3. Build the field map — only include keys the caller explicitly provided.
	// Using a map (not struct) ensures zero-value strings are written (T1 trade-off).
	fields := map[string]any{
		"updated_at": time.Now(),
	}
	if req.Titulo != nil {
		fields["titulo"] = *req.Titulo
	}
	if req.Descripcion != nil {
		fields["descripcion"] = *req.Descripcion
	}

	if err := s.repo.UpdateByID(ctx, id, fields); err != nil {
		return nil, wrapNotFound(err)
	}

	// 4. Re-read for a fresh model reflecting DB state (including updated_at). See T2.
	return s.GetByID(ctx, id, creadorID)
}

// ListByCreator delegates to the repository and maps the page items.
func (s *serviceImpl) ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[CourseModel], error) {
	repoPage, err := s.repo.ListByCreator(ctx, creadorID, p)
	if err != nil {
		return pagination.Page[CourseModel]{}, err
	}

	items := make([]CourseModel, 0, len(repoPage.Items))
	for i := range repoPage.Items {
		items = append(items, *toModel(&repoPage.Items[i]))
	}

	return pagination.Page[CourseModel]{
		Items:      items,
		Page:       repoPage.Page,
		Size:       repoPage.Size,
		Total:      repoPage.Total,
		TotalPages: repoPage.TotalPages,
	}, nil
}

// ── private helpers ────────────────────────────────────────────────────────────

// wrapNotFound converts repository.ErrCourseNotFound to service.ErrCourseNotFound.
func wrapNotFound(err error) error {
	if errors.Is(err, repository.ErrCourseNotFound) {
		return ErrCourseNotFound
	}
	return err
}

// toModel converts a domain.Course GORM model to a CourseModel read-model.
func toModel(c *domain.Course) *CourseModel {
	return &CourseModel{
		ID:          c.ID,
		CreadorID:   c.CreadorID,
		Titulo:      c.Titulo,
		Descripcion: c.Descripcion,
		Estado:      c.Estado,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}
