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

// ── Catalog read models (C2.4) ─────────────────────────────────────────────────

// CatalogCourseModel is the read model for an approved-course catalog card,
// carrying the creator's DISPLAY NAME (joined from "user") — not the UUID.
type CatalogCourseModel struct {
	ID            string
	Titulo        string
	Descripcion   string
	CreadorNombre string
	CreatedAt     time.Time
}

// EnrollmentWithCourseModel is the read model for GET /users/me/courses.
// It joins enrollment + course + "user" to get all fields needed by the alumno.
type EnrollmentWithCourseModel struct {
	CourseID      string
	Titulo        string
	CreadorNombre string
	Completado    bool
	InscritoEn    time.Time
}

// ErrSectionNotFound is returned when a section lookup finds no matching row.
var ErrSectionNotFound = errors.New("section not found")

// ErrVideoNotFound is returned when a video lookup finds no matching row.
var ErrVideoNotFound = errors.New("video not found")

// ErrMaterialNotFound is returned when a material lookup finds no matching row.
var ErrMaterialNotFound = errors.New("material not found")

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

	// ── C4.1 additions ────────────────────────────────────────────────────────

	// UpdateEstadoPublicado sets estado + publicado_en + updated_at in one UPDATE.
	// Used exclusively for the "aprobado" transition — the only state that stamps publicado_en.
	// Returns ErrCourseNotFound when no row matches.
	UpdateEstadoPublicado(ctx context.Context, id string, estado domain.Estado, publicadoEn time.Time) error

	// ListByEstado returns all courses with the given estado, ordered by created_at ASC.
	// Used by the approvals module to list pending courses (en_revision).
	ListByEstado(ctx context.Context, estado domain.Estado) ([]domain.Course, error)

	// ── Catalog + enrollment methods (C2.4) ─────────────────────────────────────

	// ListApproved returns a paginated page of courses with estado='aprobado',
	// optionally filtered by titulo ILIKE (case-insensitive). The CatalogCourseModel
	// carries the creator's nombre (joined from "user"), not the UUID.
	// Count runs BEFORE the Select chain to avoid the GORM Count+Select gotcha.
	ListApproved(ctx context.Context, p pagination.Params, q string) (pagination.Page[CatalogCourseModel], error)

	// GetApprovedDetail fetches one aprobado course + creator name.
	// Returns ErrCourseNotFound when the course is missing OR not aprobado (draft-invisibility).
	GetApprovedDetail(ctx context.Context, courseID string) (*CatalogCourseModel, error)

	// CreateEnrollment creates an enrollment row for (userID, courseID).
	// Idempotent: ON CONFLICT (user_id, course_id) DO NOTHING — no error on duplicate.
	CreateEnrollment(ctx context.Context, userID, courseID string) error

	// IsEnrolled returns true when an enrollment row for (userID, courseID) exists.
	IsEnrolled(ctx context.Context, userID, courseID string) (bool, error)

	// ListEnrollmentsByUser returns all enrollment rows for the given userID,
	// joining course + "user" for titulo, creadorNombre, completado, inscritoEn.
	// Ordered by inscrito_en DESC. Scoped strictly to userID.
	ListEnrollmentsByUser(ctx context.Context, userID string) ([]EnrollmentWithCourseModel, error)

	// MarkCompleted sets completado=true for the (userID, courseID) enrollment row.
	// No-op (nil, no error) when no matching row exists (0 rows affected).
	MarkCompleted(ctx context.Context, userID, courseID string) error

	// ── Material methods (C2.3) ────────────────────────────────────────────────

	// CreateMaterial persists a new material. Assigns a UUID if ID is empty.
	CreateMaterial(ctx context.Context, m *domain.Material) error

	// GetMaterialByID fetches a material by primary key.
	// Returns ErrMaterialNotFound when no row matches.
	GetMaterialByID(ctx context.Context, id string) (*domain.Material, error)

	// ListMaterialsByCourse returns all materials for a course ordered by created_at ASC.
	ListMaterialsByCourse(ctx context.Context, courseID string) ([]domain.Material, error)

	// DeleteMaterial deletes a material by ID.
	DeleteMaterial(ctx context.Context, id string) error
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

// ── Material implementations (C2.3) ───────────────────────────────────────────

func (r *gormRepository) CreateMaterial(ctx context.Context, m *domain.Material) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(m).Error
}

func (r *gormRepository) GetMaterialByID(ctx context.Context, id string) (*domain.Material, error) {
	var m domain.Material
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&m)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrMaterialNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &m, nil
}

func (r *gormRepository) ListMaterialsByCourse(ctx context.Context, courseID string) ([]domain.Material, error) {
	var materials []domain.Material
	err := r.db.WithContext(ctx).
		Where("course_id = ?", courseID).
		Order("created_at ASC").
		Find(&materials).Error
	if err != nil {
		return nil, err
	}
	return materials, nil
}

func (r *gormRepository) DeleteMaterial(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Material{}).Error
}

// ── C4.1 additions ────────────────────────────────────────────────────────────

// UpdateEstadoPublicado sets estado + publicado_en + updated_at in one UPDATE.
// This is the ONLY path that writes publicado_en — used exclusively by SetEstado("aprobado").
// The existing UpdateEstado (2-arg) remains unchanged so callers that don't stamp publicado_en
// can continue using it without risking unintentional publicado_en changes.
func (r *gormRepository) UpdateEstadoPublicado(ctx context.Context, id string, estado domain.Estado, publicadoEn time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Course{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"estado":       estado,
			"publicado_en": publicadoEn,
			"updated_at":   time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCourseNotFound
	}
	return nil
}

// ListByEstado returns all courses with the given estado ordered by created_at ASC.
func (r *gormRepository) ListByEstado(ctx context.Context, estado domain.Estado) ([]domain.Course, error) {
	var courses []domain.Course
	err := r.db.WithContext(ctx).
		Where("estado = ?", estado).
		Order("created_at ASC").
		Find(&courses).Error
	return courses, err
}

// ── Catalog + enrollment implementations (C2.4) ────────────────────────────────

// ListApproved returns a paginated page of aprobado courses.
// GOTCHA: Count must run on a base WITHOUT Select to avoid GORM "column ambiguous" errors.
// Only AFTER Count does the chain apply Select + Order + Offset + Limit.
func (r *gormRepository) ListApproved(ctx context.Context, p pagination.Params, q string) (pagination.Page[CatalogCourseModel], error) {
	base := r.db.WithContext(ctx).
		Table("course").
		Joins(`JOIN "user" ON "user".id = course.creador_id`).
		Where("course.estado = ?", "aprobado")
	if q != "" {
		base = base.Where("course.titulo ILIKE ?", "%"+q+"%")
	}

	// COUNT first (no Select) — avoids the GORM Count+Select gotcha (§3).
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return pagination.Page[CatalogCourseModel]{}, err
	}

	var rows []CatalogCourseModel
	err := base.
		Select(`course.id AS id, course.titulo AS titulo, course.descripcion AS descripcion,
		         "user".nombre AS creador_nombre, course.created_at AS created_at`).
		Order("course.created_at DESC").
		Offset(p.Offset()).Limit(p.Limit()).
		Scan(&rows).Error
	if err != nil {
		return pagination.Page[CatalogCourseModel]{}, err
	}
	return pagination.NewPage(rows, total, p), nil
}

// GetApprovedDetail fetches one aprobado course by ID + creator name.
// Returns ErrCourseNotFound when missing OR not aprobado (hides drafts from alumno).
// NOTE: .Scan on a struct with no matching row sets RowsAffected==0, not gorm.ErrRecordNotFound.
func (r *gormRepository) GetApprovedDetail(ctx context.Context, courseID string) (*CatalogCourseModel, error) {
	var row CatalogCourseModel
	result := r.db.WithContext(ctx).
		Table("course").
		Joins(`JOIN "user" ON "user".id = course.creador_id`).
		Where("course.id = ? AND course.estado = ?", courseID, "aprobado").
		Select(`course.id AS id, course.titulo AS titulo, course.descripcion AS descripcion,
		         "user".nombre AS creador_nombre, course.created_at AS created_at`).
		Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrCourseNotFound
	}
	return &row, nil
}

// CreateEnrollment inserts an enrollment row. Idempotent via ON CONFLICT DO NOTHING
// on the existing uq_enrollment_user_course UNIQUE constraint (migration 0003).
func (r *gormRepository) CreateEnrollment(ctx context.Context, userID, courseID string) error {
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO enrollment (id, user_id, course_id, inscrito_en, completado)
		 VALUES (gen_random_uuid(), ?, ?, now(), false)
		 ON CONFLICT (user_id, course_id) DO NOTHING`,
		userID, courseID,
	).Error
}

// IsEnrolled returns true when a (userID, courseID) enrollment row exists.
func (r *gormRepository) IsEnrolled(ctx context.Context, userID, courseID string) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).Raw(
		`SELECT EXISTS(SELECT 1 FROM enrollment WHERE user_id = ? AND course_id = ?)`,
		userID, courseID,
	).Scan(&exists).Error
	return exists, err
}

// ListEnrollmentsByUser returns all enrollment rows for userID, joining course + "user".
// "user" is quoted — it is a reserved word in Postgres (users-repo precedent).
// Ordered by inscrito_en DESC.
func (r *gormRepository) ListEnrollmentsByUser(ctx context.Context, userID string) ([]EnrollmentWithCourseModel, error) {
	var rows []EnrollmentWithCourseModel
	err := r.db.WithContext(ctx).
		Table("enrollment e").
		Joins("JOIN course c ON c.id = e.course_id").
		Joins(`JOIN "user" u ON u.id = c.creador_id`).
		Where("e.user_id = ?", userID).
		Select(`e.course_id AS course_id, c.titulo AS titulo, u.nombre AS creador_nombre,
		         e.completado AS completado, e.inscrito_en AS inscrito_en`).
		Order("e.inscrito_en DESC").
		Scan(&rows).Error
	return rows, err
}

// MarkCompleted sets completado=true for the (userID, courseID) enrollment.
// No-op (nil, no error) when no matching row — RowsAffected==0 is not an error (D4).
func (r *gormRepository) MarkCompleted(ctx context.Context, userID, courseID string) error {
	return r.db.WithContext(ctx).
		Model(&domain.Enrollment{}).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		Update("completado", true).Error
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
