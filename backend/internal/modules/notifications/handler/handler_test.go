// Package handler — HTTP-layer tests for the notifications module.
// Strategy: httptest + real gin.Engine, mock service, JWT middleware injected via context.
// No build tag: runs with standard `make backend-test`.
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

	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/notifications/service"
	"github.com/yersonreyes/SkillMaker-/backend/internal/platform/pagination"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── mock service ─────────────────────────────────────────────────────────────

type mockNotifSvc struct {
	NotifyFn      func(ctx context.Context, userID, tipo, titulo, cuerpo, refID string) error
	ListMineFn    func(ctx context.Context, userID string, p pagination.Params) (pagination.Page[service.NotificationModel], error)
	UnreadCountFn func(ctx context.Context, userID string) (int64, error)
	MarkReadFn    func(ctx context.Context, id, userID string) error
	MarkAllReadFn func(ctx context.Context, userID string) error
}

func (m *mockNotifSvc) Notify(ctx context.Context, userID, tipo, titulo, cuerpo, refID string) error {
	if m.NotifyFn != nil {
		return m.NotifyFn(ctx, userID, tipo, titulo, cuerpo, refID)
	}
	return nil
}

func (m *mockNotifSvc) ListMine(ctx context.Context, userID string, p pagination.Params) (pagination.Page[service.NotificationModel], error) {
	if m.ListMineFn != nil {
		return m.ListMineFn(ctx, userID, p)
	}
	return pagination.Page[service.NotificationModel]{}, nil
}

func (m *mockNotifSvc) UnreadCount(ctx context.Context, userID string) (int64, error) {
	if m.UnreadCountFn != nil {
		return m.UnreadCountFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockNotifSvc) MarkRead(ctx context.Context, id, userID string) error {
	if m.MarkReadFn != nil {
		return m.MarkReadFn(ctx, id, userID)
	}
	return nil
}

func (m *mockNotifSvc) MarkAllRead(ctx context.Context, userID string) error {
	if m.MarkAllReadFn != nil {
		return m.MarkAllReadFn(ctx, userID)
	}
	return nil
}

// ── test helpers ──────────────────────────────────────────────────────────────

// setupRouter builds a gin.Engine with the JWT middleware simulated by
// injecting the userID directly into the context (no real JWT needed for unit tests).
func setupRouter(svc service.Service, userID string) *gin.Engine {
	r := gin.New()
	api := r.Group("/api")

	// Simulate JWT middleware by setting userID in context.
	// Uses the same key string as middleware.UserIDFrom reads ("userID").
	protected := api.Group("", func(c *gin.Context) {
		if userID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED"})
			return
		}
		c.Set("userID", userID)
		c.Next()
	})

	handler.Register(protected, svc)
	return r
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestListMine_Returns200WithItems verifies that the list endpoint returns
// items and the pagination envelope with HTTP 200.
func TestListMine_Returns200WithItems(t *testing.T) {
	userID := uuid.New().String()
	svc := &mockNotifSvc{
		ListMineFn: func(_ context.Context, uid string, _ pagination.Params) (pagination.Page[service.NotificationModel], error) {
			assert.Equal(t, userID, uid)
			items := []service.NotificationModel{
				{ID: "n1", Tipo: "curso_aprobado", Titulo: "T1", Cuerpo: "C1"},
			}
			return pagination.NewPage(items, 1, pagination.Params{Page: 1, Size: 20}), nil
		},
	}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/notifications/me", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	items, ok := body["items"].([]interface{})
	assert.True(t, ok, "response must have items array")
	assert.Len(t, items, 1)
}

// TestUnreadCount_Returns200WithCount verifies the unread-count endpoint.
func TestUnreadCount_Returns200WithCount(t *testing.T) {
	userID := uuid.New().String()
	svc := &mockNotifSvc{
		UnreadCountFn: func(_ context.Context, _ string) (int64, error) {
			return 3, nil
		},
	}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/notifications/me/unread-count", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, float64(3), body["unread"])
}

// TestMarkRead_Happy_Returns200 verifies the happy path for PATCH /:id/read.
func TestMarkRead_Happy_Returns200(t *testing.T) {
	userID := uuid.New().String()
	notifID := uuid.New().String()

	svc := &mockNotifSvc{
		MarkReadFn: func(_ context.Context, id, uid string) error {
			assert.Equal(t, notifID, id)
			assert.Equal(t, userID, uid)
			return nil
		},
	}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/notifications/"+notifID+"/read", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestMarkRead_MalformedUUID_Returns404 verifies that a malformed :id returns 404, not 500.
// This is the uuid.Parse→404 security invariant (REQ-SEC).
func TestMarkRead_MalformedUUID_Returns404(t *testing.T) {
	userID := uuid.New().String()
	svc := &mockNotifSvc{}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/notifications/not-a-valid-uuid/read", http.NoBody)
	r.ServeHTTP(w, req)

	// MUST be 404 (NOT 500, NOT 400).
	assert.Equal(t, http.StatusNotFound, w.Code,
		"malformed uuid must return 404, never 500 (REQ-SEC uuid.Parse guard)")
}

// TestMarkRead_ForeignID_Returns404 verifies that a nonexistent or cross-user id returns 404.
func TestMarkRead_ForeignID_Returns404(t *testing.T) {
	userID := uuid.New().String()
	svc := &mockNotifSvc{
		MarkReadFn: func(_ context.Context, _, _ string) error {
			return service.ErrNotFound
		},
	}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/notifications/"+uuid.New().String()+"/read", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestMarkRead_NoJWT_Returns401 verifies that missing JWT returns 401.
func TestMarkRead_NoJWT_Returns401(t *testing.T) {
	svc := &mockNotifSvc{}

	r := setupRouter(svc, "") // empty userID → simulates missing JWT
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/notifications/"+uuid.New().String()+"/read", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestMarkAllRead_Happy_Returns200 verifies the mark-all-read endpoint.
func TestMarkAllRead_Happy_Returns200(t *testing.T) {
	userID := uuid.New().String()
	called := false
	svc := &mockNotifSvc{
		MarkAllReadFn: func(_ context.Context, uid string) error {
			called = true
			assert.Equal(t, userID, uid)
			return nil
		},
	}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/notifications/me/read-all", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, called, "MarkAllRead must be called")
}

// TestListMine_NoJWT_Returns401 verifies that GET /notifications/me requires JWT.
func TestListMine_NoJWT_Returns401(t *testing.T) {
	svc := &mockNotifSvc{}

	r := setupRouter(svc, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/notifications/me", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestUnreadCount_NoJWT_Returns401 verifies that GET /notifications/me/unread-count requires JWT.
func TestUnreadCount_NoJWT_Returns401(t *testing.T) {
	svc := &mockNotifSvc{}

	r := setupRouter(svc, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/notifications/me/unread-count", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestMarkAllRead_NoJWT_Returns401 verifies that PATCH /notifications/me/read-all requires JWT.
func TestMarkAllRead_NoJWT_Returns401(t *testing.T) {
	svc := &mockNotifSvc{}

	r := setupRouter(svc, "")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/notifications/me/read-all", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestListMine_ServiceError_Returns500 verifies that a service error returns 500.
func TestListMine_ServiceError_Returns500(t *testing.T) {
	userID := uuid.New().String()
	svc := &mockNotifSvc{
		ListMineFn: func(_ context.Context, _ string, _ pagination.Params) (pagination.Page[service.NotificationModel], error) {
			return pagination.Page[service.NotificationModel]{}, service.ErrNotFound
		},
	}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/notifications/me", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestUnreadCount_ServiceError_Returns500 verifies that a service error returns 500.
func TestUnreadCount_ServiceError_Returns500(t *testing.T) {
	userID := uuid.New().String()
	svc := &mockNotifSvc{
		UnreadCountFn: func(_ context.Context, _ string) (int64, error) {
			return 0, service.ErrNotFound
		},
	}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/notifications/me/unread-count", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestMarkAllRead_ServiceError_Returns500 verifies that a service error returns 500.
func TestMarkAllRead_ServiceError_Returns500(t *testing.T) {
	userID := uuid.New().String()
	svc := &mockNotifSvc{
		MarkAllReadFn: func(_ context.Context, _ string) error {
			return service.ErrNotFound
		},
	}

	r := setupRouter(svc, userID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/notifications/me/read-all", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
