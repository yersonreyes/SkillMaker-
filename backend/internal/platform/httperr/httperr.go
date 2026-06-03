package httperr

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Error is a structured HTTP error that implements the error interface.
// Code is a machine-readable constant (e.g. "NOT_FOUND"); Message is
// human-readable. Details carries field-level validation errors when present.
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// ──────────────────────────────────────────────────────────────
// Factories
// ──────────────────────────────────────────────────────────────

func BadRequest(code, message string) *Error {
	return &Error{Status: http.StatusBadRequest, Code: code, Message: message}
}

func NotFound(code, message string) *Error {
	return &Error{Status: http.StatusNotFound, Code: code, Message: message}
}

func Forbidden(code, message string) *Error {
	return &Error{Status: http.StatusForbidden, Code: code, Message: message}
}

func Conflict(code, message string) *Error {
	return &Error{Status: http.StatusConflict, Code: code, Message: message}
}

func Internal(message string) *Error {
	return &Error{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: message}
}

func Unauthorized(code, message string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: code, Message: message}
}

// PayloadTooLarge returns a 413 Request Entity Too Large error.
// Used when the uploaded file exceeds the maximum allowed size.
func PayloadTooLarge(code, message string) *Error {
	return &Error{Status: http.StatusRequestEntityTooLarge, Code: code, Message: message}
}

// UnsupportedMediaType returns a 415 Unsupported Media Type error.
// Used when the file's content type is not in the allowed MIME whitelist.
func UnsupportedMediaType(code, message string) *Error {
	return &Error{Status: http.StatusUnsupportedMediaType, Code: code, Message: message}
}

// ──────────────────────────────────────────────────────────────
// Render
// ──────────────────────────────────────────────────────────────

// Render writes the appropriate JSON error response and aborts the request.
// It handles *Error directly; validator.ValidationErrors are converted to a
// 400 with field-level Details; any other error becomes a 500.
func Render(c *gin.Context, err error) {
	var httpErr *Error
	if errors.As(err, &httpErr) {
		c.AbortWithStatusJSON(httpErr.Status, httpErr)
		return
	}

	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		details := make(map[string]string, len(valErrs))
		for _, fe := range valErrs {
			details[fe.Field()] = fe.Tag()
		}
		c.AbortWithStatusJSON(http.StatusBadRequest, &Error{
			Status:  http.StatusBadRequest,
			Code:    "VALIDATION_ERROR",
			Message: "one or more fields are invalid",
			Details: details,
		})
		return
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, &Error{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_ERROR",
		Message: "an unexpected error occurred",
	})
}
