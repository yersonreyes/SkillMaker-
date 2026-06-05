// Package service — student attempt lifecycle for the evaluations module.
// StartAttempt, GetAttempt, SaveAnswer, SubmitAttempt.
// Scoring is computed in Go (not SQL) over service-composed lists (ADR-C).
// Forward seams (EnrollmentCompleter, CertificateIssuer) are invoked on pass
// and their errors are logged via slog — NOT returned (REQ-SEAMS).
package service

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/repository"
)

// ── StartAttempt ───────────────────────────────────────────────────────────────

// StartAttempt creates a new attempt for the student on the given evaluation.
// Spec: REQ-START.
func (s *serviceImpl) StartAttempt(ctx context.Context, evaluationID, userID string) (*AttemptModel, error) {
	// Load evaluation (returns ErrEvaluationNotFound if missing).
	eval, err := s.repo.GetEvaluationByID(ctx, evaluationID)
	if err != nil {
		return nil, wrapEvalNotFound(err)
	}

	// Block if an open (unsubmitted) attempt already exists.
	_, err = s.repo.GetOpenAttempt(ctx, userID, evaluationID)
	if err == nil {
		// An open attempt was found.
		return nil, ErrAttemptOpen
	}
	if err != repository.ErrAttemptNotFound {
		// Unexpected DB error.
		return nil, err
	}

	// Count existing attempts and enforce intentos_max.
	count, err := s.repo.CountAttemptsByUserEval(ctx, userID, evaluationID)
	if err != nil {
		return nil, err
	}
	if eval.IntentosMax > 0 && count >= int64(eval.IntentosMax) {
		return nil, ErrMaxAttemptsReached
	}

	// Create the attempt.
	a := &domain.Attempt{
		ID:           uuid.New().String(),
		UserID:       userID,
		EvaluationID: evaluationID,
		Numero:       int(count) + 1,
		IniciadoEn:   time.Now().UTC(),
	}
	if err := s.repo.CreateAttempt(ctx, a); err != nil {
		return nil, err
	}
	return toAttemptModel(a), nil
}

// ── GetAttempt ─────────────────────────────────────────────────────────────────

// GetAttempt returns the full attempt state with questions (no correcta) and answers.
// Non-owner requests return ErrAttemptNotFound (anti-enumeration, never 403).
// Spec: REQ-GET.
func (s *serviceImpl) GetAttempt(ctx context.Context, attemptID, userID string) (*AttemptStateModel, error) {
	a, err := s.repo.GetAttemptByID(ctx, attemptID)
	if err != nil {
		if err == repository.ErrAttemptNotFound {
			return nil, ErrAttemptNotFound
		}
		return nil, err
	}
	// Ownership check: wrong user → ErrAttemptNotFound (no existence leak).
	if a.UserID != userID {
		return nil, ErrAttemptNotFound
	}

	// Load evaluation to get its questions.
	eval, err := s.repo.GetEvaluationByID(ctx, a.EvaluationID)
	if err != nil {
		return nil, wrapEvalNotFound(err)
	}

	questions, err := s.repo.ListQuestionsByEvaluation(ctx, eval.ID)
	if err != nil {
		return nil, err
	}

	stateQuestions := make([]AttemptStateQuestion, 0, len(questions))
	for i := range questions {
		opts, err := s.repo.ListOptionsByQuestion(ctx, questions[i].ID)
		if err != nil {
			return nil, err
		}
		stateOpts := make([]AttemptStateOption, 0, len(opts))
		for j := range opts {
			// NOTE: correcta is deliberately NOT copied — structural no-leak (ADR-E).
			stateOpts = append(stateOpts, AttemptStateOption{
				ID:    opts[j].ID,
				Texto: opts[j].Texto,
			})
		}
		stateQuestions = append(stateQuestions, AttemptStateQuestion{
			ID:        questions[i].ID,
			Enunciado: questions[i].Enunciado,
			Tipo:      string(questions[i].Tipo),
			Puntaje:   questions[i].Puntaje,
			Options:   stateOpts,
		})
	}

	// Load the student's current answers.
	rawAnswers, err := s.repo.ListAnswersByAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	stateAnswers := make([]AttemptStateAnswer, 0, len(rawAnswers))
	for i := range rawAnswers {
		stateAnswers = append(stateAnswers, AttemptStateAnswer{
			QuestionID: rawAnswers[i].QuestionID,
			OptionID:   rawAnswers[i].OptionID,
		})
	}

	return &AttemptStateModel{
		AttemptModel: *toAttemptModel(a),
		Questions:    stateQuestions,
		Answers:      stateAnswers,
		Submitted:    a.FinalizadoEn != nil,
	}, nil
}

// ── SaveAnswer ─────────────────────────────────────────────────────────────────

// SaveAnswer records or updates the student's answer for a question.
// Spec: REQ-ANSWER.
func (s *serviceImpl) SaveAnswer(ctx context.Context, attemptID, userID, questionID, optionID string) error {
	// Load and verify ownership.
	a, err := s.repo.GetAttemptByID(ctx, attemptID)
	if err != nil {
		if err == repository.ErrAttemptNotFound {
			return ErrAttemptNotFound
		}
		return err
	}
	if a.UserID != userID {
		return ErrAttemptNotFound
	}

	// Reject answers on submitted attempts.
	if a.FinalizadoEn != nil {
		return ErrAttemptAlreadySubmitted
	}

	// Validate option belongs to question and question belongs to this evaluation.
	opt, err := s.repo.GetOptionByID(ctx, optionID)
	if err != nil {
		return ErrInvalidAnswer
	}
	if opt.QuestionID != questionID {
		return ErrInvalidAnswer
	}
	q, err := s.repo.GetQuestionByID(ctx, questionID)
	if err != nil {
		return ErrInvalidAnswer
	}
	if q.EvaluationID != a.EvaluationID {
		return ErrInvalidAnswer
	}

	// Upsert: snapshot correcta at save time.
	return s.repo.UpsertAnswer(ctx, attemptID, questionID, optionID, opt.Correcta)
}

// ── SubmitAttempt ──────────────────────────────────────────────────────────────

// SubmitAttempt finalises the attempt: scores it, persists the result, and
// invokes the forward seams if aprobado. Seam errors are logged, not returned.
// Spec: REQ-SUBMIT, REQ-SEAMS.
func (s *serviceImpl) SubmitAttempt(ctx context.Context, attemptID, userID string) (*AttemptResultModel, error) {
	// Load and verify ownership.
	a, err := s.repo.GetAttemptByID(ctx, attemptID)
	if err != nil {
		if err == repository.ErrAttemptNotFound {
			return nil, ErrAttemptNotFound
		}
		return nil, err
	}
	if a.UserID != userID {
		return nil, ErrAttemptNotFound
	}

	// Reject re-submission.
	if a.FinalizadoEn != nil {
		return nil, ErrAttemptAlreadySubmitted
	}

	// Load evaluation for nota_minima and course_id.
	eval, err := s.repo.GetEvaluationByID(ctx, a.EvaluationID)
	if err != nil {
		return nil, wrapEvalNotFound(err)
	}

	// Load all questions for total puntaje.
	questions, err := s.repo.ListQuestionsByEvaluation(ctx, eval.ID)
	if err != nil {
		return nil, err
	}

	// Load student answers for earned puntaje.
	answers, err := s.repo.ListAnswersByAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	// Build answer index: questionID → correcta.
	answerIndex := make(map[string]bool, len(answers))
	for i := range answers {
		answerIndex[answers[i].QuestionID] = answers[i].Correcta
	}

	// Compute scoring (ADR-C — in Go, not SQL).
	var total, earned int
	for i := range questions {
		total += questions[i].Puntaje
		if answerIndex[questions[i].ID] {
			earned += questions[i].Puntaje
		}
	}

	// Guard division by zero (spec: total==0 → pct=0, aprobado=false).
	var pct int
	if total > 0 {
		pct = int(math.Round(float64(earned) * 100 / float64(total)))
	}
	aprobado := pct >= eval.NotaMinima

	// Persist the result.
	now := time.Now().UTC()
	if err := s.repo.UpdateAttempt(ctx, attemptID, map[string]any{
		"finalizado_en": now,
		"puntaje":       pct,
		"aprobado":      aprobado,
	}); err != nil {
		return nil, err
	}

	// Invoke forward seams if aprobado. Errors are non-fatal (REQ-SEAMS).
	if aprobado {
		courseID := eval.CourseID
		if s.enroll != nil {
			if err := s.enroll.MarkEnrollmentCompleted(ctx, a.UserID, courseID); err != nil {
				slog.Error("evaluations: enrollment-completed seam failed",
					"err", err, "attempt", attemptID)
			}
		}
		if s.certs != nil {
			if err := s.certs.IssueOnPass(ctx, a.UserID, courseID); err != nil {
				slog.Error("evaluations: certificate seam failed",
					"err", err, "attempt", attemptID)
			}
		}
	}

	return &AttemptResultModel{Puntaje: pct, Aprobado: aprobado}, nil
}

// ── toAttemptModel ─────────────────────────────────────────────────────────────

func toAttemptModel(a *domain.Attempt) *AttemptModel {
	return &AttemptModel{
		ID:           a.ID,
		UserID:       a.UserID,
		EvaluationID: a.EvaluationID,
		Numero:       a.Numero,
		Puntaje:      a.Puntaje,
		Aprobado:     a.Aprobado,
		IniciadoEn:   a.IniciadoEn,
		FinalizadoEn: a.FinalizadoEn,
	}
}
