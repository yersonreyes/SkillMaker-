// Package handler — HTTP-layer contract tests for the approvals module.
//
// Strategy: spin up a real gin.Engine per test, mount routes with appropriate
// middleware, wire a mockApprovalSvc, fire requests via net/http/httptest.
// No build tag: runs with standard `make backend-test`.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/domain"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/approvals/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/evaluations"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock approval service ──────────────────────────────────────────────────────

type mockApprovalSvc struct {
	mock.Mock
}

func (m *mockApprovalSvc) SubmitToReview(ctx context.Context, courseID, callerID string) error {
	args := m.Called(ctx, courseID, callerID)
	return args.Error(0)
}

func (m *mockApprovalSvc) Approve(ctx context.Context, courseID, adminID, comentario string) error {
	args := m.Called(ctx, courseID, adminID, comentario)
	return args.Error(0)
}

func (m *mockApprovalSvc) Reject(ctx context.Context, courseID, adminID, comentario string) error {
	args := m.Called(ctx, courseID, adminID, comentario)
	return args.Error(0)
}

func (m *mockApprovalSvc) ListPending(ctx context.Context) ([]courses.CourseSummary, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]courses.CourseSummary), args.Error(1)
}

func (m *mockApprovalSvc) ListHistory(ctx context.Context, courseID, callerID string, isAdmin bool) ([]domain.Approval, error) {
	args := m.Called(ctx, courseID, callerID, isAdmin)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Approval), args.Error(1)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func injectIdentity(userID string, roles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("roles", roles)
		c.Next()
	}
}

func setupEngine(svc service.Service, callerID string, roles []string) *gin.Engine {
	r := gin.New()
	identity := injectIdentity(callerID, roles)
	protected := r.Group("", identity)
	creatorGrp := protected.Group("", middleware.RequireRole("creador"))
	adminGrp := protected.Group("", middleware.RequireRole("administrador"))

	handler.RegisterCreator(creatorGrp, svc)
	handler.RegisterAdmin(adminGrp, svc)
	handler.RegisterHistory(protected, svc)
	return r
}

func do(engine *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf.Write(b)
	}
	req := httptest.NewRequest(method, path, &buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// ── Submit tests ──────────────────────────────────────────────────────────────

// TestSubmit_Happy verifies 200 on successful submit.
func TestSubmit_Happy(t *testing.T) {
	svc := &mockApprovalSvc{}
	callerID := "user-123"
	courseID := "course-abc"

	svc.On("SubmitToReview", mock.Anything, courseID, callerID).Return(nil)

	engine := setupEngine(svc, callerID, []string{"creador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/submit", nil)

	assert.Equal(t, http.StatusOK, w.Code, "submit happy path must return 200")
	svc.AssertExpectations(t)
}

// TestSubmit_NotOwner_Returns403 verifies ErrNotOwner → 403.
func TestSubmit_NotOwner_Returns403(t *testing.T) {
	svc := &mockApprovalSvc{}
	callerID := "user-123"
	courseID := "course-abc"

	svc.On("SubmitToReview", mock.Anything, courseID, callerID).Return(service.ErrNotOwner)

	engine := setupEngine(svc, callerID, []string{"creador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/submit", nil)

	assert.Equal(t, http.StatusForbidden, w.Code, "ErrNotOwner on submit must return 403")
	svc.AssertExpectations(t)
}

// TestSubmit_NotSubmittable_Returns409 verifies ErrCourseNotSubmittable → 409.
func TestSubmit_NotSubmittable_Returns409(t *testing.T) {
	svc := &mockApprovalSvc{}
	callerID := "user-123"
	courseID := "course-abc"

	svc.On("SubmitToReview", mock.Anything, courseID, callerID).Return(service.ErrCourseNotSubmittable)

	engine := setupEngine(svc, callerID, []string{"creador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/submit", nil)

	assert.Equal(t, http.StatusConflict, w.Code, "ErrCourseNotSubmittable must return 409")
}

// TestSubmit_NoContent_Returns409 verifies ErrNoContent → 409.
func TestSubmit_NoContent_Returns409(t *testing.T) {
	svc := &mockApprovalSvc{}
	callerID := "user-123"
	courseID := "course-abc"

	svc.On("SubmitToReview", mock.Anything, courseID, callerID).Return(service.ErrNoContent)

	engine := setupEngine(svc, callerID, []string{"creador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/submit", nil)

	assert.Equal(t, http.StatusConflict, w.Code, "ErrNoContent must return 409")
}

// TestSubmit_CourseNotFound_Returns404 verifies ErrCourseNotFound → 404.
func TestSubmit_CourseNotFound_Returns404(t *testing.T) {
	svc := &mockApprovalSvc{}
	callerID := "user-123"
	courseID := "nonexistent-course"

	svc.On("SubmitToReview", mock.Anything, courseID, callerID).Return(service.ErrCourseNotFound)

	engine := setupEngine(svc, callerID, []string{"creador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/submit", nil)

	assert.Equal(t, http.StatusNotFound, w.Code, "ErrCourseNotFound must return 404")
}

// ── ListPending tests ─────────────────────────────────────────────────────────

// TestListPending_Admin_Returns200 verifies admin can list pending.
func TestListPending_Admin_Returns200(t *testing.T) {
	svc := &mockApprovalSvc{}
	adminID := "admin-1"

	pending := []courses.CourseSummary{
		{ID: "course-1", Titulo: "Pending Course", Estado: "en_revision"},
	}
	svc.On("ListPending", mock.Anything).Return(pending, nil)

	engine := setupEngine(svc, adminID, []string{"administrador"})
	w := do(engine, http.MethodGet, "/approvals/pending", nil)

	assert.Equal(t, http.StatusOK, w.Code, "admin list pending must return 200")
	svc.AssertExpectations(t)
}

// TestListPending_Creador_Returns403 verifies non-admin gets 403.
func TestListPending_Creador_Returns403(t *testing.T) {
	svc := &mockApprovalSvc{}

	engine := setupEngine(svc, "creador-1", []string{"creador"})
	w := do(engine, http.MethodGet, "/approvals/pending", nil)

	assert.Equal(t, http.StatusForbidden, w.Code, "non-admin list pending must return 403")
	svc.AssertNotCalled(t, "ListPending")
}

// ── Approve tests ─────────────────────────────────────────────────────────────

// TestApprove_Happy_Returns200 verifies admin approve returns 200.
func TestApprove_Happy_Returns200(t *testing.T) {
	svc := &mockApprovalSvc{}
	adminID := "admin-1"
	courseID := "course-abc"

	svc.On("Approve", mock.Anything, courseID, adminID, "").Return(nil)

	engine := setupEngine(svc, adminID, []string{"administrador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/approve", map[string]string{})

	assert.Equal(t, http.StatusOK, w.Code, "approve happy path must return 200")
	svc.AssertExpectations(t)
}

// TestApprove_NotInReview_Returns409 verifies ErrNotInReview → 409.
func TestApprove_NotInReview_Returns409(t *testing.T) {
	svc := &mockApprovalSvc{}
	adminID := "admin-1"
	courseID := "course-abc"

	svc.On("Approve", mock.Anything, courseID, adminID, mock.Anything).Return(service.ErrNotInReview)

	engine := setupEngine(svc, adminID, []string{"administrador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/approve", map[string]string{})

	assert.Equal(t, http.StatusConflict, w.Code, "ErrNotInReview must return 409")
}

// ── Reject tests ──────────────────────────────────────────────────────────────

// TestReject_Happy_Returns200 verifies reject with comment returns 200.
func TestReject_Happy_Returns200(t *testing.T) {
	svc := &mockApprovalSvc{}
	adminID := "admin-1"
	courseID := "course-abc"

	svc.On("Reject", mock.Anything, courseID, adminID, "Needs improvement").Return(nil)

	engine := setupEngine(svc, adminID, []string{"administrador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/reject",
		map[string]string{"comentario": "Needs improvement"})

	assert.Equal(t, http.StatusOK, w.Code, "reject with comment must return 200")
	svc.AssertExpectations(t)
}

// TestReject_EmptyComment_Returns400 verifies ErrCommentRequired → 400.
func TestReject_EmptyComment_Returns400(t *testing.T) {
	svc := &mockApprovalSvc{}
	adminID := "admin-1"
	courseID := "course-abc"

	svc.On("Reject", mock.Anything, courseID, adminID, "").Return(service.ErrCommentRequired)

	engine := setupEngine(svc, adminID, []string{"administrador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/reject",
		map[string]string{"comentario": ""})

	assert.Equal(t, http.StatusBadRequest, w.Code, "ErrCommentRequired must return 400")
}

// TestReject_MissingBody_Returns400 verifies missing body returns 400.
func TestReject_MissingBody_Returns400(t *testing.T) {
	svc := &mockApprovalSvc{}
	adminID := "admin-1"
	courseID := "course-abc"

	svc.On("Reject", mock.Anything, courseID, adminID, "").Return(service.ErrCommentRequired)

	engine := setupEngine(svc, adminID, []string{"administrador"})
	// No body — empty comentario will be passed to service.
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/reject", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code, "missing comentario must return 400")
}

// ── ListHistory tests ─────────────────────────────────────────────────────────

// TestListHistory_Owner_Returns200 verifies owner can see history.
func TestListHistory_Owner_Returns200(t *testing.T) {
	svc := &mockApprovalSvc{}
	ownerID := "user-123"
	courseID := "course-abc"

	rows := []domain.Approval{
		{ID: "approval-1", CourseID: courseID, Resultado: "rechazado", Comentario: "Too short"},
	}
	svc.On("ListHistory", mock.Anything, courseID, ownerID, false).Return(rows, nil)

	engine := setupEngine(svc, ownerID, []string{"creador"})
	w := do(engine, http.MethodGet, "/courses/"+courseID+"/approvals", nil)

	assert.Equal(t, http.StatusOK, w.Code, "owner history must return 200")
	svc.AssertExpectations(t)
}

// TestListHistory_NonOwner_Returns404 verifies non-owner ErrNotOwner → 404 (read route hides existence).
func TestListHistory_NonOwner_Returns404(t *testing.T) {
	svc := &mockApprovalSvc{}
	callerID := "other-user"
	courseID := "course-abc"

	svc.On("ListHistory", mock.Anything, courseID, callerID, false).Return(nil, service.ErrNotOwner)

	engine := setupEngine(svc, callerID, []string{"creador"})
	w := do(engine, http.MethodGet, "/courses/"+courseID+"/approvals", nil)

	assert.Equal(t, http.StatusNotFound, w.Code, "ErrNotOwner on history read must return 404 (hides existence)")
	svc.AssertExpectations(t)
}

// TestListHistory_Admin_Returns200 verifies admin can see any course's history.
func TestListHistory_Admin_Returns200(t *testing.T) {
	svc := &mockApprovalSvc{}
	adminID := "admin-1"
	courseID := "course-abc"

	rows := []domain.Approval{{ID: "approval-1", CourseID: courseID, Resultado: "aprobado"}}
	svc.On("ListHistory", mock.Anything, courseID, adminID, true).Return(rows, nil)

	engine := setupEngine(svc, adminID, []string{"administrador"})
	w := do(engine, http.MethodGet, "/courses/"+courseID+"/approvals", nil)

	require.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

// ── W-1: eval-incomplete sentinel tests ───────────────────────────────────────

// TestSubmit_EvalIncomplete_Returns409 verifies that when SubmitToReview returns
// evaluations.ErrNoCorrectOption (the real sentinel), the handler renders 409
// per the renderWriteError switch (W-1 verify warning).
func TestSubmit_EvalIncomplete_Returns409(t *testing.T) {
	svc := &mockApprovalSvc{}
	callerID := "user-123"
	courseID := "course-abc"

	// Return the REAL evaluations.ErrNoCorrectOption so errors.Is on the handler switch works.
	svc.On("SubmitToReview", mock.Anything, courseID, callerID).Return(evaluations.ErrNoCorrectOption)

	engine := setupEngine(svc, callerID, []string{"creador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/submit", nil)

	assert.Equal(t, http.StatusConflict, w.Code,
		"evaluations.ErrNoCorrectOption (eval incomplete) must map to 409 Conflict")
	svc.AssertExpectations(t)
}

// ── W-2: non-admin authz tests for approve and reject ─────────────────────────

// TestApprove_NonAdmin_Returns403 verifies that a creador calling the approve endpoint
// is rejected with 403 before the service is ever called (route-group gate via RequireRole).
// Mirrors TestListPending_Creador_Returns403 exactly.
func TestApprove_NonAdmin_Returns403(t *testing.T) {
	svc := &mockApprovalSvc{}
	courseID := "course-abc"

	engine := setupEngine(svc, "creador-1", []string{"creador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/approve", map[string]string{})

	assert.Equal(t, http.StatusForbidden, w.Code, "non-admin calling approve must get 403")
	svc.AssertNotCalled(t, "Approve")
}

// TestReject_NonAdmin_Returns403 verifies that a creador calling the reject endpoint
// is rejected with 403 before the service is ever called (route-group gate via RequireRole).
// Mirrors TestListPending_Creador_Returns403 exactly.
func TestReject_NonAdmin_Returns403(t *testing.T) {
	svc := &mockApprovalSvc{}
	courseID := "course-abc"

	engine := setupEngine(svc, "creador-1", []string{"creador"})
	w := do(engine, http.MethodPost, "/courses/"+courseID+"/reject",
		map[string]string{"comentario": "any comment"})

	assert.Equal(t, http.StatusForbidden, w.Code, "non-admin calling reject must get 403")
	svc.AssertNotCalled(t, "Reject")
}

// ── W-3: empty pending list returns [] not null ───────────────────────────────

// TestListPending_EmptyList_Returns200WithEmptyArray verifies that GET /approvals/pending
// with zero en_revision courses returns HTTP 200 with an empty JSON array `[]` (not null).
// This exercises the dto.ToPending nil-to-empty-slice guarantee (spec REQ-PENDING empty scenario).
func TestListPending_EmptyList_Returns200WithEmptyArray(t *testing.T) {
	svc := &mockApprovalSvc{}
	adminID := "admin-1"

	// Empty slice (not nil) — simulates zero courses in en_revision.
	svc.On("ListPending", mock.Anything).Return([]courses.CourseSummary{}, nil)

	engine := setupEngine(svc, adminID, []string{"administrador"})
	w := do(engine, http.MethodGet, "/approvals/pending", nil)

	require.Equal(t, http.StatusOK, w.Code, "empty pending list must return 200")
	svc.AssertExpectations(t)

	// Must be `[]` not `null` — JSON null is a different type from an empty array.
	// Trim whitespace to be robust against Gin's trailing newline behavior.
	body := strings.TrimSpace(w.Body.String())
	assert.Equal(t, "[]", body,
		"empty pending list must serialize to JSON array [] not null")
}
