// Package handler — HTTP-layer contract tests for the courses module.
//
// Strategy: spin up a real gin.Engine per test (gin.New(), no global state),
// mount the routes with middleware chain (injectIdentity → RequireRole("creador")),
// wire a mockCourseSvc, fire requests via net/http/httptest.
//
// This suite does NOT re-test service logic — that lives in service/service_test.go.
// It tests: RBAC enforcement, identity resolution, error→status mapping, body binding.
//
// CRITICAL (LOAD-BEARING-C): TWO SEPARATE tests for ErrNotOwner:
//   - TestGetByID_NonOwner_Returns404 → renderCourseErrorRead → 404
//   - TestUpdate_NonOwner_Returns403  → renderCourseErrorWrite → 403
//
// Both reference the SAME sentinel. They MUST NOT be collapsed into one test.
//
// No build tag: runs with standard `make backend-test`.
package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock service ───────────────────────────────────────────────────────────────

// mockCourseSvc is a local testify/mock implementation of service.Service.
type mockCourseSvc struct {
	mock.Mock
}

func (m *mockCourseSvc) Create(ctx context.Context, creadorID string, req service.CreateRequest) (*service.CourseModel, error) {
	args := m.Called(ctx, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.CourseModel), args.Error(1)
}

func (m *mockCourseSvc) GetByID(ctx context.Context, id, creadorID string) (*service.CourseModel, error) {
	args := m.Called(ctx, id, creadorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.CourseModel), args.Error(1)
}

func (m *mockCourseSvc) UpdateByID(ctx context.Context, id, creadorID string, req service.UpdateRequest) (*service.CourseModel, error) {
	args := m.Called(ctx, id, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.CourseModel), args.Error(1)
}

func (m *mockCourseSvc) ListByCreator(ctx context.Context, creadorID string, p pagination.Params) (pagination.Page[service.CourseModel], error) {
	args := m.Called(ctx, creadorID, p)
	return args.Get(0).(pagination.Page[service.CourseModel]), args.Error(1)
}

// ── Fixtures ───────────────────────────────────────────────────────────────────

func courseModel(id, creadorID string) *service.CourseModel {
	return &service.CourseModel{
		ID:          id,
		CreadorID:   creadorID,
		Titulo:      "Test Course",
		Descripcion: "Test Description",
		Estado:      "borrador",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ── Engine builder ─────────────────────────────────────────────────────────────

// injectIdentity mimics what middleware.JWT injects into the Gin context.
func injectIdentity(userID string, roles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("roles", roles)
		c.Next()
	}
}

// setupEngine builds a gin.Engine with courses routes under RequireRole("creador").
// The caller controls userID and roles to simulate different RBAC scenarios.
func setupEngine(svc service.Service, userID string, roles []string) *gin.Engine {
	r := gin.New()
	identity := injectIdentity(userID, roles)
	creatorGrp := r.Group("", identity, middleware.RequireRole("creador"))
	handler.Register(creatorGrp, svc)
	return r
}

// do fires an HTTP request against the engine and returns the recorder.
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

func respCode(w *httptest.ResponseRecorder) string {
	var out struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(w.Body).Decode(&out)
	return out.Code
}

// ── RBAC tests ─────────────────────────────────────────────────────────────────

// TestCreate_403_AdminRole verifies that a user WITHOUT the creador role cannot
// hit the create endpoint. Satisfies: REQ-CREATE "Non-creador role is rejected", AC4.
func TestCreate_403_AdminRole(t *testing.T) {
	svc := &mockCourseSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"}) // no "creador" role

	w := do(engine, http.MethodPost, "/courses", map[string]any{
		"titulo": "Test", "descripcion": "Desc",
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "FORBIDDEN", respCode(w))
}

// ── POST /courses ──────────────────────────────────────────────────────────────

// TestCreate_201_EstadoBorrador verifies the happy-path create returns 201 with estado=borrador.
// Satisfies: REQ-CREATE happy path, AC1.
func TestCreate_201_EstadoBorrador(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	model := courseModel("course-1", creadorID)
	svc.On("Create", mock.Anything, creadorID, service.CreateRequest{
		Titulo:      "Go avanzado",
		Descripcion: "Curso de Go",
	}).Return(model, nil)

	w := do(engine, http.MethodPost, "/courses", map[string]any{
		"titulo": "Go avanzado", "descripcion": "Curso de Go",
	})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require := assert.New(t)
	_ = json.NewDecoder(w.Body).Decode(&resp)
	require.Equal("borrador", resp["estado"],
		"response estado must be borrador (service forced it)")
	svc.AssertExpectations(t)
}

// TestCreate_400_MissingTitulo verifies that omitting titulo returns 400.
// Satisfies: REQ-CREATE "Missing titulo".
func TestCreate_400_MissingTitulo(t *testing.T) {
	svc := &mockCourseSvc{}
	engine := setupEngine(svc, "creador-1", []string{"creador"})

	// Body with empty titulo (binding:"required,min=3" must reject this).
	w := do(engine, http.MethodPost, "/courses", map[string]any{
		"titulo": "", "descripcion": "Desc",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── GET /courses/:id ───────────────────────────────────────────────────────────

// TestGetByID_Owner_Returns200 verifies that the owner gets a 200 with course detail.
func TestGetByID_Owner_Returns200(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	model := courseModel(courseID, creadorID)
	svc.On("GetByID", mock.Anything, courseID, creadorID).Return(model, nil)

	w := do(engine, http.MethodGet, "/courses/"+courseID, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

// [LOAD-BEARING-C] TestGetByID_NonOwner_Returns404 verifies that ErrNotOwner on
// a GET route returns 404 (via renderCourseErrorRead) — existence must not be leaked.
// Satisfies: REQ-DETAIL "Different creador gets 404", REQ-DIVERGENCE.
func TestGetByID_NonOwner_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	requesterID := "creador-requester"
	courseID := "course-1"
	engine := setupEngine(svc, requesterID, []string{"creador"})

	// Service returns ErrNotOwner — same sentinel used by the PATCH test below.
	svc.On("GetByID", mock.Anything, courseID, requesterID).Return(nil, service.ErrNotOwner)

	w := do(engine, http.MethodGet, "/courses/"+courseID, nil)

	// renderCourseErrorRead maps ErrNotOwner → 404 (hides existence).
	assert.Equal(t, http.StatusNotFound, w.Code,
		"[LOAD-BEARING-C] ErrNotOwner on GET must return 404 via renderCourseErrorRead")
	svc.AssertExpectations(t)
}

// TestGetByID_NotFound_Returns404 verifies that ErrCourseNotFound on GET returns 404.
func TestGetByID_NotFound_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "nonexistent"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("GetByID", mock.Anything, courseID, creadorID).Return(nil, service.ErrCourseNotFound)

	w := do(engine, http.MethodGet, "/courses/"+courseID, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

// ── PATCH /courses/:id ─────────────────────────────────────────────────────────

// [LOAD-BEARING-C cont.] TestUpdate_NonOwner_Returns403 verifies that ErrNotOwner on
// a PATCH route returns 403 (via renderCourseErrorWrite) — the roadmap AC3 requirement.
// This is a SEPARATE test from TestGetByID_NonOwner_Returns404, asserting the OPPOSITE
// status code for the SAME sentinel.
// Satisfies: REQ-UPDATE "Non-owner edit returns 403", REQ-DIVERGENCE, AC3.
func TestUpdate_NonOwner_Returns403(t *testing.T) {
	svc := &mockCourseSvc{}
	requesterID := "creador-requester"
	courseID := "course-1"
	engine := setupEngine(svc, requesterID, []string{"creador"})

	// Service returns ErrNotOwner — same sentinel as the GET test above.
	svc.On("UpdateByID", mock.Anything, courseID, requesterID, mock.AnythingOfType("service.UpdateRequest")).
		Return(nil, service.ErrNotOwner)

	w := do(engine, http.MethodPatch, "/courses/"+courseID, map[string]any{
		"titulo": "Nuevo titulo", // min=3 must pass binding
	})

	// renderCourseErrorWrite maps ErrNotOwner → 403 (signals authz failure).
	assert.Equal(t, http.StatusForbidden, w.Code,
		"[LOAD-BEARING-C] ErrNotOwner on PATCH must return 403 via renderCourseErrorWrite")
	svc.AssertExpectations(t)
}

// TestUpdate_EnRevision_Returns409 verifies that ErrInvalidTransition returns 409.
// Satisfies: REQ-UPDATE "Edit blocked" scenarios.
func TestUpdate_EnRevision_Returns409(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("UpdateByID", mock.Anything, courseID, creadorID, mock.AnythingOfType("service.UpdateRequest")).
		Return(nil, service.ErrInvalidTransition)

	titulo := "Nuevo titulo" // min=3 must pass binding
	w := do(engine, http.MethodPatch, "/courses/"+courseID, map[string]any{"titulo": titulo})
	assert.Equal(t, http.StatusConflict, w.Code)
	svc.AssertExpectations(t)
}

// TestUpdate_NotFound_Returns404 verifies that ErrCourseNotFound on PATCH returns 404.
func TestUpdate_NotFound_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "nonexistent"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("UpdateByID", mock.Anything, courseID, creadorID, mock.AnythingOfType("service.UpdateRequest")).
		Return(nil, service.ErrCourseNotFound)

	titulo := "Nuevo titulo" // min=3 must pass binding
	w := do(engine, http.MethodPatch, "/courses/"+courseID, map[string]any{"titulo": titulo})
	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

// ── GET /courses?creator=me ────────────────────────────────────────────────────

// TestList_CreatorNotMe_Returns400 verifies that ?creator=<anything-other-than-me> returns 400.
// Satisfies: REQ-LIST "creator value is not me".
func TestList_CreatorNotMe_Returns400(t *testing.T) {
	svc := &mockCourseSvc{}
	engine := setupEngine(svc, "creador-1", []string{"creador"})

	w := do(engine, http.MethodGet, "/courses?creator=some-uuid", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_PARAM", respCode(w))
}

// TestList_PaginatedPage verifies that ?creator=me&page=2&size=10 returns 200 with Page envelope.
// Satisfies: REQ-LIST pagination scenarios, AC5.
func TestList_PaginatedPage(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	emptyPage := pagination.Page[service.CourseModel]{
		Items:      []service.CourseModel{},
		Page:       2,
		Size:       10,
		Total:      0,
		TotalPages: 0,
	}
	svc.On("ListByCreator", mock.Anything, creadorID, pagination.Params{Page: 2, Size: 10}).
		Return(emptyPage, nil)

	w := do(engine, http.MethodGet, "/courses?creator=me&page=2&size=10", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

// TestList_MissingCreator_Returns400 verifies that missing ?creator param returns 400.
func TestList_MissingCreator_Returns400(t *testing.T) {
	svc := &mockCourseSvc{}
	engine := setupEngine(svc, "creador-1", []string{"creador"})

	w := do(engine, http.MethodGet, "/courses", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
