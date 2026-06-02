package service

import "errors"

// Domain sentinels used by the users service.
// Handlers map these via errors.Is to the appropriate HTTP status codes.
var (
	// ErrInvalidRole is returned when an unknown role name is supplied.
	// → 400 Bad Request
	ErrInvalidRole = errors.New("invalid role name")

	// ErrAddRemoveConflict is returned when the same role appears in both add
	// and remove sets (ambiguous intent).
	// → 400 Bad Request
	ErrAddRemoveConflict = errors.New("role appears in both add and remove")

	// ErrLastAdmin is returned when an operation would leave the platform with
	// zero active administrators.
	// → 409 Conflict
	ErrLastAdmin = errors.New("cannot remove the last active administrator")

	// ErrSelfSupervision is returned when supervisorID == empleadoID.
	// → 400 Bad Request
	ErrSelfSupervision = errors.New("a user cannot supervise themselves")

	// ErrSupervisionExists is returned when an employee already has a supervisor.
	// → 409 Conflict
	ErrSupervisionExists = errors.New("employee already has a supervisor")

	// ErrSupervisionNotFound is returned when a supervision relation does not
	// exist (DELETE /supervisions/:id on an unknown id).
	// → 404 Not Found
	ErrSupervisionNotFound = errors.New("supervision relation not found")

	// ErrUserNotFound is the service-layer re-export of the repository sentinel,
	// surfaced to handlers.
	// → 404 Not Found
	ErrUserNotFound = errors.New("user not found")
)

// validRoles is the closed set of role names recognised by the platform.
var validRoles = map[string]struct{}{
	"alumno":        {},
	"creador":       {},
	"supervisor":    {},
	"administrador": {},
}

// isValidRole reports whether name is one of the four fixed platform roles.
func isValidRole(name string) bool {
	_, ok := validRoles[name]
	return ok
}
