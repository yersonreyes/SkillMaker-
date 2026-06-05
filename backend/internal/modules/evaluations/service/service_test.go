// Package service — white-box unit tests for the evaluations service.
// No build tag: runs with standard `make backend-test`.
//
// Strategy: inject MockCoursesChecker (from testutil) + a local MockEvalRepo
// (testify/mock) so no real DB is needed.
//
// LOAD-BEARING tests:
//
//	(a) Non-owner → ErrNotOwner (403 write)
//	(b) Estado gate aprobado/en_revision → ErrCourseNotEditable (409); borrador/rechazado pass
//	(c) Auto V/F: CreateQuestion(verdadero_falso) → CreateOptions called with exactly 2 options
//	(d) 1-1 unique: repo returns ErrEvaluationExists → service surfaces → 409
//	(f) validateQuestionComplete: present + tested; NOT wired to mutating endpoint in C3.1
//
// Additional: ownership FIRST (non-owner on non-editable → ErrNotOwner not ErrCourseNotEditable).
// ErrInvalidQuestionType on bad tipo. CreateOption on V/F question → ErrInvalidQuestionType.
// opcion_multiple gets 0 auto-options.
package service

import (
	"context"
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

// ── MockEvalRepo ───────────────────────────────────────────────────────────────

// mockEvalRepo is a local testify/mock implementation of repository.Repository.
// Defined here (not in testutil) following the courses module pattern — repo mock
// stays close to the service tests that use it.
type mockEvalRepo struct {
	mock.Mock
}

func (m *mockEvalRepo) CreateEvaluation(ctx context.Context, e *domain.Evaluation) error {
	args := m.Called(ctx, e)
	return args.Error(0)
}

func (m *mockEvalRepo) GetEvaluationByID(ctx context.Context, id string) (*domain.Evaluation, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Evaluation), args.Error(1)
}

func (m *mockEvalRepo) GetEvaluationByCourse(ctx context.Context, courseID string) (*domain.Evaluation, error) {
	args := m.Called(ctx, courseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Evaluation), args.Error(1)
}

func (m *mockEvalRepo) UpdateEvaluation(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *mockEvalRepo) CreateQuestion(ctx context.Context, q *domain.Question) error {
	args := m.Called(ctx, q)
	return args.Error(0)
}

func (m *mockEvalRepo) GetQuestionByID(ctx context.Context, id string) (*domain.Question, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Question), args.Error(1)
}

func (m *mockEvalRepo) UpdateQuestion(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *mockEvalRepo) DeleteQuestion(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockEvalRepo) ListQuestionsByEvaluation(ctx context.Context, evaluationID string) ([]domain.Question, error) {
	args := m.Called(ctx, evaluationID)
	return args.Get(0).([]domain.Question), args.Error(1)
}

func (m *mockEvalRepo) CreateOptions(ctx context.Context, opts []domain.Option) error {
	args := m.Called(ctx, opts)
	return args.Error(0)
}

func (m *mockEvalRepo) GetOptionByID(ctx context.Context, id string) (*domain.Option, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Option), args.Error(1)
}

func (m *mockEvalRepo) UpdateOption(ctx context.Context, id string, fields map[string]any) error {
	args := m.Called(ctx, id, fields)
	return args.Error(0)
}

func (m *mockEvalRepo) DeleteOption(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockEvalRepo) ListOptionsByQuestion(ctx context.Context, questionID string) ([]domain.Option, error) {
	args := m.Called(ctx, questionID)
	return args.Get(0).([]domain.Option), args.Error(1)
}

// ── Fixtures ───────────────────────────────────────────────────────────────────

func evalFixture(courseID string) *domain.Evaluation {
	return &domain.Evaluation{
		ID:          uuid.New().String(),
		CourseID:    courseID,
		NotaMinima:  70,
		IntentosMax: 3,
		CreatedAt:   time.Now(),
	}
}

func questionFixture(evalID string, tipo domain.TipoPregunta) *domain.Question {
	return &domain.Question{
		ID:           uuid.New().String(),
		EvaluationID: evalID,
		Enunciado:    "Test question",
		Tipo:         tipo,
		Puntaje:      5,
		Orden:        0,
		CreatedAt:    time.Now(),
	}
}

func optionFixture(questionID string, correcta bool) *domain.Option {
	return &domain.Option{
		ID:         uuid.New().String(),
		QuestionID: questionID,
		Texto:      "Option text",
		Correcta:   correcta,
		Orden:      0,
		CreatedAt:  time.Now(),
	}
}

// newSvc builds a serviceImpl with the given mocks (exported enough for white-box tests
// since this file is in the same package).
func newSvc(repo repository.Repository, checker CoursesChecker) *serviceImpl {
	return New(repo, checker).(*serviceImpl)
}

// ── [LOAD-BEARING (a)] Non-owner → ErrNotOwner ────────────────────────────────

// TestCreateEvaluation_NonOwner_ReturnsErrNotOwner verifies that a non-owner gets ErrNotOwner.
// Spec: CMO-2-A, EVL-1-C.
func TestCreateEvaluation_NonOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	requesterID := uuid.New().String() // different creador
	courseID := uuid.New().String()

	checker.On("GetCourseOwnership", mock.Anything, courseID).
		Return(ownerID, "borrador", nil)

	_, err := svc.CreateEvaluation(context.Background(), courseID, requesterID, EvaluationCreateRequest{})

	assert.ErrorIs(t, err, ErrNotOwner,
		"[LOAD-BEARING (a)] non-owner must get ErrNotOwner on CreateEvaluation")
	checker.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// TestCreateQuestion_NonOwner_ReturnsErrNotOwner verifies traversal-based ownership check.
// Spec: CMO-2-A, QST-1-D.
func TestCreateQuestion_NonOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	requesterID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)

	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	_, err := svc.CreateQuestion(context.Background(), e.ID, requesterID, QuestionCreateRequest{
		Enunciado: "Q?",
		Tipo:      string(domain.TipoOpcionMultiple),
		Puntaje:   ptrInt(5),
	})

	assert.ErrorIs(t, err, ErrNotOwner,
		"[LOAD-BEARING (a)] non-owner question creation must return ErrNotOwner")
	checker.AssertExpectations(t)
}

// ── [LOAD-BEARING (b)] Estado gate ────────────────────────────────────────────

// TestCreateEvaluation_AprobadoCourse_ReturnsErrCourseNotEditable verifies estado gate.
// Spec: EVL-1-E, CMO-2-B.
func TestCreateEvaluation_AprobadoCourse_ReturnsErrCourseNotEditable(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()

	checker.On("GetCourseOwnership", mock.Anything, courseID).
		Return(ownerID, "aprobado", nil)

	_, err := svc.CreateEvaluation(context.Background(), courseID, ownerID, EvaluationCreateRequest{})

	assert.ErrorIs(t, err, ErrCourseNotEditable,
		"[LOAD-BEARING (b)] aprobado course must return ErrCourseNotEditable")
	checker.AssertExpectations(t)
}

// TestCreateEvaluation_EnRevisionCourse_ReturnsErrCourseNotEditable verifies en_revision estado.
func TestCreateEvaluation_EnRevisionCourse_ReturnsErrCourseNotEditable(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "en_revision", nil)

	_, err := svc.CreateEvaluation(context.Background(), courseID, ownerID, EvaluationCreateRequest{})
	assert.ErrorIs(t, err, ErrCourseNotEditable)
	checker.AssertExpectations(t)
}

// TestCreateEvaluation_BorradorCourse_Passes verifies borrador allows creation.
func TestCreateEvaluation_BorradorCourse_Passes(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("CreateEvaluation", mock.Anything, mock.AnythingOfType("*domain.Evaluation")).Return(nil)

	result, err := svc.CreateEvaluation(context.Background(), courseID, ownerID, EvaluationCreateRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	checker.AssertExpectations(t)
}

// TestCreateEvaluation_RechazadoCourse_Passes verifies rechazado allows creation.
func TestCreateEvaluation_RechazadoCourse_Passes(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "rechazado", nil)
	repo.On("CreateEvaluation", mock.Anything, mock.AnythingOfType("*domain.Evaluation")).Return(nil)

	result, err := svc.CreateEvaluation(context.Background(), courseID, ownerID, EvaluationCreateRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// ── [LOAD-BEARING (b+a) ordering] Non-owner on non-editable → ErrNotOwner ────

// TestCreateEvaluation_NonOwnerAprobado_ReturnsErrNotOwner verifies ordering:
// ownership FIRST (non-owner on non-editable → ErrNotOwner, NOT ErrCourseNotEditable).
// Mirrors courses service LOAD-BEARING-B.
func TestCreateEvaluation_NonOwnerAprobado_ReturnsErrNotOwner(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	requesterID := uuid.New().String() // NOT the owner
	courseID := uuid.New().String()

	checker.On("GetCourseOwnership", mock.Anything, courseID).
		Return(ownerID, "aprobado", nil) // non-editable AND non-owner

	_, err := svc.CreateEvaluation(context.Background(), courseID, requesterID, EvaluationCreateRequest{})

	assert.ErrorIs(t, err, ErrNotOwner,
		"[LOAD-BEARING ordering] non-owner on aprobado course must get ErrNotOwner, not ErrCourseNotEditable")
	assert.NotErrorIs(t, err, ErrCourseNotEditable,
		"[LOAD-BEARING ordering] must NOT be ErrCourseNotEditable — ownership outranks estado")
	checker.AssertExpectations(t)
}

// ── [LOAD-BEARING (c)] Auto V/F creates exactly 2 options ────────────────────

// TestCreateQuestion_VerdaderoFalso_AutoCreates2Options verifies auto V/F option creation.
// Spec: QST-1-B.
func TestCreateQuestion_VerdaderoFalso_AutoCreates2Options(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)

	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("CreateQuestion", mock.Anything, mock.AnythingOfType("*domain.Question")).Return(nil)

	// Capture the options passed to CreateOptions.
	var capturedOpts []domain.Option
	repo.On("CreateOptions", mock.Anything, mock.AnythingOfType("[]domain.Option")).
		Run(func(args mock.Arguments) {
			capturedOpts = args.Get(1).([]domain.Option)
		}).
		Return(nil)

	// Stub ListOptionsByQuestion for the return value.
	repo.On("ListOptionsByQuestion", mock.Anything, mock.AnythingOfType("string")).
		Return([]domain.Option{
			{ID: uuid.New().String(), QuestionID: "q", Texto: "Verdadero", Correcta: false, Orden: 0},
			{ID: uuid.New().String(), QuestionID: "q", Texto: "Falso", Correcta: false, Orden: 1},
		}, nil)

	result, err := svc.CreateQuestion(context.Background(), e.ID, ownerID, QuestionCreateRequest{
		Enunciado: "Is Go compiled?",
		Tipo:      string(domain.TipoVerdaderoFalso),
		Puntaje:   ptrInt(5),
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// [LOAD-BEARING (c)] Exactly 2 options must be created automatically.
	require.Len(t, capturedOpts, 2,
		"[LOAD-BEARING (c)] verdadero_falso must auto-create exactly 2 options")
	assert.Equal(t, "Verdadero", capturedOpts[0].Texto)
	assert.Equal(t, "Falso", capturedOpts[1].Texto)
	assert.False(t, capturedOpts[0].Correcta, "auto-options must start with correcta=false")
	assert.False(t, capturedOpts[1].Correcta, "auto-options must start with correcta=false")
	assert.Equal(t, 0, capturedOpts[0].Orden)
	assert.Equal(t, 1, capturedOpts[1].Orden)

	checker.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// TestCreateQuestion_OpcionMultiple_NoAutoOptions verifies opcion_multiple gets 0 options.
// Spec: QST-1-A.
func TestCreateQuestion_OpcionMultiple_NoAutoOptions(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)

	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("CreateQuestion", mock.Anything, mock.AnythingOfType("*domain.Question")).Return(nil)
	// ListOptionsByQuestion returns empty for opcion_multiple.
	repo.On("ListOptionsByQuestion", mock.Anything, mock.AnythingOfType("string")).
		Return([]domain.Option{}, nil)

	result, err := svc.CreateQuestion(context.Background(), e.ID, ownerID, QuestionCreateRequest{
		Enunciado: "What is Go?",
		Tipo:      string(domain.TipoOpcionMultiple),
		Puntaje:   ptrInt(5),
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	// CreateOptions must NOT be called for opcion_multiple.
	repo.AssertNotCalled(t, "CreateOptions")
	assert.Len(t, result.Options, 0, "opcion_multiple must have 0 auto-options")
	checker.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// ── [LOAD-BEARING (d)] EvaluationExists propagation ──────────────────────────

// TestCreateEvaluation_EvalExists_ReturnsErrEvaluationExists verifies 1-1 constraint.
// Spec: EVL-1-B.
func TestCreateEvaluation_EvalExists_ReturnsErrEvaluationExists(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("CreateEvaluation", mock.Anything, mock.AnythingOfType("*domain.Evaluation")).
		Return(repository.ErrEvaluationExists)

	_, err := svc.CreateEvaluation(context.Background(), courseID, ownerID, EvaluationCreateRequest{})

	assert.ErrorIs(t, err, ErrEvaluationExists,
		"[LOAD-BEARING (d)] repo.ErrEvaluationExists must surface as service.ErrEvaluationExists")
	checker.AssertExpectations(t)
}

// ── [LOAD-BEARING (f)] validateQuestionComplete ───────────────────────────────

// TestValidateQuestionComplete_NoCorrectOption_ReturnsErrNoCorrectOption verifies sentinel.
// Spec: OPT-1-B, ADR-5 Decision 5.
func TestValidateQuestionComplete_NoCorrectOption_ReturnsErrNoCorrectOption(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	questionID := uuid.New().String()
	// No options have correcta=true.
	repo.On("ListOptionsByQuestion", mock.Anything, questionID).
		Return([]domain.Option{
			{ID: uuid.New().String(), QuestionID: questionID, Texto: "A", Correcta: false},
			{ID: uuid.New().String(), QuestionID: questionID, Texto: "B", Correcta: false},
		}, nil)

	err := svc.validateQuestionComplete(context.Background(), questionID)

	assert.ErrorIs(t, err, ErrNoCorrectOption,
		"[LOAD-BEARING (f)] question with no correct option must return ErrNoCorrectOption")
	repo.AssertExpectations(t)
}

// TestValidateQuestionComplete_HasCorrectOption_ReturnsNil verifies the happy path.
func TestValidateQuestionComplete_HasCorrectOption_ReturnsNil(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	questionID := uuid.New().String()
	repo.On("ListOptionsByQuestion", mock.Anything, questionID).
		Return([]domain.Option{
			{ID: uuid.New().String(), QuestionID: questionID, Texto: "A", Correcta: true},
			{ID: uuid.New().String(), QuestionID: questionID, Texto: "B", Correcta: false},
		}, nil)

	err := svc.validateQuestionComplete(context.Background(), questionID)
	assert.NoError(t, err, "question with at least one correct option must pass validateQuestionComplete")
	repo.AssertExpectations(t)
}

// ── ErrInvalidQuestionType tests ──────────────────────────────────────────────

// TestCreateQuestion_InvalidTipo_ReturnsErrInvalidQuestionType verifies bad tipo rejection.
// Spec: QST-1-C.
func TestCreateQuestion_InvalidTipo_ReturnsErrInvalidQuestionType(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)

	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	_, err := svc.CreateQuestion(context.Background(), e.ID, ownerID, QuestionCreateRequest{
		Enunciado: "Q?",
		Tipo:      "abierta", // invalid tipo
		Puntaje:   ptrInt(5),
	})

	assert.ErrorIs(t, err, ErrInvalidQuestionType, "invalid tipo must return ErrInvalidQuestionType")
	checker.AssertExpectations(t)
}

// TestCreateOption_OnVerdaderoFalsoQuestion_ReturnsErrInvalidQuestionType verifies guard.
// V/F questions have fixed 2 options — additional CreateOption must be rejected.
func TestCreateOption_OnVerdaderoFalsoQuestion_ReturnsErrInvalidQuestionType(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoVerdaderoFalso)

	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil)
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	_, err := svc.CreateOption(context.Background(), q.ID, ownerID, OptionCreateRequest{
		Texto:    "Extra option",
		Correcta: ptrBool(true),
	})

	assert.ErrorIs(t, err, ErrInvalidQuestionType,
		"CreateOption on verdadero_falso question must return ErrInvalidQuestionType")
	checker.AssertExpectations(t)
}

// ── GetEvaluation read-gate tests ─────────────────────────────────────────────

// TestGetEvaluation_NonOwner_ReturnsErrNotOwner verifies read-gate for non-owner.
// Spec: EVL-2-C.
func TestGetEvaluation_NonOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	requesterID := uuid.New().String() // different creador
	courseID := uuid.New().String()

	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	_, err := svc.GetEvaluation(context.Background(), courseID, requesterID)
	assert.ErrorIs(t, err, ErrNotOwner,
		"GetEvaluation non-owner must return ErrNotOwner (handler maps to 404)")
	checker.AssertExpectations(t)
}

// TestGetEvaluation_EvalNotFound_ReturnsErrEvaluationNotFound verifies 404 on missing eval.
// Spec: EVL-2-B.
func TestGetEvaluation_EvalNotFound_ReturnsErrEvaluationNotFound(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("GetEvaluationByCourse", mock.Anything, courseID).Return(nil, repository.ErrEvaluationNotFound)

	_, err := svc.GetEvaluation(context.Background(), courseID, ownerID)
	assert.ErrorIs(t, err, ErrEvaluationNotFound)
	checker.AssertExpectations(t)
}

// ── DeleteQuestion tests ───────────────────────────────────────────────────────

// TestDeleteQuestion_Owner_OK verifies owner can delete their question.
func TestDeleteQuestion_Owner_OK(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoOpcionMultiple)

	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil)
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("DeleteQuestion", mock.Anything, q.ID).Return(nil)

	err := svc.DeleteQuestion(context.Background(), q.ID, ownerID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// ── ValidateEvaluationComplete tests ──────────────────────────────────────────

// TestValidateEvaluationComplete_AllQuestionsComplete_ReturnsNil verifies public helper.
func TestValidateEvaluationComplete_AllQuestionsComplete_ReturnsNil(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	evalID := uuid.New().String()
	q1ID := uuid.New().String()
	q2ID := uuid.New().String()

	repo.On("ListQuestionsByEvaluation", mock.Anything, evalID).Return([]domain.Question{
		{ID: q1ID, EvaluationID: evalID, Tipo: domain.TipoOpcionMultiple},
		{ID: q2ID, EvaluationID: evalID, Tipo: domain.TipoVerdaderoFalso},
	}, nil)
	// q1 has a correct option.
	repo.On("ListOptionsByQuestion", mock.Anything, q1ID).Return([]domain.Option{
		{ID: uuid.New().String(), QuestionID: q1ID, Correcta: true},
	}, nil)
	// q2 has a correct option.
	repo.On("ListOptionsByQuestion", mock.Anything, q2ID).Return([]domain.Option{
		{ID: uuid.New().String(), QuestionID: q2ID, Correcta: true},
	}, nil)

	err := svc.ValidateEvaluationComplete(context.Background(), evalID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// TestValidateEvaluationComplete_OneQuestionMissingCorrectOption_ReturnsErr verifies failure case.
func TestValidateEvaluationComplete_OneQuestionMissingCorrectOption_ReturnsErr(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	evalID := uuid.New().String()
	q1ID := uuid.New().String()

	repo.On("ListQuestionsByEvaluation", mock.Anything, evalID).Return([]domain.Question{
		{ID: q1ID, EvaluationID: evalID, Tipo: domain.TipoOpcionMultiple},
	}, nil)
	repo.On("ListOptionsByQuestion", mock.Anything, q1ID).Return([]domain.Option{
		{ID: uuid.New().String(), QuestionID: q1ID, Correcta: false},
	}, nil)

	err := svc.ValidateEvaluationComplete(context.Background(), evalID)
	assert.ErrorIs(t, err, ErrNoCorrectOption)
	repo.AssertExpectations(t)
}

// ── UpdateOption tests ─────────────────────────────────────────────────────────

// TestUpdateOption_Owner_OK verifies owner can update an option on a V/F question.
// The sibling option has correcta=false, so no sibling update is triggered.
func TestUpdateOption_Owner_OK(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoVerdaderoFalso)
	o := optionFixture(q.ID, false)
	sibling := optionFixture(q.ID, false) // sibling already false — no update needed

	repo.On("GetOptionByID", mock.Anything, o.ID).Return(o, nil).Once()
	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil)
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("UpdateOption", mock.Anything, o.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)
	// OPT-2 sibling check: ListOptionsByQuestion returns sibling already false → no sibling UpdateOption.
	repo.On("ListOptionsByQuestion", mock.Anything, q.ID).Return([]domain.Option{
		{ID: o.ID, QuestionID: q.ID, Texto: "Verdadero", Correcta: false},
		{ID: sibling.ID, QuestionID: q.ID, Texto: "Falso", Correcta: false},
	}, nil)
	updatedOpt := *o
	updatedOpt.Correcta = true
	repo.On("GetOptionByID", mock.Anything, o.ID).Return(&updatedOpt, nil).Once()

	result, err := svc.UpdateOption(context.Background(), o.ID, ownerID, OptionUpdateRequest{
		Correcta: ptrBool(true),
	})
	require.NoError(t, err)
	assert.True(t, result.Correcta)
	repo.AssertExpectations(t)
}

// ── UpdateEvaluation tests ─────────────────────────────────────────────────────

// TestUpdateEvaluation_Owner_OK verifies happy-path update.
func TestUpdateEvaluation_Owner_OK(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)

	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil).Once()
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("UpdateEvaluation", mock.Anything, e.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)
	updatedE := *e
	updatedE.NotaMinima = 85
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(&updatedE, nil).Once()

	result, err := svc.UpdateEvaluation(context.Background(), e.ID, ownerID, EvaluationUpdateRequest{
		NotaMinima: ptrInt(85),
	})
	require.NoError(t, err)
	assert.Equal(t, 85, result.NotaMinima)
	repo.AssertExpectations(t)
}

// TestUpdateEvaluation_NonOwner_ReturnsErrNotOwner verifies non-owner rejection.
func TestUpdateEvaluation_NonOwner_ReturnsErrNotOwner(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	requesterID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)

	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	_, err := svc.UpdateEvaluation(context.Background(), e.ID, requesterID, EvaluationUpdateRequest{
		NotaMinima: ptrInt(80),
	})
	assert.ErrorIs(t, err, ErrNotOwner)
}

// TestUpdateEvaluation_NotEditable_ReturnsErrCourseNotEditable verifies estado gate.
func TestUpdateEvaluation_NotEditable_ReturnsErrCourseNotEditable(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)

	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "aprobado", nil)

	_, err := svc.UpdateEvaluation(context.Background(), e.ID, ownerID, EvaluationUpdateRequest{
		NotaMinima: ptrInt(80),
	})
	assert.ErrorIs(t, err, ErrCourseNotEditable)
}

// ── UpdateQuestion tests ───────────────────────────────────────────────────────

// TestUpdateQuestion_Owner_OK verifies happy-path question update.
func TestUpdateQuestion_Owner_OK(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoOpcionMultiple)

	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil).Once()
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("UpdateQuestion", mock.Anything, q.ID, mock.AnythingOfType("map[string]interface {}")).Return(nil)
	updatedQ := *q
	updatedQ.Puntaje = 20
	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(&updatedQ, nil).Once()

	result, err := svc.UpdateQuestion(context.Background(), q.ID, ownerID, QuestionUpdateRequest{
		Puntaje: ptrInt(20),
	})
	require.NoError(t, err)
	assert.Equal(t, 20, result.Puntaje)
	repo.AssertExpectations(t)
}

// ── DeleteOption tests ─────────────────────────────────────────────────────────

// TestDeleteOption_Owner_OK verifies owner can delete an option.
func TestDeleteOption_Owner_OK(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoOpcionMultiple)
	o := optionFixture(q.ID, false)

	repo.On("GetOptionByID", mock.Anything, o.ID).Return(o, nil)
	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil)
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("DeleteOption", mock.Anything, o.ID).Return(nil)

	err := svc.DeleteOption(context.Background(), o.ID, ownerID)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// ── GetEvaluation full-tree compose test ──────────────────────────────────────

// TestGetEvaluation_Owner_ReturnsNestedTree verifies GetEvaluation composes the full tree.
func TestGetEvaluation_Owner_ReturnsNestedTree(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoOpcionMultiple)
	opt := optionFixture(q.ID, true)

	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("GetEvaluationByCourse", mock.Anything, courseID).Return(e, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, e.ID).Return([]domain.Question{*q}, nil)
	repo.On("ListOptionsByQuestion", mock.Anything, q.ID).Return([]domain.Option{*opt}, nil)

	detail, err := svc.GetEvaluation(context.Background(), courseID, ownerID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, e.ID, detail.ID)
	require.Len(t, detail.Questions, 1)
	require.Len(t, detail.Questions[0].Options, 1)
	assert.True(t, detail.Questions[0].Options[0].Correcta)
	checker.AssertExpectations(t)
	repo.AssertExpectations(t)
}

// ── CreateOption happy path test ───────────────────────────────────────────────

// TestCreateOption_OpcionMultiple_OK verifies adding an option to opcion_multiple.
func TestCreateOption_OpcionMultiple_OK(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoOpcionMultiple)

	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil)
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)
	repo.On("CreateOptions", mock.Anything, mock.AnythingOfType("[]domain.Option")).Return(nil)
	createdOpt := &domain.Option{ID: uuid.New().String(), QuestionID: q.ID, Texto: "Paris", Correcta: true}
	repo.On("GetOptionByID", mock.Anything, mock.AnythingOfType("string")).Return(createdOpt, nil)

	result, err := svc.CreateOption(context.Background(), q.ID, ownerID, OptionCreateRequest{
		Texto:    "Paris",
		Correcta: ptrBool(true),
	})
	require.NoError(t, err)
	assert.Equal(t, "Paris", result.Texto)
	assert.True(t, result.Correcta)
	repo.AssertExpectations(t)
}

// ── OPT-2 V/F mutual-exclusion tests ──────────────────────────────────────────

// TestUpdateOption_VF_SetVerdadero_ClearsFalso verifies OPT-2-A:
// When "Verdadero" option is set correcta=true, the sibling "Falso" (currently true)
// is automatically set to correcta=false.
// Spec: OPT-2-A.
func TestUpdateOption_VF_SetVerdadero_ClearsFalso(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoVerdaderoFalso)

	verdaderoID := uuid.New().String()
	falsoID := uuid.New().String()

	// Verdadero starts as not-correct; Falso is currently the correct one (sibling to clear).
	verdadero := &domain.Option{ID: verdaderoID, QuestionID: q.ID, Texto: "Verdadero", Correcta: false}

	// loadOwnedOption path: GetOptionByID → GetQuestionByID (loadOwnedQuestion) → GetEvaluationByID → GetCourseOwnership
	repo.On("GetOptionByID", mock.Anything, verdaderoID).Return(verdadero, nil).Once()
	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil)
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	// The targeted option update.
	repo.On("UpdateOption", mock.Anything, verdaderoID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// OPT-2: sibling "Falso" is currently correcta=true — must be cleared.
	repo.On("ListOptionsByQuestion", mock.Anything, q.ID).Return([]domain.Option{
		{ID: verdaderoID, QuestionID: q.ID, Texto: "Verdadero", Correcta: false},
		{ID: falsoID, QuestionID: q.ID, Texto: "Falso", Correcta: true},
	}, nil)
	repo.On("UpdateOption", mock.Anything, falsoID, map[string]any{"correcta": false}).Return(nil)

	// Final read-back.
	updatedVerdadero := &domain.Option{ID: verdaderoID, QuestionID: q.ID, Texto: "Verdadero", Correcta: true}
	repo.On("GetOptionByID", mock.Anything, verdaderoID).Return(updatedVerdadero, nil).Once()

	result, err := svc.UpdateOption(context.Background(), verdaderoID, ownerID, OptionUpdateRequest{
		Correcta: ptrBool(true),
	})
	require.NoError(t, err, "OPT-2-A: setting Verdadero=true must succeed")
	assert.True(t, result.Correcta, "OPT-2-A: returned option must have Correcta=true")
	repo.AssertExpectations(t)
}

// TestUpdateOption_VF_SetFalso_ClearsVerdadero verifies OPT-2-B:
// When "Falso" option is set correcta=true, the sibling "Verdadero" (currently true)
// is automatically set to correcta=false.
// Spec: OPT-2-B.
func TestUpdateOption_VF_SetFalso_ClearsVerdadero(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	ownerID := uuid.New().String()
	courseID := uuid.New().String()
	e := evalFixture(courseID)
	q := questionFixture(e.ID, domain.TipoVerdaderoFalso)

	verdaderoID := uuid.New().String()
	falsoID := uuid.New().String()

	// loadOwnedOption path: GetOptionByID (for Falso) → GetQuestionByID → GetEvaluationByID → GetCourseOwnership
	falso := &domain.Option{ID: falsoID, QuestionID: q.ID, Texto: "Falso", Correcta: false}
	repo.On("GetOptionByID", mock.Anything, falsoID).Return(falso, nil).Once()
	repo.On("GetQuestionByID", mock.Anything, q.ID).Return(q, nil)
	repo.On("GetEvaluationByID", mock.Anything, e.ID).Return(e, nil)
	checker.On("GetCourseOwnership", mock.Anything, courseID).Return(ownerID, "borrador", nil)

	// The targeted option update.
	repo.On("UpdateOption", mock.Anything, falsoID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	// OPT-2: sibling "Verdadero" is currently correcta=true — must be cleared.
	repo.On("ListOptionsByQuestion", mock.Anything, q.ID).Return([]domain.Option{
		{ID: verdaderoID, QuestionID: q.ID, Texto: "Verdadero", Correcta: true},
		{ID: falsoID, QuestionID: q.ID, Texto: "Falso", Correcta: false},
	}, nil)
	repo.On("UpdateOption", mock.Anything, verdaderoID, map[string]any{"correcta": false}).Return(nil)

	// Final read-back.
	updatedFalso := &domain.Option{ID: falsoID, QuestionID: q.ID, Texto: "Falso", Correcta: true}
	repo.On("GetOptionByID", mock.Anything, falsoID).Return(updatedFalso, nil).Once()

	result, err := svc.UpdateOption(context.Background(), falsoID, ownerID, OptionUpdateRequest{
		Correcta: ptrBool(true),
	})
	require.NoError(t, err, "OPT-2-B: setting Falso=true must succeed")
	assert.True(t, result.Correcta, "OPT-2-B: returned option must have Correcta=true")
	// Verify sibling verdadero was referenced (not in repo.AssertExpectations, but check mock calls)
	repo.AssertExpectations(t)
}

// ── T-1.10: ValidateSubmitReady tests ─────────────────────────────────────────

// TestValidateSubmitReady_NoEvaluation_ReturnsErrEvaluationNotFound verifies
// that when no evaluation exists for the course, ErrEvaluationNotFound is returned.
// Spec: REQ-XMOD XMOD-4; Design §2 evaluations addition.
func TestValidateSubmitReady_NoEvaluation_ReturnsErrEvaluationNotFound(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{} // unused by ValidateSubmitReady
	svc := newSvc(repo, checker)

	courseID := uuid.New().String()
	repo.On("GetEvaluationByCourse", mock.Anything, courseID).
		Return(nil, repository.ErrEvaluationNotFound)

	err := svc.ValidateSubmitReady(context.Background(), courseID)
	assert.ErrorIs(t, err, ErrEvaluationNotFound,
		"must surface ErrEvaluationNotFound when no evaluation exists")
	repo.AssertExpectations(t)
}

// TestValidateSubmitReady_IncompleteEvaluation_ReturnsErrNoCorrectOption verifies
// that when the evaluation exists but is incomplete, ErrNoCorrectOption is returned.
func TestValidateSubmitReady_IncompleteEvaluation_ReturnsErrNoCorrectOption(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	courseID := uuid.New().String()
	evalID := uuid.New().String()
	questionID := uuid.New().String()

	eval := &domain.Evaluation{ID: evalID, CourseID: courseID, NotaMinima: 70}
	question := domain.Question{ID: questionID, EvaluationID: evalID, Tipo: domain.TipoOpcionMultiple, Puntaje: 1}
	// Option with correcta=false → incomplete
	opt := domain.Option{ID: uuid.New().String(), QuestionID: questionID, Correcta: false}

	repo.On("GetEvaluationByCourse", mock.Anything, courseID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, evalID).Return([]domain.Question{question}, nil)
	repo.On("ListOptionsByQuestion", mock.Anything, questionID).Return([]domain.Option{opt}, nil)

	err := svc.ValidateSubmitReady(context.Background(), courseID)
	assert.ErrorIs(t, err, ErrNoCorrectOption,
		"must surface ErrNoCorrectOption when evaluation has no correct option")
	repo.AssertExpectations(t)
}

// TestValidateSubmitReady_CompleteEvaluation_ReturnsNil verifies that
// a complete evaluation returns nil (submit gate passes).
func TestValidateSubmitReady_CompleteEvaluation_ReturnsNil(t *testing.T) {
	repo := &mockEvalRepo{}
	checker := &testutil.MockCoursesChecker{}
	svc := newSvc(repo, checker)

	courseID := uuid.New().String()
	evalID := uuid.New().String()
	questionID := uuid.New().String()

	eval := &domain.Evaluation{ID: evalID, CourseID: courseID, NotaMinima: 70}
	question := domain.Question{ID: questionID, EvaluationID: evalID, Tipo: domain.TipoOpcionMultiple, Puntaje: 1}
	// Option with correcta=true → complete
	opt := domain.Option{ID: uuid.New().String(), QuestionID: questionID, Correcta: true}

	repo.On("GetEvaluationByCourse", mock.Anything, courseID).Return(eval, nil)
	repo.On("ListQuestionsByEvaluation", mock.Anything, evalID).Return([]domain.Question{question}, nil)
	repo.On("ListOptionsByQuestion", mock.Anything, questionID).Return([]domain.Option{opt}, nil)

	err := svc.ValidateSubmitReady(context.Background(), courseID)
	assert.NoError(t, err, "must return nil for a complete evaluation")
	repo.AssertExpectations(t)
}

// ── helpers ────────────────────────────────────────────────────────────────────

func ptrInt(v int) *int    { return &v }
func ptrBool(v bool) *bool { return &v }
