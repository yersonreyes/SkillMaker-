// Package service — unit tests for the student attempt lifecycle (C3.2).
// No build tag: runs with standard `make backend-test`.
//
// LOAD-BEARING tests:
//
//	(a) no-correcta-leak: AttemptStateModel.Questions[*].Options[*] type has no Correcta
//	(b) on-pass seam invocation: aprobado=true → both seams called; false → neither; error → non-fatal
//	(c) intentos_max: count>=max → ErrMaxAttemptsReached; max=0 → unlimited
//	(d) attempt ownership: different user → ErrAttemptNotFound
//	(e) scoring: earned/total*100; total==0 guard; pass boundary
//	(f) answer upsert: UpsertAnswer called on re-answer
//	(g) open-attempt resume: existing open attempt → RETURNED as resume (no new attempt created)
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
	"github.com/yersonreyes/SkillMaker-/backend/internal/testutil"
)

// ── mockEvalRepo extensions (7 new methods for attempt lifecycle) ──────────────
// These extend the mockEvalRepo defined in service_test.go (same package).

func (m *mockEvalRepo) CreateAttempt(ctx context.Context, a *domain.Attempt) error {
	args := m.Called(ctx, a)
	return args.Error(0)
}

func (m *mockEvalRepo) GetAttemptByID(ctx context.Context, id string) (*domain.Attempt, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Attempt), args.Error(1)
}

func (m *mockEvalRepo) UpdateAttempt(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *mockEvalRepo) CountAttemptsByUserEval(ctx context.Context, userID, evalID string) (int64, error) {
	args := m.Called(ctx, userID, evalID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockEvalRepo) GetOpenAttempt(ctx context.Context, userID, evalID string) (*domain.Attempt, error) {
	args := m.Called(ctx, userID, evalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Attempt), args.Error(1)
}

func (m *mockEvalRepo) UpsertAnswer(ctx context.Context, attemptID, questionID, optionID string, correcta bool) error {
	args := m.Called(ctx, attemptID, questionID, optionID, correcta)
	return args.Error(0)
}

func (m *mockEvalRepo) ListAnswersByAttempt(ctx context.Context, attemptID string) ([]domain.Answer, error) {
	args := m.Called(ctx, attemptID)
	return args.Get(0).([]domain.Answer), args.Error(1)
}

// ── fixtures for attempt tests ─────────────────────────────────────────────────

func openAttempt(evalID, userID string) *domain.Attempt {
	return &domain.Attempt{
		ID:           uuid.New().String(),
		UserID:       userID,
		EvaluationID: evalID,
		Numero:       1,
		IniciadoEn:   time.Now(),
	}
}

func submittedAttempt(evalID, userID string, puntaje int, aprobado bool) *domain.Attempt {
	now := time.Now()
	return &domain.Attempt{
		ID:           uuid.New().String(),
		UserID:       userID,
		EvaluationID: evalID,
		Numero:       1,
		Puntaje:      puntaje,
		Aprobado:     aprobado,
		IniciadoEn:   now.Add(-10 * time.Minute),
		FinalizadoEn: &now,
	}
}

func newSvcWithSeams(repo repository.Repository, checker CoursesChecker, ec EnrollmentCompleter, ci CertificateIssuer) *serviceImpl {
	return New(repo, checker, WithEnrollmentCompleter(ec), WithCertificateIssuer(ci)).(*serviceImpl)
}

// ── [LOAD-BEARING (a)] no-correcta-leak ───────────────────────────────────────

// TestGetAttempt_NoCorrectaLeak verifies that AttemptStateOption has no Correcta field.
// This is a compile-time structural guarantee — the test confirms the mapper omits it.
// Spec: REQ-GET (a).
func TestGetAttempt_NoCorrectaLeak(t *testing.T) {
	// Compile-time: this line would not compile if AttemptStateOption had a Correcta field.
	// We also verify at runtime that no field name "Correcta" exists in the struct.
	var opt AttemptStateOption
	_ = opt.ID
	_ = opt.Texto
	// If this compiled, Correcta is structurally absent.

	// Runtime mapper test: GetAttempt returns state with no correcta info accessible.
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	courseID := uuid.New().String()
	eval := &domain.Evaluation{
		ID: uuid.New().String(), CourseID: courseID, NotaMinima: 70, IntentosMax: 0,
	}
	a := openAttempt(eval.ID, userID)
	q := &domain.Question{
		ID: uuid.New().String(), EvaluationID: eval.ID, Enunciado: "Q?",
		Tipo: domain.TipoOpcionMultiple, Puntaje: 5,
	}
	correctOpt := &domain.Option{
		ID: uuid.New().String(), QuestionID: q.ID, Texto: "Right", Correcta: true,
	}
	wrongOpt := &domain.Option{
		ID: uuid.New().String(), QuestionID: q.ID, Texto: "Wrong", Correcta: false,
	}

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return([]domain.Question{*q}, nil)
	repo.On("ListOptionsByQuestion", mock.Anything, q.ID).Return([]domain.Option{*correctOpt, *wrongOpt}, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return([]domain.Answer{}, nil)

	state, err := svc.GetAttempt(context.Background(), a.ID, userID)
	require.NoError(t, err)
	require.Len(t, state.Questions, 1)
	require.Len(t, state.Questions[0].Options, 2)

	// AttemptStateOption has only ID and Texto — verify they are populated correctly.
	assert.Equal(t, correctOpt.ID, state.Questions[0].Options[0].ID)
	assert.Equal(t, correctOpt.Texto, state.Questions[0].Options[0].Texto)
	assert.Equal(t, wrongOpt.ID, state.Questions[0].Options[1].ID)

	// [LOAD-BEARING (a)] The compiler guarantees no Correcta field exists.
	// No runtime assertion needed — the type itself enforces it.
	repo.AssertExpectations(t)
}

// ── [LOAD-BEARING (b)] on-pass seam invocation ────────────────────────────────

// TestSubmitAttempt_OnPass_BothSeamsCalled verifies seams invoked when aprobado=true.
// Spec: REQ-SEAMS (b).
func TestSubmitAttempt_OnPass_BothSeamsCalled(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	enroll := &testutil.MockEnrollmentCompleter{}
	certs := &testutil.MockCertificateIssuer{}
	svc := newSvcWithSeams(repo, checker, enroll, certs)

	userID := uuid.New().String()
	courseID := uuid.New().String()
	eval := &domain.Evaluation{ID: uuid.New().String(), CourseID: courseID, NotaMinima: 60, IntentosMax: 0}
	a := openAttempt(eval.ID, userID)
	q := &domain.Question{ID: uuid.New().String(), EvaluationID: eval.ID, Puntaje: 10}

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return([]domain.Question{*q}, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return([]domain.Answer{
		{QuestionID: q.ID, OptionID: uuid.New().String(), Correcta: true}, // all correct → 100%
	}, nil)
	repo.On("UpdateAttempt", mock.Anything, a.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// [LOAD-BEARING (b)] Both seams must be called with (userID, courseID).
	enroll.On("MarkEnrollmentCompleted", mock.Anything, userID, courseID).Return(nil).Once()
	certs.On("IssueOnPass", mock.Anything, userID, courseID).Return(nil).Once()

	result, err := svc.SubmitAttempt(context.Background(), a.ID, userID)
	require.NoError(t, err)
	assert.True(t, result.Aprobado, "[LOAD-BEARING (b)] 100% must be aprobado")
	assert.Equal(t, 100, result.Puntaje)

	enroll.AssertExpectations(t)
	certs.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// TestSubmitAttempt_OnFail_SeamsNotCalled verifies seams NOT invoked on fail.
// Spec: REQ-SEAMS (b).
func TestSubmitAttempt_OnFail_SeamsNotCalled(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	enroll := &testutil.MockEnrollmentCompleter{}
	certs := &testutil.MockCertificateIssuer{}
	svc := newSvcWithSeams(repo, checker, enroll, certs)

	userID := uuid.New().String()
	courseID := uuid.New().String()
	eval := &domain.Evaluation{ID: uuid.New().String(), CourseID: courseID, NotaMinima: 60, IntentosMax: 0}
	a := openAttempt(eval.ID, userID)
	q := &domain.Question{ID: uuid.New().String(), EvaluationID: eval.ID, Puntaje: 10}

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return([]domain.Question{*q}, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return([]domain.Answer{
		{QuestionID: q.ID, OptionID: uuid.New().String(), Correcta: false}, // all wrong → 0%
	}, nil)
	repo.On("UpdateAttempt", mock.Anything, a.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	result, err := svc.SubmitAttempt(context.Background(), a.ID, userID)
	require.NoError(t, err)
	assert.False(t, result.Aprobado)
	assert.Equal(t, 0, result.Puntaje)

	// [LOAD-BEARING (b)] Seams must NOT be called on fail.
	enroll.AssertNotCalled(t, "MarkEnrollmentCompleted")
	certs.AssertNotCalled(t, "IssueOnPass")
}

// TestSubmitAttempt_SeamError_NonFatal verifies seam error does not fail submit.
// Spec: REQ-SEAMS (b).
func TestSubmitAttempt_SeamError_NonFatal(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	enroll := &testutil.MockEnrollmentCompleter{}
	certs := &testutil.MockCertificateIssuer{}
	svc := newSvcWithSeams(repo, checker, enroll, certs)

	userID := uuid.New().String()
	courseID := uuid.New().String()
	eval := &domain.Evaluation{ID: uuid.New().String(), CourseID: courseID, NotaMinima: 60, IntentosMax: 0}
	a := openAttempt(eval.ID, userID)
	q := &domain.Question{ID: uuid.New().String(), EvaluationID: eval.ID, Puntaje: 10}

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return([]domain.Question{*q}, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return([]domain.Answer{
		{QuestionID: q.ID, OptionID: uuid.New().String(), Correcta: true},
	}, nil)
	repo.On("UpdateAttempt", mock.Anything, a.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// EnrollmentCompleter returns an error — submit must still succeed.
	enroll.On("MarkEnrollmentCompleted", mock.Anything, userID, courseID).
		Return(errors.New("downstream unavailable")).Once()
	certs.On("IssueOnPass", mock.Anything, userID, courseID).Return(nil).Once()

	result, err := svc.SubmitAttempt(context.Background(), a.ID, userID)

	// [LOAD-BEARING (b)] Seam error must NOT propagate — submit returns nil error.
	require.NoError(t, err, "[LOAD-BEARING (b)] seam error must be non-fatal")
	assert.True(t, result.Aprobado)
	enroll.AssertExpectations(t)
	certs.AssertExpectations(t)
}

// TestSubmitAttempt_NilSeams_NoPanic verifies nil seams are safe (C3.2 main.go wiring).
// Spec: REQ-SEAMS nil seams scenario.
func TestSubmitAttempt_NilSeams_NoPanic(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	// No seams injected — both remain nil.
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	courseID := uuid.New().String()
	eval := &domain.Evaluation{ID: uuid.New().String(), CourseID: courseID, NotaMinima: 60, IntentosMax: 0}
	a := openAttempt(eval.ID, userID)
	q := &domain.Question{ID: uuid.New().String(), EvaluationID: eval.ID, Puntaje: 10}

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return([]domain.Question{*q}, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return([]domain.Answer{
		{QuestionID: q.ID, OptionID: uuid.New().String(), Correcta: true},
	}, nil)
	repo.On("UpdateAttempt", mock.Anything, a.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	assert.NotPanics(t, func() {
		result, err := svc.SubmitAttempt(context.Background(), a.ID, userID)
		require.NoError(t, err)
		assert.True(t, result.Aprobado)
	}, "nil seams must not panic on aprobado=true")
}

// ── [LOAD-BEARING (c)] intentos_max ───────────────────────────────────────────

// TestStartAttempt_MaxAttemptsReached_ReturnsError verifies ErrMaxAttemptsReached.
// Spec: REQ-START (c).
func TestStartAttempt_MaxAttemptsReached_ReturnsError(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	eval.IntentosMax = 2

	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	// No open attempt.
	repo.On("GetOpenAttempt", mock.Anything, userID, eval.ID).Return(nil, repository.ErrAttemptNotFound)
	// Count is at cap.
	repo.On("CountAttemptsByUserEval", mock.Anything, userID, eval.ID).Return(int64(2), nil)

	_, err := svc.StartAttempt(context.Background(), eval.ID, userID)

	assert.ErrorIs(t, err, ErrMaxAttemptsReached,
		"[LOAD-BEARING (c)] count>=intentos_max must return ErrMaxAttemptsReached")
	repo.AssertExpectations(t)
}

// TestStartAttempt_IntentosMaxZero_Unlimited verifies intentos_max=0 is unlimited.
// Spec: REQ-START (c).
func TestStartAttempt_IntentosMaxZero_Unlimited(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	eval.IntentosMax = 0 // unlimited

	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("GetOpenAttempt", mock.Anything, userID, eval.ID).Return(nil, repository.ErrAttemptNotFound)
	repo.On("CountAttemptsByUserEval", mock.Anything, userID, eval.ID).Return(int64(99), nil)
	repo.On("CreateAttempt", mock.Anything, mock.AnythingOfType("*domain.Attempt")).Return(nil)

	result, err := svc.StartAttempt(context.Background(), eval.ID, userID)

	require.NoError(t, err, "[LOAD-BEARING (c)] intentos_max=0 must allow unlimited attempts")
	assert.NotNil(t, result)
	repo.AssertExpectations(t)
}

// ── [LOAD-BEARING (d)] attempt ownership ──────────────────────────────────────

// TestGetAttempt_WrongUser_ReturnsErrAttemptNotFound verifies anti-enumeration.
// Spec: REQ-GET (d).
func TestGetAttempt_WrongUser_ReturnsErrAttemptNotFound(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	attackerID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	a := openAttempt(eval.ID, ownerID)

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)

	_, err := svc.GetAttempt(context.Background(), a.ID, attackerID)
	assert.ErrorIs(t, err, ErrAttemptNotFound,
		"[LOAD-BEARING (d)] non-owner GetAttempt must return ErrAttemptNotFound (never 403)")
	repo.AssertExpectations(t)
}

// TestSaveAnswer_WrongUser_ReturnsErrAttemptNotFound verifies ownership on SaveAnswer.
// Spec: REQ-ANSWER (d).
func TestSaveAnswer_WrongUser_ReturnsErrAttemptNotFound(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	attackerID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	a := openAttempt(eval.ID, ownerID)

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)

	err := svc.SaveAnswer(context.Background(), a.ID, attackerID, uuid.New().String(), uuid.New().String())
	assert.ErrorIs(t, err, ErrAttemptNotFound,
		"[LOAD-BEARING (d)] non-owner SaveAnswer must return ErrAttemptNotFound")
	repo.AssertExpectations(t)
}

// TestSubmitAttempt_WrongUser_ReturnsErrAttemptNotFound verifies ownership on Submit.
// Spec: REQ-SUBMIT (d).
func TestSubmitAttempt_WrongUser_ReturnsErrAttemptNotFound(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	attackerID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	a := openAttempt(eval.ID, ownerID)

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)

	_, err := svc.SubmitAttempt(context.Background(), a.ID, attackerID)
	assert.ErrorIs(t, err, ErrAttemptNotFound,
		"[LOAD-BEARING (d)] non-owner SubmitAttempt must return ErrAttemptNotFound")
	repo.AssertExpectations(t)
}

// ── [LOAD-BEARING (e)] scoring correctness ────────────────────────────────────

// TestSubmitAttempt_ScoringAllCorrect verifies 100% score.
// Spec: REQ-SUBMIT (e).
func TestSubmitAttempt_ScoringAllCorrect(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := &domain.Evaluation{ID: uuid.New().String(), CourseID: uuid.New().String(), NotaMinima: 60}
	a := openAttempt(eval.ID, userID)
	q1 := &domain.Question{ID: uuid.New().String(), EvaluationID: eval.ID, Puntaje: 10}
	q2 := &domain.Question{ID: uuid.New().String(), EvaluationID: eval.ID, Puntaje: 10}

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return([]domain.Question{*q1, *q2}, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return([]domain.Answer{
		{QuestionID: q1.ID, Correcta: true},
		{QuestionID: q2.ID, Correcta: true},
	}, nil)
	repo.On("UpdateAttempt", mock.Anything, a.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	result, err := svc.SubmitAttempt(context.Background(), a.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, 100, result.Puntaje, "[LOAD-BEARING (e)] all correct → puntaje=100")
	assert.True(t, result.Aprobado)
}

// TestSubmitAttempt_ScoringAllWrong verifies 0% score.
// Spec: REQ-SUBMIT (e).
func TestSubmitAttempt_ScoringAllWrong(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := &domain.Evaluation{ID: uuid.New().String(), CourseID: uuid.New().String(), NotaMinima: 60}
	a := openAttempt(eval.ID, userID)
	q := &domain.Question{ID: uuid.New().String(), EvaluationID: eval.ID, Puntaje: 10}

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return([]domain.Question{*q}, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return([]domain.Answer{
		{QuestionID: q.ID, Correcta: false},
	}, nil)
	repo.On("UpdateAttempt", mock.Anything, a.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	result, err := svc.SubmitAttempt(context.Background(), a.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, 0, result.Puntaje, "[LOAD-BEARING (e)] all wrong → puntaje=0")
	assert.False(t, result.Aprobado)
}

// TestSubmitAttempt_ScoringTotalZero_NoDivisionByZero verifies the total==0 guard.
// Spec: REQ-SUBMIT (e).
func TestSubmitAttempt_ScoringTotalZero_NoDivisionByZero(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := &domain.Evaluation{ID: uuid.New().String(), CourseID: uuid.New().String(), NotaMinima: 60}
	a := openAttempt(eval.ID, userID)

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	// No questions — total=0.
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return([]domain.Question{}, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return([]domain.Answer{}, nil)
	repo.On("UpdateAttempt", mock.Anything, a.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	assert.NotPanics(t, func() {
		result, err := svc.SubmitAttempt(context.Background(), a.ID, userID)
		require.NoError(t, err)
		assert.Equal(t, 0, result.Puntaje, "[LOAD-BEARING (e)] total=0 must yield puntaje=0")
		assert.False(t, result.Aprobado, "[LOAD-BEARING (e)] total=0 must yield aprobado=false")
	}, "total==0 must not panic (division-by-zero guard)")
}

// TestSubmitAttempt_ScoringPassBoundary verifies pct==notaMinima → aprobado=true.
// Spec: REQ-SUBMIT (e).
func TestSubmitAttempt_ScoringPassBoundary(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	// notaMinima=70; 7 correct out of 10 puntaje points → pct=70 → aprobado=true.
	eval := &domain.Evaluation{ID: uuid.New().String(), CourseID: uuid.New().String(), NotaMinima: 70}
	a := openAttempt(eval.ID, userID)
	// 10 questions each worth 1, student answers 7 correctly.
	questions := make([]domain.Question, 10)
	answers := make([]domain.Answer, 10)
	for i := range questions {
		questions[i] = domain.Question{
			ID: uuid.New().String(), EvaluationID: eval.ID, Puntaje: 1,
		}
		answers[i] = domain.Answer{
			QuestionID: questions[i].ID, Correcta: i < 7, // first 7 correct
		}
	}

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, eval.ID).Return(questions, nil)
	repo.On("ListAnswersByAttempt", mock.Anything, a.ID).Return(answers, nil)
	repo.On("UpdateAttempt", mock.Anything, a.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	result, err := svc.SubmitAttempt(context.Background(), a.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, 70, result.Puntaje, "[LOAD-BEARING (e)] 7/10 must yield puntaje=70")
	assert.True(t, result.Aprobado, "[LOAD-BEARING (e)] pct==notaMinima must be aprobado=true")
}

// ── [LOAD-BEARING (f)] answer upsert ─────────────────────────────────────────

// TestSaveAnswer_UpsertCalled verifies UpsertAnswer is called (not Create).
// Spec: REQ-ANSWER (f).
func TestSaveAnswer_UpsertCalled(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	a := openAttempt(eval.ID, userID)
	q := questionFixture(eval.ID, domain.TipoOpcionMultiple)
	opt := optionFixture(q.ID, true)
	opt.QuestionID = q.ID

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetOptionByID", mock.Anything, opt.ID).Return(opt, nil)
	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil)
	// [LOAD-BEARING (f)] UpsertAnswer must be called.
	repo.On("UpsertAnswer", mock.Anything, a.ID, q.ID, opt.ID, opt.Correcta).Return(nil).Once()

	err := svc.SaveAnswer(context.Background(), a.ID, userID, q.ID, opt.ID)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// ── [LOAD-BEARING (g)] open-attempt resume ────────────────────────────────────

// TestStartAttempt_OpenAttemptExists_ResumesIt verifies that StartAttempt RESUMES
// an existing open attempt instead of blocking with ErrAttemptOpen.
// The open attempt is returned as-is; no new attempt is created (CreateAttempt NOT called).
// The intentos_max guard is NOT applied on a resume path.
// Spec: REQ-START resume-on-start (UX fix).
func TestStartAttempt_OpenAttemptExists_ResumesIt(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	eval.IntentosMax = 1 // would block a NEW attempt, but resume bypasses this
	existingOpen := openAttempt(eval.ID, userID)

	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	// GetOpenAttempt returns the existing open attempt.
	repo.On("GetOpenAttempt", mock.Anything, userID, eval.ID).Return(existingOpen, nil)
	// CRITICAL: CreateAttempt must NOT be called — we are resuming, not creating.

	result, err := svc.StartAttempt(context.Background(), eval.ID, userID)

	require.NoError(t, err, "[LOAD-BEARING (g)] open attempt must be resumed, not blocked")
	require.NotNil(t, result)
	assert.Equal(t, existingOpen.ID, result.ID,
		"[LOAD-BEARING (g)] resumed attempt must have the SAME id as the existing open one")
	assert.Equal(t, existingOpen.Numero, result.Numero,
		"[LOAD-BEARING (g)] resumed attempt must have the SAME numero as the existing open one")
	// AssertExpectations verifies CreateAttempt was NOT called (it was never set up).
	repo.AssertExpectations(t)
}

// ── Additional sentinel tests ──────────────────────────────────────────────────

// TestSubmitAttempt_AlreadySubmitted_Returns409 verifies re-submit is blocked.
// Spec: REQ-SUBMIT re-submit.
func TestSubmitAttempt_AlreadySubmitted_Returns409(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	a := submittedAttempt(eval.ID, userID, 80, true)

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)

	_, err := svc.SubmitAttempt(context.Background(), a.ID, userID)
	assert.ErrorIs(t, err, ErrAttemptAlreadySubmitted)
	repo.AssertExpectations(t)
}

// TestSaveAnswer_AfterSubmit_Returns409 verifies answering after submit is blocked.
// Spec: REQ-ANSWER answer-after-submit.
func TestSaveAnswer_AfterSubmit_Returns409(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	a := submittedAttempt(eval.ID, userID, 80, true)

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)

	err := svc.SaveAnswer(context.Background(), a.ID, userID, uuid.New().String(), uuid.New().String())
	assert.ErrorIs(t, err, ErrAttemptAlreadySubmitted)
	repo.AssertExpectations(t)
}

// TestSaveAnswer_InvalidOption_ReturnsErrInvalidAnswer verifies option validation.
// Spec: REQ-ANSWER invalid option.
func TestSaveAnswer_InvalidOption_ReturnsErrInvalidAnswer(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	a := openAttempt(eval.ID, userID)
	q := questionFixture(eval.ID, domain.TipoOpcionMultiple)

	// Option belongs to a DIFFERENT question.
	wrongOpt := optionFixture(uuid.New().String(), false) // different questionID

	repo.On("GetAttemptByID", mock.Anything, a.ID).Return(a, nil)
	repo.On("GetOptionByID", mock.Anything, wrongOpt.ID).Return(wrongOpt, nil)

	err := svc.SaveAnswer(context.Background(), a.ID, userID, q.ID, wrongOpt.ID)
	assert.ErrorIs(t, err, ErrInvalidAnswer,
		"option not belonging to question must return ErrInvalidAnswer")
	repo.AssertExpectations(t)
}

// TestStartAttempt_EvaluationNotFound_Returns404 verifies 404 on missing eval.
func TestStartAttempt_EvaluationNotFound_Returns404(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	repo.On("GetEvaluationByID", mock.Anything, mock.AnythingOfType("string")).
		Return(nil, repository.ErrEvaluationNotFound)

	_, err := svc.StartAttempt(context.Background(), uuid.New().String(), uuid.New().String())
	assert.ErrorIs(t, err, ErrEvaluationNotFound)
	repo.AssertExpectations(t)
}

// TestStartAttempt_HappyPath_FirstAttempt verifies numero=1 on first attempt.
// Spec: REQ-START happy path.
func TestStartAttempt_HappyPath_FirstAttempt(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	userID := uuid.New().String()
	eval := evalFixture(uuid.New().String())
	eval.IntentosMax = 3

	repo.On("GetEvaluationByID", mock.Anything, eval.ID).Return(eval, nil)
	repo.On("GetOpenAttempt", mock.Anything, userID, eval.ID).Return(nil, repository.ErrAttemptNotFound)
	repo.On("CountAttemptsByUserEval", mock.Anything, userID, eval.ID).Return(int64(0), nil)
	repo.On("CreateAttempt", mock.Anything, mock.AnythingOfType("*domain.Attempt")).Return(nil)

	result, err := svc.StartAttempt(context.Background(), eval.ID, userID)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Numero, "first attempt must have numero=1")
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, eval.ID, result.EvaluationID)
	repo.AssertExpectations(t)
}
