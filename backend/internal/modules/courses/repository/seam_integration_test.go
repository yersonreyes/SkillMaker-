//go:build integration

// Package repository — seam end-to-end integration tests (C2.4).
//
// TestSeam_PassingAttempt_FlipsCompletado verifies the full cross-module seam:
//
//	student enrolled → submits passing attempt → enrollment.completado = true
//
// Uses real Postgres (testcontainers). Wires coursesSvc + evaluationsSvc directly
// (same as main.go composition root) to prove the seam fires end-to-end.
//
// Spec: REQ-SEAM / AC-10.
package repository_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	coursesDomain "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/domain"
	coursesRepo "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/repository"
	coursesSvc "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	evalsDomain "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/domain"
	evalsRepo "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
	evalsSvc "github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/storage"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// noopStorage is a minimal storage.Client stub for the seam test.
// The seam test does not touch storage — we only need to satisfy the interface.
type noopStorage struct{}

func (n *noopStorage) PresignPutURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (n *noopStorage) PresignGetURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", nil
}
func (n *noopStorage) Delete(_ context.Context, _ string) error { return nil }
func (n *noopStorage) Ping(_ context.Context) error             { return nil }
func (n *noopStorage) PutObject(_ context.Context, _ string, _ io.Reader, _ int64, _ string) error {
	return nil
}

// Ensure noopStorage satisfies storage.Client at compile time.
var _ storage.Client = &noopStorage{}

// seedCourseAprobado inserts an aprobado course row directly.
func seedCourseAprobado(t *testing.T, db *gorm.DB, creadorID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, ?, ?, 'aprobado')`,
		id, creadorID, "Seam Test Course", "Desc",
	).Error
	require.NoError(t, err, "seedCourseAprobado: failed to insert course")
	return id
}

// seedEvaluationWithPassingSetup seeds an evaluation + 1 true/false question + 1 correct option.
// Returns evaluationID, optionID (the correct option).
func seedEvaluationWithPassingSetup(t *testing.T, db *gorm.DB, courseID string) (evalID, optionID string) {
	t.Helper()

	evalID = uuid.New().String()
	err := db.Exec(
		`INSERT INTO evaluation (id, course_id, nota_minima, intentos_max)
		 VALUES (?, ?, 70, 5)`,
		evalID, courseID,
	).Error
	require.NoError(t, err, "seedEvaluationWithPassingSetup: failed to insert evaluation")

	questionID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO question (id, evaluation_id, enunciado, tipo, puntaje, orden)
		 VALUES (?, ?, ?, 'verdadero_falso', 100, 0)`,
		questionID, evalID, "Is Go awesome?",
	).Error
	require.NoError(t, err, "seedEvaluationWithPassingSetup: failed to insert question")

	optionID = uuid.New().String()
	err = db.Exec(
		`INSERT INTO question_option (id, question_id, texto, correcta, orden)
		 VALUES (?, ?, ?, true, 0)`,
		optionID, questionID, "Verdadero",
	).Error
	require.NoError(t, err, "seedEvaluationWithPassingSetup: failed to insert option")

	// Also insert the incorrect option (V/F requires 2 options).
	wrongID := uuid.New().String()
	err = db.Exec(
		`INSERT INTO question_option (id, question_id, texto, correcta, orden)
		 VALUES (?, ?, ?, false, 1)`,
		wrongID, questionID, "Falso",
	).Error
	require.NoError(t, err, "seedEvaluationWithPassingSetup: failed to insert wrong option")

	return evalID, optionID
}

// TestSeam_PassingAttempt_FlipsCompletado is the end-to-end seam integration test.
// Wires coursesSvc (real Postgres) + evaluationsSvc with WithEnrollmentCompleter(coursesSvc).
// Proves that submitting a passing attempt flips enrollment.completado = true.
// Also verifies the no-op path: MarkEnrollmentCompleted on a non-existent enrollment → nil.
//
// Spec: REQ-SEAM / AC-10.
func TestSeam_PassingAttempt_FlipsCompletado(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	// ── 1. Seed users + course + enrollment ─────────────────────────────────────
	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCourseAprobado(t, db, creadorID)

	// Build real coursesSvc.
	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, &noopStorage{}, 15*time.Minute, 52_428_800)

	// Enroll student via the service (tests the enrollment path too).
	err := cSvc.Enroll(ctx, studentID, courseID)
	require.NoError(t, err, "Enroll must succeed for aprobado course")

	// Verify completado=false initially.
	var completado bool
	require.NoError(t, db.Raw(
		`SELECT completado FROM enrollment WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&completado).Error)
	assert.False(t, completado, "completado must be false before passing attempt")

	// ── 2. Setup evaluation + attempt ───────────────────────────────────────────
	evalID, optionID := seedEvaluationWithPassingSetup(t, db, courseID)

	// Wire evaluationsSvc WITH the EnrollmentCompleter seam.
	eRepo := evalsRepo.New(db)
	eSvc := evalsSvc.New(eRepo, cSvc, evalsSvc.WithEnrollmentCompleter(cSvc))

	// Start an attempt for the student.
	attempt, err := eSvc.StartAttempt(ctx, evalID, studentID)
	require.NoError(t, err, "StartAttempt must succeed")

	// Save the correct answer.
	// We need to get the question ID from the attempt state.
	state, err := eSvc.GetAttempt(ctx, attempt.ID, studentID)
	require.NoError(t, err, "GetAttempt must succeed")
	require.NotEmpty(t, state.Questions, "attempt must have questions")

	questionID := state.Questions[0].ID
	err = eSvc.SaveAnswer(ctx, attempt.ID, studentID, questionID, optionID)
	require.NoError(t, err, "SaveAnswer must succeed")

	// ── 3. Submit attempt → seam must fire ──────────────────────────────────────
	result, err := eSvc.SubmitAttempt(ctx, attempt.ID, studentID)
	require.NoError(t, err, "SubmitAttempt must succeed")
	assert.True(t, result.Aprobado, "attempt with 100% correct answer must be aprobado")

	// ── 4. Verify completado was flipped via seam ────────────────────────────────
	require.NoError(t, db.Raw(
		`SELECT completado FROM enrollment WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&completado).Error)
	assert.True(t, completado,
		"[REQ-SEAM / AC-10] enrollment.completado must be true after passing attempt (seam fired)")

	// ── 5. Verify ListMyCourses shows completado=true ────────────────────────────
	myCourses, err := cSvc.ListMyCourses(ctx, studentID)
	require.NoError(t, err, "ListMyCourses must succeed")
	require.Len(t, myCourses, 1, "student must have 1 enrollment")
	assert.True(t, myCourses[0].Completado,
		"ListMyCourses must reflect completado=true after passing attempt")

	// ── 6. No-op path: MarkEnrollmentCompleted when no enrollment → nil ─────────
	randomUser := uuid.New().String()
	err = cSvc.MarkEnrollmentCompleted(ctx, randomUser, courseID)
	assert.NoError(t, err, "MarkEnrollmentCompleted on missing enrollment must be a no-op (nil)")
}

// TestSeam_MarkEnrollmentCompleted_Idempotent verifies calling MarkEnrollmentCompleted
// when completado=true already leaves it true (idempotent).
func TestSeam_MarkEnrollmentCompleted_Idempotent(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCourseAprobado(t, db, creadorID)

	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, &noopStorage{}, 15*time.Minute, 52_428_800)

	require.NoError(t, cSvc.Enroll(ctx, studentID, courseID))

	// First flip.
	require.NoError(t, cSvc.MarkEnrollmentCompleted(ctx, studentID, courseID))

	// Second flip (idempotent).
	require.NoError(t, cSvc.MarkEnrollmentCompleted(ctx, studentID, courseID))

	// Verify still true.
	var completado bool
	require.NoError(t, db.Raw(
		`SELECT completado FROM enrollment WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&completado).Error)
	assert.True(t, completado, "completado must remain true after idempotent second call")
}

// seedCatalogCourse is a helper to seed an aprobado course for catalog tests.
func seedCatalogCourse(t *testing.T, db *gorm.DB, creadorID, titulo string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, ?, 'Desc', 'aprobado')`,
		id, creadorID, titulo,
	).Error
	require.NoError(t, err, "seedCatalogCourse: failed to insert course")
	return id
}

// TestCatalogIntegration_EnrollIdempotency verifies two calls → one DB row + 200 both.
func TestCatalogIntegration_EnrollIdempotency(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCatalogCourse(t, db, creadorID, "Go Advanced")

	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, &noopStorage{}, 15*time.Minute, 52_428_800)

	// First enroll.
	require.NoError(t, cSvc.Enroll(ctx, studentID, courseID),
		"first enroll must succeed")

	// Second enroll (idempotent).
	require.NoError(t, cSvc.Enroll(ctx, studentID, courseID),
		"second enroll must be a no-op")

	// Exactly one row.
	var count int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM enrollment WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&count).Error)
	assert.Equal(t, int64(1), count, "idempotent enroll must not create duplicate row")
}

// TestCatalogIntegration_NonAprobado_Enroll_Returns404 verifies enroll on borrador → ErrCourseNotFound.
func TestCatalogIntegration_NonAprobado_Enroll_Returns404(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)

	// Seed a borrador course.
	borradorCourse := &coursesDomain.Course{
		ID:          uuid.New().String(),
		CreadorID:   creadorID,
		Titulo:      "Draft Course",
		Descripcion: "Desc",
		Estado:      coursesDomain.EstadoBorrador,
	}
	cRepo := coursesRepo.New(db)
	require.NoError(t, cRepo.Create(ctx, borradorCourse))

	cSvc := coursesSvc.New(cRepo, &noopStorage{}, 15*time.Minute, 52_428_800)

	err := cSvc.Enroll(ctx, studentID, borradorCourse.ID)
	require.Error(t, err, "enroll on borrador must return error")
	assert.ErrorIs(t, err, coursesSvc.ErrCourseNotFound,
		"borrador course must be invisible to alumnos (draft-invisibility / AC-6)")
}

// TestMyCourseIsolation_UserACannotSeeUserBCourses verifies isolation.
func TestMyCourseIsolation_UserACannotSeeUserBCourses(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	creadorID := seedUser(t, db)
	userA := seedUser(t, db)
	userB := seedUser(t, db)

	courseA := seedCatalogCourse(t, db, creadorID, "Course A")
	courseB := seedCatalogCourse(t, db, creadorID, "Course B")

	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, &noopStorage{}, 15*time.Minute, 52_428_800)

	// Enroll A in courseA, B in courseB.
	require.NoError(t, cSvc.Enroll(ctx, userA, courseA))
	require.NoError(t, cSvc.Enroll(ctx, userB, courseB))

	// A's courses must only contain courseA.
	aCourses, err := cSvc.ListMyCourses(ctx, userA)
	require.NoError(t, err)
	require.Len(t, aCourses, 1, "userA must have exactly 1 enrollment")
	assert.Equal(t, courseA, aCourses[0].CourseID, "userA must only see their own course")

	// B's courses must only contain courseB.
	bCourses, err := cSvc.ListMyCourses(ctx, userB)
	require.NoError(t, err)
	require.Len(t, bCourses, 1, "userB must have exactly 1 enrollment")
	assert.Equal(t, courseB, bCourses[0].CourseID, "userB must only see their own course")
}

// unused import guard for evalsDomain.
var _ = evalsDomain.Attempt{}

// ── course-player-progress integration tests (Change 2 / REQ-PROGRESS-READ + REQ-DECOUPLE) ──

// TestGetCatalogDetail_Tree_CarriesCallerCompletado verifies that GetCatalogDetail returns
// caller-scoped completado flags for each video in the content tree.
//
// Satisfies REQ-PROGRESS-READ / AC-4 (progress is per-user).
func TestGetCatalogDetail_Tree_CarriesCallerCompletado(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	creadorID := seedUser(t, db)
	userA := seedUser(t, db)
	userB := seedUser(t, db)

	courseID := seedCatalogCourse(t, db, creadorID, "Progress Test Course")

	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, &noopStorage{}, 15*time.Minute, 52_428_800)

	// Create section + 2 videos directly in DB.
	sectionID := uuid.New().String()
	err := db.Exec(
		`INSERT INTO section (id, course_id, titulo, orden) VALUES (?, ?, 'S1', 0)`,
		sectionID, courseID,
	).Error
	require.NoError(t, err)

	vid1 := uuid.New().String()
	vid2 := uuid.New().String()
	for i, vid := range []struct{ id, titulo string }{{vid1, "V1"}, {vid2, "V2"}} {
		err = db.Exec(
			`INSERT INTO video (id, section_id, titulo, descripcion, url, proveedor, duracion_s, orden)
			 VALUES (?, ?, ?, '', 'https://youtube.com/v', 'youtube', 120, ?)`,
			vid.id, sectionID, vid.titulo, i,
		).Error
		require.NoError(t, err)
	}

	// Enroll A and B in the course.
	require.NoError(t, cSvc.Enroll(ctx, userA, courseID))
	require.NoError(t, cSvc.Enroll(ctx, userB, courseID))

	// A completes vid1 only.
	require.NoError(t, cRepo.UpsertVideoProgress(ctx, userA, vid1, true, 0))

	// A's detail: vid1 completado=true, vid2 completado=false.
	detailA, err := cSvc.GetCatalogDetail(ctx, userA, courseID)
	require.NoError(t, err)
	require.True(t, detailA.Enrolled, "A must be enrolled")
	require.Len(t, detailA.Sections, 1)
	require.Len(t, detailA.Sections[0].Videos, 2)

	progMap := make(map[string]bool)
	for _, v := range detailA.Sections[0].Videos {
		progMap[v.ID] = v.Completado
	}
	assert.True(t, progMap[vid1], "A: vid1 must have completado=true")
	assert.False(t, progMap[vid2], "A: vid2 must have completado=false")

	// B's detail: both videos completado=false (B has no progress rows). REQ-PROGRESS-READ §3.
	detailB, err := cSvc.GetCatalogDetail(ctx, userB, courseID)
	require.NoError(t, err)
	require.True(t, detailB.Enrolled)
	for _, v := range detailB.Sections[0].Videos {
		assert.False(t, v.Completado, "B: video %s must have completado=false (B has no progress)", v.ID)
	}
}

// TestMarkAllVideosComplete_DoesNotFlipEnrollmentCompletado verifies the decouple invariant:
// marking ALL videos complete NEVER sets enrollment.completado=true (REQ-DECOUPLE / AC-5).
func TestMarkAllVideosComplete_DoesNotFlipEnrollmentCompletado(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	creadorID := seedUser(t, db)
	learnerID := seedUser(t, db)

	courseID := seedCatalogCourse(t, db, creadorID, "Decouple Test Course")

	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, &noopStorage{}, 15*time.Minute, 52_428_800)

	// Create section + 2 videos directly in DB.
	sectionID := uuid.New().String()
	err := db.Exec(
		`INSERT INTO section (id, course_id, titulo, orden) VALUES (?, ?, 'S1', 0)`,
		sectionID, courseID,
	).Error
	require.NoError(t, err)

	vid1 := uuid.New().String()
	vid2 := uuid.New().String()
	for i, vid := range []struct{ id, titulo string }{{vid1, "V1"}, {vid2, "V2"}} {
		err = db.Exec(
			`INSERT INTO video (id, section_id, titulo, descripcion, url, proveedor, duracion_s, orden)
			 VALUES (?, ?, ?, '', 'https://youtube.com/v', 'youtube', 120, ?)`,
			vid.id, sectionID, vid.titulo, i,
		).Error
		require.NoError(t, err)
	}

	// Enroll learner.
	require.NoError(t, cSvc.Enroll(ctx, learnerID, courseID))

	// Mark ALL videos complete via MarkVideoProgress (goes through IsEnrolled gate).
	require.NoError(t, cSvc.MarkVideoProgress(ctx, learnerID, vid1, true, 0))
	require.NoError(t, cSvc.MarkVideoProgress(ctx, learnerID, vid2, true, 0))

	// Verify enrollment.completado is still false (REQ-DECOUPLE — AC-5).
	var completado bool
	err = db.Raw(
		`SELECT completado FROM enrollment WHERE user_id = ? AND course_id = ?`,
		learnerID, courseID,
	).Scan(&completado).Error
	require.NoError(t, err)
	assert.False(t, completado,
		"enrollment.completado must remain false after all videos marked complete (REQ-DECOUPLE)")
}

// TestSeam_EvalPass_DoesNotCreateVideoProgressRows verifies the OTHER decouple direction:
// passing the evaluation (triggering MarkEnrollmentCompleted via the evaluations seam) MUST NOT
// create or alter any video_progress rows for that user/course.
//
// Spec: REQ-DECOUPLE AC-5 — eval-pass and video progress are fully independent. The eval seam
// only touches enrollment.completado; it has no knowledge of video_progress at all.
//
// SUGGESTION-1 from verify-report-be (#373): explicit test for the OTHER decouple direction.
func TestSeam_EvalPass_DoesNotCreateVideoProgressRows(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	ctx := context.Background()

	// ── 1. Seed course, enrollment, evaluation ────────────────────────────────────
	creadorID := seedUser(t, db)
	studentID := seedUser(t, db)
	courseID := seedCourseAprobado(t, db, creadorID)

	cRepo := coursesRepo.New(db)
	cSvc := coursesSvc.New(cRepo, &noopStorage{}, 15*time.Minute, 52_428_800)

	// Enroll student.
	require.NoError(t, cSvc.Enroll(ctx, studentID, courseID), "Enroll must succeed")

	// Verify no video_progress rows exist before the seam fires.
	var countBefore int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM video_progress WHERE user_id = ? AND video_id IN
		 (SELECT v.id FROM video v
		  JOIN section s ON s.id = v.section_id
		  WHERE s.course_id = ?)`,
		studentID, courseID,
	).Scan(&countBefore).Error)
	assert.Equal(t, int64(0), countBefore,
		"no video_progress rows must exist before seam fires")

	// ── 2. Wire evaluationsSvc + seed a passing evaluation ────────────────────────
	evalID, optionID := seedEvaluationWithPassingSetup(t, db, courseID)

	eRepo := evalsRepo.New(db)
	eSvc := evalsSvc.New(eRepo, cSvc, evalsSvc.WithEnrollmentCompleter(cSvc))

	// Start + answer + submit a passing attempt → fires MarkEnrollmentCompleted seam.
	attempt, err := eSvc.StartAttempt(ctx, evalID, studentID)
	require.NoError(t, err)

	state, err := eSvc.GetAttempt(ctx, attempt.ID, studentID)
	require.NoError(t, err)
	require.NotEmpty(t, state.Questions)

	questionID := state.Questions[0].ID
	require.NoError(t, eSvc.SaveAnswer(ctx, attempt.ID, studentID, questionID, optionID))

	result, err := eSvc.SubmitAttempt(ctx, attempt.ID, studentID)
	require.NoError(t, err)
	require.True(t, result.Aprobado, "attempt must be aprobado (seam precondition)")

	// ── 3. Verify enrollment.completado was flipped (seam fired) ─────────────────
	var completado bool
	require.NoError(t, db.Raw(
		`SELECT completado FROM enrollment WHERE user_id = ? AND course_id = ?`,
		studentID, courseID,
	).Scan(&completado).Error)
	assert.True(t, completado,
		"enrollment.completado must be true after passing attempt (seam fired correctly)")

	// ── 4. KEY ASSERTION: no video_progress rows created by the seam ─────────────
	// MarkEnrollmentCompleted must ONLY touch enrollment.completado — it must NEVER
	// create, update, or delete any video_progress rows (REQ-DECOUPLE, OTHER DIRECTION).
	var countAfter int64
	require.NoError(t, db.Raw(
		`SELECT COUNT(*) FROM video_progress WHERE user_id = ?`,
		studentID,
	).Scan(&countAfter).Error)
	assert.Equal(t, int64(0), countAfter,
		"[REQ-DECOUPLE / SUGGESTION-1] eval-pass (MarkEnrollmentCompleted) must NOT create "+
			"any video_progress rows — eval completion and video progress are fully independent")
}
