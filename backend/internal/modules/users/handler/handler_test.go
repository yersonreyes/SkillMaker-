// Package handler — HTTP-layer contract tests for the users module.
//
// Strategy: spin up a real gin.Engine per test (gin.New(), no global state),
// mount the routes WITH the real middleware chain
// (injectIdentity → RequireRole("administrador") for admin routes, injectIdentity for /me),
// wire a mockSvc, fire requests via net/http/httptest.
//
// This suite does NOT re-test service logic — that lives in service/service_test.go.
// It tests: RBAC enforcement, identity resolution, error→status mapping, body binding.
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
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/users/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock service ───────────────────────────────────────────────────────────────

// mockSvc is a local testify/mock implementation of service.Service.
// It is kept local so handler tests don't depend on auth-module compilation.
type mockSvc struct {
	mock.Mock
}

func (m *mockSvc) UpsertFromGoogle(_ context.Context, _ service.GoogleProfile) (*service.UserSummary, error) {
	panic("not called in handler tests")
}

func (m *mockSvc) GetByID(_ context.Context, _ string) (*service.UserSummary, error) {
	panic("not called in handler tests")
}

func (m *mockSvc) List(ctx context.Context, f service.ListFilters, p pagination.Params) (pagination.Page[service.UserDetailModel], error) {
	args := m.Called(ctx, f, p)
	return args.Get(0).(pagination.Page[service.UserDetailModel]), args.Error(1)
}

func (m *mockSvc) GetDetail(ctx context.Context, id string) (*service.UserDetailModel, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.UserDetailModel), args.Error(1)
}

func (m *mockSvc) PatchRoles(ctx context.Context, id string, add, remove []string) (*service.UserDetailModel, error) {
	args := m.Called(ctx, id, add, remove)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.UserDetailModel), args.Error(1)
}

func (m *mockSvc) SetActive(ctx context.Context, id string, active bool) (*service.UserDetailModel, error) {
	args := m.Called(ctx, id, active)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.UserDetailModel), args.Error(1)
}

func (m *mockSvc) CreateSupervision(ctx context.Context, supervisorID, empleadoID string) (*service.SupervisionModel, error) {
	args := m.Called(ctx, supervisorID, empleadoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*service.SupervisionModel), args.Error(1)
}

func (m *mockSvc) ListSupervisions(ctx context.Context) ([]service.SupervisionModel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]service.SupervisionModel), args.Error(1)
}

func (m *mockSvc) DeleteSupervision(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ── Fixtures ───────────────────────────────────────────────────────────────────

// userDetail returns a minimal UserDetailModel suitable for happy-path stubs.
func userDetail(id string) *service.UserDetailModel {
	return &service.UserDetailModel{
		ID:        id,
		Email:     id + "@example.com",
		Nombre:    "Test User",
		Activo:    true,
		Roles:     []string{"alumno"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// adminDetail returns a UserDetailModel with the administrador role.
func adminDetail(id string) *service.UserDetailModel {
	m := userDetail(id)
	m.Roles = []string{"administrador"}
	return m
}

// ── Engine builder ─────────────────────────────────────────────────────────────

// injectIdentity mimics what middleware.JWT injects into the Gin context.
// It writes the same context keys ("userID" / "roles") that jwt.go and rbac.go
// depend on, so middleware.UserIDFrom and middleware.RequireRole work without
// a real JWT token.
func injectIdentity(userID string, roles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("roles", roles)
		c.Next()
	}
}

// setupEngine builds a gin.Engine with the users routes mounted under two groups:
//
//   - admin: injectIdentity → RequireRole("administrador") — mirrors main.go:adminGrp
//   - me:    injectIdentity                                — mirrors main.go:protected
//
// The caller controls identity (userID, roles) to simulate different RBAC scenarios.
func setupEngine(svc service.Service, userID string, roles []string) *gin.Engine {
	r := gin.New()

	identity := injectIdentity(userID, roles)

	admin := r.Group("", identity, middleware.RequireRole("administrador"))
	me := r.Group("", identity)

	handler.Register(admin, me, svc)
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

// respErrorCode decodes the JSON body and returns the "code" field.
func respErrorCode(w *httptest.ResponseRecorder) string {
	var out struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(w.Body).Decode(&out)
	return out.Code
}

// ── RBAC: non-admin callers are rejected with 403 ─────────────────────────────

func TestRBAC_NonAdmin_GET_Users_Returns403(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "user-1", []string{"alumno"})

	w := do(engine, http.MethodGet, "/users", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "FORBIDDEN", respErrorCode(w))
}

func TestRBAC_NonAdmin_GET_UserByID_Returns403(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "user-1", []string{"alumno"})

	w := do(engine, http.MethodGet, "/users/some-uuid", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRBAC_NonAdmin_PATCH_Roles_Returns403(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "user-1", []string{"alumno"})

	w := do(engine, http.MethodPatch, "/users/some-uuid/roles", map[string]interface{}{
		"add": []string{"creador"}, "remove": []string{},
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRBAC_NonAdmin_PATCH_Active_Returns403(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "user-1", []string{"alumno"})

	active := true
	w := do(engine, http.MethodPatch, "/users/some-uuid/active", map[string]interface{}{
		"active": active,
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Admin callers pass the RBAC gate — spot-check with a happy-path List call.
func TestRBAC_Admin_GET_Users_Passes(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	emptyPage := pagination.NewPage([]service.UserDetailModel{}, 0, pagination.Params{Page: 1, Size: 20})
	svc.On("List", mock.Anything, mock.Anything, mock.Anything).Return(emptyPage, nil)

	w := do(engine, http.MethodGet, "/users", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ── GET /users/me — identity resolution ───────────────────────────────────────

// U3-self: /users/me resolves the caller's ID from the Gin context (set by
// injectIdentity ≡ middleware.JWT), NOT from a path param.
func TestGetMe_ResolvesIdentityFromContext_Returns200(t *testing.T) {
	svc := &mockSvc{}
	callerID := "caller-uuid"
	engine := setupEngine(svc, callerID, []string{"alumno"})

	detail := userDetail(callerID)
	svc.On("GetDetail", mock.Anything, callerID).Return(detail, nil)

	w := do(engine, http.MethodGet, "/users/me", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var out map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&out)
	assert.Equal(t, callerID, out["id"])

	// Critical: GetDetail must be called with the caller's ID from context, not a path param.
	svc.AssertCalled(t, "GetDetail", mock.Anything, callerID)
}

// Non-admin can access /me (it is on the "me" group, not the admin group).
func TestGetMe_NonAdmin_CanAccess_Returns200(t *testing.T) {
	svc := &mockSvc{}
	callerID := "regular-user"
	engine := setupEngine(svc, callerID, []string{"alumno"})

	detail := userDetail(callerID)
	svc.On("GetDetail", mock.Anything, callerID).Return(detail, nil)

	w := do(engine, http.MethodGet, "/users/me", nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// U3-not-found: injected userID that maps to no user → 404.
func TestGetMe_UserNotFound_Returns404(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "ghost-id", []string{"alumno"})

	svc.On("GetDetail", mock.Anything, "ghost-id").Return(nil, service.ErrUserNotFound)

	w := do(engine, http.MethodGet, "/users/me", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "USER_NOT_FOUND", respErrorCode(w))
}

// ── GET /users — happy path ────────────────────────────────────────────────────

// U1: admin calling GET /users returns the paginated JSON envelope with 200.
func TestList_HappyPath_Returns200WithEnvelope(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	items := []service.UserDetailModel{
		*userDetail("u1"),
		*adminDetail("u2"),
	}
	params := pagination.Params{Page: 1, Size: 20}
	page := pagination.NewPage(items, 2, params)

	svc.On("List", mock.Anything, mock.Anything, mock.Anything).Return(page, nil)

	w := do(engine, http.MethodGet, "/users", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var out map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&out)

	assert.Equal(t, float64(1), out["page"])
	assert.Equal(t, float64(20), out["size"])
	assert.Equal(t, float64(2), out["total"])
	assert.Equal(t, float64(1), out["totalPages"])

	rawItems, _ := out["items"].([]interface{})
	assert.Len(t, rawItems, 2)
}

// ── Error→status mapping via renderUserError ───────────────────────────────────

// E1: ErrUserNotFound → 404 USER_NOT_FOUND (via GET /users/:id)
func TestGetByID_NotFound_Returns404(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("GetDetail", mock.Anything, "missing-id").Return(nil, service.ErrUserNotFound)

	w := do(engine, http.MethodGet, "/users/missing-id", nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "USER_NOT_FOUND", respErrorCode(w))
}

// E1: ErrLastAdmin → 409 LAST_ADMIN (via PATCH /users/:id/roles)
func TestPatchRoles_LastAdmin_Returns409(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("PatchRoles", mock.Anything, "admin-1", []string{}, []string{"administrador"}).
		Return(nil, service.ErrLastAdmin)

	w := do(engine, http.MethodPatch, "/users/admin-1/roles", map[string]interface{}{
		"add": []string{}, "remove": []string{"administrador"},
	})
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "LAST_ADMIN", respErrorCode(w))
}

// E1: ErrInvalidRole → 400 INVALID_ROLE (via PATCH /users/:id/roles)
func TestPatchRoles_InvalidRole_Returns400(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("PatchRoles", mock.Anything, "user-1", []string{"god"}, []string{}).
		Return(nil, service.ErrInvalidRole)

	w := do(engine, http.MethodPatch, "/users/user-1/roles", map[string]interface{}{
		"add": []string{"god"}, "remove": []string{},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_ROLE", respErrorCode(w))
}

// E1: ErrAddRemoveConflict → 400 ROLE_CONFLICT (via PATCH /users/:id/roles)
func TestPatchRoles_AddRemoveConflict_Returns400(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("PatchRoles", mock.Anything, "user-1", []string{"creador"}, []string{"creador"}).
		Return(nil, service.ErrAddRemoveConflict)

	w := do(engine, http.MethodPatch, "/users/user-1/roles", map[string]interface{}{
		"add": []string{"creador"}, "remove": []string{"creador"},
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "ROLE_CONFLICT", respErrorCode(w))
}

// E1: ErrLastAdmin → 409 LAST_ADMIN (via PATCH /users/:id/active)
func TestSetActive_LastAdmin_Returns409(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("SetActive", mock.Anything, "admin-1", false).Return(nil, service.ErrLastAdmin)

	active := false
	w := do(engine, http.MethodPatch, "/users/admin-1/active", map[string]interface{}{
		"active": active,
	})
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "LAST_ADMIN", respErrorCode(w))
}

// ── Happy paths: PATCH routes ──────────────────────────────────────────────────

// PATCH /users/:id/roles success → 200 with updated detail.
func TestPatchRoles_HappyPath_Returns200(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	updated := userDetail("user-1")
	updated.Roles = []string{"alumno", "creador"}
	svc.On("PatchRoles", mock.Anything, "user-1", []string{"creador"}, []string{}).
		Return(updated, nil)

	w := do(engine, http.MethodPatch, "/users/user-1/roles", map[string]interface{}{
		"add": []string{"creador"}, "remove": []string{},
	})
	assert.Equal(t, http.StatusOK, w.Code)

	var out map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&out)
	assert.Equal(t, "user-1", out["id"])
}

// PATCH /users/:id/active success → 200 with updated detail.
func TestSetActive_HappyPath_Returns200(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	deactivated := userDetail("user-1")
	deactivated.Activo = false
	svc.On("SetActive", mock.Anything, "user-1", false).Return(deactivated, nil)

	active := false
	w := do(engine, http.MethodPatch, "/users/user-1/active", map[string]interface{}{
		"active": active,
	})
	assert.Equal(t, http.StatusOK, w.Code)

	var out map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&out)
	assert.Equal(t, false, out["activo"])
}

// ── Body binding ───────────────────────────────────────────────────────────────

// Malformed JSON on PATCH /users/:id/roles → 400 INVALID_BODY
func TestPatchRoles_MalformedJSON_Returns400(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	req := httptest.NewRequest(http.MethodPatch, "/users/user-1/roles", bytes.NewBufferString("{not valid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_BODY", respErrorCode(w))
}

// Malformed JSON on PATCH /users/:id/active → 400 INVALID_BODY
func TestSetActive_MalformedJSON_Returns400(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	req := httptest.NewRequest(http.MethodPatch, "/users/user-1/active", bytes.NewBufferString("{not valid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_BODY", respErrorCode(w))
}

// Missing required "active" field (binding:"required") → 400
func TestSetActive_MissingActiveField_Returns400(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	// Empty JSON object — "active" is absent; binding:"required" rejects it
	w := do(engine, http.MethodPatch, "/users/user-1/active", map[string]interface{}{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Supervision fixtures ────────────────────────────────────────────────────────

// Valid UUIDs used in supervision tests (binding:"uuid" requires proper UUID format).
const (
	testSupUUID  = "11111111-1111-1111-1111-111111111111"
	testEmpUUID  = "22222222-2222-2222-2222-222222222222"
	testEmp2UUID = "33333333-3333-3333-3333-333333333333"
	testSvUUID   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	testSv2UUID  = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
)

func supervisionModel(id, supID, empID string) *service.SupervisionModel {
	return &service.SupervisionModel{
		ID:           id,
		SupervisorID: supID,
		EmpleadoID:   empID,
		CreadoEn:     time.Now(),
	}
}

// ── GET /supervisions — RBAC ──────────────────────────────────────────────────

// V1-forbidden: non-admin caller is rejected with 403.
func TestRBAC_NonAdmin_GET_Supervisions_Returns403(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "user-1", []string{"supervisor"})

	w := do(engine, http.MethodGet, "/supervisions", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "FORBIDDEN", respErrorCode(w))
}

// V2-forbidden: non-admin caller on POST /supervisions → 403.
func TestRBAC_NonAdmin_POST_Supervisions_Returns403(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "user-1", []string{"creador"})

	w := do(engine, http.MethodPost, "/supervisions", map[string]interface{}{
		"supervisorId": "sup-id", "empleadoId": "emp-id",
	})
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// V3-forbidden: non-admin caller on DELETE /supervisions/:id → 403.
func TestRBAC_NonAdmin_DELETE_Supervision_Returns403(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "user-1", []string{"alumno"})

	w := do(engine, http.MethodDelete, "/supervisions/sv-1", nil)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ── GET /supervisions — happy path ────────────────────────────────────────────

// V1-list: admin calling GET /supervisions returns 200 with array.
func TestListSupervisions_HappyPath_Returns200(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svs := []service.SupervisionModel{
		*supervisionModel(testSvUUID, testSupUUID, testEmpUUID),
		*supervisionModel(testSv2UUID, testSupUUID, testEmp2UUID),
	}
	svc.On("ListSupervisions", mock.Anything).Return(svs, nil)

	w := do(engine, http.MethodGet, "/supervisions", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var out []map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&out)
	assert.Len(t, out, 2)
	assert.Equal(t, testSvUUID, out[0]["id"])
}

// Empty list → 200 with empty array (never null).
func TestListSupervisions_Empty_Returns200EmptyArray(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("ListSupervisions", mock.Anything).Return([]service.SupervisionModel{}, nil)

	w := do(engine, http.MethodGet, "/supervisions", nil)
	assert.Equal(t, http.StatusOK, w.Code)

	var out []interface{}
	_ = json.NewDecoder(w.Body).Decode(&out)
	assert.Len(t, out, 0)
}

// ── POST /supervisions — happy path + error mapping ───────────────────────────

// V2-create: valid body → 201 with created relation.
func TestCreateSupervision_HappyPath_Returns201(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	sv := supervisionModel(testSvUUID, testSupUUID, testEmpUUID)
	svc.On("CreateSupervision", mock.Anything, testSupUUID, testEmpUUID).Return(sv, nil)

	w := do(engine, http.MethodPost, "/supervisions", map[string]interface{}{
		"supervisorId": testSupUUID,
		"empleadoId":   testEmpUUID,
	})
	assert.Equal(t, http.StatusCreated, w.Code)

	var out map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&out)
	assert.Equal(t, testSvUUID, out["id"])
	assert.Equal(t, testSupUUID, out["supervisorId"])
	assert.Equal(t, testEmpUUID, out["empleadoId"])
}

// V2-self: service returns ErrSelfSupervision → 400 SELF_SUPERVISION.
// Uses same UUID for both supervisor and employee (passes binding:uuid, fails in service).
func TestCreateSupervision_SelfSupervision_Returns400(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("CreateSupervision", mock.Anything, testSupUUID, testSupUUID).Return(nil, service.ErrSelfSupervision)

	w := do(engine, http.MethodPost, "/supervisions", map[string]interface{}{
		"supervisorId": testSupUUID,
		"empleadoId":   testSupUUID,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "SELF_SUPERVISION", respErrorCode(w))
}

// V2-duplicate: service returns ErrSupervisionExists → 409 SUPERVISION_EXISTS.
func TestCreateSupervision_Duplicate_Returns409(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("CreateSupervision", mock.Anything, testSupUUID, testEmpUUID).Return(nil, service.ErrSupervisionExists)

	w := do(engine, http.MethodPost, "/supervisions", map[string]interface{}{
		"supervisorId": testSupUUID,
		"empleadoId":   testEmpUUID,
	})
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, "SUPERVISION_EXISTS", respErrorCode(w))
}

// V2-missing-user: service returns ErrUserNotFound → 404 USER_NOT_FOUND.
func TestCreateSupervision_UserNotFound_Returns404(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	// testEmp2UUID as a "ghost" supervisor
	svc.On("CreateSupervision", mock.Anything, testEmp2UUID, testEmpUUID).Return(nil, service.ErrUserNotFound)

	w := do(engine, http.MethodPost, "/supervisions", map[string]interface{}{
		"supervisorId": testEmp2UUID,
		"empleadoId":   testEmpUUID,
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "USER_NOT_FOUND", respErrorCode(w))
}

// Malformed JSON on POST /supervisions → 400 INVALID_BODY.
func TestCreateSupervision_MalformedJSON_Returns400(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	req := httptest.NewRequest(http.MethodPost, "/supervisions", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_BODY", respErrorCode(w))
}

// ── DELETE /supervisions/:id — happy path + error mapping ─────────────────────

// V3-delete: relation exists → 204 no content.
func TestDeleteSupervision_HappyPath_Returns204(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("DeleteSupervision", mock.Anything, testSvUUID).Return(nil)

	w := do(engine, http.MethodDelete, "/supervisions/"+testSvUUID, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// V3-not-found: service returns ErrSupervisionNotFound → 404 SUPERVISION_NOT_FOUND.
func TestDeleteSupervision_NotFound_Returns404(t *testing.T) {
	svc := &mockSvc{}
	engine := setupEngine(svc, "admin-1", []string{"administrador"})

	svc.On("DeleteSupervision", mock.Anything, testSv2UUID).Return(service.ErrSupervisionNotFound)

	w := do(engine, http.MethodDelete, "/supervisions/"+testSv2UUID, nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "SUPERVISION_NOT_FOUND", respErrorCode(w))
}
