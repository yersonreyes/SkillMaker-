//go:build integration

// Package repository — integration tests using testcontainers + real Postgres.
// Run with: make backend-test-integration
// T-1.1: Migration 0008 round-trip test (RED — files absent).
// T-1.17: Approval CRUD integration tests.
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── helpers ────────────────────────────────────────────────────────────────────

func seedUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO "user" (id, google_sub, email, nombre, activo)
		 VALUES (?, ?, ?, ?, true)`,
		id, "sub-"+id, id+"@example.com", "Test User",
	).Error
	require.NoError(t, err, "seedUser: failed to insert user")
	return id
}

func seedCourse(t *testing.T, db *gorm.DB, creadorID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, ?, ?, 'en_revision')`,
		id, creadorID, "Test Course", "Desc",
	).Error
	require.NoError(t, err, "seedCourse: failed to insert course")
	return id
}

// ── T-1.1: Migration 0008 round-trip ─────────────────────────────────────────

// TestMigration0008RoundTrip verifies that migration 0008 up creates:
//   - publicado_en column on course
//   - approval table with constraints
//
// And that m.Steps(-1) removes both.
func TestMigration0008RoundTrip(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer teardown()

	// After SetupPostgresWithMigrate, all migrations (0001-0008) are applied.

	// Verify publicado_en column exists on course.
	var publicadoEnCount int64
	err := db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'course' AND column_name = 'publicado_en'`,
	).Scan(&publicadoEnCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), publicadoEnCount,
		"course.publicado_en must exist after 0008 up")

	// Verify approval table exists.
	var approvalTableCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = 'approval'`,
	).Scan(&approvalTableCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), approvalTableCount,
		"approval table must exist after 0008 up")

	// Verify approval.resultado CHECK constraint rejects invalid values.
	creadorID := seedUser(t, db)
	courseID := seedCourse(t, db, creadorID)
	adminID := seedUser(t, db)

	err = db.Exec(
		`INSERT INTO approval (id, course_id, admin_id, resultado, comentario)
		 VALUES (?, ?, ?, ?, ?)`,
		uuid.New().String(), courseID, adminID, "pendiente", "",
	).Error
	assert.Error(t, err, "inserting resultado='pendiente' must fail with CHECK constraint")
	assert.Contains(t, err.Error(), "23514", "expected CHECK constraint violation (23514)")

	// Verify idx_approval_course index exists.
	var idxCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM pg_indexes WHERE indexname = 'idx_approval_course'`,
	).Scan(&idxCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), idxCount, "idx_approval_course index must exist after 0008 up")

	// Roll back 2 steps (0009 first, then 0008).
	// NOTE (C2.4): migration 0009 was added; -1 now rolls back 0009 only,
	// so we need -2 to reach the state where 0008 has been reversed.
	err = m.Steps(-2)
	require.NoError(t, err, "m.Steps(-2) must roll back 0009+0008 without error")

	// Verify publicado_en is gone after 0008 down.
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.columns
		 WHERE table_name = 'course' AND column_name = 'publicado_en'`,
	).Scan(&publicadoEnCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), publicadoEnCount,
		"course.publicado_en must be removed after 0008 down")

	// Verify approval table is gone after 0008 down.
	err = db.Raw(
		`SELECT COUNT(*) FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_name = 'approval'`,
	).Scan(&approvalTableCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), approvalTableCount,
		"approval table must be removed after 0008 down")
}

// ── T-1.17: Approval repository CRUD ─────────────────────────────────────────

// TestApprovalCreate_PersistsRow verifies Create persists a row fetchable by ListByCourse.
func TestApprovalCreate_PersistsRow(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	creadorID := seedUser(t, db)
	courseID := seedCourse(t, db, creadorID)
	adminID := seedUser(t, db)

	repo := repository.New(db)

	a := &domain.Approval{
		ID:         uuid.New().String(),
		CourseID:   courseID,
		AdminID:    adminID,
		Resultado:  "aprobado",
		Comentario: "Looks great!",
		ResueltoEn: time.Now().UTC(),
	}
	err := repo.Create(context.Background(), a)
	require.NoError(t, err, "Create must succeed")

	rows, err := repo.ListByCourse(context.Background(), courseID)
	require.NoError(t, err)
	require.Len(t, rows, 1, "must have exactly 1 approval row")
	assert.Equal(t, a.ID, rows[0].ID)
	assert.Equal(t, "aprobado", rows[0].Resultado)
	assert.Equal(t, "Looks great!", rows[0].Comentario)
}

// TestApprovalListByCourse_OrderedByResueltoEnDESC verifies ordering DESC.
func TestApprovalListByCourse_OrderedByResueltoEnDESC(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	creadorID := seedUser(t, db)
	courseID := seedCourse(t, db, creadorID)
	adminID := seedUser(t, db)

	repo := repository.New(db)

	earlier := time.Now().UTC().Add(-5 * time.Minute)
	later := time.Now().UTC()

	a1 := &domain.Approval{
		ID: uuid.New().String(), CourseID: courseID, AdminID: adminID,
		Resultado: "rechazado", Comentario: "Needs work", ResueltoEn: earlier,
	}
	a2 := &domain.Approval{
		ID: uuid.New().String(), CourseID: courseID, AdminID: adminID,
		Resultado: "aprobado", Comentario: "Approved", ResueltoEn: later,
	}

	require.NoError(t, repo.Create(context.Background(), a1))
	require.NoError(t, repo.Create(context.Background(), a2))

	rows, err := repo.ListByCourse(context.Background(), courseID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	// DESC order: later first
	assert.Equal(t, a2.ID, rows[0].ID, "most recent must be first (DESC)")
	assert.Equal(t, a1.ID, rows[1].ID, "older must be second (DESC)")
}

// TestApprovalListByCourse_EmptyList verifies empty list when no rows.
func TestApprovalListByCourse_EmptyList(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	creadorID := seedUser(t, db)
	courseID := seedCourse(t, db, creadorID)

	repo := repository.New(db)

	rows, err := repo.ListByCourse(context.Background(), courseID)
	require.NoError(t, err)
	assert.Empty(t, rows, "must return empty slice when no approvals exist")
}
