// Package repository contains the data-access layer for the courses module.
// It mirrors the users gormRepository pattern exactly: interface + sentinel +
// gormRepository struct + New(db). All cross-module access goes through the
// courses.go facade, never directly into this package.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// ErrCourseNotFound is returned when a course lookup finds no matching row.
var ErrCourseNotFound = errors.New("course not found")

// Repository defines the data-access contract for the courses module.
// UpdateEstado is a seam for C2.2/C4.1; it is defined now but not called in C2.1.
type Repository interface {
	// Create persists a new course. The caller must populate all required fields;
	// the repo assigns a UUID if ID is empty.
	Create(ctx context.Context, c *domain.Course) error

	// GetByID fetches a course by primary key.
	// Returns ErrCourseNotFound when no row matches.
	GetByID(ctx context.Context, id string) (*domain.Course, error)

	// UpdateByID applies a partial update via a field map. Only keys present in
	// fields are written (zero-value strings are written if present — unlike
	// struct Updates which skips zero values). RowsAffected==0 → ErrCourseNotFound.
	UpdateByID(ctx context.Context, id string, fields map[string]any) error

	// ListByCreator returns a paginated page of courses owned by creadorID,
	// ordered by created_at DESC. Mirrors users.List pagination pattern.
	ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[domain.Course], error)

	// UpdateEstado sets the estado field on the given course.
	// Seam for C2.2/C4.1 — implemented now, not called in C2.1.
	// Returns ErrCourseNotFound when no row matches.
	UpdateEstado(ctx context.Context, id string, estado domain.Estado) error
}

// ── gormRepository ─────────────────────────────────────────────────────────────

type gormRepository struct {
	db *gorm.DB
}

// New returns a Repository backed by GORM.
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, c *domain.Course) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *gormRepository) GetByID(ctx context.Context, id string) (*domain.Course, error) {
	var c domain.Course
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&c)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrCourseNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &c, nil
}

// UpdateByID applies a field map via GORM's Model+Where+Updates pattern.
// Using a map (not a struct) ensures zero-value strings (e.g. empty descripcion)
// are written to the database and not silently skipped.
func (r *gormRepository) UpdateByID(ctx context.Context, id string, fields map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Course{}).
		Where("id = ?", id).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCourseNotFound
	}
	return nil
}

// ListByCreator mirrors users.List: build base → Count → Offset/Limit/Find → NewPage.
// Re-using the base query after Count is safe — GORM's chain is immutable for
// additional clauses (Count does not consume the chain).
func (r *gormRepository) ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[domain.Course], error) {
	base := r.db.WithContext(ctx).
		Model(&domain.Course{}).
		Where("creador_id = ?", creadorID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return pagination.Page[domain.Course]{}, err
	}

	var courses []domain.Course
	err := base.
		Order("created_at DESC").
		Offset(p.Offset()).
		Limit(p.Limit()).
		Find(&courses).Error
	if err != nil {
		return pagination.Page[domain.Course]{}, err
	}

	return pagination.NewPage(courses, total, p), nil
}

// UpdateEstado sets the estado column. Seam for C2.2/C4.1.
func (r *gormRepository) UpdateEstado(ctx context.Context, id string, estado domain.Estado) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Course{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"estado":     estado,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCourseNotFound
	}
	return nil
}

// isPgUniqueViolation reports whether err is a Postgres UNIQUE violation (23505).
// Mirrors the users repository helper — avoids importing pgconn directly.
func isPgUniqueViolation(err error) bool { //nolint:unused // kept for future Create unique-guard needs (C2.x)
	type pgcoder interface{ SQLState() string }
	var pg pgcoder
	if errors.As(err, &pg) {
		return pg.SQLState() == "23505"
	}
	return false
}
