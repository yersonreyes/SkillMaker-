// Package repository contains the data-access layer for the approvals module.
// Mirrors the courses/evaluations gormRepository pattern exactly:
// interface + sentinels + gormRepository struct + New(db).
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/domain"
)

// Repository defines the data-access contract for the approvals module.
type Repository interface {
	// Create persists a new approval row.
	Create(ctx context.Context, a *domain.Approval) error

	// ListByCourse returns all approval rows for a given course,
	// ordered by resuelto_en DESC (most recent first).
	ListByCourse(ctx context.Context, courseID string) ([]domain.Approval, error)
}

// ── gormRepository ─────────────────────────────────────────────────────────────

type gormRepository struct {
	db *gorm.DB
}

// New returns a Repository backed by GORM.
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// Create persists a new approval row. The caller must populate all required fields;
// ID must be a pre-generated UUID.
func (r *gormRepository) Create(ctx context.Context, a *domain.Approval) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// ListByCourse returns all approval rows for the given course, ordered by resuelto_en DESC.
func (r *gormRepository) ListByCourse(ctx context.Context, courseID string) ([]domain.Approval, error) {
	var rows []domain.Approval
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("resuelto_en DESC").
		Find(&rows).Error
	return rows, err
}
