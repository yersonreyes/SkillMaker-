// Package handler_test — HTTP layer tests for the auth module.
// Strategy: httptest + gin.TestMode, function-field mock service, JWT middleware for 401 tests.
// No build tag: runs with standard `make backend-test`.
//
// Covers:
//   - Login/Refresh: ip+ua threading (non-empty captured values reach service)
//   - GET /auth/sessions/me: 200 + JSON array (caller-scoped), 401 without JWT
//   - DELETE /auth/sessions/:id: 204 (own), 404 SESSION_NOT_FOUND, 401 without JWT
//   - Caller-scoping: service called with userID from JWT
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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yersonreyes/SkillMaker-/backend/internal/middleware"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/dto"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/auth/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock service ───────────────────────────────────────────────────────────────
// Function-field mock (mirrors reporting/handler/handler_test.go style).
// Satisfies the 5-method service.Service interface.

type mockAuthSvc struct {
	LoginWithGoogleFn    func(ctx context.Context, idTokenStr, ip, userAgent string) (dto.LoginResponse, error)
	RefreshFn            func(ctx context.Context, refreshTokenPlain, ip, userAgent string) (dto.LoginResponse, error)
	LogoutFn             func(ctx context.Context, refreshTokenPlain string) error
	ListActiveSessionsFn func(ctx context.Context, userID string) ([]dto.SessionResponse, error)
	RevokeSessionFn      func(ctx context.Context, userID, sessionID string) error
}

func (m *mockAuthSvc) LoginWithGoogle(ctx context.Context, idTokenStr, ip, userAgent string) (dto.LoginResponse, error) {
	if m.LoginWithGoogleFn != nil {
		return m.LoginWithGoogleFn(ctx, idTokenStr, ip, userAgent)
	}
	return dto.LoginResponse{}, nil
}
func (m *mockAuthSvc) Refresh(ctx context.Context, refreshTokenPlain, ip, userAgent string) (dto.LoginResponse, error) {
	if m.RefreshFn != nil {
		return m.RefreshFn(ctx, refreshTokenPlain, ip, userAgent)
	}
	return dto.LoginResponse{}, nil
}
func (m *mockAuthSvc) Logout(ctx context.Context, refreshTokenPlain string) error {
	if m.LogoutFn != nil {
		return m.LogoutFn(ctx, refreshTokenPlain)
	}
	return nil
}
func (m *mockAuthSvc) ListActiveSessions(ctx context.Context, userID string) ([]dto.SessionResponse, error) {
	if m.ListActiveSessionsFn != nil {
		return m.ListActiveSessionsFn(ctx, userID)
	}
	return []dto.SessionResponse{}, nil
}
func (m *mockAuthSvc) RevokeSession(ctx context.Context, userID, sessionID string) error {
	if m.RevokeSessionFn != nil {
		return m.RevokeSessionFn(ctx, userID, sessionID)
	}
	return nil
}

// ── Engine factories ───────────────────────────────────────────────────────────

// injectIdentity returns a middleware that sets userID and roles without a real JWT.
func injectIdentity(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("roles", []string{"alumno"})
		c.Next()
	}
}

// setupEngineWithIdentity builds an engine with a simulated-auth protected group
// + the pre-auth group for Login/Logout/Refresh routes.
func setupEngineWithIdentity(svc service.Service, callerID string) *gin.Engine {
	r := gin.New()
	api := r.Group("/api")
	handler.Register(api, svc)

	protected := api.Group("", injectIdentity(callerID))
	handler.RegisterSessionRoutes(protected, svc)
	return r
}

// setupEngineWithJWT builds an engine using REAL JWT middleware (for 401 tests).
func setupEngineWithJWT(svc service.Service) *gin.Engine {
	r := gin.New()
	api := r.Group("/api")
	handler.Register(api, svc)

	protected := api.Group("", middleware.JWT("test-secret"))
	handler.RegisterSessionRoutes(protected, svc)
	return r
}

// ── Login/Refresh ip+ua capture tests ─────────────────────────────────────────

// TestLoginWithGoogle_CapturesIPAndUserAgent verifies that the handler passes
// non-empty ip and user-agent to the service.
func TestLoginWithGoogle_CapturesIPAndUserAgent(t *testing.T) {
	var gotIP, gotUA string

	svc := &mockAuthSvc{
		LoginWithGoogleFn: func(_ context.Context, _, ip, ua string) (dto.LoginResponse, error) {
			gotIP = ip
			gotUA = ua
			return dto.LoginResponse{}, nil
		},
	}

	r := gin.New()
	api := r.Group("/api")
	handler.Register(api, svc)

	body, _ := json.Marshal(map[string]string{"idToken": "fake-token"})
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestClient/1.0")
	// Gin's ClientIP() reads RemoteAddr when X-Forwarded-For is absent.
	req.RemoteAddr = "10.0.0.1:9999"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// ip and ua must be non-empty — exact values depend on Gin's extraction.
	assert.NotEmpty(t, gotIP, "login handler must pass non-empty ip to service")
	assert.Equal(t, "TestClient/1.0", gotUA, "login handler must pass User-Agent to service")
}

// TestRefresh_CapturesIPAndUserAgent verifies that the handler passes
// non-empty ip and user-agent to the service on refresh.
func TestRefresh_CapturesIPAndUserAgent(t *testing.T) {
	var gotIP, gotUA string

	svc := &mockAuthSvc{
		RefreshFn: func(_ context.Context, _, ip, ua string) (dto.LoginResponse, error) {
			gotIP = ip
			gotUA = ua
			return dto.LoginResponse{}, nil
		},
	}

	r := gin.New()
	api := r.Group("/api")
	handler.Register(api, svc)

	body, _ := json.Marshal(map[string]string{"refreshToken": "plain-token"})
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "RefreshClient/2.0")
	req.RemoteAddr = "10.0.0.2:9999"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, gotIP, "refresh handler must pass non-empty ip to service")
	assert.Equal(t, "RefreshClient/2.0", gotUA, "refresh handler must pass User-Agent to service")
}

// ── GET /auth/sessions/me ──────────────────────────────────────────────────────

// TestGetMySessions_WithJWT_Returns200 verifies 200 + JSON array with callerID scope.
func TestGetMySessions_WithIdentity_Returns200(t *testing.T) {
	callerID := uuid.New().String()
	var gotCallerID string

	now := time.Now().UTC()
	ip := "10.0.0.1"
	ua := "Chrome/120"
	sessions := []dto.SessionResponse{
		{
			ID:        uuid.New().String(),
			IP:        &ip,
			UserAgent: &ua,
			CreatedAt: now.Add(-time.Hour),
			ExpiresAt: now.Add(7 * 24 * time.Hour),
		},
	}

	svc := &mockAuthSvc{
		ListActiveSessionsFn: func(_ context.Context, userID string) ([]dto.SessionResponse, error) {
			gotCallerID = userID
			return sessions, nil
		},
	}

	r := setupEngineWithIdentity(svc, callerID)

	req, _ := http.NewRequest(http.MethodGet, "/api/auth/sessions/me", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, callerID, gotCallerID, "service must be called with the JWT caller's userID")

	var body []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Len(t, body, 1)
	assert.Equal(t, sessions[0].ID, body[0]["id"])
}

// TestGetMySessions_NoJWT_Returns401 verifies 401 when Authorization header is absent.
func TestGetMySessions_NoJWT_Returns401(t *testing.T) {
	svc := &mockAuthSvc{}
	r := setupEngineWithJWT(svc)

	req, _ := http.NewRequest(http.MethodGet, "/api/auth/sessions/me", http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── DELETE /auth/sessions/:id ──────────────────────────────────────────────────

// TestRevokeSession_OwnSession_Returns204 verifies 204 on successful revoke.
func TestRevokeSession_OwnSession_Returns204(t *testing.T) {
	callerID := uuid.New().String()
	sessionID := uuid.New().String()
	var calledWith struct{ userID, sessID string }

	svc := &mockAuthSvc{
		RevokeSessionFn: func(_ context.Context, userID, sessID string) error {
			calledWith.userID = userID
			calledWith.sessID = sessID
			return nil
		},
	}

	r := setupEngineWithIdentity(svc, callerID)

	req, _ := http.NewRequest(http.MethodDelete, "/api/auth/sessions/"+sessionID, http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, callerID, calledWith.userID, "RevokeSession must be called with callerID from JWT")
	assert.Equal(t, sessionID, calledWith.sessID, "RevokeSession must be called with :id param")
}

// TestRevokeSession_NotFound_Returns404 verifies 404 with SESSION_NOT_FOUND code.
func TestRevokeSession_NotFound_Returns404(t *testing.T) {
	callerID := uuid.New().String()
	sessionID := uuid.New().String()

	svc := &mockAuthSvc{
		RevokeSessionFn: func(_ context.Context, _, _ string) error {
			return service.ErrSessionNotFound
		},
	}

	r := setupEngineWithIdentity(svc, callerID)

	req, _ := http.NewRequest(http.MethodDelete, "/api/auth/sessions/"+sessionID, http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "SESSION_NOT_FOUND", body["code"])
}

// TestRevokeSession_NoJWT_Returns401 verifies 401 without Authorization header.
func TestRevokeSession_NoJWT_Returns401(t *testing.T) {
	svc := &mockAuthSvc{}
	r := setupEngineWithJWT(svc)

	req, _ := http.NewRequest(http.MethodDelete, "/api/auth/sessions/"+uuid.New().String(), http.NoBody)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── S1: missing User-Agent degrades gracefully ────────────────────────────────

// TestLoginWithGoogle_MissingUserAgent_Returns200 verifies that a POST to
// /auth/google without a User-Agent header succeeds with HTTP 200.
// The absent UA degrades gracefully: strPtr("") → nil in the service layer,
// so the login is never blocked by a missing header (S1).
func TestLoginWithGoogle_MissingUserAgent_Returns200(t *testing.T) {
	var gotUA string

	svc := &mockAuthSvc{
		LoginWithGoogleFn: func(_ context.Context, _, _, ua string) (dto.LoginResponse, error) {
			gotUA = ua
			return dto.LoginResponse{AccessToken: "tok", RefreshToken: "ref"}, nil
		},
	}

	r := gin.New()
	api := r.Group("/api")
	handler.Register(api, svc)

	body, _ := json.Marshal(map[string]string{"idToken": "fake-token"})
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Intentionally NO User-Agent header set.
	req.RemoteAddr = "10.0.0.1:9999"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "login must succeed even without a User-Agent header")
	assert.Empty(t, gotUA, "empty User-Agent must reach service as empty string (not blocked)")
}
