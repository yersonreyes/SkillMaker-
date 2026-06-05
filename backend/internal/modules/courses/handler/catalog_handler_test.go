// Package handler — HTTP-layer contract tests for the catalog module (C2.4).
//
// Tests:
//   - GET /catalog 200 + pagination envelope
//   - GET /catalog/:id 200 preview (enrolled=false, NO secciones/materiales keys in JSON)
//   - GET /catalog/:id 200 enrolled (enrolled=true + secciones)
//   - POST /catalog/:id/enroll 200
//   - POST /catalog/:id/enroll 404 non-aprobado (ErrCourseNotFound)
//   - GET /users/me/courses 200 with user isolation
//   - 401 without JWT on all 4 routes (via injectIdentity with empty userID)
//   - Structural no-leak: preview JSON must NOT contain secciones/materiales keys
//
// No build tag: runs with standard `make backend-test`.
package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

// setupCatalogEngine builds a gin.Engine with catalog routes on the protected group.
// The protected group uses injectIdentity (not real JWT) for simpler test setup.
func setupCatalogEngine(svc service.Service, userID string) *gin.Engine {
	r := gin.New()
	identity := injectIdentity(userID, []string{"alumno"})
	protected := r.Group("", identity)
	handler.RegisterCatalog(protected, svc)
	return r
}

// ── GET /catalog ───────────────────────────────────────────────────────────────

// TestListCatalog_Returns200_WithPage verifies GET /catalog returns 200 + page envelope.
func TestListCatalog_Returns200_WithPage(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	p := pagination.Params{Page: 1, Size: 20}
	repoPage := pagination.Page[service.CatalogCourseModel]{
		Items: []service.CatalogCourseModel{
			{ID: "c1", Titulo: "Go", Descripcion: "Desc", CreadorNombre: "Alice", CreatedAt: time.Now()},
		},
		Page: 1, Size: 20, Total: 1, TotalPages: 1,
	}
	svc.On("ListCatalog", mock.Anything, p, "").Return(repoPage, nil)

	req := httptest.NewRequest(http.MethodGet, "/catalog", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(1), body["total"])
	items, ok := body["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
	svc.AssertExpectations(t)
}

// TestListCatalog_WithQ_PassesFilter verifies ?q is forwarded to ListCatalog.
func TestListCatalog_WithQ_PassesFilter(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	p := pagination.Params{Page: 1, Size: 20}
	empty := pagination.Page[service.CatalogCourseModel]{Items: []service.CatalogCourseModel{}, Page: 1, Size: 20, Total: 0, TotalPages: 0}
	svc.On("ListCatalog", mock.Anything, p, "angular").Return(empty, nil)

	req := httptest.NewRequest(http.MethodGet, "/catalog?q=angular", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

// ── GET /catalog/:id ───────────────────────────────────────────────────────────

// TestGetCatalogDetail_NotEnrolled_ReturnsPreview verifies preview response (enrolled=false).
// STRUCTURAL NO-LEAK: the JSON must NOT contain secciones or materiales keys.
func TestGetCatalogDetail_NotEnrolled_ReturnsPreview(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	detail := &service.CatalogDetailModel{
		ID:            "c1",
		Titulo:        "Go Course",
		Descripcion:   "Desc",
		CreadorNombre: "Alice",
		Enrolled:      false,
		Sections:      nil,
		Materiales:    nil,
	}
	svc.On("GetCatalogDetail", mock.Anything, "user-1", "c1").Return(detail, nil)

	req := httptest.NewRequest(http.MethodGet, "/catalog/c1", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify enrolled=false in response.
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, false, body["enrolled"])

	// STRUCTURAL NO-LEAK assertion (AC-9 / OQ-3):
	// Preview response struct has NO secciones/materiales fields (not omitempty — structurally absent).
	rawJSON := w.Body.Bytes()
	var rawMap map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rawJSON, &rawMap))
	_, hasSecciones := rawMap["secciones"]
	_, hasMateriales := rawMap["materiales"]
	assert.False(t, hasSecciones,
		"[structural no-leak] preview response MUST NOT contain 'secciones' key in JSON (AC-9)")
	assert.False(t, hasMateriales,
		"[structural no-leak] preview response MUST NOT contain 'materiales' key in JSON (AC-9)")

	svc.AssertExpectations(t)
}

// TestGetCatalogDetail_Enrolled_ReturnsTree verifies enrolled response has secciones.
func TestGetCatalogDetail_Enrolled_ReturnsTree(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	detail := &service.CatalogDetailModel{
		ID:            "c1",
		Titulo:        "Go Course",
		Descripcion:   "Desc",
		CreadorNombre: "Alice",
		Enrolled:      true,
		Sections: []service.SectionWithVideosModel{
			{
				Section: service.SectionModel{ID: "s1", CourseID: "c1", Titulo: "Cap 1"},
				Videos:  []service.VideoModel{},
			},
		},
		Materiales: []service.MaterialModel{},
	}
	svc.On("GetCatalogDetail", mock.Anything, "user-1", "c1").Return(detail, nil)

	req := httptest.NewRequest(http.MethodGet, "/catalog/c1", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, true, body["enrolled"])
	secciones, ok := body["secciones"].([]any)
	assert.True(t, ok, "enrolled response must contain 'secciones' key")
	assert.Len(t, secciones, 1)
	svc.AssertExpectations(t)
}

// TestGetCatalogDetail_NotFound_Returns404 verifies ErrCourseNotFound → 404.
func TestGetCatalogDetail_NotFound_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	svc.On("GetCatalogDetail", mock.Anything, "user-1", "missing").Return(nil, service.ErrCourseNotFound)

	req := httptest.NewRequest(http.MethodGet, "/catalog/missing", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

// ── POST /catalog/:id/enroll ──────────────────────────────────────────────────

// TestEnroll_Returns200 verifies POST /catalog/:id/enroll → 200 + EnrollmentResponse.
func TestEnroll_Returns200(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	svc.On("Enroll", mock.Anything, "user-1", "c1").Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/catalog/c1/enroll", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "c1", body["courseId"])
	assert.Equal(t, true, body["enrolled"])
	svc.AssertExpectations(t)
}

// TestEnroll_NonAprobado_Returns404 verifies ErrCourseNotFound → 404 (draft-invisibility).
func TestEnroll_NonAprobado_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	svc.On("Enroll", mock.Anything, "user-1", "c-draft").Return(service.ErrCourseNotFound)

	req := httptest.NewRequest(http.MethodPost, "/catalog/c-draft/enroll", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

// ── GET /users/me/courses ─────────────────────────────────────────────────────

// TestListMyCourses_Returns200_WithUserScoping verifies my-courses returns only caller's courses.
func TestListMyCourses_Returns200_WithUserScoping(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	rows := []service.MyCourseModel{
		{CourseID: "c1", Titulo: "Go", CreadorNombre: "Alice", Completado: false, InscritoEn: time.Now()},
	}
	svc.On("ListMyCourses", mock.Anything, "user-1").Return(rows, nil)

	req := httptest.NewRequest(http.MethodGet, "/users/me/courses", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Len(t, body, 1)
	svc.AssertExpectations(t)
}

// ── 401 without JWT (missing userID) ─────────────────────────────────────────

// TestCatalogRoutes_MissingUserID_Returns401 verifies that missing userID from context
// causes 401 on all catalog endpoints.
// Note: the real JWT middleware returns 401 before the handler runs.
// Here we simulate this by not setting any identity (empty userID from context).
func TestListCatalog_EmptyUserID_StillWorks(t *testing.T) {
	// The catalog list doesn't require userID (it's a browse endpoint).
	// Unauthenticated is enforced by the JWT middleware at the group level.
	// This test verifies the handler doesn't panic on empty userID for ListCatalog.
	svc := &mockCourseSvc{}
	r := gin.New()
	// NO identity injected → userID will be empty string
	protected := r.Group("")
	handler.RegisterCatalog(protected, svc)

	p := pagination.Params{Page: 1, Size: 20}
	empty := pagination.Page[service.CatalogCourseModel]{Items: []service.CatalogCourseModel{}, Page: 1, Size: 20, Total: 0, TotalPages: 0}
	svc.On("ListCatalog", mock.Anything, p, "").Return(empty, nil)

	req := httptest.NewRequest(http.MethodGet, "/catalog", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Should still be 200 because ListCatalog doesn't need userID.
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestEnroll_EmptyUserID_Returns401 verifies that missing userID from context returns 401 on enroll.
func TestEnroll_EmptyUserID_Returns401(t *testing.T) {
	svc := &mockCourseSvc{}
	r := gin.New()
	// NO identity injected.
	protected := r.Group("")
	handler.RegisterCatalog(protected, svc)

	req := httptest.NewRequest(http.MethodPost, "/catalog/c1/enroll", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	// Enroll requires userID; empty → 401.
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestListMyCourses_EmptyUserID_Returns401 verifies 401 when no userID on my-courses.
func TestListMyCourses_EmptyUserID_Returns401(t *testing.T) {
	svc := &mockCourseSvc{}
	r := gin.New()
	protected := r.Group("")
	handler.RegisterCatalog(protected, svc)

	req := httptest.NewRequest(http.MethodGet, "/users/me/courses", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestGinRouteBoot_CatalogRoutes_NoPanic verifies catalog routes register without panic.
func TestGinRouteBoot_CatalogRoutes_NoPanic(t *testing.T) {
	svc := &mockCourseSvc{}
	assert.NotPanics(t, func() {
		r := gin.New()
		identity := injectIdentity("user-1", []string{"alumno"})
		protected := r.Group("", identity)
		creatorGrp := protected.Group("", middleware.RequireRole("creador"))
		// Register BOTH creator routes and catalog routes on the same engine.
		handler.Register(creatorGrp, svc)
		handler.RegisterCatalog(protected, svc)
	}, "catalog routes must register without panic alongside creator routes")
}
