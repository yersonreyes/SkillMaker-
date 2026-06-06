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

	// ErrSectionNotFound is returned when a section lookup finds no row.
	// → 404 Not Found
	ErrSectionNotFound = errors.New("section not found")

	// ErrVideoNotFound is returned when a video lookup finds no row.
	// → 404 Not Found
	ErrVideoNotFound = errors.New("video not found")

	// ErrNotOwner is returned when the requesting creador does not own the course.
	// Handler maps this to:
	//   GET  /api/courses/:id  → 404 (existence must not be leaked)
	//   PATCH /api/courses/:id → 403 (explicit authz signal per AC3)
	//   POST/PATCH/DELETE section/video → 403 (mutating routes signal authz)
	ErrNotOwner = errors.New("not the course owner")

	// ErrInvalidTransition is returned when the course's current estado does not
	// permit the requested mutation (only borrador and rechazado allow edits).
	// → 409 Conflict
	ErrInvalidTransition = errors.New("course estado does not permit this edit")

	// ErrURLProviderMismatch is returned when the URL host does not match the
	// declared proveedor (e.g. vimeo.com URL with proveedor=youtube).
	// → 400 Bad Request
	ErrURLProviderMismatch = errors.New("url host does not match declared proveedor")

	// ErrInvalidReorderSet is returned when the ids passed to ReorderSections do
	// not exactly match the course's current section set (foreign id present, or
	// one or more sections missing from the provided list).
	// This is a VALIDATION error (caller sent wrong data), not a state conflict.
	// → 400 Bad Request  (distinct from ErrInvalidTransition → 409)
	ErrInvalidReorderSet = errors.New("reorder ids must exactly match the course's section set")

	// ── Material sentinels (C2.3) ────────────────────────────────────────────

	// ErrMaterialNotFound is returned when a material lookup finds no matching row.
	// → 404 Not Found
	ErrMaterialNotFound = errors.New("material not found")

	// ErrFileTooLarge is returned when the uploaded file exceeds the maximum allowed size.
	// → 413 Request Entity Too Large
	ErrFileTooLarge = errors.New("file exceeds max upload size")

	// ErrMIMENotAllowed is returned when the file's content type is not in the allowed whitelist.
	// → 415 Unsupported Media Type
	ErrMIMENotAllowed = errors.New("content type not allowed")

	// ErrInvalidMaterialKey is returned when the key in ConfirmUpload does not start
	// with the expected "courses/{courseID}/videos/{videoID}/materials/" prefix.
	// → 400 Bad Request
	ErrInvalidMaterialKey = errors.New("material key prefix mismatch")

	// ErrInvalidCategoria is returned when a categoriaId provided in CreateRequest
	// or UpdateRequest does not exist in the categoria table.
	// → 400 Bad Request
	ErrInvalidCategoria = errors.New("one or more categoria IDs are invalid")
)
