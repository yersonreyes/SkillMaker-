// Package service — student-facing evaluation summary (student-eval-discovery).
//
// GetEvaluationSummaryForStudent provides the slim read-model a student needs to
// discover that a course has an evaluation and to navigate to the attempt page.
//
// Security invariant: ONLY "aprobado" courses are visible to students.
// Any other estado returns ErrEvaluationNotFound (do NOT leak unpublished courses).
package service

import "context"

// EvaluationSummaryModel is the slim read-model returned to students.
// It contains only the fields needed to render the "Rendir evaluación" entry point.
type EvaluationSummaryModel struct {
	EvaluationID string
	NotaMinima   int
	IntentosMax  int
}

// GetEvaluationSummaryForStudent returns the evaluation summary for a student.
//
// Rules:
//  1. Call CoursesChecker.GetCourseOwnership to get course estado.
//  2. If the course is not found → ErrCourseNotFound.
//  3. If estado != "aprobado" → ErrEvaluationNotFound (hides unpublished courses).
//  4. Fetch evaluation by course → ErrEvaluationNotFound if none.
//  5. Return slim summary {EvaluationID, NotaMinima, IntentosMax}.
func (s *serviceImpl) GetEvaluationSummaryForStudent(ctx context.Context, courseID string) (*EvaluationSummaryModel, error) {
	// 1. Resolve course estado via seam (no ownership check — student route, any user).
	_, estado, err := s.courses.GetCourseOwnership(ctx, courseID)
	if err != nil {
		return nil, wrapCourseNotFound(err)
	}

	// 2. Only "aprobado" courses are visible to students (do NOT leak unpublished).
	if estado != "aprobado" {
		return nil, ErrEvaluationNotFound
	}

	// 3. Fetch the evaluation for this course.
	e, err := s.repo.GetEvaluationByCourse(ctx, courseID)
	if err != nil {
		return nil, wrapEvalNotFound(err)
	}

	return &EvaluationSummaryModel{
		EvaluationID: e.ID,
		NotaMinima:   e.NotaMinima,
		IntentosMax:  e.IntentosMax,
	}, nil
}
