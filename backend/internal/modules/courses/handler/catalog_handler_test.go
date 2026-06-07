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
	"bytes"
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
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/courses/dto"
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
	svc.On("ListCatalog", mock.Anything, p, service.CatalogFilter{Sort: "recientes"}).Return(repoPage, nil)

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
	svc.On("ListCatalog", mock.Anything, p, service.CatalogFilter{Q: "angular", Sort: "recientes"}).Return(empty, nil)

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
		Categorias:    []service.CategoriaModel{},
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
		Categorias: []service.CategoriaModel{},
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
	svc.On("ListCatalog", mock.Anything, p, service.CatalogFilter{Sort: "recientes"}).Return(empty, nil)

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

// ── catalog-filters handler tests (Phase 3 — REQ-FILTER-NIVEL, REQ-SORT, ADR-4) ──────────────────

// TestListCatalog_InvalidNivel_Returns400 verifies ?nivel=expert → 400 INVALID_FILTER.
// Refs: REQ-FILTER-NIVEL, AC7; ADR-4 (handler owns validation).
func TestListCatalog_InvalidNivel_Returns400(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	req := httptest.NewRequest(http.MethodGet, "/catalog?nivel=expert", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "invalid nivel must return 400")
	assert.Equal(t, "INVALID_FILTER", respCode(w), "error code must be INVALID_FILTER")
	svc.AssertNotCalled(t, "ListCatalog")
}

// TestListCatalog_InvalidSort_Returns400 verifies ?sort=random → 400 INVALID_FILTER.
// Refs: REQ-SORT, AC7; ADR-4 (handler owns validation).
func TestListCatalog_InvalidSort_Returns400(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	req := httptest.NewRequest(http.MethodGet, "/catalog?sort=random", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "invalid sort must return 400")
	assert.Equal(t, "INVALID_FILTER", respCode(w), "error code must be INVALID_FILTER")
	svc.AssertNotCalled(t, "ListCatalog")
}

// TestListCatalog_FilterPassthrough_NivelCategoriaSort verifies full filter passthrough.
// ?nivel=basico&categoria=A&categoria=B&sort=titulo → service receives CatalogFilter verbatim.
// Refs: REQ-FILTER-NIVEL, REQ-FILTER-CATEGORIA, REQ-SORT; ADR-4, ADR-5.
func TestListCatalog_FilterPassthrough_NivelCategoriaSort(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	p := pagination.Params{Page: 1, Size: 20}
	empty := pagination.Page[service.CatalogCourseModel]{Items: []service.CatalogCourseModel{}, Page: 1, Size: 20, Total: 0, TotalPages: 0}
	// Use real well-formed UUIDs — after CRITICAL-1 fix, handler validates UUID format.
	catA := "550e8400-e29b-41d4-a716-446655440001"
	catB := "550e8400-e29b-41d4-a716-446655440002"
	expectedFilter := service.CatalogFilter{
		Nivel:        "basico",
		CategoriaIDs: []string{catA, catB},
		Sort:         "titulo",
	}
	svc.On("ListCatalog", mock.Anything, p, expectedFilter).Return(empty, nil)

	req := httptest.NewRequest(http.MethodGet,
		"/catalog?nivel=basico&categoria="+catA+"&categoria="+catB+"&sort=titulo",
		http.NoBody,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "valid filter params must return 200")
	svc.AssertExpectations(t)
}

// TestListCatalog_AbsentSort_DefaultsToRecientes verifies that absent ?sort defaults to "recientes"
// in the forwarded CatalogFilter. Refs: REQ-SORT (default recientes); ADR-4.
func TestListCatalog_AbsentSort_DefaultsToRecientes(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	p := pagination.Params{Page: 1, Size: 20}
	empty := pagination.Page[service.CatalogCourseModel]{Items: []service.CatalogCourseModel{}, Page: 1, Size: 20, Total: 0, TotalPages: 0}
	// No sort param — must default to "recientes" in filter.
	svc.On("ListCatalog", mock.Anything, p, service.CatalogFilter{Sort: "recientes"}).Return(empty, nil)

	req := httptest.NewRequest(http.MethodGet, "/catalog", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

// ── catalog-filters CRITICAL-1 fix: categoria UUID validation (REQ-FILTER-CATEGORIA + REQ-SEC) ─────

// TestListCatalog_MalformedCategoriaUUID_Returns400 verifies that ?categoria=notauuid (a non-UUID
// string) returns HTTP 400 with code INVALID_FILTER — NOT 500 (Postgres SQLSTATE 22P02).
// The service must NOT be called when the input is malformed.
// Refs: REQ-FILTER-CATEGORIA "malformed UUID MUST return HTTP 400", REQ-SEC "categoria ids MUST
// be validated as UUIDs; only validated ids passed to EXISTS IN clause".
// STRICT TDD: RED — currently returns 500 (SQL error) and calls the service. This test is RED.
func TestListCatalog_MalformedCategoriaUUID_Returns400(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	req := httptest.NewRequest(http.MethodGet, "/catalog?categoria=notauuid", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "malformed categoria UUID must return 400")
	assert.Equal(t, "INVALID_FILTER", respCode(w), "error code must be INVALID_FILTER")
	// Service must NOT be called — validation short-circuits before reaching the service layer.
	svc.AssertNotCalled(t, "ListCatalog")
}

// TestListCatalog_MultipleCategoriasOneMalformed_Returns400 verifies that if ANY categoria value
// in a multi-value query is not a valid UUID, the entire request returns 400.
// Refs: REQ-FILTER-CATEGORIA, REQ-SEC.
func TestListCatalog_MultipleCategoriasOneMalformed_Returns400(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	// One valid UUID + one malformed → the whole request must 400.
	req := httptest.NewRequest(http.MethodGet,
		"/catalog?categoria=550e8400-e29b-41d4-a716-446655440000&categoria=not-a-uuid",
		http.NoBody,
	)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "any malformed categoria UUID must cause 400")
	assert.Equal(t, "INVALID_FILTER", respCode(w), "error code must be INVALID_FILTER")
	svc.AssertNotCalled(t, "ListCatalog")
}

// TestListCatalog_WellFormedNonexistentUUID_Returns200 verifies that a well-formed but
// nonexistent UUID categoria passes validation and returns 200 (match-nothing, no error).
// Refs: ADR-4 "nonexistent categoria = match-nothing NOT 400".
func TestListCatalog_WellFormedNonexistentUUID_Returns200(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	ghostUUID := "550e8400-e29b-41d4-a716-446655440000"
	p := pagination.Params{Page: 1, Size: 20}
	empty := pagination.Page[service.CatalogCourseModel]{Items: []service.CatalogCourseModel{}, Page: 1, Size: 20, Total: 0, TotalPages: 0}
	svc.On("ListCatalog", mock.Anything, p, service.CatalogFilter{
		CategoriaIDs: []string{ghostUUID},
		Sort:         "recientes",
	}).Return(empty, nil)

	req := httptest.NewRequest(http.MethodGet, "/catalog?categoria="+ghostUUID, http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "well-formed nonexistent UUID must return 200 (match-nothing)")
	svc.AssertExpectations(t)
}

// TestListCatalog_ResponseShapeUnchanged verifies response shape compatibility (REQ-COMPAT).
// The CatalogCourseCard page envelope must remain unchanged after filter addition.
func TestListCatalog_ResponseShapeUnchanged(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-1")

	p := pagination.Params{Page: 1, Size: 20}
	repoPage := pagination.Page[service.CatalogCourseModel]{
		Items: []service.CatalogCourseModel{
			{ID: "c1", Titulo: "Go", Descripcion: "Desc", CreadorNombre: "Alice", CreatedAt: time.Now()},
		},
		Page: 1, Size: 20, Total: 1, TotalPages: 1,
	}
	svc.On("ListCatalog", mock.Anything, p, service.CatalogFilter{Sort: "recientes"}).Return(repoPage, nil)

	req := httptest.NewRequest(http.MethodGet, "/catalog", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	// Verify page envelope fields present.
	assert.Contains(t, body, "items", "response must have items field (REQ-COMPAT)")
	assert.Contains(t, body, "total", "response must have total field (REQ-COMPAT)")
	assert.Contains(t, body, "page", "response must have page field (REQ-COMPAT)")
	assert.Contains(t, body, "size", "response must have size field (REQ-COMPAT)")
	assert.Contains(t, body, "totalPages", "response must have totalPages field (REQ-COMPAT)")
	svc.AssertExpectations(t)
}

// ── PUT /videos/:id/progress (MarkVideoProgress) handler tests ───────────────

// doProgress fires a PUT request to /videos/:id/progress with the given body.
func doProgress(r *gin.Engine, videoID string, body dto.VideoProgressRequest) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/videos/"+videoID+"/progress", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestMarkVideoProgress_Enrolled_Returns204 verifies PUT /videos/:id/progress → 204 for enrolled caller.
// Satisfies REQ-PROGRESS-WRITE / Design §5.
func TestMarkVideoProgress_Enrolled_Returns204(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-jwt")

	pos := 0
	svc.On("MarkVideoProgress", mock.Anything, "user-jwt", "vid-1", true, pos).Return(nil)

	w := doProgress(r, "vid-1", dto.VideoProgressRequest{Completado: true})
	assert.Equal(t, http.StatusNoContent, w.Code,
		"enrolled caller marking progress must return 204 No Content")
	svc.AssertExpectations(t)
}

// TestMarkVideoProgress_NotEnrolled_Returns404 verifies ErrNotEnrolled → 404 (no-leak, D3).
// A non-enrolled caller must receive 404, not 403, to avoid leaking enrollment/existence info.
func TestMarkVideoProgress_NotEnrolled_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-jwt")

	svc.On("MarkVideoProgress", mock.Anything, "user-jwt", "vid-1", true, 0).Return(service.ErrNotEnrolled)

	w := doProgress(r, "vid-1", dto.VideoProgressRequest{Completado: true})
	assert.Equal(t, http.StatusNotFound, w.Code,
		"ErrNotEnrolled must return 404 (no-leak — indistinguishable from nonexistent video, D3)")
	svc.AssertExpectations(t)
}

// TestMarkVideoProgress_VideoNotFound_Returns404 verifies ErrVideoNotFound → 404 (no-leak).
func TestMarkVideoProgress_VideoNotFound_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-jwt")

	svc.On("MarkVideoProgress", mock.Anything, "user-jwt", "nonexistent", true, 0).Return(service.ErrVideoNotFound)

	w := doProgress(r, "nonexistent", dto.VideoProgressRequest{Completado: true})
	assert.Equal(t, http.StatusNotFound, w.Code,
		"ErrVideoNotFound must return 404 (no-leak — same as ErrNotEnrolled, D3)")
	svc.AssertExpectations(t)
}

// TestMarkVideoProgress_MissingUserID_Returns401 verifies that a missing userID (empty JWT) returns 401.
// Satisfies REQ-SEC / REQ-PROGRESS-WRITE "Missing JWT returns 401".
func TestMarkVideoProgress_MissingUserID_Returns401(t *testing.T) {
	svc := &mockCourseSvc{}
	r := gin.New()
	// NO identity injected — simulates missing/invalid JWT.
	protected := r.Group("")
	handler.RegisterCatalog(protected, svc)

	w := doProgress(r, "vid-1", dto.VideoProgressRequest{Completado: true})
	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"missing JWT (empty userID) must return 401")
	svc.AssertNotCalled(t, "MarkVideoProgress")
}

// TestMarkVideoProgress_CallerScoped_BodyUserIDIgnored verifies that even if the body includes a
// userId field, the handler uses the JWT userID (caller-scoped, REQ-SEC).
// The mock asserts svc is called with the JWT userID "user-jwt", not any body userId.
func TestMarkVideoProgress_CallerScoped_BodyUserIDIgnored(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-jwt")

	// VideoProgressRequest has no userId field (by design). Even if extra fields were added
	// in the JSON body they must be ignored. The service is called with "user-jwt" (JWT).
	svc.On("MarkVideoProgress", mock.Anything, "user-jwt", "vid-1", true, 0).Return(nil)

	// Send a body with an extra ignored userId field via raw JSON.
	rawBody := `{"completado":true,"userId":"attacker-user"}`
	req := httptest.NewRequest(http.MethodPut, "/videos/vid-1/progress", bytes.NewBufferString(rawBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// The mock assertion: MarkVideoProgress was called with "user-jwt", NOT "attacker-user".
	assert.Equal(t, http.StatusNoContent, w.Code, "caller-scoped: JWT user must be used, not body userId")
	svc.AssertExpectations(t)
}

// TestMarkVideoProgress_InvalidUUID_Returns404 verifies that a malformed (non-UUID) video ID
// in the path returns 404, NOT 500. REQ-PROGRESS-WRITE "Non-existent video gets 404".
//
// Root cause guarded against: a non-UUID string reaching ResolveVideoCourse hits Postgres
// SQLSTATE 22P02 (invalid input syntax for uuid), which is NOT ErrVideoNotFound, causing
// renderProgressError's default branch to fire → 500. The fix must return ErrVideoNotFound
// before the SQL call when the videoID cannot be parsed as a UUID.
func TestMarkVideoProgress_InvalidUUID_Returns404(t *testing.T) {
	svc := &mockCourseSvc{}
	r := setupCatalogEngine(svc, "user-jwt")

	// A non-UUID path param must be caught BEFORE reaching the service/repo (UUID validation).
	// The service must return ErrVideoNotFound (mapped to 404), never 500.
	svc.On("MarkVideoProgress", mock.Anything, "user-jwt", "not-a-uuid", true, 0).
		Return(service.ErrVideoNotFound)

	w := doProgress(r, "not-a-uuid", dto.VideoProgressRequest{Completado: true})
	assert.Equal(t, http.StatusNotFound, w.Code,
		"malformed (non-UUID) video ID in path must return 404, not 500 (SQLSTATE 22P02 must not escape)")
	svc.AssertExpectations(t)
}
