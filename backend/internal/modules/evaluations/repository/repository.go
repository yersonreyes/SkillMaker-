// Package repository contains the data-access layer for the evaluations module.
// It mirrors the courses gormRepository pattern exactly: interface + sentinels +
// gormRepository struct + New(db). All cross-module access goes through the
// evaluations.go facade, never directly into this package.
package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/domain"
)

// Repository sentinels returned when a lookup finds no matching row.
var (
	// ErrEvaluationNotFound is returned when an evaluation lookup finds no row.
	ErrEvaluationNotFound = errors.New("evaluation not found")

	// ErrQuestionNotFound is returned when a question lookup finds no row.
	ErrQuestionNotFound = errors.New("question not found")

	// ErrOptionNotFound is returned when an option lookup finds no row.
	ErrOptionNotFound = errors.New("option not found")

	// ErrEvaluationExists is returned when a second evaluation is created for
	// a course that already has one (maps Postgres UNIQUE violation 23505).
	ErrEvaluationExists = errors.New("evaluation already exists for this course")
)

// Repository defines the data-access contract for the evaluations module.
type Repository interface {
	// CreateEvaluation persists a new evaluation.
	// Maps Postgres UNIQUE violation on course_id → ErrEvaluationExists.
	CreateEvaluation(ctx context.Context, e *domain.Evaluation) error

	// GetEvaluationByID fetches an evaluation by primary key.
	// Returns ErrEvaluationNotFound when no row matches.
	GetEvaluationByID(ctx context.Context, id string) (*domain.Evaluation, error)

	// GetEvaluationByCourse fetches the evaluation for a course.
	// Returns ErrEvaluationNotFound when no row matches.
	GetEvaluationByCourse(ctx context.Context, courseID string) (*domain.Evaluation, error)

	// UpdateEvaluation applies a partial field map update to an evaluation.
	UpdateEvaluation(ctx context.Context, id string, fields map[string]any) error

	// CreateQuestion persists a new question. Assigns a UUID if ID is empty.
	CreateQuestion(ctx context.Context, q *domain.Question) error

	// GetQuestionByID fetches a question by primary key.
	// Returns ErrQuestionNotFound when no row matches.
	GetQuestionByID(ctx context.Context, id string) (*domain.Question, error)

	// UpdateQuestion applies a partial field map update to a question.
	UpdateQuestion(ctx context.Context, id string, fields map[string]any) error

	// DeleteQuestion deletes a question by ID. FK ON DELETE CASCADE removes child options.
	DeleteQuestion(ctx context.Context, id string) error

	// ListQuestionsByEvaluation returns all questions for an evaluation ordered by orden ASC.
	ListQuestionsByEvaluation(ctx context.Context, evaluationID string) ([]domain.Question, error)

	// CreateOptions batch-inserts options in a single call.
	// Used to atomically create the two auto-options for verdadero_falso questions.
	CreateOptions(ctx context.Context, opts []domain.Option) error

	// GetOptionByID fetches an option by primary key.
	// Returns ErrOptionNotFound when no row matches.
	GetOptionByID(ctx context.Context, id string) (*domain.Option, error)

	// UpdateOption applies a partial field map update to an option.
	UpdateOption(ctx context.Context, id string, fields map[string]any) error

	// DeleteOption deletes an option by ID.
	DeleteOption(ctx context.Context, id string) error

	// ListOptionsByQuestion returns all options for a question ordered by orden ASC.
	ListOptionsByQuestion(ctx context.Context, questionID string) ([]domain.Option, error)
}

// ── gormRepository ─────────────────────────────────────────────────────────────

type gormRepository struct {
	db *gorm.DB
}

// New returns a Repository backed by GORM.
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// ── Evaluation implementations ─────────────────────────────────────────────────

func (r *gormRepository) CreateEvaluation(ctx context.Context, e *domain.Evaluation) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	err := r.db.WithContext(ctx).Create(e).Error
	if isPgUniqueViolation(err) {
		return ErrEvaluationExists
	}
	return err
}

func (r *gormRepository) GetEvaluationByID(ctx context.Context, id string) (*domain.Evaluation, error) {
	var e domain.Evaluation
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&e)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrEvaluationNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &e, nil
}

func (r *gormRepository) GetEvaluationByCourse(ctx context.Context, courseID string) (*domain.Evaluation, error) {
	var e domain.Evaluation
	result := r.db.WithContext(ctx).Where("course_id = ?", courseID).First(&e)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrEvaluationNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &e, nil
}

func (r *gormRepository) UpdateEvaluation(ctx context.Context, id string, fields map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Evaluation{}).
		Where("id = ?", id).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrEvaluationNotFound
	}
	return nil
}

// ── Question implementations ───────────────────────────────────────────────────

func (r *gormRepository) CreateQuestion(ctx context.Context, q *domain.Question) error {
	if q.ID == "" {
		q.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(q).Error
}

func (r *gormRepository) GetQuestionByID(ctx context.Context, id string) (*domain.Question, error) {
	var q domain.Question
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&q)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrQuestionNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &q, nil
}

func (r *gormRepository) UpdateQuestion(ctx context.Context, id string, fields map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Question{}).
		Where("id = ?", id).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrQuestionNotFound
	}
	return nil
}

func (r *gormRepository) DeleteQuestion(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Question{}).Error
}

func (r *gormRepository) ListQuestionsByEvaluation(ctx context.Context, evaluationID string) ([]domain.Question, error) {
	var questions []domain.Question
	err := r.db.WithContext(ctx).
		Where("evaluation_id = ?", evaluationID).
		Order("orden ASC").
		Find(&questions).Error
	if err != nil {
		return nil, err
	}
	return questions, nil
}

// ── Option implementations ─────────────────────────────────────────────────────

func (r *gormRepository) CreateOptions(ctx context.Context, opts []domain.Option) error {
	for i := range opts {
		if opts[i].ID == "" {
			opts[i].ID = uuid.New().String()
		}
	}
	return r.db.WithContext(ctx).Create(&opts).Error
}

func (r *gormRepository) GetOptionByID(ctx context.Context, id string) (*domain.Option, error) {
	var o domain.Option
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&o)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrOptionNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &o, nil
}

func (r *gormRepository) UpdateOption(ctx context.Context, id string, fields map[string]any) error {
	result := r.db.WithContext(ctx).
		Model(&domain.Option{}).
		Where("id = ?", id).
		Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrOptionNotFound
	}
	return nil
}

func (r *gormRepository) DeleteOption(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Option{}).Error
}

func (r *gormRepository) ListOptionsByQuestion(ctx context.Context, questionID string) ([]domain.Option, error) {
	var options []domain.Option
	err := r.db.WithContext(ctx).
		Where("question_id = ?", questionID).
		Order("orden ASC").
		Find(&options).Error
	if err != nil {
		return nil, err
	}
	return options, nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

// isPgUniqueViolation reports whether err is a Postgres UNIQUE violation (23505).
// Mirrors the courses repository helper.
func isPgUniqueViolation(err error) bool {
	type pgcoder interface{ SQLState() string }
	var pg pgcoder
	if errors.As(err, &pg) {
		return pg.SQLState() == "23505"
	}
	return false
}
