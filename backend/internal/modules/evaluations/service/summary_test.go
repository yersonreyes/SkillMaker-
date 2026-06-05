// Package service — unit tests for GetEvaluationSummaryForStudent (student-facing summary).
//
// TDD Cycle: RED (this file written first, before summary.go exists) → GREEN.
//
// LOAD-BEARING tests:
//
//	(a) aprobado + eval exists → returns EvaluationSummaryModel
//	(b) non-aprobado estado → ErrEvaluationNotFound (do NOT leak unpublished courses)
//	(c) course not found → ErrCourseNotFound
//	(d) eval not found for aprobado course → ErrEvaluationNotFound
package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── [LOAD-BEARING (a)] aprobado course + eval → returns summary ───────────────

// TestGetEvaluationSummaryForStudent_AprobadoWithEval_ReturnsSummary verifies the happy path.
// Spec: aprobado estado + evaluation exists → EvaluationSummaryModel with correct fields.
func TestGetEvaluationSummaryForStudent_AprobadoWithEval_ReturnsSummary(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	courseID := uuid.New().String()
	e := evalFixture(courseID)

	checker.On("GetCourseOwnership", mock.Anything, courseID).
		Return(uuid.New().String(), "aprobado", nil)
	repo.On("GetEvaluationByCourse", mock.Anything, courseID).
		Return(e, nil)

	summary, err := svc.GetEvaluationSummaryForStudent(context.Background(), courseID)

	require.NoError(t, err, "[LOAD-BEARING (a)] aprobado+eval must return summary without error")
	require.NotNil(t, summary)
	assert.Equal(t, e.ID, summary.EvaluationID)
	assert.Equal(t, e.NotaMinima, summary.NotaMinima)
	assert.Equal(t, e.IntentosMax, summary.IntentosMax)
	checker.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// ── [LOAD-BEARING (b)] non-aprobado → ErrEvaluationNotFound ──────────────────

// TestGetEvaluationSummaryForStudent_NonAprobado_ReturnsErrEvaluationNotFound verifies
// that courses not in "aprobado" estado are hidden from students.
// Spec: do NOT leak unpublished courses.
func TestGetEvaluationSummaryForStudent_NonAprobado_ReturnsErrEvaluationNotFound(t *testing.T) {
	for _, estado := range []string{"borrador", "en_revision", "rechazado"} {
		t.Run("estado="+estado, func(t *testing.T) {
			repo := &mockEvalRepo{}
			checker := &testutil.MockCoursesChecker{}
			svc := newSvc(repo, checker)

			courseID := uuid.New().String()
			checker.On("GetCourseOwnership", mock.Anything, courseID).
				Return(uuid.New().String(), estado, nil)

			_, err := svc.GetEvaluationSummaryForStudent(context.Background(), courseID)

			assert.ErrorIs(t, err, ErrEvaluationNotFound,
				"[LOAD-BEARING (b)] estado=%s must return ErrEvaluationNotFound (do not leak)", estado)
			checker.AssertExpectations(t)
			repo.AssertExpectations(t)
		})
	}
}

// ── [LOAD-BEARING (c)] course not found → ErrCourseNotFound ──────────────────

// TestGetEvaluationSummaryForStudent_CourseNotFound_ReturnsErrCourseNotFound verifies
// that when CoursesChecker returns courses.ErrCourseNotFound, the service wraps to ErrCourseNotFound.
func TestGetEvaluationSummaryForStudent_CourseNotFound_ReturnsErrCourseNotFound(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	courseID := uuid.New().String()
	checker.On("GetCourseOwnership", mock.Anything, courseID).
		Return("", "", courses.ErrCourseNotFound)

	_, err := svc.GetEvaluationSummaryForStudent(context.Background(), courseID)

	assert.ErrorIs(t, err, ErrCourseNotFound,
		"[LOAD-BEARING (c)] course not found must return ErrCourseNotFound")
	checker.AssertExpectations(t)
}

// ── [LOAD-BEARING (d)] aprobado course but no evaluation → ErrEvaluationNotFound ──

// TestGetEvaluationSummaryForStudent_AprobadoNoEval_ReturnsErrEvaluationNotFound verifies
// that when the course is aprobado but has no evaluation, ErrEvaluationNotFound is returned.
func TestGetEvaluationSummaryForStudent_AprobadoNoEval_ReturnsErrEvaluationNotFound(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	courseID := uuid.New().String()
	checker.On("GetCourseOwnership", mock.Anything, courseID).
		Return(uuid.New().String(), "aprobado", nil)
	repo.On("GetEvaluationByCourse", mock.Anything, courseID).
		Return((*domain.Evaluation)(nil), repository.ErrEvaluationNotFound)

	_, err := svc.GetEvaluationSummaryForStudent(context.Background(), courseID)

	assert.ErrorIs(t, err, ErrEvaluationNotFound,
		"[LOAD-BEARING (d)] aprobado course with no eval must return ErrEvaluationNotFound")
	checker.AssertExpectations(t)
	repo.AssertExpectations(t)
}
