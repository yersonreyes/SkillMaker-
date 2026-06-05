//go:build integration

// Package approvals_test — full integration tests for the approvals module.
// Run with: make backend-test-integration
// Requires a real Postgres DB via testcontainers.
//
// Tests:
//  (a) Full submit→pending→approve flow: estado=aprobado, publicado_en non-null, approval row.
//  (b) Full submit→reject-with-comment: estado=rechazado, comentario persisted.
//  (c) Cross-module: course with video but NO evaluation → ErrEvaluationNotFound, estado UNCHANGED.
//  (d) Migration 0008 round-trip: Steps(-1) removes approval table + publicado_en column.
package approvals_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals"
	approvalsRepo "github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/repository"
	approvalsSvc "github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
	coursesRepo "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
	evalRepo "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── seed helpers ──────────────────────────────────────────────────────────────

func seedUserRow(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO "user" (id, google_sub, email, nombre, activo)
		 VALUES (?, ?, ?, ?, true)`,
		id, "sub-"+id, id+"@example.com", "Test User",
	).Error
	require.NoError(t, err, "seedUserRow: failed")
	return id
}

func seedCourseRow(t *testing.T, db *gorm.DB, creadorID, estado string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, ?, ?, ?)`,
		id, creadorID, "Integration Course", "Desc", estado,
	).Error
	require.NoError(t, err, "seedCourseRow: failed")
	return id
}

func seedSectionRow(t *testing.T, db *gorm.DB, courseID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO section (id, course_id, titulo, orden)
		 VALUES (?, ?, ?, ?)`,
		id, courseID, "Section 1", 0,
	).Error
	require.NoError(t, err, "seedSectionRow: failed")
	return id
}

func seedVideoRow(t *testing.T, db *gorm.DB, sectionID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO video (id, section_id, titulo, url, proveedor, duracion_s, orden)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, sectionID, "Video 1", "https://www.youtube.com/watch?v=test", "youtube", 120, 0,
	).Error
	require.NoError(t, err, "seedVideoRow: failed")
	return id
}

func seedCompleteEvaluation(t *testing.T, db *gorm.DB, courseID string) string {
	t.Helper()
	evalID := uuid.New().String()
	err := db.Exec(
		`INSERT INTO evaluation (id, course_id, nota_minima, intentos_max)
		 VALUES (?, ?, ?, ?)`,
		evalID, courseID, 70, 0,
	).Error
	require.NoError(t, err, "seedCompleteEvaluation: insert evaluation failed")

	// Add one question with a correct option.
	questionID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO question (id, evaluation_id, enunciado, tipo, puntaje, orden)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		questionID, evalID, "¿Cual es la respuesta?", "opcion_multiple", 1, 0,
	).Error
	require.NoError(t, err, "seedCompleteEvaluation: insert question failed")

	// Add a correct option (table is question_option per migration 0006).
	optionID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO question_option (id, question_id, texto, correcta, orden)
		 VALUES (?, ?, ?, ?, ?)`,
		optionID, questionID, "La respuesta correcta", true, 0,
	).Error
	require.NoError(t, err, "seedCompleteEvaluation: insert option failed")

	return evalID
}

// buildServices wires real GORM-backed services for integration tests.
func buildServices(db *gorm.DB) (courses.Service, evaluations.Service, approvals.Service) {
	// Use nil storage client — presign methods not exercised in integration tests.
	// Use realistic TTL and maxUploadBytes values.
	coursesSvc := courses.NewService(coursesRepo.New(db), nil, 15*time.Minute, 52_428_800)
	evalsSvc := evaluations.NewService(evalRepo.New(db), coursesSvc)
	approvalsSvc := approvals.NewService(approvalsRepo.New(db), coursesSvc, evalsSvc)
	return coursesSvc, evalsSvc, approvalsSvc
}

// ── (a) Full submit→pending→approve flow ─────────────────────────────────────

// TestApprovals_FullFlow_SubmitPendingApprove verifies the complete submit→approve flow.
// Spec: AC-1, AC-6, AC-7; REQ-SUBMIT happy, REQ-PENDING, REQ-APPROVE happy.
func TestApprovals_FullFlow_SubmitPendingApprove(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()
	_, _, svc := buildServices(db)

	creadorID := seedUserRow(t, db)
	adminID := seedUserRow(t, db)
	courseID := seedCourseRow(t, db, creadorID, "borrador")
	sectionID := seedSectionRow(t, db, courseID)
	seedVideoRow(t, db, sectionID)
	seedCompleteEvaluation(t, db, courseID)

	// 1. SubmitToReview.
	err := svc.SubmitToReview(ctx, courseID, creadorID)
	require.NoError(t, err, "SubmitToReview must succeed for valid borrador course")

	// Verify estado = en_revision.
	var estado string
	err = db.Raw(`SELECT estado FROM course WHERE id = ?`, courseID).Scan(&estado).Error
	require.NoError(t, err)
	assert.Equal(t, "en_revision", estado, "estado must be en_revision after submit")

	// 2. ListPending.
	pending, err := svc.ListPending(ctx)
	require.NoError(t, err)
	found := false
	for _, p := range pending {
		if p.ID == courseID {
			found = true
			break
		}
	}
	assert.True(t, found, "submitted course must appear in ListPending")

	// 3. Approve.
	err = svc.Approve(ctx, courseID, adminID, "Well done!")
	require.NoError(t, err, "Approve must succeed for en_revision course")

	// Verify estado = aprobado.
	err = db.Raw(`SELECT estado FROM course WHERE id = ?`, courseID).Scan(&estado).Error
	require.NoError(t, err)
	assert.Equal(t, "aprobado", estado, "estado must be aprobado after approve")

	// Verify publicado_en is non-null.
	var publicadoEn *time.Time
	err = db.Raw(`SELECT publicado_en FROM course WHERE id = ?`, courseID).Scan(&publicadoEn).Error
	require.NoError(t, err)
	assert.NotNil(t, publicadoEn, "publicado_en must be non-null after approve (AC-7)")

	// Verify approval row exists with resultado=aprobado.
	type approvalRow struct {
		Resultado  string
		Comentario string
	}
	var row approvalRow
	err = db.Raw(`SELECT resultado, comentario FROM approval WHERE course_id = ?`, courseID).Scan(&row).Error
	require.NoError(t, err)
	assert.Equal(t, "aprobado", row.Resultado, "approval row must have resultado=aprobado (AC-7)")
	assert.Equal(t, "Well done!", row.Comentario, "approval row must store the comentario")
}

// ── (b) Full submit→reject-with-comment flow ──────────────────────────────────

// TestApprovals_FullFlow_SubmitRejectWithComment verifies the complete submit→reject flow.
// Spec: AC-9, AC-10; REQ-SUBMIT happy, REQ-REJECT happy.
func TestApprovals_FullFlow_SubmitRejectWithComment(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()
	_, _, svc := buildServices(db)

	creadorID := seedUserRow(t, db)
	adminID := seedUserRow(t, db)
	courseID := seedCourseRow(t, db, creadorID, "borrador")
	sectionID := seedSectionRow(t, db, courseID)
	seedVideoRow(t, db, sectionID)
	seedCompleteEvaluation(t, db, courseID)

	// Submit.
	require.NoError(t, svc.SubmitToReview(ctx, courseID, creadorID))

	// Reject with comment.
	rejectionComment := "Needs more depth in chapter 2"
	err := svc.Reject(ctx, courseID, adminID, rejectionComment)
	require.NoError(t, err, "Reject with comment must succeed")

	// Verify estado = rechazado.
	var estado string
	err = db.Raw(`SELECT estado FROM course WHERE id = ?`, courseID).Scan(&estado).Error
	require.NoError(t, err)
	assert.Equal(t, "rechazado", estado, "estado must be rechazado after reject")

	// Verify publicado_en remains NULL (not cleared by reject).
	// Use COUNT to check NULL without scan type issues.
	var publicadoEnCount int64
	err = db.Raw(
		`SELECT COUNT(*) FROM course WHERE id = ? AND publicado_en IS NULL`,
		courseID,
	).Scan(&publicadoEnCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), publicadoEnCount, "publicado_en must remain NULL after reject")

	// Verify approval row with resultado=rechazado and correct comentario.
	type approvalRow struct {
		Resultado  string
		Comentario string
	}
	var row approvalRow
	err = db.Raw(`SELECT resultado, comentario FROM approval WHERE course_id = ?`, courseID).Scan(&row).Error
	require.NoError(t, err)
	assert.Equal(t, "rechazado", row.Resultado, "approval row must have resultado=rechazado")
	assert.Equal(t, rejectionComment, row.Comentario, "rejection comentario must be persisted")
}

// ── (c) Cross-module: course with video but NO evaluation ─────────────────────

// TestApprovals_CrossModule_NoEvaluation_Blocked verifies that a course with video
// but no evaluation cannot be submitted (ErrEvaluationNotFound surfaced).
// Spec: AC-3; REQ-SUBMIT eval-missing; REQ-XMOD XMOD-4.
func TestApprovals_CrossModule_NoEvaluation_Blocked(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()
	_, _, svc := buildServices(db)

	creadorID := seedUserRow(t, db)
	courseID := seedCourseRow(t, db, creadorID, "borrador")
	sectionID := seedSectionRow(t, db, courseID)
	seedVideoRow(t, db, sectionID)
	// NO evaluation seeded.

	err := svc.SubmitToReview(ctx, courseID, creadorID)
	require.Error(t, err, "submit without evaluation must fail")

	// The error must be from the evaluations seam (ErrEvaluationNotFound).
	// approvals surfaces evaluations' sentinel verbatim.
	assert.ErrorIs(t, err, evaluations.ErrEvaluationNotFound,
		"must surface ErrEvaluationNotFound from evaluations seam")

	// Verify estado is UNCHANGED (still borrador).
	var estado string
	dbErr := db.Raw(`SELECT estado FROM course WHERE id = ?`, courseID).Scan(&estado).Error
	require.NoError(t, dbErr)
	assert.Equal(t, "borrador", estado, "estado must remain borrador when submit is blocked")
}

// ── (d) SEC-7: Reject with empty comment leaves DB untouched ─────────────────

// TestApprovals_RejectEmptyComment_LeavesDBUntouched verifies SEC-7 at integration level.
// Spec: SEC-7, AC-9.
func TestApprovals_RejectEmptyComment_LeavesDBUntouched(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()
	_, _, svc := buildServices(db)

	creadorID := seedUserRow(t, db)
	adminID := seedUserRow(t, db)
	courseID := seedCourseRow(t, db, creadorID, "borrador")
	sectionID := seedSectionRow(t, db, courseID)
	seedVideoRow(t, db, sectionID)
	seedCompleteEvaluation(t, db, courseID)

	// Submit to get to en_revision.
	require.NoError(t, svc.SubmitToReview(ctx, courseID, creadorID))

	// Reject with empty comment — must fail.
	err := svc.Reject(ctx, courseID, adminID, "")
	assert.ErrorIs(t, err, approvalsSvc.ErrCommentRequired,
		"empty comment must return ErrCommentRequired")

	// Verify no approval row was written.
	var count int64
	dbErr := db.Raw(`SELECT COUNT(*) FROM approval WHERE course_id = ?`, courseID).Scan(&count).Error
	require.NoError(t, dbErr)
	assert.Equal(t, int64(0), count, "no approval row must be written for empty comment (SEC-7)")

	// Verify estado is UNCHANGED (still en_revision).
	var estado string
	dbErr = db.Raw(`SELECT estado FROM course WHERE id = ?`, courseID).Scan(&estado).Error
	require.NoError(t, dbErr)
	assert.Equal(t, "en_revision", estado, "estado must remain en_revision after failed reject")
}
