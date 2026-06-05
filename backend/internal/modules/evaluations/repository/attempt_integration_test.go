//go:build integration

// Package repository — integration tests for the evaluations attempt repository.
// Covers migration 0007 (uq_answer_attempt_question), attempt CRUD, answer upsert
// UNIQUE semantics, and CASCADE delete behaviour.
// Run with: make backend-test-integration
package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// seedAttempt inserts an open attempt for the given user + evaluation.
func seedAttempt(t *testing.T, repo repository.Repository, userID, evalID string, numero int) *domain.Attempt {
	t.Helper()
	a := &domain.Attempt{
		ID:           uuid.New().String(),
		UserID:       userID,
		EvaluationID: evalID,
		Numero:       numero,
		IniciadoEn:   time.Now().UTC(),
	}
	require.NoError(t, repo.CreateAttempt(context.Background(), a), "seedAttempt: failed")
	return a
}

// ── TestMigration0007RoundTrip ─────────────────────────────────────────────────

// TestMigration0007RoundTrip verifies that migration 0007 can be applied and rolled back.
// Spec: REQ-SCH.
func TestMigration0007RoundTrip(t *testing.T) {
	db, m, teardown := testutil.SetupPostgresWithMigrate(t)
	defer func() { m.Close(); teardown() }()

	ctx := context.Background()

	// Verify the UNIQUE constraint exists after migration up (already applied by SetupPostgresWithMigrate).
	var constraintCount int64
	err := db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.table_constraints
		 WHERE table_schema = 'public'
		   AND table_name = 'answer'
		   AND constraint_name = 'uq_answer_attempt_question'
		   AND constraint_type = 'UNIQUE'`,
	).Scan(&constraintCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), constraintCount,
		"uq_answer_attempt_question UNIQUE constraint must exist after 0007 up")

	// Roll back migration 0007 only (1 step).
	require.NoError(t, m.Steps(-1), "migration 0007 down must succeed")

	// Verify the constraint is gone.
	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.table_constraints
		 WHERE table_schema = 'public'
		   AND table_name = 'answer'
		   AND constraint_name = 'uq_answer_attempt_question'
		   AND constraint_type = 'UNIQUE'`,
	).Scan(&constraintCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(0), constraintCount,
		"uq_answer_attempt_question must NOT exist after 0007 down")

	// Re-apply migration 0007 up — proves idempotent round-trip.
	require.NoError(t, m.Steps(1), "migration 0007 re-up must succeed")

	err = db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM information_schema.table_constraints
		 WHERE table_schema = 'public'
		   AND table_name = 'answer'
		   AND constraint_name = 'uq_answer_attempt_question'
		   AND constraint_type = 'UNIQUE'`,
	).Scan(&constraintCount).Error
	require.NoError(t, err)
	assert.Equal(t, int64(1), constraintCount,
		"uq_answer_attempt_question must exist after 0007 re-up")
}

// ── TestRepository_AttemptCRUD ─────────────────────────────────────────────────

// TestRepository_AttemptCRUD verifies create, get, update, and not-found sentinel.
func TestRepository_AttemptCRUD(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)

	// CreateAttempt.
	a := &domain.Attempt{
		ID:           uuid.New().String(),
		UserID:       userID,
		EvaluationID: e.ID,
		Numero:       1,
		IniciadoEn:   time.Now().UTC(),
	}
	require.NoError(t, repo.CreateAttempt(ctx, a))

	// GetAttemptByID.
	got, err := repo.GetAttemptByID(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.ID, got.ID)
	assert.Equal(t, userID, got.UserID)
	assert.Equal(t, e.ID, got.EvaluationID)
	assert.Equal(t, 1, got.Numero)
	assert.Nil(t, got.FinalizadoEn, "new attempt must have finalizado_en = null")

	// UpdateAttempt — mark as submitted.
	now := time.Now().UTC()
	require.NoError(t, repo.UpdateAttempt(ctx, a.ID, map[string]any{
		"puntaje":       80,
		"aprobado":      true,
		"finalizado_en": now,
	}))

	updated, err := repo.GetAttemptByID(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, 80, updated.Puntaje)
	assert.True(t, updated.Aprobado)
	assert.NotNil(t, updated.FinalizadoEn, "updated attempt must have finalizado_en set")

	// GetAttemptByID not found.
	_, err = repo.GetAttemptByID(ctx, uuid.New().String())
	assert.ErrorIs(t, err, repository.ErrAttemptNotFound)
}

// ── TestRepository_CountAttemptsByUserEval ─────────────────────────────────────

// TestRepository_CountAttemptsByUserEval verifies counting attempts per (user, eval).
func TestRepository_CountAttemptsByUserEval(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)

	// Zero attempts initially.
	n, err := repo.CountAttemptsByUserEval(ctx, userID, e.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)

	// Create two attempts.
	_ = seedAttempt(t, repo, userID, e.ID, 1)
	_ = seedAttempt(t, repo, userID, e.ID, 2)

	n, err = repo.CountAttemptsByUserEval(ctx, userID, e.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

// ── TestRepository_GetOpenAttempt ──────────────────────────────────────────────

// TestRepository_GetOpenAttempt verifies filtering by finalizado_en IS NULL.
func TestRepository_GetOpenAttempt(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)

	// No open attempt → ErrAttemptNotFound.
	_, err := repo.GetOpenAttempt(ctx, userID, e.ID)
	assert.ErrorIs(t, err, repository.ErrAttemptNotFound,
		"GetOpenAttempt must return ErrAttemptNotFound when none open")

	// Create open attempt.
	a := seedAttempt(t, repo, userID, e.ID, 1)
	got, err := repo.GetOpenAttempt(ctx, userID, e.ID)
	require.NoError(t, err)
	assert.Equal(t, a.ID, got.ID)

	// Submit the attempt → close it.
	now := time.Now().UTC()
	require.NoError(t, repo.UpdateAttempt(ctx, a.ID, map[string]any{"finalizado_en": now}))

	// Now no open attempt again.
	_, err = repo.GetOpenAttempt(ctx, userID, e.ID)
	assert.ErrorIs(t, err, repository.ErrAttemptNotFound,
		"GetOpenAttempt must return ErrAttemptNotFound after attempt is submitted")
}

// ── TestRepository_AnswerUpsert_UNIQUE ────────────────────────────────────────

// TestRepository_AnswerUpsert_UNIQUE verifies the LOAD-BEARING upsert behaviour:
// re-answering the same (attempt, question) must update the single row without
// violating the UNIQUE constraint added by migration 0007.
// Spec: REQ-ANSWER, REQ-SCH (f).
func TestRepository_AnswerUpsert_UNIQUE(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)
	q := seedQuestion(t, repo, e.ID, domain.TipoOpcionMultiple)
	opts := seedOptions(t, repo, q.ID, 2)
	a := seedAttempt(t, repo, userID, e.ID, 1)

	// First answer — must be inserted cleanly.
	require.NoError(t, repo.UpsertAnswer(ctx, a.ID, q.ID, opts[0].ID, false),
		"first UpsertAnswer must succeed")

	// Verify exactly one row exists.
	answers, err := repo.ListAnswersByAttempt(ctx, a.ID)
	require.NoError(t, err)
	require.Len(t, answers, 1, "exactly one answer row must exist after first upsert")
	assert.Equal(t, opts[0].ID, answers[0].OptionID)
	assert.False(t, answers[0].Correcta)

	// Re-answer the same question with a different option (correcta=true now).
	// Must update the existing row, NOT insert a second one.
	require.NoError(t, repo.UpsertAnswer(ctx, a.ID, q.ID, opts[1].ID, true),
		"second UpsertAnswer (re-answer) must NOT violate UNIQUE constraint")

	answers, err = repo.ListAnswersByAttempt(ctx, a.ID)
	require.NoError(t, err)
	require.Len(t, answers, 1, "[LOAD-BEARING] re-answer must UPDATE single row, not insert duplicate")
	assert.Equal(t, opts[1].ID, answers[0].OptionID, "answer must be updated to new optionID")
	assert.True(t, answers[0].Correcta, "correcta must be updated to new value")
}

// ── TestRepository_AnswerCascadeDelete ────────────────────────────────────────

// TestRepository_AnswerCascadeDelete verifies that deleting an attempt cascades
// to its answers (FK ON DELETE CASCADE from attempt to answer).
func TestRepository_AnswerCascadeDelete(t *testing.T) {
	db, teardown := testutil.SetupPostgres(t)
	defer teardown()
	repo := repository.New(db)
	ctx := context.Background()

	userID := seedUser(t, db)
	courseID := seedCourse(t, db, userID)
	e := seedEvaluation(t, repo, courseID)
	q := seedQuestion(t, repo, e.ID, domain.TipoOpcionMultiple)
	opts := seedOptions(t, repo, q.ID, 2)
	a := seedAttempt(t, repo, userID, e.ID, 1)

	require.NoError(t, repo.UpsertAnswer(ctx, a.ID, q.ID, opts[0].ID, false))

	// Confirm answer exists.
	answers, err := repo.ListAnswersByAttempt(ctx, a.ID)
	require.NoError(t, err)
	require.Len(t, answers, 1)

	// Delete the attempt directly (CASCADE to answer).
	require.NoError(t, db.Exec(`DELETE FROM attempt WHERE id = ?`, a.ID).Error)

	// Answers must be gone.
	answers, err = repo.ListAnswersByAttempt(ctx, a.ID)
	require.NoError(t, err)
	assert.Len(t, answers, 0, "answers must be cascade-deleted when attempt is deleted")
}
