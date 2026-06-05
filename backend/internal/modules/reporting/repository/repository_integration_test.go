//go:build integration

// Package repository_test — integration tests for the reporting repository.
// Tests aggregate SQL queries against a real Postgres container.
// Run with: make backend-test-integration
//
// STRICT TDD: every test was written RED (methods absent) before GREEN (implementation).
package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)


// ── Reporting-local seed helpers (D6 — do NOT touch testutil) ────────────────
// All helpers are self-contained INSERT helpers that return the created ID.
// Wrong seeds silently skew every aggregate assertion, so we keep them minimal
// and explicit with explicit column lists.

// seedUserWithRole inserts a user + user_role JOIN row.
// roleName must be one of: alumno, creador, supervisor, administrador.
func seedUserWithRole(t *testing.T, db *gorm.DB, roleName string) string {
	t.Helper()
	id := uuid.New().String()
	// Insert the user row (activo=true by default).
	err := db.Exec(
		`INSERT INTO "user" (id, google_sub, email, nombre, activo)
		 VALUES (?, ?, ?, ?, true)`,
		id, "sub-"+id, id+"@example.com", "User "+id[:8],
	).Error
	require.NoError(t, err, "seedUserWithRole: failed to insert user")

	// Link to role by nombre.
	err = db.Exec(
		`INSERT INTO user_role (user_id, role_id)
		 SELECT ?, id FROM role WHERE nombre = ?`,
		id, roleName,
	).Error
	require.NoError(t, err, "seedUserWithRole: failed to insert user_role")
	return id
}

// seedUserInactive inserts an inactive user (activo=false), no role needed.
func seedUserInactive(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO "user" (id, google_sub, email, nombre, activo)
		 VALUES (?, ?, ?, ?, false)`,
		id, "sub-"+id, id+"@example.com", "InactiveUser "+id[:8],
	).Error
	require.NoError(t, err, "seedUserInactive: failed to insert user")
	return id
}

// seedCourse inserts a course. For estado='aprobado', publicado_en is stamped with now().
// For other estados, publicado_en is left NULL (nullable per migration 0008).
func seedCourse(t *testing.T, db *gorm.DB, creadorID, estado string) string {
	t.Helper()
	id := uuid.New().String()

	if estado == "aprobado" {
		// Stamp publicado_en for aprobado courses (avoids NULL bucket in ApprovedCoursesPerMonth).
		err := db.Exec(
			`INSERT INTO course (id, creador_id, titulo, descripcion, estado, publicado_en)
			 VALUES (?, ?, ?, ?, ?, NOW())`,
			id, creadorID, "Course "+id[:8], "Desc", estado,
		).Error
		require.NoError(t, err, "seedCourse: failed to insert aprobado course")
	} else {
		err := db.Exec(
			`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
			 VALUES (?, ?, ?, ?, ?)`,
			id, creadorID, "Course "+id[:8], "Desc", estado,
		).Error
		require.NoError(t, err, "seedCourse: failed to insert course")
	}
	return id
}

// seedEnrollment inserts an enrollment row. completado controls enrollment.completado column.
func seedEnrollment(t *testing.T, db *gorm.DB, userID, courseID string, completado bool) {
	t.Helper()
	err := db.Exec(
		`INSERT INTO enrollment (id, user_id, course_id, inscrito_en, completado)
		 VALUES (?, ?, ?, NOW(), ?)`,
		uuid.New().String(), userID, courseID, completado,
	).Error
	require.NoError(t, err, "seedEnrollment: failed to insert enrollment")
}

// seedEvaluation inserts an evaluation for a course. Returns the evaluation ID.
func seedEvaluation(t *testing.T, db *gorm.DB, courseID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO evaluation (id, course_id, nota_minima, intentos_max)
		 VALUES (?, ?, 60, 3)`,
		id, courseID,
	).Error
	require.NoError(t, err, "seedEvaluation: failed to insert evaluation")
	return id
}

// seedAttempt inserts an attempt row. numero is auto-incremented per (user, eval).
// aprobado controls whether the attempt passed.
func seedAttempt(t *testing.T, db *gorm.DB, userID, evalID string, aprobado bool) {
	t.Helper()
	// Auto-increment numero per (user_id, evaluation_id).
	var numero int
	err := db.Raw(
		`SELECT COALESCE(MAX(numero), 0) + 1 FROM attempt WHERE user_id = ? AND evaluation_id = ?`,
		userID, evalID,
	).Scan(&numero).Error
	require.NoError(t, err, "seedAttempt: failed to compute numero")

	err = db.Exec(
		`INSERT INTO attempt (id, user_id, evaluation_id, numero, puntaje, aprobado, iniciado_en, finalizado_en)
		 VALUES (?, ?, ?, ?, 80, ?, NOW(), NOW())`,
		uuid.New().String(), userID, evalID, numero, aprobado,
	).Error
	require.NoError(t, err, "seedAttempt: failed to insert attempt")
}

// seedCertificate inserts a certificate row.
func seedCertificate(t *testing.T, db *gorm.DB, userID, courseID string) {
	t.Helper()
	err := db.Exec(
		`INSERT INTO certificate (id, user_id, course_id, codigo, storage_key, emitido_en)
		 VALUES (?, ?, ?, ?, ?, NOW())`,
		uuid.New().String(), userID, courseID,
		"CERT-"+uuid.New().String()[:8],
		"certificates/"+uuid.New().String()+".pdf",
	).Error
	require.NoError(t, err, "seedCertificate: failed to insert certificate")
}

// seedSupervision inserts a supervision row linking supervisor → employee.
func seedSupervision(t *testing.T, db *gorm.DB, supervisorID, empleadoID string) {
	t.Helper()
	err := db.Exec(
		`INSERT INTO supervision (id, supervisor_id, empleado_id, creado_en)
		 VALUES (?, ?, ?, NOW())`,
		uuid.New().String(), supervisorID, empleadoID,
	).Error
	require.NoError(t, err, "seedSupervision: failed to insert supervision")
}

// ── T2.1: TestActiveUsers ─────────────────────────────────────────────────────

// TestActiveUsers seeds 3 active + 1 inactive user and asserts ActiveUsers == 3.
func TestActiveUsers(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	// Seed 3 active users (with any role).
	seedUserWithRole(t, db, "alumno")
	seedUserWithRole(t, db, "creador")
	seedUserWithRole(t, db, "supervisor")
	// Seed 1 inactive user.
	seedUserInactive(t, db)

	count, err := repo.ActiveUsers(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count, "ActiveUsers must return only activo=true users")
}

// ── T2.1: TestCoursesByEstado ─────────────────────────────────────────────────

// TestCoursesByEstado seeds 2 aprobado + 1 borrador courses.
// Repo returns only present estados (service fills missing ones with 0).
func TestCoursesByEstado(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUserWithRole(t, db, "creador")
	seedCourse(t, db, creadorID, "aprobado")
	seedCourse(t, db, creadorID, "aprobado")
	seedCourse(t, db, creadorID, "borrador")

	rows, err := repo.CoursesByEstado(ctx)
	require.NoError(t, err)

	// Build a map for easy lookup.
	m := make(map[string]int64)
	for _, r := range rows {
		m[r.Estado] = r.Total
	}

	assert.Equal(t, int64(2), m["aprobado"], "must have 2 aprobado courses")
	assert.Equal(t, int64(1), m["borrador"], "must have 1 borrador course")
	// en_revision and rechazado should be absent (repo returns only present estados).
	_, hasEnRevision := m["en_revision"]
	assert.False(t, hasEnRevision, "en_revision must be absent when no courses exist")
}

// ── T2.1: TestTotalAttempts ───────────────────────────────────────────────────

func TestTotalAttempts(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	count0, err := repo.TotalAttempts(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count0, "must start at 0")

	creadorID := seedUserWithRole(t, db, "creador")
	studentID := seedUserWithRole(t, db, "alumno")
	courseID := seedCourse(t, db, creadorID, "aprobado")
	evalID := seedEvaluation(t, db, courseID)
	seedAttempt(t, db, studentID, evalID, true)
	seedAttempt(t, db, studentID, evalID, false)

	count2, err := repo.TotalAttempts(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count2)
}

// ── T2.1: TestCertificatesIssued ─────────────────────────────────────────────

func TestCertificatesIssued(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	count0, err := repo.CertificatesIssued(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count0)

	creadorID := seedUserWithRole(t, db, "creador")
	studentID := seedUserWithRole(t, db, "alumno")
	courseID := seedCourse(t, db, creadorID, "aprobado")
	seedCertificate(t, db, studentID, courseID)

	count1, err := repo.CertificatesIssued(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count1)
}

// ── T2.3: TestTopCreators ─────────────────────────────────────────────────────

// TestTopCreators seeds creator A=3 aprobado courses, creator B=1. Asserts A is first.
func TestTopCreators(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creatorA := seedUserWithRole(t, db, "creador")
	creatorB := seedUserWithRole(t, db, "creador")

	// Creator A: 3 aprobado courses.
	seedCourse(t, db, creatorA, "aprobado")
	seedCourse(t, db, creatorA, "aprobado")
	seedCourse(t, db, creatorA, "aprobado")
	// Creator B: 1 aprobado course.
	seedCourse(t, db, creatorB, "aprobado")

	rows, err := repo.TopCreators(ctx, 10)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 2, "must return at least 2 creators")

	// A must appear before B (DESC by total).
	var idxA, idxB int = -1, -1
	for i, r := range rows {
		if r.Total == 3 {
			idxA = i
		}
		if r.Total == 1 {
			idxB = i
		}
	}
	assert.GreaterOrEqual(t, idxA, 0, "creator with 3 courses must be in results")
	assert.GreaterOrEqual(t, idxB, 0, "creator with 1 course must be in results")
	if idxA >= 0 && idxB >= 0 {
		assert.Less(t, idxA, idxB, "creator A (3 courses) must appear before creator B (1 course)")
	}
}

// ── T2.3: TestUsersPerMonth ───────────────────────────────────────────────────

// TestUsersPerMonth verifies the month bucketing. Seeds users in the same month
// and confirms at least 1 bucket is returned with a non-zero count.
func TestUsersPerMonth(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	// Seed at least 2 users — they'll share the current month bucket.
	seedUserWithRole(t, db, "alumno")
	seedUserWithRole(t, db, "alumno")

	rows, err := repo.UsersPerMonth(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, rows, "UsersPerMonth must return at least 1 bucket")

	// All totals must be > 0.
	for _, r := range rows {
		assert.Greater(t, r.Total, int64(0), "every bucket must have count > 0")
	}
}

// ── T2.3: TestApprovedCoursesPerMonth ────────────────────────────────────────

// TestApprovedCoursesPerMonth seeds 2 aprobado courses (publicado_en set) and
// 1 borrador (publicado_en NULL). Must return only non-NULL publicado_en rows.
func TestApprovedCoursesPerMonth(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUserWithRole(t, db, "creador")
	// These get publicado_en stamped by seedCourse.
	seedCourse(t, db, creadorID, "aprobado")
	seedCourse(t, db, creadorID, "aprobado")
	// Borrador: publicado_en is NULL.
	seedCourse(t, db, creadorID, "borrador")

	rows, err := repo.ApprovedCoursesPerMonth(ctx)
	require.NoError(t, err)
	// Must return at least 1 bucket (the aprobado ones share current month).
	require.NotEmpty(t, rows, "ApprovedCoursesPerMonth must return at least 1 bucket")

	// Sum must equal 2 (the 2 aprobado courses).
	var total int64
	for _, r := range rows {
		total += r.Total
	}
	assert.Equal(t, int64(2), total, "only aprobado+non-null publicado_en courses must be counted")
}

// ── T2.5: TestCourseStats_ApprovalRate ───────────────────────────────────────

// TestCourseStats_ApprovalRate seeds ONE course, 3 attempts (2 aprobado).
// approval_rate must be ≈ 0.6667 (assert InDelta 0.001).
func TestCourseStats_ApprovalRate(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUserWithRole(t, db, "creador")
	studentID := seedUserWithRole(t, db, "alumno")

	courseID := seedCourse(t, db, creadorID, "aprobado")
	seedEnrollment(t, db, studentID, courseID, false)
	evalID := seedEvaluation(t, db, courseID)

	// 3 attempts: 2 aprobado, 1 not.
	seedAttempt(t, db, studentID, evalID, true)
	seedAttempt(t, db, studentID, evalID, true)
	seedAttempt(t, db, studentID, evalID, false)

	rows, err := repo.CourseStats(ctx)
	require.NoError(t, err)

	// Find our course.
	var found *repository.CourseStatRow
	for i := range rows {
		if rows[i].ID == courseID {
			found = &rows[i]
			break
		}
	}
	require.NotNil(t, found, "course must appear in CourseStats")
	assert.Equal(t, int64(1), found.Enrollments, "enrollments must be 1")
	assert.Equal(t, int64(3), found.Attempts, "attempts must be 3")
	assert.InDelta(t, 0.6667, found.ApprovalRate, 0.001, "approval_rate must be ≈ 2/3")
}

// TestCourseStats_NoAttempts verifies that a course with enrollments but 0 attempts
// has approvalRate=0.0 (no division by zero).
func TestCourseStats_NoAttempts(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUserWithRole(t, db, "creador")
	studentID := seedUserWithRole(t, db, "alumno")

	courseID := seedCourse(t, db, creadorID, "borrador")
	seedEnrollment(t, db, studentID, courseID, false)

	rows, err := repo.CourseStats(ctx)
	require.NoError(t, err)

	var found *repository.CourseStatRow
	for i := range rows {
		if rows[i].ID == courseID {
			found = &rows[i]
			break
		}
	}
	require.NotNil(t, found, "course must appear even with 0 attempts")
	assert.Equal(t, float64(0), found.ApprovalRate, "approvalRate must be 0.0 with no attempts")
}

// ── T2.7: TestUserProgress ────────────────────────────────────────────────────

// TestUserProgress seeds enrollments(2, 1 completed), attempts(3, 2 passed), certs(1).
// Asserts each field matches.
func TestUserProgress(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUserWithRole(t, db, "creador")
	studentID := seedUserWithRole(t, db, "alumno")

	course1 := seedCourse(t, db, creadorID, "aprobado")
	course2 := seedCourse(t, db, creadorID, "aprobado")

	// 2 enrollments: 1 completed, 1 not.
	seedEnrollment(t, db, studentID, course1, true)
	seedEnrollment(t, db, studentID, course2, false)

	// 3 attempts: 2 passed.
	eval1 := seedEvaluation(t, db, course1)
	seedAttempt(t, db, studentID, eval1, true)
	seedAttempt(t, db, studentID, eval1, true)
	seedAttempt(t, db, studentID, eval1, false)

	// 1 certificate.
	seedCertificate(t, db, studentID, course1)

	row, err := repo.UserProgress(ctx, studentID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), row.Enrolled, "Enrolled must be 2")
	assert.Equal(t, int64(1), row.Completed, "Completed must be 1")
	assert.Equal(t, int64(3), row.Attempts, "Attempts must be 3")
	assert.Equal(t, int64(2), row.PassedAttempts, "PassedAttempts must be 2")
	assert.Equal(t, int64(1), row.Certificates, "Certificates must be 1")
}

// TestUserProgress_NonExistentID verifies that a valid UUID with no data returns all zeros.
func TestUserProgress_NonExistentID(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	row, err := repo.UserProgress(ctx, uuid.New().String())
	require.NoError(t, err, "non-existent user must not error")
	assert.Equal(t, int64(0), row.Enrolled, "all zeros for non-existent user")
	assert.Equal(t, int64(0), row.Completed)
	assert.Equal(t, int64(0), row.Attempts)
	assert.Equal(t, int64(0), row.PassedAttempts)
	assert.Equal(t, int64(0), row.Certificates)
}

// ── C1: TestTeamProgress_LastAttemptDate (CRITICAL spec gap fix) ─────────────

// TestTeamProgress_LastAttemptDate verifies the lastAttemptDate field in TeamMemberRow.
//
// Spec (REQ-TEAM): "lastAttemptDate string|null — ISO 8601 date of most recent attempt;
// null if none". This is a REQUIRED field; its absence was flagged as CRITICAL in verify #333.
//
// Scenario A: employee WITH a finalized attempt → LastAttemptDate is set (non-nil, non-empty).
// Scenario B: employee with NO attempts → LastAttemptDate is nil.
//
// RED phase: this test will FAIL until LastAttemptDate is added to TeamMemberRow +
// the TeamProgress SQL computes MAX(finalizado_en)::date::text via correlated subquery.
func TestTeamProgress_LastAttemptDate(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	supervisorID := seedUserWithRole(t, db, "supervisor")

	// Employee A: has a finalized attempt.
	empWithAttempt := seedUserWithRole(t, db, "alumno")
	seedSupervision(t, db, supervisorID, empWithAttempt)
	creadorID := seedUserWithRole(t, db, "creador")
	courseID := seedCourse(t, db, creadorID, "aprobado")
	evalID := seedEvaluation(t, db, courseID)
	// Seed 1 finalized attempt (finalizado_en = NOW() via seedAttempt).
	seedAttempt(t, db, empWithAttempt, evalID, true)

	// Employee B: no attempts at all.
	empNoAttempt := seedUserWithRole(t, db, "alumno")
	seedSupervision(t, db, supervisorID, empNoAttempt)

	rows, err := repo.TeamProgress(ctx, supervisorID)
	require.NoError(t, err)
	require.Len(t, rows, 2, "supervisor must see exactly 2 employees")

	rowByID := make(map[string]repository.TeamMemberRow)
	for _, r := range rows {
		rowByID[r.EmpleadoID] = r
	}

	// Scenario A: employee WITH a finalized attempt → lastAttemptDate is non-nil and non-empty.
	rowA, ok := rowByID[empWithAttempt]
	require.True(t, ok, "employee with attempt must be in results")
	require.NotNil(t, rowA.LastAttemptDate, "employee with finalized attempt: LastAttemptDate must be non-nil")
	assert.NotEmpty(t, *rowA.LastAttemptDate, "employee with finalized attempt: LastAttemptDate must be non-empty")

	// Scenario B: employee with NO attempts → lastAttemptDate is nil.
	rowB, ok := rowByID[empNoAttempt]
	require.True(t, ok, "employee without attempt must be in results")
	assert.Nil(t, rowB.LastAttemptDate, "employee with no attempts: LastAttemptDate must be nil")
}

// ── W2: TestTeamProgress_CompletedViaEnrollment (spec compliance fix) ────────

// TestTeamProgress_CompletedViaEnrollment verifies that completedCount uses
// enrollment.completado=true, NOT certificate count.
//
// Spec (REQ-TEAM, OQ-2): "completedCount int // via enrollment.completado=true".
// The original implementation used COUNT(DISTINCT cert.course_id) which is wrong.
//
// Seed: employee with 2 enrollments (1 completado=true, 1 completado=false) and
// 0 certificates. Expected: completedCount=1, enrolledCount=2.
//
// RED phase: this test will FAIL until the TeamProgress SQL is fixed to use
// enrollment.completado instead of certificate count.
func TestTeamProgress_CompletedViaEnrollment(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	supervisorID := seedUserWithRole(t, db, "supervisor")
	empID := seedUserWithRole(t, db, "alumno")
	seedSupervision(t, db, supervisorID, empID)

	creadorID := seedUserWithRole(t, db, "creador")
	course1 := seedCourse(t, db, creadorID, "aprobado")
	course2 := seedCourse(t, db, creadorID, "aprobado")

	// 2 enrollments: 1 completado=true, 1 completado=false. NO certificates.
	seedEnrollment(t, db, empID, course1, true)
	seedEnrollment(t, db, empID, course2, false)

	rows, err := repo.TeamProgress(ctx, supervisorID)
	require.NoError(t, err)
	require.Len(t, rows, 1, "supervisor must see exactly 1 employee")

	row := rows[0]
	assert.Equal(t, int64(2), row.Enrolled, "enrolledCount must be 2")
	assert.Equal(t, int64(1), row.Completed, "completedCount must be 1 (via enrollment.completado, NOT certificate count)")
}

// ── W1: TestApprovedCoursesPerMonth_NullPublicadoEn (NULL sentinel fix) ──────

// TestApprovedCoursesPerMonth_NullPublicadoEn makes the IS NOT NULL filter non-vacuous.
//
// The verify report (P5 probe) found that removing the IS NOT NULL filter from
// ApprovedCoursesPerMonth did NOT cause any test to fail because no test seeded an
// aprobado course with publicado_en = NULL. This test closes that gap.
//
// Scenario: seed 2 aprobado courses with publicado_en stamped (via seedCourse) +
// 1 aprobado course with publicado_en forced to NULL via UPDATE. Assert total = 2.
//
// RED behavior if IS NOT NULL is removed: the NULL-date course would create a
// NULL-bucket row (or be counted anyway), making total = 3, and this test FAILS.
func TestApprovedCoursesPerMonth_NullPublicadoEn(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	creadorID := seedUserWithRole(t, db, "creador")

	// 2 aprobado courses with a real publicado_en (set by seedCourse).
	seedCourse(t, db, creadorID, "aprobado")
	seedCourse(t, db, creadorID, "aprobado")

	// 1 aprobado course with publicado_en forced to NULL (simulates a bad-state course).
	nullCourseID := seedCourse(t, db, creadorID, "aprobado")
	err := db.Exec(`UPDATE course SET publicado_en = NULL WHERE id = ?`, nullCourseID).Error
	require.NoError(t, err, "forcing publicado_en = NULL must succeed")

	rows, err := repo.ApprovedCoursesPerMonth(ctx)
	require.NoError(t, err)

	// Only the 2 courses WITH publicado_en set should be counted.
	var total int64
	for _, r := range rows {
		total += r.Total
	}
	assert.Equal(t, int64(2), total,
		"aprobado course with publicado_en=NULL must be excluded by the IS NOT NULL filter")
}

// ── T2.9: TestTeamProgress_Scoping (CRITICAL — adversarial) ──────────────────

// TestTeamProgress_Scoping seeds supervisor A → [a1, a2] and supervisor B → [b1].
// Asserts TeamProgress(A) returns exactly {a1, a2} and NEVER b1.
// Asserts TeamProgress(B) returns only {b1}.
func TestTeamProgress_Scoping(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	repo := repository.New(db)
	ctx := context.Background()

	supervisorA := seedUserWithRole(t, db, "supervisor")
	supervisorB := seedUserWithRole(t, db, "supervisor")

	a1 := seedUserWithRole(t, db, "alumno")
	a2 := seedUserWithRole(t, db, "alumno")
	b1 := seedUserWithRole(t, db, "alumno")

	seedSupervision(t, db, supervisorA, a1)
	seedSupervision(t, db, supervisorA, a2)
	seedSupervision(t, db, supervisorB, b1)

	// TeamProgress for supervisor A.
	rowsA, err := repo.TeamProgress(ctx, supervisorA)
	require.NoError(t, err)

	idsA := make(map[string]bool)
	for _, r := range rowsA {
		idsA[r.EmpleadoID] = true
	}
	assert.True(t, idsA[a1], "supervisor A must see employee a1")
	assert.True(t, idsA[a2], "supervisor A must see employee a2")
	assert.False(t, idsA[b1], "supervisor A must NEVER see employee b1 (belongs to B)")
	assert.Len(t, rowsA, 2, "supervisor A must see exactly 2 employees")

	// TeamProgress for supervisor B.
	rowsB, err := repo.TeamProgress(ctx, supervisorB)
	require.NoError(t, err)
	assert.Len(t, rowsB, 1, "supervisor B must see exactly 1 employee")
	if len(rowsB) > 0 {
		assert.Equal(t, b1, rowsB[0].EmpleadoID, "supervisor B must see employee b1")
	}
}
