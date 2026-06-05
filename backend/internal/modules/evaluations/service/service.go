// Package service contains the business logic for the evaluations module.
// It is HTTP-agnostic: it returns domain sentinels and read-models.
// Handlers are responsible for mapping sentinels → HTTP status codes.
//
// Cross-module ownership is enforced via the CoursesChecker interface — evaluations
// never imports courses internals directly (ADR-1, Decision 1).
package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
)

// ── Cross-module seams ─────────────────────────────────────────────────────────

// CoursesChecker is the narrow cross-module seam into courses.
// coursesSvc satisfies this structurally — evaluations never imports courses internals.
// The string estado (not courses.Estado) keeps the seam minimal, zero domain leak (Decision 1).
type CoursesChecker interface {
	GetCourseOwnership(ctx context.Context, courseID string) (creadorID, estado string, err error)
}

// EnrollmentCompleter is a nil-able forward seam wired by C2.4.
// In C3.2 it is nil (no-op).
type EnrollmentCompleter interface {
	MarkEnrollmentCompleted(ctx context.Context, userID, courseID string) error
}

// CertificateIssuer is a nil-able forward seam wired by C5.1.
// In C3.2 it is nil (no-op).
type CertificateIssuer interface {
	IssueOnPass(ctx context.Context, userID, courseID string) error
}

// Option configures optional serviceImpl dependencies via functional options (ADR-A).
type Option func(*serviceImpl)

// WithEnrollmentCompleter injects an EnrollmentCompleter seam into the service.
func WithEnrollmentCompleter(ec EnrollmentCompleter) Option {
	return func(s *serviceImpl) { s.enroll = ec }
}

// WithCertificateIssuer injects a CertificateIssuer seam into the service.
func WithCertificateIssuer(ci CertificateIssuer) Option {
	return func(s *serviceImpl) { s.certs = ci }
}

// ── Read models ────────────────────────────────────────────────────────────────

// EvaluationModel is the flat read model for a single evaluation.
type EvaluationModel struct {
	ID          string
	CourseID    string
	NotaMinima  int
	IntentosMax int
}

// QuestionModel is the flat read model for a single question.
type QuestionModel struct {
	ID           string
	EvaluationID string
	Enunciado    string
	Tipo         string
	Puntaje      int
	Orden        int
}

// OptionModel is the flat read model for a single option.
type OptionModel struct {
	ID         string
	QuestionID string
	Texto      string
	Correcta   bool
	Orden      int
}

// QuestionWithOptionsModel is a question with its options already fetched.
type QuestionWithOptionsModel struct {
	QuestionModel
	Options []OptionModel
}

// EvaluationDetailModel is the full nested tree: evaluation + questions + options.
type EvaluationDetailModel struct {
	EvaluationModel
	Questions []QuestionWithOptionsModel
}

// ── Attempt read-models ────────────────────────────────────────────────────────

// AttemptModel is the flat read model returned by StartAttempt.
type AttemptModel struct {
	ID           string
	UserID       string
	EvaluationID string
	Numero       int
	Puntaje      int
	Aprobado     bool
	IniciadoEn   time.Time
	FinalizadoEn *time.Time
}

// AttemptStateOption is the student-facing option shape.
// It has NO Correcta field — the omission is structural and compile-time guaranteed (ADR-E, R1).
type AttemptStateOption struct {
	ID    string
	Texto string
}

// AttemptStateQuestion is a question with student-safe options (no Correcta).
type AttemptStateQuestion struct {
	ID        string
	Enunciado string
	Tipo      string
	Puntaje   int
	Options   []AttemptStateOption
}

// AttemptStateAnswer records the student's current selection for a question.
type AttemptStateAnswer struct {
	QuestionID string
	OptionID   string
}

// AttemptStateModel is the full state of an attempt including questions and
// the student's current answers. Submitted is true when finalizado_en is set.
// Post-submit: Puntaje and Aprobado carry the final score; Correcta is still absent.
type AttemptStateModel struct {
	AttemptModel
	Questions []AttemptStateQuestion
	Answers   []AttemptStateAnswer
	Submitted bool
}

// AttemptResultModel is the slim response returned by SubmitAttempt.
type AttemptResultModel struct {
	Puntaje  int
	Aprobado bool
}

// ── Request types ──────────────────────────────────────────────────────────────

// EvaluationCreateRequest carries data for creating an evaluation.
type EvaluationCreateRequest struct {
	NotaMinima  *int
	IntentosMax *int
}

// EvaluationUpdateRequest carries partial fields for updating an evaluation.
type EvaluationUpdateRequest struct {
	NotaMinima  *int
	IntentosMax *int
}

// QuestionCreateRequest carries data for creating a question.
type QuestionCreateRequest struct {
	Enunciado string
	Tipo      string
	Puntaje   *int
}

// QuestionUpdateRequest carries partial fields for updating a question.
// Tipo is intentionally absent — it is immutable after creation.
type QuestionUpdateRequest struct {
	Enunciado *string
	Puntaje   *int
	Orden     *int
}

// OptionCreateRequest carries data for creating an option.
type OptionCreateRequest struct {
	Texto    string
	Correcta *bool
}

// OptionUpdateRequest carries partial fields for updating an option.
type OptionUpdateRequest struct {
	Texto    *string
	Correcta *bool
}

// ── Service interface ──────────────────────────────────────────────────────────

// Service is the public interface of the evaluations domain.
// Other modules (handlers) depend on this interface — never on serviceImpl.
type Service interface {
	// CreateEvaluation creates a new evaluation for a course.
	// 1-1 invariant: returns ErrEvaluationExists if one already exists.
	// Ownership and estado checks are applied FIRST (ownership before estado, like courses).
	CreateEvaluation(ctx context.Context, courseID, creadorID string, req EvaluationCreateRequest) (*EvaluationModel, error)

	// GetEvaluation returns the nested evaluation tree for a course.
	// Read-gated: returns ErrNotOwner (handler maps to 404) for non-owners.
	// Returns ErrEvaluationNotFound if no evaluation exists for the course.
	GetEvaluation(ctx context.Context, courseID, creadorID string) (*EvaluationDetailModel, error)

	// UpdateEvaluation partially updates an evaluation the caller owns.
	UpdateEvaluation(ctx context.Context, evalID, creadorID string, req EvaluationUpdateRequest) (*EvaluationModel, error)

	// CreateQuestion creates a question on an evaluation the caller owns.
	// For verdadero_falso, auto-creates 2 options ("Verdadero"/"Falso", correcta=false).
	CreateQuestion(ctx context.Context, evalID, creadorID string, req QuestionCreateRequest) (*QuestionWithOptionsModel, error)

	// UpdateQuestion partially updates a question the caller owns.
	UpdateQuestion(ctx context.Context, questionID, creadorID string, req QuestionUpdateRequest) (*QuestionModel, error)

	// DeleteQuestion deletes a question the caller owns (FK cascade removes options).
	DeleteQuestion(ctx context.Context, questionID, creadorID string) error

	// CreateOption adds an option to an opcion_multiple question the caller owns.
	// Returns ErrInvalidQuestionType for verdadero_falso questions.
	CreateOption(ctx context.Context, questionID, creadorID string, req OptionCreateRequest) (*OptionModel, error)

	// UpdateOption partially updates an option the caller owns.
	// Permitted for both tipos (e.g. toggling correcta on V/F questions).
	UpdateOption(ctx context.Context, optionID, creadorID string, req OptionUpdateRequest) (*OptionModel, error)

	// DeleteOption deletes an option the caller owns.
	// Only permitted for opcion_multiple questions.
	DeleteOption(ctx context.Context, optionID, creadorID string) error

	// ValidateEvaluationComplete checks that every question has at least one correct option.
	// NOT wired to any mutating endpoint in C3.1 — used as a pre-submit gate in C4.1.
	// Implemented and tested here per Decision 5.
	ValidateEvaluationComplete(ctx context.Context, evalID string) error

	// StartAttempt starts or resumes a student attempt on the evaluation.
	// If an open (unsubmitted) attempt already exists for (user, evaluation), it is
	// returned immediately (resume-on-start) — no new attempt is created.
	// Returns ErrEvaluationNotFound if the evaluation does not exist.
	// Returns ErrMaxAttemptsReached if intentos_max > 0 and count >= intentos_max
	// (only checked when NO open attempt exists, i.e. on new-attempt path).
	StartAttempt(ctx context.Context, evaluationID, userID string) (*AttemptModel, error)

	// GetAttempt returns the full attempt state including questions (no correcta) and
	// the student's current answers. Returns ErrAttemptNotFound for non-owners (anti-enum).
	GetAttempt(ctx context.Context, attemptID, userID string) (*AttemptStateModel, error)

	// SaveAnswer records or updates the student's answer for a question.
	// Returns ErrAttemptNotFound for non-owners, ErrAttemptAlreadySubmitted if submitted,
	// and ErrInvalidAnswer if the option/question pair is invalid.
	SaveAnswer(ctx context.Context, attemptID, userID, questionID, optionID string) error

	// SubmitAttempt finalises the attempt, computing puntaje and aprobado.
	// Returns ErrAttemptNotFound for non-owners, ErrAttemptAlreadySubmitted if already done.
	// Calls EnrollmentCompleter + CertificateIssuer seams if aprobado (non-fatal on error).
	SubmitAttempt(ctx context.Context, attemptID, userID string) (*AttemptResultModel, error)
}

// ── concrete implementation ────────────────────────────────────────────────────

type serviceImpl struct {
	repo    repository.Repository
	courses CoursesChecker
	enroll  EnrollmentCompleter // nil in C3.2; wired by C2.4
	certs   CertificateIssuer   // nil in C3.2; wired by C5.1
}

// New creates a Service backed by the given Repository and CoursesChecker.
// Optional seam dependencies are injected via functional options (ADR-A).
// The existing 2-arg call site (main.go) stays valid — opts is variadic.
func New(repo repository.Repository, courses CoursesChecker, opts ...Option) Service {
	s := &serviceImpl{repo: repo, courses: courses}
	for _, o := range opts {
		o(s)
	}
	return s
}

// ── CreateEvaluation ───────────────────────────────────────────────────────────

// CreateEvaluation enforces ownership+estado gate, then persists the evaluation.
func (s *serviceImpl) CreateEvaluation(ctx context.Context, courseID, creadorID string, req EvaluationCreateRequest) (*EvaluationModel, error) {
	if err := s.assertEvaluationEditable(ctx, courseID, creadorID); err != nil {
		return nil, err
	}

	e := &domain.Evaluation{
		ID:       uuid.New().String(),
		CourseID: courseID,
	}
	if req.NotaMinima != nil {
		e.NotaMinima = *req.NotaMinima
	} else {
		e.NotaMinima = 70 // default
	}
	if req.IntentosMax != nil {
		e.IntentosMax = *req.IntentosMax
	}

	if err := s.repo.CreateEvaluation(ctx, e); err != nil {
		return nil, wrapEvalExists(err)
	}
	return toEvaluationModel(e), nil
}

// ── GetEvaluation ──────────────────────────────────────────────────────────────

// GetEvaluation verifies ownership, then composes the full nested tree.
// Estado is NOT checked on read — the owner can view any estado (like courses.GetByID).
func (s *serviceImpl) GetEvaluation(ctx context.Context, courseID, creadorID string) (*EvaluationDetailModel, error) {
	// 1. Verify ownership via seam (no estado check on reads).
	owner, _, err := s.courses.GetCourseOwnership(ctx, courseID)
	if err != nil {
		return nil, wrapCourseNotFound(err)
	}
	if owner != creadorID {
		return nil, ErrNotOwner // handler maps to 404 (read convention)
	}

	// 2. Fetch evaluation.
	e, err := s.repo.GetEvaluationByCourse(ctx, courseID)
	if err != nil {
		return nil, wrapEvalNotFound(err)
	}

	// 3. Compose nested tree (evaluation → questions → options) in Go.
	return s.composeDetail(ctx, e)
}

// ── UpdateEvaluation ───────────────────────────────────────────────────────────

// UpdateEvaluation resolves the evaluation, asserts editability, then applies the patch.
func (s *serviceImpl) UpdateEvaluation(ctx context.Context, evalID, creadorID string, req EvaluationUpdateRequest) (*EvaluationModel, error) {
	e, err := s.loadOwnedEvaluation(ctx, evalID, creadorID)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}
	if req.NotaMinima != nil {
		fields["nota_minima"] = *req.NotaMinima
	}
	if req.IntentosMax != nil {
		fields["intentos_max"] = *req.IntentosMax
	}
	if len(fields) == 0 {
		return toEvaluationModel(e), nil
	}

	if err := s.repo.UpdateEvaluation(ctx, evalID, fields); err != nil {
		return nil, wrapEvalNotFound(err)
	}

	updated, err := s.repo.GetEvaluationByID(ctx, evalID)
	if err != nil {
		return nil, wrapEvalNotFound(err)
	}
	return toEvaluationModel(updated), nil
}

// ── CreateQuestion ─────────────────────────────────────────────────────────────

// CreateQuestion validates tipo, asserts editability, creates the question, and
// auto-creates 2 options for verdadero_falso questions (Decision 4).
func (s *serviceImpl) CreateQuestion(ctx context.Context, evalID, creadorID string, req QuestionCreateRequest) (*QuestionWithOptionsModel, error) {
	// 1. Load and assert editability via evaluation.
	_, err := s.loadOwnedEvaluation(ctx, evalID, creadorID)
	if err != nil {
		return nil, err
	}

	// 2. Validate tipo.
	tipo := domain.TipoPregunta(req.Tipo)
	if !tipo.Valid() {
		return nil, ErrInvalidQuestionType
	}

	// 3. Persist question.
	puntaje := 1
	if req.Puntaje != nil {
		puntaje = *req.Puntaje
	}
	q := &domain.Question{
		ID:           uuid.New().String(),
		EvaluationID: evalID,
		Enunciado:    req.Enunciado,
		Tipo:         tipo,
		Puntaje:      puntaje,
	}
	if err := s.repo.CreateQuestion(ctx, q); err != nil {
		return nil, err
	}

	// 4. Auto-create 2 V/F options for verdadero_falso (Decision 4).
	if tipo == domain.TipoVerdaderoFalso {
		opts := []domain.Option{
			{ID: uuid.New().String(), QuestionID: q.ID, Texto: "Verdadero", Correcta: false, Orden: 0},
			{ID: uuid.New().String(), QuestionID: q.ID, Texto: "Falso", Correcta: false, Orden: 1},
		}
		if err := s.repo.CreateOptions(ctx, opts); err != nil {
			return nil, err
		}
	}

	// 5. Compose return value with options.
	options, err := s.repo.ListOptionsByQuestion(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	return toQuestionWithOptions(q, options), nil
}

// ── UpdateQuestion ─────────────────────────────────────────────────────────────

func (s *serviceImpl) UpdateQuestion(ctx context.Context, questionID, creadorID string, req QuestionUpdateRequest) (*QuestionModel, error) {
	_, err := s.loadOwnedQuestion(ctx, questionID, creadorID)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}
	if req.Enunciado != nil {
		fields["enunciado"] = *req.Enunciado
	}
	if req.Puntaje != nil {
		fields["puntaje"] = *req.Puntaje
	}
	if req.Orden != nil {
		fields["orden"] = *req.Orden
	}
	if len(fields) == 0 {
		q, _ := s.repo.GetQuestionByID(ctx, questionID)
		return toQuestionModel(q), nil
	}

	if err := s.repo.UpdateQuestion(ctx, questionID, fields); err != nil {
		return nil, wrapQuestionNotFound(err)
	}

	updated, err := s.repo.GetQuestionByID(ctx, questionID)
	if err != nil {
		return nil, wrapQuestionNotFound(err)
	}
	return toQuestionModel(updated), nil
}

// ── DeleteQuestion ─────────────────────────────────────────────────────────────

func (s *serviceImpl) DeleteQuestion(ctx context.Context, questionID, creadorID string) error {
	_, err := s.loadOwnedQuestion(ctx, questionID, creadorID)
	if err != nil {
		return err
	}
	return s.repo.DeleteQuestion(ctx, questionID)
}

// ── CreateOption ───────────────────────────────────────────────────────────────

// CreateOption adds an option to an opcion_multiple question.
// Rejects CreateOption on verdadero_falso questions (only PATCH correcta is allowed).
func (s *serviceImpl) CreateOption(ctx context.Context, questionID, creadorID string, req OptionCreateRequest) (*OptionModel, error) {
	q, err := s.loadOwnedQuestion(ctx, questionID, creadorID)
	if err != nil {
		return nil, err
	}

	// Guard: V/F questions have fixed 2 options — reject additional CreateOption.
	if q.Tipo == domain.TipoVerdaderoFalso {
		return nil, ErrInvalidQuestionType
	}

	correcta := false
	if req.Correcta != nil {
		correcta = *req.Correcta
	}
	o := domain.Option{
		ID:         uuid.New().String(),
		QuestionID: questionID,
		Texto:      req.Texto,
		Correcta:   correcta,
	}
	if err := s.repo.CreateOptions(ctx, []domain.Option{o}); err != nil {
		return nil, err
	}

	created, err := s.repo.GetOptionByID(ctx, o.ID)
	if err != nil {
		return nil, wrapOptionNotFound(err)
	}
	return toOptionModel(created), nil
}

// ── UpdateOption ───────────────────────────────────────────────────────────────

// UpdateOption updates an option. Permitted for both tipos (e.g. toggling correcta on V/F).
// OPT-2 invariant: when updating a verdadero_falso option to correcta=true, the sibling
// option is automatically set to correcta=false (exactly one correct at all times).
func (s *serviceImpl) UpdateOption(ctx context.Context, optionID, creadorID string, req OptionUpdateRequest) (*OptionModel, error) {
	o, err := s.loadOwnedOption(ctx, optionID, creadorID)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}
	if req.Texto != nil {
		fields["texto"] = *req.Texto
	}
	if req.Correcta != nil {
		fields["correcta"] = *req.Correcta
	}
	if len(fields) == 0 {
		current, _ := s.repo.GetOptionByID(ctx, optionID)
		return toOptionModel(current), nil
	}

	if err := s.repo.UpdateOption(ctx, optionID, fields); err != nil {
		return nil, wrapOptionNotFound(err)
	}

	// OPT-2: for verdadero_falso questions, enforce mutual exclusion.
	// When correcta=true is being set, clear the sibling option.
	if req.Correcta != nil && *req.Correcta {
		q, err := s.repo.GetQuestionByID(ctx, o.QuestionID)
		if err == nil && q.Tipo == domain.TipoVerdaderoFalso {
			siblings, err := s.repo.ListOptionsByQuestion(ctx, o.QuestionID)
			if err == nil {
				for _, sibling := range siblings {
					if sibling.ID != optionID && sibling.Correcta {
						_ = s.repo.UpdateOption(ctx, sibling.ID, map[string]any{"correcta": false})
					}
				}
			}
		}
	}

	updated, err := s.repo.GetOptionByID(ctx, optionID)
	if err != nil {
		return nil, wrapOptionNotFound(err)
	}
	return toOptionModel(updated), nil
}

// ── DeleteOption ───────────────────────────────────────────────────────────────

func (s *serviceImpl) DeleteOption(ctx context.Context, optionID, creadorID string) error {
	_, err := s.loadOwnedOption(ctx, optionID, creadorID)
	if err != nil {
		return err
	}
	return s.repo.DeleteOption(ctx, optionID)
}

// ── ValidateEvaluationComplete (Decision 5 — ungated in C3.1) ────────────────

// ValidateEvaluationComplete checks that every question has at least one correct option.
// NOT wired to any mutating endpoint in C3.1. Called by C4.1 submit flow.
func (s *serviceImpl) ValidateEvaluationComplete(ctx context.Context, evalID string) error {
	questions, err := s.repo.ListQuestionsByEvaluation(ctx, evalID)
	if err != nil {
		return err
	}
	for _, q := range questions {
		if err := s.validateQuestionComplete(ctx, q.ID); err != nil {
			return err
		}
	}
	return nil
}

// validateQuestionComplete checks that the given question has at least one correct option.
// Returns ErrNoCorrectOption if no option has correcta=true.
func (s *serviceImpl) validateQuestionComplete(ctx context.Context, questionID string) error {
	opts, err := s.repo.ListOptionsByQuestion(ctx, questionID)
	if err != nil {
		return err
	}
	for _, o := range opts {
		if o.Correcta {
			return nil
		}
	}
	return ErrNoCorrectOption
}

// ── Ownership traversal helpers ────────────────────────────────────────────────

// assertEvaluationEditable resolves the course via the seam and enforces owner + estado.
// ORDERING: ownership FIRST (ErrNotOwner), then estado (ErrCourseNotEditable).
// This matches courses.assertCourseEditable and locks the load-bearing ordering invariant.
func (s *serviceImpl) assertEvaluationEditable(ctx context.Context, courseID, creadorID string) error {
	owner, estado, err := s.courses.GetCourseOwnership(ctx, courseID)
	if err != nil {
		return wrapCourseNotFound(err)
	}
	if owner != creadorID {
		return ErrNotOwner // 403 write / 404 read
	}
	if estado != "borrador" && estado != "rechazado" {
		return ErrCourseNotEditable // 409
	}
	return nil
}

// loadOwnedEvaluation resolves eval → course and asserts editability.
func (s *serviceImpl) loadOwnedEvaluation(ctx context.Context, evalID, creadorID string) (*domain.Evaluation, error) {
	e, err := s.repo.GetEvaluationByID(ctx, evalID)
	if err != nil {
		return nil, wrapEvalNotFound(err)
	}
	if err := s.assertEvaluationEditable(ctx, e.CourseID, creadorID); err != nil {
		return nil, err
	}
	return e, nil
}

// loadOwnedQuestion resolves question → eval → course and asserts editability.
func (s *serviceImpl) loadOwnedQuestion(ctx context.Context, questionID, creadorID string) (*domain.Question, error) {
	q, err := s.repo.GetQuestionByID(ctx, questionID)
	if err != nil {
		return nil, wrapQuestionNotFound(err)
	}
	if _, err := s.loadOwnedEvaluation(ctx, q.EvaluationID, creadorID); err != nil {
		return nil, err
	}
	return q, nil
}

// loadOwnedOption resolves option → question → eval → course and asserts editability.
func (s *serviceImpl) loadOwnedOption(ctx context.Context, optionID, creadorID string) (*domain.Option, error) {
	o, err := s.repo.GetOptionByID(ctx, optionID)
	if err != nil {
		return nil, wrapOptionNotFound(err)
	}
	if _, err := s.loadOwnedQuestion(ctx, o.QuestionID, creadorID); err != nil {
		return nil, err
	}
	return o, nil
}

// ── wrap helpers ───────────────────────────────────────────────────────────────

func wrapEvalNotFound(err error) error {
	if errors.Is(err, repository.ErrEvaluationNotFound) {
		return ErrEvaluationNotFound
	}
	return err
}

func wrapQuestionNotFound(err error) error {
	if errors.Is(err, repository.ErrQuestionNotFound) {
		return ErrQuestionNotFound
	}
	return err
}

func wrapOptionNotFound(err error) error {
	if errors.Is(err, repository.ErrOptionNotFound) {
		return ErrOptionNotFound
	}
	return err
}

func wrapEvalExists(err error) error {
	if errors.Is(err, repository.ErrEvaluationExists) {
		return ErrEvaluationExists
	}
	return err
}

func wrapCourseNotFound(err error) error {
	// courses.ErrCourseNotFound is re-exported through the courses facade (courses.go),
	// so evaluations can use errors.Is without importing courses internals (ADR-1 safe).
	if errors.Is(err, courses.ErrCourseNotFound) {
		return ErrCourseNotFound
	}
	return err
}

// ── to-model converters ────────────────────────────────────────────────────────

func toEvaluationModel(e *domain.Evaluation) *EvaluationModel {
	return &EvaluationModel{
		ID:          e.ID,
		CourseID:    e.CourseID,
		NotaMinima:  e.NotaMinima,
		IntentosMax: e.IntentosMax,
	}
}

func toQuestionModel(q *domain.Question) *QuestionModel {
	return &QuestionModel{
		ID:           q.ID,
		EvaluationID: q.EvaluationID,
		Enunciado:    q.Enunciado,
		Tipo:         string(q.Tipo),
		Puntaje:      q.Puntaje,
		Orden:        q.Orden,
	}
}

func toOptionModel(o *domain.Option) *OptionModel {
	return &OptionModel{
		ID:         o.ID,
		QuestionID: o.QuestionID,
		Texto:      o.Texto,
		Correcta:   o.Correcta,
		Orden:      o.Orden,
	}
}

func toQuestionWithOptions(q *domain.Question, opts []domain.Option) *QuestionWithOptionsModel {
	options := make([]OptionModel, 0, len(opts))
	for i := range opts {
		options = append(options, *toOptionModel(&opts[i]))
	}
	return &QuestionWithOptionsModel{
		QuestionModel: *toQuestionModel(q),
		Options:       options,
	}
}

// composeDetail fetches the full nested tree for an evaluation.
func (s *serviceImpl) composeDetail(ctx context.Context, e *domain.Evaluation) (*EvaluationDetailModel, error) {
	questions, err := s.repo.ListQuestionsByEvaluation(ctx, e.ID)
	if err != nil {
		return nil, err
	}
	qModels := make([]QuestionWithOptionsModel, 0, len(questions))
	for i := range questions {
		opts, err := s.repo.ListOptionsByQuestion(ctx, questions[i].ID)
		if err != nil {
			return nil, err
		}
		qModels = append(qModels, *toQuestionWithOptions(&questions[i], opts))
	}
	return &EvaluationDetailModel{
		EvaluationModel: *toEvaluationModel(e),
		Questions:       qModels,
	}, nil
}
