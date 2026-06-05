// Package handler_test — HTTP layer tests for the reporting module.
// Strategy: httptest + gin.TestMode, mock service, JWT middleware for authz tests.
// No build tag: runs with standard `make backend-test`.
//
// Covers the full authz matrix from REQ-SEC:
//   - admin → 200 for /reports/global, /reports/courses, /reports/users/:id/progress
//   - supervisor → 200 for /reports/team; 403 for /reports/global, /reports/courses
//   - alumno → 403 for all except own /reports/users/:id/progress
//   - self (callerID == :id) → 200 for /reports/users/:id/progress (in-handler authz)
//   - no JWT → 401 (protected group middleware)
package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/reporting/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock service ───────────────────────────────────────────────────────────────

type mockReportSvc struct {
	GlobalReportFn       func(ctx context.Context) (*service.GlobalReportModel, error)
	CourseReportFn       func(ctx context.Context) ([]service.CourseStatModel, error)
	UserProgressReportFn func(ctx context.Context, userID string) (service.UserProgressModel, error)
	TeamReportFn         func(ctx context.Context, supervisorID string) ([]service.TeamMemberModel, error)
}

func (m *mockReportSvc) GlobalReport(ctx context.Context) (*service.GlobalReportModel, error) {
	if m.GlobalReportFn != nil {
		return m.GlobalReportFn(ctx)
	}
	return &service.GlobalReportModel{}, nil
}
func (m *mockReportSvc) CourseReport(ctx context.Context) ([]service.CourseStatModel, error) {
	if m.CourseReportFn != nil {
		return m.CourseReportFn(ctx)
	}
	return []service.CourseStatModel{}, nil
}
func (m *mockReportSvc) UserProgressReport(ctx context.Context, userID string) (service.UserProgressModel, error) {
	if m.UserProgressReportFn != nil {
		return m.UserProgressReportFn(ctx, userID)
	}
	return service.UserProgressModel{}, nil
}
func (m *mockReportSvc) TeamReport(ctx context.Context, supervisorID string) ([]service.TeamMemberModel, error) {
	if m.TeamReportFn != nil {
		return m.TeamReportFn(ctx, supervisorID)
	}
	return []service.TeamMemberModel{}, nil
}

// ── Engine factories ───────────────────────────────────────────────────────────

// setupEngineWithRoles creates a gin engine with:
//   - protected group (JWT middleware injecting userID and roles)
//   - adminGrp (RequireRole("administrador"))
//   - supervisorGrp (RequireRole("supervisor"))
//   - All 4 reporting routes registered.
func setupEngineWithRoles(svc service.Service, callerID string, roles []string) *gin.Engine {
	r := gin.New()
	protected := r.Group("", injectIdentity(callerID, roles))
	adminGrp := protected.Group("", middleware.RequireRole("administrador"))
	supervisorGrp := protected.Group("", middleware.RequireRole("supervisor"))

	handler.RegisterAdminRoutes(adminGrp, svc)
	handler.RegisterSupervisorRoutes(supervisorGrp, svc)
	handler.RegisterSelfRoutes(protected, svc)
	return r
}

// setupEngineWithJWT creates an engine using REAL JWT middleware (for 401 tests).
func setupEngineWithJWT(svc service.Service) *gin.Engine {
	r := gin.New()
	protected := r.Group("", middleware.JWT("test-secret"))
	adminGrp := protected.Group("", middleware.RequireRole("administrador"))
	supervisorGrp := protected.Group("", middleware.RequireRole("supervisor"))

	handler.RegisterAdminRoutes(adminGrp, svc)
	handler.RegisterSupervisorRoutes(supervisorGrp, svc)
	handler.RegisterSelfRoutes(protected, svc)
	return r
}

// injectIdentity returns a middleware that sets userID and roles in the context
// without requiring a real JWT token (for authz unit tests).
func injectIdentity(userID string, roles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("roles", roles)
		c.Next()
	}
}

// ── 200 shape tests ────────────────────────────────────────────────────────────

func TestGlobal_Admin_Returns200(t *testing.T) {
	svc := &mockReportSvc{
		GlobalReportFn: func(_ context.Context) (*service.GlobalReportModel, error) {
			return &service.GlobalReportModel{
				ActiveUsers: 42,
				CoursesByEstado: []service.EstadoItem{
					{Estado: "aprobado", Total: 5},
				},
			}, nil
		},
	}
	r := setupEngineWithRoles(svc, uuid.New().String(), []string{"administrador"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reports/global", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.EqualValues(t, 42, body["activeUsers"])
}

func TestCourses_Admin_Returns200(t *testing.T) {
	svc := &mockReportSvc{
		CourseReportFn: func(_ context.Context) ([]service.CourseStatModel, error) {
			return []service.CourseStatModel{
				{ID: "c1", Titulo: "Go", Estado: "aprobado", ApprovalRate: 0.8},
			}, nil
		},
	}
	r := setupEngineWithRoles(svc, uuid.New().String(), []string{"administrador"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reports/courses", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body, 1)
	assert.Equal(t, "Go", body[0]["titulo"])
}

func TestTeam_Supervisor_Returns200(t *testing.T) {
	svc := &mockReportSvc{
		TeamReportFn: func(_ context.Context, _ string) ([]service.TeamMemberModel, error) {
			return []service.TeamMemberModel{
				{EmpleadoID: "e1", EmpleadoNombre: "Alice", Enrolled: 2, Completed: 1},
			}, nil
		},
	}
	r := setupEngineWithRoles(svc, uuid.New().String(), []string{"supervisor"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reports/team", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body, 1)
	assert.Equal(t, "Alice", body[0]["empleadoNombre"])
}

func TestUserProgress_SameUser_Returns200(t *testing.T) {
	callerID := uuid.New().String()
	svc := &mockReportSvc{
		UserProgressReportFn: func(_ context.Context, _ string) (service.UserProgressModel, error) {
			return service.UserProgressModel{Enrolled: 3, Completed: 2}, nil
		},
	}
	r := setupEngineWithRoles(svc, callerID, []string{"alumno"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reports/users/"+callerID+"/progress", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.EqualValues(t, 3, body["enrolledCount"])
}

func TestUserProgress_Admin_Returns200ForAnyUser(t *testing.T) {
	adminID := uuid.New().String()
	targetID := uuid.New().String()
	svc := &mockReportSvc{}
	r := setupEngineWithRoles(svc, adminID, []string{"administrador"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reports/users/"+targetID+"/progress", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ── 403 tests ──────────────────────────────────────────────────────────────────

func TestGlobal_NonAdmin_Returns403(t *testing.T) {
	svc := &mockReportSvc{}
	for _, role := range []string{"creador", "alumno", "supervisor"} {
		r := setupEngineWithRoles(svc, uuid.New().String(), []string{role})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/reports/global", http.NoBody)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code, "role %q must get 403 on /reports/global", role)
	}
}

func TestCourses_NonAdmin_Returns403(t *testing.T) {
	svc := &mockReportSvc{}
	for _, role := range []string{"creador", "alumno", "supervisor"} {
		r := setupEngineWithRoles(svc, uuid.New().String(), []string{role})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/reports/courses", http.NoBody)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code, "role %q must get 403 on /reports/courses", role)
	}
}

func TestTeam_NonSupervisor_Returns403(t *testing.T) {
	svc := &mockReportSvc{}
	for _, role := range []string{"creador", "alumno", "administrador"} {
		r := setupEngineWithRoles(svc, uuid.New().String(), []string{role})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/reports/team", http.NoBody)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusForbidden, w.Code, "role %q must get 403 on /reports/team", role)
	}
}

// TestUserProgress_DifferentAlumno_Returns403 verifies that an alumno cannot
// view another user's progress (in-handler admin-or-self authz).
func TestUserProgress_DifferentAlumno_Returns403(t *testing.T) {
	callerID := uuid.New().String()
	targetID := uuid.New().String() // different user
	svc := &mockReportSvc{}
	r := setupEngineWithRoles(svc, callerID, []string{"alumno"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reports/users/"+targetID+"/progress", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "FORBIDDEN", body["code"])
}

// TestUserProgress_Supervisor_DifferentUser_Returns403 verifies that a supervisor
// calling /reports/users/:employeeId/progress gets 403 (they must use /reports/team).
func TestUserProgress_Supervisor_DifferentUser_Returns403(t *testing.T) {
	supervisorID := uuid.New().String()
	employeeID := uuid.New().String()
	svc := &mockReportSvc{}
	r := setupEngineWithRoles(svc, supervisorID, []string{"supervisor"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/reports/users/"+employeeID+"/progress", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── 401 tests (real JWT middleware) ───────────────────────────────────────────

func TestAllEndpoints_NoJWT_Returns401(t *testing.T) {
	svc := &mockReportSvc{}
	r := setupEngineWithJWT(svc)

	endpoints := []string{
		"/reports/global",
		"/reports/courses",
		"/reports/team",
		"/reports/users/" + uuid.New().String() + "/progress",
	}

	for _, path := range endpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "no JWT must return 401 on %s", path)
	}
}
