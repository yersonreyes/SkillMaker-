package service

import "errors"

// Domain sentinels for the courses service.
// Handlers map these via errors.Is to the appropriate HTTP status codes.
// The SAME sentinel (ErrNotOwner) maps to different statuses depending on the
// route — 404 on GET (hides existence), 403 on PATCH (signals auth failure).
// This asymmetry is INTENTIONAL and LOCKED per REQ-DIVERGENCE.
var (
	// ErrCourseNotFound is returned when a course lookup finds no row.
	// → 404 Not Found
	ErrCourseNotFound = errors.New("course not found")

	// ErrNotOwner is returned when the requesting creador does not own the course.
	// Handler maps this to:
	//   GET  /api/courses/:id  → 404 (existence must not be leaked)
	//   PATCH /api/courses/:id → 403 (explicit authz signal per AC3)
	ErrNotOwner = errors.New("not the course owner")

	// ErrInvalidTransition is returned when the course's current estado does not
	// permit the requested mutation (only borrador and rechazado allow edits).
	// → 409 Conflict
	ErrInvalidTransition = errors.New("course estado does not permit this edit")
)
