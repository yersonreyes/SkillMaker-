// Package dto contains the response DTOs for the reporting module.
// JSON field names use camelCase per project convention.
// Swagger annotations follow the same pattern as other modules.
//
// DTO collision guard: none of these names collide with other modules.
// Verified with: grep -rhoE "^type [A-Za-z]+ " backend/internal/modules/*/dto/*.go | sort | uniq -d
// (must remain empty after adding this file — NEVER add --parseDependency to swag).
package dto

// GlobalReportResponse is the response body for GET /api/reports/global.
// @Description System-wide aggregate metrics.
type GlobalReportResponse struct {
	ActiveUsers             int64                 `json:"activeUsers"`
	CoursesByEstado         []CoursesByEstadoItem `json:"coursesByEstado"`
	TotalAttempts           int64                 `json:"totalAttempts"`
	CertificatesIssued      int64                 `json:"certificatesIssued"`
	TopCreators             []TopCreatorItem      `json:"topCreators"`
	UsersPerMonth           []MonthCountItem      `json:"usersPerMonth"`
	ApprovedCoursesPerMonth []MonthCountItem      `json:"approvedCoursesPerMonth"`
}

// CoursesByEstadoItem is one entry in GlobalReportResponse.CoursesByEstado.
// All 4 estados (borrador, en_revision, aprobado, rechazado) are always present.
// @Description Course count by estado.
type CoursesByEstadoItem struct {
	Estado string `json:"estado"`
	Total  int64  `json:"total"`
}

// TopCreatorItem is one entry in GlobalReportResponse.TopCreators.
// @Description Creator ranked by aprobado course count.
type TopCreatorItem struct {
	Nombre string `json:"nombre"`
	Total  int64  `json:"total"`
}

// MonthCountItem is one entry in UsersPerMonth or ApprovedCoursesPerMonth.
// Month is an ISO 8601 month string ("2026-01").
// @Description Month bucket with count.
type MonthCountItem struct {
	Month string `json:"month"` // ISO "2006-01"
	Total int64  `json:"total"`
}

// CourseReportItem is one item in the GET /api/reports/courses response (bare array).
// @Description Per-course aggregate stats.
type CourseReportItem struct {
	ID           string  `json:"courseId"`
	Titulo       string  `json:"titulo"`
	Estado       string  `json:"estado"`
	Enrollments  int64   `json:"enrollments"`
	Attempts     int64   `json:"attempts"`
	ApprovalRate float64 `json:"approvalRate"` // range [0.0, 1.0]; frontend renders as %
}

// UserProgressResponse is the response body for GET /api/reports/users/:id/progress.
// All-zero when the target user has no activity (OQ-1 zero-return invariant).
// @Description User progress aggregate.
type UserProgressResponse struct {
	Enrolled       int64 `json:"enrolledCount"`
	Completed      int64 `json:"completedCount"`
	Attempts       int64 `json:"attemptsCount"`
	PassedAttempts int64 `json:"passedAttemptsCount"`
	Certificates   int64 `json:"certificatesCount"`
}

// TeamReportItem is one item in the GET /api/reports/team response (bare array).
// Supervisor sees ONLY their own employees (WHERE supervision.supervisor_id = callerID).
// LastAttemptDate is the ISO 8601 date of the employee's most recent finalized attempt,
// or null when the employee has no finalized attempts (spec REQ-TEAM).
// @Description Team member progress entry.
type TeamReportItem struct {
	EmpleadoID      string  `json:"empleadoId"`
	EmpleadoNombre  string  `json:"empleadoNombre"`
	Enrolled        int64   `json:"enrolledCount"`
	Completed       int64   `json:"completedCount"`
	LastAttemptDate *string `json:"lastAttemptDate"`
}
