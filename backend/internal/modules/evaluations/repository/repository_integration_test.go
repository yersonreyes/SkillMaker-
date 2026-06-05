//go:build integration

// Package repository — integration tests for the evaluations repository.
// Run with: make backend-test-integration
// These tests exercise migration 0006, FK constraints, UNIQUE constraints,
// CASCADE behaviour, and repository CRUD using a real Postgres container.
package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── helpers ────────────────────────────────────────────────────────────────────

// seedUser inserts a minimal "user" row and returns its ID.
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

// seedCourse inserts a course row for the given creador and returns its ID.
func seedCourse(t *testing.T, db *gorm.DB, creadorID string) string {
	t.Helper()
	id := uuid.New().String()
	err := db.Exec(
		`INSERT INTO course (id, creador_id, titulo, descripcion, estado)
		 VALUES (?, ?, ?, ?, 'borrador')`,
		id, creadorID, "Test Course", "Desc",
	).Error
	require.NoError(t, err, "seedCourse: failed to insert course")
	return id
}

// seedEvaluation inserts an evaluation via the repository.
func seedEvaluation(t *testing.T, repo repository.Repository, courseID string) *domain.Evaluation {
	t.Helper()
	e := &domain.Evaluation{
		ID:          uuid.New().String(),
		CourseID:    courseID,
		NotaMinima:  70,
		IntentosMax: 3,
	}
	require.NoError(t, repo.CreateEvaluation(context.Background(), e), "seedEvaluation: failed")
	return e
}

// seedQuestion inserts a question via the repository.
func seedQuestion(t *testing.T, repo repository.Repository, evalID string, tipo domain.TipoPregunta) *domain.Question {
	t.Helper()
	q := &domain.Question{
		ID:           uuid.New().String(),
		EvaluationID: evalID,
		Enunciado:    "Question text",
		Tipo:         tipo,
		Puntaje:      5,
		Orden:        0,
	}
	require.NoError(t, repo.CreateQuestion(context.Background(), q), "seedQuestion: failed")
	return q
}

// seedOptions inserts options via the repository batch call.
func seedOptions(t *testing.T, repo repository.Repository, questionID string, n int) []domain.Option {
	t.Helper()
	opts := make([]domain.Option, n)
	for i := range opts {
		opts[i] = domain.Option{
			ID:         uuid.New().String(),
			QuestionID: questionID,
			Texto:      "Option " + uuid.New().String()[:8],
			Correcta:   i == 0,
			Orden:      i,
		}
	}
	require.NoError(t, repo.CreateOptions(context.Background(), opts), "seedOptions: failed")
	return opts
}

// ── SCH-1-A: round-trip migration ─────────────────────────────────────────────

// TestMigration0006RoundTrip verifies that migration 0006 can be applied and rolled back.
// Spec: SCH-1-A.
func TestMigration0006RoundTrip(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer func() { m.Close(); teardown() }()

	ctx := context.Background()

	// Verify all five tables exist after migration up (already applied by SetupPostgresWithMigrate).
	tables := []string{"evaluation", "question", "question_option", "attempt", "answer"}
	for _, tbl := range tables {
		var exists bool
		err := db.Raw(
			`SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_schema='public' AND table_name=?
			)`, tbl,
		).Scan(&exists).Error
		require.NoError(t, err)
		assert.True(t, exists, "table %q must exist after migration 0006 up", tbl)
	}

	// Roll back migration 0006 (and any above it).
	require.NoError(t, m.Down(), "migration down must succeed")

	// Verify all five tables are gone.
	for _, tbl := range tables {
		var exists bool
		err := db.Raw(
			`SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_schema='public' AND table_name=?
			)`, tbl,
		).Scan(&exists).Error
		require.NoError(t, err)
		assert.False(t, exists, "table %q must NOT exist after migration 0006 down", tbl)
	}

	// Re-apply migration up — proves idempotent round-trip.
	require.NoError(t, m.Up(), "migration re-up must succeed")

	for _, tbl := range tables {
		var exists bool
		err := db.Raw(
			`SELECT EXISTS(
				SELECT 1 FROM information_schema.tables
				WHERE table_schema='public' AND table_name=?
			)`, tbl,
		).Scan(&exists).Error
		require.NoError(t, err)
		assert.True(t, exists, "table %q must exist after migration 0006 re-up", tbl)
	}

	_ = ctx // suppress unused warning — context is used implicitly via repo calls below
}

// TestMigration0006UniqueCourseid verifies the UNIQUE(course_id) constraint on evaluation.
// Spec: SCH-1-B.
func TestMigration0006UniqueCourseid(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)

	// First insert must succeed.
	err := db.Exec(
		`INSERT INTO evaluation (id, course_id, nota_minima, intentos_max)
		 VALUES (gen_random_uuid(), ?, 70, 0)`,
		courseID,
	).Error
	require.NoError(t, err, "first evaluation insert must succeed")

	// Second insert with same course_id must fail with unique violation.
	err = db.Exec(
		`INSERT INTO evaluation (id, course_id, nota_minima, intentos_max)
		 VALUES (gen_random_uuid(), ?, 80, 1)`,
		courseID,
	).Error
	require.Error(t, err, "second evaluation for same course must fail")
	assert.Contains(t, err.Error(), "23505", "unique violation must be error code 23505")
}

// TestMigration0006TipoCheckConstraint verifies the CHECK constraint on question.tipo.
// Spec: SCH-1-C.
func TestMigration0006TipoCheckConstraint(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)

	// Create an evaluation first.
	evalID := uuid.New().String()
	err := db.Exec(
		`INSERT INTO evaluation (id, course_id, nota_minima, intentos_max)
		 VALUES (?, ?, 70, 0)`,
		evalID, courseID,
	).Error
	require.NoError(t, err)

	// Attempt to insert a question with tipo='libre' — must fail.
	err = db.Exec(
		`INSERT INTO question (id, evaluation_id, enunciado, tipo, puntaje, orden)
		 VALUES (gen_random_uuid(), ?, 'Q?', 'libre', 1, 0)`,
		evalID,
	).Error
	require.Error(t, err, "tipo='libre' must violate CHECK constraint")
	assert.Contains(t, err.Error(), "ck_question_tipo", "must cite the check constraint name")
}

// ── Phase 2: Repository CRUD tests ────────────────────────────────────────────

// TestRepository_EvaluationCRUD verifies create + get (by ID and by course) + update.
func TestRepository_EvaluationCRUD(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)

	// CreateEvaluation happy path.
	e := &domain.Evaluation{
		ID:          uuid.New().String(),
		CourseID:    courseID,
		NotaMinima:  75,
		IntentosMax: 2,
	}
	require.NoError(t, repo.CreateEvaluation(ctx, e))

	// GetEvaluationByID.
	got, err := repo.GetEvaluationByID(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, courseID, got.CourseID)
	assert.Equal(t, 75, got.NotaMinima)

	// GetEvaluationByCourse.
	gotBC, err := repo.GetEvaluationByCourse(ctx, courseID)
	require.NoError(t, err)
	assert.Equal(t, e.ID, gotBC.ID)

	// UpdateEvaluation.
	require.NoError(t, repo.UpdateEvaluation(ctx, e.ID, map[string]any{"nota_minima": 90}))
	updated, err := repo.GetEvaluationByID(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, 90, updated.NotaMinima)
}

// TestRepository_EvaluationExists_ReturnsErrEvaluationExists verifies UNIQUE sentinel.
// Spec: SCH-1-B.
func TestRepository_EvaluationExists_ReturnsErrEvaluationExists(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)

	e1 := &domain.Evaluation{ID: uuid.New().String(), CourseID: courseID, NotaMinima: 70, IntentosMax: 0}
	require.NoError(t, repo.CreateEvaluation(ctx, e1))

	e2 := &domain.Evaluation{ID: uuid.New().String(), CourseID: courseID, NotaMinima: 80, IntentosMax: 1}
	err := repo.CreateEvaluation(ctx, e2)
	assert.ErrorIs(t, err, repository.ErrEvaluationExists,
		"second evaluation for same course must return ErrEvaluationExists")
}

// TestRepository_EvaluationNotFound verifies ErrEvaluationNotFound sentinel.
func TestRepository_EvaluationNotFound(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	_, err := repo.GetEvaluationByID(ctx, uuid.New().String())
	assert.ErrorIs(t, err, repository.ErrEvaluationNotFound)

	_, err = repo.GetEvaluationByCourse(ctx, uuid.New().String())
	assert.ErrorIs(t, err, repository.ErrEvaluationNotFound)
}

// TestRepository_QuestionCRUD verifies question create + get + update + delete.
func TestRepository_QuestionCRUD(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)

	// CreateQuestion.
	q := &domain.Question{
		ID:           uuid.New().String(),
		EvaluationID: e.ID,
		Enunciado:    "What is Go?",
		Tipo:         domain.TipoOpcionMultiple,
		Puntaje:      10,
		Orden:        0,
	}
	require.NoError(t, repo.CreateQuestion(ctx, q))

	// GetQuestionByID.
	got, err := repo.GetQuestionByID(ctx, q.ID)
	require.NoError(t, err)
	assert.Equal(t, q.ID, got.ID)
	assert.Equal(t, "What is Go?", got.Enunciado)

	// UpdateQuestion.
	require.NoError(t, repo.UpdateQuestion(ctx, q.ID, map[string]any{"puntaje": 15}))
	updated, err := repo.GetQuestionByID(ctx, q.ID)
	require.NoError(t, err)
	assert.Equal(t, 15, updated.Puntaje)

	// DeleteQuestion.
	require.NoError(t, repo.DeleteQuestion(ctx, q.ID))
	_, err = repo.GetQuestionByID(ctx, q.ID)
	assert.ErrorIs(t, err, repository.ErrQuestionNotFound)
}

// TestRepository_ListQuestionsByEvaluation_OrderByOrden verifies ordering.
func TestRepository_ListQuestionsByEvaluation_OrderByOrden(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)

	q1 := &domain.Question{ID: uuid.New().String(), EvaluationID: e.ID, Enunciado: "Q1", Tipo: domain.TipoOpcionMultiple, Puntaje: 1, Orden: 2}
	q2 := &domain.Question{ID: uuid.New().String(), EvaluationID: e.ID, Enunciado: "Q2", Tipo: domain.TipoOpcionMultiple, Puntaje: 1, Orden: 0}
	q3 := &domain.Question{ID: uuid.New().String(), EvaluationID: e.ID, Enunciado: "Q3", Tipo: domain.TipoOpcionMultiple, Puntaje: 1, Orden: 1}
	require.NoError(t, repo.CreateQuestion(ctx, q1))
	require.NoError(t, repo.CreateQuestion(ctx, q2))
	require.NoError(t, repo.CreateQuestion(ctx, q3))

	list, err := repo.ListQuestionsByEvaluation(ctx, e.ID)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, 0, list[0].Orden, "first question must have orden=0")
	assert.Equal(t, 1, list[1].Orden, "second question must have orden=1")
	assert.Equal(t, 2, list[2].Orden, "third question must have orden=2")
}

// TestRepository_OptionCRUD verifies option batch create + get + update + delete.
func TestRepository_OptionCRUD(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)
	q := seedQuestion(t, repo, e.ID, domain.TipoOpcionMultiple)

	// CreateOptions batch.
	opts := []domain.Option{
		{ID: uuid.New().String(), QuestionID: q.ID, Texto: "A", Correcta: true, Orden: 0},
		{ID: uuid.New().String(), QuestionID: q.ID, Texto: "B", Correcta: false, Orden: 1},
	}
	require.NoError(t, repo.CreateOptions(ctx, opts))

	// ListOptionsByQuestion.
	list, err := repo.ListOptionsByQuestion(ctx, q.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "A", list[0].Texto)
	assert.True(t, list[0].Correcta)

	// GetOptionByID.
	got, err := repo.GetOptionByID(ctx, opts[0].ID)
	require.NoError(t, err)
	assert.Equal(t, opts[0].ID, got.ID)

	// UpdateOption.
	require.NoError(t, repo.UpdateOption(ctx, opts[0].ID, map[string]any{"correcta": false}))
	updated, err := repo.GetOptionByID(ctx, opts[0].ID)
	require.NoError(t, err)
	assert.False(t, updated.Correcta)

	// DeleteOption.
	require.NoError(t, repo.DeleteOption(ctx, opts[0].ID))
	_, err = repo.GetOptionByID(ctx, opts[0].ID)
	assert.ErrorIs(t, err, repository.ErrOptionNotFound)
}

// TestRepository_CascadeDeleteQuestion_RemovesOptions verifies CASCADE on question delete.
func TestRepository_CascadeDeleteQuestion_RemovesOptions(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)
	q := seedQuestion(t, repo, e.ID, domain.TipoOpcionMultiple)
	opts := seedOptions(t, repo, q.ID, 3)

	// Delete the question.
	require.NoError(t, repo.DeleteQuestion(ctx, q.ID))

	// Options must be gone.
	for _, o := range opts {
		_, err := repo.GetOptionByID(ctx, o.ID)
		assert.ErrorIs(t, err, repository.ErrOptionNotFound,
			"option %s must be cascade-deleted with question", o.ID)
	}
}

// TestRepository_CascadeDeleteCourse_RemovesEvaluation verifies CASCADE from course.
func TestRepository_CascadeDeleteCourse_RemovesEvaluation(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)

	// Delete the course (FK ON DELETE CASCADE → evaluation).
	require.NoError(t, db.Exec(`DELETE FROM course WHERE id = ?`, courseID).Error)

	// Evaluation must be gone.
	_, err := repo.GetEvaluationByID(ctx, e.ID)
	assert.ErrorIs(t, err, repository.ErrEvaluationNotFound,
		"evaluation must be cascade-deleted with course")
}
