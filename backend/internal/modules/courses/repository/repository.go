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

// ErrSectionNotFound is returned when a section lookup finds no matching row.
var ErrSectionNotFound = errors.New("section not found")

// ErrVideoNotFound is returned when a video lookup finds no matching row.
var ErrVideoNotFound = errors.New("video not found")

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

	// ── Section methods (C2.2) ─────────────────────────────────────────────────

	// CreateSection persists a new section. Assigns a UUID if ID is empty.
	CreateSection(ctx context.Context, s *domain.Section) error

	// GetSectionByID fetches a section by primary key.
	// Returns ErrSectionNotFound when no row matches.
	GetSectionByID(ctx context.Context, id string) (*domain.Section, error)

	// ListSectionsByCourse returns all sections for a course ordered by orden ASC.
	ListSectionsByCourse(ctx context.Context, courseID string) ([]domain.Section, error)

	// UpdateSection applies a partial field map update to a section.
	// RowsAffected==0 → ErrSectionNotFound.
	UpdateSection(ctx context.Context, id string, fields map[string]any) error

	// DeleteSection deletes a section by ID. FK ON DELETE CASCADE removes child videos.
	DeleteSection(ctx context.Context, id string) error

	// ReorderSections updates orden=index for each section in ids within a transaction.
	// ids must match the course's sections exactly (validated at service layer).
	ReorderSections(ctx context.Context, courseID string, ids []string) error

	// ── Video methods (C2.2) ───────────────────────────────────────────────────

	// CreateVideo persists a new video. Assigns a UUID if ID is empty.
	CreateVideo(ctx context.Context, v *domain.Video) error

	// GetVideoByID fetches a video by primary key.
	// Returns ErrVideoNotFound when no row matches.
	GetVideoByID(ctx context.Context, id string) (*domain.Video, error)

	// ListVideosBySection returns all videos for a section ordered by orden ASC.
	ListVideosBySection(ctx context.Context, sectionID string) ([]domain.Video, error)

	// UpdateVideo applies a partial field map update to a video.
	// RowsAffected==0 → ErrVideoNotFound.
	UpdateVideo(ctx context.Context, id string, fields map[string]any) error

	// DeleteVideo deletes a video by ID.
	DeleteVideo(ctx context.Context, id string) error

	// HasContent returns true if the course has at least one video (via any section).
	// Uses an EXISTS subquery joining video → section → course.
	HasContent(ctx context.Context, courseID string) (bool, error)
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

// ── Section implementations ────────────────────────────────────────────────────

func (r *gormRepository) CreateSection(ctx context.Context, s *domain.Section) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *gormRepository) GetSectionByID(ctx context.Context, id string) (*domain.Section, error) {
	var s domain.Section
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&s)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrSectionNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &s, nil
}

func (r *gormRepository) ListSectionsByCourse(ctx context.Context, courseID string) ([]domain.Section, error) {
	var sections []domain.Section
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("orden ASC").
		Find(&sections).Error
	if err != nil {
		return nil, err
	}
	return sections, nil
}

func (r *gormRepository) UpdateSection(ctx context.Context, id string, fields map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Section{}).
		Where("id = ?", id).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrSectionNotFound
	}
	return nil
}

func (r *gormRepository) DeleteSection(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Section{}).Error
}

// ReorderSections updates orden=index for each section in ids within a single transaction.
// The service layer must validate set-equality before calling this.
func (r *gormRepository) ReorderSections(ctx context.Context, courseID string, ids []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			result := tx.Model(&domain.Section{}).
				Where("id = ? AND course_id = ?", id, courseID).
				Update("orden", i)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}

// ── Video implementations ──────────────────────────────────────────────────────

func (r *gormRepository) CreateVideo(ctx context.Context, v *domain.Video) error {
	if v.ID == "" {
		v.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *gormRepository) GetVideoByID(ctx context.Context, id string) (*domain.Video, error) {
	var v domain.Video
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&v)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrVideoNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &v, nil
}

func (r *gormRepository) ListVideosBySection(ctx context.Context, sectionID string) ([]domain.Video, error) {
	var videos []domain.Video
	err := r.db.WithContext(ctx).
		Where("section_id = ?", sectionID).
		Order("orden ASC").
		Find(&videos).Error
	if err != nil {
		return nil, err
	}
	return videos, nil
}

func (r *gormRepository) UpdateVideo(ctx context.Context, id string, fields map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Video{}).
		Where("id = ?", id).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrVideoNotFound
	}
	return nil
}

func (r *gormRepository) DeleteVideo(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Video{}).Error
}

// HasContent checks whether a course has at least one video via an EXISTS subquery.
// Uses raw SQL to join video → section → course for a single DB round-trip.
func (r *gormRepository) HasContent(ctx context.Context, courseID string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).Raw(
		`SELECT EXISTS(
			SELECT 1 FROM video v
			JOIN section s ON v.section_id = s.id
			WHERE s.course_id = ?
		)`, courseID,
	).Scan(&exists).Error
	if err != nil {
		return false, err
	}
	return exists, nil
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
