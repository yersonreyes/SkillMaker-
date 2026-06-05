// Package service contains the business logic for the approvals module.
// It is HTTP-agnostic: it returns domain sentinels and read-models.
// Handlers are responsible for mapping sentinels → HTTP status codes.
//
// Cross-module seams (CourseStateManager, EvaluationValidator) are defined HERE
// and satisfied structurally by coursesSvc/evaluationsSvc — identical pattern
// to evaluations.CoursesChecker (evaluations/service.go:26).
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
)

// ── Cross-module seam interfaces ───────────────────────────────────────────────

// CourseStateManager is the narrow cross-module seam into courses.
// coursesSvc satisfies this structurally (Go structural typing on interfaces).
// Defined here per design §2 — approvals/service, NOT domain.
type CourseStateManager interface {
	GetCourseOwnership(ctx context.Context, courseID string) (creadorID, estado string, err error)
	HasContent(ctx context.Context, courseID, creadorID string) (bool, error)
	SetEstado(ctx context.Context, courseID, estado string) error
	ListByEstado(ctx context.Context, estado string) ([]courses.CourseSummary, error)
}

// EvaluationValidator is the narrow cross-module seam into evaluations.
// evaluationsSvc satisfies this structurally.
type EvaluationValidator interface {
	ValidateSubmitReady(ctx context.Context, courseID string) error
}

// ── Repository interface ───────────────────────────────────────────────────────

// Repository defines the data-access contract for the approvals module.
type Repository interface {
	Create(ctx context.Context, a *domain.Approval) error
	ListByCourse(ctx context.Context, courseID string) ([]domain.Approval, error)
}

// ── Service interface ──────────────────────────────────────────────────────────

// Service is the public interface of the approvals domain.
type Service interface {
	// SubmitToReview transitions a course from borrador/rechazado to en_revision.
	// Guards in order: ownership → transition → content → evaluation readiness.
	SubmitToReview(ctx context.Context, courseID, callerID string) error

	// Approve approves a course in en_revision.
	// Two-write order: Create audit row FIRST, then SetEstado (R1/D6).
	Approve(ctx context.Context, courseID, adminID, comentario string) error

	// Reject rejects a course in en_revision with a mandatory comment.
	// Comment validated FIRST before any reads or writes (SEC-7).
	Reject(ctx context.Context, courseID, adminID, comentario string) error

	// ListPending returns all courses with estado=en_revision.
	ListPending(ctx context.Context) ([]courses.CourseSummary, error)

	// ListHistory returns the approval history for a course.
	// Owner-or-admin: non-owners and non-admins receive ErrNotOwner.
	ListHistory(ctx context.Context, courseID, callerID string, isAdmin bool) ([]domain.Approval, error)
}

// ── concrete implementation ────────────────────────────────────────────────────

type serviceImpl struct {
	repo    Repository
	courses CourseStateManager
	evals   EvaluationValidator
}

// New creates a Service backed by the given Repository, CourseStateManager, and EvaluationValidator.
// Constructor mirrors evaluations.New (service.go:54 pattern): func New(r repo, courses, opts).
// No functional options needed for C4.1.
func New(r Repository, courses CourseStateManager, evals EvaluationValidator) Service {
	return &serviceImpl{repo: r, courses: courses, evals: evals}
}

// ── SubmitToReview ─────────────────────────────────────────────────────────────

// SubmitToReview enforces the submit guards in strict order per design §4 D5:
// 1. Ownership (ErrNotOwner 403)
// 2. Estado transition: only borrador/rechazado → en_revision (ErrCourseNotSubmittable 409)
// 3. Content presence (ErrNoContent 409)
// 4. Evaluation readiness (evaluation sentinels surfaced verbatim)
// Then sets estado to en_revision.
func (s *serviceImpl) SubmitToReview(ctx context.Context, courseID, callerID string) error {
	owner, estado, err := s.courses.GetCourseOwnership(ctx, courseID)
	if err != nil {
		return wrapCourseNotFound(err, courses.ErrCourseNotFound)
	}

	// 1. Ownership FIRST (D5: authz before state).
	if owner != callerID {
		return ErrNotOwner
	}

	// 2. Transition guard: only borrador and rechazado can submit.
	if estado != "borrador" && estado != "rechazado" {
		return ErrCourseNotSubmittable
	}

	// 3. Content check.
	ok, err := s.courses.HasContent(ctx, courseID, callerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNoContent
	}

	// 4. Evaluation readiness (surfaces evaluations' own sentinels verbatim).
	if err := s.evals.ValidateSubmitReady(ctx, courseID); err != nil {
		return err
	}

	return s.courses.SetEstado(ctx, courseID, "en_revision")
}

// ── Approve ────────────────────────────────────────────────────────────────────

// Approve approves a course.
// TWO-WRITE ORDERING (D6/R1): Create audit row FIRST, then SetEstado.
// If SetEstado fails after Create: audit row exists for a course whose estado did NOT change
// → retry is safe (Approve re-checks estado==en_revision and writes a second audit row on retry;
// acceptable for append-only history). The alternative (estado first) would leave a published
// course with NO audit trail — the worse failure for a compliance feature.
func (s *serviceImpl) Approve(ctx context.Context, courseID, adminID, comentario string) error {
	_, estado, err := s.courses.GetCourseOwnership(ctx, courseID)
	if err != nil {
		return wrapCourseNotFound(err, courses.ErrCourseNotFound)
	}
	if estado != "en_revision" {
		return ErrNotInReview
	}

	// (a) Write audit row FIRST.
	a := &domain.Approval{
		ID:         uuid.New().String(),
		CourseID:   courseID,
		AdminID:    adminID,
		Resultado:  "aprobado",
		Comentario: comentario,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return err
	}

	// (b) Set estado (stamps publicado_en via courses.SetEstado → UpdateEstadoPublicado).
	return s.courses.SetEstado(ctx, courseID, "aprobado")
}

// ── Reject ─────────────────────────────────────────────────────────────────────

// Reject rejects a course with a mandatory comment (SEC-7: comment checked FIRST).
// Two-write ordering: Create audit row FIRST, then SetEstado (same as Approve).
func (s *serviceImpl) Reject(ctx context.Context, courseID, adminID, comentario string) error {
	// SEC-7: Comment must be non-empty BEFORE any reads or writes.
	if strings.TrimSpace(comentario) == "" {
		return ErrCommentRequired
	}

	_, estado, err := s.courses.GetCourseOwnership(ctx, courseID)
	if err != nil {
		return wrapCourseNotFound(err, courses.ErrCourseNotFound)
	}
	if estado != "en_revision" {
		return ErrNotInReview
	}

	// (a) Write audit row FIRST.
	a := &domain.Approval{
		ID:         uuid.New().String(),
		CourseID:   courseID,
		AdminID:    adminID,
		Resultado:  "rechazado",
		Comentario: comentario,
	}
	if err := s.repo.Create(ctx, a); err != nil {
		return err
	}

	// (b) Set estado to rechazado (does NOT clear publicado_en per XMOD-3).
	return s.courses.SetEstado(ctx, courseID, "rechazado")
}

// ── ListPending ────────────────────────────────────────────────────────────────

// ListPending returns all courses with estado=en_revision ordered by created_at ASC.
func (s *serviceImpl) ListPending(ctx context.Context) ([]courses.CourseSummary, error) {
	return s.courses.ListByEstado(ctx, "en_revision")
}

// ── ListHistory ────────────────────────────────────────────────────────────────

// ListHistory returns approval rows for the given course, owner-or-admin gated (D7).
// Non-owners and non-admins receive ErrNotOwner (handler maps to 404 for read routes).
func (s *serviceImpl) ListHistory(ctx context.Context, courseID, callerID string, isAdmin bool) ([]domain.Approval, error) {
	owner, _, err := s.courses.GetCourseOwnership(ctx, courseID)
	if err != nil {
		return nil, wrapCourseNotFound(err, courses.ErrCourseNotFound)
	}

	if !isAdmin && owner != callerID {
		return nil, ErrNotOwner
	}

	return s.repo.ListByCourse(ctx, courseID)
}
