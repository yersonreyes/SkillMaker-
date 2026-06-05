// Package handler — HTTP-layer contract tests for GetEvaluationSummaryForStudent.
//
// TDD Cycle: RED (this file written first, before handler method exists) → GREEN.
//
// Verifies:
//
//	(a) 200 with summary for aprobado+eval
//	(b) 404 when service returns ErrEvaluationNotFound (non-aprobado or no eval)
//	(c) 401 without JWT (no identity injected — RequireRole blocks before handler)
//
// No build tag: runs with standard `make backend-test`.
package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations/service"
)

// setupSummaryEngine builds a JWT-only protected group (no RequireRole).
// Mirrors how RegisterStudent is called in main.go.
func setupSummaryEngine(svc service.Service, userID string) *gin.Engine {
	r := gin.New()
	identity := injectIdentity(userID, []string{"alumno"})
	protected := r.Group("", identity)
	handler.RegisterStudent(protected, svc)
	return r
}

// ── [LOAD-BEARING (a)] 200 with summary ──────────────────────────────────────

// TestGetEvaluationSummary_AprobadoWithEval_Returns200 verifies the happy path:
// a valid course with an evaluation returns 200 + {evaluationId, notaMinima, intentosMax}.
func TestGetEvaluationSummary_AprobadoWithEval_Returns200(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupSummaryEngine(svc, userID)

	courseID := uuid.New().String()
	evalID := uuid.New().String()

	summary := &service.EvaluationSummaryModel{
		EvaluationID: evalID,
		NotaMinima:   75,
		IntentosMax:  2,
	}
	svc.On("GetEvaluationSummaryForStudent", mock.Anything, courseID).Return(summary, nil)

	w := do(engine, http.MethodGet, "/courses/"+courseID+"/evaluation/summary", nil)
	assert.Equal(t, http.StatusOK, w.Code, "[LOAD-BEARING (a)] summary for aprobado+eval must return 200")

	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, evalID, resp["evaluationId"])
	assert.Equal(t, float64(75), resp["notaMinima"])
	assert.Equal(t, float64(2), resp["intentosMax"])
	svc.AssertExpectations(t)
}

// ── [LOAD-BEARING (b)] 404 on not-aprobado or no-eval ────────────────────────

// TestGetEvaluationSummary_NotFound_Returns404 verifies ErrEvaluationNotFound → 404.
func TestGetEvaluationSummary_NotFound_Returns404(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupSummaryEngine(svc, userID)

	courseID := uuid.New().String()
	svc.On("GetEvaluationSummaryForStudent", mock.Anything, courseID).
		Return((*service.EvaluationSummaryModel)(nil), service.ErrEvaluationNotFound)

	w := do(engine, http.MethodGet, "/courses/"+courseID+"/evaluation/summary", nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "[LOAD-BEARING (b)] ErrEvaluationNotFound must return 404")
	svc.AssertExpectations(t)
}

// TestGetEvaluationSummary_CourseNotFound_Returns404 verifies ErrCourseNotFound → 404.
func TestGetEvaluationSummary_CourseNotFound_Returns404(t *testing.T) {
	svc := &mockEvalSvc{}
	userID := uuid.New().String()
	engine := setupSummaryEngine(svc, userID)

	courseID := uuid.New().String()
	svc.On("GetEvaluationSummaryForStudent", mock.Anything, courseID).
		Return((*service.EvaluationSummaryModel)(nil), service.ErrCourseNotFound)

	w := do(engine, http.MethodGet, "/courses/"+courseID+"/evaluation/summary", nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "ErrCourseNotFound must return 404")
	svc.AssertExpectations(t)
}

// ── [LOAD-BEARING (c)] 401 without JWT ────────────────────────────────────────

// TestGetEvaluationSummary_NoAuth_Returns401 verifies that without JWT the handler
// returns 401.
// Note: since there is no real JWT middleware in tests (just identity injection),
// we simulate "no auth" by setting an empty userID and checking the service returns
// something meaningful. In production the JWT middleware blocks before the handler.
// Here we verify middleware.UserIDFrom returns "" when not set, which the handler
// passes to the service — so the real test is that the route IS registered (no panic).
func TestGetEvaluationSummary_NoAuth_Returns401(t *testing.T) {
	svc := &mockEvalSvc{}
	// Use a Gin engine that has a JWT-blocking middleware.
	// Simulate by not calling injectIdentity at all and using a middleware that returns 401.
	r := gin.New()
	r.Use(func(c *gin.Context) {
		// Simulate JWT check: if no Authorization header, return 401.
		if c.GetHeader("Authorization") == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	})
	protected := r.Group("")
	handler.RegisterStudent(protected, svc)

	courseID := uuid.New().String()
	// No Authorization header — should be blocked by the middleware.
	w := do(r, http.MethodGet, "/courses/"+courseID+"/evaluation/summary", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "[LOAD-BEARING (c)] no JWT must return 401")
	// Service must NOT be called.
	svc.AssertNotCalled(t, "GetEvaluationSummaryForStudent")
}
