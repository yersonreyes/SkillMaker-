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
// C2.2 additions:
//   - TestCreateVideo_UrlProviderMismatch_Returns400 [LB-1]
//   - TestCreateVideo_NonOwner_Returns403 [LB-2]
//   - TestCreateSection_NonOwner_Returns403
//   - TestGetByID_HasContent_*
//   - TestReorderSections_* error mapping
//   - Error map: ErrSectionNotFound→404, ErrVideoNotFound→404, ErrInvalidTransition→409
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

func (m *mockCourseSvc) CreateSection(ctx context.Context, creadorID string, req service.SectionCreateRequest) (*service.SectionModel, error) {
	args := m.Called(ctx, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SectionModel), args.Error(1)
}

func (m *mockCourseSvc) GetSectionByID(ctx context.Context, id string) (*service.SectionModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SectionModel), args.Error(1)
}

func (m *mockCourseSvc) UpdateSection(ctx context.Context, id, creadorID string, req service.SectionUpdateRequest) (*service.SectionModel, error) {
	args := m.Called(ctx, id, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.SectionModel), args.Error(1)
}

func (m *mockCourseSvc) DeleteSection(ctx context.Context, id, creadorID string) error {
	args := m.Called(ctx, id, creadorID)
	return args.Error(0)
}

func (m *mockCourseSvc) ListSections(ctx context.Context, courseID string) ([]service.SectionModel, error) {
	args := m.Called(ctx, courseID)
	return args.Get(0).([]service.SectionModel), args.Error(1)
}

func (m *mockCourseSvc) ReorderSections(ctx context.Context, courseID, creadorID string, ids []string) error {
	args := m.Called(ctx, courseID, creadorID, ids)
	return args.Error(0)
}

func (m *mockCourseSvc) CreateVideo(ctx context.Context, creadorID string, req service.VideoCreateRequest) (*service.VideoModel, error) {
	args := m.Called(ctx, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.VideoModel), args.Error(1)
}

func (m *mockCourseSvc) UpdateVideo(ctx context.Context, id, creadorID string, req service.VideoUpdateRequest) (*service.VideoModel, error) {
	args := m.Called(ctx, id, creadorID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.VideoModel), args.Error(1)
}

func (m *mockCourseSvc) DeleteVideo(ctx context.Context, id, creadorID string) error {
	args := m.Called(ctx, id, creadorID)
	return args.Error(0)
}

func (m *mockCourseSvc) ListVideos(ctx context.Context, sectionID string) ([]service.VideoModel, error) {
	args := m.Called(ctx, sectionID)
	return args.Get(0).([]service.VideoModel), args.Error(1)
}

func (m *mockCourseSvc) ListContent(ctx context.Context, courseID, creadorID string) ([]service.SectionWithVideosModel, error) {
	args := m.Called(ctx, courseID, creadorID)
	return args.Get(0).([]service.SectionWithVideosModel), args.Error(1)
}

func (m *mockCourseSvc) HasContent(ctx context.Context, courseID, creadorID string) (bool, error) {
	args := m.Called(ctx, courseID, creadorID)
	return args.Bool(0), args.Error(1)
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

func videoModel(id, sectionID string) *service.VideoModel {
	return &service.VideoModel{
		ID:        id,
		SectionID: sectionID,
		Titulo:    "Test Video",
		URL:       "https://www.youtube.com/watch?v=abc123",
		Proveedor: "youtube",
		DuracionS: 120,
		Orden:     0,
		CreatedAt: time.Now(),
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
	svc.On("HasContent", mock.Anything, courseID, creadorID).Return(false, nil)

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

// TestGetByID_HasContent_True verifies GET returns hasContent:true when svc.HasContent returns true.
// Spec: HC-1-A.
func TestGetByID_HasContent_True(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	model := courseModel(courseID, creadorID)
	svc.On("GetByID", mock.Anything, courseID, creadorID).Return(model, nil)
	svc.On("HasContent", mock.Anything, courseID, creadorID).Return(true, nil)

	w := do(engine, http.MethodGet, "/courses/"+courseID, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, true, resp["hasContent"], "hasContent must be true when course has videos")
	svc.AssertExpectations(t)
}

// TestGetByID_HasContent_False verifies GET returns hasContent:false when no videos.
// Spec: HC-1-B, HC-1-C.
func TestGetByID_HasContent_False(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	model := courseModel(courseID, creadorID)
	svc.On("GetByID", mock.Anything, courseID, creadorID).Return(model, nil)
	svc.On("HasContent", mock.Anything, courseID, creadorID).Return(false, nil)

	w := do(engine, http.MethodGet, "/courses/"+courseID, nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, false, resp["hasContent"], "hasContent must be false when no videos")
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

// ── Section handler tests (C2.2) ──────────────────────────────────────────────

// TestCreateSection_NonOwner_Returns403 verifies that a non-owner gets 403.
// Spec: SEC-1-B.
func TestCreateSection_NonOwner_Returns403(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("CreateSection", mock.Anything, creadorID, mock.AnythingOfType("service.SectionCreateRequest")).
		Return(nil, service.ErrNotOwner)

	w := do(engine, http.MethodPost, "/courses/"+courseID+"/sections", map[string]any{
		"titulo": "Intro",
	})
	assert.Equal(t, http.StatusForbidden, w.Code, "non-owner CreateSection must return 403")
	svc.AssertExpectations(t)
}

// TestDeleteSection_Returns204 verifies a successful delete returns 204.
// Spec: SEC-3-A.
func TestDeleteSection_Returns204(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	sectionID := "section-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("DeleteSection", mock.Anything, sectionID, creadorID).Return(nil)

	w := do(engine, http.MethodDelete, "/sections/"+sectionID, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
	svc.AssertExpectations(t)
}

// TestReorderSections_ForeignID_Returns400 verifies that a foreign section ID returns 400.
// Spec: ROR-1-B. ErrInvalidReorderSet is a validation error (wrong input set) → 400, NOT 409.
func TestReorderSections_ForeignID_Returns400(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("ReorderSections", mock.Anything, courseID, creadorID, mock.Anything).
		Return(service.ErrInvalidReorderSet)

	w := do(engine, http.MethodPatch, "/courses/"+courseID+"/sections/reorder", map[string]any{
		"ids": []string{"s1", "s2", "foreign-id"},
	})
	// ErrInvalidReorderSet → 400 (INVALID_REORDER_SET): caller sent wrong section IDs.
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"foreign section ID in reorder must return 400 (ROR-1-B)")
	assert.Equal(t, "INVALID_REORDER_SET", respCode(w))
	svc.AssertExpectations(t)
}

// TestReorderSections_ValidReorder_Returns200 verifies successful reorder returns 200.
// Spec: ROR-1-A.
func TestReorderSections_ValidReorder_Returns200(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	ids := []string{"s3", "s1", "s2"}
	svc.On("ReorderSections", mock.Anything, courseID, creadorID, ids).Return(nil)

	w := do(engine, http.MethodPatch, "/courses/"+courseID+"/sections/reorder", map[string]any{
		"ids": ids,
	})
	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

// TestReorderSections_NonOwner_Returns403 verifies that a non-owner calling the
// reorder route receives 403. Spec: ROR-1-D.
func TestReorderSections_NonOwner_Returns403(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-other"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("ReorderSections", mock.Anything, courseID, creadorID, mock.Anything).
		Return(service.ErrNotOwner)

	w := do(engine, http.MethodPatch, "/courses/"+courseID+"/sections/reorder", map[string]any{
		"ids": []string{"s1", "s2"},
	})
	assert.Equal(t, http.StatusForbidden, w.Code,
		"non-owner reorder must return 403 (ROR-1-D)")
	assert.Equal(t, "NOT_OWNER", respCode(w))
	svc.AssertExpectations(t)
}

// ── Video handler tests (C2.2) ────────────────────────────────────────────────

// [LOAD-BEARING: LB-1] TestCreateVideo_UrlProviderMismatch_Returns400 verifies that a
// Vimeo URL with proveedor=youtube returns 400.
// Spec: VID-1-E.
func TestCreateVideo_UrlProviderMismatch_Returns400(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	sectionID := "section-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("CreateVideo", mock.Anything, creadorID, mock.AnythingOfType("service.VideoCreateRequest")).
		Return(nil, service.ErrURLProviderMismatch)

	w := do(engine, http.MethodPost, "/sections/"+sectionID+"/videos", map[string]any{
		"titulo":    "Test Video",
		"url":       "https://vimeo.com/123",
		"proveedor": "youtube", // MISMATCH
	})
	assert.Equal(t, http.StatusBadRequest, w.Code,
		"[LB-1] vimeo URL + proveedor=youtube must return 400")
	assert.Equal(t, "URL_PROVIDER_MISMATCH", respCode(w))
	svc.AssertExpectations(t)
}

// [LOAD-BEARING: LB-2] TestCreateVideo_NonOwner_Returns403 verifies that a non-owner gets 403.
// Spec: VID-1-F.
func TestCreateVideo_NonOwner_Returns403(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	sectionID := "section-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("CreateVideo", mock.Anything, creadorID, mock.AnythingOfType("service.VideoCreateRequest")).
		Return(nil, service.ErrNotOwner)

	w := do(engine, http.MethodPost, "/sections/"+sectionID+"/videos", map[string]any{
		"titulo":    "Test Video",
		"url":       "https://www.youtube.com/watch?v=abc",
		"proveedor": "youtube",
	})
	assert.Equal(t, http.StatusForbidden, w.Code,
		"[LB-2] non-owner video creation must return 403")
	svc.AssertExpectations(t)
}

// TestDeleteVideo_Returns204 verifies a successful video delete returns 204.
// Spec: VID-3-A.
func TestDeleteVideo_Returns204(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	videoID := "video-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("DeleteVideo", mock.Anything, videoID, creadorID).Return(nil)

	w := do(engine, http.MethodDelete, "/videos/"+videoID, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
	svc.AssertExpectations(t)
}

// ── Error map coverage tests ──────────────────────────────────────────────────

// TestSectionNotFound_Returns404 verifies ErrSectionNotFound → 404.
// Spec: ERR-1.
func TestSectionNotFound_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	sectionID := "nonexistent-section"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("DeleteSection", mock.Anything, sectionID, creadorID).Return(service.ErrSectionNotFound)

	w := do(engine, http.MethodDelete, "/sections/"+sectionID, nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "ErrSectionNotFound must return 404")
	assert.Equal(t, "SECTION_NOT_FOUND", respCode(w))
	svc.AssertExpectations(t)
}

// TestVideoNotFound_Returns404 verifies ErrVideoNotFound → 404.
// Spec: ERR-1.
func TestVideoNotFound_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	videoID := "nonexistent-video"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("DeleteVideo", mock.Anything, videoID, creadorID).Return(service.ErrVideoNotFound)

	w := do(engine, http.MethodDelete, "/videos/"+videoID, nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "ErrVideoNotFound must return 404")
	assert.Equal(t, "VIDEO_NOT_FOUND", respCode(w))
	svc.AssertExpectations(t)
}

// TestCreateSection_EnRevision_Returns409 verifies ErrInvalidTransition → 409.
// Spec: SEC-1-E, ERR-1.
func TestCreateSection_EnRevision_Returns409(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("CreateSection", mock.Anything, creadorID, mock.AnythingOfType("service.SectionCreateRequest")).
		Return(nil, service.ErrInvalidTransition)

	w := do(engine, http.MethodPost, "/courses/"+courseID+"/sections", map[string]any{
		"titulo": "Intro",
	})
	assert.Equal(t, http.StatusConflict, w.Code, "ErrInvalidTransition must return 409")
	svc.AssertExpectations(t)
}

// TestUpdateVideo_NonOwner_Returns403 verifies ErrNotOwner on PATCH video → 403.
// Spec: VID-2-C.
func TestUpdateVideo_NonOwner_Returns403(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-other"
	videoID := "video-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("UpdateVideo", mock.Anything, videoID, creadorID, mock.AnythingOfType("service.VideoUpdateRequest")).
		Return(nil, service.ErrNotOwner)

	titulo := "New Title"
	w := do(engine, http.MethodPatch, "/videos/"+videoID, map[string]any{"titulo": titulo})
	assert.Equal(t, http.StatusForbidden, w.Code, "non-owner UpdateVideo must return 403")
	svc.AssertExpectations(t)
}

// ── GET /courses/:courseId/sections (ListContent) ─────────────────────────────
// [CRITICAL] These tests verify the read path for a course's content tree.
// The frontend curso-editar calls this on page load to render existing sections+videos.

// TestListContent_Owner_Returns200WithNestedTree verifies the owner receives a 200
// with sections nested with their videos. Spec: ERR-1-A (ownership-guarded read → 404 on non-owner).
func TestListContent_Owner_Returns200WithNestedTree(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	content := []service.SectionWithVideosModel{
		{
			Section: service.SectionModel{
				ID:       "sec-1",
				CourseID: courseID,
				Titulo:   "Intro",
				Orden:    0,
			},
			Videos: []service.VideoModel{
				{
					ID:        "vid-1",
					SectionID: "sec-1",
					Titulo:    "Video 1",
					URL:       "https://www.youtube.com/watch?v=abc",
					Proveedor: "youtube",
					Orden:     0,
				},
				{
					ID:        "vid-2",
					SectionID: "sec-1",
					Titulo:    "Video 2",
					URL:       "https://vimeo.com/999",
					Proveedor: "vimeo",
					Orden:     1,
				},
			},
		},
	}
	svc.On("ListContent", mock.Anything, courseID, creadorID).Return(content, nil)

	w := do(engine, http.MethodGet, "/courses/"+courseID+"/sections", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []map[string]any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	assert.Len(t, resp, 1, "response must contain one section")
	videos, ok := resp[0]["videos"].([]any)
	assert.True(t, ok, "section must have a videos array")
	assert.Len(t, videos, 2, "section must have 2 nested videos")
	svc.AssertExpectations(t)
}

// TestListContent_NonOwner_Returns404 verifies that ErrNotOwner on this GET read
// route returns 404 (consistent with GET /courses/:id behavior, ERR-1-A).
func TestListContent_NonOwner_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-other"
	courseID := "course-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("ListContent", mock.Anything, courseID, creadorID).Return([]service.SectionWithVideosModel{}, service.ErrNotOwner)

	w := do(engine, http.MethodGet, "/courses/"+courseID+"/sections", nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "ErrNotOwner on GET sections must return 404")
	svc.AssertExpectations(t)
}

// TestListContent_EmptyCourse_Returns200EmptyArray verifies that a course with no sections
// returns 200 with an empty array (not null, not 404).
func TestListContent_EmptyCourse_Returns200EmptyArray(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	courseID := "course-empty"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	svc.On("ListContent", mock.Anything, courseID, creadorID).Return([]service.SectionWithVideosModel{}, nil)

	w := do(engine, http.MethodGet, "/courses/"+courseID+"/sections", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp []any
	_ = json.NewDecoder(w.Body).Decode(&resp)
	assert.NotNil(t, resp, "empty course must return [] not null")
	assert.Len(t, resp, 0)
	svc.AssertExpectations(t)
}

// TestCreateVideo_HappyPath_Returns201 verifies a valid YouTube video creation returns 201.
// Spec: VID-1-A.
func TestCreateVideo_HappyPath_Returns201(t *testing.T) {
	svc := &mockCourseSvc{}
	creadorID := "creador-1"
	sectionID := "section-1"
	engine := setupEngine(svc, creadorID, []string{"creador"})

	vm := videoModel("video-1", sectionID)
	svc.On("CreateVideo", mock.Anything, creadorID, mock.AnythingOfType("service.VideoCreateRequest")).
		Return(vm, nil)

	w := do(engine, http.MethodPost, "/sections/"+sectionID+"/videos", map[string]any{
		"titulo":    "Test Video",
		"url":       "https://www.youtube.com/watch?v=abc123",
		"proveedor": "youtube",
	})
	assert.Equal(t, http.StatusCreated, w.Code)
	svc.AssertExpectations(t)
}
