package service

import "errors"

// Service-layer sentinels for the evaluations module.
// Handlers map these via errors.Is to the appropriate HTTP status codes.
// The SAME sentinel (ErrNotOwner) maps to different statuses depending on the
// route — 404 on GET (hides existence), 403 on write ops (signals authz failure).
// This asymmetry mirrors the courses module pattern exactly.
var (
	// ErrEvaluationNotFound is returned when an evaluation lookup finds no row.
	ErrEvaluationNotFound = errors.New("evaluation not found")

	// ErrQuestionNotFound is returned when a question lookup finds no row.
	ErrQuestionNotFound = errors.New("question not found")

	// ErrOptionNotFound is returned when an option lookup finds no row.
	ErrOptionNotFound = errors.New("option not found")

	// ErrEvaluationExists is returned when creating a second evaluation for a course
	// that already has one (1-1 invariant). Maps to 409 Conflict.
	ErrEvaluationExists = errors.New("evaluation already exists for this course")

	// ErrNoCorrectOption is returned by validateQuestionComplete when no option
	// has correcta=true. Maps to 409 at the completeness gate (C3.2 submit flow).
	// Not wired to any mutating endpoint in C3.1 — incremental authoring is allowed.
	ErrNoCorrectOption = errors.New("question has no correct option")

	// ErrInvalidQuestionType is returned when a tipo value is not in the allowed set,
	// or when CreateOption is attempted on a verdadero_falso question.
	// Maps to 400 Bad Request.
	ErrInvalidQuestionType = errors.New("invalid question tipo")

	// ErrNotOwner is returned when the requesting creador does not own the course.
	// Handler maps this to:
	//   GET  routes → 404 (hides existence)
	//   POST/PATCH/DELETE routes → 403 (explicit authz signal)
	ErrNotOwner = errors.New("not the course owner")

	// ErrCourseNotFound is returned when the CoursesChecker cannot find the course.
	// Maps to 404 Not Found.
	ErrCourseNotFound = errors.New("course not found")

	// ErrCourseNotEditable is returned when the course estado is not in
	// {borrador, rechazado} and a mutating operation is attempted.
	// Maps to 409 Conflict.
	ErrCourseNotEditable = errors.New("course estado does not permit this edit")

	// ErrAttemptNotFound is returned when an attempt lookup finds no row OR when the
	// requesting user does not own the attempt (anti-enumeration — never 403).
	// Maps to 404 Not Found.
	ErrAttemptNotFound = errors.New("attempt not found")

	// ErrMaxAttemptsReached is returned when the student has used all allowed attempts
	// (intentos_max > 0 AND count >= intentos_max).
	// Maps to 409 Conflict.
	ErrMaxAttemptsReached = errors.New("max attempts reached")

	// ErrAttemptAlreadySubmitted is returned when a student tries to answer or
	// re-submit an already finalised attempt (finalizado_en is set).
	// Maps to 409 Conflict.
	ErrAttemptAlreadySubmitted = errors.New("attempt already submitted")

	// ErrAttemptOpen is returned when StartAttempt is called but the user already
	// has an open (unsubmitted) attempt for the same evaluation.
	// Maps to 409 Conflict.
	ErrAttemptOpen = errors.New("an open attempt already exists")

	// ErrInvalidAnswer is returned when the supplied (questionID, optionID) pair
	// is not valid for the attempt's evaluation (option does not belong to the
	// question, or question does not belong to the evaluation).
	// Maps to 400 Bad Request.
	ErrInvalidAnswer = errors.New("invalid answer")
)
