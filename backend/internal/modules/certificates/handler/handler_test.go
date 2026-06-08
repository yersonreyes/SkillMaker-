// Package handler — HTTP-layer tests for the certificates module.
// Strategy: httptest + real gin.Engine, mock service, JWT middleware injected.
// No build tag: runs with standard `make backend-test`.
package handler_test

import (
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
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/handler"
	"github.com/yersonreyes/SkillMaker-/backend/internal/modules/certificates/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ── Mock service ────────────────────────────────────────────────────────────────

type mockCertSvc struct {
	ListMyCertificatesFn func(ctx context.Context, userID string) ([]service.CertificateModel, error)
	GetCertificateFn     func(ctx context.Context, certID, userID string) (*service.CertificateModel, error)
	GetDownloadURLFn     func(ctx context.Context, certID, userID string) (service.DownloadResult, error)
	VerifyCertificateFn  func(ctx context.Context, codigo string) (*service.VerifyResult, error)
	ListMyBadgesFn       func(ctx context.Context, userID string) ([]service.BadgeModel, error)
	RankingFn            func(ctx context.Context, n int) ([]service.RankingModel, error)
}

func (m *mockCertSvc) IssueOnPass(_ context.Context, _, _ string) error { return nil }
func (m *mockCertSvc) EvaluateBadges(_ context.Context, _ string) error { return nil }

func (m *mockCertSvc) ListMyCertificates(ctx context.Context, userID string) ([]service.CertificateModel, error) {
	if m.ListMyCertificatesFn != nil {
		return m.ListMyCertificatesFn(ctx, userID)
	}
	return []service.CertificateModel{}, nil
}

func (m *mockCertSvc) GetCertificate(ctx context.Context, certID, userID string) (*service.CertificateModel, error) {
	if m.GetCertificateFn != nil {
		return m.GetCertificateFn(ctx, certID, userID)
	}
	return nil, service.ErrCertificateNotFound
}

func (m *mockCertSvc) GetDownloadURL(ctx context.Context, certID, userID string) (service.DownloadResult, error) {
	if m.GetDownloadURLFn != nil {
		return m.GetDownloadURLFn(ctx, certID, userID)
	}
	return service.DownloadResult{}, service.ErrCertificateNotFound
}

func (m *mockCertSvc) VerifyCertificate(ctx context.Context, codigo string) (*service.VerifyResult, error) {
	if m.VerifyCertificateFn != nil {
		return m.VerifyCertificateFn(ctx, codigo)
	}
	return nil, service.ErrCertificateNotFound
}

func (m *mockCertSvc) ListMyBadges(ctx context.Context, userID string) ([]service.BadgeModel, error) {
	if m.ListMyBadgesFn != nil {
		return m.ListMyBadgesFn(ctx, userID)
	}
	return []service.BadgeModel{}, nil
}

func (m *mockCertSvc) Ranking(ctx context.Context, n int) ([]service.RankingModel, error) {
	if m.RankingFn != nil {
		return m.RankingFn(ctx, n)
	}
	return []service.RankingModel{}, nil
}

// ── Engine setup ────────────────────────────────────────────────────────────────

func setupEngine(svc service.Service) *gin.Engine {
	r := gin.New()
	protected := r.Group("", middleware.JWT("test-secret"))
	handler.Register(protected, svc)
	return r
}

func injectIdentity(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("roles", []string{"alumno"})
		c.Next()
	}
}

func setupEngineWithIdentity(svc service.Service, userID string) *gin.Engine {
	r := gin.New()
	r.Use(injectIdentity(userID))
	handler.Register(r.Group(""), svc)
	return r
}

// ── Tests ────────────────────────────────────────────────────────────────────────

func TestListMyCertificates_200(t *testing.T) {
	userID := uuid.New().String()
	certID := uuid.New().String()

	svc := &mockCertSvc{
		ListMyCertificatesFn: func(_ context.Context, uid string) ([]service.CertificateModel, error) {
			assert.Equal(t, userID, uid)
			return []service.CertificateModel{
				{ID: certID, UserID: userID, CourseID: uuid.New().String(),
					CourseTitulo: "Go Avanzado", Codigo: "ABCD1234EFG12", EmitidoEn: time.Now()},
			}, nil
		},
	}
	r := setupEngineWithIdentity(svc, userID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/me", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	certs, ok := resp["certificates"].([]interface{})
	require.True(t, ok, "response must have certificates array")
	assert.Len(t, certs, 1)
}

func TestListMyCertificates_EmptyArray(t *testing.T) {
	r := setupEngineWithIdentity(&mockCertSvc{}, uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/me", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	certs := resp["certificates"].([]interface{})
	assert.Empty(t, certs)
}

func TestGetCertificate_200_Owner(t *testing.T) {
	userID := uuid.New().String()
	certID := uuid.New().String()

	svc := &mockCertSvc{
		GetCertificateFn: func(_ context.Context, cid, uid string) (*service.CertificateModel, error) {
			assert.Equal(t, certID, cid)
			assert.Equal(t, userID, uid)
			return &service.CertificateModel{ID: certID, UserID: userID, Codigo: "ABC123"}, nil
		},
	}
	r := setupEngineWithIdentity(svc, userID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/"+certID, http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, certID, resp["id"])
}

func TestGetCertificate_404_NonOwner(t *testing.T) {
	svc := &mockCertSvc{
		GetCertificateFn: func(_ context.Context, _, _ string) (*service.CertificateModel, error) {
			return nil, service.ErrCertificateNotFound
		},
	}
	r := setupEngineWithIdentity(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/"+uuid.New().String(), http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCertificate_401_NoJWT(t *testing.T) {
	r := setupEngine(&mockCertSvc{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/some-id", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDownloadCertificate_200(t *testing.T) {
	userID := uuid.New().String()
	certID := uuid.New().String()

	svc := &mockCertSvc{
		GetDownloadURLFn: func(_ context.Context, _, _ string) (service.DownloadResult, error) {
			return service.DownloadResult{
				URL:       "https://minio/cert.pdf",
				ExpiresAt: time.Now().Add(15 * time.Minute),
			}, nil
		},
	}
	r := setupEngineWithIdentity(svc, userID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/"+certID+"/download", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "https://minio/cert.pdf", resp["url"])
	assert.NotEmpty(t, resp["expiresAt"])
}

func TestDownloadCertificate_404_NoPDF(t *testing.T) {
	svc := &mockCertSvc{
		GetDownloadURLFn: func(_ context.Context, _, _ string) (service.DownloadResult, error) {
			return service.DownloadResult{}, service.ErrNoPDF
		},
	}
	r := setupEngineWithIdentity(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/"+uuid.New().String()+"/download", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "CERT_NO_PDF", resp["code"])
}

func TestListMyBadges_200(t *testing.T) {
	userID := uuid.New().String()

	svc := &mockCertSvc{
		ListMyBadgesFn: func(_ context.Context, _ string) ([]service.BadgeModel, error) {
			return []service.BadgeModel{
				{ID: uuid.New().String(), Nombre: "Primer curso completado", OtorgadoEn: time.Now()},
			}, nil
		},
	}
	r := setupEngineWithIdentity(svc, userID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/badges/me", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	badges := resp["badges"].([]interface{})
	assert.Len(t, badges, 1)
}

func TestRanking_200(t *testing.T) {
	svc := &mockCertSvc{
		RankingFn: func(_ context.Context, _ int) ([]service.RankingModel, error) {
			return []service.RankingModel{
				{UserID: uuid.New().String(), Nombre: "Ana", Total: 3},
				{UserID: uuid.New().String(), Nombre: "Bob", Total: 1},
			}, nil
		},
	}
	r := setupEngineWithIdentity(svc, uuid.New().String())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/badges/ranking", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	ranking := resp["ranking"].([]interface{})
	require.Len(t, ranking, 2)

	// Verify posicion field.
	first := ranking[0].(map[string]interface{})
	assert.Equal(t, float64(1), first["posicion"], "first entry must have posicion=1")
	second := ranking[1].(map[string]interface{})
	assert.Equal(t, float64(2), second["posicion"], "second entry must have posicion=2")
}

func TestAllCertRoutes_401_NoJWT(t *testing.T) {
	r := setupEngine(&mockCertSvc{})

	routes := []string{
		"/certificates/me",
		"/certificates/" + uuid.New().String(),
		"/certificates/" + uuid.New().String() + "/download",
		"/badges/me",
		"/badges/ranking",
	}

	for _, path := range routes {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, path, http.NoBody)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code, "route %s must return 401 without JWT", path)
	}
}

// ── Public verify endpoint ─────────────────────────────────────────────────────

// setupEngineWithPublicAndProtected registers BOTH the protected routes (which include
// the param route /certificates/:id) AND the public verify route (/certificates/verify/:codigo)
// on the SAME gin engine. This is the canonical Gin static-vs-param sibling case — the test
// passing (no panic during registration) proves the routes coexist.
func setupEngineWithPublicAndProtected(svc service.Service) *gin.Engine {
	r := gin.New()
	r.Use(injectIdentity(uuid.New().String()))
	handler.Register(r.Group(""), svc)       // /certificates/:id (param)
	handler.RegisterPublic(r.Group(""), svc) // /certificates/verify/:codigo (static + param)
	return r
}

func TestVerifyByCodigo_200(t *testing.T) {
	emitido := time.Now()
	svc := &mockCertSvc{
		VerifyCertificateFn: func(_ context.Context, codigo string) (*service.VerifyResult, error) {
			assert.Equal(t, "ABCD1234EFG12", codigo)
			return &service.VerifyResult{
				Codigo: "ABCD1234EFG12", HolderNombre: "Ana Perez",
				CourseTitulo: "Go Avanzado", EmitidoEn: emitido,
			}, nil
		},
	}
	r := setupEngineWithPublicAndProtected(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/verify/ABCD1234EFG12", http.NoBody)
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Ana Perez", resp["holderNombre"])
	assert.Equal(t, "Go Avanzado", resp["courseTitulo"])
	assert.Equal(t, "ABCD1234EFG12", resp["codigo"])
}

func TestVerifyByCodigo_404_OnUnknownCode(t *testing.T) {
	svc := &mockCertSvc{
		VerifyCertificateFn: func(_ context.Context, _ string) (*service.VerifyResult, error) {
			return nil, service.ErrCertificateNotFound
		},
	}
	r := setupEngineWithPublicAndProtected(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/certificates/verify/UNKNOWN", http.NoBody)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestVerifyRoute_CoexistsWithParamRoute proves the static "verify" segment and the
// param ":id" sibling both resolve correctly on the same engine (no shadowing).
func TestVerifyRoute_CoexistsWithParamRoute(t *testing.T) {
	certID := uuid.New().String()
	svc := &mockCertSvc{
		GetCertificateFn: func(_ context.Context, id, _ string) (*service.CertificateModel, error) {
			assert.Equal(t, certID, id, "GET /certificates/:id must still match the UUID, not be shadowed by verify")
			return &service.CertificateModel{ID: id, Codigo: "X", EmitidoEn: time.Now()}, nil
		},
		VerifyCertificateFn: func(_ context.Context, codigo string) (*service.VerifyResult, error) {
			return &service.VerifyResult{Codigo: codigo, EmitidoEn: time.Now()}, nil
		},
	}
	r := setupEngineWithPublicAndProtected(svc)

	// /certificates/:id resolves to GetByID.
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/certificates/"+certID, http.NoBody)
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// /certificates/verify/:codigo resolves to VerifyByCodigo.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/certificates/verify/SOMECODE", http.NoBody)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}
