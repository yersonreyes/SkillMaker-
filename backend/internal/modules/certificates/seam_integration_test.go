//go:build integration

// Package certificates — seam end-to-end integration tests (C5.1).
//
// TestSeam_PassingAttempt_CreatesCertificateAndBadge verifies the full cross-module seam:
//   - student enrolled → submits passing attempt → certificate row created + badge awarded
//
// TestSeam_CertFailure_NonFatal verifies:
//   - IssueOnPass returning error (via storage failure) → attempt still submits with aprobado=true
//
// Uses real Postgres (testcontainers). Wires coursesSvc + evaluationsSvc + certsSvc directly.
//
// Spec: REQ-ISSUE / AC-6 / REQ-BADGE-EVAL (first cert → badge).
package certificates_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates"
	certsRepo "github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/repository"
	certsService "github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/service"
	coursesRepo "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	coursesSvc "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	evalsRepo "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
	evalsSvc "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── noopStorage ────────────────────────────────────────────────────────────────

type noopStorage struct {
	putObjectErr error // if set, PutObject returns this error
	putCalls     []string
}

func (n *noopStorage) PresignPutURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "https://noop-presign-put", nil
}
func (n *noopStorage) PresignGetURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://noop-presign-get/" + key, nil
}
func (n *noopStorage) Delete(_ context.Context, _ string) error { return nil }
func (n *noopStorage) Ping(_ context.Context) error             { return nil }
func (n *noopStorage) PutObject(_ context.Context, key string, _ io.Reader, _ int64, _ string) error {
	n.putCalls = append(n.putCalls, key)
	if n.putObjectErr != nil {
		return n.putObjectErr
	}
	return nil
}

var _ storage.Client = &noopStorage{}

// ── helpers ────────────────────────────────────────────────────────────────────

func seedUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO "user" (id, google_sub, email, nombre, activo)
		 VALUES (?, ?, ?, ?, true)`,
		id, "sub-"+id, id+"@example.com", "Test User "+id[:8],
	).Error
	require.NoError(t, err, "seedUser: failed to insert user")
	return id
}

func seedCourseAprobado(t *testing.T, db *gorm.DB, creadorID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, ?, ?, 'aprobado')`,
		id, creadorID, "Seam Test Course "+id[:8], "Desc",
	).Error
	require.NoError(t, err, "seedCourseAprobado: failed to insert course")
	return id
}

func seedEvaluationWithPassingSetup(t *testing.T, db *gorm.DB, courseID string) (evalID, optionID string) {
	t.Helper()
	evalID = uuid.New().String()
	require.NoError(t, db.Exec(
		`INSERT INTO evaluation (id, course_id, nota_minima, intentos_max) VALUES (?, ?, 70, 5)`,
		evalID, courseID,
	).Error)
	qID := uuid.New().String()
	require.NoError(t, db.Exec(
		`INSERT INTO question (id, evaluation_id, enunciado, tipo, puntaje, orden)
		 VALUES (?, ?, ?, 'verdadero_falso', 100, 0)`,
		qID, evalID, "Is Go awesome?",
	).Error)
	optionID = uuid.New().String()
	require.NoError(t, db.Exec(
		`INSERT INTO question_option (id, question_id, texto, correcta, orden) VALUES (?, ?, ?, true, 0)`,
		optionID, qID, "Verdadero",
	).Error)
	wrongID := uuid.New().String()
	require.NoError(t, db.Exec(
		`INSERT INTO question_option (id, question_id, texto, correcta, orden) VALUES (?, ?, ?, false, 1)`,
		wrongID, qID, "Falso",
	).Error)
	return evalID, optionID
}

// userNameAdapterSeam adapts a real DB user lookup for tests.
type userNameAdapterSeam struct{ db *gorm.DB }

func (a *userNameAdapterSeam) GetUserNombre(_ context.Context, userID string) (string, error) {
	var nombre string
	err := a.db.Raw(`SELECT nombre FROM "user" WHERE id = ?`, userID).Scan(&nombre).Error
	return nombre, err
}

// courseTituloAdapterSeam adapts a real coursesSvc for tests.
type courseTituloAdapterSeam struct{ svc coursesSvc.Service }

func (a *courseTituloAdapterSeam) GetCourseTitulo(ctx context.Context, courseID string) (string, error) {
	return a.svc.GetCourseTitulo(ctx, courseID)
}

// ── TestSeam_PassingAttempt_CreatesCertificateAndBadge ────────────────────────

// TestSeam_PassingAttempt_CreatesCertificateAndBadge is the primary seam e2e test.
// It proves: passing attempt → certificate row created + "Primer curso completado" badge awarded.
// Spec: REQ-ISSUE / REQ-BADGE-EVAL / AC-4.
func TestSeam_PassingAttempt_CreatesCertificateAndBadge(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()
	store := &noopStorage{}

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCourseAprobado(t, db, creadorID)

	// Wire real services.
	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, store, 15*time.Minute, 52_428_800)

	crRepo := certsRepo.New(db)
	crSvc := certsService.New(crRepo, store, &userNameAdapterSeam{db}, &courseTituloAdapterSeam{cSvc}, 15*time.Minute)

	// Enroll student.
	require.NoError(t, cSvc.Enroll(ctx, studentID, courseID))

	// Setup evaluation.
	evalID, optionID := seedEvaluationWithPassingSetup(t, db, courseID)

	// Wire evaluationsSvc WITH CertificateIssuer seam.
	eRepo := evalsRepo.New(db)
	eSvc := evalsSvc.New(eRepo, cSvc,
		evalsSvc.WithEnrollmentCompleter(cSvc),
		evalsSvc.WithCertificateIssuer(crSvc),
	)

	// Start attempt.
	attempt, err := eSvc.StartAttempt(ctx, evalID, studentID)
	require.NoError(t, err)

	state, err := eSvc.GetAttempt(ctx, attempt.ID, studentID)
	require.NoError(t, err)
	require.NotEmpty(t, state.Questions)

	qID := state.Questions[0].ID
	require.NoError(t, eSvc.SaveAnswer(ctx, attempt.ID, studentID, qID, optionID))

	// Submit — must trigger IssueOnPass seam.
	result, err := eSvc.SubmitAttempt(ctx, attempt.ID, studentID)
	require.NoError(t, err)
	assert.True(t, result.Aprobado, "attempt must be aprobado")

	// ── Verify certificate row was created ───────────────────────────────────
	var certCount int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM certificate WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&certCount).Error)
	assert.Equal(t, int64(1), certCount,
		"[REQ-ISSUE / AC-1] exactly 1 certificate row must be created after passing attempt")

	// ── Verify PutObject was called ──────────────────────────────────────────
	assert.Len(t, store.putCalls, 1, "PutObject must be called once for the certificate PDF")
	assert.Contains(t, store.putCalls[0], "certificates/", "storage key must be in certificates/ prefix")

	// ── Verify "Primer curso completado" badge was awarded ───────────────────
	var badgeCount int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM user_badge ub
		 JOIN badge b ON b.id = ub.badge_id
		 WHERE ub.user_id = ? AND b.nombre = 'Primer curso completado'`,
		studentID,
	).Scan(&badgeCount).Error)
	assert.Equal(t, int64(1), badgeCount,
		"[REQ-BADGE-EVAL / AC-4] 'Primer curso completado' badge must be awarded after first certificate")

	// ── Verify idempotency: submit same course again → still 1 cert ─────────
	// IssueOnPass must be idempotent — second call should be a no-op.
	require.NoError(t, crSvc.IssueOnPass(ctx, studentID, courseID))
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM certificate WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&certCount).Error)
	assert.Equal(t, int64(1), certCount,
		"[REQ-ISSUE idempotency] second IssueOnPass must not create a duplicate cert row")
}

// ── TestSeam_CertFailure_NonFatal ─────────────────────────────────────────────

// TestSeam_CertFailure_NonFatal proves that a storage error in IssueOnPass does NOT
// block the student's attempt submission. The attempt is stored with aprobado=true.
// Spec: REQ-ISSUE scenario "storage failure is non-fatal to attempt" / AC-6.
func TestSeam_CertFailure_NonFatal(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	// Storage that fails on PutObject.
	failStore := &noopStorage{putObjectErr: errors.New("storage is down")}

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCourseAprobado(t, db, creadorID)

	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, failStore, 15*time.Minute, 52_428_800)

	crRepo := certsRepo.New(db)
	crSvc := certsService.New(crRepo, failStore, &userNameAdapterSeam{db}, &courseTituloAdapterSeam{cSvc}, 15*time.Minute)

	require.NoError(t, cSvc.Enroll(ctx, studentID, courseID))
	evalID, optionID := seedEvaluationWithPassingSetup(t, db, courseID)

	eRepo := evalsRepo.New(db)
	eSvc := evalsSvc.New(eRepo, cSvc,
		evalsSvc.WithEnrollmentCompleter(cSvc),
		evalsSvc.WithCertificateIssuer(crSvc),
	)

	attempt, err := eSvc.StartAttempt(ctx, evalID, studentID)
	require.NoError(t, err)

	state, err := eSvc.GetAttempt(ctx, attempt.ID, studentID)
	require.NoError(t, err)
	require.NotEmpty(t, state.Questions)

	qID := state.Questions[0].ID
	require.NoError(t, eSvc.SaveAnswer(ctx, attempt.ID, studentID, qID, optionID))

	// Submit — IssueOnPass fails (storage error) but attempt must still succeed.
	result, err := eSvc.SubmitAttempt(ctx, attempt.ID, studentID)
	require.NoError(t, err,
		"[AC-6 / REQ-ISSUE non-fatal] SubmitAttempt must succeed even when IssueOnPass fails")
	assert.True(t, result.Aprobado,
		"[AC-6] attempt must be aprobado=true even when cert issuance fails (non-fatal seam)")

	// Verify the attempt row has aprobado=true in DB.
	var aprobado bool
	require.NoError(t, db.Raw(
		`SELECT aprobado FROM attempt WHERE id = ?`,
		attempt.ID,
	).Scan(&aprobado).Error)
	assert.True(t, aprobado,
		"[AC-6] attempt.aprobado must be true in DB even when IssueOnPass fails")

	// Verify no certificate was created (storage failed before Create).
	var certCount int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM certificate WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&certCount).Error)
	assert.Equal(t, int64(0), certCount,
		"when storage fails, no cert row should be created")

	// Verify the certificates facade exports match the service.
	_ = certificates.ErrCertificateNotFound
	_ = certificates.ErrNoPDF
}
