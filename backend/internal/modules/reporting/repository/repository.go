// Package repository contains the data-access layer for the reporting module.
// This is a pure-SQL read-only repository: NO domain entities, NO migrations.
// It holds a *gorm.DB and queries other modules' tables directly via Raw SQL.
// This is the SANCTIONED cross-module exception (design §2, REQ-MODULE).
//
// Coupling mitigation: every method carries a doc comment listing the tables
// and columns it reads, along with the migration that created them (design §6).
package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// ── Scan structs (gorm column tags) ──────────────────────────────────────────

// EstadoCountRow holds one row from CoursesByEstado.
type EstadoCountRow struct {
	Estado string `gorm:"column:estado"`
	Total  int64  `gorm:"column:total"`
}

// CreatorCountRow holds one row from TopCreators.
type CreatorCountRow struct {
	Nombre string `gorm:"column:nombre"`
	Total  int64  `gorm:"column:total"`
}

// MonthCountRow holds one row from UsersPerMonth / ApprovedCoursesPerMonth.
// Month is scanned as time.Time (date_trunc result); the service converts it
// to an ISO "2006-01" string for the DTO.
type MonthCountRow struct {
	Month time.Time `gorm:"column:m"`
	Total int64     `gorm:"column:total"`
}

// CourseStatRow holds one row from CourseStats.
type CourseStatRow struct {
	ID           string  `gorm:"column:id"`
	Titulo       string  `gorm:"column:titulo"`
	Estado       string  `gorm:"column:estado"`
	Enrollments  int64   `gorm:"column:enrollments"`
	Attempts     int64   `gorm:"column:attempts"`
	ApprovalRate float64 `gorm:"column:approval_rate"`
}

// UserProgressRow holds the five progress counters for a single user.
type UserProgressRow struct {
	Enrolled       int64 `gorm:"column:enrolled"`
	Completed      int64 `gorm:"column:completed"`
	Attempts       int64 `gorm:"column:attempts"`
	PassedAttempts int64 `gorm:"column:passed_attempts"`
	Certificates   int64 `gorm:"column:certificates"`
}

// TeamMemberRow holds one employee's progress entry for a supervisor's team.
// LastAttemptDate is the ISO 8601 date (YYYY-MM-DD) of the employee's most
// recent finalized attempt, or nil when the employee has no finalized attempts.
// Spec (REQ-TEAM): "lastAttemptDate string|null".
type TeamMemberRow struct {
	EmpleadoID      string  `gorm:"column:empleado_id"`
	EmpleadoNombre  string  `gorm:"column:empleado_nombre"`
	Enrolled        int64   `gorm:"column:enrolled"`
	Completed       int64   `gorm:"column:completed"`
	LastAttemptDate *string `gorm:"column:last_attempt_date"`
}

// ── Repository interface ──────────────────────────────────────────────────────

// Repository defines the read-only data-access contract for the reporting module.
// All methods perform raw SQL aggregates against other modules' tables.
// No writes. No transactions needed (read-only).
type Repository interface {
	// ActiveUsers returns COUNT(*) of "user" WHERE activo=true.
	// Tables: "user"(activo) — migration 0001_init.
	ActiveUsers(ctx context.Context) (int64, error)

	// CoursesByEstado returns COUNT(*) GROUP BY estado for all courses.
	// The repository returns only present estados; the service fills missing
	// ones (borrador, en_revision, aprobado, rechazado) with 0.
	// Tables: course(estado) — migration 0003_add_courses.
	CoursesByEstado(ctx context.Context) ([]EstadoCountRow, error)

	// TotalAttempts returns COUNT(*) from attempt.
	// Tables: attempt — migration 0006_add_evaluations.
	TotalAttempts(ctx context.Context) (int64, error)

	// CertificatesIssued returns COUNT(*) from certificate.
	// Tables: certificate — migration 0010_add_certificates.
	CertificatesIssued(ctx context.Context) (int64, error)

	// TopCreators returns top-n creators by aprobado course count (DESC).
	// Tables: course(estado, creador_id), "user"(nombre) — migrations 0001, 0003.
	TopCreators(ctx context.Context, n int) ([]CreatorCountRow, error)

	// UsersPerMonth returns month-bucket counts of user registrations.
	// Tables: "user"(created_at) — migration 0001_init.
	UsersPerMonth(ctx context.Context) ([]MonthCountRow, error)

	// ApprovedCoursesPerMonth returns month-bucket counts for aprobado courses.
	// CRITICAL: filters AND publicado_en IS NOT NULL (publicado_en is nullable
	// per migration 0008; date_trunc(NULL) would yield a phantom NULL bucket).
	// Tables: course(publicado_en, estado) — migrations 0003, 0008_add_approvals.
	ApprovedCoursesPerMonth(ctx context.Context) ([]MonthCountRow, error)

	// CourseStats returns per-course aggregate stats (enrollments, attempts, approval rate).
	// Tables: course, enrollment(user_id), evaluation(course_id), attempt(aprobado)
	//         — migrations 0003, 0006, 0009_add_enrollment_completado.
	CourseStats(ctx context.Context) ([]CourseStatRow, error)

	// UserProgress returns the 5 progress counters for a single user.
	// Non-existent userID returns all-zero UserProgressRow (no error).
	// Tables: enrollment(user_id, completado), attempt(user_id, aprobado),
	//         certificate(user_id) — migrations 0006, 0009, 0010.
	UserProgress(ctx context.Context, userID string) (UserProgressRow, error)

	// TeamProgress returns the supervised employees for a supervisor, with
	// their enrollment/completion counts and last finalized attempt date.
	// CRITICAL: WHERE s.supervisor_id = ? — prevents cross-team data leakage.
	// completedCount uses enrollment.completado=true (spec REQ-TEAM, OQ-2).
	// lastAttemptDate is computed via correlated subquery on attempt.finalizado_en
	// to avoid join fan-out inflating the enrollment COUNT(DISTINCT) aggregates.
	// Tables: supervision(supervisor_id, empleado_id), "user"(nombre),
	//         enrollment(completado), attempt(user_id, finalizado_en)
	//         — migrations 0002_add_supervisions, 0001, 0006, 0009.
	TeamProgress(ctx context.Context, supervisorID string) ([]TeamMemberRow, error)
}

// ── gormRepository implementation ────────────────────────────────────────────

type gormRepository struct {
	db *gorm.DB
}

// New constructs a GORM-backed Repository.
func New(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

// ActiveUsers reads "user"(activo) — migration 0001_init.
func (r *gormRepository) ActiveUsers(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM "user" WHERE activo`,
	).Scan(&n).Error
	return n, err
}

// CoursesByEstado reads course(estado) — migration 0003_add_courses.
func (r *gormRepository) CoursesByEstado(ctx context.Context) ([]EstadoCountRow, error) {
	var rows []EstadoCountRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT estado, COUNT(*) AS total FROM course GROUP BY estado`,
	).Scan(&rows).Error
	return rows, err
}

// TotalAttempts reads attempt — migration 0006_add_evaluations.
func (r *gormRepository) TotalAttempts(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM attempt`,
	).Scan(&n).Error
	return n, err
}

// CertificatesIssued reads certificate — migration 0010_add_certificates.
func (r *gormRepository) CertificatesIssued(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM certificate`,
	).Scan(&n).Error
	return n, err
}

// TopCreators reads course(estado, creador_id), "user"(nombre) — migrations 0001, 0003.
func (r *gormRepository) TopCreators(ctx context.Context, n int) ([]CreatorCountRow, error) {
	var rows []CreatorCountRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT u.nombre AS nombre, COUNT(*) AS total
		 FROM course c
		 JOIN "user" u ON u.id = c.creador_id
		 WHERE c.estado = 'aprobado'
		 GROUP BY u.id, u.nombre
		 ORDER BY total DESC, u.nombre ASC
		 LIMIT ?`,
		n,
	).Scan(&rows).Error
	return rows, err
}

// UsersPerMonth reads "user"(created_at) — migration 0001_init.
func (r *gormRepository) UsersPerMonth(ctx context.Context) ([]MonthCountRow, error) {
	var rows []MonthCountRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT date_trunc('month', created_at) AS m, COUNT(*) AS total
		 FROM "user"
		 GROUP BY m
		 ORDER BY m`,
	).Scan(&rows).Error
	return rows, err
}

// ApprovedCoursesPerMonth reads course(publicado_en, estado) — migrations 0003, 0008_add_approvals.
// CRITICAL: AND publicado_en IS NOT NULL prevents NULL bucket from date_trunc(NULL).
func (r *gormRepository) ApprovedCoursesPerMonth(ctx context.Context) ([]MonthCountRow, error) {
	var rows []MonthCountRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT date_trunc('month', publicado_en) AS m, COUNT(*) AS total
		 FROM course
		 WHERE estado = 'aprobado' AND publicado_en IS NOT NULL
		 GROUP BY m
		 ORDER BY m`,
	).Scan(&rows).Error
	return rows, err
}

// CourseStats reads course, enrollment(user_id), evaluation(course_id), attempt(aprobado)
// — migrations 0003, 0006, 0009_add_enrollment_completado.
func (r *gormRepository) CourseStats(ctx context.Context) ([]CourseStatRow, error) {
	var rows []CourseStatRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT c.id AS id, c.titulo AS titulo, c.estado AS estado,
		        COUNT(DISTINCT e.user_id) AS enrollments,
		        COUNT(a.id) AS attempts,
		        COALESCE(
		            COUNT(a.id) FILTER (WHERE a.aprobado)::float / NULLIF(COUNT(a.id), 0),
		            0
		        ) AS approval_rate
		 FROM course c
		 LEFT JOIN enrollment e  ON e.course_id  = c.id
		 LEFT JOIN evaluation ev ON ev.course_id = c.id
		 LEFT JOIN attempt a     ON a.evaluation_id = ev.id
		 GROUP BY c.id, c.titulo, c.estado
		 ORDER BY c.titulo ASC`,
	).Scan(&rows).Error
	return rows, err
}

// UserProgress reads enrollment(user_id, completado), attempt(user_id, aprobado),
// certificate(user_id) — migrations 0006, 0009, 0010.
// 5 separate parameterized Scans per design §2 (D4 — separate Scans).
// Non-existent userID returns all-zero UserProgressRow.
func (r *gormRepository) UserProgress(ctx context.Context, userID string) (UserProgressRow, error) {
	var row UserProgressRow

	if err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) AS enrolled FROM enrollment WHERE user_id = ?`, userID,
	).Scan(&row.Enrolled).Error; err != nil {
		return row, err
	}

	if err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) AS completed FROM enrollment WHERE user_id = ? AND completado`, userID,
	).Scan(&row.Completed).Error; err != nil {
		return row, err
	}

	if err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) AS attempts FROM attempt WHERE user_id = ?`, userID,
	).Scan(&row.Attempts).Error; err != nil {
		return row, err
	}

	if err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) AS passed_attempts FROM attempt WHERE user_id = ? AND aprobado`, userID,
	).Scan(&row.PassedAttempts).Error; err != nil {
		return row, err
	}

	if err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) AS certificates FROM certificate WHERE user_id = ?`, userID,
	).Scan(&row.Certificates).Error; err != nil {
		return row, err
	}

	return row, nil
}

// TeamProgress reads supervision(supervisor_id, empleado_id), "user"(nombre),
// enrollment(completado), attempt(user_id, finalizado_en)
// — migrations 0002_add_supervisions, 0001, 0006, 0009.
// CRITICAL: WHERE s.supervisor_id = ? prevents cross-team data leakage.
//
// Fan-out avoidance strategy:
//   - enrolled   = COUNT(DISTINCT e.course_id) over the enrollment LEFT JOIN — safe.
//   - completed  = COUNT(DISTINCT e.course_id) FILTER (WHERE e.completado) — uses
//     enrollment.completado=true per spec REQ-TEAM/OQ-2; no certificate join needed.
//   - lastAttemptDate = correlated subquery on attempt.user_id — deliberately NOT a
//     JOIN so the enrollment aggregate rows are never multiplied by attempt rows.
func (r *gormRepository) TeamProgress(ctx context.Context, supervisorID string) ([]TeamMemberRow, error) {
	var rows []TeamMemberRow
	err := r.db.WithContext(ctx).Raw(
		`SELECT emp.id AS empleado_id,
		        emp.nombre AS empleado_nombre,
		        COUNT(DISTINCT e.course_id)                          AS enrolled,
		        COUNT(DISTINCT e.course_id) FILTER (WHERE e.completado) AS completed,
		        (SELECT MAX(a.finalizado_en)::date::text
		           FROM attempt a
		          WHERE a.user_id = emp.id
		            AND a.finalizado_en IS NOT NULL)                  AS last_attempt_date
		 FROM supervision s
		 JOIN "user" emp ON emp.id = s.empleado_id
		 LEFT JOIN enrollment e ON e.user_id = emp.id
		 WHERE s.supervisor_id = ?
		 GROUP BY emp.id, emp.nombre
		 ORDER BY emp.nombre ASC`,
		supervisorID,
	).Scan(&rows).Error
	return rows, err
}
