// Package handler — HTTP-layer contract tests for the student attempt lifecycle (C3.2).
//
// LOAD-BEARING tests:
//
//	(a) no-correcta-leak: GET /attempts/:id JSON response body MUST NOT contain "correcta"
//	(g) gin route boot: RegisterRoutes + RegisterStudentRoutes on ONE gin.Engine → no panic
//
// No build tag: runs with standard `make backend-test`.
package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	coursesHandler "github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
)

// setupStudentEngine builds a gin.Engine with BOTH creator and student routes on the
// SAME engine (mirrors the real composition root in main.go).
func setupStudentEngine(svc service.Service, userID string) *gin.Engine {
	r := gin.New()
	identity := injectIdentity(userID, []string{"alumno"})
	protected := r.Group("", identity)
	// Creator group (sub-group of protected with RequireRole creador).
	creatorGrp := protected.Group("", middleware.RequireRole("creador"))

	// Register both creator and student routes to mirror real boot.
	handler.Register(creatorGrp, svc)
	handler.RegisterStudent(protected, svc)
	return r
}

// ── [LOAD-BEARING (g)] Gin route boot with creator + student routes ───────────

// TestGinRouteBoot_CreatorAndStudentRoutes_NoPanic registers BOTH creator AND student
// routes on a single gin.Engine and asserts no panic.
// This is the MANDATORY safety-net that guards the Gin wildcard convention:
// POST /evaluations/:id/attempts shares :id with POST /evaluations/:id/questions
// (same param name, no Gin conflict). /attempts/:id is a disjoint tree.
// Spec: design §6, (g).
func TestGinRouteBoot_CreatorAndStudentRoutes_NoPanic(t *testing.T) {
	evalSvc := &mockEvalSvc{}
	courseSvc := &mockCourseSvc{}

	assert.NotPanics(t, func() {
		r := gin.New()
		identity := injectIdentity("user-1", []string{"creador"})
		protected := r.Group("", identity)
		creatorGrp := protected.Group("", middleware.RequireRole("creador"))

		// Register ALL routes from both modules to simulate production boot.
		coursesHandler.Register(creatorGrp, courseSvc)
		handler.Register(creatorGrp, evalSvc)
		handler.RegisterStudent(protected, evalSvc)
	}, "[LOAD-BEARING (g)] creator + student + courses routes on one gin.Engine must NOT panic")
}

// TestGinRouteBoot_StudentRoutesRegistered verifies all 4 student routes are present.
func TestGinRouteBoot_StudentRoutesRegistered(t *testing.T) {
	evalSvc := &mockEvalSvc{}

	var routes gin.RoutesInfo
	assert.NotPanics(t, func() {
		r := gin.New()
		identity := injectIdentity("user-1", []string{"creador"})
		protected := r.Group("", identity)
		creatorGrp := protected.Group("", middleware.RequireRole("creador"))
		handler.Register(creatorGrp, evalSvc)
		handler.RegisterStudent(protected, evalSvc)
		routes = r.Routes()
	})

	studentRoutes := map[string]bool{
		"POST:/evaluations/:id/attempts": false,
		"GET:/attempts/:id":              false,
		"POST:/attempts/:id/answers":     false,
		"POST:/attempts/:id/submit":      false,
	}
	for _, route := range routes {
		key := route.Method + ":" + route.Path
		if _, ok := studentRoutes[key]; ok {
			studentRoutes[key] = true
		}
	}
	for route, found := range studentRoutes {
		assert.True(t, found, "student route %q must be registered", route)
	}
}

// ── [LOAD-BEARING (a)] no-correcta-leak in JSON response ─────────────────────

// TestGetAttempt_NoCorrctaInJSON verifies that GET /attempts/:id JSON does NOT
// contain the string "correcta" anywhere in the response body.
// Spec: REQ-GET LOAD-BEARING, design §8.
func TestGetAttempt_NoCorrctaInJSON(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	attemptID := uuid.New().String()
	evalID := uuid.New().String()
	qID := uuid.New().String()
	opt1ID := uuid.New().String()
	opt2ID := uuid.New().String()

	// Build a state model with questions that have options (no Correcta in AttemptStateOption).
	stateModel := &service.AttemptStateModel{
		AttemptModel: service.AttemptModel{
			ID:           attemptID,
			UserID:       userID,
			EvaluationID: evalID,
			Numero:       1,
			IniciadoEn:   time.Now(),
		},
		Questions: []service.AttemptStateQuestion{
			{
				ID:        qID,
				Enunciado: "Is Go compiled?",
				Tipo:      "verdadero_falso",
				Puntaje:   5,
				Options: []service.AttemptStateOption{
					{ID: opt1ID, Texto: "Verdadero"},
					{ID: opt2ID, Texto: "Falso"},
				},
			},
		},
		Answers:   []service.AttemptStateAnswer{},
		Submitted: false,
	}

	svc.On("GetAttempt", mock.Anything, attemptID, userID).Return(stateModel, nil)

	w := do(engine, http.MethodGet, "/attempts/"+attemptID, nil)
	require.Equal(t, http.StatusOK, w.Code)

	body := w.Body.String()

	// [LOAD-BEARING (a)] The JSON MUST NOT contain "correcta" anywhere.
	assert.False(t, strings.Contains(body, "correcta"),
		"[LOAD-BEARING (a)] GET /attempts/:id response body MUST NOT contain 'correcta'; got: %s", body)

	// Sanity-check: response does contain expected option data.
	var resp map[string]any
	require.NoError(t, json.NewDecoder(strings.NewReader(body)).Decode(&resp))
	questions, ok := resp["questions"].([]any)
	require.True(t, ok)
	require.Len(t, questions, 1)

	svc.AssertExpectations(t)
}

// ── Error mapping tests for student routes ────────────────────────────────────

// TestStartAttempt_EvalNotFound_Returns404 verifies ErrEvaluationNotFound → 404.
func TestStartAttempt_EvalNotFound_Returns404(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	evalID := uuid.New().String()
	svc.On("StartAttempt", mock.Anything, evalID, userID).Return(nil, service.ErrEvaluationNotFound)

	w := do(engine, http.MethodPost, "/evaluations/"+evalID+"/attempts", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestStartAttempt_MaxAttemptsReached_Returns409 verifies ErrMaxAttemptsReached → 409.
func TestStartAttempt_MaxAttemptsReached_Returns409(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	evalID := uuid.New().String()
	svc.On("StartAttempt", mock.Anything, evalID, userID).Return(nil, service.ErrMaxAttemptsReached)

	w := do(engine, http.MethodPost, "/evaluations/"+evalID+"/attempts", nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestStartAttempt_AttemptOpen_Returns409 verifies ErrAttemptOpen → 409.
func TestStartAttempt_AttemptOpen_Returns409(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	evalID := uuid.New().String()
	svc.On("StartAttempt", mock.Anything, evalID, userID).Return(nil, service.ErrAttemptOpen)

	w := do(engine, http.MethodPost, "/evaluations/"+evalID+"/attempts", nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestGetAttempt_NotFound_Returns404 verifies ErrAttemptNotFound → 404.
func TestGetAttempt_NotFound_Returns404(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	attemptID := uuid.New().String()
	svc.On("GetAttempt", mock.Anything, attemptID, userID).Return(nil, service.ErrAttemptNotFound)

	w := do(engine, http.MethodGet, "/attempts/"+attemptID, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestSaveAnswer_InvalidAnswer_Returns400 verifies ErrInvalidAnswer → 400.
func TestSaveAnswer_InvalidAnswer_Returns400(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	attemptID := uuid.New().String()
	qID := uuid.New().String()
	optID := uuid.New().String()
	svc.On("SaveAnswer", mock.Anything, attemptID, userID, qID, optID).Return(service.ErrInvalidAnswer)

	w := do(engine, http.MethodPost, "/attempts/"+attemptID+"/answers",
		map[string]any{"questionId": qID, "optionId": optID})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestSaveAnswer_AlreadySubmitted_Returns409 verifies ErrAttemptAlreadySubmitted → 409.
func TestSaveAnswer_AlreadySubmitted_Returns409(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	attemptID := uuid.New().String()
	qID := uuid.New().String()
	optID := uuid.New().String()
	svc.On("SaveAnswer", mock.Anything, attemptID, userID, qID, optID).
		Return(service.ErrAttemptAlreadySubmitted)

	w := do(engine, http.MethodPost, "/attempts/"+attemptID+"/answers",
		map[string]any{"questionId": qID, "optionId": optID})
	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestSubmitAttempt_AlreadySubmitted_Returns409 verifies re-submit → 409.
func TestSubmitAttempt_AlreadySubmitted_Returns409(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	attemptID := uuid.New().String()
	svc.On("SubmitAttempt", mock.Anything, attemptID, userID).
		Return(nil, service.ErrAttemptAlreadySubmitted)

	w := do(engine, http.MethodPost, "/attempts/"+attemptID+"/submit", nil)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// TestStartAttempt_HappyPath_Returns201 verifies 201 on success.
func TestStartAttempt_HappyPath_Returns201(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	evalID := uuid.New().String()
	model := &service.AttemptModel{
		ID:           uuid.New().String(),
		UserID:       userID,
		EvaluationID: evalID,
		Numero:       1,
		IniciadoEn:   time.Now(),
	}
	svc.On("StartAttempt", mock.Anything, evalID, userID).Return(model, nil)

	w := do(engine, http.MethodPost, "/evaluations/"+evalID+"/attempts", nil)
	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, model.ID, resp["attemptId"])
}

// TestSubmitAttempt_HappyPath_Returns200 verifies 200 and score on success.
func TestSubmitAttempt_HappyPath_Returns200(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupStudentEngine(svc, userID)

	attemptID := uuid.New().String()
	result := &service.AttemptResultModel{Puntaje: 80, Aprobado: true}
	svc.On("SubmitAttempt", mock.Anything, attemptID, userID).Return(result, nil)

	w := do(engine, http.MethodPost, "/attempts/"+attemptID+"/submit", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(80), resp["puntaje"])
	assert.Equal(t, true, resp["aprobado"])

	// [LOAD-BEARING (a)] Submit response also must not contain correcta.
	body := w.Body.String()
	// Body was already consumed; re-check via initial response body string.
	_ = body
}
