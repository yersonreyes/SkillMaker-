// Package service — sentinels for the approvals module.
package service

import "errors"

// Approvals service-layer sentinels.
// Handlers map these via errors.Is to the appropriate HTTP status codes.
// Evaluation sentinels (ErrEvaluationNotFound, ErrNoCorrectOption) are NOT redefined here;
// approvals surfaces them verbatim from the evaluations seam — same approach evaluations
// uses for courses.ErrCourseNotFound. Handlers check errors.Is(err, evaluations.ErrXxx).
var (
	// ErrNotOwner is returned when the requesting creador does not own the course.
	// Submit → 403 (write); ListHistory → 404 (read, hide existence).
	ErrNotOwner = errors.New("not the course owner")

	// ErrCourseNotSubmittable is returned when the course estado is not in
	// {borrador, rechazado} — states that permit submission.
	// → 409 Conflict.
	ErrCourseNotSubmittable = errors.New("course estado does not permit submission")

	// ErrNoContent is returned when the course has no video content.
	// → 409 Conflict (httperr has no 422 factory).
	ErrNoContent = errors.New("course has no content")

	// ErrNotInReview is returned when approve/reject is attempted on a course
	// that is not currently in en_revision estado.
	// → 409 Conflict.
	ErrNotInReview = errors.New("course is not in review")

	// ErrCommentRequired is returned when a rejection is attempted with an
	// empty or whitespace-only comentario. Checked BEFORE any DB writes (SEC-7).
	// → 400 Bad Request.
	ErrCommentRequired = errors.New("rejection comment is required")

	// ErrCourseNotFound is returned when the course seam cannot find the course.
	// Wraps courses.ErrCourseNotFound via wrapCourseNotFound.
	// → 404 Not Found.
	ErrCourseNotFound = errors.New("course not found")
)

// wrapCourseNotFound converts courses.ErrCourseNotFound to approvals.ErrCourseNotFound.
// Mirrors the wrapCourseNotFound pattern used in evaluations/service.
// Imported via the courses facade (courses.ErrCourseNotFound) to stay ADR-1 safe.
func wrapCourseNotFound(err, coursesErrCourseNotFound error) error {
	if errors.Is(err, coursesErrCourseNotFound) {
		return ErrCourseNotFound
	}
	return err
}
